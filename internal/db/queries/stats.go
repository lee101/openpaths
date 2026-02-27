package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpath/openpath/internal/model"
)

type StatsQueries struct {
	pool *pgxpool.Pool
}

func NewStatsQueries(pool *pgxpool.Pool) *StatsQueries {
	return &StatsQueries{pool: pool}
}

func parsePeriod(period string) time.Duration {
	switch period {
	case "1h":
		return time.Hour
	case "24h":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

func (q *StatsQueries) GetModelStats(ctx context.Context, period string) ([]model.ModelStats, error) {
	since := time.Now().Add(-parsePeriod(period))

	rows, err := q.pool.Query(ctx,
		`SELECT
			model,
			provider,
			COUNT(*) as total_requests,
			COALESCE(SUM(tokens_in), 0) as total_tokens_in,
			COALESCE(SUM(tokens_out), 0) as total_tokens_out,
			COALESCE(AVG(latency_ms), 0) as avg_latency_ms,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY latency_ms), 0) as p50_latency,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms), 0) as p95_latency,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms), 0) as p99_latency,
			COALESCE(AVG(tps), 0) as avg_tps,
			COALESCE(SUM(CASE WHEN error IS NOT NULL THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0) as error_rate,
			COALESCE(SUM(cost_cents), 0) as total_cost_cents
		 FROM usage_logs
		 WHERE created_at >= $1
		 GROUP BY model, provider
		 ORDER BY total_requests DESC`,
		since,
	)
	if err != nil {
		return nil, fmt.Errorf("get model stats: %w", err)
	}
	defer rows.Close()

	var stats []model.ModelStats
	for rows.Next() {
		var s model.ModelStats
		if err := rows.Scan(&s.Model, &s.Provider, &s.TotalRequests,
			&s.TotalTokensIn, &s.TotalTokensOut, &s.AvgLatencyMs,
			&s.P50LatencyMs, &s.P95LatencyMs, &s.P99LatencyMs,
			&s.AvgTPS, &s.ErrorRate, &s.TotalCostCents); err != nil {
			return nil, fmt.Errorf("scan model stats: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, nil
}

func (q *StatsQueries) GetProviderStats(ctx context.Context, period string) ([]model.ModelStats, error) {
	since := time.Now().Add(-parsePeriod(period))

	rows, err := q.pool.Query(ctx,
		`SELECT
			'' as model,
			provider,
			COUNT(*) as total_requests,
			COALESCE(SUM(tokens_in), 0),
			COALESCE(SUM(tokens_out), 0),
			COALESCE(AVG(latency_ms), 0),
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY latency_ms), 0),
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms), 0),
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms), 0),
			COALESCE(AVG(tps), 0),
			COALESCE(SUM(CASE WHEN error IS NOT NULL THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0),
			COALESCE(SUM(cost_cents), 0)
		 FROM usage_logs
		 WHERE created_at >= $1
		 GROUP BY provider
		 ORDER BY total_requests DESC`,
		since,
	)
	if err != nil {
		return nil, fmt.Errorf("get provider stats: %w", err)
	}
	defer rows.Close()

	var stats []model.ModelStats
	for rows.Next() {
		var s model.ModelStats
		if err := rows.Scan(&s.Model, &s.Provider, &s.TotalRequests,
			&s.TotalTokensIn, &s.TotalTokensOut, &s.AvgLatencyMs,
			&s.P50LatencyMs, &s.P95LatencyMs, &s.P99LatencyMs,
			&s.AvgTPS, &s.ErrorRate, &s.TotalCostCents); err != nil {
			return nil, fmt.Errorf("scan provider stats: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, nil
}

func (q *StatsQueries) GetTimeSeries(ctx context.Context, period, interval, metricName string, modelFilter string) ([]model.TimeSeriesPoint, error) {
	since := time.Now().Add(-parsePeriod(period))

	intervalSQL := "1 hour"
	switch interval {
	case "5m":
		intervalSQL = "5 minutes"
	case "15m":
		intervalSQL = "15 minutes"
	case "1h":
		intervalSQL = "1 hour"
	case "6h":
		intervalSQL = "6 hours"
	case "1d":
		intervalSQL = "1 day"
	}

	var metricExpr string
	switch metricName {
	case "latency":
		metricExpr = "AVG(latency_ms)"
	case "tps":
		metricExpr = "AVG(tps)"
	case "requests":
		metricExpr = "COUNT(*)"
	case "error_rate":
		metricExpr = "SUM(CASE WHEN error IS NOT NULL THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0)"
	default:
		metricExpr = "COUNT(*)"
	}

	query := fmt.Sprintf(
		`SELECT
			date_trunc('hour', created_at) +
				(EXTRACT(EPOCH FROM created_at - date_trunc('hour', created_at))::int /
				 EXTRACT(EPOCH FROM interval '%s')::int) * interval '%s' as bucket,
			%s as value
		 FROM usage_logs
		 WHERE created_at >= $1`,
		intervalSQL, intervalSQL, metricExpr,
	)

	args := []any{since}
	if modelFilter != "" {
		query += " AND model = $2"
		args = append(args, modelFilter)
	}

	query += fmt.Sprintf(" GROUP BY bucket ORDER BY bucket ASC")

	rows, err := q.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get time series: %w", err)
	}
	defer rows.Close()

	var points []model.TimeSeriesPoint
	for rows.Next() {
		var p model.TimeSeriesPoint
		if err := rows.Scan(&p.Timestamp, &p.Value); err != nil {
			return nil, fmt.Errorf("scan time series point: %w", err)
		}
		points = append(points, p)
	}
	return points, nil
}

func (q *StatsQueries) GetUserUsage(ctx context.Context, userID, period string) ([]model.ModelStats, error) {
	since := time.Now().Add(-parsePeriod(period))

	rows, err := q.pool.Query(ctx,
		`SELECT
			model,
			provider,
			COUNT(*),
			COALESCE(SUM(tokens_in), 0),
			COALESCE(SUM(tokens_out), 0),
			COALESCE(AVG(latency_ms), 0),
			0, 0, 0,
			COALESCE(AVG(tps), 0),
			COALESCE(SUM(CASE WHEN error IS NOT NULL THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0),
			COALESCE(SUM(cost_cents), 0)
		 FROM usage_logs
		 WHERE user_id = $1 AND created_at >= $2
		 GROUP BY model, provider
		 ORDER BY COUNT(*) DESC`,
		userID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("get user usage: %w", err)
	}
	defer rows.Close()

	var stats []model.ModelStats
	for rows.Next() {
		var s model.ModelStats
		if err := rows.Scan(&s.Model, &s.Provider, &s.TotalRequests,
			&s.TotalTokensIn, &s.TotalTokensOut, &s.AvgLatencyMs,
			&s.P50LatencyMs, &s.P95LatencyMs, &s.P99LatencyMs,
			&s.AvgTPS, &s.ErrorRate, &s.TotalCostCents); err != nil {
			return nil, fmt.Errorf("scan user usage: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, nil
}
