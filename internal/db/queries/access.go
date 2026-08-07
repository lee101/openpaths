package queries

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AccessQueries backs Model IAM: opt-in, open-by-default allow/deny rules over
// model ids, attached per-user and per-team. No matching rule => allowed.
type AccessQueries struct {
	pool *pgxpool.Pool
}

func NewAccessQueries(pool *pgxpool.Pool) *AccessQueries {
	return &AccessQueries{pool: pool}
}

type ModelRule struct {
	ID        string    `json:"id"`
	Scope     string    `json:"scope"`
	ScopeID   string    `json:"scope_id"`
	ModelGlob string    `json:"model_glob"`
	Effect    string    `json:"effect"`
	CreatedAt time.Time `json:"created_at"`
}

func globMatch(glob, modelID string) bool {
	g := strings.ToLower(strings.TrimSpace(glob))
	m := strings.ToLower(strings.TrimSpace(modelID))
	if g == "" {
		return false
	}
	if g == "*" {
		return true
	}
	if strings.HasSuffix(g, "*") {
		return strings.HasPrefix(m, strings.TrimSuffix(g, "*"))
	}
	return g == m
}

func globSpecificity(glob string) int {
	g := strings.ToLower(strings.TrimSpace(glob))
	if g == "*" {
		return 0
	}
	return len(strings.TrimRight(g, "*")) + 1
}

// UserTeamIDs returns every team id the user belongs to.
func (q *AccessQueries) UserTeamIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := q.pool.Query(ctx, "SELECT team_id::text FROM team_members WHERE user_id = $1", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			out = append(out, id)
		}
	}
	return out, rows.Err()
}

// TeamRole returns the user's role in a team ("" if not a member).
func (q *AccessQueries) TeamRole(ctx context.Context, userID, teamID string) (string, error) {
	var role string
	err := q.pool.QueryRow(ctx, "SELECT role FROM team_members WHERE team_id = $1 AND user_id = $2", teamID, userID).Scan(&role)
	if err != nil {
		return "", err
	}
	return role, nil
}

// ModelAllowed evaluates the hierarchical rule set. Default-open: no matching
// rule returns true. Fail-open on any lookup error so policy never hard-blocks
// traffic by accident. Returns a short reason when denied.
func (q *AccessQueries) ModelAllowed(ctx context.Context, userID, modelID string) (bool, string) {
	model := strings.ToLower(strings.TrimSpace(modelID))
	if model == "" || userID == "" {
		return true, ""
	}
	scopeIDs := []string{userID}
	if teams, err := q.UserTeamIDs(ctx, userID); err == nil {
		scopeIDs = append(scopeIDs, teams...)
	}
	rows, err := q.pool.Query(ctx,
		"SELECT model_glob, effect FROM model_rules WHERE scope_id = ANY($1)", scopeIDs)
	if err != nil {
		return true, ""
	}
	defer rows.Close()

	var rules []RuleLite
	for rows.Next() {
		var r RuleLite
		if rows.Scan(&r.Glob, &r.Effect) == nil {
			rules = append(rules, r)
		}
	}
	return DecideModel(rules, model)
}

// RuleLite is the minimal rule shape the evaluator needs.
type RuleLite struct {
	Glob   string
	Effect string
}

// DecideModel applies the rule set to a model id: most-specific-wins, with
// deny-overriding allow at equal specificity. No matching rule => allowed.
// Pure (no DB) so the IAM core is unit-testable.
func DecideModel(rules []RuleLite, model string) (bool, string) {
	bestSpec := -1
	deny := false
	matched := false
	for _, r := range rules {
		if !globMatch(r.Glob, model) {
			continue
		}
		spec := globSpecificity(r.Glob)
		isDeny := strings.EqualFold(r.Effect, "deny")
		switch {
		case spec > bestSpec:
			bestSpec = spec
			deny = isDeny
			matched = true
		case spec == bestSpec && isDeny:
			deny = true
		}
	}
	if matched && deny {
		return false, "model " + strings.ToLower(strings.TrimSpace(model)) + " is blocked by an access policy"
	}
	return true, ""
}

func (q *AccessQueries) ListRules(ctx context.Context, scope, scopeID string) ([]ModelRule, error) {
	rows, err := q.pool.Query(ctx,
		"SELECT id::text, scope, scope_id, model_glob, effect, created_at FROM model_rules WHERE scope = $1 AND scope_id = $2 ORDER BY created_at",
		scope, scopeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ModelRule{}
	for rows.Next() {
		var r ModelRule
		if err := rows.Scan(&r.ID, &r.Scope, &r.ScopeID, &r.ModelGlob, &r.Effect, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *AccessQueries) AddRule(ctx context.Context, scope, scopeID, glob, effect, createdBy string) (string, error) {
	var id string
	err := q.pool.QueryRow(ctx,
		`INSERT INTO model_rules (scope, scope_id, model_glob, effect, created_by)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id::text`,
		scope, scopeID, glob, effect, createdBy).Scan(&id)
	return id, err
}

// RuleScope returns the (scope, scopeID) of a rule for authorization checks.
func (q *AccessQueries) RuleScope(ctx context.Context, id string) (string, string, error) {
	var scope, scopeID string
	err := q.pool.QueryRow(ctx, "SELECT scope, scope_id FROM model_rules WHERE id = $1", id).Scan(&scope, &scopeID)
	return scope, scopeID, err
}

func (q *AccessQueries) DeleteRule(ctx context.Context, id string) error {
	_, err := q.pool.Exec(ctx, "DELETE FROM model_rules WHERE id = $1", id)
	return err
}
