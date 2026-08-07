package handler

import (
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/middleware"
)

// prepaidGate enforces prepay before a request reaches an upstream provider on
// OpenPaths' own key. It writes a 402 and returns true when the user cannot
// cover the request's worst-case cost.
//
// The BalanceCheck middleware skips its pre-check for any user holding a BYOK
// key, so without this gate a $0-balance user with an unrelated BYOK key could
// drive our keys for free on the media/audio endpoints (none of which support
// BYOK). This is the hard "prepay only" guarantee for those endpoints.
//
// modelID should be the resolved config model ID where available (so pricing is
// exact); for an unresolved alias the engine falls back to a positive-balance
// floor. maxOut is the worst-case output token count for token-priced models and
// is ignored for media/per-request pricing. A nil engine or empty userID is a
// no-op (unit tests, anonymous internal calls).
func prepaidGate(ctx *fasthttp.RequestCtx, b *billing.Engine, modelID string, maxOut int) bool {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if b == nil || userID == "" {
		return false
	}
	if maxOut <= 0 {
		maxOut = 4096
	}
	if err := b.PreCheck(ctx, userID, modelID, maxOut); err != nil {
		writeError(ctx, 402, "billing_error", "Insufficient credits. Please add credits to continue.")
		return true
	}
	return false
}
