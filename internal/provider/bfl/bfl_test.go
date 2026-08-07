package bfl

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func TestGenerateVideoDraft(t *testing.T) {
	var submitted map[string]any
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	mux.HandleFunc("/v1/flux-3-video", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("x-key"); got != "test-key" {
			t.Fatalf("x-key = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&submitted); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "job-1", "polling_url": server.URL + "/result"})
	})
	mux.HandleFunc("/result", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "job-1", "status": "Ready", "result": map[string]any{"sample": "https://cdn.test/video.mp4"}})
	})
	p := New("test-key", server.URL)
	p.pollInterval = time.Millisecond
	audio := true
	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{Model: "flux-3-video-draft", Prompt: "a forest", Duration: "5", Resolution: "FHD", AspectRatio: "16:9", GenerateAudio: &audio})
	if err != nil {
		t.Fatal(err)
	}
	if resp.VideoURL != "https://cdn.test/video.mp4" {
		t.Fatalf("video URL = %q", resp.VideoURL)
	}
	if submitted["mode"] != "t2v" || submitted["draft"] != true || submitted["resolution"] != "hd" {
		t.Fatalf("submission = %#v", submitted)
	}
	if submitted["safety_tolerance"] != float64(4) {
		t.Fatalf("safety_tolerance = %#v", submitted["safety_tolerance"])
	}
}

func TestVideoPayloadModes(t *testing.T) {
	i2v, err := videoPayload(&model.VideoGenerationRequest{Model: "flux-3-video", Prompt: "move", ImageURL: "https://x.test/a.jpg", EndImageURL: "https://x.test/b.jpg"})
	if err != nil || i2v["mode"] != "i2v" || len(i2v["keyframes"].([]string)) != 2 {
		t.Fatalf("i2v = %#v, err=%v", i2v, err)
	}
	v2v, err := videoPayload(&model.VideoGenerationRequest{Model: "flux-3-video", Prompt: "continue", VideoURL: "https://x.test/a.mp4", Resolution: "FHD"})
	if err != nil || v2v["mode"] != "v2v" || v2v["resolution"] != "fhd" {
		t.Fatalf("v2v = %#v, err=%v", v2v, err)
	}
}

func TestGenerateImageUsesFixedSafetyAndPromptUpsampling(t *testing.T) {
	var submitted map[string]any
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	mux.HandleFunc("/v1/flux-2-pro-preview", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&submitted); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "image-1", "polling_url": server.URL + "/image-result"})
	})
	mux.HandleFunc("/image-result", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "Ready", "result": map[string]any{"sample": "https://cdn.test/image.webp"}})
	})
	p := New("test-key", server.URL)
	p.pollInterval = time.Millisecond
	disablePUP := true
	seed := 42
	resp, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model: "flux-2-pro-preview", Prompt: "a glass forest", Size: "1920x1088", OutputFormat: "webp",
		DisablePUP: &disablePUP, SafetyTolerance: "0", Seed: &seed,
		ReferenceImageURLs: []string{"https://cdn.test/reference.webp"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 || resp.Data[0].URL != "https://cdn.test/image.webp" || resp.Data[0].Width != 1920 || resp.Data[0].Height != 1088 {
		t.Fatalf("response = %#v", resp)
	}
	if submitted["safety_tolerance"] != float64(5) || submitted["disable_pup"] != true {
		t.Fatalf("fixed controls = %#v", submitted)
	}
	if submitted["input_image"] != "https://cdn.test/reference.webp" || submitted["seed"] != float64(42) {
		t.Fatalf("image payload = %#v", submitted)
	}
}
