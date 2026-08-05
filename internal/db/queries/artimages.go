package queries

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpaths/openpaths/internal/model"
)

// ArtImageQueries is the data layer for the image prompt search engine (art_images).
type ArtImageQueries struct {
	pool *pgxpool.Pool
}

func NewArtImageQueries(pool *pgxpool.Pool) *ArtImageQueries {
	return &ArtImageQueries{pool: pool}
}

const artImageColumns = `id, slug, prompt, title, width, height, aspect, image_url, thumb_url, model, seed, steps, source, tags, nsfw, media_type, video_url, poster_url, duration_seconds, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`

func scanArtImage(rows pgx.Rows) (model.ArtImage, error) {
	var a model.ArtImage
	err := rows.Scan(&a.ID, &a.Slug, &a.Prompt, &a.Title, &a.Width, &a.Height, &a.Aspect,
		&a.ImageURL, &a.ThumbURL, &a.Model, &a.Seed, &a.Steps, &a.Source, &a.Tags, &a.NSFW,
		&a.MediaType, &a.VideoURL, &a.PosterURL, &a.DurationSeconds, &a.CreatedAt)
	return a, err
}

// artWhereClause builds a parameterized filter fragment starting at $start.
// Returns the SQL (without leading "WHERE"/"AND") and the args in order.
func artWhereClause(f model.ArtImageFilters, start int) (string, []any) {
	var parts []string
	var args []any
	n := start
	parts = append(parts, "nsfw = FALSE")
	if f.Aspect != "" {
		parts = append(parts, fmt.Sprintf("aspect = $%d", n))
		args = append(args, f.Aspect)
		n++
	}
	if f.Tag != "" {
		parts = append(parts, fmt.Sprintf("$%d = ANY(tags)", n))
		args = append(args, strings.ToLower(f.Tag))
		n++
	}
	if f.Source != "" {
		parts = append(parts, fmt.Sprintf("source = $%d", n))
		args = append(args, f.Source)
		n++
	}
	if f.MediaType != "" {
		parts = append(parts, fmt.Sprintf("media_type = $%d", n))
		args = append(args, f.MediaType)
		n++
	}
	if f.MinW > 0 {
		parts = append(parts, fmt.Sprintf("width >= $%d", n))
		args = append(args, f.MinW)
		n++
	}
	if f.MinH > 0 {
		parts = append(parts, fmt.Sprintf("height >= $%d", n))
		args = append(args, f.MinH)
		n++
	}
	return strings.Join(parts, " AND "), args
}

// List browses images with optional filters, newest first.
func (q *ArtImageQueries) List(ctx context.Context, f model.ArtImageFilters, limit, offset int) ([]model.ArtImage, error) {
	if limit <= 0 || limit > 200 {
		limit = 48
	}
	if offset < 0 {
		offset = 0
	}
	where, args := artWhereClause(f, 1)
	args = append(args, limit, offset)
	sql := fmt.Sprintf(`SELECT %s FROM art_images WHERE %s ORDER BY created_at DESC, id LIMIT $%d OFFSET $%d`,
		artImageColumns, where, len(args)-1, len(args))
	return q.queryList(ctx, sql, args)
}

// SearchTrigram does exact/substring/fuzzy prompt search using pg_trgm, ranked by similarity.
func (q *ArtImageQueries) SearchTrigram(ctx context.Context, query string, f model.ArtImageFilters, limit, offset int) ([]model.ArtImage, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return q.List(ctx, f, limit, offset)
	}
	if limit <= 0 || limit > 200 {
		limit = 48
	}
	if offset < 0 {
		offset = 0
	}
	where, args := artWhereClause(f, 2) // $1 reserved for query
	allArgs := append([]any{query}, args...)
	allArgs = append(allArgs, limit, offset)
	sql := fmt.Sprintf(`SELECT %s FROM art_images
		WHERE %s AND (prompt ILIKE '%%' || $1 || '%%' OR prompt %% $1)
		ORDER BY similarity(prompt, $1) DESC, created_at DESC
		LIMIT $%d OFFSET $%d`,
		artImageColumns, where, len(allArgs)-1, len(allArgs))
	return q.queryList(ctx, sql, allArgs)
}

