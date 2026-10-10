package handler

import (
	"os/exec"
	"strings"
	"testing"
	"unicode/utf8"

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

func requireFFmpeg(t *testing.T, bins ...string) {
	t.Helper()
	if len(bins) == 0 {
		bins = []string{"ffmpeg"}
	}
	for _, bin := range bins {
		if err := exec.Command(bin, "-version").Run(); err != nil {
			t.Skipf("%s not runnable: %v", bin, err)
		}
	}
}

func TestWriteErrorClipsLongMessages(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	writeError(ctx, 502, "provider_error", strings.Repeat("é", 20000))
	if len(ctx.Response.Body()) > maxErrorMessageBytes+256 {
		t.Fatalf("body = %d bytes", len(ctx.Response.Body()))
	}
	if !utf8.Valid(ctx.Response.Body()) {
		t.Fatal("invalid utf8")
	}
}

func TestUpstreamClientStatusHidesProviderAuthFailures(t *testing.T) {
	for in, want := range map[int]int{401: 502, 403: 502, 400: 400, 429: 429, 500: 500} {
		if got := upstreamClientStatus(in); got != want {
			t.Errorf("%d -> %d, want %d", in, got, want)
		}
	}
}
