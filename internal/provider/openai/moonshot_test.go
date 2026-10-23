package openai

import (
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func msgs(roles ...string) []model.ChatMessage {
	out := make([]model.ChatMessage, len(roles))
	for i, r := range roles {
		out[i] = model.ChatMessage{Role: r, Content: "x"}
	}
	return out
}

func TestSanitizeMoonshot(t *testing.T) {
	temp := 0.7
	req := &model.ChatCompletionRequest{Temperature: &temp, ReasoningEffort: "xhigh", Messages: msgs("user")}
	SanitizeMoonshot(req)
	if req.Temperature != nil || req.ReasoningEffort != "max" || req.PromptCacheOptions != nil {
		t.Fatalf("unexpected %+v", req)
	}
	for in, want := range map[string]string{"": "", "auto": "", "none": "low", "medium": "high", "high": "high", "max": "max"} {
		if got := moonshotEffort(in); got != want {
			t.Errorf("effort %q = %q want %q", in, got, want)
		}
	}
}

func TestMoonshotCacheTTL(t *testing.T) {
	big := strings.Repeat("a", 80000)
	chat := msgs("system", "user", "assistant", "user")
	chat[0].Content = big
	req := &model.ChatCompletionRequest{Messages: chat}
	if o := moonshotCacheOptions(req); o == nil || o.TTL != "1h" {
		t.Fatalf("big human-paced chat should use 1h, got %+v", o)
	}
	agent := msgs("system", "user", "assistant", "tool")
	agent[0].Content = big
	if o := moonshotCacheOptions(&model.ChatCompletionRequest{Messages: agent}); o != nil {
		t.Fatalf("tool loop should default to 5m, got %+v", o)
	}
	if o := moonshotCacheOptions(&model.ChatCompletionRequest{Messages: msgs("user", "assistant", "user")}); o != nil {
		t.Fatalf("small prefix should default, got %+v", o)
	}
	explicit := &model.ChatCompletionRequest{Messages: msgs("user"), PromptCacheOptions: &model.PromptCacheOptions{TTL: "1h"}}
	if o := moonshotCacheOptions(explicit); o == nil || o.TTL != "1h" || o.Mode != "implicit" {
		t.Fatalf("explicit ttl not honored: %+v", o)
	}
}
