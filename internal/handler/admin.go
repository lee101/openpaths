package handler

import (
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
)

type AdminHandler struct {
	userQ *queries.UserQueries
}

func NewAdminHandler(userQ *queries.UserQueries) *AdminHandler {
	return &AdminHandler{userQ: userQ}
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
