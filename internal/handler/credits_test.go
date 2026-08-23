package handler

import (
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestHandleAddCreditsRejectsClientSuppliedCreditMint(t *testing.T) {
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetBodyString(`{"amount_cents":1000000,"description":"probe"}`)

	NewCreditsHandler().HandleAddCredits(&ctx)

	if got := ctx.Response.StatusCode(); got != fasthttp.StatusGone {
		t.Fatalf("status = %d, want %d", got, fasthttp.StatusGone)
	}
	if body := string(ctx.Response.Body()); !strings.Contains(body, "verified Stripe payment") {
		t.Fatalf("response does not explain the payment requirement: %s", body)
	}
}
