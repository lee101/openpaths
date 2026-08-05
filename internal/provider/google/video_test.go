package google

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestGenerateVideoPostsGeminiInteractionDefaults(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1beta/interactions" {
			t.Fatalf("path = %q, want /v1beta/interactions", r.URL.Path)
		}
		if r.URL.Query().Get("key") != "test-key" {
			t.Fatalf("query key = %q", r.URL.Query().Get("key"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"output":{"video":{"url":"https://example.com/out.mp4"}}}`))
	}))
	defer srv.Close()

	p := New("test-key", srv.URL)
	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:    "models/gemini-omni-flash-preview",
		Prompt:   "A cinematic city timelapse",
		Duration: model.VideoDuration("10"),
	})
	if err != nil {
		t.Fatalf("GenerateVideo error: %v", err)
	}
	if resp.VideoURL != "https://example.com/out.mp4" {
		t.Fatalf("VideoURL = %q", resp.VideoURL)
	}
	if got["model"] != "models/gemini-omni-flash-preview" {
		t.Fatalf("model = %#v", got["model"])
	}
	if got["input"] != "A cinematic city timelapse" {
		t.Fatalf("input = %#v", got["input"])
	}
	cfg := got["generation_config"].(map[string]any)
	if cfg["thinking_level"] != "high" || int(cfg["max_output_tokens"].(float64)) != geminiOmniDefaultMaxOutputTokens {
		t.Fatalf("generation_config = %#v", cfg)
	}
	videoCfg := cfg["video_config"].(map[string]any)
	if videoCfg["task"] != "unspecified" {
		t.Fatalf("video_config = %#v", videoCfg)
	}
	format := got["response_format"].(map[string]any)
	if format["type"] != "video" || format["duration"] != "10s" || format["delivery"] != "uri" {
		t.Fatalf("response_format = %#v", format)
	}
	if got["store"] != true || got["stream"] != false {
		t.Fatalf("store/stream = %#v/%#v", got["store"], got["stream"])
	}
}

func TestGenerateVideoConvertsImageURLToInteractionInput(t *testing.T) {
	imageBytes := []byte("fakepng")
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/image.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(imageBytes)
		case "/v1beta/interactions":
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			_, _ = w.Write([]byte(`{"output":{"video":{"url":"https://example.com/out.mp4"}}}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	p := New("test-key", srv.URL)
	serverURL, _ := url.Parse(srv.URL)
	p.imageClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		clone := req.Clone(req.Context())
		clone.URL.Scheme = serverURL.Scheme
		clone.URL.Host = serverURL.Host
		return srv.Client().Transport.RoundTrip(clone)
	})}
	_, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:    "models/gemini-omni-flash-preview",
		Prompt:   "Animate this product shot",
		ImageURL: "https://images.example.com/image.png",
	})
	if err != nil {
		t.Fatalf("GenerateVideo error: %v", err)
	}
	parts := got["input"].([]any)
	if len(parts) != 2 {
		t.Fatalf("input parts = %#v", parts)
	}
	image := parts[0].(map[string]any)
	if image["type"] != "image" || image["mime_type"] != "image/png" || image["data"] != base64.StdEncoding.EncodeToString(imageBytes) {
		t.Fatalf("image part = %#v", image)
	}
	text := parts[1].(map[string]any)
	if text["text"] != "Animate this product shot" {
		t.Fatalf("text part = %#v", text)
	}
	cfg := got["generation_config"].(map[string]any)
	videoCfg := cfg["video_config"].(map[string]any)
	if videoCfg["task"] != "image_to_video" {
		t.Fatalf("video_config = %#v", videoCfg)
	}
}

func TestGenerateVideoPreservesGeminiInteractionArgs(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_, _ = w.Write([]byte(`{"steps":[{"output_video":{"mime_type":"video/mp4","data":"ZmFrZQ=="}}]}`))
	}))
	defer srv.Close()

	p := New("test-key", srv.URL)
	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:                 "models/gemini-omni-flash-preview",
		Input:                 json.RawMessage(`{"text":"animate this frame"}`),
		GenerationConfig:      json.RawMessage(`{"thinking_level":"medium","video_config":{"task":"animate"}}`),
		ResponseModalities:    []string{"video"},
		ResponseFormat:        json.RawMessage(`{"type":"video","duration":"8s"}`),
		PreviousInteractionID: "int-123",
		Store:                 boolPtr(false),
		Stream:                boolPtr(true),
	})
	if err != nil {
		t.Fatalf("GenerateVideo error: %v", err)
	}
	if resp.VideoURL != "data:video/mp4;base64,ZmFrZQ==" {
		t.Fatalf("VideoURL = %q", resp.VideoURL)
	}
	if got["previous_interaction_id"] != "int-123" {
		t.Fatalf("previous_interaction_id = %#v", got["previous_interaction_id"])
	}
	input := got["input"].(map[string]any)
	if input["text"] != "animate this frame" {
		t.Fatalf("input = %#v", input)
	}
	cfg := got["generation_config"].(map[string]any)
	if cfg["thinking_level"] != "medium" {
		t.Fatalf("generation_config = %#v", cfg)
	}
	format := got["response_format"].(map[string]any)
	if format["duration"] != "8s" {
		t.Fatalf("response_format = %#v", format)
	}
	if got["store"] != false || got["stream"] != true {
		t.Fatalf("store/stream = %#v/%#v", got["store"], got["stream"])
	}
}

func boolPtr(v bool) *bool {
	return &v
}
