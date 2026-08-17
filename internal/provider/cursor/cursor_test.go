package cursor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestParseModelSelection(t *testing.T) {
	id, fast := parseModelSelection("composer-2.5-fast")
	if id != "composer-2.5" || !fast {
		t.Fatalf("got id=%q fast=%v", id, fast)
	}
	id, fast = parseModelSelection("composer-2.5")
	if id != "composer-2.5" || fast {
		t.Fatalf("got id=%q fast=%v", id, fast)
	}
	id, fast = parseModelSelection("grok-4.6-fast")
	if id != "grok-4.6" || !fast {
		t.Fatalf("got id=%q fast=%v", id, fast)
	}
}

func TestCursorGrokEffort(t *testing.T) {
	tests := []struct {
		model, requested, want string
	}{
		{"grok-4.5", "high", "high"},
		{"grok-4.5", "xhigh", "high"},
		{"grok-4.6", "xhigh", "xhigh"},
		{"grok-4.6", "minimal", "low"},
		{"composer-2.5", "high", ""},
	}
	for _, tt := range tests {
		if got := cursorGrokEffort(tt.model, tt.requested); got != tt.want {
			t.Errorf("cursorGrokEffort(%q, %q) = %q, want %q", tt.model, tt.requested, got, tt.want)
		}
	}
}

func TestMessagesToPrompt(t *testing.T) {
	got := messagesToPrompt([]model.ChatMessage{
		{Role: "system", Content: "be terse"},
		{Role: "user", Content: "say hi"},
	})
	if !strings.Contains(got, "System: be terse") || !strings.Contains(got, "User: say hi") {
		t.Fatalf("unexpected prompt: %q", got)
	}
}

func TestChatCompletionSuccess(t *testing.T) {
	var createBody createAgentRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/agents":
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"agent": map[string]string{"id": "bc-test"},
				"run":   map[string]string{"id": "run-test", "status": "CREATING"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/agents/bc-test/runs/run-test":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     "run-test",
				"status": "FINISHED",
				"result": "hi",
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/agents/bc-test":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"bc-test"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key", server.URL)
	resp, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
		Model: "composer-2.5-fast",
		Messages: []model.ChatMessage{
			{Role: "user", Content: "say hi nothing else"},
		},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if createBody.Model.ID != "composer-2.5" || len(createBody.Model.Params) != 1 || createBody.Model.Params[0].Value != "true" {
		t.Fatalf("unexpected create body: %+v", createBody)
	}
	if resp.Choices[0].Message.Content != "hi" {
		t.Fatalf("got content %v", resp.Choices[0].Message.Content)
	}
}

func TestChatCompletionStandard(t *testing.T) {
	var createBody createAgentRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/agents":
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"agent": map[string]string{"id": "bc-std"},
				"run":   map[string]string{"id": "run-std", "status": "CREATING"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/agents/bc-std/runs/run-std":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     "run-std",
				"status": "FINISHED",
				"result": "ok",
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/agents/bc-std":
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key", server.URL)
	_, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
		Model:    "composer-2.5",
		Messages: []model.ChatMessage{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if createBody.Model.ID != "composer-2.5" || len(createBody.Model.Params) != 0 {
		t.Fatalf("unexpected create body: %+v", createBody)
	}
}

func TestChatCompletionGrokParams(t *testing.T) {
	var createBody createAgentRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/agents":
			if err := json.NewDecoder(r.Body).Decode(&createBody); err != nil {
				t.Fatalf("decode create: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"agent": map[string]string{"id": "bc-grok"},
				"run":   map[string]string{"id": "run-grok", "status": "CREATING"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/agents/bc-grok/runs/run-grok":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "run-grok", "status": "FINISHED", "result": "grok ok",
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/agents/bc-grok":
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key", server.URL)
	_, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
		Model:           "grok-4.6-fast",
		ReasoningEffort: "xhigh",
		Messages:        []model.ChatMessage{{Role: "user", Content: "say hi"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion: %v", err)
	}
	if createBody.Model.ID != "grok-4.6" || len(createBody.Model.Params) != 2 {
		t.Fatalf("unexpected create body: %+v", createBody)
	}
	if createBody.Model.Params[0] != (modelParam{ID: "effort", Value: "xhigh"}) || createBody.Model.Params[1] != (modelParam{ID: "fast", Value: "true"}) {
		t.Fatalf("unexpected Grok params: %+v", createBody.Model.Params)
	}
}

func TestChatCompletionFeatureUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "feature_unavailable",
				"message": "Storage mode is disabled.",
			},
		})
	}))
	defer server.Close()

	p := New("test-key", server.URL)
	_, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
		Model: "composer-2.5-fast",
		Messages: []model.ChatMessage{
			{Role: "user", Content: "hi"},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
