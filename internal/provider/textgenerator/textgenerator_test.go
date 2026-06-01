package textgenerator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestGenerateSpeech(t *testing.T) {
	const audio = "wav-bytes"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/generate_speech" {
			t.Fatalf("path = %q, want /api/v1/generate_speech", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("secret"); got != "test-secret" {
			t.Fatalf("secret header = %q, want test-secret", got)
		}

		var req speechRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Text != "hello speech" {
			t.Fatalf("text = %q, want hello speech", req.Text)
		}
		if req.Voice != "M1" {
			t.Fatalf("voice = %q, want M1", req.Voice)
		}
		if req.Language != "en" {
			t.Fatalf("language = %q, want en", req.Language)
		}
		if req.Speed != 1 {
			t.Fatalf("speed = %v, want 1", req.Speed)
		}
		if req.Steps != 4 {
			t.Fatalf("steps = %d, want 4", req.Steps)
		}

		w.Header().Set("Content-Type", "audio/wav")
		_, _ = w.Write([]byte(audio))
	}))
	defer server.Close()

	p := New("test-secret", server.URL)
	resp, err := p.GenerateSpeech(context.Background(), &model.SpeechRequest{
		Input:    "hello speech",
		Voice:    "M1",
		Language: "en",
		Speed:    1,
	})
	if err != nil {
		t.Fatalf("GenerateSpeech error: %v", err)
	}

	if resp.Audio != base64.StdEncoding.EncodeToString([]byte(audio)) {
		t.Fatalf("audio = %q, want base64 encoded response body", resp.Audio)
	}
	if resp.Format != "wav" {
		t.Fatalf("format = %q, want wav", resp.Format)
	}
	if resp.Characters != len("hello speech") {
		t.Fatalf("characters = %d, want %d", resp.Characters, len("hello speech"))
	}
}

func TestGenerateSpeechDefaults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req speechRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Text != "fallback text" {
			t.Fatalf("text = %q, want fallback text", req.Text)
		}
		if req.Voice != "M1" {
			t.Fatalf("voice = %q, want M1", req.Voice)
		}
		if req.Language != "en" {
			t.Fatalf("language = %q, want en", req.Language)
		}
		if req.Speed != 1 {
			t.Fatalf("speed = %v, want 1", req.Speed)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("mp3-bytes"))
	}))
	defer server.Close()

	p := New("test-secret", server.URL)
	resp, err := p.GenerateSpeech(context.Background(), &model.SpeechRequest{Text: "fallback text"})
	if err != nil {
		t.Fatalf("GenerateSpeech error: %v", err)
	}
	if resp.Format != "mp3" {
		t.Fatalf("format = %q, want mp3", resp.Format)
	}
}
