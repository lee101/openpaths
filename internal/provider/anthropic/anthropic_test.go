package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/promptcache"
	"github.com/openpaths/openpaths/internal/provider"
)

func TestAnthropicProviderName(t *testing.T) {
	p := New("test-key", "")
	if p.Name() != "anthropic" {
		t.Errorf("got name %q, want %q", p.Name(), "anthropic")
	}
}

func TestChatCompletionSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("unexpected api key header: %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != anthropicVersion {
			t.Errorf("unexpected version header: %s", r.Header.Get("anthropic-version"))
		}

		// Verify the request body was translated correctly
		var anthReq anthropicRequest
		json.NewDecoder(r.Body).Decode(&anthReq)

		if anthReq.System != "You are helpful." {
			t.Errorf("system not extracted, got %q", anthReq.System)
		}
		if len(anthReq.Messages) != 1 {
			t.Errorf("got %d messages, want 1 (system should be extracted)", len(anthReq.Messages))
		}

		resp := anthropicResponse{
			ID:         "msg_123",
			Type:       "message",
			Role:       "assistant",
			Content:    []anthropicContent{{Type: "text", Text: "Hello!"}},
			Model:      "claude-sonnet-4-20250514",
			StopReason: "end_turn",
			Usage:      anthropicUsage{InputTokens: 10, OutputTokens: 5},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := New("test-key", server.URL)
	resp, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
		Model: "claude-sonnet-4-20250514",
		Messages: []model.ChatMessage{
			{Role: "system", Content: "You are helpful."},
			{Role: "user", Content: "Hi"},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Object != "chat.completion" {
		t.Errorf("got object %q, want %q", resp.Object, "chat.completion")
	}
	if resp.Usage.PromptTokens != 10 {
		t.Errorf("got prompt tokens %d, want 10", resp.Usage.PromptTokens)
	}
	if resp.Usage.CompletionTokens != 5 {
		t.Errorf("got completion tokens %d, want 5", resp.Usage.CompletionTokens)
	}

	if len(resp.Choices) != 1 {
		t.Fatalf("got %d choices, want 1", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Hello!" {
		t.Errorf("got content %q, want %q", resp.Choices[0].Message.Content, "Hello!")
	}
	if *resp.Choices[0].FinishReason != "stop" {
		t.Errorf("got finish reason %q, want %q", *resp.Choices[0].FinishReason, "stop")
	}
}

func TestChatCompletionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer server.Close()

	p := New("test-key", server.URL)
	_, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
		Model:    "claude-sonnet-4-20250514",
		Messages: []model.ChatMessage{{Role: "user", Content: "Hi"}},
	})

	if err == nil {
		t.Fatal("expected error")
	}

	pe, ok := err.(*provider.ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if pe.StatusCode != 429 {
		t.Errorf("got status %d, want 429", pe.StatusCode)
	}
	if !pe.Retryable {
		t.Error("expected 429 to be retryable")
	}
}

func TestTranslateRequest(t *testing.T) {
	maxTokens := 100
	req := &model.ChatCompletionRequest{
		Model: "claude-sonnet-4-20250514",
		Messages: []model.ChatMessage{
			{Role: "system", Content: "Be concise."},
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi!"},
			{Role: "user", Content: "How are you?"},
		},
		MaxTokens: &maxTokens,
	}

	anthReq := translateRequest(req)

	if anthReq.System != "Be concise." {
		t.Errorf("system not extracted, got %q", anthReq.System)
	}
	if len(anthReq.Messages) != 3 {
		t.Errorf("got %d messages, want 3", len(anthReq.Messages))
	}
	if anthReq.MaxTokens != 100 {
		t.Errorf("got max tokens %d, want 100", anthReq.MaxTokens)
	}
}

func TestTranslateRequestDefaults(t *testing.T) {
	req := &model.ChatCompletionRequest{
		Model:    "claude-sonnet-4-20250514",
		Messages: []model.ChatMessage{{Role: "user", Content: "Hi"}},
	}

	anthReq := translateRequest(req)

	if anthReq.MaxTokens != 4096 {
		t.Errorf("got default max tokens %d, want 4096", anthReq.MaxTokens)
	}
}

