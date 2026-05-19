package handler

import (
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
)

type StatsHandler struct {
	statsQ *queries.StatsQueries
}

func NewStatsHandler(statsQ *queries.StatsQueries) *StatsHandler {
	return &StatsHandler{statsQ: statsQ}
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
