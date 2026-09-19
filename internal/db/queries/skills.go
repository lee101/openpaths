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
	owner_id, current_version,
	to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')`

func scanSkill(rows pgx.Rows) (model.Skill, error) {
	var s model.Skill
	err := rows.Scan(&s.ID, &s.Slug, &s.Name, &s.Description, &s.Body, &s.Source, &s.SourceRepo,
		&s.Category, &s.Tags, &s.SetupPreamble, &s.OwnerID, &s.CurrentVersion, &s.CreatedAt, &s.UpdatedAt)
	if s.Tags == nil {
		s.Tags = []string{}
	}
	if s.CurrentVersion == "" {
		s.CurrentVersion = "1.0.0"
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	q.hydrateHeads(ctx, out)
	return out, nil
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
	rows.Close()
	h := []model.Skill{s}
	q.hydrateHeads(ctx, h)
	return &h[0], nil
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
	cur := s.CurrentVersion
	if cur == "" {
		cur = "1.0.0"
	}
	_, err := q.pool.Exec(ctx,
		`INSERT INTO skills (id, slug, name, description, body, source, source_repo, category, tags, setup_preamble, owner_id, current_version)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,COALESCE(NULLIF($11,''),''), $12)
		 ON CONFLICT (slug) DO UPDATE SET
			name=EXCLUDED.name, description=EXCLUDED.description, body=EXCLUDED.body,
			source=EXCLUDED.source, source_repo=EXCLUDED.source_repo, category=EXCLUDED.category,
			tags=EXCLUDED.tags, setup_preamble=EXCLUDED.setup_preamble, updated_at=now()`,
		s.ID, s.Slug, s.Name, s.Description, s.Body, s.Source, s.SourceRepo, s.Category, s.Tags, s.SetupPreamble, s.OwnerID, cur)
	if err != nil {
		return fmt.Errorf("upsert skill: %w", err)
	}
	// Best-effort head version row for ingest/legacy rows (skill_versions may
	// not exist on DBs predating the migration).
	_, _ = q.pool.Exec(ctx,
		`INSERT INTO skill_versions (skill_id, version, body, setup_script, setup_prompt, skill_prompt)
		 VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (skill_id, version) DO NOTHING`,
		s.ID, cur, s.Body, s.SetupScript, s.SetupPrompt, s.SkillPrompt)
	return nil
}

// --- Versioned marketplace ---

// CreateSkill inserts a new skill row owned by s.OwnerID at version 1.0.0
// together with its first immutable version snapshot.
func (q *SkillQueries) CreateSkill(ctx context.Context, s *model.Skill) error {
	if s == nil || s.Slug == "" || s.Name == "" {
		return fmt.Errorf("skill slug and name required")
	}
	if s.Tags == nil {
		s.Tags = []string{}
	}
	if s.CurrentVersion == "" {
		s.CurrentVersion = "1.0.0"
	}
	if s.ID == "" {
		s.ID = s.Slug
	}
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("create skill: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx,
		`INSERT INTO skills (id, slug, name, description, body, source, source_repo, category, tags, setup_preamble, owner_id, current_version)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		s.ID, s.Slug, s.Name, s.Description, s.Body, s.Source, s.SourceRepo, s.Category, s.Tags, s.SetupPreamble, s.OwnerID, s.CurrentVersion); err != nil {
		return fmt.Errorf("create skill: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO skill_versions (skill_id, version, body, setup_script, setup_prompt, skill_prompt)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		s.ID, s.CurrentVersion, s.Body, s.SetupScript, s.SetupPrompt, s.SkillPrompt); err != nil {
		return fmt.Errorf("create skill version: %w", err)
	}
	return tx.Commit(ctx)
}

// CreateVersion appends a new immutable version snapshot for skillID.
// The head row is left untouched; call UpdateHead to move the pin.
func (q *SkillQueries) CreateVersion(ctx context.Context, skillID string, v *model.SkillVersion) error {
	if skillID == "" || v == nil || v.Version == "" {
		return fmt.Errorf("skill id and version required")
	}
	if !model.IsValidVersion(v.Version) {
		return fmt.Errorf("invalid version %q: want semver '1.0.0'", v.Version)
	}
	_, err := q.pool.Exec(ctx,
		`INSERT INTO skill_versions (skill_id, version, body, setup_script, setup_prompt, skill_prompt)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		skillID, v.Version, v.Body, v.SetupScript, v.SetupPrompt, v.SkillPrompt)
	if err != nil {
		return fmt.Errorf("create skill version: %w", err)
	}
	return nil
}

// GetVersion fetches one immutable snapshot. Returns pgx.ErrNoRows when the
// skill or the pinned version does not exist.
func (q *SkillQueries) GetVersion(ctx context.Context, skillID, version string) (*model.SkillVersion, error) {
	version = strings.TrimSpace(version)
	if skillID == "" || version == "" {
		return nil, pgx.ErrNoRows
	}
	var v model.SkillVersion
	err := q.pool.QueryRow(ctx,
		`SELECT skill_id, version, body, setup_script, setup_prompt, skill_prompt,
			to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		 FROM skill_versions WHERE skill_id = $1 AND version = $2`,
		skillID, version).Scan(&v.SkillID, &v.Version, &v.Body, &v.SetupScript, &v.SetupPrompt, &v.SkillPrompt, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListVersions returns every version string for a skill, oldest first.
func (q *SkillQueries) ListVersions(ctx context.Context, skillID string) ([]string, error) {
	if skillID == "" {
		return []string{}, nil
	}
	rows, err := q.pool.Query(ctx,
		`SELECT version FROM skill_versions WHERE skill_id = $1 ORDER BY created_at, version`, skillID)
	if err != nil {
		if isMissingRelation(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("list skill versions: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// UpdateHead moves the head row (content + current_version pin).
func (q *SkillQueries) UpdateHead(ctx context.Context, s *model.Skill) error {
	if s == nil || s.ID == "" {
		return fmt.Errorf("skill id required")
	}
	if s.Tags == nil {
		s.Tags = []string{}
	}
	tag, err := q.pool.Exec(ctx,
		`UPDATE skills SET name=$2, description=$3, body=$4, source=$5, source_repo=$6,
		 category=$7, tags=$8, setup_preamble=$9, current_version=$10, updated_at=now()
		 WHERE id = $1`,
		s.ID, s.Name, s.Description, s.Body, s.Source, s.SourceRepo, s.Category, s.Tags, s.SetupPreamble, s.CurrentVersion)
	if err != nil {
		return fmt.Errorf("update skill head: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// DeleteSkill removes a skill and (via FK cascade) its versions and files.
func (q *SkillQueries) DeleteSkill(ctx context.Context, slug string) error {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return pgx.ErrNoRows
	}
	tag, err := q.pool.Exec(ctx, `DELETE FROM skills WHERE slug = $1`, slug)
	if err != nil {
		return fmt.Errorf("delete skill: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// UpsertFile stores one file's bytes for an immutable version.
func (q *SkillQueries) UpsertFile(ctx context.Context, skillID, version, path, mime string, content []byte) error {
	if skillID == "" || version == "" || path == "" {
		return fmt.Errorf("skill id, version and path required")
	}
	_, err := q.pool.Exec(ctx,
		`INSERT INTO skill_files (skill_id, version, path, mime, size, content)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (skill_id, version, path) DO UPDATE SET mime=EXCLUDED.mime, size=EXCLUDED.size, content=EXCLUDED.content`,
		skillID, version, path, mime, len(content), content)
	if err != nil {
		return fmt.Errorf("upsert skill file: %w", err)
	}
	return nil
}

// ListFiles returns the file manifest (no bytes) for a version.
func (q *SkillQueries) ListFiles(ctx context.Context, skillID, version string) ([]model.SkillFile, error) {
	out := []model.SkillFile{}
	if skillID == "" || version == "" {
		return out, nil
	}
	rows, err := q.pool.Query(ctx,
		`SELECT path, mime, size FROM skill_files WHERE skill_id = $1 AND version = $2 ORDER BY path`,
		skillID, version)
	if err != nil {
		if isMissingRelation(err) {
			return out, nil
		}
		return nil, fmt.Errorf("list skill files: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var f model.SkillFile
		if err := rows.Scan(&f.Path, &f.Mime, &f.Size); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// GetFile fetches one file's raw bytes plus mime.
func (q *SkillQueries) GetFile(ctx context.Context, skillID, version, path string) ([]byte, string, error) {
	if skillID == "" || version == "" || path == "" {
		return nil, "", pgx.ErrNoRows
	}
	var content []byte
	var mime string
	err := q.pool.QueryRow(ctx,
		`SELECT content, mime FROM skill_files WHERE skill_id = $1 AND version = $2 AND path = $3`,
		skillID, version, path).Scan(&content, &mime)
	if err != nil {
		return nil, "", err
	}
	return content, mime, nil
}

// CopyVersionFiles duplicates every file row from one version to another
// (used on PUT when the caller omits files so the manifest carries over).
func (q *SkillQueries) CopyVersionFiles(ctx context.Context, skillID, fromVersion, toVersion string) error {
	if skillID == "" || fromVersion == "" || toVersion == "" || fromVersion == toVersion {
		return nil
	}
	_, err := q.pool.Exec(ctx,
		`INSERT INTO skill_files (skill_id, version, path, mime, size, content)
		 SELECT skill_id, $3, path, mime, size, content FROM skill_files
		 WHERE skill_id = $1 AND version = $2
		 ON CONFLICT (skill_id, version, path) DO NOTHING`,
		skillID, fromVersion, toVersion)
	if err != nil {
		if isMissingRelation(err) {
			return nil
		}
		return fmt.Errorf("copy skill files: %w", err)
	}
	return nil
}

// BackfillHeadVersions ensures every head row has a matching immutable version
// snapshot (legacy ingest rows predate skill_versions) and defaults empty
// current_version pins to '1.0.0'.
func (q *SkillQueries) BackfillHeadVersions(ctx context.Context) error {
	if _, err := q.pool.Exec(ctx,
		`UPDATE skills SET current_version = '1.0.0' WHERE current_version IS NULL OR current_version = ''`); err != nil {
		if isMissingRelation(err) {
			return nil
		}
		return fmt.Errorf("backfill skill pins: %w", err)
	}
	_, err := q.pool.Exec(ctx,
		`INSERT INTO skill_versions (skill_id, version, body, setup_script, setup_prompt, skill_prompt)
		 SELECT id, current_version, body, '', '', '' FROM skills
		 ON CONFLICT (skill_id, version) DO NOTHING`)
	if err != nil {
		if isMissingRelation(err) {
			return nil
		}
		return fmt.Errorf("backfill skill versions: %w", err)
	}
	return nil
}

// hydrateHeads attaches each skill's head-version content (setup fields),
// version count. Best-effort: silently skips when skill_versions does not
// exist yet (DB predating the migration).
func (q *SkillQueries) hydrateHeads(ctx context.Context, skills []model.Skill) {
	if q == nil || q.pool == nil || len(skills) == 0 {
		return
	}
	ids := make([]any, 0, len(skills))
	byID := make(map[string]int, len(skills))
	for i := range skills {
		if skills[i].CurrentVersion == "" {
			skills[i].CurrentVersion = "1.0.0"
		}
		if _, ok := byID[skills[i].ID]; !ok {
			byID[skills[i].ID] = i
			ids = append(ids, skills[i].ID)
		}
	}
	rows, err := q.pool.Query(ctx,
		`SELECT skill_id, version, body, setup_script, setup_prompt, skill_prompt, COUNT(*) OVER (PARTITION BY skill_id)
		 FROM skill_versions WHERE skill_id = ANY($1)`, idsToArray(ids))
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, version, body, script, prompt, sprompt string
		var count int
		if err := rows.Scan(&id, &version, &body, &script, &prompt, &sprompt, &count); err != nil {
			continue
		}
		i, ok := byID[id]
		if !ok {
			continue
		}
		if skills[i].VersionCount < count {
			skills[i].VersionCount = count
		}
		if version == skills[i].CurrentVersion {
			skills[i].SetupScript = script
			skills[i].SetupPrompt = prompt
			skills[i].SkillPrompt = sprompt
		}
	}
}

func idsToArray(ids []any) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if s, ok := id.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func isMissingRelation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "does not exist")
}
