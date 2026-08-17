package queries

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Artifact struct {
	ID          string
	UserID      string
	Slug        string
	Title       string
	Description string
	ImageURL    string
	Files       json.RawMessage
	Entry       string
	Visibility  string
	Tags        []string
	ForkOf      *string
	ViewCount   int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt *time.Time
}

type ArtifactQueries struct {
	pool *pgxpool.Pool
}

func NewArtifactQueries(pool *pgxpool.Pool) *ArtifactQueries {
	return &ArtifactQueries{pool: pool}
}

const artifactCols = `id, user_id::text, slug, title, description, image_url, files, entry,
	visibility, tags, fork_of, view_count, created_at, updated_at, published_at`

func scanArtifact(row rowScanner) (*Artifact, error) {
	a := &Artifact{}
	err := row.Scan(&a.ID, &a.UserID, &a.Slug, &a.Title, &a.Description, &a.ImageURL, &a.Files, &a.Entry,
		&a.Visibility, &a.Tags, &a.ForkOf, &a.ViewCount, &a.CreatedAt, &a.UpdatedAt, &a.PublishedAt)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (q *ArtifactQueries) scanOne(ctx context.Context, sql string, args ...any) (*Artifact, error) {
	return scanArtifact(q.pool.QueryRow(ctx, sql, args...))
}

func (q *ArtifactQueries) queryList(ctx context.Context, sql string, args ...any) ([]*Artifact, error) {
	rows, err := q.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Artifact
	for rows.Next() {
		a, err := scanArtifact(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (q *ArtifactQueries) Create(ctx context.Context, a *Artifact) (*Artifact, error) {
	if a.Title == "" {
		a.Title = "Untitled"
	}
	if a.Entry == "" {
		a.Entry = "index.html"
	}
	if a.Visibility == "" {
		a.Visibility = "private"
	}
	if len(a.Files) == 0 {
		a.Files = json.RawMessage("[]")
	}
	if a.Tags == nil {
		a.Tags = []string{}
	}
	published := "NULL"
	if a.Visibility == "public" || a.Visibility == "unlisted" {
		published = "now()"
	}
	return q.scanOne(ctx, `
		INSERT INTO artifacts (id, user_id, slug, title, description, image_url, files, entry, visibility, tags, fork_of, published_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10,$11,`+published+`)
		RETURNING `+artifactCols,
		a.ID, a.UserID, a.Slug, a.Title, a.Description, a.ImageURL, a.Files, a.Entry, a.Visibility, a.Tags, nullStrPtr(a.ForkOf))
}

func (q *ArtifactQueries) GetByID(ctx context.Context, id string) (*Artifact, error) {
	return q.scanOne(ctx, `SELECT `+artifactCols+` FROM artifacts WHERE id=$1`, id)
}

func (q *ArtifactQueries) GetBySlug(ctx context.Context, slug string) (*Artifact, error) {
	return q.scanOne(ctx, `SELECT `+artifactCols+` FROM artifacts WHERE slug=$1`, slug)
}

func (q *ArtifactQueries) ListByUser(ctx context.Context, userID string, limit int) ([]*Artifact, error) {
	limit = clampLimit(limit, 50, 200)
	return q.queryList(ctx, `SELECT `+artifactCols+`
		FROM artifacts WHERE user_id=$1 ORDER BY updated_at DESC LIMIT $2`, userID, limit)
}

func (q *ArtifactQueries) ListPublic(ctx context.Context, limit, offset int) ([]*Artifact, error) {
	limit = clampLimit(limit, 48, 200)
	if offset < 0 {
		offset = 0
	}
	return q.queryList(ctx, `SELECT `+artifactCols+`
		FROM artifacts WHERE visibility='public'
		ORDER BY published_at DESC NULLS LAST, created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
}

// Search matches title/description/tags. When userID is empty only public artifacts are returned;
// otherwise public artifacts plus the user's own are searched.
func (q *ArtifactQueries) Search(ctx context.Context, term, userID string, limit int) ([]*Artifact, error) {
	limit = clampLimit(limit, 24, 100)
	term = strings.TrimSpace(term)
	pattern := "%" + term + "%"
	if userID == "" {
		return q.queryList(ctx, `SELECT `+artifactCols+`
			FROM artifacts
			WHERE visibility='public' AND ($1='' OR title ILIKE $2 OR description ILIKE $2 OR $1 = ANY(tags))
			ORDER BY view_count DESC, published_at DESC NULLS LAST LIMIT $3`, term, pattern, limit)
	}
	return q.queryList(ctx, `SELECT `+artifactCols+`
		FROM artifacts
		WHERE (visibility='public' OR user_id=$4)
		  AND ($1='' OR title ILIKE $2 OR description ILIKE $2 OR $1 = ANY(tags))
		ORDER BY view_count DESC, updated_at DESC LIMIT $3`, term, pattern, limit, userID)
}

func (q *ArtifactQueries) Update(ctx context.Context, a *Artifact) (*Artifact, error) {
	if len(a.Files) == 0 {
		a.Files = json.RawMessage("[]")
	}
	if a.Tags == nil {
		a.Tags = []string{}
	}
	return q.scanOne(ctx, `
		UPDATE artifacts SET
			title=$2, description=$3, image_url=$4, files=$5::jsonb, entry=$6, visibility=$7, tags=$8,
			published_at=CASE WHEN $7 IN ('public','unlisted') AND published_at IS NULL THEN now() ELSE published_at END,
			updated_at=now()
		WHERE id=$1 AND user_id=$9
		RETURNING `+artifactCols,
		a.ID, a.Title, a.Description, a.ImageURL, a.Files, a.Entry, a.Visibility, a.Tags, a.UserID)
}

func (q *ArtifactQueries) Delete(ctx context.Context, id, userID string) error {
	_, err := q.pool.Exec(ctx, `DELETE FROM artifacts WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

func (q *ArtifactQueries) IncrementViews(ctx context.Context, id string) {
	_, _ = q.pool.Exec(ctx, `UPDATE artifacts SET view_count=view_count+1 WHERE id=$1`, id)
}

func nullStrPtr(s *string) any {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}

func clampLimit(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}
