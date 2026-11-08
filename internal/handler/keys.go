package handler

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/auth"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
)

const maxRotateGraceSeconds = 86400

type keyView struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Prefix       string     `json:"prefix"`
	KeyPrefix    string     `json:"key_prefix"`
	Scopes       []string   `json:"scopes"`
	IsManagement bool       `json:"is_management"`
	CreatedAt    time.Time  `json:"created_at"`
	LastUsedAt   *time.Time `json:"last_used_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
	RevokedAt    *time.Time `json:"revoked_at"`
	RotatedFrom  *string    `json:"rotated_from"`
	RateLimitRPM int        `json:"rate_limit_rpm"`
}

func newKeyView(k *model.APIKey) keyView {
	return keyView{
		ID: k.ID, Name: k.Name, Prefix: k.KeyPrefix, KeyPrefix: k.KeyPrefix,
		Scopes: k.EffectiveScopes(), IsManagement: k.IsManagement(),
		CreatedAt: k.CreatedAt, LastUsedAt: k.LastUsedAt, ExpiresAt: k.ExpiresAt,
		RevokedAt: k.RevokedAt, RotatedFrom: k.RotatedFrom, RateLimitRPM: k.RateLimitRPM,
	}
}

func writeKeyError(ctx *fasthttp.RequestCtx, status int, errType, code, message, hint string) {
	data, _ := json.Marshal(model.ErrorResponse{Error: model.ErrorDetail{Message: message, Type: errType, Code: &code, Hint: hint}})
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	ctx.SetBody(data)
}

func writeKeyNotFound(ctx *fasthttp.RequestCtx) {
	writeKeyError(ctx, 404, "invalid_request_error", "not_found", "API key not found", "List keys with GET /v1/keys.")
}

func keySecretResponse(raw string, k *model.APIKey, message string) map[string]any {
	v := newKeyView(k)
	return map[string]any{
		"key": raw, "secret": raw, "id": v.ID, "name": v.Name, "prefix": v.Prefix,
		"scopes": v.Scopes, "is_management": v.IsManagement, "created_at": v.CreatedAt,
		"expires_at": v.ExpiresAt, "rotated_from": v.RotatedFrom, "message": message,
	}
}

func (h *AccountHandler) HandleKeyScopes(ctx *fasthttp.RequestCtx) {
	writeJSON(ctx, 200, map[string]any{"scopes": model.ScopeCatalog()})
}

func (h *AccountHandler) HandleKeyMe(ctx *fasthttp.RequestCtx) {
	k, ok := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	if !ok {
		writeJSON(ctx, 200, map[string]any{"id": nil, "name": "session", "scopes": model.AllScopes(), "expires_at": nil, "is_management": true})
		return
	}
	v := newKeyView(k)
	writeJSON(ctx, 200, map[string]any{"id": v.ID, "name": v.Name, "prefix": v.Prefix, "scopes": v.Scopes, "expires_at": v.ExpiresAt, "is_management": v.IsManagement})
}

func (h *AccountHandler) HandleListAPIKeys(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	keys, err := h.apiKeyQ.ListByUser(ctx, userID)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to list API keys")
		return
	}
	views := make([]keyView, 0, len(keys))
	for i := range keys {
		views = append(views, newKeyView(&keys[i]))
	}
	writeJSON(ctx, 200, map[string]any{"keys": views})
}

type createKeyRequest struct {
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes"`
	ExpiresInDays *int     `json:"expires_in_days"`
}

