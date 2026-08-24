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

func TestGenerateImage_HiDreamEditUsesQueueImageSizeAndSafetyDefault(t *testing.T) {
	var gotSubmit map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fal-ai/hidream-o1-image/edit":
			if err := json.NewDecoder(r.Body).Decode(&gotSubmit); err != nil {
				t.Fatalf("decode submit: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req_edit"})
		case "/fal-ai/hidream-o1-image/edit/requests/req_edit/status":
			w.WriteHeader(http.StatusNotFound)
		case "/fal-ai/hidream-o1-image/requests/req_edit/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/fal-ai/hidream-o1-image/requests/req_edit":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"images": []map[string]any{{"url": "https://example.com/edit.png"}},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()
	guidanceScale := 5.0

	resp, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model:              "fal-ai/hidream-o1-image/edit",
		Prompt:             "Replace the perfume bottle with a lipstick",
		ReferenceImageURLs: []string{"https://example.com/perfume.jpg"},
		ImageSize:          "landscape_16_9",
		NumInferenceSteps:  50,
		GuidanceScale:      &guidanceScale,
		N:                  1,
		OutputFormat:       "png",
	})
	if err != nil {
		t.Fatalf("GenerateImage() error = %v", err)
	}
	if gotSubmit["prompt"] != "Replace the perfume bottle with a lipstick" ||
		gotSubmit["image_size"] != "landscape_16_9" ||
		gotSubmit["num_inference_steps"] != float64(50) ||
		gotSubmit["guidance_scale"] != float64(5) ||
		gotSubmit["num_images"] != float64(1) ||
		gotSubmit["output_format"] != "png" ||
		gotSubmit["enable_safety_checker"] != false {
		t.Fatalf("submit = %#v", gotSubmit)
	}
	refs, ok := gotSubmit["reference_image_urls"].([]any)
	if !ok || len(refs) != 1 || refs[0] != "https://example.com/perfume.jpg" {
		t.Fatalf("reference_image_urls = %#v", gotSubmit["reference_image_urls"])
	}
	if resp.Data[0].URL != "https://example.com/edit.png" {
		t.Fatalf("URL = %q", resp.Data[0].URL)
	}
}

func TestGenerateImage_OutpaintUsesQueueAndImageURL(t *testing.T) {
	var gotSubmit map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fal-ai/flux-2-pro/outpaint":
			if err := json.NewDecoder(r.Body).Decode(&gotSubmit); err != nil {
				t.Fatalf("decode submit: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req_outpaint"})
		case "/fal-ai/flux-2-pro/outpaint/requests/req_outpaint/status":
			w.WriteHeader(http.StatusNotFound)
		case "/fal-ai/flux-2-pro/requests/req_outpaint/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/fal-ai/flux-2-pro/requests/req_outpaint":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"images": []map[string]any{{"url": "https://example.com/outpaint.jpg"}},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()
	left := 200
	right := 200
	bottom := 200
	safety := true

	resp, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model:               "fal-ai/flux-2-pro/outpaint",
		ImageURL:            "https://example.com/input.png",
		ExpandBottom:        &bottom,
		ExpandLeft:          &left,
		ExpandRight:         &right,
		EnableSafetyChecker: &safety,
		OutputFormat:        "jpeg",
	})
	if err != nil {
		t.Fatalf("GenerateImage() error = %v", err)
	}
	if gotSubmit["image_url"] != "https://example.com/input.png" ||
		gotSubmit["expand_bottom"] != float64(200) ||
		gotSubmit["expand_left"] != float64(200) ||
		gotSubmit["expand_right"] != float64(200) ||
		gotSubmit["enable_safety_checker"] != true ||
		gotSubmit["output_format"] != "jpeg" {
		t.Fatalf("submit = %#v", gotSubmit)
	}
	if _, ok := gotSubmit["prompt"]; ok {
		t.Fatalf("outpaint submit should not include prompt: %#v", gotSubmit)
	}
	if resp.Data[0].URL != "https://example.com/outpaint.jpg" {
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

func TestGenerateVideo_FallsBackToHappyHorseBaseQueuePath(t *testing.T) {
	var gotSubmit map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/alibaba/happy-horse/image-to-video":
			if err := json.NewDecoder(r.Body).Decode(&gotSubmit); err != nil {
				t.Fatalf("decode submit: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req_horse"})
		case "/alibaba/happy-horse/image-to-video/requests/req_horse/status":
			w.WriteHeader(http.StatusMethodNotAllowed)
		case "/alibaba/happy-horse/requests/req_horse/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/alibaba/happy-horse/requests/req_horse":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"video": map[string]any{"url": "https://example.com/happy-horse.mp4"},
				"seed":  float64(2072132269),
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()
	safety := true

	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:               "alibaba/happy-horse/image-to-video",
		Prompt:              "Bring the scene in the image to life.",
		ImageURL:            "https://example.com/rap.png",
		Resolution:          "1080p",
		Duration:            "5",
		EnableSafetyChecker: &safety,
	})
	if err != nil {
		t.Fatalf("GenerateVideo() error = %v", err)
	}
	if resp.VideoURL != "https://example.com/happy-horse.mp4" {
		t.Fatalf("VideoURL = %q", resp.VideoURL)
	}
	if gotSubmit["image_url"] != "https://example.com/rap.png" || gotSubmit["enable_safety_checker"] != true {
		t.Fatalf("submit = %#v", gotSubmit)
	}
	if gotSubmit["duration"] != float64(5) {
		t.Fatalf("duration = %#v, want numeric 5", gotSubmit["duration"])
	}
}

