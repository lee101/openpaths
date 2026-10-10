package config

import (
	"testing"

	"github.com/openpaths/openpaths/internal/billing"
)

func TestGeminiImageModels(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	pt := billing.NewPricingTable(cfg.Models)
	for _, id := range []string{"gemini-3.1-flash-lite-image", "gemini-nano-banana-2.1"} {
		got, err := pt.CalculateImageCostWithInputsAndSize(id, 1, 0, "")
		if err != nil || got != 336 {
			t.Fatalf("%s: got %d err %v", id, got, err)
		}
	}
	byID := map[string]bool{}
	for _, m := range cfg.Models {
		byID[m.ID] = true
		if m.ID == "gemini-3-flash-preview" && (m.InputPricePer1M != 0.50 || m.OutputPricePer1M != 3.00) {
			t.Fatalf("flash preview price %v/%v", m.InputPricePer1M, m.OutputPricePer1M)
		}
		if m.ID == "openpaths/image-edit" && (len(m.FallbackModels) == 0 || m.FallbackModels[0] != "gemini-3.1-flash-lite-image") {
			t.Fatalf("image-edit fallbacks %v", m.FallbackModels)
		}
	}
	for _, id := range []string{"gemini-3-flash-preview", "gemini-3.1-flash-lite", "gemini-3.1-flash-lite-image", "gemini-nano-banana-2.1"} {
		if !byID[id] {
			t.Fatalf("missing %s", id)
		}
	}
}
