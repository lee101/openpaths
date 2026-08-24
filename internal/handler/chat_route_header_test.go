package handler

import (
	"strings"
	"testing"
)

func TestChatCompletionRouteHeaderDirectModel(t *testing.T) {
	h, _ := newCacheTestHandler()
	body := mustJSON(t, map[string]any{
		"model": "cache-model",
		"messages": []map[string]string{
			{"role": "user", "content": "route header"},
		},
	})

	ctx := runCacheTestRequest(h, body, "")
	if ctx.Response.StatusCode() != 200 {
		t.Fatalf("status = %d body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	got := string(ctx.Response.Header.Peek("X-OpenPaths-Route"))
	want := "model=cache-model; provider=cache-provider; strategy=price; byok=false"
	if got != want {
		t.Fatalf("X-OpenPaths-Route = %q, want %q", got, want)
	}
}

func TestChatCompletionRouteHeaderStrategyEcho(t *testing.T) {
	h, _ := newCacheTestHandler()
	body := mustJSON(t, map[string]any{
		"model":            "cache-model",
		"routing_strategy": "Config",
		"messages": []map[string]string{
			{"role": "user", "content": "route header"},
		},
	})

	ctx := runCacheTestRequest(h, body, "")
	if ctx.Response.StatusCode() != 200 {
		t.Fatalf("status = %d body=%s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	got := string(ctx.Response.Header.Peek("X-OpenPaths-Route"))
	if !strings.Contains(got, "strategy=config") {
		t.Fatalf("X-OpenPaths-Route = %q, want normalized strategy=config", got)
	}
	if strings.Contains(got, "requested=") {
		t.Fatalf("X-OpenPaths-Route = %q, direct model must not set requested=", got)
	}
}
