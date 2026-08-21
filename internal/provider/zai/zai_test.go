package zai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

func TestGenerateImage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/paas/v4/images/generations" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("bad auth header: %s", r.Header.Get("Authorization"))
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "glm-image" {
			t.Errorf("expected model glm-image, got %v", body["model"])
		}
		if body["size"] != "1280x1280" {
			t.Errorf("expected size 1280x1280, got %v", body["size"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"created": 1700000000,
			"data": []map[string]any{
				{"url": "https://example.com/image.png"},
			},
		})
	}))
	defer ts.Close()

	p := New("test-key", ts.URL)
	resp, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model:  "glm-image",
		Prompt: "a cute kitten",
		Size:   "1280x1280",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 image, got %d", len(resp.Data))
	}
	if resp.Data[0].URL != "https://example.com/image.png" {
		t.Errorf("unexpected URL: %s", resp.Data[0].URL)
	}
}

func TestChatCompletion_StandardEndpoint(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/paas/v4/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"role": "assistant", "content": "ok"}}},
		})
	}))
	defer ts.Close()

	p := New("test-key", ts.URL)
	resp, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{Model: "glm-5.1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
}

func TestChatCompletion_CodingEndpointPreferred(t *testing.T) {
	var hitCoding bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/coding/paas/v4/chat/completions" {
			hitCoding = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{"message": map[string]any{"role": "assistant", "content": "coding"}}},
			})
			return
		}
		t.Errorf("coding key should not fall through; hit %s", r.URL.Path)
		w.WriteHeader(500)
	}))
	defer ts.Close()

	p := NewCoding("coding-key", ts.URL)
	resp, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{Model: "glm-5.2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hitCoding {
		t.Fatal("expected coding endpoint to be used")
	}
	if resp.Choices[0].Message.Content != "coding" {
		t.Errorf("unexpected content: %v", resp.Choices[0].Message.Content)
	}
}

func TestChatCompletion_CodingFallsBackToStandard(t *testing.T) {
	var paths []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		// Coding surface rejects the key (it's a plain API key) -> 401, then the
		// provider must retry the standard surface.
		if r.URL.Path == "/api/coding/paas/v4/chat/completions" {
			w.WriteHeader(401)
			w.Write([]byte(`{"error":"not a coding plan key"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"role": "assistant", "content": "std"}}},
		})
	}))
	defer ts.Close()

	p := NewCoding("plain-key", ts.URL)
	resp, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{Model: "glm-5.2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(paths) != 2 || paths[0] != "/api/coding/paas/v4/chat/completions" || paths[1] != "/api/paas/v4/chat/completions" {
		t.Fatalf("expected coding then standard, got %v", paths)
	}
	if resp.Choices[0].Message.Content != "std" {
		t.Errorf("unexpected content: %v", resp.Choices[0].Message.Content)
	}
}

func TestChatCompletion_RealErrorNotRetried(t *testing.T) {
	var calls int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		// 429 is a genuine rate-limit, not a wrong-surface signal: must not
		// fall through to the standard endpoint.
		w.WriteHeader(429)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer ts.Close()

	p := NewCoding("coding-key", ts.URL)
	_, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{Model: "glm-5.2"})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 upstream call for 429, got %d", calls)
	}
	pe, ok := err.(*provider.ProviderError)
	if !ok || pe.StatusCode != 429 {
		t.Fatalf("expected 429 ProviderError, got %v", err)
	}
	if !pe.Retryable {
		t.Error("429 should be retryable")
	}
}

func TestGenerateImage_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer ts.Close()

	p := New("test-key", ts.URL)
	_, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model:  "glm-image",
		Prompt: "test",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerateImage_EmptyData(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"created": 1700000000,
			"data":    []map[string]any{},
		})
	}))
	defer ts.Close()

	p := New("test-key", ts.URL)
	_, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model:  "glm-image",
		Prompt: "test",
	})
	if err == nil {
		t.Fatal("expected error for empty data")
	}
}

func TestSanitizeForZAIPromotesSystemOnly(t *testing.T) {
	req := &model.ChatCompletionRequest{
		Prefill:  "H",
		Messages: []model.ChatMessage{{Role: "system", Content: "say only hi"}},
	}
	sanitizeForZAI(req)
	if req.Prefill != "" {
		t.Errorf("prefill not cleared: %q", req.Prefill)
	}
	if req.Messages[0].Role != "user" {
		t.Errorf("role = %q, want user", req.Messages[0].Role)
	}
	if req.Messages[0].Content != "say only hi" {
		t.Errorf("content mutated: %v", req.Messages[0].Content)
	}
}
