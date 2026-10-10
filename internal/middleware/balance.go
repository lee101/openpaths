package middleware

import (
	"bytes"
	"encoding/json"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
)

// BalanceCheck verifies the user has credits before processing.
// Skipped if user has BYOK provider keys loaded.
func BalanceCheck(engine *billing.Engine) Middleware {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			userID, _ := ctx.UserValue(CtxKeyUserID).(string)
			if userID == "" {
				next(ctx)
				return
			}

			// BYOK users pay their own provider, so they are exempt from the
			// credit pre-check here. This bypass is intentionally broad (any BYOK
			// key skips the check), so the per-attempt prepay gate in the chat /
			// messages handlers (enforcePrepaid) is what actually guarantees a
			// request routed to OpenPaths' own key is backed by a balance.
			if keys := ctx.UserValue(CtxKeyUserProviderKeys); keys != nil {
				next(ctx)
				return
			}

			modelID := requestModel(ctx)
			err := engine.PreCheck(ctx, userID, modelID, 100)
			if err != nil {
				ctx.SetStatusCode(402)
				ctx.SetContentType("application/json")
				ctx.SetBodyString(`{"error":{"message":"Insufficient credits. Please add credits to continue.","type":"billing_error","code":"insufficient_balance"}}`)
				return
			}

			next(ctx)
		}
	}
}

func requestModel(ctx *fasthttp.RequestCtx) string {
	dec := json.NewDecoder(bytes.NewReader(ctx.PostBody()))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return ""
	}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return ""
		}
		key, _ := kt.(string)
		if key == "model" {
			var m string
			if err := dec.Decode(&m); err != nil {
				return ""
			}
			return m
		}
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return ""
		}
	}
	return ""
}
