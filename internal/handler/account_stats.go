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

// HandleUserSpendByProduct handles GET /account/stats/by-product?period=30d
func (h *AccountStatsHandler) HandleUserSpendByProduct(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "30d"
	}

	spend, err := h.statsQ.GetUserSpendByProduct(ctx, userID, period)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get product spend")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"period":   period,
		"products": spend,
	})
}

// HandleUserSpendByModel handles GET /account/stats/by-model?period=30d.
func (h *AccountStatsHandler) HandleUserSpendByModel(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "30d"
	}
	models, err := h.statsQ.GetUserUsage(ctx, userID, period)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get model spend")
		return
	}
	writeJSON(ctx, 200, map[string]any{"period": period, "models": models})
}

// HandleUserSpendByApp handles GET /account/stats/by-app?period=30d.
func (h *AccountStatsHandler) HandleUserSpendByApp(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "30d"
	}
	apps, err := h.statsQ.GetUserSpendByApp(ctx, userID, period)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get app spend")
		return
	}
	writeJSON(ctx, 200, map[string]any{"period": period, "apps": apps})
}

// HandleUserRecentUsage handles GET /account/stats/recent?period=30d&limit=50.
func (h *AccountStatsHandler) HandleUserRecentUsage(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "30d"
	}
	limit := ctx.QueryArgs().GetUintOrZero("limit")
	if limit == 0 {
		limit = 50
	}
	events, err := h.statsQ.GetUserRecentUsage(ctx, userID, period, int(limit))
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get recent usage")
		return
	}
	writeJSON(ctx, 200, map[string]any{"period": period, "events": events})
}

// HandleUserActivity handles GET /account/stats/activity?days=365
// It powers the GitHub-style contribution heatmap.
func (h *AccountStatsHandler) HandleUserActivity(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	days := ctx.QueryArgs().GetUintOrZero("days")
	if days <= 0 {
		days = 365
	}

	activity, err := h.statsQ.GetUserDailyActivity(ctx, userID, days)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get activity")
		return
	}

	writeJSON(ctx, 200, map[string]any{
		"days": days,
		"data": activity,
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
