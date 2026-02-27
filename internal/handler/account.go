package handler

import (
	"encoding/json"
	"strconv"

	"github.com/valyala/fasthttp"

	"github.com/openpath/openpath/internal/auth"
	"github.com/openpath/openpath/internal/billing"
	"github.com/openpath/openpath/internal/db/queries"
	"github.com/openpath/openpath/internal/middleware"
)

type AccountHandler struct {
	apiKeyQ *queries.APIKeyQueries
	creditQ *queries.CreditQueries
	billing *billing.Engine
}

func NewAccountHandler(apiKeyQ *queries.APIKeyQueries, creditQ *queries.CreditQueries, billing *billing.Engine) *AccountHandler {
	return &AccountHandler{apiKeyQ: apiKeyQ, creditQ: creditQ, billing: billing}
}

// HandleListAPIKeys handles GET /account/keys.
func (h *AccountHandler) HandleListAPIKeys(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	keys, err := h.apiKeyQ.ListByUser(ctx, userID)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to list API keys")
		return
	}

	writeJSON(ctx, 200, map[string]any{"keys": keys})
}

type createKeyRequest struct {
	Name string `json:"name"`
}

// HandleCreateAPIKey handles POST /account/keys.
func (h *AccountHandler) HandleCreateAPIKey(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	var req createKeyRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		req.Name = "Default"
	}
	if req.Name == "" {
		req.Name = "Default"
	}

	rawKey, hash, prefix, err := auth.GenerateAPIKey()
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to generate API key")
		return
	}

	key, err := h.apiKeyQ.Create(ctx, userID, hash, prefix, req.Name)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to create API key")
		return
	}

	writeJSON(ctx, 201, map[string]any{
		"key":     rawKey, // Only shown once
		"id":      key.ID,
		"name":    key.Name,
		"prefix":  key.KeyPrefix,
		"message": "Save this key securely. It will not be shown again.",
	})
}

// HandleRevokeAPIKey handles DELETE /account/keys/{id}.
func (h *AccountHandler) HandleRevokeAPIKey(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	keyID, _ := ctx.UserValue("id").(string)

	if err := h.apiKeyQ.Revoke(ctx, keyID, userID); err != nil {
		writeError(ctx, 404, "not_found", "API key not found")
		return
	}

	writeJSON(ctx, 200, map[string]any{"message": "API key revoked"})
}

// HandleGetBalance handles GET /account/balance.
func (h *AccountHandler) HandleGetBalance(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	balance, err := h.billing.GetBalance(ctx, userID)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get balance")
		return
	}

	// Convert from hundredths-of-a-cent to dollars for display
	dollars := float64(balance) / 10000.0

	writeJSON(ctx, 200, map[string]any{
		"balance_cents": balance,
		"balance_usd":   dollars,
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