func TestGenerateVideo_FallsBackToLTX23BaseQueuePath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fal-ai/ltx-2.3/image-to-video":
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req_ltx"})
		case "/fal-ai/ltx-2.3/image-to-video/requests/req_ltx/status":
			w.WriteHeader(http.StatusMethodNotAllowed)
		case "/fal-ai/ltx-2.3/requests/req_ltx/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/fal-ai/ltx-2.3/requests/req_ltx":
			_ = json.NewEncoder(w).Encode(map[string]any{"video": map[string]any{"url": "https://example.com/ltx23.mp4"}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()

	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:      "fal-ai/ltx-2.3/image-to-video",
		Prompt:     "slow zoom",
		ImageURL:   "https://example.com/listing.jpg",
		Duration:   "6",
		Resolution: "1080p",
	})
	if err != nil {
		t.Fatalf("GenerateVideo() error = %v", err)
	}
	if resp.VideoURL != "https://example.com/ltx23.mp4" {
		t.Fatalf("VideoURL = %q", resp.VideoURL)
	}
}

func TestGenerateVideo_SyncLipsyncSingularInputsAndQueueFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fal-ai/sync-lipsync/v3/image-to-video":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["image_url"] != "https://example.com/avatar.jpg" {
				t.Fatalf("image_url = %#v", body["image_url"])
			}
			if body["audio_url"] != "https://example.com/voice.wav" {
				t.Fatalf("audio_url = %#v", body["audio_url"])
			}
			if body["sync_mode"] != "bounce" {
				t.Fatalf("sync_mode = %#v", body["sync_mode"])
			}
			if _, ok := body["audio_urls"]; ok {
				t.Fatalf("audio_urls should not be sent for sync-lipsync")
			}
			if _, ok := body["video_urls"]; ok {
				t.Fatalf("video_urls should not be sent for sync-lipsync")
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "req_sync"})
		case "/fal-ai/sync-lipsync/v3/image-to-video/requests/req_sync/status":
			w.WriteHeader(http.StatusNotFound)
		case "/fal-ai/sync-lipsync/requests/req_sync/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/fal-ai/sync-lipsync/requests/req_sync":
			_ = json.NewEncoder(w).Encode(map[string]any{"video": map[string]any{"url": "https://example.com/sync.mp4"}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()

	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:     "fal-ai/sync-lipsync/v3/image-to-video",
		ImageURL:  "https://example.com/avatar.jpg",
		AudioURLs: []string{"https://example.com/voice.wav"},
		SyncMode:  "bounce",
	})
	if err != nil {
		t.Fatalf("GenerateVideo() error = %v", err)
	}
	if resp.VideoURL != "https://example.com/sync.mp4" {
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
		if body["fps"] != float64(25) {
			t.Fatalf("fps = %#v", body["fps"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"video": map[string]any{"url": "https://example.com/i2v.mp4"}})
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()

	_, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:           "bytedance/seedance-2.0/image-to-video",
		Prompt:          "slow tilt down",
		ImageURL:        "https://example.com/start.jpg",
		EndImageURL:     "https://example.com/end.jpg",
		Duration:        "10",
		Resolution:      "720p",
		FramesPerSecond: 25,
	})
	if err != nil {
		t.Fatalf("GenerateVideo() error = %v", err)
	}
}

