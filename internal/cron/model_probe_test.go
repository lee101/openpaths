package cron

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestIsChatProbeModel(t *testing.T) {
	if !IsChatProbeModel(model.ModelConfig{
		ID: "composer-2.5", Provider: "cursor",
		InputPricePer1M: 0.5, OutputPricePer1M: 2.5, ContextWindow: 256000, MaxOutputTokens: 128000,
	}) {
		t.Fatal("expected composer-2.5 to be probeable")
	}
	if IsChatProbeModel(model.ModelConfig{
		ID: "zimage", Provider: "netwrck", PricePerImage: 0.007,
	}) {
		t.Fatal("image model should not be probeable")
	}
	if IsChatProbeModel(model.ModelConfig{
		ID: "auto-video", Provider: "fal", PricePerVideo: 0.1, InputPricePer1M: 1,
	}) {
		t.Fatal("video model should not be probeable")
	}
	// Embedding models produce no generated output (output price 0, max output 0).
	if IsChatProbeModel(model.ModelConfig{
		ID: "text-embedding", Provider: "textgenerator",
		InputPricePer1M: 0.1, OutputPricePer1M: 0, ContextWindow: 8192, MaxOutputTokens: 0,
	}) {
		t.Fatal("embedding model should not be probeable")
	}
	// Codex models are only served on /v1/responses, not /v1/chat/completions.
	if IsChatProbeModel(model.ModelConfig{
		ID: "gpt-5-codex", Provider: "openai",
		InputPricePer1M: 1.25, OutputPricePer1M: 10, ContextWindow: 400000, MaxOutputTokens: 128000,
	}) {
		t.Fatal("codex model should not be probeable")
	}
}
