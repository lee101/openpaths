package anthropic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/promptcache"
)

const testModel = "claude-sonnet-4-20250514"

func eph(ttl string) map[string]any {
	m := map[string]any{"type": "ephemeral"}
	if ttl != "" {
		m["ttl"] = ttl
	}
	return m
}

func textPart(text string, cc any) map[string]any {
	m := map[string]any{"type": "text", "text": text}
	if cc != nil {
		m["cache_control"] = cc
	}
	return m
}

func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func optProvider(ttl promptcache.TTL) *AnthropicProvider {
	p := New("k", "")
	p.SetCacheOptimizer(promptcache.New(promptcache.Config{ColdDefault: ttl}))
	return p
}

func longTools() []model.Tool {
	return []model.Tool{
		{Type: "function", Function: &model.ToolFunction{Name: "a", Description: strings.Repeat("x", 3000)}},
		{Type: "function", Function: &model.ToolFunction{Name: "b", Description: strings.Repeat("y", 3000)}},
	}
}

func TestSystem_MergesAllMessagesInOrder(t *testing.T) {
	req := translateRequest(&model.ChatCompletionRequest{
		Model: testModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: "first"},
			{Role: "system", Content: []any{textPart("second", nil), textPart("third", nil)}},
			{Role: "user", Content: "hi"},
			{Role: "system", Content: "fourth"},
		},
	})
	blocks, ok := req.System.([]anthropicTextBlock)
	if !ok {
		t.Fatalf("system = %T, want []anthropicTextBlock", req.System)
	}
	var got []string
	for _, b := range blocks {
		got = append(got, b.Text)
	}
	if strings.Join(got, "|") != "first|second|third|fourth" {
		t.Fatalf("system texts = %v", got)
	}
	if len(req.Messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(req.Messages))
	}
}

func TestSystem_ArrayOnlyIsKept(t *testing.T) {
	req := translateRequest(&model.ChatCompletionRequest{
		Model: testModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: []any{textPart("only", nil)}},
			{Role: "user", Content: "hi"},
		},
	})
	if got := systemText(req.System); got != "only" {
		t.Fatalf("system = %q, want only", got)
	}
}

func TestSystem_SingleStringStaysPlain(t *testing.T) {
	req := translateRequest(&model.ChatCompletionRequest{
		Model: testModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: "plain"},
			{Role: "user", Content: "hi"},
		},
	})
	if s, ok := req.System.(string); !ok || s != "plain" {
		t.Fatalf("system = %#v, want plain string", req.System)
	}
}

func TestCacheControl_PreservedOnSystemTextAndTools(t *testing.T) {
	tools := longTools()
	tools[0].CacheControl = &model.CacheControl{Type: "ephemeral", TTL: "1h"}
	req := translateRequest(&model.ChatCompletionRequest{
		Model: testModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: []any{textPart("sys", eph("1h"))}},
			{Role: "user", Content: []any{textPart("q", eph("5m")), textPart("plain", nil)}},
		},
		Tools: tools,
	})
	sys := req.System.([]anthropicTextBlock)
	if sys[0].CacheControl == nil || sys[0].CacheControl.TTL != "1h" {
		t.Fatalf("system cc = %#v", sys[0].CacheControl)
	}
	parts := req.Messages[0].Content.([]any)
	if b := parts[0].(anthropicTextBlock); b.CacheControl == nil || b.CacheControl.Type != "ephemeral" || b.CacheControl.TTL != "5m" {
		t.Fatalf("text cc = %#v", b.CacheControl)
	}
	if b := parts[1].(anthropicTextBlock); b.CacheControl != nil {
		t.Fatalf("plain part got cc %#v", b.CacheControl)
	}
	if req.Tools[0].CacheControl == nil || req.Tools[0].CacheControl.TTL != "1h" {
		t.Fatalf("tool cc = %#v", req.Tools[0].CacheControl)
	}
	if req.Tools[1].CacheControl != nil {
		t.Fatalf("tool 1 got cc")
	}
}

func TestCacheControl_InvalidDropped(t *testing.T) {
	tools := longTools()
	tools[0].CacheControl = &model.CacheControl{Type: "persistent"}
	tools[1].CacheControl = &model.CacheControl{Type: "ephemeral", TTL: "2h"}
	req := translateRequest(&model.ChatCompletionRequest{
		Model: testModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: []any{textPart("sys", map[string]any{"type": "weird"})}},
			{Role: "user", Content: []any{
				textPart("a", map[string]any{"type": "ephemeral", "ttl": "10m"}),
				textPart("b", "ephemeral"),
				map[string]any{"type": "image", "cache_control": map[string]any{"type": "bad"}},
			}},
		},
		Tools: tools,
	})
	if strings.Contains(marshal(t, req), "cache_control") {
		t.Fatalf("invalid cache_control leaked: %s", marshal(t, req))
	}
}

