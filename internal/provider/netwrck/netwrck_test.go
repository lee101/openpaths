package netwrck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestGenerateImageRoutesByModelID(t *testing.T) {
	for _, modelID := range []string{"ra1", "ra2"} {
		t.Run(modelID, func(t *testing.T) {
			var gotPath string
			var gotBody map[string]any
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Errorf("decode request: %v", err)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"image_url": "https://example.com/art.webp"}`))
			}))
			defer ts.Close()

			p := New("test-key", ts.URL)
			resp, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
				Model:  modelID,
				Prompt: "a lighthouse keeper",
				Size:   "1024x1024",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if want := "/api/" + modelID; gotPath != want {
				t.Errorf("path = %q, want %q", gotPath, want)
			}
			if gotBody["api_key"] != "test-key" || gotBody["prompt"] != "a lighthouse keeper" || gotBody["size"] != "1024x1024" {
				t.Errorf("body = %v", gotBody)
			}
			if len(resp.Data) != 1 || resp.Data[0].URL != "https://example.com/art.webp" {
				t.Errorf("response = %+v", resp)
			}
		})
	}
}

func TestGenerateImageDefaultsSizeTo1024(t *testing.T) {
	var gotBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"image_url": "https://example.com/art.webp"}`))
	}))
	defer ts.Close()

	p := New("test-key", ts.URL)
	if _, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "ra2", Prompt: "x"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBody["size"] != "1024x1024" {
		t.Errorf("size = %v, want 1024x1024", gotBody["size"])
	}
}
