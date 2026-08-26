package manifoldgen

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

func newTestProvider(t *testing.T, handler http.HandlerFunc) (*Provider, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New("sk-mg-test", srv.URL), srv
}

func TestGenerateKFoldVideoSubmitsAndPolls(t *testing.T) {
	polls := 0
	var gotService string
	var gotAuth string
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/service":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			gotService, _ = body["service"].(string)
			if body["prompt"] != "a lighthouse in a storm" {
				t.Fatalf("prompt = %v", body["prompt"])
			}
			if body["aspect_ratio"] != "9:16" {
				t.Fatalf("aspect_ratio = %v", body["aspect_ratio"])
			}
			if body["duration"] != float64(5) {
				t.Fatalf("duration = %v", body["duration"])
			}
			json.NewEncoder(w).Encode(map[string]any{
				"service": "video",
				"result":  map[string]any{"job_id": "job1", "status": "queued", "status_url": "/api/video-jobs/job1"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/video-jobs/job1":
			polls++
			status := "processing"
			result := map[string]any{}
			if polls >= 2 {
				status = "completed"
				result["video_url"] = "https://static.example/clip.webm"
			}
			w.Header().Set("Content-Type", "application/json")
			code := http.StatusOK
			if status != "completed" {
				code = http.StatusAccepted
			}
			w.WriteHeader(code)
			json.NewEncoder(w).Encode(map[string]any{
				"job": map[string]any{"job_id": "job1", "status": status, "result": result},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:       "kfold-video",
		Prompt:      "a lighthouse in a storm",
		Duration:    "5",
		AspectRatio: "9:16",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.VideoURL != "https://static.example/clip.webm" || resp.BackendUsed != "manifoldgen" {
		t.Fatalf("resp = %#v", resp)
	}
	if gotService != "h3_video" {
		t.Fatalf("service = %q", gotService)
	}
	if gotAuth != "Bearer sk-mg-test" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if polls < 2 {
		t.Fatalf("polls = %d, want at least 2", polls)
	}
}

func TestGenerateCharacterAnimationValidationAndSubmit(t *testing.T) {
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no HTTP call expected for invalid requests")
	})

	if _, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{Model: "wan-animate", Prompt: "p", VideoURL: "https://x/v.mp4"}); err == nil {
		t.Fatal("expected image_url validation error")
	}
	if _, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{Model: "wan-animate", Prompt: "p", ImageURL: "https://x/i.png"}); err == nil {
		t.Fatal("expected video_url validation error")
	}
}

func TestGenerateCharacterAnimationHappyPath(t *testing.T) {
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/service" {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["service"] != "character_animation" {
				t.Fatalf("service = %v", body["service"])
			}
			if body["image_url"] != "https://x/i.png" || body["video_url"] != "https://x/v.mp4" {
				t.Fatalf("body = %#v", body)
			}
			if body["service_tier"] != "xfast" {
				t.Fatalf("service_tier = %v", body["service_tier"])
			}
			json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{"job_id": "j2", "status": "queued", "status_url": "/api/video-jobs/j2"},
			})
			return
		}
		if r.URL.Path == "/api/video-jobs/j2" {
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]any{
				"job": map[string]any{
					"status": "completed",
					"result": map[string]any{"video_url": "https://static.example/dance.mp4"},
				},
			})
			return
		}
		t.Fatalf("unexpected path %s", r.URL.Path)
	})

	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model: "wan-animate-xfast", Prompt: "dance", ImageURL: "https://x/i.png", VideoURL: "https://x/v.mp4", Duration: "6",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.VideoURL != "https://static.example/dance.mp4" {
		t.Fatalf("resp = %#v", resp)
	}
}

func TestGenerateRestyleRequiresVideoAndSendsLicense(t *testing.T) {
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no HTTP call expected")
	})
	if _, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{Model: "h3-control", Prompt: "p", ControlType: "pose"}); err == nil {
		t.Fatal("expected video_url validation error")
	}
	if _, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{Model: "h3-control", Prompt: "p", VideoURL: "https://x/v.mp4", ControlType: "inpaint"}); err == nil {
		t.Fatal("expected mask_video_url validation error for inpaint")
	}
}

func TestGenerateRestyleHappyPath(t *testing.T) {
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/service" {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["accept_h3_license"] != true {
				t.Fatalf("accept_h3_license = %v", body["accept_h3_license"])
			}
			if body["control_type"] != "depth" {
				t.Fatalf("control_type = %v", body["control_type"])
			}
			json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{"job_id": "j3", "status": "queued", "status_url": "/api/video-jobs/j3"},
			})
			return
		}
		if r.URL.Path == "/api/video-jobs/j3" {
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]any{
				"job": map[string]any{"status": "failed", "error": "worker crashed"},
			})
			return
		}
		t.Fatalf("unexpected path %s", r.URL.Path)
	})

	_, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model: "h3-control", Prompt: "restyle", VideoURL: "https://x/v.mp4", ControlType: "Depth",
	})
	pe, ok := err.(*provider.ProviderError)
	if !ok || pe.Retryable {
		t.Fatalf("err = %v, want non-retryable ProviderError", err)
	}
}

