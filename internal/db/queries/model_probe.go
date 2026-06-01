package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpaths/openpaths/internal/model"
)

type ModelProbeQueries struct {
	pool *pgxpool.Pool
}

func NewModelProbeQueries(pool *pgxpool.Pool) *ModelProbeQueries {
	return &ModelProbeQueries{pool: pool}
}

func (q *ModelProbeQueries) Upsert(ctx context.Context, result model.ModelProbeResult) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO model_probe_results (model, provider, latency_ms, ok, status_code, error, response_preview, probed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (model) DO UPDATE SET
		   provider = EXCLUDED.provider,
		   latency_ms = EXCLUDED.latency_ms,
		   ok = EXCLUDED.ok,
		   status_code = EXCLUDED.status_code,
		   error = EXCLUDED.error,
		   response_preview = EXCLUDED.response_preview,
		   probed_at = EXCLUDED.probed_at`,
		result.Model,
		result.Provider,
		result.LatencyMs,
		result.OK,
		result.StatusCode,
		result.Error,
		result.ResponsePreview,
		result.ProbedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert model probe: %w", err)
	}
	return nil
}

func (q *ModelProbeQueries) List(ctx context.Context) ([]model.ModelProbeResult, error) {
	rows, err := q.pool.Query(ctx,
		`SELECT model, provider, latency_ms, ok, status_code, error, response_preview, probed_at
		 FROM model_probe_results
		 ORDER BY model ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list model probes: %w", err)
	}
	defer rows.Close()

	var out []model.ModelProbeResult
	for rows.Next() {
		var r model.ModelProbeResult
		if err := rows.Scan(
			&r.Model, &r.Provider, &r.LatencyMs, &r.OK, &r.StatusCode,
			&r.Error, &r.ResponsePreview, &r.ProbedAt,
		); err != nil {
			return nil, fmt.Errorf("scan model probe: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (q *ModelProbeQueries) LatestProbedAt(ctx context.Context) (*time.Time, error) {
	var t *time.Time
	err := q.pool.QueryRow(ctx, `SELECT MAX(probed_at) FROM model_probe_results`).Scan(&t)
	if err != nil {
		return nil, fmt.Errorf("latest model probe time: %w", err)
	}
	return t, nil
}
