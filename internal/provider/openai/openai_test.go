package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

func TestOpenAIProviderName(t *testing.T) {
	p := New("test-key", "")
	if p.Name() != "openai" {
		t.Errorf("got name %q, want %q", p.Name(), "openai")
	}
}

func TestSanitizeForOpenAI_StripsPrefillAndTaskTier(t *testing.T) {
	req := &model.ChatCompletionRequest{
		Model:    "gpt-5-mini",
		Messages: []model.ChatMessage{{Role: "user", Content: "Hi"}},
		Prefill:  "{",
		TaskTier: "hard",
	}

	sanitizeForOpenAI(req)

	if req.Prefill != "" {
		t.Errorf("Prefill not stripped: %q", req.Prefill)
	}
	if req.TaskTier != "" {
		t.Errorf("TaskTier not stripped: %q", req.TaskTier)
	}

	// And ensure the marshaled body never contains those keys.
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if bytes := string(body); containsAny(bytes, "prefill", "task_tier") {
		t.Errorf("marshaled body still references prefill/task_tier: %s", bytes)
	}
}

func containsAny(s string, needles ...string) bool {
	for _, n := range needles {
		for i := 0; i+len(n) <= len(s); i++ {
			if s[i:i+len(n)] == n {
				return true
			}
		}
	}
	return false
}

func TestChatCompletionSuccess(t *testing.T) {
	expectedResp := model.ChatCompletionResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "gpt-4o",
		Choices: []model.ChatChoice{{
			Index: 0,
			Message: &model.ChatMessage{
				Role:    "assistant",
				Content: "Hello!",
			},
		}},
		Usage: &model.UsageInfo{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResp)
	}))
	defer server.Close()

	p := New("test-key", server.URL)
	resp, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
		Model: "gpt-4o",
		Messages: []model.ChatMessage{{
			Role:    "user",
			Content: "Hi",
		}},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "chatcmpl-123" {
		t.Errorf("got ID %q, want %q", resp.ID, "chatcmpl-123")
	}
	if resp.Usage.TotalTokens != 15 {
		t.Errorf("got total tokens %d, want 15", resp.Usage.TotalTokens)
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
		Model:    "gpt-4o",
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
		t.Error("expected error to be retryable")
	}
}

func TestChatCompletionServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":{"message":"internal error"}}`))
	}))
	defer server.Close()

	p := New("test-key", server.URL)
	_, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
		Model:    "gpt-4o",
		Messages: []model.ChatMessage{{Role: "user", Content: "Hi"}},
	})

	if err == nil {
		t.Fatal("expected error")
	}

	pe, ok := err.(*provider.ProviderError)
	if !ok {
		t.Fatalf("expected ProviderError, got %T", err)
	}
	if pe.StatusCode != 500 {
		t.Errorf("got status %d, want 500", pe.StatusCode)
	}
	if !pe.Retryable {
		t.Error("expected 500 to be retryable")
	}
}

func TestChatCompletionStreamSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req model.ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.StreamOptions == nil || !req.StreamOptions.IncludeUsage {
			t.Fatalf("expected stream_options.include_usage to be true")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)

		chunks := []string{
			`{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"}}]}`,
			`{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":" world"}}]}`,
		}

		flusher, _ := w.(http.Flusher)
		for _, chunk := range chunks {
			w.Write([]byte("data: " + chunk + "\n\n"))
			flusher.Flush()
		}
		w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	p := New("test-key", server.URL)
	ch, err := p.ChatCompletionStream(context.Background(), &model.ChatCompletionRequest{
		Model:    "gpt-4o",
		Messages: []model.ChatMessage{{Role: "user", Content: "Hi"}},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var chunks []*model.ChatCompletionChunk
	var gotDone bool

	for event := range ch {
		if event.Err != nil {
			t.Fatalf("unexpected stream error: %v", event.Err)
		}
		if event.Done {
			gotDone = true
			continue
		}
		chunks = append(chunks, event.Chunk)
	}

	if !gotDone {
		t.Error("expected done event")
	}
	if len(chunks) != 2 {
		t.Fatalf("got %d chunks, want 2", len(chunks))
	}
}

func TestHealthCheckSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	p := New("test-key", server.URL)
	err := p.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHealthCheckFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer server.Close()

	p := New("test-key", server.URL)
	err := p.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
