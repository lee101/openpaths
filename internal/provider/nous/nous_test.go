package nous

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

func TestNousCompatibleEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected request: %s", r.URL.Path)
		}
		var req model.ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.Model != "stealth/missingno" || req.ReasoningEffort != "high" || req.TaskTier != "" || req.RoutingStrategy != "" || req.PromptCacheKey != "" {
			t.Errorf("unexpected request body: %+v", req)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"model":"stealth/missingno","choices":[],"usage":{"prompt_tokens":2,"completion_tokens":1,"total_tokens":3}}`))
	}))
	defer server.Close()
	for _, url := range []string{server.URL, server.URL + "/v1/"} {
		p := New("test-key", url)
		if p.Name() != "nous" {
			t.Fatal(p.Name())
		}
		_, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{Model: "stealth/missingno", ReasoningEffort: "high", TaskTier: "hard", RoutingStrategy: "config", PromptCacheKey: "private"})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestNousProviderErrorIdentity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()
	_, err := New("test-key", server.URL).ChatCompletion(context.Background(), &model.ChatCompletionRequest{Model: "stealth/missingno"})
	pe, ok := err.(*provider.ProviderError)
	if !ok || pe.Provider != "nous" || !pe.Retryable {
		t.Fatalf("unexpected provider error: %v", err)
	}
}
