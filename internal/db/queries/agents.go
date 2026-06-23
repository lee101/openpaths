package queries

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/openpaths/openpaths/internal/model"
)

func defaultStr(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

type AgentQueries struct {
	pool *pgxpool.Pool
}

func NewAgentQueries(pool *pgxpool.Pool) *AgentQueries {
	return &AgentQueries{pool: pool}
}

const agentCols = `id, user_id, slug, name, description, system_prompt, model, config, preset, created_at, updated_at`

func scanAgent(row interface{ Scan(dest ...any) error }) (*model.Agent, error) {
	var a model.Agent
	var cfg []byte
	if err := row.Scan(&a.ID, &a.UserID, &a.Slug, &a.Name, &a.Description, &a.SystemPrompt, &a.Model, &cfg, &a.Preset, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	if len(cfg) > 0 {
		_ = json.Unmarshal(cfg, &a.Config)
	}
	return &a, nil
}

func (q *AgentQueries) Create(ctx context.Context, userID string, req model.CreateAgentRequest) (*model.Agent, error) {
	cfg, _ := json.Marshal(req.Config)
	row := q.pool.QueryRow(ctx,
		`INSERT INTO agents (user_id, slug, name, description, system_prompt, model, config, preset)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING `+agentCols,
		userID, slugify(req.Name), req.Name, req.Description, req.SystemPrompt, defaultStr(req.Model, "auto"), cfg, req.Preset,
	)
	return scanAgent(row)
}

func (q *AgentQueries) Get(ctx context.Context, userID, id string) (*model.Agent, error) {
	a, err := scanAgent(q.pool.QueryRow(ctx,
		`SELECT `+agentCols+` FROM agents WHERE id = $1 AND user_id = $2`, id, userID))
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}
	return a, nil
}

func (q *AgentQueries) ListByUser(ctx context.Context, userID string) ([]model.Agent, error) {
	rows, err := q.pool.Query(ctx, `SELECT `+agentCols+` FROM agents WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Agent
	for rows.Next() {
		a, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (q *AgentQueries) Update(ctx context.Context, userID, id string, req model.UpdateAgentRequest) (*model.Agent, error) {
	cur, err := q.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		cur.Name = *req.Name
	}
	if req.Description != nil {
		cur.Description = *req.Description
	}
	if req.SystemPrompt != nil {
		cur.SystemPrompt = *req.SystemPrompt
	}
	if req.Model != nil {
		cur.Model = *req.Model
	}
	if req.Config != nil {
		cur.Config = *req.Config
	}
	cfg, _ := json.Marshal(cur.Config)
	row := q.pool.QueryRow(ctx,
		`UPDATE agents SET name=$3, description=$4, system_prompt=$5, model=$6, config=$7, updated_at=now()
		 WHERE id=$1 AND user_id=$2 RETURNING `+agentCols,
		id, userID, cur.Name, cur.Description, cur.SystemPrompt, cur.Model, cfg)
	return scanAgent(row)
}

func (q *AgentQueries) Delete(ctx context.Context, userID, id string) error {
	_, err := q.pool.Exec(ctx, `DELETE FROM agents WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

// ---- data sources ----

const sourceCols = `id, agent_id, user_id, kind, name, status, meta, chunk_count, created_at`

func scanSource(row interface{ Scan(dest ...any) error }) (*model.AgentDataSource, error) {
	var s model.AgentDataSource
	var meta []byte
	if err := row.Scan(&s.ID, &s.AgentID, &s.UserID, &s.Kind, &s.Name, &s.Status, &meta, &s.ChunkCount, &s.CreatedAt); err != nil {
		return nil, err
	}
	if len(meta) > 0 {
		_ = json.Unmarshal(meta, &s.Meta)
	}
	return &s, nil
}

func (q *AgentQueries) CreateSource(ctx context.Context, userID, agentID, kind, name string, meta map[string]any) (*model.AgentDataSource, error) {
	mb, _ := json.Marshal(meta)
	row := q.pool.QueryRow(ctx,
		`INSERT INTO agent_data_sources (agent_id, user_id, kind, name, meta)
		 VALUES ($1,$2,$3,$4,$5) RETURNING `+sourceCols,
		agentID, userID, kind, name, mb)
	return scanSource(row)
}

func (q *AgentQueries) ListSources(ctx context.Context, agentID string) ([]model.AgentDataSource, error) {
	rows, err := q.pool.Query(ctx, `SELECT `+sourceCols+` FROM agent_data_sources WHERE agent_id=$1 ORDER BY created_at DESC`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.AgentDataSource
	for rows.Next() {
		s, err := scanSource(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (q *AgentQueries) GetSource(ctx context.Context, userID, id string) (*model.AgentDataSource, error) {
	return scanSource(q.pool.QueryRow(ctx, `SELECT `+sourceCols+` FROM agent_data_sources WHERE id=$1 AND user_id=$2`, id, userID))
}

func (q *AgentQueries) SetSourceStatus(ctx context.Context, id, status string, chunkCount int) error {
	_, err := q.pool.Exec(ctx, `UPDATE agent_data_sources SET status=$2, chunk_count=$3 WHERE id=$1`, id, status, chunkCount)
	return err
}

func (q *AgentQueries) DeleteSource(ctx context.Context, userID, id string) error {
	_, err := q.pool.Exec(ctx, `DELETE FROM agent_data_sources WHERE id=$1 AND user_id=$2`, id, userID)
	return err
}

// ---- documents (chunks) ----

func (q *AgentQueries) InsertDocument(ctx context.Context, d model.AgentDocument) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO agent_documents (data_source_id, agent_id, user_id, title, chunk_index, content, embedding)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		d.DataSourceID, d.AgentID, d.UserID, d.Title, d.ChunkIndex, d.Content, d.Embedding)
	return err
}

// ListVectors loads all embedded chunks for an agent for in-process search.
func (q *AgentQueries) ListVectors(ctx context.Context, agentID string) ([]model.AgentDocument, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, data_source_id, title, chunk_index, content, embedding FROM agent_documents
		 WHERE agent_id=$1 AND embedding IS NOT NULL`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.AgentDocument
	for rows.Next() {
		var d model.AgentDocument
		if err := rows.Scan(&d.ID, &d.DataSourceID, &d.Title, &d.ChunkIndex, &d.Content, &d.Embedding); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// SearchTrigram is the keyword fallback over chunk content.
func (q *AgentQueries) SearchTrigram(ctx context.Context, agentID, query string, limit int) ([]model.AgentDocument, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, data_source_id, title, chunk_index, content FROM agent_documents
		 WHERE agent_id=$1 AND (content ILIKE '%'||$2||'%' OR content % $2)
		 ORDER BY similarity(content, $2) DESC LIMIT $3`, agentID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.AgentDocument
	for rows.Next() {
		var d model.AgentDocument
		if err := rows.Scan(&d.ID, &d.DataSourceID, &d.Title, &d.ChunkIndex, &d.Content); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ---- runs ----

func (q *AgentQueries) InsertRun(ctx context.Context, r model.AgentRun) (string, error) {
	steps, _ := json.Marshal(r.Steps)
	var id string
	err := q.pool.QueryRow(ctx,
		`INSERT INTO agent_runs (agent_id, user_id, input, output, steps, status, cost_cents)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		r.AgentID, r.UserID, r.Input, r.Output, steps, r.Status, r.CostCents).Scan(&id)
	return id, err
}

func (q *AgentQueries) ListRuns(ctx context.Context, agentID string, limit int) ([]model.AgentRun, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT id, agent_id, input, output, steps, status, cost_cents, created_at FROM agent_runs
		 WHERE agent_id=$1 ORDER BY created_at DESC LIMIT $2`, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.AgentRun
	for rows.Next() {
		var r model.AgentRun
		var steps []byte
		if err := rows.Scan(&r.ID, &r.AgentID, &r.Input, &r.Output, &steps, &r.Status, &r.CostCents, &r.CreatedAt); err != nil {
			return nil, err
		}
		if len(steps) > 0 {
			_ = json.Unmarshal(steps, &r.Steps)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
