package nvidia

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestChatCompletion_AddsDeepSeekV4ProTemplateKwargs(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.ChatCompletionResponse{
			ID:      "chatcmpl-1",
			Object:  "chat.completion",
			Created: 1,
			Model:   "deepseek-ai/deepseek-v4-pro",
			Choices: []model.ChatChoice{{
				Index:   0,
				Message: &model.ChatMessage{Role: "assistant", Content: "ok"},
			}},
		})
	}))
	defer server.Close()

	p := New("test-key", server.URL)
	req := &model.ChatCompletionRequest{
		Model:           "deepseek-ai/deepseek-v4-pro",
		Messages:        []model.ChatMessage{{Role: "user", Content: "Hi"}},
		ReasoningEffort: "high",
		Prefill:         "{",
		TaskTier:        "hard",
		Thinking:        &model.ThinkingConfig{Type: "enabled"},
	}
	_, err := p.ChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}

	if req.ReasoningEffort != "high" {
		t.Fatalf("provider mutated caller request reasoning_effort = %q", req.ReasoningEffort)
	}
	if _, ok := captured["thinking"]; ok {
		t.Fatalf("top-level thinking should be stripped: %#v", captured)
	}
	if _, ok := captured["reasoning_effort"]; ok {
		t.Fatalf("top-level reasoning_effort should be moved into chat_template_kwargs: %#v", captured)
	}
	if _, ok := captured["prefill"]; ok {
		t.Fatalf("prefill should be stripped: %#v", captured)
	}
	if _, ok := captured["task_tier"]; ok {
		t.Fatalf("task_tier should be stripped: %#v", captured)
	}

	kwargs, ok := captured["chat_template_kwargs"].(map[string]any)
	if !ok {
		t.Fatalf("chat_template_kwargs missing from request: %#v", captured)
	}
	if kwargs["thinking"] != true {
		t.Fatalf("chat_template_kwargs.thinking = %v, want true", kwargs["thinking"])
	}
	if _, ok := kwargs["reasoning_effort"]; ok {
		t.Fatalf("undocumented chat_template_kwargs.reasoning_effort should be omitted: %#v", kwargs)
	}
}
