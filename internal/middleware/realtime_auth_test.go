package middleware

import (
	"github.com/valyala/fasthttp"
	"testing"
)

func TestRealtimeCredential(t *testing.T) {
	for _, tc := range []struct{ auth, protocol, want string }{
		{"", "openpaths-realtime, openpaths-api-key.sk-op-secret", "sk-op-secret"},
		{"Bearer header-key", "openpaths-api-key.protocol-key", "header-key"},
		{"Basic bad", "openpaths-api-key.protocol-key", ""},
		{"", "openpaths-realtime", ""},
		{"", "openpaths-api-key.", ""},
	} {
		var ctx fasthttp.RequestCtx
		ctx.Request.Header.Set("Authorization", tc.auth)
		ctx.Request.Header.Set("Sec-WebSocket-Protocol", tc.protocol)
		if got := RealtimeCredentialFromSubprotocol(&ctx); got != tc.want {
			t.Fatalf("credential extraction failed")
		}
	}
	protocols := RealtimeSubprotocols()
	if len(protocols) != 1 || protocols[0] != "openpaths-realtime" {
		t.Fatal("credential could be echoed")
	}
}

func TestRealtimeProxyHeaderCasing(t *testing.T) {
	for _, header := range []string{"sec-websocket-protocol", "Sec-WebSocket-Protocol", "Sec-Websocket-Protocol"} {
		var ctx fasthttp.RequestCtx
		ctx.Request.Header.DisableNormalizing()
		ctx.Request.Header.Set(header, "openpaths-realtime, openpaths-api-key.sk-op-secret")
		ctx.Request.Header.Set("sec-websocket-key", "websocket-nonce")
		ctx.Request.Header.Set("origin", "https://openpaths.io")
		if RealtimeCredentialFromSubprotocol(&ctx) != "sk-op-secret" {
			t.Fatalf("credential lost with %s", header)
		}
		if string(ctx.Request.Header.Peek("Sec-Websocket-Key")) != "websocket-nonce" {
			t.Fatal("upgrader cannot read normalized nonce")
		}
		if string(ctx.Request.Header.Peek("Origin")) != "https://openpaths.io" {
			t.Fatal("origin check cannot read proxy origin")
		}
	}
}