func TestGenerateVideo_MiniMaxH3SelectsTextEndpointAndNativeTypes(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/minimax/h3/text-to-video" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"video": map[string]any{"url": "https://example.com/h3.mp4"}})
	}))
	defer server.Close()
	p := New("test-key")
	p.baseURL, p.client = server.URL, server.Client()

	_, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model: "minimax/h3/text-to-video", Prompt: "cinematic garden", Duration: "5",
		Resolution: "2k", AspectRatio: "16:9", FramesPerSecond: 24,
	})
	if err != nil {
		t.Fatal(err)
	}
	if body["duration"] != float64(5) || body["resolution"] != "2K" || body["aspect_ratio"] != "16:9" {
		t.Fatalf("body = %#v", body)
	}
	if _, ok := body["fps"]; ok {
		t.Fatalf("H3 must not receive fps: %#v", body)
	}
}

func TestGenerateVideo_MiniMaxH3SelectsImageAndReferenceEndpoints(t *testing.T) {
	paths := make([]string, 0, 2)
	bodies := make([]map[string]any, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		paths, bodies = append(paths, r.URL.Path), append(bodies, body)
		_ = json.NewEncoder(w).Encode(map[string]any{"video": map[string]any{"url": "https://example.com/h3.mp4"}})
	}))
	defer server.Close()
	p := New("test-key")
	p.baseURL, p.client = server.URL, server.Client()

	_, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model: "minimax/h3/text-to-video", Prompt: "animate", ImageURL: "https://example.com/first.jpg",
		EndImageURL: "https://example.com/last.jpg", Duration: "10", Resolution: "2K", AspectRatio: "9:16",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model: "minimax/h3/text-to-video", Prompt: "Image 1 follows Audio 1",
		ImageURLs: []string{"https://example.com/ref.jpg"}, AudioURLs: []string{"https://example.com/ref.wav"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if paths[0] != "/minimax/h3/image-to-video" || bodies[0]["image_url"] != "https://example.com/first.jpg" || bodies[0]["end_image_url"] != "https://example.com/last.jpg" {
		t.Fatalf("image request path/body = %s %#v", paths[0], bodies[0])
	}
	if _, ok := bodies[0]["aspect_ratio"]; ok {
		t.Fatalf("image request must follow input aspect: %#v", bodies[0])
	}
	if paths[1] != "/minimax/h3/reference-to-video" {
		t.Fatalf("reference path = %s", paths[1])
	}
	if got := bodies[1]["reference_image_urls"].([]any); len(got) != 1 {
		t.Fatalf("reference images = %#v", got)
	}
	if got := bodies[1]["reference_audio_urls"].([]any); len(got) != 1 {
		t.Fatalf("reference audio = %#v", got)
	}
}

