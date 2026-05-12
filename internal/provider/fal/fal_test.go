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

func TestGenerateImage_HiDreamUsesQueueAndReferenceImages(t *testing.T) {
	var gotSubmit map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fal-ai/hidream-o1-image/dev":
			if got := r.Header.Get("Authorization"); got != "Key test-key" {
				t.Fatalf("unexpected auth header: %s", got)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotSubmit); err != nil {
				t.Fatalf("decode submit: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req_hidream"})
		case "/fal-ai/hidream-o1-image/dev/requests/req_hidream/status":
			w.WriteHeader(http.StatusNotFound)
		case "/fal-ai/hidream-o1-image/requests/req_hidream/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/fal-ai/hidream-o1-image/requests/req_hidream":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"images": []map[string]any{
					{"url": "https://example.com/hidream.png"},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()
	guidanceScale := 0.0

	resp, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model:              "fal-ai/hidream-o1-image/dev",
		Prompt:             "cinematic mug",
		Size:               "1024x1024",
		N:                  1,
		ReferenceImageURLs: []string{"https://example.com/ref.png"},
		NumInferenceSteps:  28,
		GuidanceScale:      &guidanceScale,
		OutputFormat:       "png",
	})
	if err != nil {
		t.Fatalf("GenerateImage() error = %v", err)
	}
	if gotSubmit["prompt"] != "cinematic mug" || gotSubmit["num_images"] != float64(1) {
		t.Fatalf("submit = %#v", gotSubmit)
	}
	if gotSubmit["num_inference_steps"] != float64(28) || gotSubmit["guidance_scale"] != float64(0) {
		t.Fatalf("submit = %#v", gotSubmit)
	}
	if gotSubmit["enable_safety_checker"] != false {
		t.Fatalf("enable_safety_checker = %#v", gotSubmit["enable_safety_checker"])
	}
	refs, ok := gotSubmit["reference_image_urls"].([]any)
	if !ok || len(refs) != 1 || refs[0] != "https://example.com/ref.png" {
		t.Fatalf("reference_image_urls = %#v", gotSubmit["reference_image_urls"])
	}
	imageSize, ok := gotSubmit["image_size"].(map[string]any)
	if !ok || imageSize["width"] != float64(1024) || imageSize["height"] != float64(1024) {
		t.Fatalf("image_size = %#v", gotSubmit["image_size"])
	}
	if resp.Data[0].URL != "https://example.com/hidream.png" {
		t.Fatalf("URL = %q", resp.Data[0].URL)
	}
}

func TestGenerateVideo_SubmitsSeedanceAndPollsResult(t *testing.T) {
	var gotSubmit map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bytedance/seedance-2.0/fast/text-to-video":
			if got := r.Header.Get("Authorization"); got != "Key test-key" {
				t.Fatalf("unexpected auth header: %s", got)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotSubmit); err != nil {
				t.Fatalf("decode submit: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req_123"})
		case "/bytedance/seedance-2.0/fast/text-to-video/requests/req_123/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/bytedance/seedance-2.0/fast/text-to-video/requests/req_123":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"video": map[string]any{"url": "https://example.com/out.mp4"},
				"seed":  float64(42),
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
	seed := 7

	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:         "bytedance/seedance-2.0/fast/text-to-video",
		Prompt:        "cinematic room",
		Resolution:    "720p",
		Duration:      "10",
		AspectRatio:   "16:9",
		GenerateAudio: &audio,
		Seed:          &seed,
		EndUserID:     "user-1",
	})
	if err != nil {
		t.Fatalf("GenerateVideo() error = %v", err)
	}
	if resp.VideoURL != "https://example.com/out.mp4" {
		t.Fatalf("VideoURL = %q", resp.VideoURL)
	}
	if gotSubmit["prompt"] != "cinematic room" || gotSubmit["resolution"] != "720p" || gotSubmit["duration"] != "10" {
		t.Fatalf("submit = %#v", gotSubmit)
	}
	if gotSubmit["generate_audio"] != true || gotSubmit["aspect_ratio"] != "16:9" || gotSubmit["end_user_id"] != "user-1" {
		t.Fatalf("submit = %#v", gotSubmit)
	}
}

func TestGenerateVideo_FallsBackToSeedanceBaseQueuePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bytedance/seedance-2.0/fast/text-to-video":
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req_123"})
		case "/bytedance/seedance-2.0/fast/text-to-video/requests/req_123/status":
			w.WriteHeader(http.StatusMethodNotAllowed)
		case "/bytedance/seedance-2.0/requests/req_123/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/bytedance/seedance-2.0/requests/req_123":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"video": map[string]any{"url": "https://example.com/out.mp4"},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()

	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:  "bytedance/seedance-2.0/fast/text-to-video",
		Prompt: "cinematic room",
	})
	if err != nil {
		t.Fatalf("GenerateVideo() error = %v", err)
	}
	if resp.VideoURL != "https://example.com/out.mp4" {
		t.Fatalf("VideoURL = %q", resp.VideoURL)
	}
}

func TestGenerateVideo_PassesReferenceInputs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bytedance/seedance-2.0/reference-to-video" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if got := len(body["image_urls"].([]any)); got != 2 {
			t.Fatalf("image_urls len = %d", got)
		}
		if got := len(body["video_urls"].([]any)); got != 1 {
			t.Fatalf("video_urls len = %d", got)
		}
		if got := len(body["audio_urls"].([]any)); got != 1 {
			t.Fatalf("audio_urls len = %d", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"video": map[string]any{"url": "https://example.com/ref.mp4"}})
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()

	_, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:     "bytedance/seedance-2.0/reference-to-video",
		Prompt:    "@Image1 and @Video1",
		ImageURLs: []string{"https://example.com/a.png", "https://example.com/b.png"},
		VideoURLs: []string{"https://example.com/in.mp4"},
		AudioURLs: []string{"https://example.com/in.mp3"},
	})
	if err != nil {
		t.Fatalf("GenerateVideo() error = %v", err)
	}
}

func TestGenerateVideo_PassesImageToVideoInputs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bytedance/seedance-2.0/image-to-video" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["image_url"] != "https://example.com/start.jpg" {
			t.Fatalf("image_url = %#v", body["image_url"])
		}
		if body["end_image_url"] != "https://example.com/end.jpg" {
			t.Fatalf("end_image_url = %#v", body["end_image_url"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"video": map[string]any{"url": "https://example.com/i2v.mp4"}})
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()

	_, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:       "bytedance/seedance-2.0/image-to-video",
		Prompt:      "slow tilt down",
		ImageURL:    "https://example.com/start.jpg",
		EndImageURL: "https://example.com/end.jpg",
		Duration:    "10",
		Resolution:  "720p",
	})
	if err != nil {
		t.Fatalf("GenerateVideo() error = %v", err)
	}
}
