//go:build integration

package sakana

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func TestChatCompletion_FuguUltra_Integration(t *testing.T) {
	apiKey := os.Getenv("SAKANA_API_KEY")
	if apiKey == "" {
		t.Skip("SAKANA_API_KEY not set")
	}

	p := New(apiKey, "")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	resp, err := p.ChatCompletion(ctx, &model.ChatCompletionRequest{
		Model: "fugu-ultra",
		Messages: []model.ChatMessage{
			{Role: "user", Content: "Say hi in three words."},
		},
	})
	if err != nil {
		t.Fatalf("fugu-ultra ChatCompletion failed: %v", err)
	}
	if len(resp.Choices) == 0 {
		t.Fatal("no choices returned")
	}
	t.Logf("fugu-ultra reply: %s", resp.Choices[0].Message.Content)

	// Orchestration tokens must be folded into billable prompt tokens: the
	// normalized PromptTokens should equal total - completion, never the tiny
	// surface prompt_tokens.
	if resp.Usage == nil {
		t.Fatal("no usage returned")
	}
	t.Logf("billable usage: prompt=%d completion=%d total=%d cached=%d",
		resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens,
		resp.Usage.CachedPromptTokens())
	if resp.Usage.PromptTokens+resp.Usage.CompletionTokens != resp.Usage.TotalTokens {
		t.Fatalf("usage not internally consistent: %+v", resp.Usage)
	}
}
