package queries

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AutotopupQueries struct {
	pool *pgxpool.Pool
}

func NewAutotopupQueries(pool *pgxpool.Pool) *AutotopupQueries {
	return &AutotopupQueries{pool: pool}
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
