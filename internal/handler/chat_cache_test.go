package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	responsecache "github.com/openpaths/openpaths/internal/cache"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
)

type cacheProvider struct {
	chatCalls   int32
	streamCalls int32
}

func (p *cacheProvider) Name() string { return "cache-provider" }

func (p *cacheProvider) ChatCompletion(_ context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	call := atomic.AddInt32(&p.chatCalls, 1)
	return &model.ChatCompletionResponse{
		ID:      "chatcmpl-cache-test-" + strconv.Itoa(int(call)),
		Object:  "chat.completion",
		Created: 123,
		Model:   req.Model,
		Choices: []model.ChatChoice{{
			Index: 0,
			Message: &model.ChatMessage{
				Role:    "assistant",
				Content: "cached response",
			},
			FinishReason: strPtr("stop"),
		}},
		Usage: &model.UsageInfo{},
	}, nil
}

func (p *cacheProvider) ChatCompletionStream(_ context.Context, _ *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	atomic.AddInt32(&p.streamCalls, 1)
	ch := make(chan provider.StreamEvent, 1)
	ch <- provider.StreamEvent{Done: true, Usage: &model.UsageInfo{}}
	close(ch)
	return ch, nil
}

func (p *cacheProvider) HealthCheck(context.Context) error { return nil }

func TestChatCompletionResponseCacheHit(t *testing.T) {
	h, p := newCacheTestHandler()
	body := mustJSON(t, map[string]any{
		"model": "cache-model",
		"messages": []map[string]string{
			{"role": "user", "content": "same"},
		},
		"temperature": 0.2,
	})

	first := runCacheTestRequest(h, body, "")
	if first.Response.StatusCode() != http.StatusOK {
		t.Fatalf("first status = %d body=%s", first.Response.StatusCode(), first.Response.Body())
	}
	if got := string(first.Response.Header.Peek("X-OpenPaths-Cache")); got != "miss" {
		t.Fatalf("first cache header = %q, want miss", got)
	}
	second := runCacheTestRequest(h, body, "")
	if second.Response.StatusCode() != http.StatusOK {
		t.Fatalf("second status = %d body=%s", second.Response.StatusCode(), second.Response.Body())
	}
	if got := string(second.Response.Header.Peek("X-OpenPaths-Cache")); got != "hit" {
		t.Fatalf("second cache header = %q, want hit", got)
	}
	if atomic.LoadInt32(&p.chatCalls) != 1 {
		t.Fatalf("provider chat calls = %d, want 1", atomic.LoadInt32(&p.chatCalls))
	}
	if string(first.Response.Body()) != string(second.Response.Body()) {
		t.Fatal("cache hit returned a different response body")
	}
}

func TestChatCompletionResponseCacheBypass(t *testing.T) {
	h, p := newCacheTestHandler()
	body := mustJSON(t, map[string]any{
		"model": "cache-model",
		"messages": []map[string]string{
			{"role": "user", "content": "same"},
		},
	})

	runCacheTestRequest(h, body, "no-cache")
	runCacheTestRequest(h, body, "no-cache")
	if atomic.LoadInt32(&p.chatCalls) != 2 {
		t.Fatalf("no-cache provider chat calls = %d, want 2", atomic.LoadInt32(&p.chatCalls))
	}

	streamBody := mustJSON(t, map[string]any{
		"model": "cache-model",
		"messages": []map[string]string{
			{"role": "user", "content": "same"},
		},
		"stream": true,
	})
	streamCtx := runCacheTestRequest(h, streamBody, "")
	if streamCtx.Response.StatusCode() != http.StatusOK {
		t.Fatalf("stream status = %d body=%s", streamCtx.Response.StatusCode(), streamCtx.Response.Body())
	}
	if got := string(streamCtx.Response.Header.Peek("X-OpenPaths-Cache")); got != "miss" {
		t.Fatalf("stream cache header = %q, want miss", got)
	}
	if atomic.LoadInt32(&p.streamCalls) != 1 {
		t.Fatalf("provider stream calls = %d, want 1", atomic.LoadInt32(&p.streamCalls))
	}
}

func newCacheTestHandler() (*ChatHandler, *cacheProvider) {
	p := &cacheProvider{}
	reg := provider.NewRegistry()
	reg.Register(p)
	models := []model.ModelConfig{{
		ID:              "cache-model",
		Provider:        p.Name(),
		ProviderModelID: "provider-cache-model",
	}}
	r := router.New(reg, models)
	pricing := billing.NewPricingTable(models)
	h := NewChatHandler(
		r,
		billing.NewEngine(pricing, nil),
		metrics.NewRecorder(metrics.NewCollector(nil, time.Hour)),
		nil,
		nil,
	).SetResponseCache(responsecache.NewResponseCache(responsecache.ResponseCacheOptions{
		TTL:          time.Minute,
		MaxEntries:   8,
		MaxValueSize: 4096,
	}))
	return h, p
}

func runCacheTestRequest(h *ChatHandler, body []byte, cacheControl string) *fasthttp.RequestCtx {
	var req fasthttp.Request
	req.Header.SetMethod(http.MethodPost)
	req.SetRequestURI("/v1/chat/completions")
	req.Header.SetContentType("application/json")
	if cacheControl != "" {
		req.Header.Set("Cache-Control", cacheControl)
	}
	req.SetBody(body)

	var ctx fasthttp.RequestCtx
	ctx.Init(&req, nil, nil)
	ctx.SetUserValue(middleware.CtxKeyUserID, "cache-test-user")
	h.HandleChatCompletion(&ctx)
	return &ctx
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}
