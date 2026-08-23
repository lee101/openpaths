package queries

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Guardrail struct {
	ID               string          `json:"id"`
	UserID           string          `json:"user_id"`
	Name             string          `json:"name"`
	LimitCents       *int64          `json:"limit_cents"`
	ResetInterval    *string         `json:"reset_interval"`
	BudgetActions    []string        `json:"budget_actions"`
	EmailOnViolation bool            `json:"email_on_violation"`
	AllowedModels    []string        `json:"allowed_models"`
	AllowedProviders []string        `json:"allowed_providers"`
	BlockedModels    []string        `json:"blocked_models"`
	BlockedProviders []string        `json:"blocked_providers"`
	PromptInjection  json.RawMessage `json:"prompt_injection"`
	SensitiveInfo    json.RawMessage `json:"sensitive_info"`
	CustomFilters    json.RawMessage `json:"custom_filters"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	// Populated on list/get
	Assignments []GuardrailAssignment `json:"assignments,omitempty"`
	SpendCents  *int64                `json:"spend_cents,omitempty"`
}

type GuardrailAssignment struct {
	GuardrailID string    `json:"guardrail_id"`
	TargetType  string    `json:"target_type"`
	TargetID    string    `json:"target_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type GuardrailEvent struct {
	ID          string          `json:"id"`
	GuardrailID *string         `json:"guardrail_id,omitempty"`
	UserID      string          `json:"user_id"`
	APIKeyID    *string         `json:"api_key_id,omitempty"`
	Stage       string          `json:"stage"`
	Action      string          `json:"action"`
	Detail      json.RawMessage `json:"detail"`
	CreatedAt   time.Time       `json:"created_at"`
}

type GuardrailQueries struct {
	pool *pgxpool.Pool
}

func NewGuardrailQueries(pool *pgxpool.Pool) *GuardrailQueries {
	return &GuardrailQueries{pool: pool}
}

const guardrailCols = `id::text, user_id::text, name, limit_cents, reset_interval, budget_actions,
	email_on_violation, allowed_models, allowed_providers, blocked_models, blocked_providers,
	prompt_injection, sensitive_info, custom_filters, created_at, updated_at`

