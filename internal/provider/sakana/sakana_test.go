package sakana

import "testing"

func TestNormalizeUsage_FoldsOrchestration(t *testing.T) {
	u := &sakanaUsage{PromptTokens: 17, CompletionTokens: 172, TotalTokens: 1449}
	u.PromptTokensDetails.OrchestrationInputTokens = 1260
	u.PromptTokensDetails.OrchestrationInputCachedTokens = 100
	u.PromptTokensDetails.CachedTokens = 5
	u.CompletionTokensDetails.OrchestrationOutputTokens = 30

	got := u.normalize()
	if got.PromptTokens != 1277 {
		t.Errorf("PromptTokens = %d, want 1277", got.PromptTokens)
	}
	if got.CompletionTokens != 202 {
		t.Errorf("CompletionTokens = %d, want 202", got.CompletionTokens)
	}
	if got.TotalTokens != 1479 {
		t.Errorf("TotalTokens = %d, want 1479", got.TotalTokens)
	}
	if got.CachedPromptTokens() != 105 {
		t.Errorf("CachedPromptTokens = %d, want 105", got.CachedPromptTokens())
	}
}
