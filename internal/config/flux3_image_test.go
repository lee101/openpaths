package config

import (
	"github.com/openpaths/openpaths/internal/billing"
	"testing"
)

func TestFlux3ImageResolutionPricing(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	pt := billing.NewPricingTable(cfg.Models)
	for _, tc := range []struct {
		resolution string
		want       int64
	}{{"768sq", 410}, {"1k", 480}, {"2k", 1000}, {"4k", 6070}} {
		got, err := pt.CalculateImageCostWithInputsAndSize("flux-3-image", 1, 10, tc.resolution)
		if err != nil || got != tc.want {
			t.Fatalf("%s: got %d want %d err %v", tc.resolution, got, tc.want, err)
		}
	}
}
