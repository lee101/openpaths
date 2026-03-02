package middleware

import (
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
