package middleware

import (
	"sync"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/model"
)

type rateLimiter struct {
	mu      sync.Mutex
	windows map[string]*window
}

type window struct {
	count   int
	resetAt time.Time
}

// RateLimit implements per-API-key sliding window rate limiting.
func RateLimit() Middleware {
	rl := &rateLimiter{windows: make(map[string]*window)}
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			apiKey, ok := ctx.UserValue(CtxKeyAPIKey).(*model.APIKey)
			if !ok {
				next(ctx)
				return
			}

			if !rl.allow(apiKey.ID, apiKey.RateLimitRPM) {
				ctx.SetStatusCode(429)
				ctx.SetContentType("application/json")
				ctx.SetBodyString(`{"error":{"message":"Rate limit exceeded","type":"rate_limit_error","code":"rate_limit_exceeded"}}`)
				return
			}

			next(ctx)
		}
	}
}

func (rl *rateLimiter) allow(keyID string, limitRPM int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	w, ok := rl.windows[keyID]
	if !ok || time.Now().After(w.resetAt) {
		rl.windows[keyID] = &window{count: 1, resetAt: time.Now().Add(time.Minute)}
		return true
	}
	if w.count >= limitRPM {
		return false
	}
	w.count++
	return true
}
