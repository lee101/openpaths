package middleware

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/auth"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

const (
	CtxKeyUserID = "user_id"
	CtxKeyAPIKey = "api_key"

	ctxRequiredScope = "required_scope"
)

func RequireScope(scope string, next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.SetUserValue(ctxRequiredScope, scope)
		next(ctx)
	}
}

func ScopeAllowed(ctx *fasthttp.RequestCtx, scope string) bool {
	k, ok := ctx.UserValue(CtxKeyAPIKey).(*model.APIKey)
	return !ok || k.HasScope(scope)
}

func authorizeKey(ctx *fasthttp.RequestCtx, k *model.APIKey) bool {
	need, declared := ctx.UserValue(ctxRequiredScope).(string)
	if !declared {
		writeAuthzError(ctx, "route_unscoped", "Route does not declare a scope for API keys", "", "")
		return false
	}
	if !k.HasScope(need) {
		WriteScopeError(ctx, need)
		return false
	}
	return true
}

func keyAuthFailure(ctx *fasthttp.RequestCtx, err error) {
	switch {
	case errors.Is(err, queries.ErrKeyRevoked):
		writeAuthError(ctx, "API key has been revoked", "revoked_api_key")
	case errors.Is(err, queries.ErrKeyExpired):
		writeAuthError(ctx, "API key has expired", "expired_api_key")
	default:
		writeAuthError(ctx, "Invalid API key", "invalid_api_key")
	}
}

// peekAuthHeader returns the Authorization header value, checking both
// canonical and lowercase forms for compatibility with HTTP/2 proxies.
func peekAuthHeader(ctx *fasthttp.RequestCtx) string {
	if v := ctx.Request.Header.Peek("Authorization"); len(v) > 0 {
		return string(v)
	}
	return string(ctx.Request.Header.Peek("authorization"))
}

func peekBearerToken(ctx *fasthttp.RequestCtx) string {
	authHeader := peekAuthHeader(ctx)
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}

func peekSessionToken(ctx *fasthttp.RequestCtx) string {
	if raw := peekBearerToken(ctx); raw != "" {
		return raw
	}
	if v := ctx.Request.Header.Peek("X-API-Key"); len(v) > 0 {
		return string(v)
	}
	if v := ctx.Request.Header.Cookie("op_session"); len(v) > 0 {
		return string(v)
	}
	return ""
}

// APIKeyAuth validates Bearer tokens or x-api-key header as API keys.
func APIKeyAuth(apiKeyQ *queries.APIKeyQueries) Middleware {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			var rawKey string
			authHeader := peekAuthHeader(ctx)
			xAPIKey := string(ctx.Request.Header.Peek("x-api-key"))

			if strings.HasPrefix(authHeader, "Bearer ") {
				rawKey = strings.TrimPrefix(authHeader, "Bearer ")
			} else if xAPIKey != "" {
				rawKey = xAPIKey
			} else {
				writeAuthError(ctx, "Missing API key", "missing_api_key")
				return
			}

			apiKey, err := apiKeyQ.ValidateKey(ctx, auth.HashAPIKey(rawKey))
			if err != nil {
				keyAuthFailure(ctx, err)
				return
			}
			if !authorizeKey(ctx, apiKey) {
				return
			}

			ctx.SetUserValue(CtxKeyUserID, apiKey.UserID)
			ctx.SetUserValue(CtxKeyAPIKey, apiKey)

			next(ctx)
		}
	}
}

// DashboardAuth accepts OpenPaths API keys or dashboard JWTs (Bearer header or op_session cookie).
func DashboardAuth(apiKeyQ *queries.APIKeyQueries, jwtService *auth.JWTService) Middleware {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			raw := peekSessionToken(ctx)
			if raw == "" {
				writeAuthError(ctx, "Missing credentials", "missing_auth")
				return
			}
			if strings.HasPrefix(raw, auth.APIKeyPrefix) {
				apiKey, err := apiKeyQ.ValidateKey(ctx, auth.HashAPIKey(raw))
				if err != nil {
					keyAuthFailure(ctx, err)
					return
				}
				if !authorizeKey(ctx, apiKey) {
					return
				}
				ctx.SetUserValue(CtxKeyUserID, apiKey.UserID)
				ctx.SetUserValue(CtxKeyAPIKey, apiKey)
				next(ctx)
				return
			}
			claims, err := jwtService.Validate(raw)
			if err != nil {
				writeAuthError(ctx, "Invalid token", "invalid_token")
				return
			}
			ctx.SetUserValue(CtxKeyUserID, claims.UserID)
			next(ctx)
		}
	}
}

// JWTAuth validates JWT tokens for web dashboard access.
func JWTAuth(jwtService *auth.JWTService) Middleware {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			authHeader := peekAuthHeader(ctx)
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeAuthError(ctx, "Missing token", "missing_token")
				return
			}
			token := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := jwtService.Validate(token)
			if err != nil {
				writeAuthError(ctx, "Invalid token", "invalid_token")
				return
			}

			ctx.SetUserValue(CtxKeyUserID, claims.UserID)
			next(ctx)
		}
	}
}

var authHints = map[string]string{
	"missing_api_key":     "Send the key as 'Authorization: Bearer <key>' or 'X-API-Key: <key>'.",
	"missing_auth":        "Send an API key or session token as 'Authorization: Bearer <credential>'.",
	"missing_credentials": "Send an API key or session token as 'Authorization: Bearer <credential>'.",
	"invalid_api_key":     "Check the key value, or create one at https://openpaths.io/account/apikeys.",
	"revoked_api_key":     "Create a replacement with POST /v1/keys or rotate with POST /v1/keys/{id}/rotate.",
	"expired_api_key":     "Create a replacement with POST /v1/keys or rotate with POST /v1/keys/{id}/rotate.",
}

func writeAuthzError(ctx *fasthttp.RequestCtx, code, message, hint, scope string) {
	detail := model.ErrorDetail{Message: message, Type: "permission_error", Code: &code, Hint: hint, RequiredScope: scope}
	data, _ := json.Marshal(model.ErrorResponse{Error: detail})
	ctx.SetStatusCode(403)
	ctx.SetContentType("application/json")
	if scope != "" {
		ctx.Response.Header.Set("WWW-Authenticate", `Bearer error="insufficient_scope", scope="`+scope+`"`)
	}
	ctx.SetBody(data)
}

func WriteScopeError(ctx *fasthttp.RequestCtx, scope string) {
	hint := "Use a key that includes this scope; see GET /v1/keys/scopes and GET /v1/keys/me."
	if scope == model.ScopeKeysManage {
		hint = "Use a management key (scope keys:manage) or the dashboard session to manage API keys."
	}
	writeAuthzError(ctx, "insufficient_scope", "API key lacks required scope: "+scope, hint, scope)
}

func writeAuthError(ctx *fasthttp.RequestCtx, message, code string) {
	detail := model.ErrorDetail{Message: message, Type: "auth_error", Code: &code, Hint: authHints[code]}
	data, _ := json.Marshal(model.ErrorResponse{Error: detail})
	ctx.SetStatusCode(401)
	ctx.SetContentType("application/json")
	ctx.Response.Header.Set("WWW-Authenticate", "Bearer")
	ctx.SetBody(data)
}
