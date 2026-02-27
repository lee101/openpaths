package queries

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpath/openpath/internal/model"
)

type UsageQueries struct {
	pool *pgxpool.Pool
}

func NewUsageQueries(pool *pgxpool.Pool) *UsageQueries {
	return &UsageQueries{pool: pool}
}

func (q *UsageQueries) Insert(ctx context.Context, log *model.UsageLog) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO usage_logs (user_id, api_key_id, model, provider, tokens_in, tokens_out,
		 latency_ms, tps, cost_cents, error, status_code, streaming)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		log.UserID, log.APIKeyID, log.Model, log.Provider, log.TokensIn, log.TokensOut,
		log.LatencyMs, log.TPS, log.CostCents, log.Error, log.StatusCode, log.Streaming,
	)
	if err != nil {
		return fmt.Errorf("insert usage log: %w", err)
	}
	return nil
}

func (q *UsageQueries) GetByUser(ctx context.Context, userID string, limit, offset int) ([]model.UsageLog, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := q.pool.Query(ctx,
		`SELECT id, user_id, api_key_id, model, provider, tokens_in, tokens_out,
		        latency_ms, tps, cost_cents, error, status_code, streaming, created_at
		 FROM usage_logs WHERE user_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("get usage logs: %w", err)
	}
	defer rows.Close()

	var logs []model.UsageLog
	for rows.Next() {
		var l model.UsageLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.APIKeyID, &l.Model, &l.Provider,
			&l.TokensIn, &l.TokensOut, &l.LatencyMs, &l.TPS, &l.CostCents,
			&l.Error, &l.StatusCode, &l.Streaming, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan usage log: %w", err)
		}
		logs = append(logs, l)
	}
	return logs, nil
}
