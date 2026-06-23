package google

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"
)

// GeminiCacheManager decides when to back a stable request prefix
// (systemInstruction + tools) with an explicit Gemini cachedContent resource,
// and tracks the resulting cache names + expiries.
//
// It is DEFAULT-OFF. Gemini 2.5 already does free implicit caching, so explicit
// caching only helps for large prefixes reused across gaps the implicit window
// misses — and it costs storage per token-hour. The manager therefore only
// creates a cache for prefixes that are both large enough and hot enough, and
// the provider always falls back to a full request if a cache is missing/stale.
type GeminiCacheManager struct {
	enabled bool
	apiKey  string
	baseURL string
	client  *http.Client

	minTokens int           // minimum prefix size to bother caching
	minHits   int           // observations within window before creating a cache
	ttl       time.Duration // cache lifetime
	window    time.Duration // reuse-tracking window
	now       func() time.Time

	mu     sync.Mutex
	stats  map[string][]int64    // prefixKey -> sorted unix-nano observations
	caches map[string]cacheEntry // prefixKey -> live cache
	stop   chan struct{}
	closed bool
}

type cacheEntry struct {
	name      string // "cachedContents/..."
	expiresAt int64  // unix-nano
}

// NewGeminiCacheManager builds a manager. When enabled is false every call is a
// cheap no-op and requests are sent unchanged.
func NewGeminiCacheManager(enabled bool, apiKey, baseURL string) *GeminiCacheManager {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	return &GeminiCacheManager{
		enabled:   enabled,
		apiKey:    apiKey,
		baseURL:   baseURL,
		client:    &http.Client{Timeout: 30 * time.Second},
		minTokens: 2048,
		minHits:   3,
		ttl:       30 * time.Minute,
		window:    time.Hour,
		now:       time.Now,
		stats:     make(map[string][]int64),
		caches:    make(map[string]cacheEntry),
		stop:      make(chan struct{}),
	}
}

// Ensure records that a request for prefixKey(model, system, tools) arrived, and
// returns the name of a usable explicit cache (creating one when warranted), or
// "" to send a normal request. The returned key identifies the prefix for a
// later Invalidate on a stale-cache error.
func (m *GeminiCacheManager) Ensure(ctx context.Context, modelName string, req *geminiRequest) (name, key string) {
	if m == nil || !m.enabled {
		return "", ""
	}
	key = cachePrefixKey(modelName, req)
	nowNS := m.now().UnixNano()

	m.mu.Lock()
	// Live cache already exists?
	if e, ok := m.caches[key]; ok {
		if e.expiresAt > nowNS {
			m.mu.Unlock()
			return e.name, key
		}
		delete(m.caches, key)
	}
	// Record observation and evaluate reuse.
	m.stats[key] = append(m.stats[key], nowNS)
	create := m.shouldCreateLocked(key, req, nowNS)
	m.mu.Unlock()

	if !create {
		return "", key
	}

	created, expires, err := m.createCache(ctx, modelName, req)
	if err != nil {
		log.Printf("gemini-cache: create failed for %s: %v", key, err)
		return "", key
	}
	m.mu.Lock()
	m.caches[key] = cacheEntry{name: created, expiresAt: expires}
	m.mu.Unlock()
	log.Printf("gemini-cache: created %s for prefix %s", created, key)
	return created, key
}

// shouldCreateLocked reports whether the prefix is large and hot enough to cache.
// Caller holds m.mu.
func (m *GeminiCacheManager) shouldCreateLocked(key string, req *geminiRequest, nowNS int64) bool {
	if estimatePrefixTokens(req) < m.minTokens {
		return false
	}
	cutoff := nowNS - m.window.Nanoseconds()
	m.stats[key] = pruneSortedNS(m.stats[key], cutoff)
	return len(m.stats[key]) >= m.minHits
}

// Invalidate drops a cache entry whose server-side resource is gone/expired.
func (m *GeminiCacheManager) Invalidate(key string) {
	if m == nil || key == "" {
		return
	}
	m.mu.Lock()
	delete(m.caches, key)
	m.mu.Unlock()
}

