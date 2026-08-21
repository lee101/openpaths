package handler

import (
	"github.com/valyala/fasthttp"
)

type CreditsHandler struct{}

func NewCreditsHandler() *CreditsHandler {
	return &CreditsHandler{}
}

// HandleAddCredits is retained as a compatibility response for old clients.
// Credits must never be minted from client-supplied amounts or descriptions;
// Stripe webhooks and Stripe reconciliation are the only Stripe credit paths.
func (h *CreditsHandler) HandleAddCredits(ctx *fasthttp.RequestCtx) {
	writeError(ctx, 410, "credits_payment_required", "Credits are added only after a verified Stripe payment")
}
