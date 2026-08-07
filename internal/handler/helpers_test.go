package handler

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestFormatUSDExact(t *testing.T) {
	tests := []struct {
		name   string
		units  int64
		expect string
	}{
		{name: "zero", units: 0, expect: "0.0000"},
		{name: "whole dollars", units: 100000, expect: "10.0000"},
		{name: "sub-cent balance", units: 99999, expect: "9.9999"},
		{name: "negative amount", units: -1, expect: "-0.0001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatUSDExact(tt.units); got != tt.expect {
				t.Fatalf("formatUSDExact(%d) = %q, want %q", tt.units, got, tt.expect)
			}
		})
	}
}

func TestStripeSignatureHeaderIsCaseInsensitive(t *testing.T) {
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.DisableNormalizing()
	ctx.Request.Header.Set("stripe-signature", "t=123,v1=abc")

	if got := stripeSignatureHeader(&ctx); got != "t=123,v1=abc" {
		t.Fatalf("stripeSignatureHeader() = %q, want lower-case header value", got)
	}
}
