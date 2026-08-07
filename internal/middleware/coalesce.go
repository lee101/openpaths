package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/valyala/fasthttp"
)

type coalesceEntry struct {
	done      chan struct{}
	status    int
	body      []byte
	content   string
	expiresAt time.Time
}

type requestCoalescer struct {
	mu            sync.Mutex
	entries       map[string]*coalesceEntry
	ttl           time.Duration
	pruneInterval time.Duration
	nextPrune     time.Time
}

var coalesceablePathPrefixes = [][]byte{
	[]byte("/v1/images/generations"),
	[]byte("/v1/images/edits"),
	[]byte("/v1/3d/generations"),
	[]byte("/v1/videos/generations"),
	[]byte("/v1/videos/edits"),
	[]byte("/v1/videos/extensions"),
	[]byte("/v1/music/generations"),
	[]byte("/v1/audio/speech"),
	[]byte("/v1/tts"),
	[]byte("/v1/audio/transcriptions"),
	[]byte("/v1/stt"),
	[]byte("/v1/embeddings"),
}

func NewRequestCoalescer(ttl time.Duration) Middleware {
	pruneInterval := ttl / 4
	if pruneInterval <= 0 || pruneInterval > time.Minute {
		pruneInterval = time.Minute
	}
	c := &requestCoalescer{
		entries:       make(map[string]*coalesceEntry),
		ttl:           ttl,
		pruneInterval: pruneInterval,
		nextPrune:     time.Now().Add(pruneInterval),
	}
	return c.middleware
}

func (c *requestCoalescer) middleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		if !coalesceable(ctx) {
			next(ctx)
			return
		}
		key := coalesceKey(ctx)
		entry, owner := c.getOrCreate(key)
		if !owner {
			<-entry.done
			if entry.status >= 200 && entry.status < 300 && time.Now().Before(entry.expiresAt) {
				ctx.SetStatusCode(entry.status)
				if entry.content != "" {
					ctx.SetContentType(entry.content)
				}
				ctx.SetBody(entry.body)
				return
			}
			next(ctx)
			return
		}

		next(ctx)

		status := ctx.Response.StatusCode()
		if status >= 200 && status < 300 {
			entry.status = status
			entry.content = string(ctx.Response.Header.ContentType())
			entry.body = append(entry.body[:0], ctx.Response.Body()...)
			entry.expiresAt = time.Now().Add(c.ttl)
		} else {
			entry.expiresAt = time.Now()
		}
		close(entry.done)
		if status < 200 || status >= 300 {
			c.delete(key)
		}
	}
}

func (c *requestCoalescer) getOrCreate(key string) (*coalesceEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	if now.After(c.nextPrune) {
		c.pruneLocked(now)
		c.nextPrune = now.Add(c.pruneInterval)
	}
	if entry := c.entries[key]; entry != nil {
		select {
		case <-entry.done:
			if now.Before(entry.expiresAt) {
				return entry, false
			}
			delete(c.entries, key)
		default:
			return entry, false
		}
	}
	entry := &coalesceEntry{done: make(chan struct{})}
	c.entries[key] = entry
	return entry, true
}

func (c *requestCoalescer) delete(key string) {
	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()
}

func (c *requestCoalescer) pruneLocked(now time.Time) {
	for key, entry := range c.entries {
		select {
		case <-entry.done:
			if now.After(entry.expiresAt) {
				delete(c.entries, key)
			}
		default:
		}
	}
}

func coalesceable(ctx *fasthttp.RequestCtx) bool {
	if !ctx.IsPost() {
		return false
	}
	path := ctx.Path()
	for _, prefix := range coalesceablePathPrefixes {
		if bytes.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func coalesceKey(ctx *fasthttp.RequestCtx) string {
	sum := sha256.New()
	if userID, _ := ctx.UserValue(CtxKeyUserID).(string); userID != "" {
		sum.Write([]byte(userID))
	}
	sum.Write([]byte{0})
	if apiKey, ok := ctx.UserValue(CtxKeyAPIKey).(*model.APIKey); ok && apiKey != nil {
		sum.Write([]byte(apiKey.ID))
	}
	sum.Write([]byte{0})
	sum.Write(ctx.Method())
	sum.Write([]byte{0})
	sum.Write(ctx.Path())
	sum.Write([]byte{0})
	sum.Write(ctx.PostBody())
	return hex.EncodeToString(sum.Sum(nil))
}
