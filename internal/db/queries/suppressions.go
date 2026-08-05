package queries

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SuppressionQueries struct {
	pool *pgxpool.Pool
}

func NewSuppressionQueries(pool *pgxpool.Pool) *SuppressionQueries {
	return &SuppressionQueries{pool: pool}
}

func (q *SuppressionQueries) Add(ctx context.Context, email, reason string) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO email_suppressions (email, reason) VALUES (LOWER($1), $2) ON CONFLICT (email) DO NOTHING`,
		strings.TrimSpace(email), reason)
	return err
}

func (q *SuppressionQueries) IsSuppressed(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := q.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM email_suppressions WHERE email = LOWER($1))`,
		strings.TrimSpace(email)).Scan(&exists)
	return exists, err
}
