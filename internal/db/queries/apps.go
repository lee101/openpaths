package queries

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openpaths/openpaths/internal/model"
)

type AppQueries struct {
	pool *pgxpool.Pool
}

func NewAppQueries(pool *pgxpool.Pool) *AppQueries {
	return &AppQueries{pool: pool}
}

func (q *AppQueries) Upsert(ctx context.Context, app *model.AIApp) (string, error) {
	if app == nil {
		return "", fmt.Errorf("app is nil")
	}
	app.URL = NormalizeAppURL(app.URL)
	app.Name = truncateAppField(app.Name, 160)
	app.Description = truncateAppField(app.Description, 600)
	app.FaviconURL = truncateAppField(app.FaviconURL, 2048)
	app.ExternalPath = truncateAppField(app.ExternalPath, 512)
	if app.Name == "" {
		app.Name = app.URL
	}
	if app.Slug == "" {
		app.Slug = SlugForApp(app.Name, app.URL)
	}
	if app.Source == "" {
		app.Source = "openpaths"
	}
	cats, _ := json.Marshal(app.Categories)

	var id string
	err := q.pool.QueryRow(ctx,
		`INSERT INTO ai_apps (slug, name, url, description, favicon_url, categories, source, external_path, total_tokens, last_seen_at)
		 VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, now())
		 ON CONFLICT (slug) DO UPDATE SET
			name=EXCLUDED.name,
			url=CASE WHEN EXCLUDED.url <> '' THEN EXCLUDED.url ELSE ai_apps.url END,
			description=CASE WHEN EXCLUDED.description <> '' THEN EXCLUDED.description ELSE ai_apps.description END,
			favicon_url=CASE WHEN EXCLUDED.favicon_url <> '' THEN EXCLUDED.favicon_url ELSE ai_apps.favicon_url END,
			categories=CASE WHEN EXCLUDED.categories <> '[]'::jsonb THEN EXCLUDED.categories ELSE ai_apps.categories END,
			source=EXCLUDED.source,
			external_path=CASE WHEN EXCLUDED.external_path <> '' THEN EXCLUDED.external_path ELSE ai_apps.external_path END,
			total_tokens=GREATEST(ai_apps.total_tokens, EXCLUDED.total_tokens),
			last_seen_at=now(),
			updated_at=now()
		 RETURNING id::text`,
		app.Slug, app.Name, app.URL, app.Description, app.FaviconURL, string(cats), app.Source, app.ExternalPath, app.TotalTokens,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("upsert app: %w", err)
	}
	return id, nil
}

func (q *AppQueries) UpsertAttribution(ctx context.Context, appURL, title string, categories []string) (string, error) {
	appURL = NormalizeAppURL(appURL)
	title = truncateAppField(title, 160)
	if appURL == "" {
		return "", nil
	}
	if title == "" {
		title = appURL
	}
	return q.Upsert(ctx, &model.AIApp{
		Slug:       SlugForApp("", appURL),
		Name:       title,
		URL:        appURL,
		FaviconURL: FaviconURL(appURL),
		Categories: categories,
		Source:     "openpaths",
	})
}

func (q *AppQueries) UpsertDailyUsage(ctx context.Context, day time.Time, appID, source, modelName, provider string, requests, tokensIn, tokensOut, totalTokens, costCents int64) error {
	_, err := q.pool.Exec(ctx,
		`INSERT INTO app_daily_usage (day, app_id, source, model, provider, requests, tokens_in, tokens_out, total_tokens, cost_cents, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
		 ON CONFLICT (day, app_id, source, model, provider) DO UPDATE SET
			requests=EXCLUDED.requests,
			tokens_in=EXCLUDED.tokens_in,
			tokens_out=EXCLUDED.tokens_out,
			total_tokens=EXCLUDED.total_tokens,
			cost_cents=EXCLUDED.cost_cents,
			updated_at=now()`,
		day, appID, source, modelName, provider, requests, tokensIn, tokensOut, totalTokens, costCents,
	)
	if err != nil {
		return fmt.Errorf("upsert app daily usage: %w", err)
	}
	return nil
}