func scanGuardrail(row pgx.Row) (*Guardrail, error) {
	g := &Guardrail{}
	err := row.Scan(
		&g.ID, &g.UserID, &g.Name, &g.LimitCents, &g.ResetInterval, &g.BudgetActions, &g.EmailOnViolation,
		&g.AllowedModels, &g.AllowedProviders, &g.BlockedModels, &g.BlockedProviders,
		&g.PromptInjection, &g.SensitiveInfo, &g.CustomFilters,
		&g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if g.BudgetActions == nil {
		g.BudgetActions = []string{}
	}
	if g.AllowedModels == nil {
		g.AllowedModels = []string{}
	}
	if g.AllowedProviders == nil {
		g.AllowedProviders = []string{}
	}
	if g.BlockedModels == nil {
		g.BlockedModels = []string{}
	}
	if g.BlockedProviders == nil {
		g.BlockedProviders = []string{}
	}
	if len(g.PromptInjection) == 0 {
		g.PromptInjection = json.RawMessage(`{}`)
	}
	if len(g.SensitiveInfo) == 0 {
		g.SensitiveInfo = json.RawMessage(`{}`)
	}
	if len(g.CustomFilters) == 0 {
		g.CustomFilters = json.RawMessage(`[]`)
	}
	return g, nil
}

func (q *GuardrailQueries) ListByUser(ctx context.Context, userID string) ([]*Guardrail, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT `+guardrailCols+` FROM guardrails WHERE user_id = $1 ORDER BY updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Guardrail
	for rows.Next() {
		g, err := scanGuardrail(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, g := range out {
		as, err := q.ListAssignments(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		g.Assignments = as
	}
	return out, nil
}

func (q *GuardrailQueries) Get(ctx context.Context, id, userID string) (*Guardrail, error) {
	g, err := scanGuardrail(q.pool.QueryRow(ctx,
		`SELECT `+guardrailCols+` FROM guardrails WHERE id = $1 AND user_id = $2`, id, userID))
	if err != nil {
		return nil, err
	}
	as, err := q.ListAssignments(ctx, g.ID)
	if err != nil {
		return nil, err
	}
	g.Assignments = as
	return g, nil
}

func (q *GuardrailQueries) Create(ctx context.Context, g *Guardrail) (*Guardrail, error) {
	normalizeGuardrail(g)
	row := q.pool.QueryRow(ctx, `
		INSERT INTO guardrails (user_id, name, limit_cents, reset_interval, budget_actions,
			email_on_violation, allowed_models, allowed_providers, blocked_models, blocked_providers,
			prompt_injection, sensitive_info, custom_filters)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING `+guardrailCols,
		g.UserID, g.Name, g.LimitCents, g.ResetInterval, g.BudgetActions, g.EmailOnViolation,
		g.AllowedModels, g.AllowedProviders, g.BlockedModels, g.BlockedProviders,
		g.PromptInjection, g.SensitiveInfo, g.CustomFilters,
	)
	return scanGuardrail(row)
}

func (q *GuardrailQueries) Update(ctx context.Context, g *Guardrail) (*Guardrail, error) {
	normalizeGuardrail(g)
	row := q.pool.QueryRow(ctx, `
		UPDATE guardrails SET
			name = $3, limit_cents = $4, reset_interval = $5, budget_actions = $6,
			email_on_violation = $7, allowed_models = $8, allowed_providers = $9,
			blocked_models = $10, blocked_providers = $11,
			prompt_injection = $12, sensitive_info = $13, custom_filters = $14,
			updated_at = now()
		WHERE id = $1 AND user_id = $2
		RETURNING `+guardrailCols,
		g.ID, g.UserID, g.Name, g.LimitCents, g.ResetInterval, g.BudgetActions, g.EmailOnViolation,
		g.AllowedModels, g.AllowedProviders, g.BlockedModels, g.BlockedProviders,
		g.PromptInjection, g.SensitiveInfo, g.CustomFilters,
	)
	return scanGuardrail(row)
}

func (q *GuardrailQueries) Delete(ctx context.Context, id, userID string) error {
	tag, err := q.pool.Exec(ctx, `DELETE FROM guardrails WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (q *GuardrailQueries) ListAssignments(ctx context.Context, guardrailID string) ([]GuardrailAssignment, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT guardrail_id::text, target_type, target_id, created_at
		 FROM guardrail_assignments WHERE guardrail_id = $1 ORDER BY created_at`, guardrailID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []GuardrailAssignment{}
	for rows.Next() {
		var a GuardrailAssignment
		if err := rows.Scan(&a.GuardrailID, &a.TargetType, &a.TargetID, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ReplaceAssignments sets key targets + optional user-default. Clears prior assignments for this policy.
func (q *GuardrailQueries) ReplaceAssignments(ctx context.Context, guardrailID, userID string, keyIDs []string, userDefault bool) error {
	tx, err := q.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var owner string
	if err := tx.QueryRow(ctx, `SELECT user_id::text FROM guardrails WHERE id = $1`, guardrailID).Scan(&owner); err != nil {
		return err
	}
	if owner != userID {
		return pgx.ErrNoRows
	}

	if _, err := tx.Exec(ctx, `DELETE FROM guardrail_assignments WHERE guardrail_id = $1`, guardrailID); err != nil {
		return err
	}

	for _, keyID := range keyIDs {
		keyID = strings.TrimSpace(keyID)
		if keyID == "" {
			continue
		}
		var keyOwner string
		if err := tx.QueryRow(ctx, `SELECT user_id::text FROM api_keys WHERE id = $1 AND NOT revoked`, keyID).Scan(&keyOwner); err != nil {
			return fmt.Errorf("api key %s: %w", keyID, err)
		}
		if keyOwner != userID {
			return fmt.Errorf("api key %s not owned by user", keyID)
		}
		// Steal assignment from any other guardrail for this key.
		if _, err := tx.Exec(ctx, `DELETE FROM guardrail_assignments WHERE target_type = 'api_key' AND target_id = $1`, keyID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO guardrail_assignments (guardrail_id, target_type, target_id) VALUES ($1,'api_key',$2)`,
			guardrailID, keyID); err != nil {
			return err
		}
	}

	if userDefault {
		if _, err := tx.Exec(ctx, `DELETE FROM guardrail_assignments WHERE target_type = 'user' AND target_id = $1`, userID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO guardrail_assignments (guardrail_id, target_type, target_id) VALUES ($1,'user',$2)`,
			guardrailID, userID); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE guardrails SET updated_at = now() WHERE id = $1`, guardrailID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ResolveForRequest returns the key-specific policy, or the account default
// when the key has no specific assignment. The account default is therefore
// inherited by keys created in the future.
func (q *GuardrailQueries) ResolveForRequest(ctx context.Context, userID, apiKeyID string) ([]*Guardrail, error) {
	var out []*Guardrail
	seen := map[string]bool{}

	load := func(targetType, targetID string) error {
		var id string
		err := q.pool.QueryRow(ctx,
			`SELECT guardrail_id::text FROM guardrail_assignments WHERE target_type = $1 AND target_id = $2`,
			targetType, targetID).Scan(&id)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		if seen[id] {
			return nil
		}
		g, err := scanGuardrail(q.pool.QueryRow(ctx,
			`SELECT `+guardrailCols+` FROM guardrails WHERE id = $1 AND user_id = $2`, id, userID))
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		seen[id] = true
		out = append(out, g)
		return nil
	}

	if apiKeyID != "" {
		if err := load("api_key", apiKeyID); err != nil {
			return nil, err
		}
	}
	// A key-specific policy overrides the account default. This is important
	// for accounts that use a restrictive default for all current and future
	// keys but need a separately configured exception for a key.
	if len(out) == 0 {
		if err := load("user", userID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (q *GuardrailQueries) SpendSince(ctx context.Context, userID, apiKeyID string, since time.Time, perKey bool) (int64, error) {
	var spend int64
	if perKey && apiKeyID != "" {
		err := q.pool.QueryRow(ctx,
			`SELECT COALESCE(SUM(cost_cents), 0)::bigint FROM usage_logs
			 WHERE user_id = $1 AND api_key_id = $2 AND created_at >= $3 AND status_code < 400`,
			userID, apiKeyID, since).Scan(&spend)
		return spend, err
	}
	err := q.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(cost_cents), 0)::bigint FROM usage_logs
		 WHERE user_id = $1 AND created_at >= $2 AND status_code < 400`,
		userID, since).Scan(&spend)
	return spend, err
}

func (q *GuardrailQueries) HasBudgetEmailSince(ctx context.Context, guardrailID string, since time.Time) (bool, error) {
	var n int
	err := q.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM guardrail_events
		 WHERE guardrail_id = $1 AND stage = 'budget' AND action = 'email' AND created_at >= $2`,
		guardrailID, since).Scan(&n)
	return n > 0, err
}

func (q *GuardrailQueries) InsertEvent(ctx context.Context, e *GuardrailEvent) error {
	if e.Detail == nil {
		e.Detail = json.RawMessage(`{}`)
	}
	_, err := q.pool.Exec(ctx,
		`INSERT INTO guardrail_events (guardrail_id, user_id, api_key_id, stage, action, detail)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		e.GuardrailID, e.UserID, e.APIKeyID, e.Stage, e.Action, e.Detail)
	return err
}

func (q *GuardrailQueries) ListEvents(ctx context.Context, userID string, limit int) ([]GuardrailEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := q.pool.Query(ctx,
		`SELECT id::text, guardrail_id::text, user_id::text, api_key_id::text, stage, action, detail, created_at
		 FROM guardrail_events WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []GuardrailEvent{}
	for rows.Next() {
		var e GuardrailEvent
		var gid, kid *string
		if err := rows.Scan(&e.ID, &gid, &e.UserID, &kid, &e.Stage, &e.Action, &e.Detail, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.GuardrailID = gid
		e.APIKeyID = kid
		out = append(out, e)
	}
	return out, rows.Err()
}

func normalizeGuardrail(g *Guardrail) {
	g.Name = strings.TrimSpace(g.Name)
	if g.Name == "" {
		g.Name = "Untitled"
	}
	if g.BudgetActions == nil {
		g.BudgetActions = []string{}
	}
	if g.AllowedModels == nil {
		g.AllowedModels = []string{}
	}
	if g.AllowedProviders == nil {
		g.AllowedProviders = []string{}
	}
	if g.BlockedModels == nil {
		g.BlockedModels = []string{}
	}
	if g.BlockedProviders == nil {
		g.BlockedProviders = []string{}
	}
	if len(g.PromptInjection) == 0 {
		g.PromptInjection = json.RawMessage(`{}`)
	}
	if len(g.SensitiveInfo) == 0 {
		g.SensitiveInfo = json.RawMessage(`{}`)
	}
	if len(g.CustomFilters) == 0 {
		g.CustomFilters = json.RawMessage(`[]`)
	}
}
