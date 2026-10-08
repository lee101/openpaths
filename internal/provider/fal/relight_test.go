package fal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestH3MaxRelightRouteAndPayload(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/minimax/h3-max/relight":
			json.NewDecoder(r.Body).Decode(&got)
			w.Write([]byte(`{"request_id":"rl1"}`))
		case "/minimax/h3-max/relight/requests/rl1/status":
			w.WriteHeader(404)
		case "/minimax/h3-max/requests/rl1/status":
			w.Write([]byte(`{"status":"COMPLETED"}`))
		case "/minimax/h3-max/requests/rl1":
			w.Write([]byte(`{"video":{"url":"https://example.com/relit.mp4"},"seed":7}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()
	seed := 7
	audio := true
	req := &model.VideoGenerationRequest{Model: "minimax/h3-max/relight", Prompt: "ignored", VideoURL: "https://example.com/in.mp4", ReferenceImageURL: "https://example.com/sphere.png", Resolution: "480p", AspectRatio: "auto", Duration: "9", GenerateAudio: &audio, Seed: &seed}
	resp, err := p.GenerateVideo(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.VideoURL != "https://example.com/relit.mp4" || resp.Seed == nil || *resp.Seed != 7 {
		t.Fatalf("result: %#v", resp)
	}
	if got["video_url"] != req.VideoURL || got["reference_image_url"] != "https://example.com/sphere.png" || got["resolution"] != "480P" || got["aspect_ratio"] != "adaptive" || got["seed"] != float64(7) {
		t.Fatalf("input: %#v", got)
	}
	if req.Resolution != "480P" {
		t.Fatalf("resolution not normalised: %s", req.Resolution)
	}
	for _, key := range []string{"prompt", "duration", "generate_audio", "image_url", "image_urls", "reference_image_urls", "video_urls", "prompt_expansion_mode"} {
		if _, exists := got[key]; exists {
			t.Errorf("unexpected %s in %#v", key, got)
		}
	}
}

func TestH3MaxRelightDefaultsAndValidation(t *testing.T) {
	req := &model.VideoGenerationRequest{VideoURL: "https://x/v.mp4", ImageURL: "https://x/s.png"}
	input, err := h3MaxRelightInput(req)
	if err != nil || input["resolution"] != "768P" || input["aspect_ratio"] != "16:9" {
		t.Fatalf("defaults: %#v %v", input, err)
	}
	for _, bad := range []model.VideoGenerationRequest{
		{ImageURL: "https://x/s.png"},
		{VideoURL: "https://x/v.mp4"},
		{VideoURL: "https://x/v.mp4", ImageURL: "https://x/s.png", Resolution: "720P"},
		{VideoURL: "https://x/v.mp4", ImageURL: "https://x/s.png", AspectRatio: "5:4"},
	} {
		if _, err := h3MaxRelightInput(&bad); err == nil {
			t.Errorf("accepted invalid request: %#v", bad)
		}
	}
}
