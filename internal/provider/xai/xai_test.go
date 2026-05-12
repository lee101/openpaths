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
	})
	if err != nil {
		t.Fatalf("GenerateImage error: %v", err)
	}
	if got["model"] != "grok-imagine-image" || got["prompt"] != "A cat on a rocket" || got["aspect_ratio"] != "16:9" {
		t.Fatalf("request = %#v", got)
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
