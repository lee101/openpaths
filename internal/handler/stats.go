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

	appHost := appHost(app.URL)
	if appHost == "" {
		appHost = app.Source
	}
	title := truncateRunes(app.Name, 24)
	description := wrapTextLines(app.Description, 50, 2)
	for len(description) < 2 {
		description = append(description, "")
	}
	tokens := compactTokens(app.TotalTokens)
	initials := appInitials(app.Name)

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630">
  <defs><pattern id="grid" width="60" height="60" patternUnits="userSpaceOnUse"><path d="M60 0H0V60" fill="none" stroke="#0d131c"/></pattern></defs>
  <rect width="1200" height="630" fill="#05070b"/><rect width="1200" height="630" fill="url(#grid)"/>
  <rect width="1200" height="5" fill="#38bdf8"/><rect y="626" width="1200" height="4" fill="#102c3a"/>
  <g transform="translate(68 54) scale(.0969388) translate(-120 -60)"><g transform="rotate(180 256 256)" fill="none" stroke="#f6f8fb" stroke-width="34" stroke-linecap="square" stroke-linejoin="round">
    <path d="M340 256C315 256 282 256 260 256"/><path d="M260 256C214 256 176 226 162 178C148 132 172 94 70 108"/><path d="M260 256C214 256 176 286 162 334 C148 380 172 418 70 404"/>
    <g fill="#f6f8fb" stroke="none"><path d="M4 108l78 38V70z"/><path d="M4 404l78 38v-76z"/></g>
  </g></g>
  <text x="116" y="82" fill="#f6f8fb" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="18" font-weight="700">OPENPATHS</text>
  <text x="232" y="82" fill="#606d7e" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="18">/  APPS &amp; AGENTS</text>
  <text x="68" y="156" fill="#38bdf8" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="17" font-weight="700">APP USAGE</text>
  <text x="68" y="241" fill="#f6f8fb" font-family="Inter, ui-sans-serif, system-ui, sans-serif" font-size="64" font-weight="800">%s</text>
  <text x="70" y="285" fill="#8190a4" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="22">%s</text>
  <text x="70" y="361" fill="#cbd5e1" font-family="Inter, ui-sans-serif, system-ui, sans-serif" font-size="27">%s</text>
  <text x="70" y="401" fill="#cbd5e1" font-family="Inter, ui-sans-serif, system-ui, sans-serif" font-size="27">%s</text>
  <text x="70" y="488" fill="#64748b" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="17">MODEL ACTIVITY ATTRIBUTED THROUGH OPENPATHS</text>
  <rect x="778" y="72" width="354" height="486" rx="34" fill="#090d14" stroke="#155e75" stroke-width="2"/>
  <circle cx="955" cy="230" r="100" fill="#071b26" stroke="#0e7490" stroke-width="2"/><circle cx="955" cy="230" r="78" fill="#0c2531" stroke="#38bdf8"/>
  <text x="955" y="255" text-anchor="middle" fill="#e0f2fe" font-family="Inter, ui-sans-serif, system-ui, sans-serif" font-size="70" font-weight="800">%s</text>
  <text x="823" y="376" fill="#64748b" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="17">TOKENS ROUTED</text>
  <text x="823" y="422" fill="#f6f8fb" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="34" font-weight="800">%s</text>
  <line x1="823" y1="454" x2="1087" y2="454" stroke="#1f2937"/>
  <text x="823" y="494" fill="#7dd3fc" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="18" font-weight="700">OPENPATHS APP NETWORK</text>
  <text x="68" y="594" fill="#4b5869" font-family="ui-monospace, SFMono-Regular, Menlo, monospace" font-size="17">openpaths.io/apps/%s</text>
</svg>`,
		html.EscapeString(title),
		html.EscapeString(appHost),
		html.EscapeString(description[0]),
		html.EscapeString(description[1]),
		html.EscapeString(initials),
		html.EscapeString(tokens),
		html.EscapeString(strings.TrimSuffix(slug, ".svg")),
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

func wrapTextLines(value string, maxRunes, limit int) []string {
	words := strings.Fields(strings.TrimSpace(value))
	if limit <= 0 || len(words) == 0 {
		return nil
	}
	lines := make([]string, 0, limit)
	for _, word := range words {
		if len(lines) == 0 {
			lines = append(lines, word)
			continue
		}
		candidate := lines[len(lines)-1] + " " + word
		if len([]rune(candidate)) <= maxRunes {
			lines[len(lines)-1] = candidate
			continue
		}
		if len(lines) == limit {
			runes := []rune(candidate)
			if len(runes) > maxRunes-3 {
				runes = runes[:maxRunes-3]
			}
			lines[len(lines)-1] = string(runes) + "..."
			break
		}
		lines = append(lines, word)
	}
	return lines
}

func appInitials(value string) string {
	words := strings.FieldsFunc(strings.TrimSpace(value), func(r rune) bool {
		return r == ' ' || r == '\t' || r == '.' || r == '_' || r == '-'
	})
	initials := make([]rune, 0, 2)
	for _, word := range words {
		runes := []rune(word)
		if len(runes) > 0 {
			initials = append(initials, runes[0])
		}
		if len(initials) == 2 {
			break
		}
	}
	if len(initials) == 0 {
		return "APP"
	}
	return strings.ToUpper(string(initials))
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
