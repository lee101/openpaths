package queries

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpaths/openpaths/internal/model"
)

// SkillQueries is the data layer for the agent-skill library (skills table).
type SkillQueries struct {
	pool *pgxpool.Pool
}

func NewSkillQueries(pool *pgxpool.Pool) *SkillQueries {
	return &SkillQueries{pool: pool}
}

const skillColumns = `id, slug, name, description, body, source, source_repo, category, tags, setup_preamble,
	to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`

func scanSkill(rows pgx.Rows) (model.Skill, error) {
	var s model.Skill
	err := rows.Scan(&s.ID, &s.Slug, &s.Name, &s.Description, &s.Body, &s.Source, &s.SourceRepo,
		&s.Category, &s.Tags, &s.SetupPreamble, &s.CreatedAt, &s.UpdatedAt)
	if s.Tags == nil {
		s.Tags = []string{}
	}
	return s, err
}

func skillWhereClause(f model.SkillFilters, start int) (string, []any) {
	parts := []string{"TRUE"}
	var args []any
	n := start
	if f.Source != "" {
		parts = append(parts, fmt.Sprintf("source = $%d", n))
		args = append(args, f.Source)
		n++
	}
	if f.Category != "" {
		parts = append(parts, fmt.Sprintf("category = $%d", n))
		args = append(args, f.Category)
		n++
	}
	return strings.Join(parts, " AND "), args
}

// List browses skills with optional filters, by name.
func (q *SkillQueries) List(ctx context.Context, f model.SkillFilters, limit, offset int) ([]model.Skill, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	where, args := skillWhereClause(f, 1)
	args = append(args, limit, offset)
	sql := fmt.Sprintf(`SELECT %s FROM skills WHERE %s ORDER BY name LIMIT $%d OFFSET $%d`,
		skillColumns, where, len(args)-1, len(args))
	return q.queryList(ctx, sql, args)
}

// SearchILIKE is the lexical fallback (pg_trgm): substring/fuzzy over name +
// description + body, ranked name-first.
func (q *SkillQueries) SearchILIKE(ctx context.Context, query string, f model.SkillFilters, limit int) ([]model.Skill, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return q.List(ctx, f, limit, 0)
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	where, args := skillWhereClause(f, 2) // $1 reserved for query
	allArgs := append([]any{query}, args...)
	allArgs = append(allArgs, limit)
	sql := fmt.Sprintf(`SELECT %s FROM skills
		WHERE %s AND (name ILIKE '%%' || $1 || '%%' OR description ILIKE '%%' || $1 || '%%' OR body ILIKE '%%' || $1 || '%%')
		ORDER BY GREATEST(similarity(name, $1) * 2, similarity(description, $1)) DESC, name
		LIMIT $%d`,
		skillColumns, where, len(allArgs))
	return q.queryList(ctx, sql, allArgs)
}

func (q *SkillQueries) queryList(ctx context.Context, sql string, args []any) ([]model.Skill, error) {
	rows, err := q.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query skills: %w", err)
	}
	defer rows.Close()
	out := make([]model.Skill, 0, 48)
	for rows.Next() {
		s, err := scanSkill(rows)
		if err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetBySlug fetches a single skill (with body) by slug.
func (q *SkillQueries) GetBySlug(ctx context.Context, slug string) (*model.Skill, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, pgx.ErrNoRows
	}
	rows, err := q.pool.Query(ctx, fmt.Sprintf(`SELECT %s FROM skills WHERE slug = $1 LIMIT 1`, skillColumns), slug)
	if err != nil {
		return nil, fmt.Errorf("get skill: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, pgx.ErrNoRows
	}
	s, err := scanSkill(rows)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Count returns the number of skills.
func (q *SkillQueries) Count(ctx context.Context) (int, error) {
	var n int
	err := q.pool.QueryRow(ctx, `SELECT COUNT(*) FROM skills`).Scan(&n)
	return n, err
}

// Facet is a value with its skill count (sources / categories).
type Facet struct {
	Value string `json:"value"`
	Repo  string `json:"repo,omitempty"`
	Count int    `json:"count"`
}

// SourceCounts returns skill counts per source (with a representative repo).
func (q *SkillQueries) SourceCounts(ctx context.Context) ([]Facet, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT source, COALESCE(MAX(source_repo), ''), COUNT(*) FROM skills GROUP BY source ORDER BY source`)
	if err != nil {
		return nil, fmt.Errorf("source counts: %w", err)
	}
	defer rows.Close()
	out := []Facet{}
	for rows.Next() {
		var f Facet
		if err := rows.Scan(&f.Value, &f.Repo, &f.Count); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// CategoryCounts returns skill counts per non-empty category.
func (q *SkillQueries) CategoryCounts(ctx context.Context) ([]Facet, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT category, COUNT(*) FROM skills WHERE category != '' GROUP BY category ORDER BY category`)
	if err != nil {
		return nil, fmt.Errorf("category counts: %w", err)
	}
	defer rows.Close()
	out := []Facet{}
	for rows.Next() {
		var f Facet
		if err := rows.Scan(&f.Value, &f.Count); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// IterForIndex streams all skills for building the semantic index, ordered by
// slug for a stable, cacheable order. cap <= 0 means all rows.
func (q *SkillQueries) IterForIndex(ctx context.Context, cap int, fn func(model.Skill) error) (int, error) {
	sql := fmt.Sprintf(`SELECT %s FROM skills ORDER BY slug`, skillColumns)
	if cap > 0 {
		sql += fmt.Sprintf(" LIMIT %d", cap)
	}
	rows, err := q.pool.Query(ctx, sql)
	if err != nil {
		return 0, fmt.Errorf("iter skills: %w", err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		s, err := scanSkill(rows)
		if err != nil {
			return n, fmt.Errorf("scan skill: %w", err)
		}
		if err := fn(s); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

// Upsert inserts or updates a single skill by slug (used by cmd/skill-ingest).
func (q *SkillQueries) Upsert(ctx context.Context, s *model.Skill) error {
	if s == nil || s.ID == "" || s.Slug == "" {
		return fmt.Errorf("skill id and slug required")
	}
	if s.Tags == nil {
		s.Tags = []string{}
	}
	_, err := q.pool.Exec(ctx,
		`INSERT INTO skills (id, slug, name, description, body, source, source_repo, category, tags, setup_preamble)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 ON CONFLICT (slug) DO UPDATE SET
			name=EXCLUDED.name, description=EXCLUDED.description, body=EXCLUDED.body,
			source=EXCLUDED.source, source_repo=EXCLUDED.source_repo, category=EXCLUDED.category,
			tags=EXCLUDED.tags, setup_preamble=EXCLUDED.setup_preamble, updated_at=now()`,
		s.ID, s.Slug, s.Name, s.Description, s.Body, s.Source, s.SourceRepo, s.Category, s.Tags, s.SetupPreamble)
	if err != nil {
		return fmt.Errorf("upsert skill: %w", err)
	}
	return nil
}
