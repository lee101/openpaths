package middleware

import (
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/auth"
	"github.com/openpaths/openpaths/internal/db/queries"
)

// Realtime subprotocol scheme: the browser WebSocket API cannot set an
// Authorization header, so the dashboard client carries its credential in a
// requested subprotocol ("openpaths-api-key.<key>"). Only "openpaths-realtime"
// is actually negotiated; credential subprotocols are consumed, never echoed
// back in the handshake response.
const (
	// RealtimeProtocol is the one subprotocol we negotiate with clients.
	RealtimeProtocol = "openpaths-realtime"
	// RealtimeKeyPrefix marks a credential-bearing subprotocol. The remainder
	// of the token after the prefix is the API key.
	RealtimeKeyPrefix = "openpaths-api-key."
)

// RealtimeCredentialFromSubprotocol extracts a credential from the
// Sec-WebSocket-Protocol header, preferring an explicit Authorization header
// when present (API clients keep working). Returns "" when neither is set.
func RealtimeCredentialFromSubprotocol(ctx *fasthttp.RequestCtx) string {
	// The API server preserves HTTP/2 header casing. The WebSocket upgrader
	// expects canonical names, so normalize just this request's handshake and
	// auth headers before reading them. Browsers/proxies often send lowercase.
	names := []string{"Authorization", "X-Api-Key", "Connection", "Upgrade", "Origin", "Sec-Websocket-Key", "Sec-Websocket-Version", "Sec-Websocket-Protocol", "Sec-Websocket-Extensions"}
	values := make(map[string]string)
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		for _, name := range names {
			if strings.EqualFold(string(key), name) {
				if existing := values[name]; existing != "" {
					values[name] = existing + ", " + string(value)
				} else {
					values[name] = string(value)
				}
				break
			}
		}
	})
	ctx.Request.Header.EnableNormalizing()
	for name, value := range values {
		ctx.Request.Header.Set(name, value)
	}
	if v := peekAuthHeader(ctx); v != "" {
		if strings.HasPrefix(v, "Bearer ") {
			return strings.TrimSpace(strings.TrimPrefix(v, "Bearer "))
		}
		return ""
	}
	if key := strings.TrimSpace(string(ctx.Request.Header.Peek("x-api-key"))); key != "" {
		return key
	}
	for _, raw := range ctx.Request.Header.PeekAll("Sec-Websocket-Protocol") {
		for _, part := range strings.Split(string(raw), ",") {
			token := strings.TrimSpace(part)
			if strings.HasPrefix(token, RealtimeKeyPrefix) {
				key := strings.TrimPrefix(token, RealtimeKeyPrefix)
				if key = strings.TrimSpace(key); key != "" {
					return key
				}
			}
		}
	}
	return ""
}

// RealtimeProtocolHeaderOverride is the response header value negotiated back
// to the client. Only the neutral protocol name is ever echoed; credential
// subprotocols must never appear in the negotiated response.
func RealtimeSubprotocols() []string {
	return []string{RealtimeProtocol}
}

// RealtimeAuth resolves the realtime credential (Authorization header or
// subprotocol-carried API key) into CtxKeyUserID/CtxKeyAPIKey, mirroring
// DashboardAuth semantics so JWT dashboard sessions and API keys compose the
// same way downstream. Unlike DashboardAuth it never inspects the op_session
// cookie: browser realtime clients authenticate via subprotocol.
func RealtimeAuth(apiKeyQ *queries.APIKeyQueries, jwtService *auth.JWTService) Middleware {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			credential := RealtimeCredentialFromSubprotocol(ctx)
			if credential == "" {
				writeAuthError(ctx, "Missing realtime credential", "missing_credentials")
				return
			}
			if strings.HasPrefix(credential, auth.APIKeyPrefix) {
				apiKey, err := apiKeyQ.ValidateKey(ctx, auth.HashAPIKey(credential))
				if err != nil {
					writeAuthError(ctx, "Invalid API key", "invalid_api_key")
					return
				}
				ctx.SetUserValue(CtxKeyUserID, apiKey.UserID)
				ctx.SetUserValue(CtxKeyAPIKey, apiKey)
				next(ctx)
				return
			}
			claims, err := jwtService.Validate(credential)
			if err != nil {
				writeAuthError(ctx, "Invalid token", "invalid_token")
				return
			}
			ctx.SetUserValue(CtxKeyUserID, claims.UserID)
			next(ctx)
		}
	}
}
