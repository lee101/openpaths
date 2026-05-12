package deepseek

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestChatCompletion_AddsThinkingForV4Reasoning(t *testing.T) {
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
			Model:   "deepseek-v4-pro",
			Choices: []model.ChatChoice{{
				Index:   0,
				Message: &model.ChatMessage{Role: "assistant", Content: "ok"},
			}},
		})
	}))
	defer server.Close()

	p := New("test-key", server.URL)
	_, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
		Model:           "deepseek-v4-pro",
		Messages:        []model.ChatMessage{{Role: "user", Content: "Hi"}},
		ReasoningEffort: "high",
		Prefill:         "{",
		TaskTier:        "hard",
	})
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}

	thinking, ok := captured["thinking"].(map[string]any)
	if !ok {
		t.Fatalf("thinking missing from request: %#v", captured)
	}
	if thinking["type"] != "enabled" {
		t.Fatalf("thinking.type = %v, want enabled", thinking["type"])
	}
	if _, ok := captured["prefill"]; ok {
		t.Fatalf("prefill should be stripped: %#v", captured)
	}
	if _, ok := captured["task_tier"]; ok {
		t.Fatalf("task_tier should be stripped: %#v", captured)
	}
}

func TestChatCompletion_PreservesExplicitThinking(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.ChatCompletionResponse{
			ID:      "chatcmpl-1",
			Object:  "chat.completion",
			Created: 1,
			Model:   "deepseek-v4-flash",
			Choices: []model.ChatChoice{{
				Index:   0,
				Message: &model.ChatMessage{Role: "assistant", Content: "ok"},
			}},
		})
	}))
	defer server.Close()

	p := New("test-key", server.URL)
	_, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
		Model:    "deepseek-v4-flash",
		Messages: []model.ChatMessage{{Role: "user", Content: "Hi"}},
		Thinking: &model.ThinkingConfig{Type: "disabled"},
	})
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}

	thinking, ok := captured["thinking"].(map[string]any)
	if !ok {
		t.Fatalf("thinking missing from request: %#v", captured)
	}
	if thinking["type"] != "disabled" {
		t.Fatalf("thinking.type = %v, want disabled", thinking["type"])
	}
}
