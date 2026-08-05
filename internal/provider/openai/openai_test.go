package openai

import (
	"context"
	"encoding/json"
	"io"
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

func TestGenerateImageResponseFormatByModel(t *testing.T) {
	tests := []struct {
		model         string
		wantRespField bool // whether response_format should be sent upstream
	}{
		{"dall-e-3", false}, // newer images backend rejects response_format
		{"gpt-image-2", false},
		{"dall-e-2", true}, // legacy model still honours it
	}
	for _, tt := range tests {
		var got struct {
			ResponseFormat string `json:"response_format"`
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &got)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(model.ImageGenerationResponse{})
		}))
		p := New("test-key", server.URL)
		_, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
			Model:          tt.model,
			Prompt:         "a cat",
			ResponseFormat: "url",
		})
		server.Close()
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tt.model, err)
		}
		if (got.ResponseFormat != "") != tt.wantRespField {
			t.Errorf("%s: response_format sent=%q, want present=%v", tt.model, got.ResponseFormat, tt.wantRespField)
		}
	}
}

func TestSanitizeForOpenAI_StripsPrefillAndTaskTier(t *testing.T) {
	req := &model.ChatCompletionRequest{
		Model:    "gpt-5-mini",
		Messages: []model.ChatMessage{{Role: "user", Content: "Hi"}},
		Prefill:  "{",
		TaskTier: "hard",
		Thinking: &model.ThinkingConfig{Type: "enabled"},
	}

	sanitizeForOpenAI(req)

	if req.Prefill != "" {
		t.Errorf("Prefill not stripped: %q", req.Prefill)
	}
	if req.TaskTier != "" {
		t.Errorf("TaskTier not stripped: %q", req.TaskTier)
	}
	if req.Thinking != nil {
		t.Errorf("Thinking not stripped: %#v", req.Thinking)
	}

	// And ensure the marshaled body never contains those keys.
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if bytes := string(body); containsAny(bytes, "prefill", "task_tier", "thinking") {
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

func TestIsReasoningModel(t *testing.T) {
	cases := []struct {
		model string
		want  bool
	}{
		{"gpt-5.5", true},
		{"gpt-5.4-mini", true},
		{"gpt-5.4-nano", true},
		{"gpt-5-codex", true},
		{"gpt-5-mini", true},
		{"o1", true},
		{"o3", true},
		{"o4-mini", true},
		{"gpt-5-chat-latest", false},
		{"gpt-4o", false},
		{"gemini-3.5-flash", false},
		{"deepseek-v4-flash", false},
	}
	for _, c := range cases {
		if got := isReasoningModel(c.model); got != c.want {
			t.Errorf("isReasoningModel(%q) = %v, want %v", c.model, got, c.want)
		}
	}
}

func TestNormalizeSamplingParams(t *testing.T) {
	temp, topP, pen := 0.7, 0.9, 0.5

	// Reasoning model: sampling params are stripped.
	req := &model.ChatCompletionRequest{
		Model:            "gpt-5.5",
		Temperature:      &temp,
		TopP:             &topP,
		PresencePenalty:  &pen,
		FrequencyPenalty: &pen,
	}
	normalizeSamplingParams(req)
	if req.Temperature != nil || req.TopP != nil || req.PresencePenalty != nil || req.FrequencyPenalty != nil {
		t.Errorf("reasoning model sampling params not stripped: %#v", req)
	}

	// Non-reasoning model: sampling params are preserved.
	temp2 := 0.7
	keep := &model.ChatCompletionRequest{Model: "gpt-4o", Temperature: &temp2}
	normalizeSamplingParams(keep)
	if keep.Temperature == nil || *keep.Temperature != 0.7 {
		t.Errorf("non-reasoning temperature should be preserved, got %v", keep.Temperature)
	}
}

func TestChatCompletionStripsTemperatureForReasoningModel(t *testing.T) {
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req model.ChatCompletionRequest
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		_ = json.Unmarshal(body, &req)
		if req.Temperature != nil {
			t.Errorf("temperature reached the provider for reasoning model: %v", *req.Temperature)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(model.ChatCompletionResponse{ID: "ok"})
	}))
	defer server.Close()

	temp := 0.7
	p := New("test-key", server.URL)
	_, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
		Model:       "gpt-5.5",
		Messages:    []model.ChatMessage{{Role: "user", Content: "Hi"}},
		Temperature: &temp,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if containsAny(gotBody, "temperature") {
		t.Errorf("marshaled body still references temperature: %s", gotBody)
	}
}

func TestChatCompletionPassesThroughGPT56TierModel(t *testing.T) {
	for _, modelID := range []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"} {
		t.Run(modelID, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req model.ChatCompletionRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("decode provider request: %v", err)
				}
				if req.Model != modelID {
					t.Fatalf("provider model = %q, want %q", req.Model, modelID)
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(model.ChatCompletionResponse{ID: "ok", Model: modelID})
			}))
			defer server.Close()

			p := New("test-key", server.URL)
			if _, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
				Model:    modelID,
				Messages: []model.ChatMessage{{Role: "user", Content: "Hi"}},
			}); err != nil {
				t.Fatalf("chat completion: %v", err)
			}
		})
	}
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
			PromptTokens:          10,
			CompletionTokens:      5,
			TotalTokens:           15,
			PromptCacheHitTokens:  7,
			PromptCacheMissTokens: 3,
			PromptTokensDetails:   &model.PromptTokensDetails{CachedTokens: 7},
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
	if resp.Usage.PromptCacheHitTokens != 7 {
		t.Errorf("got cache hit tokens %d, want 7", resp.Usage.PromptCacheHitTokens)
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

func TestIsRetryableStatus_ResponsesOnlyModel(t *testing.T) {
	cases := []struct {
		name string
		code int
		body string
		want bool
	}{
		{"500 transient", 500, "boom", true},
		{"429 rate limit", 429, "slow down", true},
		{"400 responses-only codex", 400, `{"error":{"message":"This model is only supported in v1/responses and not in v1/chat/completions."}}`, true},
		{"400 ordinary bad request", 400, `{"error":{"message":"invalid 'messages'"}}`, false},
		{"402 upstream insufficient balance", 402, `{"error":{"message":"Insufficient Balance","code":"invalid_request_error"}}`, true},
		{"200 ok", 200, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRetryableStatus(tc.code, tc.body); got != tc.want {
				t.Errorf("isRetryableStatus(%d, %q) = %v, want %v", tc.code, tc.body, got, tc.want)
			}
		})
	}
}
