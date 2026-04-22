package google

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestTranslateRequest_PrefillAppendsModelTurn(t *testing.T) {
	req := &model.ChatCompletionRequest{
		Model:    "gemini-3-pro-preview",
		Messages: []model.ChatMessage{{Role: "user", Content: "Return JSON"}},
		Prefill:  "{",
	}

	gemReq := translateRequest(req)
	if len(gemReq.Contents) != 2 {
		t.Fatalf("got %d contents, want 2 (user + model prefill)", len(gemReq.Contents))
	}
	last := gemReq.Contents[len(gemReq.Contents)-1]
	if last.Role != "model" {
		t.Errorf("last role = %q, want model", last.Role)
	}
	if len(last.Parts) != 1 || last.Parts[0].Text != "{" {
		t.Errorf("last parts = %+v, want text={", last.Parts)
	}
}

func TestTranslateRequest_ClampsThinkingBudgetToMaxOutput(t *testing.T) {
	maxTokens := 128
	req := &model.ChatCompletionRequest{
		Model:           "gemini-2.5-flash",
		ReasoningEffort: "high",
		MaxTokens:       &maxTokens,
		Messages:        []model.ChatMessage{{Role: "user", Content: "Solve this carefully."}},
	}

	gemReq := translateRequest(req)
	if gemReq.GenerationConfig == nil || gemReq.GenerationConfig.ThinkingConfig == nil || gemReq.GenerationConfig.ThinkingConfig.ThinkingBudget == nil {
		t.Fatal("expected thinking config to be set")
	}
	if got := *gemReq.GenerationConfig.ThinkingConfig.ThinkingBudget; got != 128 {
		t.Fatalf("thinking budget = %d, want %d", got, 128)
	}
}

func TestTranslateUsage_CountsThoughtTokensAsBillableOutput(t *testing.T) {
	usage := translateUsage(&geminiUsage{
		PromptTokenCount:     100,
		CandidatesTokenCount: 40,
		ThoughtsTokenCount:   60,
		TotalTokenCount:      200,
	})

	if usage.PromptTokens != 100 {
		t.Fatalf("PromptTokens = %d, want %d", usage.PromptTokens, 100)
	}
	if usage.CompletionTokens != 100 {
		t.Fatalf("CompletionTokens = %d, want %d", usage.CompletionTokens, 100)
	}
	if usage.TotalTokens != 200 {
		t.Fatalf("TotalTokens = %d, want %d", usage.TotalTokens, 200)
	}
}
