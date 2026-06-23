package queries

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpaths/openpaths/internal/model"
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
	case "90d":
		return 90 * 24 * time.Hour
	case "180d":
		return 180 * 24 * time.Hour
	case "365d", "1y":
		return 365 * 24 * time.Hour
	default:
		return 24 * time.Hour
	}
}

// ClassifyProduct maps a model name to a coarse "product" (capability/modality):
// chat, image, video, music, speech, transcription, embedding, or 3d.
// Order matters: more specific buckets (3d, video) are checked before image
// because their model names often contain "image" (e.g. image-to-3d,
// image-to-video). Anything unmatched falls back to "chat".
func ClassifyProduct(modelName string) string {
	m := strings.ToLower(modelName)
	has := func(subs ...string) bool {
		for _, s := range subs {
			if strings.Contains(m, s) {
				return true
			}
		}
		return false
	}
	switch {
	case has("3d", "trellis", "meshy", "tripo", "rig", "rodin", "hunyuan3d", "pixal3d"):
		return "3d"
	case has("video", "veo", "sora", "kling", "hailuo", "seedance", "ltx", "wan", "runway", "mochi", "pika", "ra2v", "happy-horse", "auto-video"):
		return "video"
	case has("whisper", "-stt", "transcrib", "auto-transcription"):
		return "transcription"
	case has("tts", "-voice", "voice-", "speech", "speak", "elevenlabs", "auto-speech"):
		return "speech"
	case has("lyria", "suno", "udio", "music", "auto-music"):
		return "music"
	case has("embed", "auto-embedding"):
		return "embedding"
	case has("flux", "image", "dall", "sdxl", "stable-diffusion", "imagen", "ideogram",
		"recraft", "nano-banana", "qwen-image", "seedream", "glm-image", "zimage",
		"hidream", "klein", "photon", "smart-resize", "ra1", "auto-image", "auto-img"):
		return "image"
	default:
		return "chat"
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

func (q *StatsQueries) GetUsageBreakdown(ctx context.Context, period string) ([]model.UsageBreakdown, error) {
	since := time.Now().Add(-parsePeriod(period))

	rows, err := q.pool.Query(ctx,
		`SELECT
			CASE
				WHEN model IN ('openpaths/auto', 'auto', 'auto-chat', 'auto-text') THEN 'openpaths/auto'
				WHEN model IN ('openpaths/auto-cheap', 'auto-cheap', 'auto-easy-task', 'auto-easy') THEN 'openpaths/auto-cheap'
				WHEN model IN ('openpaths/auto-fast', 'auto-fast') THEN 'openpaths/auto-fast'
				WHEN model IN ('openpaths/auto-code', 'auto-code', 'auto-medium-task', 'auto-medium') THEN 'openpaths/auto-code'
				WHEN model IN ('openpaths/auto-reasoning', 'auto-reasoning', 'auto-think', 'auto-think-task', 'autothink', 'auto-hard', 'auto-hard-task', 'auto-opus') THEN 'openpaths/auto-reasoning'
				WHEN model IN ('openpaths/auto-vision', 'auto-vision') THEN 'openpaths/auto-vision'
				WHEN model IN ('openpaths/auto-image', 'auto-image', 'auto-img') THEN 'openpaths/auto-image'
				WHEN model IN ('auto-video', 'auto-music', 'auto-speech', 'auto-transcription', 'auto-embedding') THEN model
				ELSE 'direct'
			END AS task,
			provider,
			model,
			COUNT(*)::bigint AS total_requests,
			COALESCE(SUM(tokens_in), 0)::bigint AS total_tokens_in,
			COALESCE(SUM(tokens_out), 0)::bigint AS total_tokens_out,
			COALESCE(SUM(cost_cents), 0)::bigint AS total_cost_cents,
			COALESCE(AVG(latency_ms), 0) AS avg_latency_ms,
			COALESCE(SUM(CASE WHEN error IS NOT NULL THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0) AS error_rate
		 FROM usage_logs
		 WHERE created_at >= $1
		 GROUP BY task, provider, model
		 ORDER BY total_requests DESC, task ASC, provider ASC, model ASC`,
		since,
	)
	if err != nil {
		return nil, fmt.Errorf("get usage breakdown: %w", err)
	}
	defer rows.Close()

	var stats []model.UsageBreakdown
	for rows.Next() {
		var s model.UsageBreakdown
		if err := rows.Scan(
			&s.Task,
			&s.Provider,
			&s.Model,
			&s.TotalRequests,
			&s.TotalTokensIn,
			&s.TotalTokensOut,
			&s.TotalCostCents,
			&s.AvgLatencyMs,
			&s.ErrorRate,
		); err != nil {
			return nil, fmt.Errorf("scan usage breakdown: %w", err)
		}
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate usage breakdown: %w", err)
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

func (q *StatsQueries) GetModelDailyUsage(ctx context.Context, period string, limit int) ([]model.ModelDailyUsagePoint, error) {
	since := time.Now().Add(-parsePeriod(period))
	if limit <= 0 || limit > 12 {
		limit = 8
	}

	rows, err := q.pool.Query(ctx,
		`WITH top_models AS (
			SELECT model, provider, COUNT(*) AS total_requests
			FROM usage_logs
			WHERE created_at >= $1
			GROUP BY model, provider
			ORDER BY total_requests DESC
			LIMIT $2
		)
		SELECT
			date_trunc('day', ul.created_at)::date AS day,
			ul.model,
			ul.provider,
			COUNT(*)::bigint AS requests
		FROM usage_logs ul
		JOIN top_models tm ON tm.model = ul.model AND tm.provider = ul.provider
		WHERE ul.created_at >= $1
		GROUP BY day, ul.model, ul.provider
		ORDER BY day ASC, requests DESC, ul.model ASC`,
		since, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get model daily usage: %w", err)
	}
	defer rows.Close()

	var points []model.ModelDailyUsagePoint
	for rows.Next() {
		var p model.ModelDailyUsagePoint
		if err := rows.Scan(&p.Date, &p.Model, &p.Provider, &p.Requests); err != nil {
			return nil, fmt.Errorf("scan model daily usage: %w", err)
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate model daily usage: %w", err)
	}
	return points, nil
}

func (q *StatsQueries) GetUserSpendByAPIKey(ctx context.Context, userID, period string) ([]model.APIKeySpend, error) {
	since := time.Now().Add(-parsePeriod(period))
	rows, err := q.pool.Query(ctx,
		`SELECT ul.api_key_id, ak.key_prefix, ak.name,
			COUNT(*) as total_requests,
			COALESCE(SUM(ul.cost_cents), 0) as total_cost_cents
		 FROM usage_logs ul
		 JOIN api_keys ak ON ak.id = ul.api_key_id
		 WHERE ul.user_id = $1 AND ul.created_at >= $2
		 GROUP BY ul.api_key_id, ak.key_prefix, ak.name
		 ORDER BY total_cost_cents DESC`,
		userID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("get user spend by api key: %w", err)
	}
	defer rows.Close()

	var result []model.APIKeySpend
	for rows.Next() {
		var s model.APIKeySpend
		if err := rows.Scan(&s.APIKeyID, &s.KeyPrefix, &s.KeyName, &s.TotalRequests, &s.TotalCostCents); err != nil {
			return nil, fmt.Errorf("scan api key spend: %w", err)
		}
		result = append(result, s)
	}
	return result, nil
}

func (q *StatsQueries) GetUserSpendByProvider(ctx context.Context, userID, period string) ([]model.ProviderSpend, error) {
	since := time.Now().Add(-parsePeriod(period))
	rows, err := q.pool.Query(ctx,
		`SELECT provider,
			COUNT(*) as total_requests,
			COALESCE(SUM(cost_cents), 0) as total_cost_cents
		 FROM usage_logs
		 WHERE user_id = $1 AND created_at >= $2
		 GROUP BY provider
		 ORDER BY total_cost_cents DESC`,
		userID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("get user spend by provider: %w", err)
	}
	defer rows.Close()

	var result []model.ProviderSpend
	for rows.Next() {
		var s model.ProviderSpend
		if err := rows.Scan(&s.Provider, &s.TotalRequests, &s.TotalCostCents); err != nil {
			return nil, fmt.Errorf("scan provider spend: %w", err)
		}
		result = append(result, s)
	}
	return result, nil
}

// GetUserSpendByProduct groups usage by model in SQL, then folds each model
// into its product bucket (via ClassifyProduct) in Go so the classification
// has a single, unit-testable source of truth.
func (q *StatsQueries) GetUserSpendByProduct(ctx context.Context, userID, period string) ([]model.ProductSpend, error) {
	since := time.Now().Add(-parsePeriod(period))
	rows, err := q.pool.Query(ctx,
		`SELECT model,
			COUNT(*) AS total_requests,
			COALESCE(SUM(tokens_in), 0) AS total_tokens_in,
			COALESCE(SUM(tokens_out), 0) AS total_tokens_out,
			COALESCE(SUM(cost_cents), 0) AS total_cost_cents
		 FROM usage_logs
		 WHERE user_id = $1 AND created_at >= $2
		 GROUP BY model`,
		userID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("get user spend by product: %w", err)
	}
	defer rows.Close()

	byProduct := make(map[string]*model.ProductSpend)
	for rows.Next() {
		var modelName string
		var reqs, tokIn, tokOut, cost int64
		if err := rows.Scan(&modelName, &reqs, &tokIn, &tokOut, &cost); err != nil {
			return nil, fmt.Errorf("scan product spend: %w", err)
		}
		p := ClassifyProduct(modelName)
		agg := byProduct[p]
		if agg == nil {
			agg = &model.ProductSpend{Product: p}
			byProduct[p] = agg
		}
		agg.TotalRequests += reqs
		agg.TotalTokensIn += tokIn
		agg.TotalTokensOut += tokOut
		agg.TotalCostCents += cost
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product spend: %w", err)
	}

	result := make([]model.ProductSpend, 0, len(byProduct))
	for _, s := range byProduct {
		result = append(result, *s)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].TotalCostCents != result[j].TotalCostCents {
			return result[i].TotalCostCents > result[j].TotalCostCents
		}
		return result[i].TotalRequests > result[j].TotalRequests
	})
	return result, nil
}

// GetUserDailyActivity returns per-day request counts and spend for the last
// `days` days, used to render a GitHub-style contribution heatmap.
func (q *StatsQueries) GetUserDailyActivity(ctx context.Context, userID string, days int) ([]model.DailyActivityPoint, error) {
	if days <= 0 || days > 366 {
		days = 365
	}
	since := time.Now().AddDate(0, 0, -days)
	rows, err := q.pool.Query(ctx,
		`SELECT to_char(date_trunc('day', created_at), 'YYYY-MM-DD') AS day,
			COUNT(*) AS total_requests,
			COALESCE(SUM(cost_cents), 0) AS total_cost_cents
		 FROM usage_logs
		 WHERE user_id = $1 AND created_at >= $2
		 GROUP BY day
		 ORDER BY day ASC`,
		userID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("get user daily activity: %w", err)
	}
	defer rows.Close()

	var result []model.DailyActivityPoint
	for rows.Next() {
		var p model.DailyActivityPoint
		if err := rows.Scan(&p.Date, &p.TotalRequests, &p.TotalCostCents); err != nil {
			return nil, fmt.Errorf("scan daily activity: %w", err)
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate daily activity: %w", err)
	}
	return result, nil
}

func (q *StatsQueries) GetUserTimeSeries(ctx context.Context, userID, period, interval, metricName string) ([]model.TimeSeriesPoint, error) {
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
	case "cost":
		metricExpr = "COALESCE(SUM(cost_cents), 0)"
	case "requests":
		metricExpr = "COUNT(*)"
	case "latency":
		metricExpr = "COALESCE(AVG(latency_ms), 0)"
	case "tokens":
		metricExpr = "COALESCE(SUM(tokens_in + tokens_out), 0)"
	default:
		metricExpr = "COALESCE(SUM(cost_cents), 0)"
	}

	query := fmt.Sprintf(
		`SELECT
			date_trunc('hour', created_at) +
				(EXTRACT(EPOCH FROM created_at - date_trunc('hour', created_at))::int /
				 EXTRACT(EPOCH FROM interval '%s')::int) * interval '%s' as bucket,
			%s as value
		 FROM usage_logs
		 WHERE user_id = $1 AND created_at >= $2
		 GROUP BY bucket ORDER BY bucket ASC`,
		intervalSQL, intervalSQL, metricExpr,
	)

	rows, err := q.pool.Query(ctx, query, userID, since)
	if err != nil {
		return nil, fmt.Errorf("get user time series: %w", err)
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

func (q *StatsQueries) GetUserAPIKeyDrilldown(ctx context.Context, userID, apiKeyID, period string) ([]model.ModelStats, error) {
	since := time.Now().Add(-parsePeriod(period))
	rows, err := q.pool.Query(ctx,
		`SELECT model, provider,
			COUNT(*),
			COALESCE(SUM(tokens_in), 0),
			COALESCE(SUM(tokens_out), 0),
			COALESCE(AVG(latency_ms), 0),
			0, 0, 0,
			COALESCE(AVG(tps), 0),
			COALESCE(SUM(CASE WHEN error IS NOT NULL THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0),
			COALESCE(SUM(cost_cents), 0)
		 FROM usage_logs
		 WHERE user_id = $1 AND api_key_id = $2 AND created_at >= $3
		 GROUP BY model, provider
		 ORDER BY SUM(cost_cents) DESC`,
		userID, apiKeyID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("get api key drilldown: %w", err)
	}
	defer rows.Close()

	var stats []model.ModelStats
	for rows.Next() {
		var s model.ModelStats
		if err := rows.Scan(&s.Model, &s.Provider, &s.TotalRequests,
			&s.TotalTokensIn, &s.TotalTokensOut, &s.AvgLatencyMs,
			&s.P50LatencyMs, &s.P95LatencyMs, &s.P99LatencyMs,
			&s.AvgTPS, &s.ErrorRate, &s.TotalCostCents); err != nil {
			return nil, fmt.Errorf("scan drilldown: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, nil
}

func (q *StatsQueries) GetUserProviderDrilldown(ctx context.Context, userID, prov, period string) ([]model.ModelStats, error) {
	since := time.Now().Add(-parsePeriod(period))
	rows, err := q.pool.Query(ctx,
		`SELECT model, provider,
			COUNT(*),
			COALESCE(SUM(tokens_in), 0),
			COALESCE(SUM(tokens_out), 0),
			COALESCE(AVG(latency_ms), 0),
			0, 0, 0,
			COALESCE(AVG(tps), 0),
			COALESCE(SUM(CASE WHEN error IS NOT NULL THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0),
			COALESCE(SUM(cost_cents), 0)
		 FROM usage_logs
		 WHERE user_id = $1 AND provider = $2 AND created_at >= $3
		 GROUP BY model, provider
		 ORDER BY SUM(cost_cents) DESC`,
		userID, prov, since,
	)
	if err != nil {
		return nil, fmt.Errorf("get provider drilldown: %w", err)
	}
	defer rows.Close()

	var stats []model.ModelStats
	for rows.Next() {
		var s model.ModelStats
		if err := rows.Scan(&s.Model, &s.Provider, &s.TotalRequests,
			&s.TotalTokensIn, &s.TotalTokensOut, &s.AvgLatencyMs,
			&s.P50LatencyMs, &s.P95LatencyMs, &s.P99LatencyMs,
			&s.AvgTPS, &s.ErrorRate, &s.TotalCostCents); err != nil {
			return nil, fmt.Errorf("scan drilldown: %w", err)
		}
		stats = append(stats, s)
	}
	return stats, nil
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
