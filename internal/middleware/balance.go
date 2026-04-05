package middleware

import (
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

			if keys, _ := ctx.UserValue(CtxKeyUserProviderKeys).(map[string]any); keys != nil && len(keys) > 0 {
				next(ctx)
				return
			}
			if keys := ctx.UserValue(CtxKeyUserProviderKeys); keys != nil {
				next(ctx)
				return
			}

			err := engine.PreCheck(ctx, userID, "", 100)
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
