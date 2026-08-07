//go:build integration

package minimax

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

// TestChatCompletion_M3Series_Integration verifies the MiniMax M3-series chat
// models are reachable through the OpenAI-compatible endpoint and can reply.
// These are reasoning models that emit <think> blocks, so we give them a
// generous max_tokens budget to ensure a non-reasoning answer survives.
func TestChatCompletion_M3Series_Integration(t *testing.T) {
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		t.Skip("MINIMAX_API_KEY not set")
	}

	models := []string{
		"MiniMax-M3",
		"MiniMax-M2.7",
		"MiniMax-M2.7-highspeed",
	}

	p := New(apiKey)

	for _, m := range models {
		m := m
		t.Run(m, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			maxTok := 1024
			resp, err := p.ChatCompletion(ctx, &model.ChatCompletionRequest{
				Model: m,
				Messages: []model.ChatMessage{
					{Role: "user", Content: "Say hi back in one word."},
				},
				MaxTokens: &maxTok,
			})
			if err != nil {
				t.Fatalf("%s ChatCompletion failed: %v", m, err)
			}
			if len(resp.Choices) == 0 {
				t.Fatalf("%s: no choices returned", m)
			}
			raw := fmt.Sprintf("%v", resp.Choices[0].Message.Content)
			content := strings.ToLower(raw)
			t.Logf("%s reply: %s", m, raw)
			if !strings.Contains(content, "hi") && !strings.Contains(content, "hello") {
				t.Errorf("%s: expected a greeting in reply, got %q", m, raw)
			}
		})
	}
}