func (h *AccountHandler) HandleCreateAPIKey(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)

	var req createKeyRequest
	if body := ctx.PostBody(); len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeKeyError(ctx, 400, "invalid_request_error", "invalid_json", "Request body is not valid JSON", `Expected {"name": string, "scopes": [string], "expires_in_days": number}.`)
			return
		}
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "Default"
	}
	if len(name) > 100 {
		writeKeyError(ctx, 400, "invalid_request_error", "invalid_name", "name must be at most 100 characters", "")
		return
	}

	scopes := model.UsageScopes()
	if req.Scopes != nil {
		var err error
		if scopes, err = model.NormalizeScopes(req.Scopes); err != nil || len(scopes) == 0 {
			msg := "scopes must contain at least one scope"
			if err != nil {
				msg = err.Error()
			}
			writeKeyError(ctx, 400, "invalid_request_error", "invalid_scope", msg, "See GET /v1/keys/scopes for the catalog.")
			return
		}
	}

	var expiresAt *time.Time
	if req.ExpiresInDays != nil {
		if *req.ExpiresInDays < 1 || *req.ExpiresInDays > 3650 {
			writeKeyError(ctx, 400, "invalid_request_error", "invalid_expiry", "expires_in_days must be between 1 and 3650", "")
			return
		}
		t := time.Now().Add(time.Duration(*req.ExpiresInDays) * 24 * time.Hour)
		expiresAt = &t
	}

	rawKey, hash, prefix, err := auth.GenerateAPIKey()
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to generate API key")
		return
	}

	key, err := h.apiKeyQ.CreateScoped(ctx, userID, hash, prefix, name, scopes, expiresAt)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to create API key")
		return
	}

	writeJSON(ctx, 201, keySecretResponse(rawKey, key, "Save this key securely. It will not be shown again."))
}

type rotateKeyRequest struct {
	GraceSeconds int `json:"grace_seconds"`
}

func (h *AccountHandler) HandleRotateAPIKey(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	keyID, _ := ctx.UserValue("id").(string)

	if _, err := uuid.Parse(keyID); err != nil {
		writeKeyNotFound(ctx)
		return
	}
	var req rotateKeyRequest
	if body := ctx.PostBody(); len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeKeyError(ctx, 400, "invalid_request_error", "invalid_json", "Request body is not valid JSON", `Expected {"grace_seconds": number}.`)
			return
		}
	}
	if req.GraceSeconds < 0 || req.GraceSeconds > maxRotateGraceSeconds {
		writeKeyError(ctx, 400, "invalid_request_error", "invalid_grace", "grace_seconds must be between 0 and 86400", "")
		return
	}

	rawKey, hash, prefix, err := auth.GenerateAPIKey()
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to generate API key")
		return
	}
	newKey, oldKey, err := h.apiKeyQ.Rotate(ctx, keyID, userID, hash, prefix, time.Duration(req.GraceSeconds)*time.Second)
	switch {
	case errors.Is(err, queries.ErrKeyNotFound):
		writeKeyNotFound(ctx)
		return
	case errors.Is(err, queries.ErrKeyRevoked), errors.Is(err, queries.ErrKeyExpired):
		writeKeyError(ctx, 409, "invalid_request_error", "key_inactive", "API key is revoked, expired or already rotated", "Create a new key with POST /v1/keys.")
		return
	case err != nil:
		writeError(ctx, 500, "server_error", "Failed to rotate API key")
		return
	}

	resp := keySecretResponse(rawKey, newKey, "Save this key securely. It will not be shown again.")
	resp["old_key"] = map[string]any{"id": oldKey.ID, "revoked_at": oldKey.RevokedAt}
	writeJSON(ctx, 200, resp)
}

type revokeKeysRequest struct {
	IDs []string `json:"ids"`
}

func (h *AccountHandler) HandleRevokeAPIKeys(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	var req revokeKeysRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil || len(req.IDs) == 0 {
		writeError(ctx, 400, "invalid_request", "ids are required")
		return
	}
	if err := h.apiKeyQ.RevokeMany(ctx, req.IDs, userID); err != nil {
		writeError(ctx, 500, "server_error", "Failed to revoke API keys")
		return
	}
	writeJSON(ctx, 200, map[string]any{"message": "API keys revoked", "count": len(req.IDs)})
}

func (h *AccountHandler) HandleRevokeAPIKey(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	keyID, _ := ctx.UserValue("id").(string)

	if _, err := uuid.Parse(keyID); err != nil {
		writeKeyNotFound(ctx)
		return
	}
	err := h.apiKeyQ.Revoke(ctx, keyID, userID)
	switch {
	case errors.Is(err, queries.ErrKeyNotFound):
		writeKeyNotFound(ctx)
		return
	case err != nil:
		writeError(ctx, 500, "server_error", "Failed to revoke API key")
		return
	}

	writeJSON(ctx, 200, map[string]any{"id": keyID, "revoked": true, "message": "API key revoked"})
}
