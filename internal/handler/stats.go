package handler

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

type StatsHandler struct {
	statsQ *queries.StatsQueries
	appQ   *queries.AppQueries
	probeQ *queries.ModelProbeQueries
}

func (h *StatsHandler) HandleAppOGImage(ctx *fasthttp.RequestCtx) {
	slug := toString(ctx.UserValue("slug"))
	if slug == "" || h.appQ == nil {
		ctx.SetStatusCode(404)
		return
	}
	cleanSlug := strings.TrimSuffix(slug, ".svg")
	app, err := h.appQ.GetAppStats(ctx, cleanSlug, "30d")
	if err != nil && cleanSlug == "openpaths-apps" {
		app = &model.AppUsageStats{
			Slug:        "openpaths-apps",
			Name:        "Apps And Agents",
			URL:         "https://openpaths.io/apps",
			Description: "Opt-in app and agent usage stats across OpenPaths and OpenRouter.",
			FaviconURL:  "https://openpaths.io/favicon.ico",
			Source:      "openpaths",
		}
	} else if err != nil {
		ctx.SetStatusCode(404)
		return
	}

	icon := app.FaviconURL
	if icon == "" {
		icon = "https://openpaths.io/favicon.ico"
	}
	appHost := appHost(app.URL)
	if appHost == "" {
		appHost = app.Source
	}
	title := truncateRunes(app.Name, 36)
	description := truncateRunes(app.Description, 105)
	tokens := compactTokens(app.TotalTokens)
	model := "Model usage tracked on OpenPaths"
	if len(app.Models) > 0 && app.Models[0].Model != "" {
		model = truncateRunes(app.Models[0].Model, 44)
	}

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630">
  <rect width="1200" height="630" fill="#030303"/>
  <rect x="56" y="54" width="1088" height="522" rx="28" fill="#0a0a0a" stroke="#202020"/>
  <circle cx="120" cy="122" r="21" fill="#0f172a" stroke="#38bdf8" stroke-width="3"/>
  <path d="M112 122h16M120 114v16" stroke="#e5e7eb" stroke-width="3" stroke-linecap="round"/>
  <text x="156" y="132" fill="#f8fafc" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="30" font-weight="700">OpenPaths</text>
  <text x="56" y="608" fill="#525252" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="18">openpaths.io/apps/%s</text>
  <image href="%s" x="86" y="205" width="116" height="116" preserveAspectRatio="xMidYMid slice"/>
  <text x="230" y="248" fill="#ffffff" font-family="Inter, ui-sans-serif, system-ui, sans-serif" font-size="64" font-weight="800">%s</text>
  <text x="232" y="292" fill="#94a3b8" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="24">%s</text>
  <text x="86" y="384" fill="#cbd5e1" font-family="Inter, ui-sans-serif, system-ui, sans-serif" font-size="30">%s</text>
  <rect x="86" y="448" width="300" height="82" rx="16" fill="#111111" stroke="#262626"/>
  <text x="112" y="482" fill="#64748b" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="18">TOKENS</text>
  <text x="112" y="518" fill="#ffffff" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="34" font-weight="800">%s</text>
  <rect x="414" y="448" width="574" height="82" rx="16" fill="#111111" stroke="#262626"/>
  <text x="440" y="482" fill="#64748b" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="18">TOP MODEL</text>
  <text x="440" y="518" fill="#ffffff" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="26" font-weight="700">%s</text>
  <rect x="1010" y="448" width="88" height="82" rx="16" fill="#083344" stroke="#155e75"/>
  <text x="1030" y="498" fill="#67e8f9" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="28" font-weight="800">APP</text>
</svg>`,
		html.EscapeString(strings.TrimSuffix(slug, ".svg")),
		html.EscapeString(icon),
		html.EscapeString(title),
		html.EscapeString(appHost),
		html.EscapeString(description),
		html.EscapeString(tokens),
		html.EscapeString(model),
	)

	ctx.SetContentType("image/svg+xml; charset=utf-8")
	ctx.Response.Header.Set("Cache-Control", "public, max-age=86400, stale-while-revalidate=604800")
	ctx.SetBodyString(svg)
}

func NewStatsHandler(statsQ *queries.StatsQueries, appQ *queries.AppQueries, probeQ *queries.ModelProbeQueries) *StatsHandler {
	return &StatsHandler{statsQ: statsQ, appQ: appQ, probeQ: probeQ}
}

// HandleModelProbes handles GET /stats/model-probes.
func (h *StatsHandler) HandleModelProbes(ctx *fasthttp.RequestCtx) {
	if h.probeQ == nil {
		writeError(ctx, 500, "server_error", "Model probes are not configured")
		return
	}

	probes, err := h.probeQ.List(ctx)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get model probes")
		return
	}

	var okCount, failCount int
	for _, p := range probes {
		if p.OK {
			okCount++
		} else {
			failCount++
		}
	}

	latest, _ := h.probeQ.LatestProbedAt(ctx)
	setPublicStatsCache(ctx)
	writeJSON(ctx, 200, map[string]any{
		"probes": probes,
		"summary": map[string]any{
			"total":  len(probes),
			"ok":     okCount,
			"failed": failCount,
		},
		"latest_probed_at": latest,
	})
}

// HandleModelStats handles GET /stats/models?period=24h.
func (h *StatsHandler) HandleModelStats(ctx *fasthttp.RequestCtx) {
	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "24h"
	}

	stats, err := h.statsQ.GetModelStats(ctx, period)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get model stats")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"period": period,
		"models": stats,
	})
}

// HandleProviderStats handles GET /stats/providers?period=24h.
func (h *StatsHandler) HandleProviderStats(ctx *fasthttp.RequestCtx) {
	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "24h"
	}

	stats, err := h.statsQ.GetProviderStats(ctx, period)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get provider stats")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"period":    period,
		"providers": stats,
	})
}

// HandleUsageBreakdown handles GET /stats/breakdown?period=24h.
func (h *StatsHandler) HandleUsageBreakdown(ctx *fasthttp.RequestCtx) {
	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "24h"
	}

	stats, err := h.statsQ.GetUsageBreakdown(ctx, period)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get usage breakdown")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"period":    period,
		"breakdown": stats,
	})
}

// HandleTimeSeries handles GET /stats/timeseries?period=24h&interval=1h&metric=latency&model=gpt-4o.
func (h *StatsHandler) HandleTimeSeries(ctx *fasthttp.RequestCtx) {
	period := string(ctx.QueryArgs().Peek("period"))
	interval := string(ctx.QueryArgs().Peek("interval"))
	metric := string(ctx.QueryArgs().Peek("metric"))
	modelFilter := string(ctx.QueryArgs().Peek("model"))

	if period == "" {
		period = "24h"
	}
	if interval == "" {
		interval = "1h"
	}
	if metric == "" {
		metric = "requests"
	}

	points, err := h.statsQ.GetTimeSeries(ctx, period, interval, metric, modelFilter)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get time series")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"period":   period,
		"interval": interval,
		"metric":   metric,
		"data":     points,
	})
}

// HandleModelDailyUsage handles GET /stats/models/timeseries?period=30d&limit=8.
func (h *StatsHandler) HandleModelDailyUsage(ctx *fasthttp.RequestCtx) {
	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "30d"
	}

	limit := ctx.QueryArgs().GetUintOrZero("limit")
	if limit == 0 {
		limit = 8
	}

	points, err := h.statsQ.GetModelDailyUsage(ctx, period, limit)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get model usage over time")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"period": period,
		"limit":  limit,
		"data":   points,
	})
}

func (h *StatsHandler) HandleAppStats(ctx *fasthttp.RequestCtx) {
	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "30d"
	}
	limit := ctx.QueryArgs().GetUintOrZero("limit")
	if limit == 0 {
		limit = 100
	}
	if h.appQ == nil {
		writeError(ctx, 500, "server_error", "App stats are not configured")
		return
	}

	apps, err := h.appQ.ListAppStats(ctx, period, limit)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get app stats")
		return
	}
	setPublicStatsCache(ctx)
	writeJSON(ctx, 200, map[string]any{
		"period": period,
		"limit":  limit,
		"apps":   apps,
	})
}

func (h *StatsHandler) HandleAppDetailStats(ctx *fasthttp.RequestCtx) {
	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "30d"
	}
	slug := ctx.UserValue("slug")
	if h.appQ == nil {
		writeError(ctx, 500, "server_error", "App stats are not configured")
		return
	}
	app, err := h.appQ.GetAppStats(ctx, toString(slug), period)
	if err != nil {
		writeError(ctx, 404, "not_found", "App not found")
		return
	}
	setPublicStatsCache(ctx)
	writeJSON(ctx, 200, map[string]any{
		"period": period,
		"app":    app,
	})
}

func setPublicStatsCache(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Cache-Control", "public, max-age=300, stale-while-revalidate=3600")
}

func toString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func appHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return rawURL
	}
	return strings.TrimPrefix(u.Host, "www.")
}

func truncateRunes(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return string(runes)
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + "..."
}

func compactTokens(value int64) string {
	switch {
	case value >= 1_000_000_000_000:
		return fmt.Sprintf("%.2fT tokens", float64(value)/1_000_000_000_000)
	case value >= 1_000_000_000:
		return fmt.Sprintf("%.2fB tokens", float64(value)/1_000_000_000)
	case value >= 1_000_000:
		return fmt.Sprintf("%.2fM tokens", float64(value)/1_000_000)
	case value >= 1_000:
		return fmt.Sprintf("%.1fK tokens", float64(value)/1_000)
	default:
		return fmt.Sprintf("%d tokens", value)
	}
}
