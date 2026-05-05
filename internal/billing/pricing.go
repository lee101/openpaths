package billing

import (
	"fmt"

	"github.com/openpaths/openpaths/internal/model"
)

// PricingTable holds per-model pricing loaded from config.
type PricingTable struct {
	models map[string]*model.ModelConfig
}

func NewPricingTable(models []model.ModelConfig) *PricingTable {
	pt := &PricingTable{models: make(map[string]*model.ModelConfig)}
	for i := range models {
		pt.models[models[i].ID] = &models[i]
	}
	return pt
}

// CalculateCost returns cost in hundredths-of-a-cent for a given model usage.
// $1.00 = 10000 units.
func (pt *PricingTable) CalculateCost(modelID string, inputTokens, outputTokens int) (int64, error) {
	cfg, ok := pt.models[modelID]
	if !ok {
		return 0, fmt.Errorf("unknown model %q for pricing", modelID)
	}
	if cfg.PricePerRequest > 0 {
		if inputTokens == 0 && outputTokens == 0 {
			return 0, nil
		}
		totalCents := int64(cfg.PricePerRequest * 10000)
		if totalCents < 1 {
			totalCents = 1
		}
		return totalCents, nil
	}

	inputCost := float64(inputTokens) * cfg.InputPricePer1M / 1_000_000.0
	outputCost := float64(outputTokens) * cfg.OutputPricePer1M / 1_000_000.0
	totalDollars := inputCost + outputCost

	// Convert dollars to hundredths-of-a-cent: $1 = 10000 units
	totalCents := int64(totalDollars * 10000)
	if totalCents < 1 && (inputTokens > 0 || outputTokens > 0) {
		totalCents = 1 // minimum charge
	}
	return totalCents, nil
}

// CalculateImageCost returns cost in hundredths-of-a-cent for image generation.
func (pt *PricingTable) CalculateImageCost(modelID string, imageCount int) (int64, error) {
	cfg, ok := pt.models[modelID]
	if !ok {
		return 0, fmt.Errorf("unknown model %q for pricing", modelID)
	}
	if cfg.PricePerImage <= 0 {
		return 0, fmt.Errorf("model %q has no per-image pricing", modelID)
	}
	totalDollars := cfg.PricePerImage * float64(imageCount)
	totalCents := int64(totalDollars * 10000)
	if totalCents < 1 && imageCount > 0 {
		totalCents = 1
	}
	return totalCents, nil
}

// CalculateVideoCost returns cost in hundredths-of-a-cent for video generation.
func (pt *PricingTable) CalculateVideoCost(modelID string) (int64, error) {
	cfg, ok := pt.models[modelID]
	if !ok {
		return 0, fmt.Errorf("unknown model %q for pricing", modelID)
	}
	if cfg.PricePerVideo <= 0 {
		return 0, fmt.Errorf("model %q has no per-video pricing", modelID)
	}
	totalCents := int64(cfg.PricePerVideo * 10000)
	if totalCents < 1 {
		totalCents = 1
	}
	return totalCents, nil
}

// CalculateTranscriptionCost returns cost in hundredths-of-a-cent for audio transcription.
// durationSecs is the audio duration in seconds.
func (pt *PricingTable) CalculateTranscriptionCost(modelID string, durationSecs float64) (int64, error) {
	cfg, ok := pt.models[modelID]
	if !ok {
		return 0, fmt.Errorf("unknown model %q for pricing", modelID)
	}
	if cfg.PricePerMinute <= 0 {
		return 0, fmt.Errorf("model %q has no per-minute pricing", modelID)
	}
	minutes := durationSecs / 60.0
	totalDollars := cfg.PricePerMinute * minutes
	totalCents := int64(totalDollars * 10000)
	if totalCents < 1 && durationSecs > 0 {
		totalCents = 1
	}
	return totalCents, nil
}

// LookupModel returns the ModelConfig for a given model ID, or nil if not found.
func (pt *PricingTable) LookupModel(modelID string) *model.ModelConfig {
	return pt.models[modelID]
}

// EstimateMaxCost returns a conservative cost estimate for balance pre-check.
func (pt *PricingTable) EstimateMaxCost(modelID string, maxOutputTokens int) (int64, error) {
	if maxOutputTokens <= 0 {
		maxOutputTokens = 4096
	}
	return pt.CalculateCost(modelID, 0, maxOutputTokens)
}
