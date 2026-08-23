package handler

import (
	"encoding/json"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
)

type CreditsHandler struct {
	billing *billing.Engine
	users   *queries.UserQueries
}

func NewCreditsHandler(billing *billing.Engine, users *queries.UserQueries) *CreditsHandler {
	return &CreditsHandler{billing: billing, users: users}
}

type addCreditsRequest struct {
	AmountCents int64  `json:"amount_cents"`
	Description string `json:"description"`
}

// HandleAddCredits handles POST /account/credits/add.
// This endpoint is retained only for controlled administrative/test grants.
// Normal users must receive paid credits through the Stripe webhook.
func (h *CreditsHandler) HandleAddCredits(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if h.users == nil {
		writeError(ctx, 503, "server_error", "Credit grants are unavailable")
		return
	}
	user, err := h.users.GetByID(ctx, userID)
	if err != nil || user == nil || !user.IsAdmin {
		writeError(ctx, 403, "forbidden", "Only administrators can grant credits")
		return
	}

	var req addCreditsRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON")
		return
	}

	if req.AmountCents <= 0 {
		writeError(ctx, 400, "invalid_request", "amount_cents must be positive")
		return
	}

	if req.Description == "" {
		req.Description = "Credit top-up"
	}

	if err := h.billing.Deposit(ctx, userID, req.AmountCents, req.Description); err != nil {
		writeError(ctx, 500, "server_error", "Failed to add credits")
		return
	}

	balance, _ := h.billing.GetBalance(ctx, userID)

	writeJSON(ctx, 200, map[string]any{
		"message":           "Credits added successfully",
		"added_cents":       req.AmountCents,
		"balance_cents":     balance,
		"balance_usd":       float64(balance) / 10000.0,
		"balance_usd_exact": formatUSDExact(balance),
	})
}
