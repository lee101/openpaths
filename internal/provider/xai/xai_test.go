package xai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func TestGenerateSpeechCallsXAITTS(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/tts" {
			t.Fatalf("path = %s, want /v1/tts", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header = %q", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("mp3-bytes"))
	}))
	defer srv.Close()

	p := New("test-key", srv.URL)
	resp, err := p.GenerateSpeech(context.Background(), &model.SpeechRequest{
		Text:    "Hello",
		VoiceID: "ara",
	})
	if err != nil {
		t.Fatalf("GenerateSpeech error: %v", err)
	}
	if got["text"] != "Hello" || got["voice_id"] != "ara" || got["language"] != "en" {
		t.Fatalf("request = %#v", got)
	}
	wantAudio := base64.StdEncoding.EncodeToString([]byte("mp3-bytes"))
	if resp.Audio != wantAudio {
		t.Fatalf("audio = %q, want %q", resp.Audio, wantAudio)
	}
	if resp.Characters != 5 {
		t.Fatalf("characters = %d, want 5", resp.Characters)
	}
}

func TestGenerateImageCallsXAIImageGeneration(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Fatalf("path = %s, want /v1/images/generations", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"created":123,"data":[{"url":"https://example.test/out.png"}]}`))
	}))
	defer srv.Close()

	p := New("test-key", srv.URL)
	resp, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model:       "grok-imagine-image",
		Prompt:      "A cat on a rocket",
		N:           2,
		AspectRatio: "16:9",
		Resolution:  "2k",
	})
	if err != nil {
		t.Fatalf("GenerateImage error: %v", err)
	}
	if got["model"] != "grok-imagine-image" || got["prompt"] != "A cat on a rocket" || got["aspect_ratio"] != "16:9" {
		t.Fatalf("request = %#v", got)
	}
	if got["resolution"] != "2k" {
		t.Fatalf("resolution = %#v", got["resolution"])
	}
	if resp.Created != 123 || len(resp.Data) != 1 || resp.Data[0].URL == "" {
		t.Fatalf("response = %#v", resp)
	}
}

func TestGenerateImageCallsXAIImageEditsForMultipleInputs(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/edits" {
			t.Fatalf("path = %s, want /v1/images/edits", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"created":123,"data":[{"url":"https://example.test/edit.png"}]}`))
	}))
	defer srv.Close()

	p := New("test-key", srv.URL)
	_, err := p.GenerateImage(context.Background(), &model.ImageGenerationRequest{
		Model:       "grok-imagine-image",
		Prompt:      "merge subjects",
		ImageURLs:   []string{"https://example.test/a.jpg", "https://example.test/b.jpg"},
		AspectRatio: "3:2",
	})
	if err != nil {
		t.Fatalf("GenerateImage error: %v", err)
	}
	images, ok := got["images"].([]any)
	if !ok || len(images) != 2 {
		t.Fatalf("images = %#v, want two image inputs", got["images"])
	}
	first, _ := images[0].(map[string]any)
	if first["type"] != "image_url" || first["url"] != "https://example.test/a.jpg" {
		t.Fatalf("first image = %#v", first)
	}
	if got["aspect_ratio"] != "3:2" {
		t.Fatalf("aspect_ratio = %#v", got["aspect_ratio"])
	}
}

