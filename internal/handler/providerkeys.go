package handler

import (
	"encoding/json"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
)

type ProviderKeysHandler struct {
	pkQ *queries.ProviderKeyQueries
}

func NewProviderKeysHandler(pkQ *queries.ProviderKeyQueries) *ProviderKeysHandler {
	return &ProviderKeysHandler{pkQ: pkQ}
}

func (h *ProviderKeysHandler) HandleList(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	keys, err := h.pkQ.ListByUser(ctx, userID)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to list provider keys")
		return
	}
	masked := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		m := map[string]any{
			"provider":   k.Provider,
			"has_key":    k.APIKey != "",
			"has_auth":   k.AuthJSON != "",
			"updated_at": k.UpdatedAt,
		}
		if k.APIKey != "" && len(k.APIKey) > 8 {
			m["key_preview"] = k.APIKey[:4] + "..." + k.APIKey[len(k.APIKey)-4:]
		}
		masked = append(masked, m)
	}
	writeJSON(ctx, 200, map[string]any{"keys": masked})
}

type upsertProviderKeyReq struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	AuthJSON string `json:"auth_json"`
}

func (h *ProviderKeysHandler) HandleUpsert(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	var req upsertProviderKeyReq
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON")
		return
	}
	if req.Provider == "" {
		writeError(ctx, 400, "invalid_request", "provider is required")
		return
	}
	if req.APIKey == "" && req.AuthJSON == "" {
		writeError(ctx, 400, "invalid_request", "api_key or auth_json is required")
		return
	}

	if err := h.pkQ.Upsert(ctx, userID, req.Provider, req.APIKey, req.AuthJSON); err != nil {
		writeError(ctx, 500, "server_error", "Failed to save provider key")
		return
	}
	writeJSON(ctx, 200, map[string]any{"message": "Provider key saved", "provider": req.Provider})
}

type bulkUpsertReq struct {
	Keys []upsertProviderKeyReq `json:"keys"`
}

func (h *ProviderKeysHandler) HandleBulkUpsert(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	var req bulkUpsertReq
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON")
		return
	}

	saved := 0
	for _, k := range req.Keys {
		if k.Provider == "" || (k.APIKey == "" && k.AuthJSON == "") {
			continue
		}
		if err := h.pkQ.Upsert(ctx, userID, k.Provider, k.APIKey, k.AuthJSON); err != nil {
			writeError(ctx, 500, "server_error", "Failed to save key for "+k.Provider)
			return
		}
		saved++
	}
	writeJSON(ctx, 200, map[string]any{"message": "Saved", "count": saved})
}

func (h *ProviderKeysHandler) HandleDelete(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	provider := string(ctx.QueryArgs().Peek("provider"))
	if provider == "" {
		writeError(ctx, 400, "invalid_request", "provider query param required")
		return
	}
	if err := h.pkQ.Delete(ctx, userID, provider); err != nil {
		writeError(ctx, 500, "server_error", "Failed to delete provider key")
		return
	}
	writeJSON(ctx, 200, map[string]any{"message": "Deleted"})
}
