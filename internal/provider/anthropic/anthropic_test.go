package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpath/openpath/internal/model"
	"github.com/openpath/openpath/internal/provider"
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