func TestCacheControl_DefaultTTLOmitted(t *testing.T) {
	req := translateRequest(&model.ChatCompletionRequest{
		Model:    testModel,
		Messages: []model.ChatMessage{{Role: "user", Content: []any{textPart("a", eph(""))}}},
	})
	if !strings.Contains(marshal(t, req), `"cache_control":{"type":"ephemeral"}`) {
		t.Fatalf("body = %s", marshal(t, req))
	}
}

func TestCacheControl_MapBlocksValidatedWithoutMutatingCaller(t *testing.T) {
	img := map[string]any{"type": "image", "source": map[string]any{"type": "url", "url": "u"}, "cache_control": map[string]any{"type": "bad"}}
	ok := map[string]any{"type": "tool_result", "tool_use_id": "t", "cache_control": eph("")}
	chat := &model.ChatCompletionRequest{
		Model:    testModel,
		Messages: []model.ChatMessage{{Role: "user", Content: []any{img, ok}}},
	}
	req := translateRequest(chat)
	out := marshal(t, req)
	if strings.Count(out, "cache_control") != 1 {
		t.Fatalf("body = %s", out)
	}
	if _, still := img["cache_control"]; !still {
		t.Fatal("caller map was mutated")
	}
}

func TestCacheControl_CapsAtFourDroppingEarliest(t *testing.T) {
	tools := longTools()
	for i := range tools {
		tools[i].CacheControl = &model.CacheControl{Type: "ephemeral"}
	}
	req := translateRequest(&model.ChatCompletionRequest{
		Model: testModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: []any{textPart("s1", eph("")), textPart("s2", eph(""))}},
			{Role: "user", Content: []any{textPart("m1", eph("")), textPart("m2", eph(""))}},
		},
		Tools: tools,
	})
	p := New("k", "")
	p.applyCacheControl(req)
	if n := strings.Count(marshal(t, req), "cache_control"); n != 4 {
		t.Fatalf("breakpoints = %d, want 4", n)
	}
	if req.Tools[0].CacheControl != nil || req.Tools[1].CacheControl != nil {
		t.Fatalf("earliest (tools) should be dropped")
	}
	sys := req.System.([]anthropicTextBlock)
	if sys[0].CacheControl == nil || sys[1].CacheControl == nil {
		t.Fatalf("system breakpoints dropped: %#v", sys)
	}
	parts := req.Messages[0].Content.([]any)
	if parts[1].(anthropicTextBlock).CacheControl == nil {
		t.Fatal("last message breakpoint dropped")
	}
}

func TestCacheBeta_SetFor1hOnMessageBlock(t *testing.T) {
	req := translateRequest(&model.ChatCompletionRequest{
		Model:    testModel,
		Messages: []model.ChatMessage{{Role: "user", Content: []any{textPart("a", eph("1h"))}}},
	})
	hr, _ := http.NewRequest("POST", "http://x", nil)
	setCacheBeta(hr, req)
	if hr.Header.Get("anthropic-beta") != extendedCacheTTLBeta {
		t.Fatalf("beta header = %q", hr.Header.Get("anthropic-beta"))
	}
}

func TestCacheBeta_1hAfter5mStillSetsHeader(t *testing.T) {
	req := translateRequest(&model.ChatCompletionRequest{
		Model: testModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: []any{textPart("s", eph("5m"))}},
			{Role: "user", Content: []any{textPart("a", eph("1h"))}},
		},
	})
	hr, _ := http.NewRequest("POST", "http://x", nil)
	setCacheBeta(hr, req)
	if hr.Header.Get("anthropic-beta") == "" {
		t.Fatal("beta header missing when a later block asks for 1h")
	}
}

func TestCacheBeta_NotSetFor5m(t *testing.T) {
	req := translateRequest(&model.ChatCompletionRequest{
		Model:    testModel,
		Messages: []model.ChatMessage{{Role: "user", Content: []any{textPart("a", eph("5m"))}}},
	})
	hr, _ := http.NewRequest("POST", "http://x", nil)
	setCacheBeta(hr, req)
	if hr.Header.Get("anthropic-beta") != "" {
		t.Fatal("beta header set for 5m")
	}
}

func TestAuto_CoversToolsSystemAndRollingMessage(t *testing.T) {
	p := optProvider(promptcache.TTL1h)
	req := translateRequest(&model.ChatCompletionRequest{
		Model: testModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: "be brief"},
			{Role: "user", Content: "first"},
			{Role: "assistant", Content: "ok"},
			{Role: "user", Content: "second"},
		},
		Tools: longTools(),
	})
	p.applyCacheControl(req)
	if req.Tools[0].CacheControl != nil || req.Tools[1].CacheControl == nil || req.Tools[1].CacheControl.TTL != "1h" {
		t.Fatalf("tools cc = %#v %#v", req.Tools[0].CacheControl, req.Tools[1].CacheControl)
	}
	sys, ok := req.System.([]anthropicTextBlock)
	if !ok || sys[len(sys)-1].CacheControl == nil || sys[len(sys)-1].CacheControl.TTL != "1h" {
		t.Fatalf("system = %#v", req.System)
	}
	if req.Messages[0].Content != "first" {
		t.Fatalf("earlier user message modified: %#v", req.Messages[0].Content)
	}
	last := req.Messages[2].Content.([]anthropicTextBlock)
	if last[0].Text != "second" || last[0].CacheControl == nil || last[0].CacheControl.TTL != "1h" {
		t.Fatalf("rolling = %#v", last)
	}
	if n := strings.Count(marshal(t, req), "cache_control"); n != 3 {
		t.Fatalf("breakpoints = %d, want 3", n)
	}
}

