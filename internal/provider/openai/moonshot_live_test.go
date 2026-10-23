package openai

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestMoonshotLiveSayHi(t *testing.T) {
	key := os.Getenv("KIMI_API_KEY")
	if key == "" {
		t.Skip("KIMI_API_KEY not set")
	}
	p := NewCompatible(MoonshotProvider, key, "https://api.moonshot.ai", SanitizeMoonshot)
	temp := 0.7
	for _, m := range []string{"kimi-k3.1-pro", "kimi-k3"} {
		resp, err := p.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
			Model:           m,
			Temperature:     &temp,
			ReasoningEffort: "minimal",
			Messages:        []model.ChatMessage{{Role: "user", Content: "say hi nothing else"}},
		})
		if err != nil {
			t.Fatalf("%s: %v", m, err)
		}
		got, _ := resp.Choices[0].Message.Content.(string)
		if !strings.Contains(strings.ToLower(got), "hi") || len(strings.TrimSpace(got)) > 8 {
			t.Fatalf("%s: unexpected reply %q", m, got)
		}
		t.Logf("%s -> %q usage=%+v", m, got, resp.Usage)
	}
}