func TestBackgroundRemovalMapsOutput(t *testing.T) {
	var gotBody map[string]any
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/service" {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["service"] != "video_background_removal" || body["output_format"] != "webm_vp9" {
				t.Fatalf("body = %#v", body)
			}
			gotBody = body
			json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{"job_id": "j4", "status_url": "/api/video-jobs/j4"},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"job": map[string]any{
				"status": "completed",
				"result": map[string]any{"video_url": "https://static.example/matte.webm", "content_type": "video/webm"},
			},
		})
	})

	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model: "remove-video-background", VideoURL: "https://x/in.mp4",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.VideoURL != "https://static.example/matte.webm" || resp.OutputFormat != "video/webm" {
		t.Fatalf("resp = %#v", resp)
	}
	if gotBody["background_color"] != "transparent" {
		t.Fatalf("default background_color = %v, want transparent", gotBody["background_color"])
	}

	if _, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model: "remove-video-background", VideoURL: "https://x/in.mp4", BackgroundColor: "#00ff00",
	}); err != nil {
		t.Fatal(err)
	}
	if gotBody["background_color"] != "#00ff00" {
		t.Fatalf("background_color = %v, want #00ff00", gotBody["background_color"])
	}
}

func TestGenerateDramatizeHappyPath(t *testing.T) {
	polls := 0
	var gotService string
	var gotBody map[string]any
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/service":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			gotService, _ = body["service"].(string)
			gotBody = body
			json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{"job_id": "d1", "status": "queued", "status_url": "/api/video-jobs/d1"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/video-jobs/d1":
			polls++
			status := "processing"
			result := map[string]any{}
			if polls >= 2 {
				status = "completed"
				result["video_url"] = "https://static.example/dramatize.mp4"
			}
			code := http.StatusOK
			if status != "completed" {
				code = http.StatusAccepted
			}
			w.WriteHeader(code)
			json.NewEncoder(w).Encode(map[string]any{
				"job": map[string]any{"status": status, "result": result},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:    "video-dramatize",
		Prompt:   "turn this into a six-shot trailer",
		VideoURL: "https://x/source.mp4",
		Duration: "30",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.VideoURL != "https://static.example/dramatize.mp4" || resp.BackendUsed != "manifoldgen" {
		t.Fatalf("resp = %#v", resp)
	}
	if gotService != "video_dramatize" {
		t.Fatalf("service = %q", gotService)
	}
	if gotBody["video_url"] != "https://x/source.mp4" {
		t.Fatalf("video_url = %v", gotBody["video_url"])
	}
	if gotBody["max_shots"] != float64(6) || gotBody["seconds"] != float64(5) {
		t.Fatalf("plan = %#v", gotBody)
	}
	if polls < 2 {
		t.Fatalf("polls = %d, want at least 2", polls)
	}
}

func TestGenerateDramatizeRequiresPromptVideoAndDuration(t *testing.T) {
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no HTTP call expected for invalid requests")
	})
	if _, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{Model: "video-dramatize", VideoURL: "https://x/v.mp4", Duration: "30"}); err == nil {
		t.Fatal("expected prompt validation error")
	}
	if _, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{Model: "video-dramatize", Prompt: "p", Duration: "30"}); err == nil {
		t.Fatal("expected video_url validation error")
	}
	if _, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{Model: "video-dramatize", Prompt: "p", VideoURL: "https://x/v.mp4"}); err == nil {
		t.Fatal("expected duration validation error")
	}
}

