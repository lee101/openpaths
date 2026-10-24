package fal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestGenerateVideo_Kling4RequestShape(t *testing.T) {
	audio := true
	seed := 7
	cases := []struct {
		name    string
		req     model.VideoGenerationRequest
		want    map[string]any
		absent  []string
		wantURL string
	}{
		{
			name:   "text",
			req:    model.VideoGenerationRequest{Model: "kling/v4/text-to-video", Prompt: "p", Resolution: "720p", Duration: "8", AspectRatio: "9:16", GenerateAudio: &audio, Seed: &seed},
			want:   map[string]any{"prompt": "p", "resolution": "720p", "duration": float64(8), "aspect_ratio": "9:16"},
			absent: []string{"generate_audio", "seed"},
		},
		{
			name:   "image",
			req:    model.VideoGenerationRequest{Model: "kling/v4/image-to-video", Prompt: "p", ImageURL: "https://x/a.png", EndImageURL: "https://x/b.png", Duration: "auto", AspectRatio: "16:9"},
			want:   map[string]any{"start_image_url": "https://x/a.png", "end_image_url": "https://x/b.png"},
			absent: []string{"image_url", "duration", "aspect_ratio"},
		},
		{
			name:   "reference",
			req:    model.VideoGenerationRequest{Model: "kling/v4/reference-to-video", Prompt: "@Video1", ImageURLs: []string{"https://x/a.png"}, VideoURLs: []string{"https://x/v.mp4"}, AspectRatio: "auto"},
			want:   map[string]any{"aspect_ratio": "auto"},
			absent: []string{"image_url"},
		},
		{
			name:   "edit",
			req:    model.VideoGenerationRequest{Model: "kling/v4/video-to-video/edit", Prompt: "@Video1 sunset", VideoURL: "https://x/v.mp4", ImageURLs: []string{"https://x/a.png"}, AspectRatio: "16:9"},
			want:   map[string]any{"video_url": "https://x/v.mp4"},
			absent: []string{"video_urls", "aspect_ratio"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got map[string]any
			path := "/" + tc.req.Model
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case path:
					_ = json.NewDecoder(r.Body).Decode(&got)
					_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req_k4"})
				case path + "/requests/req_k4/status":
					w.WriteHeader(http.StatusMethodNotAllowed)
				case "/kling/v4/requests/req_k4/status":
					_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
				case "/kling/v4/requests/req_k4":
					_ = json.NewEncoder(w).Encode(map[string]any{"video": map[string]any{"url": "https://example.com/k4.mp4"}})
				default:
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
			}))
			defer server.Close()
			p := New("test-key")
			p.baseURL = server.URL
			p.client = server.Client()
			req := tc.req
			resp, err := p.GenerateVideo(context.Background(), &req)
			if err != nil {
				t.Fatalf("GenerateVideo: %v", err)
			}
			if resp.VideoURL != "https://example.com/k4.mp4" {
				t.Fatalf("resp = %+v", resp)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("%s = %v, want %v", k, got[k], v)
				}
			}
			for _, k := range tc.absent {
				if _, ok := got[k]; ok {
					t.Errorf("%s should be absent: %v", k, got)
				}
			}
			if tc.name == "reference" {
				if imgs, _ := got["image_urls"].([]any); len(imgs) != 1 {
					t.Errorf("image_urls = %v", got["image_urls"])
				}
				if vids, _ := got["video_urls"].([]any); len(vids) != 1 {
					t.Errorf("video_urls = %v", got["video_urls"])
				}
			}
		})
	}
}
