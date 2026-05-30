package middleware

import (
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/auth"
	"github.com/openpaths/openpaths/internal/db/queries"
)

const (
	CtxKeyUserID = "user_id"
	CtxKeyAPIKey = "api_key"
)

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
				writeAuthError(ctx, "Invalid API key", "invalid_api_key")
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
					writeAuthError(ctx, "Invalid API key", "invalid_api_key")
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

func writeAuthError(ctx *fasthttp.RequestCtx, message, code string) {
	ctx.SetStatusCode(401)
	ctx.SetContentType("application/json")
	ctx.SetBodyString(`{"error":{"message":"` + message + `","type":"auth_error","code":"` + code + `"}}`)
}
