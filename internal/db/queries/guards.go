package queries

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GuardQueries backs billshock guards: per-user/team billing alert thresholds
// and a max top-up cap. All values default to 0 (disabled / unlimited).
type GuardQueries struct {
	pool *pgxpool.Pool
}

func NewGuardQueries(pool *pgxpool.Pool) *GuardQueries {
	return &GuardQueries{pool: pool}
}

type UserGuards struct {
	AlertEnabled        bool  `json:"alert_enabled"`
	AlertThresholdCents int64 `json:"alert_threshold_cents"`
	MaxTopupCapCents    int64 `json:"max_topup_cap_cents"`
}

func (q *GuardQueries) GetUserGuards(ctx context.Context, userID string) (UserGuards, error) {
	var g UserGuards
	err := q.pool.QueryRow(ctx,
		`SELECT COALESCE(billing_alert_enabled,false), COALESCE(billing_alert_threshold_cents,0), COALESCE(max_topup_cap_cents,0)
		   FROM users WHERE id = $1`, userID,
	).Scan(&g.AlertEnabled, &g.AlertThresholdCents, &g.MaxTopupCapCents)
	return g, err
}

func (q *GuardQueries) SetUserGuards(ctx context.Context, userID string, g UserGuards) error {
	_, err := q.pool.Exec(ctx,
		`UPDATE users SET billing_alert_enabled = $1, billing_alert_threshold_cents = $2, max_topup_cap_cents = $3, updated_at = now()
		   WHERE id = $4`,
		g.AlertEnabled, g.AlertThresholdCents, g.MaxTopupCapCents, userID)
	return err
}

func (q *GuardQueries) SetTeamGuards(ctx context.Context, teamID string, alertThresholdCents, maxTopupCapCents int64) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO team_billing (team_id, alert_threshold_cents, max_topup_cap_cents, updated_at)
		 VALUES ($1, $2, $3, now())
		 ON CONFLICT (team_id) DO UPDATE SET alert_threshold_cents = EXCLUDED.alert_threshold_cents,
		   max_topup_cap_cents = EXCLUDED.max_topup_cap_cents, updated_at = now()`,
		teamID, alertThresholdCents, maxTopupCapCents)
	return err
}

// EffectiveTopupCapCents returns the smallest non-zero cap across the user's own
// cap and any of their teams' caps. 0 means unlimited.
func (q *GuardQueries) EffectiveTopupCapCents(ctx context.Context, userID string) int64 {
	var cap int64
	consider := func(v int64) {
		if v > 0 && (cap == 0 || v < cap) {
			cap = v
		}
	}
	var userCap int64
	_ = q.pool.QueryRow(ctx, "SELECT COALESCE(max_topup_cap_cents,0) FROM users WHERE id = $1", userID).Scan(&userCap)
	consider(userCap)
	rows, err := q.pool.Query(ctx,
		`SELECT COALESCE(tb.max_topup_cap_cents,0) FROM team_billing tb
		 JOIN team_members m ON m.team_id::text = tb.team_id WHERE m.user_id = $1`, userID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var c int64
			if rows.Scan(&c) == nil {
				consider(c)
			}
		}
	}
	return cap
}

// TopupWithinCap reports whether a top-up of amountCents is allowed; returns the
// effective cap for messaging.
func (q *GuardQueries) TopupWithinCap(ctx context.Context, userID string, amountCents int64) (bool, int64) {
	cap := q.EffectiveTopupCapCents(ctx, userID)
	if cap > 0 && amountCents > cap {
		return false, cap
	}
	return true, cap
}

// AlertState returns the data needed to decide on a low-balance alert.
func (q *GuardQueries) AlertState(ctx context.Context, userID string) (enabled bool, thresholdCents int64, lastAt *time.Time, email string, err error) {
	err = q.pool.QueryRow(ctx,
		`SELECT COALESCE(billing_alert_enabled,false), COALESCE(billing_alert_threshold_cents,0), billing_alert_last_at, COALESCE(email,'')
		   FROM users WHERE id = $1`, userID,
	).Scan(&enabled, &thresholdCents, &lastAt, &email)
	return
}

func (q *GuardQueries) MarkAlerted(ctx context.Context, userID string) error {
	_, err := q.pool.Exec(ctx, "UPDATE users SET billing_alert_last_at = now() WHERE id = $1", userID)
	return err
}