func (q *AppQueries) AggregateOpenPathsDaily(ctx context.Context, day time.Time) error {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	rows, err := q.pool.Query(ctx,
		`SELECT app_id::text, model, provider, COUNT(*)::bigint,
			COALESCE(SUM(tokens_in), 0)::bigint,
			COALESCE(SUM(tokens_out), 0)::bigint,
			COALESCE(SUM(tokens_in + tokens_out), 0)::bigint,
			COALESCE(SUM(cost_cents), 0)::bigint
		 FROM usage_logs
		 WHERE app_id IS NOT NULL AND created_at >= $1 AND created_at < $2
		 GROUP BY app_id, model, provider`,
		start, end,
	)
	if err != nil {
		return fmt.Errorf("aggregate openpaths app usage: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var appID, modelName, provider string
		var requests, tokensIn, tokensOut, totalTokens, costCents int64
		if err := rows.Scan(&appID, &modelName, &provider, &requests, &tokensIn, &tokensOut, &totalTokens, &costCents); err != nil {
			return fmt.Errorf("scan app aggregate: %w", err)
		}
		if err := q.UpsertDailyUsage(ctx, start, appID, "openpaths", modelName, provider, requests, tokensIn, tokensOut, totalTokens, costCents); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (q *AppQueries) ListAppStats(ctx context.Context, period string, limit int) ([]model.AppUsageStats, error) {
	since := time.Now().Add(-parsePeriod(period))
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	rows, err := q.pool.Query(ctx,
		`WITH live AS (
			SELECT app_id, 'openpaths'::text AS source, model, provider, COUNT(*)::bigint AS requests,
				COALESCE(SUM(tokens_in), 0)::bigint AS tokens_in,
				COALESCE(SUM(tokens_out), 0)::bigint AS tokens_out,
				COALESCE(SUM(tokens_in + tokens_out), 0)::bigint AS total_tokens
			FROM usage_logs
			WHERE app_id IS NOT NULL AND created_at >= $1
			GROUP BY app_id, model, provider
		), snapshots AS (
			SELECT app_id, source, model, provider, SUM(requests)::bigint AS requests,
				SUM(tokens_in)::bigint AS tokens_in,
				SUM(tokens_out)::bigint AS tokens_out,
				SUM(total_tokens)::bigint AS total_tokens
			FROM app_daily_usage
			WHERE day >= $1::date AND source <> 'openpaths'
			GROUP BY app_id, source, model, provider
		), usage AS (
			SELECT * FROM live
			UNION ALL
			SELECT * FROM snapshots
		), ranked AS (
			SELECT app_id, SUM(requests)::bigint AS total_requests, SUM(total_tokens)::bigint AS total_tokens
			FROM usage
			GROUP BY app_id
			ORDER BY total_tokens DESC, total_requests DESC
			LIMIT $2
		)
		SELECT a.id::text, a.slug, a.name, a.url, a.description, a.favicon_url, a.categories, a.source,
			r.total_requests, GREATEST(r.total_tokens, a.total_tokens)::bigint,
			u.model, u.provider, u.source, SUM(u.requests)::bigint, SUM(u.tokens_in)::bigint, SUM(u.tokens_out)::bigint, SUM(u.total_tokens)::bigint
		FROM ranked r
		JOIN ai_apps a ON a.id = r.app_id
		JOIN usage u ON u.app_id = r.app_id
		GROUP BY a.id, a.slug, a.name, a.url, a.description, a.favicon_url, a.categories, a.source,
			r.total_requests, r.total_tokens, a.total_tokens, u.model, u.provider, u.source
		ORDER BY GREATEST(r.total_tokens, a.total_tokens) DESC, a.name ASC, SUM(u.total_tokens) DESC`,
		since, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list app stats: %w", err)
	}
	defer rows.Close()

	byID := make(map[string]*model.AppUsageStats)
	var ordered []string
	for rows.Next() {
		var app model.AppUsageStats
		var categories []byte
		var usage model.AppModelUsage
		if err := rows.Scan(&app.AppID, &app.Slug, &app.Name, &app.URL, &app.Description, &app.FaviconURL, &categories, &app.Source,
			&app.TotalRequests, &app.TotalTokens, &usage.Model, &usage.Provider, &usage.Source, &usage.Requests, &usage.TokensIn, &usage.TokensOut, &usage.TotalTokens); err != nil {
			return nil, fmt.Errorf("scan app stats: %w", err)
		}
		_ = json.Unmarshal(categories, &app.Categories)
		existing := byID[app.AppID]
		if existing == nil {
			existing = &app
			byID[app.AppID] = existing
			ordered = append(ordered, app.AppID)
		}
		existing.Models = append(existing.Models, usage)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate app stats: %w", err)
	}

	stats := make([]model.AppUsageStats, 0, len(ordered))
	for _, id := range ordered {
		stats = append(stats, *byID[id])
	}
	return stats, nil
}

func (q *AppQueries) GetAppStats(ctx context.Context, slug, period string) (*model.AppUsageStats, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, pgx.ErrNoRows
	}
	since := time.Now().Add(-parsePeriod(period))
	rows, err := q.pool.Query(ctx,
		`WITH target AS (
			SELECT * FROM ai_apps WHERE slug = $2
		), live AS (
			SELECT app_id, 'openpaths'::text AS source, model, provider, COUNT(*)::bigint AS requests,
				COALESCE(SUM(tokens_in), 0)::bigint AS tokens_in,
				COALESCE(SUM(tokens_out), 0)::bigint AS tokens_out,
				COALESCE(SUM(tokens_in + tokens_out), 0)::bigint AS total_tokens
			FROM usage_logs
			WHERE app_id IN (SELECT id FROM target) AND created_at >= $1
			GROUP BY app_id, model, provider
		), snapshots AS (
			SELECT app_id, source, model, provider, SUM(requests)::bigint AS requests,
				SUM(tokens_in)::bigint AS tokens_in,
				SUM(tokens_out)::bigint AS tokens_out,
				SUM(total_tokens)::bigint AS total_tokens
			FROM app_daily_usage
			WHERE app_id IN (SELECT id FROM target) AND day >= $1::date AND source <> 'openpaths'
			GROUP BY app_id, source, model, provider
		), usage AS (
			SELECT * FROM live
			UNION ALL
			SELECT * FROM snapshots
		), model_usage AS (
			SELECT app_id, source, model, provider, SUM(requests)::bigint AS requests,
				SUM(tokens_in)::bigint AS tokens_in,
				SUM(tokens_out)::bigint AS tokens_out,
				SUM(total_tokens)::bigint AS total_tokens
			FROM usage
			GROUP BY app_id, source, model, provider
		), totals AS (
			SELECT app_id, SUM(requests)::bigint AS requests, SUM(total_tokens)::bigint AS total_tokens
			FROM model_usage
			GROUP BY app_id
		)
		SELECT a.id::text, a.slug, a.name, a.url, a.description, a.favicon_url, a.categories, a.source,
			COALESCE(t.requests, 0)::bigint, GREATEST(COALESCE(t.total_tokens, 0), a.total_tokens)::bigint,
			COALESCE(u.model, ''), COALESCE(u.provider, ''), COALESCE(u.source, ''),
			COALESCE(u.requests, 0)::bigint, COALESCE(u.tokens_in, 0)::bigint, COALESCE(u.tokens_out, 0)::bigint, COALESCE(u.total_tokens, 0)::bigint
		FROM target a
		LEFT JOIN totals t ON t.app_id = a.id
		LEFT JOIN model_usage u ON u.app_id = a.id
		ORDER BY u.total_tokens DESC NULLS LAST`,
		since, slug,
	)
	if err != nil {
		return nil, fmt.Errorf("get app stats: %w", err)
	}
	defer rows.Close()

	var app *model.AppUsageStats
	for rows.Next() {
		var row model.AppUsageStats
		var categories []byte
		var usage model.AppModelUsage
		if err := rows.Scan(&row.AppID, &row.Slug, &row.Name, &row.URL, &row.Description, &row.FaviconURL, &categories, &row.Source,
			&row.TotalRequests, &row.TotalTokens, &usage.Model, &usage.Provider, &usage.Source, &usage.Requests, &usage.TokensIn, &usage.TokensOut, &usage.TotalTokens); err != nil {
			return nil, fmt.Errorf("scan app stats: %w", err)
		}
		if app == nil {
			_ = json.Unmarshal(categories, &row.Categories)
			app = &row
		}
		if usage.TotalTokens > 0 || usage.Requests > 0 || usage.Model != "" {
			app.Models = append(app.Models, usage)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate app stats: %w", err)
	}
	if app == nil {
		return nil, pgx.ErrNoRows
	}
	return app, nil
}

var slugCleaner = regexp.MustCompile(`[^a-z0-9]+`)

func SlugForApp(name, rawURL string) string {
	base := strings.ToLower(strings.TrimSpace(name))
	if base == "" {
		base = rawURL
	}
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" && (base == rawURL || strings.Contains(base, "://")) {
		base = u.Host
	}
	base = slugCleaner.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "app"
	}
	return base
}

func FaviconURL(rawURL string) string {
	rawURL = NormalizeAppURL(rawURL)
	if rawURL == "" {
		return ""
	}
	return "https://t0.gstatic.com/faviconV2?client=SOCIAL&type=FAVICON&fallback_opts=TYPE,SIZE,URL&url=" + url.QueryEscape(rawURL) + "&size=256"
}

func NormalizeAppURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || len(rawURL) > 2048 {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return ""
	}
	u.Fragment = ""
	return u.String()
}

func truncateAppField(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max])
}

func ScanOptionalString(row pgx.Row) (string, error) {
	var value string
	err := row.Scan(&value)
	return value, err
}
