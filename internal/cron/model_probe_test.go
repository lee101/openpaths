package cron

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestIsChatProbeModel(t *testing.T) {
	if !IsChatProbeModel(model.ModelConfig{
		ID: "composer-2.5", Provider: "cursor",
		InputPricePer1M: 0.5, OutputPricePer1M: 2.5, ContextWindow: 256000,
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
}
