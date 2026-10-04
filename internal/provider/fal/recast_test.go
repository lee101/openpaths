package fal

import (
	"context"
	"encoding/json"
	"github.com/openpaths/openpaths/internal/model"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestH3MaxRecastRouteAndPayload(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/minimax/h3-max/recast":
			json.NewDecoder(r.Body).Decode(&got)
			w.Write([]byte(`{"request_id":"recast1"}`))
		case "/minimax/h3-max/recast/requests/recast1/status":
			w.WriteHeader(404)
		case "/minimax/h3-max/requests/recast1/status":
			w.Write([]byte(`{"status":"COMPLETED"}`))
		case "/minimax/h3-max/requests/recast1":
			w.Write([]byte(`{"video":{"url":"https://example.com/recast.mp4"},"seed":42}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()
	seed := 42
	audio := false
	req := &model.VideoGenerationRequest{Model: "minimax/h3-max/recast", VideoURL: "https://example.com/source.mp4", ReferenceImageURLs: []string{"https://example.com/person1.png", "https://example.com/person2.png"}, Duration: "1", AspectRatio: "16:9", GenerateAudio: &audio, Seed: &seed}
	resp, err := p.GenerateVideo(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.VideoURL != "https://example.com/recast.mp4" || resp.Seed == nil || *resp.Seed != 42 {
		t.Fatalf("result: %#v", resp)
	}
	if got["resolution"] != "1080P" || got["video_url"] != req.VideoURL || len(got["reference_image_urls"].([]any)) != 2 {
		t.Fatalf("input: %#v", got)
	}
	for _, key := range []string{"prompt", "duration", "aspect_ratio", "generate_audio", "image_url", "image_urls", "video_urls", "prompt_expansion_mode"} {
		if _, exists := got[key]; exists {
			t.Errorf("unexpected %s in %#v", key, got)
		}
	}
}

func TestRecastValidatesPhotosAndResolution(t *testing.T) {
	for _, req := range []model.VideoGenerationRequest{
		{VideoURL: "https://x/video.mp4"},
		{ReferenceImageURLs: []string{"https://x/image.png"}},
		{VideoURL: "https://x/video.mp4", ReferenceImageURLs: []string{"a", "b", "c", "d", "e"}},
		{VideoURL: "https://x/video.mp4", ReferenceImageURLs: []string{"a"}, Resolution: "480P"},
	} {
		if _, err := h3MaxRecastInput(&req); err == nil {
			t.Errorf("accepted invalid request: %#v", req)
		}
	}
}
