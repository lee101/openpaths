package middleware

import (
	"net"
	"testing"

	"github.com/valyala/fasthttp"
)

func TestClientIPIgnoresHeadersUnlessPeerIsTrusted(t *testing.T) {
	var ctx fasthttp.RequestCtx
	ctx.SetRemoteAddr(&net.TCPAddr{IP: net.ParseIP("203.0.113.20"), Port: 1234})
	ctx.Request.Header.Set("CF-Connecting-IP", "198.51.100.9")
	ctx.Request.Header.Set("X-Forwarded-For", "198.51.100.8")

	t.Setenv("TRUST_PROXY_HEADERS", "false")
	if got := clientIPFor(&ctx); got != "203.0.113.20" {
		t.Fatalf("trust disabled: got %q", got)
	}

	t.Setenv("TRUST_PROXY_HEADERS", "true")
	t.Setenv("TRUSTED_PROXY_CIDRS", "127.0.0.1/32")
	if got := clientIPFor(&ctx); got != "203.0.113.20" {
		t.Fatalf("untrusted peer: got %q", got)
	}

	t.Setenv("TRUSTED_PROXY_CIDRS", "203.0.113.0/24")
	if got := clientIPFor(&ctx); got != "198.51.100.9" {
		t.Fatalf("trusted peer: got %q", got)
	}
}