func TestGenerateVideoCallsXAIGenerationAndPolls(t *testing.T) {
	var submitted map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/videos/generations":
			if err := json.NewDecoder(r.Body).Decode(&submitted); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			_, _ = w.Write([]byte(`{"request_id":"req_123"}`))
		case "/v1/videos/req_123":
			_, _ = w.Write([]byte(`{"status":"done","video":{"url":"https://example.test/out.mp4","duration":6,"respect_moderation":true}}`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	p := New("test-key", srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := p.GenerateVideo(ctx, &model.VideoGenerationRequest{
		Model:       "grok-imagine-video-1.5",
		Prompt:      "A quiet lake at sunrise",
		Duration:    "6",
		Resolution:  "480p",
		AspectRatio: "16:9",
		ImageURL:    "https://example.test/start.png",
	})
	if err != nil {
		t.Fatalf("GenerateVideo error: %v", err)
	}
	if submitted["model"] != "grok-imagine-video-1.5" || submitted["duration"] != float64(6) || submitted["resolution"] != "480p" {
		t.Fatalf("request = %#v", submitted)
	}
	image, _ := submitted["image"].(map[string]any)
	if image["url"] != "https://example.test/start.png" {
		t.Fatalf("image = %#v", image)
	}
	if resp.VideoURL != "https://example.test/out.mp4" {
		t.Fatalf("video_url = %q", resp.VideoURL)
	}
}

func TestGenerateVideoCallsXAIVideoExtension(t *testing.T) {
	var submitted map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/videos/extensions":
			if err := json.NewDecoder(r.Body).Decode(&submitted); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			_, _ = w.Write([]byte(`{"request_id":"req_ext"}`))
		case "/v1/videos/req_ext":
			_, _ = w.Write([]byte(`{"status":"done","video":{"url":"https://example.test/extended.mp4","duration":12}}`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	p := New("test-key", srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := p.GenerateVideo(ctx, &model.VideoGenerationRequest{
		Model:     "grok-imagine-video",
		Prompt:    "The camera pulls back to reveal skyline",
		Duration:  "6",
		Operation: "extension",
		Video:     &model.VideoInput{URL: "https://example.test/source.mp4"},
	})
	if err != nil {
		t.Fatalf("GenerateVideo extension error: %v", err)
	}
	video, _ := submitted["video"].(map[string]any)
	if video["url"] != "https://example.test/source.mp4" {
		t.Fatalf("video = %#v", video)
	}
	if _, ok := submitted["aspect_ratio"]; ok {
		t.Fatalf("extension request should not send aspect_ratio: %#v", submitted)
	}
	if resp.VideoURL != "https://example.test/extended.mp4" {
		t.Fatalf("video_url = %q", resp.VideoURL)
	}
}

func TestGenerateVideoCallsXAIVideoEdit(t *testing.T) {
	var submitted map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/videos/edits":
			if err := json.NewDecoder(r.Body).Decode(&submitted); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			_, _ = w.Write([]byte(`{"request_id":"req_edit"}`))
		case "/v1/videos/req_edit":
			_, _ = w.Write([]byte(`{"status":"done","video":{"url":"https://example.test/edited.mp4","duration":7}}`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	p := New("test-key", srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := p.GenerateVideo(ctx, &model.VideoGenerationRequest{
		Model:       "grok-imagine-video",
		Prompt:      "Give the woman a silver necklace",
		Operation:   "edit",
		VideoURL:    "https://example.test/source.mp4",
		Duration:    "6",
		Resolution:  "720p",
		AspectRatio: "16:9",
	})
	if err != nil {
		t.Fatalf("GenerateVideo edit error: %v", err)
	}
	video, _ := submitted["video"].(map[string]any)
	if video["url"] != "https://example.test/source.mp4" {
		t.Fatalf("video = %#v", video)
	}
	if _, ok := submitted["duration"]; ok {
		t.Fatalf("edit request should not send duration: %#v", submitted)
	}
	if _, ok := submitted["resolution"]; ok {
		t.Fatalf("edit request should not send resolution: %#v", submitted)
	}
	if _, ok := submitted["aspect_ratio"]; ok {
		t.Fatalf("edit request should not send aspect_ratio: %#v", submitted)
	}
	if resp.VideoURL != "https://example.test/edited.mp4" {
		t.Fatalf("video_url = %q", resp.VideoURL)
	}
}

func TestTranscribeCallsXAISTT(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/stt" {
			t.Fatalf("path = %s, want /v1/stt", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("authorization header = %q", r.Header.Get("Authorization"))
		}
		if err := r.ParseMultipartForm(1024); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if r.FormValue("language") != "en" || r.FormValue("format") != "true" {
			t.Fatalf("form language=%q format=%q", r.FormValue("language"), r.FormValue("format"))
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if strings.TrimSpace(string(data)) != "audio" {
			t.Fatalf("file data = %q", string(data))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"text":"hello world"}`))
	}))
	defer srv.Close()

	p := New("test-key", srv.URL)
	resp, err := p.Transcribe(context.Background(), &model.TranscriptionRequest{
		File:     []byte("audio\n"),
		Filename: "audio.mp3",
		Language: "en",
	})
	if err != nil {
		t.Fatalf("Transcribe error: %v", err)
	}
	if resp.Text != "hello world" {
		t.Fatalf("text = %q, want hello world", resp.Text)
	}
}