func (q *ArtImageQueries) queryList(ctx context.Context, sql string, args []any) ([]model.ArtImage, error) {
	rows, err := q.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query art images: %w", err)
	}
	defer rows.Close()
	out := make([]model.ArtImage, 0, 48)
	for rows.Next() {
		a, err := scanArtImage(rows)
		if err != nil {
			return nil, fmt.Errorf("scan art image: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Count returns the number of (non-nsfw) images matching the filters.
func (q *ArtImageQueries) Count(ctx context.Context, f model.ArtImageFilters) (int, error) {
	where, args := artWhereClause(f, 1)
	var count int
	err := q.pool.QueryRow(ctx, `SELECT COUNT(*) FROM art_images WHERE `+where, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count art images: %w", err)
	}
	return count, nil
}

// AspectCounts returns counts per aspect bucket for facet display.
func (q *ArtImageQueries) AspectCounts(ctx context.Context) (map[string]int, error) {
	rows, err := q.pool.Query(ctx, `SELECT aspect, COUNT(*) FROM art_images WHERE nsfw = FALSE GROUP BY aspect`)
	if err != nil {
		return nil, fmt.Errorf("aspect counts: %w", err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var aspect string
		var n int
		if err := rows.Scan(&aspect, &n); err != nil {
			return nil, err
		}
		out[aspect] = n
	}
	return out, rows.Err()
}

// GetBySlug fetches a single image by slug.
func (q *ArtImageQueries) GetBySlug(ctx context.Context, slug string) (*model.ArtImage, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, pgx.ErrNoRows
	}
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`SELECT %s FROM art_images WHERE slug = $1 LIMIT 1`, artImageColumns), slug)
	if err != nil {
		return nil, fmt.Errorf("get art image: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, pgx.ErrNoRows
	}
	a, err := scanArtImage(rows)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// TagCount is a tag facet with its image count.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// TopTags returns the most common tags. This unnests over the table so callers
// should cache the result (see artindex.Service tag cache) rather than calling per request.
func (q *ArtImageQueries) TopTags(ctx context.Context, limit int) ([]TagCount, error) {
	if limit <= 0 || limit > 500 {
		limit = 60
	}
	rows, err := q.pool.Query(ctx,
		`SELECT tag, COUNT(*) AS n FROM (
			SELECT unnest(tags) AS tag FROM art_images WHERE nsfw = FALSE
		) t
		GROUP BY tag
		HAVING COUNT(*) >= 2
		ORDER BY n DESC, tag ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("top tags: %w", err)
	}
	defer rows.Close()
	out := make([]TagCount, 0, limit)
	for rows.Next() {
		var tc TagCount
		if err := rows.Scan(&tc.Tag, &tc.Count); err != nil {
			return nil, err
		}
		out = append(out, tc)
	}
	return out, rows.Err()
}

// IterForIndex streams id+slug+prompt+dims+urls+tags for building the semantic index.
// cap <= 0 means all rows. Ordered by id for a stable, resumable/cacheable order.
func (q *ArtImageQueries) IterForIndex(ctx context.Context, cap int, fn func(model.ArtImage) error) (int, error) {
	sql := fmt.Sprintf(`SELECT %s FROM art_images WHERE nsfw = FALSE ORDER BY id`, artImageColumns)
	if cap > 0 {
		sql += fmt.Sprintf(" LIMIT %d", cap)
	}
	rows, err := q.pool.Query(ctx, sql)
	if err != nil {
		return 0, fmt.Errorf("iter art images: %w", err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		a, err := scanArtImage(rows)
		if err != nil {
			return n, fmt.Errorf("scan art image: %w", err)
		}
		if err := fn(a); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

// SitemapImage is the minimal row needed to emit an image sitemap entry.
type SitemapImage struct {
	Slug     string
	Prompt   string
	ThumbURL string
	ImageURL string
}

// ListForSitemap returns slugs + image URLs for sitemap generation, ordered stably by id.
func (q *ArtImageQueries) ListForSitemap(ctx context.Context, limit, offset int) ([]SitemapImage, error) {
	if limit <= 0 || limit > 50000 {
		limit = 45000
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := q.pool.Query(ctx,
		`SELECT slug, prompt, thumb_url, image_url FROM art_images
		 WHERE nsfw = FALSE ORDER BY id LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list for sitemap: %w", err)
	}
	defer rows.Close()
	out := make([]SitemapImage, 0, limit)
	for rows.Next() {
		var s SitemapImage
		if err := rows.Scan(&s.Slug, &s.Prompt, &s.ThumbURL, &s.ImageURL); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Upsert inserts or updates a single image (used by the generator). Slug must be set.
func (q *ArtImageQueries) Upsert(ctx context.Context, a *model.ArtImage) error {
	if a == nil || a.ID == "" || a.Slug == "" {
		return fmt.Errorf("art image id and slug required")
	}
	if a.Aspect == "" {
		a.Aspect = model.AspectFor(a.Width, a.Height)
	}
	if a.Model == "" {
		a.Model = "zimage"
	}
	if a.Source == "" {
		a.Source = "cutedsl"
	}
	if a.MediaType == "" {
		a.MediaType = "image"
	}
	_, err := q.pool.Exec(ctx,
		`INSERT INTO art_images (id, slug, prompt, title, width, height, aspect, image_url, thumb_url, model, seed, steps, source, tags, nsfw, media_type, video_url, poster_url, duration_seconds)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		 ON CONFLICT (id) DO UPDATE SET
			prompt=EXCLUDED.prompt, title=EXCLUDED.title, width=EXCLUDED.width, height=EXCLUDED.height,
			aspect=EXCLUDED.aspect, image_url=EXCLUDED.image_url, thumb_url=EXCLUDED.thumb_url,
			model=EXCLUDED.model, seed=EXCLUDED.seed, steps=EXCLUDED.steps, source=EXCLUDED.source,
			tags=EXCLUDED.tags, nsfw=EXCLUDED.nsfw, media_type=EXCLUDED.media_type, video_url=EXCLUDED.video_url,
			poster_url=EXCLUDED.poster_url, duration_seconds=EXCLUDED.duration_seconds`,
		a.ID, a.Slug, a.Prompt, a.Title, a.Width, a.Height, a.Aspect, a.ImageURL, a.ThumbURL,
		a.Model, a.Seed, a.Steps, a.Source, a.Tags, a.NSFW, a.MediaType, a.VideoURL, a.PosterURL, a.DurationSeconds)
	if err != nil {
		return fmt.Errorf("upsert art image: %w", err)
	}
	return nil
}
