//go:build integration

package xai

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func init() {
	loadEnvFile(".env")
	loadEnvFile("../../../.env")
}

func loadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range bytes.Split(data, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		parts := bytes.SplitN(line, []byte("="), 2)
		if len(parts) != 2 {
			continue
		}
		key := string(bytes.TrimSpace(parts[0]))
		val := string(bytes.TrimSpace(parts[1]))
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func liveProvider(t *testing.T) *XAIProvider {
	t.Helper()
	apiKey := os.Getenv("XAI_API_KEY")
	if apiKey == "" {
		t.Skip("XAI_API_KEY not set")
	}
	return New(apiKey, "")
}

func TestGrok43ChatCompletion_Integration(t *testing.T) {
	p := liveProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	maxTokens := 8
	temperature := 0.0
	resp, err := p.ChatCompletion(ctx, &model.ChatCompletionRequest{
		Model: "grok-4.3",
		Messages: []model.ChatMessage{
			{Role: "user", Content: "say hi and nothing else"},
		},
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
	})
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}
	if len(resp.Choices) == 0 || resp.Choices[0].Message == nil {
		t.Fatalf("no message choices returned: %#v", resp)
	}
	got := normalizeShortReply(messageContent(resp.Choices[0].Message.Content))
	if got != "hi" {
		t.Fatalf("reply = %q, want hi", resp.Choices[0].Message.Content)
	}
}

func TestGenerateSpeechAndTranscribe_Integration(t *testing.T) {
	p := liveProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	speech, err := p.GenerateSpeech(ctx, &model.SpeechRequest{
		Text:    "hi",
		VoiceID: "eve",
	})
	if err != nil {
		t.Fatalf("GenerateSpeech failed: %v", err)
	}
	audio, err := base64.StdEncoding.DecodeString(speech.Audio)
	if err != nil {
		t.Fatalf("decode audio: %v", err)
	}
	if len(audio) < 100 {
		t.Fatalf("audio length = %d, want generated audio bytes", len(audio))
	}
	if speech.Characters != 2 {
		t.Fatalf("characters = %d, want 2", speech.Characters)
	}

	text, err := p.Transcribe(ctx, &model.TranscriptionRequest{
		File:     audio,
		Filename: "xai-tts-hi.mp3",
		Language: "en",
	})
	if err != nil {
		t.Fatalf("Transcribe failed: %v", err)
	}
	got := normalizeShortReply(text.Text)
	if got != "hi" {
		t.Fatalf("transcription = %q, want hi", text.Text)
	}
}

func TestGenerateImage_Integration(t *testing.T) {
	p := liveProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	resp, err := p.GenerateImage(ctx, &model.ImageGenerationRequest{
		Model:       "grok-imagine-image",
		Prompt:      "A simple square app icon with the lowercase letters hi, clean vector style",
		N:           1,
		AspectRatio: "1:1",
	})
	if err != nil {
		t.Fatalf("GenerateImage failed: %v", err)
	}
	if len(resp.Data) == 0 {
		t.Fatal("no image data returned")
	}
	if resp.Data[0].URL == "" && resp.Data[0].B64JSON == "" {
		t.Fatalf("image response has neither url nor b64_json: %#v", resp.Data[0])
	}
	t.Logf("generated image url=%q b64=%t", resp.Data[0].URL, resp.Data[0].B64JSON != "")
}

func TestGenerateImageEdit_Integration(t *testing.T) {
	p := liveProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	resp, err := p.GenerateImage(ctx, &model.ImageGenerationRequest{
		Model:       "grok-imagine-image",
		Prompt:      "Render this as a pencil sketch with detailed shading",
		ImageURL:    "https://docs.x.ai/assets/api-examples/images/style-realistic.png",
		AspectRatio: "1:1",
	})
	if err != nil {
		t.Fatalf("GenerateImage edit failed: %v", err)
	}
	if len(resp.Data) == 0 {
		t.Fatal("no edited image data returned")
	}
	if resp.Data[0].URL == "" && resp.Data[0].B64JSON == "" {
		t.Fatalf("edit response has neither url nor b64_json: %#v", resp.Data[0])
	}
	t.Logf("edited image url=%q b64=%t", resp.Data[0].URL, resp.Data[0].B64JSON != "")
}

func normalizeShortReply(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.Trim(s, "\"'`.,! \n\t")
	return s
}

func messageContent(content any) string {
	switch v := content.(type) {
	case string:
		return v
	default:
		return ""
	}
}
