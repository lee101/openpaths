//go:build integration

package seedream

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func live(t *testing.T) *Provider {
	key := os.Getenv("SEEDREAM_API_KEY")
	if key == "" {
		t.Skip("SEEDREAM_API_KEY not set")
	}
	return New(key, "")
}

func TestLiveChat(t *testing.T) {
	p := live(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, m := range []string{"deepseek-v4-1-flash-261017", "deepseek-v4-1-flash-economy-261017", "dola-seed-2-1-turbo-260628"} {
		resp, err := p.ChatCompletion(ctx, &model.ChatCompletionRequest{
			Model: m, Messages: []model.ChatMessage{{Role: "user", Content: "say hi nothing else"}}, ReasoningEffort: "none",
		})
		if err != nil || len(resp.Choices) == 0 {
			t.Fatalf("%s: resp=%v err=%v", m, resp, err)
		}
		t.Logf("%s -> %v", m, resp.Choices[0].Message.Content)
	}
}

func TestLiveImage(t *testing.T) {
	p := live(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	for _, m := range []string{"dola-seedream-5-0-flash-260915", "dola-seedream-5-0-pro-260628"} {
		resp, err := p.GenerateImage(ctx, &model.ImageGenerationRequest{Model: m, Prompt: "a red fox in the snow", Size: "2K"})
		if err != nil || len(resp.Data) == 0 || len(resp.Data[0].B64JSON) < 1000 {
			t.Fatalf("%s: err=%v resp=%v", m, err, resp)
		}
		t.Logf("%s b64 len %d", m, len(resp.Data[0].B64JSON))
	}
}

func TestLiveVideo(t *testing.T) {
	p := live(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	resp, err := p.GenerateVideo(ctx, &model.VideoGenerationRequest{
		Model: "dreamina-seedance-2-5-260628", Prompt: "a red fox trots through snow", Duration: "4", Resolution: "480p", Ratio: "16:9",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp.VideoURL)
}