func TestRigMesh_SubmitsModelURLAndParsesRiggedAssets(t *testing.T) {
	var gotSubmit map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fal-ai/meshy/rigging":
			if got := r.Header.Get("Authorization"); got != "Key test-key" {
				t.Fatalf("unexpected auth header: %s", got)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotSubmit); err != nil {
				t.Fatalf("decode submit: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "rig_1"})
		case "/fal-ai/meshy/rigging/requests/rig_1/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/fal-ai/meshy/rigging/requests/rig_1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"rigged_character_glb": map[string]any{"url": "https://example.com/rigged.glb", "content_type": "model/gltf-binary"},
				"rigged_character_fbx": map[string]any{"url": "https://example.com/rigged.fbx"},
				"basic_animations":     map[string]any{"walking_glb": map[string]any{"url": "https://example.com/walk.glb"}},
				"rig_task_id":          "task_abc",
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()
	height := 1.8

	resp, err := p.RigMesh(context.Background(), &model.MeshRiggingRequest{
		Model:        "fal-ai/meshy/rigging",
		ModelURL:     "https://example.com/input.glb",
		HeightMeters: &height,
	})
	if err != nil {
		t.Fatalf("RigMesh() error = %v", err)
	}
	if resp.RiggedCharacterGLB.URL != "https://example.com/rigged.glb" {
		t.Fatalf("rigged glb url = %q", resp.RiggedCharacterGLB.URL)
	}
	if resp.RiggedCharacterFBX == nil || resp.RiggedCharacterFBX.URL != "https://example.com/rigged.fbx" {
		t.Fatalf("rigged fbx = %#v", resp.RiggedCharacterFBX)
	}
	if resp.BasicAnimations == nil || resp.BasicAnimations.WalkingGLB == nil {
		t.Fatalf("basic animations = %#v", resp.BasicAnimations)
	}
	if resp.RigTaskID != "task_abc" {
		t.Fatalf("rig task id = %q", resp.RigTaskID)
	}
	if gotSubmit["model_url"] != "https://example.com/input.glb" || gotSubmit["height_meters"] != 1.8 {
		t.Fatalf("submit = %#v", gotSubmit)
	}
	if _, ok := gotSubmit["enable_animation"]; ok {
		t.Fatalf("enable_animation should be omitted when false: %#v", gotSubmit)
	}
	if resp.Billing == nil || resp.Billing.ExternalCostCents != 20 {
		t.Fatalf("billing = %#v", resp.Billing)
	}
}

func TestRigMesh_AnimationSurcharge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fal-ai/meshy/rigging":
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "rig_2"})
		case "/fal-ai/meshy/rigging/requests/rig_2/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/fal-ai/meshy/rigging/requests/rig_2":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"rigged_character_glb": map[string]any{"url": "https://example.com/rigged.glb"},
				"animation_glb":        map[string]any{"url": "https://example.com/anim.glb"},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()

	resp, err := p.RigMesh(context.Background(), &model.MeshRiggingRequest{
		Model:           "fal-ai/meshy/rigging",
		ModelURL:        "https://example.com/input.glb",
		EnableAnimation: true,
	})
	if err != nil {
		t.Fatalf("RigMesh() error = %v", err)
	}
	if resp.Billing == nil || resp.Billing.ExternalCostCents != 32 || !resp.Billing.Animation {
		t.Fatalf("billing = %#v", resp.Billing)
	}
	if resp.AnimationGLB == nil || resp.AnimationGLB.URL != "https://example.com/anim.glb" {
		t.Fatalf("animation glb = %#v", resp.AnimationGLB)
	}
}

func TestRigMesh_ErrorsWhenNoRiggedGLB(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fal-ai/meshy/rigging":
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "rig_3"})
		case "/fal-ai/meshy/rigging/requests/rig_3/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/fal-ai/meshy/rigging/requests/rig_3":
			_ = json.NewEncoder(w).Encode(map[string]any{"rig_task_id": "task_x"})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()

	if _, err := p.RigMesh(context.Background(), &model.MeshRiggingRequest{
		Model:    "fal-ai/meshy/rigging",
		ModelURL: "https://example.com/input.glb",
	}); err == nil {
		t.Fatal("expected error when rigged_character_glb is missing")
	}
}

