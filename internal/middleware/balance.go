package middleware

import (
	"github.com/valyala/fasthttp"

	"github.com/openpath/openpath/internal/billing"
)

// BalanceCheck verifies the user has credits before processing.
func BalanceCheck(engine *billing.Engine) Middleware {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			userID, _ := ctx.UserValue(CtxKeyUserID).(string)
			if userID == "" {
				next(ctx)
				return
			}

			// Lightweight check with minimum estimate
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
