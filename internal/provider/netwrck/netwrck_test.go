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

func TestGenerateImageSendsReferenceForEdits(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ra2-image-editor" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]string{"image_url": "https://cdn.example/edited.webp"})
	}))
	defer srv.Close()
	p := New("key", srv.URL)
	resp, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "ra2-image-editor", Prompt: "make it night", ImageURL: "https://cdn.example/src.png"})
	if err != nil {
		t.Fatal(err)
	}
	if got["image_url"] != "https://cdn.example/src.png" || resp.Data[0].URL != "https://cdn.example/edited.webp" {
		t.Fatalf("request %v response %v", got, resp.Data)
	}
	resp, err = p.GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "ra2-image-editor", Prompt: "x", Images: []model.ImageInput{{URL: "data:image/png;base64,AAAA"}}})
	if err != nil || got["image_base64"] != "AAAA" || resp == nil {
		t.Fatalf("data url request %v err %v", got, err)
	}
}
