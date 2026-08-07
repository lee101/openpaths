//go:build integration

package zai

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func TestChatCompletion_GLM52_StandardIntegration(t *testing.T) {
	apiKey := os.Getenv("Z_API_KEY")
	if apiKey == "" {
		t.Skip("Z_API_KEY not set")
	}

	p := New(apiKey, "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	maxTok := 256
	resp, err := p.ChatCompletion(ctx, &model.ChatCompletionRequest{
		Model: "glm-5.2",
		Messages: []model.ChatMessage{
			{Role: "user", Content: "Reply with exactly the word: pong"},
		},
		MaxTokens: &maxTok,
	})
	if err != nil {
		t.Fatalf("glm-5.2 ChatCompletion failed: %v", err)
	}
	if len(resp.Choices) == 0 {
		t.Fatal("no choices returned")
	}
	t.Logf("glm-5.2 reply: %s", resp.Choices[0].Message.Content)
}

func TestGenerateImage_Integration(t *testing.T) {
	apiKey := os.Getenv("Z_API_KEY")
	if apiKey == "" {
		t.Skip("Z_API_KEY not set")
	}

	p := New(apiKey, "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	resp, err := p.GenerateImage(ctx, &model.ImageGenerationRequest{
		Model:  "glm-image",
		Prompt: "A cute little kitten sitting on a sunny windowsill, with blue sky and white clouds in the background.",
		Size:   "1280x1280",
	})
	if err != nil {
		t.Fatalf("GenerateImage failed: %v", err)
	}
	if len(resp.Data) == 0 {
		t.Fatal("no images returned")
	}

	outDir := filepath.Join("..", "..", "..", "test-results")
	os.MkdirAll(outDir, 0755)

	for i, img := range resp.Data {
		if img.URL == "" {
			t.Errorf("image %d: empty URL", i)
			continue
		}
		t.Logf("image %d URL: %s", i, img.URL)

		imgResp, err := http.Get(img.URL)
		if err != nil {
			t.Errorf("image %d: download failed: %v", i, err)
			continue
		}
		defer imgResp.Body.Close()

		outPath := filepath.Join(outDir, "glm-image-test.png")
		f, err := os.Create(outPath)
		if err != nil {
			t.Errorf("image %d: create file: %v", i, err)
			continue
		}
		n, err := io.Copy(f, imgResp.Body)
		f.Close()
		if err != nil {
			t.Errorf("image %d: write: %v", i, err)
			continue
		}
		t.Logf("image %d saved to %s (%d bytes)", i, outPath, n)
	}
}
