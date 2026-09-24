package middleware

import (
	"encoding/json"
	"testing"

	"github.com/valyala/fasthttp"
)

func runPaywall(t *testing.T, h fasthttp.RequestHandler) (*fasthttp.RequestCtx, map[string]any) {
	t.Helper()
	ctx := &fasthttp.RequestCtx{}
	PaywallContract(h)(ctx)
	var body struct {
		Error map[string]any `json:"error"`
	}
	_ = json.Unmarshal(ctx.Response.Body(), &body)
	return ctx, body.Error
}

func TestPaywallMissingKeyStays401(t *testing.T) {
	ctx, e := runPaywall(t, func(ctx *fasthttp.RequestCtx) { writeAuthError(ctx, "Missing API key", "missing_api_key") })
	if ctx.Response.StatusCode() != 401 || e["code"] != "missing_api_key" {
		t.Fatalf("got %d %v", ctx.Response.StatusCode(), e)
	}
	if len(ctx.Response.Header.Peek(SubscribeURLHeader)) != 0 {
		t.Fatal("401 must not carry subscribe header")
	}
}

func TestPaywallBalance402(t *testing.T) {
	t.Setenv("OPENPATHS_SUBSCRIBE_URL", "https://x.test/pricing")
	ctx, e := runPaywall(t, func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(402)
		ctx.SetBodyString(`{"error":{"message":"Insufficient credits. Please add credits to continue.","type":"billing_error","code":"insufficient_balance"}}`)
	})
	if ctx.Response.StatusCode() != 402 || e["code"] != SubscriptionRequiredCode || e["subscribe_url"] != "https://x.test/pricing" || e["reason"] != "insufficient_balance" {
		t.Fatalf("got %d %v", ctx.Response.StatusCode(), e)
	}
	if string(ctx.Response.Header.Peek(SubscribeURLHeader)) != "https://x.test/pricing" {
		t.Fatal("missing X-Subscribe-URL")
	}
}

func TestPaywallUpstream401And402(t *testing.T) {
	for _, st := range []int{401, 402} {
		ctx, e := runPaywall(t, func(ctx *fasthttp.RequestCtx) {
			ctx.SetStatusCode(st)
			ctx.SetBodyString(`{"error":{"message":"upstream said no","type":"provider_error"}}`)
		})
		if ctx.Response.StatusCode() != 402 || e["code"] != SubscriptionRequiredCode || e["subscribe_url"] != defaultSubscribeURL {
			t.Fatalf("%d: got %d %v", st, ctx.Response.StatusCode(), e)
		}
	}
}

func TestPaywallPassThrough(t *testing.T) {
	ctx, _ := runPaywall(t, func(ctx *fasthttp.RequestCtx) { ctx.SetStatusCode(502); ctx.SetBodyString(`{}`) })
	if ctx.Response.StatusCode() != 502 {
		t.Fatal("non paywall status rewritten")
	}
}
