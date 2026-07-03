package cache

import (
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func TestChatCompletionKeyStableForMapOrderAndIgnoresUser(t *testing.T) {
	temp := 0.7
	maxTokens := 32
	reqA := &model.ChatCompletionRequest{
		Model:       "public-model",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		User:        "user-a",
		Messages: []model.ChatMessage{{
			Role: "user",
			Content: []any{
				map[string]any{"type": "text", "text": "hello"},
			},
		}},
		Tools: []model.Tool{{
			Type: "function",
			Function: model.ToolFunction{
				Name:       "lookup",
				Parameters: map[string]any{"b": 2, "a": 1},
			},
		}},
	}
	reqB := &model.ChatCompletionRequest{
		Model:       "public-model",
		Temperature: &temp,
		MaxTokens:   &maxTokens,
		User:        "user-b",
		Messages: []model.ChatMessage{{
			Role: "user",
			Content: []any{
				map[string]any{"text": "hello", "type": "text"},
			},
		}},
		Tools: []model.Tool{{
			Type: "function",
			Function: model.ToolFunction{
				Name:       "lookup",
				Parameters: map[string]any{"a": 1, "b": 2},
			},
		}},
	}

	keyA, err := ChatCompletionKey("provider-model", reqA)
	if err != nil {
		t.Fatalf("key A: %v", err)
	}
	keyB, err := ChatCompletionKey("provider-model", reqB)
	if err != nil {
		t.Fatalf("key B: %v", err)
	}
	if keyA != keyB {
		t.Fatalf("keys differ for semantically identical requests: %s != %s", keyA, keyB)
	}

	keyC, err := ChatCompletionKey("other-provider-model", reqB)
	if err != nil {
		t.Fatalf("key C: %v", err)
	}
	if keyA == keyC {
		t.Fatal("key did not include resolved provider model")
	}
}

func TestResponseCacheTTLAndValueCopy(t *testing.T) {
	now := time.Unix(100, 0)
	c := NewResponseCache(ResponseCacheOptions{
		TTL:        time.Second,
		MaxEntries: 4,
		Clock: func() time.Time {
			return now
		},
	})

	value := []byte(`{"ok":true}`)
	if !c.Set("k", value) {
		t.Fatal("set returned false")
	}
	value[6] = 'f'

	got, ok := c.Get("k")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got) != `{"ok":true}` {
		t.Fatalf("cached value mutated: %s", got)
	}
	got[6] = 'f'
	got, ok = c.Get("k")
	if !ok || string(got) != `{"ok":true}` {
		t.Fatalf("cache returned non-copy on get: hit=%v value=%s", ok, got)
	}

	now = now.Add(2 * time.Second)
	if _, ok := c.Get("k"); ok {
		t.Fatal("expected cache miss after TTL")
	}
}

func TestResponseCacheLRUEvictionAndMaxValueSize(t *testing.T) {
	c := NewResponseCache(ResponseCacheOptions{
		TTL:          time.Minute,
		MaxEntries:   2,
		MaxValueSize: 4,
	})

	if !c.Set("a", []byte("one")) || !c.Set("b", []byte("two")) {
		t.Fatal("initial set failed")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatal("expected a hit before eviction")
	}
	c.Set("c", []byte("tri"))
	if _, ok := c.Get("b"); ok {
		t.Fatal("expected least recently used entry b to be evicted")
	}
	if _, ok := c.Get("a"); !ok {
		t.Fatal("expected recently used entry a to remain")
	}
	if c.Set("large", []byte("12345")) {
		t.Fatal("oversized value should not be cached")
	}
	if _, ok := c.Get("large"); ok {
		t.Fatal("oversized value was cached")
	}
}

func TestRequestCacheBypass(t *testing.T) {
	if !RequestCacheBypass([]byte(`{"model":"m","cache":false}`)) {
		t.Fatal("cache:false should bypass")
	}
	if RequestCacheBypass([]byte(`{"model":"m","cache":true}`)) {
		t.Fatal("cache:true should not bypass")
	}
	if RequestCacheBypass([]byte(`{"model":"m"}`)) {
		t.Fatal("missing cache field should not bypass")
	}
}
