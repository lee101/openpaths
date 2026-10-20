package manifoldgen

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

const testLyrics = "I walk the line between the night and the morning light, hold on tight"

func TestGenerateLyricSong(t *testing.T) {
	var body map[string]any
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"audio_url": "https://cdn.example/song.mp3", "duration_seconds": 150}})
	})
	resp, err := p.GenerateMusic(context.Background(), &model.MusicGenerationRequest{Model: "mg-lyric-song", Prompt: "indie rock", Lyrics: testLyrics, Quality: "Lite"})
	if err != nil {
		t.Fatal(err)
	}
	if body["service"] != "lyric_music" || body["quality"] != "lite" || body["lyrics"] != testLyrics {
		t.Fatalf("body = %v", body)
	}
	if resp.Data.Audio != "https://cdn.example/song.mp3" || resp.ExtraInfo.Duration != 150 {
		t.Fatalf("resp = %+v", resp)
	}
	if _, err := p.GenerateMusic(context.Background(), &model.MusicGenerationRequest{Model: "mg-lyric-song", Lyrics: "too short"}); err == nil {
		t.Fatal("expected short lyrics rejection")
	}
	long := strings.Repeat("word ", maxLyricSongWords+1)
	if _, err := p.GenerateMusic(context.Background(), &model.MusicGenerationRequest{Model: "mg-lyric-song", Lyrics: long}); err == nil {
		t.Fatal("expected long lyrics rejection")
	}
}

func TestGenerateVoiceSongRehostsAudio(t *testing.T) {
	var put []byte
	var service map[string]any
	var p *Provider
	p, srv := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/uploads/presign":
			if r.Header.Get("Authorization") != "Bearer sk-mg-test" {
				t.Errorf("presign auth = %q", r.Header.Get("Authorization"))
			}
			json.NewEncoder(w).Encode(map[string]any{"upload_url": "http://" + r.Host + "/put/take", "public_url": "https://cdn.example/voice/take.wav"})
		case r.Method == http.MethodPut:
			put, _ = io.ReadAll(r.Body)
		default:
			json.NewDecoder(r.Body).Decode(&service)
			json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"audio_url": "https://cdn.example/full.mp3"}})
		}
	})
	_ = srv
	ctx := context.Background()
	if _, err := p.GenerateMusic(ctx, &model.MusicGenerationRequest{Model: "mg-voice-song", AudioURL: "data:audio/wav;base64,AAAA"}); err == nil {
		t.Fatal("expected consent rejection")
	}
	resp, err := p.GenerateMusic(ctx, &model.MusicGenerationRequest{Model: "mg-voice-song", AudioURL: "data:audio/wav;base64,AAAA", VoiceConsent: true, Prompt: "hard hip-hop", Duration: 400})
	if err != nil {
		t.Fatal(err)
	}
	if len(put) != 3 {
		t.Fatalf("uploaded %d bytes", len(put))
	}
	if service["service"] != "voice_enhancer" || service["audio_url"] != "https://cdn.example/voice/take.wav" || service["voice_consent"] != true || service["duration"] != float64(180) {
		t.Fatalf("service = %v", service)
	}
	if resp.Data.Audio != "https://cdn.example/full.mp3" {
		t.Fatalf("resp = %+v", resp)
	}
	if _, err := p.GenerateMusic(ctx, &model.MusicGenerationRequest{Model: "mg-voice-song", AudioURL: "http://10.0.0.1/a.mp3", VoiceConsent: true}); err == nil {
		t.Fatal("expected non-https rejection")
	}
}

func TestMgCaptionChat(t *testing.T) {
	var body map[string]any
	p, _ := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"caption": "a red fox in snow"}})
	})
	req := &model.ChatCompletionRequest{Model: "mg-caption", Messages: []model.ChatMessage{{Role: "user", Content: []any{
		map[string]any{"type": "text", "text": "caption this"},
		map[string]any{"type": "image_url", "image_url": map[string]any{"url": "https://img.example/fox.jpg"}},
	}}}}
	resp, err := p.ChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if body["service"] != "caption" || body["image_url"] != "https://img.example/fox.jpg" {
		t.Fatalf("body = %v", body)
	}
	if resp.Choices[0].Message.Content != "a red fox in snow" {
		t.Fatalf("resp = %+v", resp.Choices[0].Message)
	}
	req.Messages = []model.ChatMessage{{Role: "user", Content: "hi"}}
	if _, err := p.ChatCompletion(context.Background(), req); err == nil {
		t.Fatal("expected missing image error")
	}
}