func TestGenerateDramatizeClampsAndHonorsConfig(t *testing.T) {
	var gotBody map[string]any
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		gotBody = body
		json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{"video_url": "https://static.example/edit.mp4"},
		})
	})

	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:            "video-dramatize",
		Prompt:           "p",
		VideoURL:         "https://x/v.mp4",
		Duration:         "250",
		GenerationConfig: json.RawMessage(`{"max_shots": 99, "seconds_per_shot": 1}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.VideoURL != "https://static.example/edit.mp4" {
		t.Fatalf("resp = %#v", resp)
	}
	if gotBody["max_shots"] != float64(10) || gotBody["seconds"] != float64(2) {
		t.Fatalf("overrides not clamped: %#v", gotBody)
	}
}

func TestGenerateMusicPollsAudioJob(t *testing.T) {
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/service" {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["service"] != "music_generation" {
				t.Fatalf("service = %v", body["service"])
			}
			if body["lyrics"] != "verse one" {
				t.Fatalf("lyrics = %v", body["lyrics"])
			}
			json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{"job_id": "a1", "status_url": "/api/audio-jobs/a1"},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"job": map[string]any{
				"status": "completed",
				"result": map[string]any{"audio_url": "https://static.example/song.mp3", "duration_seconds": 32.0},
			},
		})
	})

	resp, err := p.GenerateMusic(context.Background(), &model.MusicGenerationRequest{
		Model: "mg-music", Prompt: "dreamlike ambient score with glass harmonics", Lyrics: "verse one",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Data == nil || resp.Data.Audio != "https://static.example/song.mp3" {
		t.Fatalf("resp = %#v", resp)
	}
	if resp.ExtraInfo == nil || resp.ExtraInfo.Duration != 32 {
		t.Fatalf("extra = %#v", resp.ExtraInfo)
	}
}

func TestGenerateMusicRequiresPrompt(t *testing.T) {
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no HTTP call expected")
	})
	if _, err := p.GenerateMusic(context.Background(), &model.MusicGenerationRequest{Model: "mg-music", Prompt: "short"}); err == nil {
		t.Fatal("expected short-prompt validation error")
	}
}

func TestGenerateMusicPassesDuration(t *testing.T) {
	var gotBody map[string]any
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		gotBody = body
		json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{"audio_url": "https://static.example/song.mp3"},
		})
	})

	if _, err := p.GenerateMusic(context.Background(), &model.MusicGenerationRequest{
		Model: "mg-music", Prompt: "dreamlike ambient score with glass harmonics", Duration: 500,
	}); err != nil {
		t.Fatal(err)
	}
	if gotBody["duration"] != float64(300) {
		t.Fatalf("duration = %v, want clamped 300", gotBody["duration"])
	}

	if _, err := p.GenerateMusic(context.Background(), &model.MusicGenerationRequest{
		Model: "mg-sfx", Prompt: "thunder crack", Duration: 120,
	}); err != nil {
		t.Fatal(err)
	}
	if _, ok := gotBody["duration"]; ok {
		t.Fatalf("sfx must not carry duration: %#v", gotBody)
	}
}

func TestGenerateSFX(t *testing.T) {
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/service" {
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["service"] != "sfx_generation" || body["size"] != "audio" {
				t.Fatalf("body = %#v", body)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{"job_id": "s1", "status_url": "/api/audio-jobs/s1"},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"job": map[string]any{"status": "completed", "result": map[string]any{"audio_url": "https://static.example/boom.wav"}},
		})
	})

	resp, err := p.GenerateMusic(context.Background(), &model.MusicGenerationRequest{Model: "mg-sfx", Prompt: "thunder crack"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Data.Audio != "https://static.example/boom.wav" {
		t.Fatalf("audio = %q", resp.Data.Audio)
	}
}

func TestGenerateSpeechSyncEnvelope(t *testing.T) {
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["service"] != "tts" {
			t.Fatalf("service = %v", body["service"])
		}
		if body["text"] != "hello there" || body["voice"] != "F1" {
			t.Fatalf("body = %#v", body)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"result":         map[string]any{"audio_base64": "QUJD", "format": "wav"},
			"credits_remain": 99.0,
		})
	})

	resp, err := p.GenerateSpeech(context.Background(), &model.SpeechRequest{Model: "mg-tts", Input: "hello there", Voice: "F1"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Audio != "QUJD" || resp.Format != "wav" {
		t.Fatalf("resp = %#v", resp)
	}
}

func TestImageEditValidationAndResult(t *testing.T) {
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no HTTP call expected")
	})
	if _, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{Model: "h3-image-edit", Prompt: "p"}); err == nil {
		t.Fatal("expected image_url validation error")
	}
}

func TestGenerateH3ImageJobResult(t *testing.T) {
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/service" {
			var payload map[string]any
			json.NewDecoder(r.Body).Decode(&payload)
			if payload["service"] != "h3_image_edit" {
				t.Fatalf("service = %v", payload["service"])
			}
			if payload["reference_image_urls"] == nil {
				t.Fatalf("references missing: %#v", payload)
			}
			json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{"job_id": "i1", "status_url": "/api/video-jobs/i1"},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"job": map[string]any{
				"status": "completed",
				"result": map[string]any{"image_url": "https://static.example/out.webp", "is_nsfw": false},
			},
		})
	})

	resp, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model: "h3-image-edit", Prompt: "make it dusk", ImageURL: "https://x/src.png",
		ReferenceImageURLs: []string{"https://x/ref.png"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 || resp.Data[0].URL != "https://static.example/out.webp" {
		t.Fatalf("resp = %#v", resp)
	}
}

func TestPaymentRequiredSurfaces402(t *testing.T) {
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/service" {
			json.NewEncoder(w).Encode(map[string]any{
				"result": map[string]any{"job_id": "p1", "status_url": "/api/video-jobs/p1"},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"job": map[string]any{"status": "payment_required", "error": "top up to release"},
		})
	})

	_, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model: "kfold-video", Prompt: "clip",
	})
	pe, ok := err.(*provider.ProviderError)
	if !ok || pe.StatusCode != 402 {
		t.Fatalf("err = %v, want 402 ProviderError", err)
	}
}

func TestParseSize(t *testing.T) {
	if w, h, ok := parseSize("1024x768"); !ok || w != 1024 || h != 768 {
		t.Fatalf("parseSize = %d %d %v", w, h, ok)
	}
	if _, _, ok := parseSize("auto"); ok {
		t.Fatal("auto should not parse")
	}
}
