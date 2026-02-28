package zai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpath/openpath/internal/model"
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