// Start prunes expired local cache entries periodically.
func (m *GeminiCacheManager) Start() {
	if m == nil || !m.enabled {
		return
	}
	go func() {
		ticker := time.NewTicker(m.ttl)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				m.prune()
			case <-m.stop:
				return
			}
		}
	}()
}

// Stop halts the prune loop.
func (m *GeminiCacheManager) Stop() {
	if m == nil {
		return
	}
	m.mu.Lock()
	if !m.closed {
		m.closed = true
		close(m.stop)
	}
	m.mu.Unlock()
}

func (m *GeminiCacheManager) prune() {
	nowNS := m.now().UnixNano()
	cutoff := nowNS - m.window.Nanoseconds()
	m.mu.Lock()
	for k, e := range m.caches {
		if e.expiresAt <= nowNS {
			delete(m.caches, k)
		}
	}
	for k := range m.stats {
		m.stats[k] = pruneSortedNS(m.stats[k], cutoff)
		if len(m.stats[k]) == 0 {
			delete(m.stats, k)
		}
	}
	m.mu.Unlock()
}

// cachedContentCreate is the POST body for creating an explicit cache.
type cachedContentCreate struct {
	Model             string           `json:"model"`
	SystemInstruction *geminiContent   `json:"systemInstruction,omitempty"`
	Tools             []geminiToolDecl `json:"tools,omitempty"`
	TTL               string           `json:"ttl"`
}

type cachedContentResponse struct {
	Name       string `json:"name"`
	ExpireTime string `json:"expireTime"`
}

func (m *GeminiCacheManager) createCache(ctx context.Context, modelName string, req *geminiRequest) (name string, expiresNS int64, err error) {
	payload := cachedContentCreate{
		Model:             "models/" + modelName,
		SystemInstruction: req.SystemInstruction,
		Tools:             req.Tools,
		TTL:               fmt.Sprintf("%ds", int(m.ttl.Seconds())),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, err
	}
	url := fmt.Sprintf("%s/v1beta/cachedContents?key=%s", m.baseURL, m.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := m.client.Do(httpReq)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", 0, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}
	var cc cachedContentResponse
	if err := json.Unmarshal(respBody, &cc); err != nil {
		return "", 0, err
	}
	if cc.Name == "" {
		return "", 0, fmt.Errorf("empty cache name in response")
	}
	// Trust our requested TTL for local expiry (server expireTime parsing is
	// optional; if present and parseable, prefer it).
	expiresNS = m.now().Add(m.ttl).UnixNano()
	if cc.ExpireTime != "" {
		if t, perr := time.Parse(time.RFC3339, cc.ExpireTime); perr == nil {
			expiresNS = t.UnixNano()
		}
	}
	return cc.Name, expiresNS, nil
}

// cachePrefixKey hashes the stable head (model + systemInstruction + tools).
func cachePrefixKey(modelName string, req *geminiRequest) string {
	h := sha256.New()
	io.WriteString(h, modelName)
	h.Write([]byte{0})
	if req.SystemInstruction != nil {
		if b, err := json.Marshal(req.SystemInstruction); err == nil {
			h.Write(b)
		}
	}
	h.Write([]byte{0})
	if len(req.Tools) > 0 {
		if b, err := json.Marshal(req.Tools); err == nil {
			h.Write(b)
		}
	}
	return hex.EncodeToString(h.Sum(nil)[:16])
}

// estimatePrefixTokens approximates the cacheable prefix size at ~4 chars/token.
func estimatePrefixTokens(req *geminiRequest) int {
	chars := 0
	if req.SystemInstruction != nil {
		if b, err := json.Marshal(req.SystemInstruction); err == nil {
			chars += len(b)
		}
	}
	if len(req.Tools) > 0 {
		if b, err := json.Marshal(req.Tools); err == nil {
			chars += len(b)
		}
	}
	return chars / 4
}

func pruneSortedNS(times []int64, cutoff int64) []int64 {
	i := sort.Search(len(times), func(i int) bool { return times[i] >= cutoff })
	if i == 0 {
		return times
	}
	return append(times[:0], times[i:]...)
}