func TestAuto_SystemOnlyLongPrefix(t *testing.T) {
	p := optProvider(promptcache.TTL5m)
	req := translateRequest(&model.ChatCompletionRequest{
		Model: testModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: strings.Repeat("cache me ", 600)},
			{Role: "user", Content: "hi"},
		},
	})
	p.applyCacheControl(req)
	if n := strings.Count(marshal(t, req), "cache_control"); n != 2 {
		t.Fatalf("breakpoints = %d, want system + rolling", n)
	}
}

func TestAuto_RollingOnLastBlockOfArrayUserMessage(t *testing.T) {
	p := optProvider(promptcache.TTL5m)
	req := translateRequest(&model.ChatCompletionRequest{
		Model: testModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: strings.Repeat("cache me ", 600)},
			{Role: "user", Content: []any{
				map[string]any{"type": "tool_result", "tool_use_id": "t1", "content": "r1"},
				map[string]any{"type": "tool_result", "tool_use_id": "t2", "content": "r2"},
			}},
		},
	})
	p.applyCacheControl(req)
	parts := req.Messages[0].Content.([]any)
	if _, has := parts[0].(map[string]any)["cache_control"]; has {
		t.Fatal("first tool_result got cc")
	}
	if _, has := parts[1].(map[string]any)["cache_control"]; !has {
		t.Fatal("last tool_result missing cc")
	}
}

func TestAuto_ClientBreakpointsSuppressAutomatic(t *testing.T) {
	p := optProvider(promptcache.TTL1h)
	req := translateRequest(&model.ChatCompletionRequest{
		Model: testModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: strings.Repeat("cache me ", 600)},
			{Role: "user", Content: []any{textPart("hello", eph("5m"))}},
		},
		Tools: longTools(),
	})
	p.applyCacheControl(req)
	if n := strings.Count(marshal(t, req), "cache_control"); n != 1 {
		t.Fatalf("breakpoints = %d, want only the client's: %s", n, marshal(t, req))
	}
	if strings.Contains(marshal(t, req), `"1h"`) {
		t.Fatal("automatic 1h leaked in")
	}
}

func TestAuto_ShortPrefixUnchangedByteForByte(t *testing.T) {
	p := optProvider(promptcache.TTL1h)
	chat := &model.ChatCompletionRequest{
		Model: testModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: "short"},
			{Role: "user", Content: strings.Repeat("long user text ", 1000)},
		},
	}
	req := translateRequest(chat)
	before := marshal(t, req)
	p.applyCacheControl(req)
	if marshal(t, req) != before {
		t.Fatal("short-prefix request changed")
	}
	if strings.Contains(before, "cache_control") {
		t.Fatal("unexpected cache_control")
	}
}

func TestAuto_NoOptimizerKeepsClientBreakpointsOnly(t *testing.T) {
	p := New("k", "")
	req := translateRequest(&model.ChatCompletionRequest{
		Model: testModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: strings.Repeat("cache me ", 600)},
			{Role: "user", Content: []any{textPart("hello", eph(""))}},
		},
	})
	p.applyCacheControl(req)
	if n := strings.Count(marshal(t, req), "cache_control"); n != 1 {
		t.Fatalf("breakpoints = %d, want 1", n)
	}
}

func TestAuto_NoOptimizerNoCacheControl(t *testing.T) {
	p := New("k", "")
	req := translateRequest(&model.ChatCompletionRequest{
		Model: testModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: strings.Repeat("cache me ", 600)},
			{Role: "user", Content: "hi"},
		},
		Tools: longTools(),
	})
	p.applyCacheControl(req)
	if strings.Contains(marshal(t, req), "cache_control") {
		t.Fatal("cache_control without optimizer")
	}
}

func TestChatCompletion_WireBodyAndBetaHeader(t *testing.T) {
	var gotBeta string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBeta = r.Header.Get("anthropic-beta")
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"m","type":"message","role":"assistant","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer srv.Close()
	p := New("k", srv.URL)
	_, err := p.ChatCompletion(t.Context(), &model.ChatCompletionRequest{
		Model: testModel,
		Messages: []model.ChatMessage{
			{Role: "system", Content: "a"},
			{Role: "system", Content: []any{textPart("b", eph("1h"))}},
			{Role: "user", Content: "hi"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBeta != extendedCacheTTLBeta {
		t.Fatalf("beta = %q", gotBeta)
	}
	sys, ok := gotBody["system"].([]any)
	if !ok || len(sys) != 2 {
		t.Fatalf("system on wire = %#v", gotBody["system"])
	}
	if sys[1].(map[string]any)["cache_control"] == nil {
		t.Fatal("cache_control missing on wire")
	}
}
