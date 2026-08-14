package openrouter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestGenerateImageUsesOpenRouterUnifiedImageEndpoint(t *testing.T) {
	var got map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Fatalf("authorization = %q", auth)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(model.ImageGenerationResponse{
			Data: []model.ImageData{{B64JSON: "aW1hZ2U="}},
		})
	}))
	defer srv.Close()

	p := New("test-key", srv.URL)
	resp, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model: "google/gemini-3.1-flash-image", Prompt: "fallback art", N: 1, Size: "1024x1024",
	})
	if err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if got["model"] != "google/gemini-3.1-flash-image" || len(resp.Data) != 1 {
		t.Fatalf("request = %#v, response = %#v", got, resp)
	}
}
