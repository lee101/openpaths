package handler

import (
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
)

type AccountStatsHandler struct {
	statsQ *queries.StatsQueries
}

func NewAccountStatsHandler(statsQ *queries.StatsQueries) *AccountStatsHandler {
	return &AccountStatsHandler{statsQ: statsQ}
}

// HandleUserTimeSeries handles GET /account/stats/timeseries?period=30d&interval=1d&metric=cost
func (h *AccountStatsHandler) HandleUserTimeSeries(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	period := string(ctx.QueryArgs().Peek("period"))
	interval := string(ctx.QueryArgs().Peek("interval"))
	metric := string(ctx.QueryArgs().Peek("metric"))

	if period == "" {
		period = "30d"
	}
	if interval == "" {
		interval = "1d"
	}
	if metric == "" {
		metric = "cost"
	}

	points, err := h.statsQ.GetUserTimeSeries(ctx, userID, period, interval, metric)
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

// HandleUserSpendByAPIKey handles GET /account/stats/by-api-key?period=30d
func (h *AccountStatsHandler) HandleUserSpendByAPIKey(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "30d"
	}

	spend, err := h.statsQ.GetUserSpendByAPIKey(ctx, userID, period)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get API key spend")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"period": period,
		"keys":   spend,
	})
}

// HandleUserSpendByProvider handles GET /account/stats/by-provider?period=30d
func (h *AccountStatsHandler) HandleUserSpendByProvider(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "30d"
	}

	spend, err := h.statsQ.GetUserSpendByProvider(ctx, userID, period)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get provider spend")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"period":    period,
		"providers": spend,
	})
}

// HandleUserAPIKeyDrilldown handles GET /account/stats/by-api-key/{key_id}/models?period=30d
func (h *AccountStatsHandler) HandleUserAPIKeyDrilldown(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	keyID, _ := ctx.UserValue("key_id").(string)

	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "30d"
	}

	stats, err := h.statsQ.GetUserAPIKeyDrilldown(ctx, userID, keyID, period)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get API key drilldown")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"period": period,
		"models": stats,
	})
}

// HandleUserProviderDrilldown handles GET /account/stats/by-provider/{provider}/models?period=30d
func (h *AccountStatsHandler) HandleUserProviderDrilldown(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	prov, _ := ctx.UserValue("provider").(string)

	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "30d"
	}

	stats, err := h.statsQ.GetUserProviderDrilldown(ctx, userID, prov, period)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get provider drilldown")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"period":   period,
		"provider": prov,
		"models":   stats,
	})
}