func TestGenerate3D_MeshyV6UsesImageToShapeRequestAndParsesGLB(t *testing.T) {
	var gotSubmit map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fal-ai/meshy/v6/image-to-3d":
			if err := json.NewDecoder(r.Body).Decode(&gotSubmit); err != nil {
				t.Fatalf("decode submit: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "m6_1"})
		case "/fal-ai/meshy/v6/image-to-3d/requests/m6_1/status":
			w.WriteHeader(http.StatusMethodNotAllowed)
		case "/fal-ai/meshy/requests/m6_1/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/fal-ai/meshy/requests/m6_1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"model_glb": map[string]any{"url": "https://example.com/meshy.glb"},
				"seed":      float64(11),
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()
	pc := 12000

	resp, err := p.Generate3D(context.Background(), &model.Model3DGenerationRequest{
		Model:           "fal-ai/meshy/v6/image-to-3d",
		ImageURL:        "https://example.com/in.png",
		Topology:        "quad",
		TargetPolycount: pc,
	})
	if err != nil {
		t.Fatalf("Generate3D() error = %v", err)
	}
	if resp.ModelGLB.URL != "https://example.com/meshy.glb" {
		t.Fatalf("model glb url = %q", resp.ModelGLB.URL)
	}
	if resp.Billing == nil || resp.Billing.ExternalCostCents != 80 {
		t.Fatalf("billing = %#v", resp.Billing)
	}
	if gotSubmit["image_url"] != "https://example.com/in.png" || gotSubmit["topology"] != "quad" {
		t.Fatalf("submit = %#v", gotSubmit)
	}
	if gotSubmit["target_polycount"] != float64(pc) {
		t.Fatalf("target_polycount = %#v", gotSubmit["target_polycount"])
	}
	// Pixal-only slat params must NOT be sent to meshy.
	if _, ok := gotSubmit["ss_guidance_strength"]; ok {
		t.Fatalf("meshy request leaked pixal params: %#v", gotSubmit)
	}
}

func TestGenerate3D_TripoP1ParsesModelUrlsGLBAndTextureCost(t *testing.T) {
	var gotSubmit map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tripo3d/p1/image-to-3d":
			if err := json.NewDecoder(r.Body).Decode(&gotSubmit); err != nil {
				t.Fatalf("decode submit: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "tp_1"})
		case "/tripo3d/p1/image-to-3d/requests/tp_1/status":
			w.WriteHeader(http.StatusMethodNotAllowed)
		case "/tripo3d/p1/requests/tp_1/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/tripo3d/p1/requests/tp_1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"model_mesh": map[string]any{"url": "https://example.com/mesh.glb"},
				"model_urls": map[string]any{"glb": map[string]any{"url": "https://example.com/tripo.glb"}},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()
	off := false

	resp, err := p.Generate3D(context.Background(), &model.Model3DGenerationRequest{
		Model:         "tripo3d/p1/image-to-3d",
		ImageURL:      "https://example.com/in.png",
		ShouldTexture: &off,
	})
	if err != nil {
		t.Fatalf("Generate3D() error = %v", err)
	}
	if resp.ModelGLB.URL != "https://example.com/tripo.glb" {
		t.Fatalf("expected model_urls.glb preferred, got %q", resp.ModelGLB.URL)
	}
	if resp.Billing == nil || resp.Billing.ExternalCostCents != 40 {
		t.Fatalf("texture-off billing = %#v, want 40c", resp.Billing)
	}
	if gotSubmit["texture"] != false || gotSubmit["image_url"] != "https://example.com/in.png" {
		t.Fatalf("submit = %#v", gotSubmit)
	}
}

func TestGenerate3D_TrellisRetextureSendsMeshAndResolutionCost(t *testing.T) {
	var gotSubmit map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fal-ai/trellis-2/retexture":
			if err := json.NewDecoder(r.Body).Decode(&gotSubmit); err != nil {
				t.Fatalf("decode submit: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "tr_1"})
		case "/fal-ai/trellis-2/retexture/requests/tr_1/status":
			w.WriteHeader(http.StatusMethodNotAllowed)
		case "/fal-ai/trellis-2/requests/tr_1/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/fal-ai/trellis-2/requests/tr_1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"model_glb": map[string]any{"url": "https://example.com/retex.glb"},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()

	resp, err := p.Generate3D(context.Background(), &model.Model3DGenerationRequest{
		Model:      "fal-ai/trellis-2/retexture",
		ImageURL:   "https://example.com/ref.png",
		MeshURL:    "https://example.com/in.glb",
		Resolution: 512,
	})
	if err != nil {
		t.Fatalf("Generate3D() error = %v", err)
	}
	if resp.ModelGLB.URL != "https://example.com/retex.glb" {
		t.Fatalf("model glb url = %q", resp.ModelGLB.URL)
	}
	if resp.Billing == nil || resp.Billing.ExternalCostCents != 20 {
		t.Fatalf("512p billing = %#v, want 20c", resp.Billing)
	}
	if gotSubmit["mesh_url"] != "https://example.com/in.glb" || gotSubmit["image_url"] != "https://example.com/ref.png" {
		t.Fatalf("submit = %#v", gotSubmit)
	}
	if gotSubmit["resolution"] != float64(512) {
		t.Fatalf("resolution = %#v", gotSubmit["resolution"])
	}
}

func TestGenerateVideo_FluxUpscaleSendsSingularVideoURLAndStripsGenerationFields(t *testing.T) {
	var gotSubmit map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/blackforestlabs/flux-video-upscale":
			if err := json.NewDecoder(r.Body).Decode(&gotSubmit); err != nil {
				t.Fatalf("decode submit: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"request_id": "fu_1"})
		case "/blackforestlabs/flux-video-upscale/requests/fu_1/status":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "COMPLETED"})
		case "/blackforestlabs/flux-video-upscale/requests/fu_1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"video": map[string]any{"url": "https://example.com/upscaled.mp4"},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()
	factor := 2.5
	creativity := 1

	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:           "blackforestlabs/flux-video-upscale",
		Prompt:          "sharpen fine texture detail",
		Resolution:      "1080p",
		Duration:        "10",
		AspectRatio:     "16:9",
		FramesPerSecond: 24,
		Seed:            new(int),
		GenerateAudio:   new(bool),
		ImageURLs:       []string{"https://example.com/ref.png"},
		VideoURLs:       []string{"https://example.com/in.mp4"},
		UpscaleFactor:   &factor,
		Creativity:      &creativity,
		SafetyTolerance: 3,
	})
	if err != nil {
		t.Fatalf("GenerateVideo() error = %v", err)
	}
	if resp.VideoURL != "https://example.com/upscaled.mp4" {
		t.Fatalf("VideoURL = %q", resp.VideoURL)
	}
	if gotSubmit["video_url"] != "https://example.com/in.mp4" {
		t.Fatalf("video_url = %#v", gotSubmit["video_url"])
	}
	if gotSubmit["upscale_factor"] != float64(2.5) || gotSubmit["creativity"] != float64(1) {
		t.Fatalf("upscale args = %#v", gotSubmit)
	}
	if gotSubmit["safety_tolerance"] != float64(3) {
		t.Fatalf("safety_tolerance = %#v", gotSubmit["safety_tolerance"])
	}
	for _, key := range []string{"resolution", "duration", "aspect_ratio", "fps", "generate_audio", "seed", "image_urls", "video_urls"} {
		if _, ok := gotSubmit[key]; ok {
			t.Fatalf("%s should not be sent for flux-video-upscale: %#v", key, gotSubmit)
		}
	}
}
