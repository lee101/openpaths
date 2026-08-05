package minimax

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

func TestGenerateVideoH3UsesV2TaskAPI(t *testing.T) {
	var submitted minimaxVideoV2Req
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("Authorization = %q", got)
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/video_generation":
			if err := json.NewDecoder(r.Body).Decode(&submitted); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"task_id": "task-123"})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/query/video_generation/task-123":
			_ = json.NewEncoder(w).Encode(map[string]any{"task": map[string]any{
				"status":  "succeeded",
				"content": map[string]string{"url": "https://cdn.example.com/h3.mp4"},
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	p := New("test-key")
	p.baseURL = server.URL
	p.client = server.Client()
	p.pollInterval = time.Millisecond
	resp, err := p.GenerateVideo(context.Background(), &model.VideoGenerationRequest{
		Model:       "MiniMax-H3",
		Prompt:      "A lighthouse in a storm",
		Resolution:  "2k",
		Duration:    "7",
		AspectRatio: "21:9",
	})
	if err != nil {
		t.Fatalf("GenerateVideo: %v", err)
	}
	if resp.VideoURL != "https://cdn.example.com/h3.mp4" {
		t.Fatalf("VideoURL = %q", resp.VideoURL)
	}
	if submitted.Model != "MiniMax-H3" || submitted.Resolution != "2K" || submitted.Duration != 7 || submitted.Ratio != "21:9" {
		t.Fatalf("submitted request = %+v", submitted)
	}
	if len(submitted.Content) != 1 || submitted.Content[0].Type != "text" || submitted.Content[0].Text == "" {
		t.Fatalf("content = %#v", submitted.Content)
	}
}

func TestBuildVideoV2RequestMapsFrameInputs(t *testing.T) {
	req, err := buildVideoV2Request(&model.VideoGenerationRequest{
		Model:       "MiniMax-H3",
		Prompt:      "Grow from seedling to flower",
		ImageURL:    "https://cdn.example.com/first.png",
		EndImageURL: "https://cdn.example.com/last.png",
		Resolution:  "768p",
		Duration:    "15",
		AspectRatio: "9:16",
	})
	if err != nil {
		t.Fatalf("buildVideoV2Request: %v", err)
	}
	if req.Ratio != "adaptive" {
		t.Fatalf("Ratio = %q, want adaptive", req.Ratio)
	}
	if len(req.Content) != 3 || req.Content[1].Role != "first_frame" || req.Content[2].Role != "last_frame" {
		t.Fatalf("content = %#v", req.Content)
	}
}

func TestBuildVideoV2RequestMapsReferenceInputs(t *testing.T) {
	req, err := buildVideoV2Request(&model.VideoGenerationRequest{
		Model:     "MiniMax-H3",
		Prompt:    "Match the character and voice",
		Ratio:     "4:3",
		ImageURLs: []string{"https://cdn.example.com/character.webp"},
		VideoURLs: []string{"https://cdn.example.com/motion.mp4"},
		AudioURLs: []string{"https://cdn.example.com/voice.mp3"},
	})
	if err != nil {
		t.Fatalf("buildVideoV2Request: %v", err)
	}
	if req.Resolution != "2K" || req.Duration != 5 || req.Ratio != "4:3" {
		t.Fatalf("defaults = resolution %q, duration %d, ratio %q", req.Resolution, req.Duration, req.Ratio)
	}
	wantRoles := []string{"", "reference_image", "reference_video", "reference_audio"}
	for i, role := range wantRoles {
		if req.Content[i].Role != role {
			t.Fatalf("content[%d].Role = %q, want %q", i, req.Content[i].Role, role)
		}
	}
}

func TestBuildVideoV2RequestRejectsAudioOnlyReference(t *testing.T) {
	_, err := buildVideoV2Request(&model.VideoGenerationRequest{
		Model: "MiniMax-H3", Prompt: "Speak", AudioURL: "https://cdn.example.com/voice.mp3",
	})
	pe, ok := err.(*provider.ProviderError)
	if !ok || pe.StatusCode != 400 {
		t.Fatalf("error = %#v, want non-retryable 400 ProviderError", err)
	}
}
