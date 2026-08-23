package middleware

import (
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/model"
)

const rlShards = 64

type rateLimiter struct {
	shards [rlShards]rlShard
}

type rlShard struct {
	mu      sync.Mutex
	windows map[string]*window
}

type window struct {
	count   int
	resetAt time.Time
}

func RateLimit() Middleware {
	rl := &rateLimiter{}
	for i := range rl.shards {
		rl.shards[i].windows = make(map[string]*window)
	}

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			apiKey, ok := ctx.UserValue(CtxKeyAPIKey).(*model.APIKey)
			if !ok {
				next(ctx)
				return
			}

			limit := apiKey.RateLimitRPM
			if limit <= 0 {
				limit = 1
			}
			// Enforce both the key limit and an account-wide ceiling. Otherwise a
			// user can create many keys and multiply the default allowance.
			userLimit := envRateLimit("USER_RATE_LIMIT_RPM", 120)
			if !rl.allow(apiKey.ID, limit) || !rl.allow("user:"+apiKey.UserID, userLimit) {
				ctx.SetStatusCode(429)
				ctx.SetContentType("application/json")
				ctx.SetBodyString(`{"error":{"message":"Rate limit exceeded","type":"rate_limit_error","code":"rate_limit_exceeded"}}`)
				return
			}

			next(ctx)
		}
	}
}

func envRateLimit(name string, def int) int {
	if raw := os.Getenv(name); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			return v
		}
	}
	return def
}

func (rl *rateLimiter) shard(key string) *rlShard {
	h := uint32(0)
	for i := 0; i < len(key); i++ {
		h = h*31 + uint32(key[i])
	}
	return &rl.shards[h%rlShards]
}

func (rl *rateLimiter) allow(keyID string, limitRPM int) bool {
	s := rl.shard(keyID)
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	w, ok := s.windows[keyID]
	if !ok || now.After(w.resetAt) {
		s.windows[keyID] = &window{count: 1, resetAt: now.Add(time.Minute)}
		return true
	}
	if w.count >= limitRPM {
		return false
	}
	w.count++
	return true
}

func (rl *rateLimiter) cleanup() {
	now := time.Now()
	for i := range rl.shards {
		s := &rl.shards[i]
		s.mu.Lock()
		for k, w := range s.windows {
			if now.After(w.resetAt) {
				delete(s.windows, k)
			}
		}
		s.mu.Unlock()
	}
}
