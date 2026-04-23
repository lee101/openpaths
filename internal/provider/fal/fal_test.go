package fal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestGenerateImage_PassesGPTImage2OptionsAndDecodesDataURI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openai/gpt-image-2" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Key test-key" {
			t.Fatalf("unexpected auth header: %s", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["prompt"] != "poster" {
			t.Fatalf("prompt = %v", body["prompt"])
		}
		if body["quality"] != "high" {
			t.Fatalf("quality = %v", body["quality"])
		}
		if body["num_images"] != float64(1) {
			t.Fatalf("num_images = %v", body["num_images"])
		}
		if body["output_format"] != "png" {
			t.Fatalf("output_format = %v", body["output_format"])
		}
		if body["sync_mode"] != true {
			t.Fatalf("sync_mode = %v", body["sync_mode"])
		}
		imageSize, ok := body["image_size"].(map[string]any)
		if !ok {
			t.Fatalf("image_size = %#v", body["image_size"])
		}
		if imageSize["width"] != float64(1024) || imageSize["height"] != float64(1024) {
			t.Fatalf("image_size = %#v", imageSize)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"url": "data:image/png;base64,ZmFrZQ=="},
			},
		})
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()

	resp, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model:   "openai/gpt-image-2",
		Prompt:  "poster",
		Size:    "1024x1024",
		N:       1,
		Quality: "high",
	})
	if err != nil {
		t.Fatalf("GenerateImage() error = %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("len(resp.Data) = %d, want 1", len(resp.Data))
	}
	if resp.Data[0].B64JSON != "ZmFrZQ==" {
		t.Fatalf("B64JSON = %q, want %q", resp.Data[0].B64JSON, "ZmFrZQ==")
	}
	if resp.Data[0].URL != "" {
		t.Fatalf("URL = %q, want empty", resp.Data[0].URL)
	}
}

func TestGenerateImage_LeavesURLResponseForURLMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"url": "https://example.com/image.webp"},
			},
		})
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()

	resp, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model:          "fal-ai/flux/dev",
		Prompt:         "cat",
		ResponseFormat: "url",
	})
	if err != nil {
		t.Fatalf("GenerateImage() error = %v", err)
	}
	if got := resp.Data[0].URL; got != "https://example.com/image.webp" {
		t.Fatalf("URL = %q", got)
	}
}
