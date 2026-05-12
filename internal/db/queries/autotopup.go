package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AutotopupQueries struct {
	pool *pgxpool.Pool
}

func NewAutotopupQueries(pool *pgxpool.Pool) *AutotopupQueries {
	return &AutotopupQueries{pool: pool}
}

type AutotopupCharge struct {
	UserID                string
	AmountCents           int64
	AmountUSD             float64
	StripePaymentIntentID string
	Status                string
	Error                 string
	CreatedAt             time.Time
}

func (q *AutotopupQueries) LogCharge(ctx context.Context, userID string, amountCents int64, amountUSD float64, stripePI, status, errMsg string) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO autotopup_charges (user_id, amount_cents, amount_usd, stripe_payment_intent_id, status, error)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, amountCents, amountUSD, stripePI, status, errMsg,
	)
	if err != nil {
		return fmt.Errorf("log autotopup charge: %w", err)
	}
	return nil
}

func (q *AutotopupQueries) LastChargeForUser(ctx context.Context, userID string) (*AutotopupCharge, error) {
	var c AutotopupCharge
	err := q.pool.QueryRow(ctx,
		`SELECT user_id, amount_cents, amount_usd, COALESCE(stripe_payment_intent_id, ''), status, COALESCE(error, ''), created_at
		 FROM autotopup_charges
		 WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT 1`,
		userID,
	).Scan(&c.UserID, &c.AmountCents, &c.AmountUSD, &c.StripePaymentIntentID, &c.Status, &c.Error, &c.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("last autotopup charge: %w", err)
	}
	return &c, nil
}
