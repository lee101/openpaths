package middleware

import (
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/valyala/fasthttp"
)

func TestRateLimitAllows(t *testing.T) {
	mw := RateLimit()
	called := false

	handler := mw(func(ctx *fasthttp.RequestCtx) {
		called = true
	})

	ctx := &fasthttp.RequestCtx{}
	ctx.SetUserValue(CtxKeyAPIKey, &model.APIKey{ID: "key1", RateLimitRPM: 10})

	handler(ctx)

	if !called {
		t.Error("handler should have been called")
	}
	if ctx.Response.StatusCode() != 200 {
		t.Errorf("got status %d, want 200", ctx.Response.StatusCode())
	}
}

func TestRateLimitBlocks(t *testing.T) {
	mw := RateLimit()

	handler := mw(func(ctx *fasthttp.RequestCtx) {})

	// Make 3 requests with limit of 2
	for i := 0; i < 3; i++ {
		ctx := &fasthttp.RequestCtx{}
		ctx.SetUserValue(CtxKeyAPIKey, &model.APIKey{ID: "key-limited", RateLimitRPM: 2})
		handler(ctx)

		if i < 2 {
			if ctx.Response.StatusCode() == 429 {
				t.Errorf("request %d should not be rate limited", i)
			}
		} else {
			if ctx.Response.StatusCode() != 429 {
				t.Errorf("request %d should be rate limited, got status %d", i, ctx.Response.StatusCode())
			}
		}
	}
}

func TestRateLimitNoAPIKey(t *testing.T) {
	mw := RateLimit()
	called := false

	handler := mw(func(ctx *fasthttp.RequestCtx) {
		called = true
	})

	ctx := &fasthttp.RequestCtx{}
	handler(ctx)

	if !called {
		t.Error("handler should pass through when no API key")
	}
}

func TestRateLimiterAllow(t *testing.T) {
	rl := &rateLimiter{windows: make(map[string]*window)}

	// First request should be allowed
	if !rl.allow("key1", 2) {
		t.Error("first request should be allowed")
	}

	// Second request should be allowed
	if !rl.allow("key1", 2) {
		t.Error("second request should be allowed")
	}

	// Third request should be blocked
	if rl.allow("key1", 2) {
		t.Error("third request should be blocked")
	}

	// Different key should be independent
	if !rl.allow("key2", 2) {
		t.Error("different key should be allowed")
	}
}

func TestRateLimiterReset(t *testing.T) {
	rl := &rateLimiter{windows: make(map[string]*window)}

	// Fill up the limit
	rl.allow("key1", 1)
	if rl.allow("key1", 1) {
		t.Error("should be blocked")
	}

	// Manually expire the window
	rl.mu.Lock()
	rl.windows["key1"].resetAt = time.Now().Add(-1 * time.Second)
	rl.mu.Unlock()

	// Should be allowed again
	if !rl.allow("key1", 1) {
		t.Error("should be allowed after reset")
	}
}
