package cache

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

const (
	DefaultResponseCacheTTL          = 10 * time.Minute
	DefaultResponseCacheMaxEntries   = 2048
	DefaultResponseCacheMaxValueSize = 256 * 1024
)

type ResponseCacheOptions struct {
	TTL          time.Duration
	MaxEntries   int
	MaxValueSize int
	Clock        func() time.Time
}

type ResponseCache struct {
	mu           sync.Mutex
	ttl          time.Duration
	maxEntries   int
	maxValueSize int
	clock        func() time.Time

	items map[string]*list.Element
	lru   *list.List
}

type responseCacheEntry struct {
	key       string
	value     []byte
	expiresAt time.Time
}

func NewResponseCache(opts ResponseCacheOptions) *ResponseCache {
	if opts.TTL <= 0 {
		opts.TTL = DefaultResponseCacheTTL
	}
	if opts.MaxEntries <= 0 {
		opts.MaxEntries = DefaultResponseCacheMaxEntries
	}
	if opts.MaxValueSize <= 0 {
		opts.MaxValueSize = DefaultResponseCacheMaxValueSize
	}
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	return &ResponseCache{
		ttl:          opts.TTL,
		maxEntries:   opts.MaxEntries,
		maxValueSize: opts.MaxValueSize,
		clock:        opts.Clock,
		items:        make(map[string]*list.Element),
		lru:          list.New(),
	}
}

func NewResponseCacheFromEnv() *ResponseCache {
	if os.Getenv("OPENPATHS_RESPONSE_CACHE") == "0" {
		return nil
	}
	ttl := DefaultResponseCacheTTL
	if raw := os.Getenv("OPENPATHS_RESPONSE_CACHE_TTL"); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			ttl = parsed
		}
	}
	return NewResponseCache(ResponseCacheOptions{TTL: ttl})
}

func (c *ResponseCache) Get(key string) ([]byte, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.clock()
	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*responseCacheEntry)
		if now.After(entry.expiresAt) {
			c.removeElement(elem)
			return nil, false
		}
		c.lru.MoveToFront(elem)
		return append([]byte(nil), entry.value...), true
	}
	return nil, false
}

func (c *ResponseCache) Set(key string, value []byte) bool {
	if c == nil || len(value) > c.maxValueSize {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if elem, ok := c.items[key]; ok {
		entry := elem.Value.(*responseCacheEntry)
		entry.value = append(entry.value[:0], value...)
		entry.expiresAt = c.clock().Add(c.ttl)
		c.lru.MoveToFront(elem)
		return true
	}
	entry := &responseCacheEntry{
		key:       key,
		value:     append([]byte(nil), value...),
		expiresAt: c.clock().Add(c.ttl),
	}
	elem := c.lru.PushFront(entry)
	c.items[key] = elem
	for len(c.items) > c.maxEntries {
		c.removeElement(c.lru.Back())
	}
	return true
}

func (c *ResponseCache) removeElement(elem *list.Element) {
	if elem == nil {
		return
	}
	c.lru.Remove(elem)
	entry := elem.Value.(*responseCacheEntry)
	delete(c.items, entry.key)
}

type chatCompletionCacheKey struct {
	Model               string                `json:"model"`
	Messages            []model.ChatMessage   `json:"messages"`
	Temperature         *float64              `json:"temperature"`
	TopP                *float64              `json:"top_p"`
	MaxTokens           *int                  `json:"max_tokens"`
	MaxCompletionTokens *int                  `json:"max_completion_tokens"`
	ReasoningEffort     string                `json:"reasoning_effort"`
	TaskTier            string                `json:"task_tier"`
	Tools               []model.Tool          `json:"tools"`
	ResponseFormat      *model.ResponseFormat `json:"response_format"`
	Stop                []string              `json:"stop"`
	Prefill             string                `json:"prefill"`
}

func ChatCompletionKey(resolvedModelID string, req *model.ChatCompletionRequest) (string, error) {
	key := chatCompletionCacheKey{
		Model:               resolvedModelID,
		Messages:            req.Messages,
		Temperature:         req.Temperature,
		TopP:                req.TopP,
		MaxTokens:           req.MaxTokens,
		MaxCompletionTokens: req.MaxCompletionTokens,
		ReasoningEffort:     req.ReasoningEffort,
		TaskTier:            req.TaskTier,
		Tools:               req.Tools,
		ResponseFormat:      req.ResponseFormat,
		Stop:                req.Stop,
		Prefill:             req.Prefill,
	}
	data, err := json.Marshal(key)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func RequestCacheBypass(body []byte) bool {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return false
	}
	v, ok := raw["cache"]
	if !ok {
		return false
	}
	var enabled bool
	if err := json.Unmarshal(v, &enabled); err != nil {
		return false
	}
	return !enabled
}