func TestTranslateRequest_MapsOpenAIImageURLBlocks(t *testing.T) {
	req := &model.ChatCompletionRequest{
		Model: "claude-sonnet-4-20250514",
		Messages: []model.ChatMessage{{
			Role: "user",
			Content: []any{
				map[string]any{"type": "text", "text": "What is in this image?"},
				map[string]any{"type": "image_url", "image_url": map[string]any{"url": "https://example.com/cat.png"}},
			},
		}},
	}

	anthReq := translateRequest(req)
	blocks, ok := anthReq.Messages[0].Content.([]any)
	if !ok {
		t.Fatalf("content type = %T, want []any", anthReq.Messages[0].Content)
	}
	if got, ok := blocks[0].(anthropicTextBlock); !ok || got.Text != "What is in this image?" {
		t.Fatalf("text block = %#v, want Anthropic text block", blocks[0])
	}
	img, ok := blocks[1].(anthropicImageBlock)
	if !ok {
		t.Fatalf("image block = %#v, want Anthropic image block", blocks[1])
	}
	if img.Source.Type != "url" || img.Source.URL != "https://example.com/cat.png" {
		t.Fatalf("image source = %#v, want URL source", img.Source)
	}
}

func TestApplyCacheControl_AttachesToSystemBlock(t *testing.T) {
	p := New("test-key", "")
	p.SetCacheOptimizer(promptcache.New(promptcache.Config{ColdDefault: promptcache.TTL1h}))
	longSystem := strings.Repeat("cache me ", 600)
	anthReq := translateRequest(&model.ChatCompletionRequest{
		Model: "claude-sonnet-4-20250514",
		Messages: []model.ChatMessage{
			{Role: "system", Content: longSystem},
			{Role: "user", Content: "Hi"},
		},
	})

	p.applyCacheControl(anthReq)

	blocks, ok := anthReq.System.([]anthropicTextBlock)
	if !ok {
		t.Fatalf("system = %T, want []anthropicTextBlock", anthReq.System)
	}
	if len(blocks) != 1 || blocks[0].CacheControl == nil {
		t.Fatalf("system blocks = %#v, want cache_control on block", blocks)
	}
	if blocks[0].CacheControl.TTL != "1h" {
		t.Fatalf("ttl = %q, want 1h", blocks[0].CacheControl.TTL)
	}
	req, err := http.NewRequest(http.MethodPost, "https://example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	setCacheBeta(req, anthReq)
	if got := req.Header.Get("anthropic-beta"); got != extendedCacheTTLBeta {
		t.Fatalf("anthropic-beta = %q, want %q", got, extendedCacheTTLBeta)
	}
}

func TestTranslateRequest_MapsReasoningEffortToThinkingBudget(t *testing.T) {
	maxTokens := 9000
	req := &model.ChatCompletionRequest{
		Model:           "claude-sonnet-4-20250514",
		Messages:        []model.ChatMessage{{Role: "user", Content: "Think carefully."}},
		MaxTokens:       &maxTokens,
		ReasoningEffort: "medium",
	}

	anthReq := translateRequest(req)
	if anthReq.Thinking == nil {
		t.Fatal("expected thinking config to be set")
	}
	if anthReq.Thinking.Type != "enabled" {
		t.Fatalf("thinking type = %q, want %q", anthReq.Thinking.Type, "enabled")
	}
	if anthReq.Thinking.BudgetTokens != 4096 {
		t.Fatalf("thinking budget = %d, want %d", anthReq.Thinking.BudgetTokens, 4096)
	}
}

func TestTranslateRequest_UsesAdaptiveThinkingAndEffortForCurrentModels(t *testing.T) {
	for _, tc := range []struct {
		model        string
		wantThinking string
	}{
		{model: "claude-sonnet-5", wantThinking: ""},
		{model: "claude-fable-5", wantThinking: ""},
		{model: "claude-opus-4-8", wantThinking: "adaptive"},
		{model: "claude-sonnet-4-6", wantThinking: "adaptive"},
	} {
		t.Run(tc.model, func(t *testing.T) {
			temp := 0.7
			req := translateRequest(&model.ChatCompletionRequest{
				Model: tc.model, Messages: []model.ChatMessage{{Role: "user", Content: "Work carefully."}},
				ReasoningEffort: "medium", Temperature: &temp,
			})
			if req.OutputConfig == nil || req.OutputConfig.Effort != "medium" {
				t.Fatalf("output_config = %#v, want medium effort", req.OutputConfig)
			}
			if tc.wantThinking == "" {
				if req.Thinking != nil {
					t.Fatalf("thinking = %#v, want model default", req.Thinking)
				}
			} else if req.Thinking == nil || req.Thinking.Type != tc.wantThinking {
				t.Fatalf("thinking = %#v, want %q", req.Thinking, tc.wantThinking)
			}
			if req.Temperature != nil {
				t.Fatal("adaptive model must not receive temperature")
			}
		})
	}
}

func TestTranslateRequest_DisablesSonnet5ThinkingForNone(t *testing.T) {
	req := translateRequest(&model.ChatCompletionRequest{
		Model: "claude-sonnet-5", Messages: []model.ChatMessage{{Role: "user", Content: "Say hi."}}, ReasoningEffort: "none",
	})
	if req.Thinking == nil || req.Thinking.Type != "disabled" {
		t.Fatalf("thinking = %#v, want disabled", req.Thinking)
	}
}

func TestTranslateRequest_AutoEnablesAdaptiveThinkingForOpus(t *testing.T) {
	req := translateRequest(&model.ChatCompletionRequest{
		Model: "claude-opus-4-8", Messages: []model.ChatMessage{{Role: "user", Content: "Choose the right depth."}}, ReasoningEffort: "auto",
	})
	if req.Thinking == nil || req.Thinking.Type != "adaptive" {
		t.Fatalf("thinking = %#v, want adaptive", req.Thinking)
	}
	if req.OutputConfig != nil {
		t.Fatalf("output_config = %#v, want provider default for auto", req.OutputConfig)
	}
}

func TestTranslateRequest_PassesXHighThroughWhenSupported(t *testing.T) {
	for _, id := range []string{"claude-opus-5", "claude-opus-4-8", "claude-sonnet-5", "claude-fable-5"} {
		req := translateRequest(&model.ChatCompletionRequest{
			Model: id, Messages: []model.ChatMessage{{Role: "user", Content: "Reason deeply."}}, ReasoningEffort: "xhigh",
		})
		if req.OutputConfig == nil || req.OutputConfig.Effort != "xhigh" {
			t.Fatalf("%s: output_config = %#v, want xhigh", id, req.OutputConfig)
		}
	}
}

func TestTranslateRequest_NormalizesXHighToAnthropicMax(t *testing.T) {
	req := translateRequest(&model.ChatCompletionRequest{
		Model: "claude-opus-4-6", Messages: []model.ChatMessage{{Role: "user", Content: "Reason deeply."}}, ReasoningEffort: "xhigh",
	})
	if req.OutputConfig == nil || req.OutputConfig.Effort != "max" {
		t.Fatalf("output_config = %#v, want max", req.OutputConfig)
	}
}

func TestTranslateRequest_Opus5ThinksByDefault(t *testing.T) {
	// Opus 5 runs adaptive thinking when no thinking field is sent, so the
	// explicit switch is redundant and "none" has to disable it outright.
	req := translateRequest(&model.ChatCompletionRequest{
		Model: "claude-opus-5", Messages: []model.ChatMessage{{Role: "user", Content: "Think."}}, ReasoningEffort: "high",
	})
	if req.Thinking != nil {
		t.Fatalf("thinking = %#v, want nil (adaptive is the Opus 5 default)", req.Thinking)
	}
	if req.OutputConfig == nil || req.OutputConfig.Effort != "high" {
		t.Fatalf("output_config = %#v, want high", req.OutputConfig)
	}

	off := translateRequest(&model.ChatCompletionRequest{
		Model: "claude-opus-5", Messages: []model.ChatMessage{{Role: "user", Content: "Answer."}}, ReasoningEffort: "none",
	})
	if off.Thinking == nil || off.Thinking.Type != "disabled" {
		t.Fatalf("thinking = %#v, want disabled", off.Thinking)
	}
	if off.OutputConfig != nil {
		t.Fatalf("output_config = %#v, want nil — disabled thinking is rejected above high effort", off.OutputConfig)
	}
}

func TestTranslateRequest_ClampsThinkingBudgetToMaxTokens(t *testing.T) {
	maxTokens := 1500
	req := &model.ChatCompletionRequest{
		Model:           "claude-sonnet-4-20250514",
		Messages:        []model.ChatMessage{{Role: "user", Content: "Think carefully."}},
		MaxTokens:       &maxTokens,
		ReasoningEffort: "high",
	}

	anthReq := translateRequest(req)
	if anthReq.Thinking == nil {
		t.Fatal("expected thinking config to be set")
	}
	if anthReq.Thinking.BudgetTokens != 1499 {
		t.Fatalf("thinking budget = %d, want %d", anthReq.Thinking.BudgetTokens, 1499)
	}
}

func TestTranslateResponse(t *testing.T) {
	resp := &anthropicResponse{
		ID:         "msg_123",
		Type:       "message",
		Role:       "assistant",
		Content:    []anthropicContent{{Type: "text", Text: "Hello!"}},
		Model:      "claude-sonnet-4-20250514",
		StopReason: "end_turn",
		Usage:      anthropicUsage{InputTokens: 10, OutputTokens: 5},
	}

	openaiResp := translateResponse(resp, "claude-sonnet")

	if openaiResp.Model != "claude-sonnet" {
		t.Errorf("model not mapped back, got %q", openaiResp.Model)
	}
	if openaiResp.Object != "chat.completion" {
		t.Errorf("got object %q", openaiResp.Object)
	}
	if openaiResp.Usage.TotalTokens != 15 {
		t.Errorf("got total tokens %d, want 15", openaiResp.Usage.TotalTokens)
	}
}

func TestMapStopReason(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"end_turn", "stop"},
		{"max_tokens", "length"},
		{"tool_use", "tool_calls"},
		{"unknown", "stop"},
	}

	for _, tc := range tests {
		got := mapStopReason(tc.input)
		if got != tc.expected {
			t.Errorf("mapStopReason(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestTranslateRequest_PrefillAppendsAssistantTurn(t *testing.T) {
	req := &model.ChatCompletionRequest{
		Model:    "claude-opus-4-7",
		Messages: []model.ChatMessage{{Role: "user", Content: "Return JSON"}},
		Prefill:  "{",
	}

	anthReq := translateRequest(req)

	if len(anthReq.Messages) != 2 {
		t.Fatalf("got %d messages, want 2 (user + prefill assistant)", len(anthReq.Messages))
	}
	last := anthReq.Messages[len(anthReq.Messages)-1]
	if last.Role != "assistant" {
		t.Errorf("last role = %q, want assistant", last.Role)
	}
	if s, ok := last.Content.(string); !ok || s != "{" {
		t.Errorf("last content = %v, want %q", last.Content, "{")
	}
}

func TestTranslateRequest_PrefillSkipsIfAssistantAlreadyLast(t *testing.T) {
	req := &model.ChatCompletionRequest{
		Model: "claude-opus-4-7",
		Messages: []model.ChatMessage{
			{Role: "user", Content: "Return JSON"},
			{Role: "assistant", Content: "{\"chart_type\": \""},
		},
		Prefill: "{",
	}

	anthReq := translateRequest(req)

	if len(anthReq.Messages) != 2 {
		t.Fatalf("got %d messages, want 2 (existing assistant kept, prefill ignored)", len(anthReq.Messages))
	}
	last := anthReq.Messages[len(anthReq.Messages)-1]
	if s, ok := last.Content.(string); !ok || s != "{\"chart_type\": \"" {
		t.Errorf("existing assistant content was overwritten: %v", last.Content)
	}
}

func TestToolCallTranslation(t *testing.T) {
	resp := &anthropicResponse{
		ID:   "msg_123",
		Role: "assistant",
		Content: []anthropicContent{
			{Type: "text", Text: "Let me check."},
			{Type: "tool_use", ID: "call_1", Name: "get_weather", Input: map[string]any{"city": "SF"}},
		},
		StopReason: "tool_use",
		Usage:      anthropicUsage{InputTokens: 20, OutputTokens: 15},
	}

	openaiResp := translateResponse(resp, "claude-sonnet")

	if len(openaiResp.Choices) != 1 {
		t.Fatalf("got %d choices", len(openaiResp.Choices))
	}

	msg := openaiResp.Choices[0].Message
	if msg.Content != "Let me check." {
		t.Errorf("got content %q", msg.Content)
	}
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("got %d tool calls, want 1", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].Function.Name != "get_weather" {
		t.Errorf("got tool name %q", msg.ToolCalls[0].Function.Name)
	}
	if *openaiResp.Choices[0].FinishReason != "tool_calls" {
		t.Errorf("got finish reason %q", *openaiResp.Choices[0].FinishReason)
	}
}
