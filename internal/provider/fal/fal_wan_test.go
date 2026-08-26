package fal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

// TestGenerateVideo_Wan30RequestShape pins the Wan 3.0 payload mapping: the
// endpoint takes an integer duration (null for smart duration), an `audio`
// toggle instead of generate_audio, "adaptive" as the auto aspect-ratio
// sentinel, and an optional enable_thinking flag.
func TestGenerateVideo_Wan30RequestShape(t *testing.T) {
	var gotSubmit map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/alibaba/wan-3.0/text-to-video":
			if err := json.NewDecoder(r.Body).Decode(&gotSubmit); err != nil {
				t.Fatalf("decode submit: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req_wan"})
		case "/alibaba/wan-3.0/text-to-video/requests/req_wan/status":
			w.WriteHeader(http.StatusMethodNotAllowed)
		case "/alibaba/wan-3.0/requests/req_wan/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/alibaba/wan-3.0/requests/req_wan":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"video":         map[string]any{"url": "https://example.com/wan30.mp4"},
				"seed":          float64(42),
				"duration":      float64(5),
				"actual_prompt": "expanded prompt",
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()
	audio := true
	thinking := true
	seed := 42

	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:          "alibaba/wan-3.0/text-to-video",
		Prompt:         "A red panda walking through a bamboo forest at sunrise",
		Resolution:     "1080p",
		Duration:       "5",
		AspectRatio:    "auto",
		GenerateAudio:  &audio,
		EnableThinking: &thinking,
		Seed:           &seed,
	})
	if err != nil {
		t.Fatalf("GenerateVideo() error = %v", err)
	}
	if resp.VideoURL != "https://example.com/wan30.mp4" {
		t.Fatalf("VideoURL = %q", resp.VideoURL)
	}

	if gotSubmit["audio"] != true {
		t.Fatalf("audio = %#v, want true", gotSubmit["audio"])
	}
	if _, ok := gotSubmit["generate_audio"]; ok {
		t.Fatalf("generate_audio must not be sent to Wan 3.0: %#v", gotSubmit)
	}
	if gotSubmit["duration"] != float64(5) {
		t.Fatalf("duration = %#v, want numeric 5", gotSubmit["duration"])
	}
	if gotSubmit["aspect_ratio"] != "adaptive" {
		t.Fatalf("aspect_ratio = %#v, want adaptive", gotSubmit["aspect_ratio"])
	}
	if gotSubmit["enable_thinking"] != true {
		t.Fatalf("enable_thinking = %#v, want true", gotSubmit["enable_thinking"])
	}
	if gotSubmit["seed"] != float64(42) {
		t.Fatalf("seed = %#v, want 42", gotSubmit["seed"])
	}
}

func TestGenerateVideo_Wan30SmartDuration(t *testing.T) {
	var gotSubmit map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/alibaba/wan-3.0/text-to-video":
			if err := json.NewDecoder(r.Body).Decode(&gotSubmit); err != nil {
				t.Fatalf("decode submit: %v", err)
			}
			_, _ = w.Write([]byte(`{"video":{"url":"https://example.com/wan30.mp4"},"seed":1,"duration":8}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()

	if _, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:      "alibaba/wan-3.0/text-to-video",
		Prompt:     "a slow orbit around a glass router",
		Resolution: "720p",
		Duration:   "auto",
	}); err != nil {
		t.Fatalf("GenerateVideo() error = %v", err)
	}
	raw, err := json.Marshal(gotSubmit["duration"])
	if err != nil {
		t.Fatalf("marshal duration: %v", err)
	}
	if string(raw) != "null" {
		t.Fatalf("duration = %s, want explicit null for smart duration", raw)
	}
}

// TestGenerateVideo_Wan30I2VRequestShape pins the image-to-video frame naming:
// the generic OpenPaths image_url/end_image_url pair is forwarded to Wan 3.0
// as start_image_url/end_image_url.
func TestGenerateVideo_Wan30I2VRequestShape(t *testing.T) {
	var gotSubmit map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/alibaba/wan-3.0/image-to-video":
			if err := json.NewDecoder(r.Body).Decode(&gotSubmit); err != nil {
				t.Fatalf("decode submit: %v", err)
			}
			_, _ = w.Write([]byte(`{"video":{"url":"https://example.com/wan30-i2v.mp4"},"seed":7,"duration":5}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()

	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:       "alibaba/wan-3.0/image-to-video",
		Prompt:      "Slow cinematic push-in",
		ImageURL:    "https://example.com/first.png",
		EndImageURL: "https://example.com/last.png",
		Resolution:  "1080p",
		Duration:    "5",
	})
	if err != nil {
		t.Fatalf("GenerateVideo() error = %v", err)
	}
	if resp.VideoURL != "https://example.com/wan30-i2v.mp4" {
		t.Fatalf("VideoURL = %q", resp.VideoURL)
	}
	if gotSubmit["start_image_url"] != "https://example.com/first.png" {
		t.Fatalf("start_image_url = %#v", gotSubmit["start_image_url"])
	}
	if gotSubmit["end_image_url"] != "https://example.com/last.png" {
		t.Fatalf("end_image_url = %#v", gotSubmit["end_image_url"])
	}
	if _, ok := gotSubmit["image_url"]; ok {
		t.Fatalf("generic image_url must not reach Wan 3.0 i2v: %#v", gotSubmit)
	}
}
