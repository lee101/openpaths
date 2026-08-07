package billing

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestH3VideoPricingTiersAndFreeReferenceImages(t *testing.T) {
	pt := NewPricingTable([]model.ModelConfig{{
		ID:                                   "minimax-h3",
		PricePerSecond:                       0.13,
		PricePerSecondByResolution:           map[string]float64{"768p": 0.08, "2k": 0.13},
		PricePerInputVideoSecond:             0.08,
		PricePerInputVideoSecondByResolution: map[string]float64{"768p": 0.08, "2k": 0.13},
		PricePerInputImage:                   0.04,
		FreeInputImageCount:                  5,
	}})

	cost, err := pt.CalculateVideoCostWithMediaInputs("minimax-h3", 5, true, 7, "2K")
	if err != nil {
		t.Fatalf("CalculateVideoCostWithMediaInputs: %v", err)
	}
	// $0.65 output + $0.65 reference video + two $0.04 billable images.
	if cost != 13800 {
		t.Fatalf("cost = %d, want 13800", cost)
	}
}
