package config

import (
	"testing"

	"github.com/openpaths/openpaths/internal/billing"
)

func TestGPTImageGenerationModels(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	pt := billing.NewPricingTable(cfg.Models)
	byID := map[string]string{}
	for _, m := range cfg.Models {
		byID[m.ID] = m.Provider
	}
	for _, id := range []string{"gpt-image-2.6-sunburst", "gpt-image-2.6-flare", "gpt-image-2.5-sunburst", "gpt-image-2.5-flare"} {
		if byID[id] != "openai" {
			t.Fatalf("%s: provider %q", id, byID[id])
		}
		for _, c := range []struct {
			size   string
			inputs int
			want   int64
		}{
			{"1024x1024", 0, 600},
			{"1536x1024", 0, 600},
			{"2048x2048", 0, 1100},
			{"3840x2160", 0, 1100},
			{"1024x1024", 1, 700},
		} {
			got, err := pt.CalculateImageCostWithInputsAndSize(id, 1, c.inputs, c.size)
			if err != nil || got < c.want-1 || got > c.want {
				t.Fatalf("%s %s inputs=%d: got %d err %v want %d", id, c.size, c.inputs, got, err, c.want)
			}
		}
	}
}
