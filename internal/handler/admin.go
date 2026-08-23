package handler

import (
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
)

type AdminHandler struct {
	userQ  *queries.UserQueries
	statsQ *queries.StatsQueries
}

func NewAdminHandler(userQ *queries.UserQueries, statsQ ...*queries.StatsQueries) *AdminHandler {
	h := &AdminHandler{userQ: userQ}
	if len(statsQ) > 0 {
		h.statsQ = statsQ[0]
	}
	return h
}

// HandleUserUsage handles GET /admin/users/{user_id}/usage. It mirrors the
// account activity data, but is protected by the admin gate so support and
// billing investigations can be done without impersonating a user.
func (h *AdminHandler) HandleUserUsage(ctx *fasthttp.RequestCtx) {
	if !h.requireAdmin(ctx) {
		return
	}
	if h.statsQ == nil {
		writeError(ctx, 500, "server_error", "Usage stats are not configured")
		return
	}
	userID, _ := ctx.UserValue("user_id").(string)
	if userID == "" {
		writeError(ctx, 400, "invalid_request", "user_id is required")
		return
	}
	period := string(ctx.QueryArgs().Peek("period"))
	if period == "" {
		period = "30d"
	}
	limit := ctx.QueryArgs().GetUintOrZero("limit")
	if limit == 0 {
		limit = 100
	}
	user, err := h.userQ.GetByID(ctx, userID)
	if err != nil {
		writeError(ctx, 404, "not_found", "User not found")
		return
	}
	models, err := h.statsQ.GetUserUsage(ctx, userID, period)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get user model usage")
		return
	}
	apps, err := h.statsQ.GetUserSpendByApp(ctx, userID, period)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get user app usage")
		return
	}
	activity, err := h.statsQ.GetUserDailyActivity(ctx, userID, 365)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get user activity")
		return
	}
	events, err := h.statsQ.GetUserRecentUsage(ctx, userID, period, int(limit))
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get user recent usage")
		return
	}
	writeJSON(ctx, 200, map[string]any{
		"period":   period,
		"user":     map[string]any{"id": user.ID, "email": user.Email, "name": user.Name, "disabled": user.Disabled, "is_admin": user.IsAdmin},
		"models":   models,
		"apps":     apps,
		"activity": activity,
		"events":   events,
	})
}

func (h *AdminHandler) requireAdmin(ctx *fasthttp.RequestCtx) bool {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	user, err := h.userQ.GetByID(ctx, userID)
	if err != nil {
		writeError(ctx, 401, "auth_error", "Invalid admin session")
		return false
	}
	if !user.IsAdmin {
		writeError(ctx, 403, "forbidden", "Admin access required")
		return false
	}
	return true
}

func (h *AdminHandler) RequireAdmin(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		if !h.requireAdmin(ctx) {
			return
		}
		next(ctx)
	}
}

// HandleUserSpend handles GET /admin/users/spend.
func (h *AdminHandler) HandleUserSpend(ctx *fasthttp.RequestCtx) {
	if !h.requireAdmin(ctx) {
		return
	}

	users, err := h.userQ.ListAdminUserSpend(ctx)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to list user spend")
		return
	}

	var stripeGrossCents int64
	var stripeRefundedCents int64
	var apiSpendCents int64
	var providerBaseCostCents int64
	var apiRequests int64
	for _, u := range users {
		stripeGrossCents += u.StripeGrossCents
		stripeRefundedCents += u.StripeRefundedCents
		apiSpendCents += u.APISpendCents
		providerBaseCostCents += u.ProviderBaseCostCents
		apiRequests += u.APIRequests
	}

	writeJSON(ctx, 200, map[string]any{
		"users": users,
		"totals": map[string]any{
			"user_count":               len(users),
			"stripe_gross_cents":       stripeGrossCents,
			"stripe_refunded_cents":    stripeRefundedCents,
			"stripe_net_cents":         stripeGrossCents - stripeRefundedCents,
			"api_requests":             apiRequests,
			"api_spend_cents":          apiSpendCents,
			"provider_base_cost_cents": providerBaseCostCents,
			"provider_estimated":       true,
		},
	})
}

// HandleOpenAIMaxPlanStatus reports the shared admin OpenAI max-plan credential
// state. GET /admin/openai-max-plan.
func (h *AdminHandler) HandleOpenAIMaxPlanStatus(ctx *fasthttp.RequestCtx) {
	if !h.requireAdmin(ctx) {
		return
	}
	email, userID, total, healthy := AdminOpenAIMaxPlanStatus()
	authMode, refreshable := AdminOpenAIMaxPlanAuthInfo()
	writeJSON(ctx, 200, map[string]any{
		"enabled":          email != "",
		"email":            email,
		"credential_user":  userID,
		"credential_count": total,
		"healthy_count":    healthy,
		"auth_mode":        authMode,
		"refreshable":      refreshable,
	})
}

// HandleOpenAIMaxPlanRefresh forces an immediate reload+refresh of the shared
// admin credential. POST /admin/openai-max-plan/refresh.
func (h *AdminHandler) HandleOpenAIMaxPlanRefresh(ctx *fasthttp.RequestCtx) {
	if !h.requireAdmin(ctx) {
		return
	}
	if !ForceAdminOpenAIMaxPlanRefresh() {
		writeError(ctx, 400, "not_configured", "ADMIN_OPENAI_MAX_PLAN_EMAIL is not set")
		return
	}
	writeJSON(ctx, 200, map[string]any{"status": "refreshing"})
}
