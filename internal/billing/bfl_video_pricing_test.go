package billing

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestBFLVideoPricing(t *testing.T) {
	table := NewPricingTable([]model.ModelConfig{
		{ID: "flux-3-video-draft", PricePerSecond: 0.06},
		{
			ID:                         "flux-3-video",
			PricePerSecond:             0.17,
			PricePerSecondByResolution: map[string]float64{"hd": 0.17, "fhd": 0.29},
			PricePerInputVideoSecond:   0.24,
		},
	})
	tests := []struct {
		name, modelID, resolution string
		videoInput                bool
		want                      int64
	}{
		{"draft hd", "flux-3-video-draft", "hd", false, 3000},
		{"full hd", "flux-3-video", "hd", false, 8500},
		{"full fhd", "flux-3-video", "fhd", false, 14500},
		{"video input hd", "flux-3-video", "hd", true, 20500},
		{"video input fhd", "flux-3-video", "fhd", true, 26500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := table.CalculateVideoCostWithResolution(tt.modelID, 5, tt.videoInput, tt.resolution)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("cost = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBFLFlux2ProRoundedMegapixelPricing(t *testing.T) {
	pt := NewPricingTable([]model.ModelConfig{{
		ID: "flux-2-pro-preview", PricePerMegapixel: 0.03,
		PricePerMegapixelWithImageInput: 0.045, RoundMegapixelPricing: true,
	}})
	textCost, err := pt.CalculateImageCostWithInputsAndSize("flux-2-pro-preview", 1, 0, "1024x1024")
	if err != nil || textCost != 300 {
		t.Fatalf("text cost = %d, err=%v", textCost, err)
	}
	editCost, err := pt.CalculateImageCostWithInputsAndSize("flux-2-pro-preview", 1, 2, "1920x1088")
	if err != nil || editCost != 900 {
		t.Fatalf("edit cost = %d, err=%v", editCost, err)
	}
}
