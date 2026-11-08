package handler

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
)

type AccountHandler struct {
	apiKeyQ    *queries.APIKeyQueries
	creditQ    *queries.CreditQueries
	billing    *billing.Engine
	reconciler *billing.Reconciler
}

func NewAccountHandler(apiKeyQ *queries.APIKeyQueries, creditQ *queries.CreditQueries, billing *billing.Engine, reconciler *billing.Reconciler) *AccountHandler {
	return &AccountHandler{apiKeyQ: apiKeyQ, creditQ: creditQ, billing: billing, reconciler: reconciler}
}

// HandleGetBalance handles GET /account/balance.
func (h *AccountHandler) HandleGetBalance(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	if h.reconciler != nil {
		rctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if credited, err := h.reconciler.ReconcileUser(rctx, userID); err != nil {
			log.Printf("reconcile on balance fetch for user %s: %v", userID, err)
		} else if credited > 0 {
			log.Printf("reconcile on balance fetch credited %d session(s) for user %s", credited, userID)
		}
		cancel()
	}

	balance, err := h.billing.GetBalance(ctx, userID)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get balance")
		return
	}

	// Convert from hundredths-of-a-cent to dollars for display
	dollars := float64(balance) / 10000.0

	writeJSON(ctx, 200, map[string]any{
		"balance_cents":     balance,
		"balance_usd":       dollars,
		"balance_usd_exact": formatUSDExact(balance),
	})
}

// HandleGetTransactions handles GET /account/transactions.
func (h *AccountHandler) HandleGetTransactions(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	limit, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("limit")))
	offset, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("offset")))
	if limit <= 0 {
		limit = 50
	}

	txns, err := h.creditQ.GetTransactions(ctx, userID, limit, offset)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get transactions")
		return
	}

	writeJSON(ctx, 200, map[string]any{"transactions": txns})
}
