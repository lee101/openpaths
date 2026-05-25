package billing

import (
	"fmt"
	"math"

	imgutil "github.com/openpaths/openpaths/internal/image"
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
		for _, alias := range models[i].Aliases {
			pt.models[alias] = &models[i]
		}
	}
	return pt
}

// CalculateCost returns cost in hundredths-of-a-cent for a given model usage.
// $1.00 = 10000 units.
func (pt *PricingTable) CalculateCost(modelID string, inputTokens, outputTokens int) (int64, error) {
	return pt.CalculateCostWithCachedInput(modelID, inputTokens, outputTokens, 0)
}

// CalculateCostWithCachedInput returns cost in hundredths-of-a-cent, charging
// cache-hit prompt tokens at the model's cache-hit input rate when configured.
func (pt *PricingTable) CalculateCostWithCachedInput(modelID string, inputTokens, outputTokens, cachedInputTokens int) (int64, error) {
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
	if cfg.InputPricePer1M == 0 && cfg.OutputPricePer1M == 0 {
		return 0, nil
	}

	if cachedInputTokens < 0 {
		cachedInputTokens = 0
	}
	if cachedInputTokens > inputTokens {
		cachedInputTokens = inputTokens
	}
	if cfg.InputCacheHitPricePer1M <= 0 {
		cachedInputTokens = 0
	}

	uncachedInputTokens := inputTokens - cachedInputTokens
	inputCost := float64(uncachedInputTokens)*cfg.InputPricePer1M/1_000_000.0 +
		float64(cachedInputTokens)*cfg.InputCacheHitPricePer1M/1_000_000.0
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
	return pt.CalculateImageCostWithInputs(modelID, imageCount, 0)
}

func (pt *PricingTable) CalculateImageCostWithInputs(modelID string, outputImageCount, inputImageCount int) (int64, error) {
	return pt.CalculateImageCostWithInputsAndSize(modelID, outputImageCount, inputImageCount, "")
}

func (pt *PricingTable) CalculateImageCostWithInputsAndSize(modelID string, outputImageCount, inputImageCount int, size string) (int64, error) {
	cfg, ok := pt.models[modelID]
	if !ok {
		return 0, fmt.Errorf("unknown model %q for pricing", modelID)
	}
	if cfg.PricePerMegapixel > 0 {
		outputSize := imgutil.Size{W: 512, H: 512}
		if parsed, ok := imgutil.ParseSize(size); ok {
			outputSize = parsed
		}
		megapixels := float64(outputSize.W*outputSize.H) / 1_000_000.0
		totalDollars := cfg.PricePerMegapixel*megapixels*float64(outputImageCount) + cfg.PricePerInputImage*float64(inputImageCount)
		totalCents := int64(totalDollars * 10000)
		if totalCents < 1 && (outputImageCount > 0 || inputImageCount > 0) {
			totalCents = 1
		}
		return totalCents, nil
	}
	if cfg.PricePerImage <= 0 {
		return 0, fmt.Errorf("model %q has no per-image pricing", modelID)
	}
	totalDollars := cfg.PricePerImage*float64(outputImageCount) + cfg.PricePerInputImage*float64(inputImageCount)
	totalCents := int64(totalDollars * 10000)
	if totalCents < 1 && (outputImageCount > 0 || inputImageCount > 0) {
		totalCents = 1
	}
	return totalCents, nil
}

func (pt *PricingTable) CalculateOutpaintCost(modelID string, inputWidth, inputHeight, outputWidth, outputHeight, outputImageCount int) (int64, error) {
	cfg, ok := pt.models[modelID]
	if !ok {
		return 0, fmt.Errorf("unknown model %q for pricing", modelID)
	}
	if cfg.PriceFirstMegapixel <= 0 || cfg.PriceExtraMegapixel <= 0 {
		return 0, fmt.Errorf("model %q has no outpaint pricing", modelID)
	}
	if outputImageCount <= 0 {
		outputImageCount = 1
	}
	inputMP := roundedMegapixels(inputWidth, inputHeight)
	outputMP := roundedMegapixels(outputWidth, outputHeight)
	if outputMP < 1 {
		outputMP = 1
	}
	extraMP := inputMP + max(0, outputMP-1)
	totalDollars := (cfg.PriceFirstMegapixel + cfg.PriceExtraMegapixel*float64(extraMP)) * float64(outputImageCount)
	totalCents := int64(totalDollars * 10000)
	if totalCents < 1 {
		totalCents = 1
	}
	return totalCents, nil
}

func roundedMegapixels(width, height int) int {
	if width <= 0 || height <= 0 {
		return 0
	}
	mp := float64(width*height) / 1_000_000.0
	rounded := int(math.Round(mp))
	if rounded < 1 {
		return 1
	}
	return rounded
}

// CalculateVideoCost returns cost in hundredths-of-a-cent for video generation.
func (pt *PricingTable) CalculateVideoCost(modelID string, durationSeconds int, hasVideoInput bool) (int64, error) {
	cfg, ok := pt.models[modelID]
	if !ok {
		return 0, fmt.Errorf("unknown model %q for pricing", modelID)
	}
	var totalDollars float64
	switch {
	case hasVideoInput && cfg.PricePerSecondWithVideoInput > 0:
		if durationSeconds <= 0 {
			durationSeconds = 10
		}
		totalDollars = cfg.PricePerSecondWithVideoInput * float64(durationSeconds)
	case cfg.PricePerSecond > 0:
		if durationSeconds <= 0 {
			durationSeconds = 10
		}
		totalDollars = cfg.PricePerSecond * float64(durationSeconds)
	case cfg.PricePerVideo > 0:
		totalDollars = cfg.PricePerVideo
	default:
		return 0, fmt.Errorf("model %q has no per-video pricing", modelID)
	}
	totalCents := int64(totalDollars * 10000)
	if totalCents < 1 {
		totalCents = 1
	}
	return totalCents, nil
}

// CalculateAudioCost returns cost in hundredths-of-a-cent for audio usage.
func (pt *PricingTable) CalculateAudioCost(modelID string, durationSeconds int) (int64, error) {
	cfg, ok := pt.models[modelID]
	if !ok {
		return 0, fmt.Errorf("unknown model %q for pricing", modelID)
	}
	if durationSeconds <= 0 {
		return 0, nil
	}

	var totalDollars float64
	switch {
	case cfg.PricePerMinute > 0:
		totalDollars = cfg.PricePerMinute * float64(durationSeconds) / 60.0
	case cfg.PricePerHour > 0:
		totalDollars = cfg.PricePerHour * float64(durationSeconds) / 3600.0
	default:
		return 0, fmt.Errorf("model %q has no per-audio pricing", modelID)
	}

	totalCents := int64(totalDollars * 10000)
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
	cfg, ok := pt.models[modelID]
	if !ok {
		return 0, fmt.Errorf("unknown model %q for pricing", modelID)
	}
	if cfg.PricePerRequest > 0 {
		totalCents := int64(cfg.PricePerRequest * 10000)
		if totalCents < 1 {
			totalCents = 1
		}
		return totalCents, nil
	}
	if cfg.PricePerMegapixel > 0 {
		return pt.CalculateImageCostWithInputsAndSize(modelID, 1, 0, "2048x2048")
	}
	if cfg.PriceFirstMegapixel > 0 && cfg.PriceExtraMegapixel > 0 {
		return pt.CalculateOutpaintCost(modelID, 1024, 1024, 2048, 2048, 1)
	}
	if cfg.PricePerImage > 0 {
		return pt.CalculateImageCostWithInputs(modelID, 1, 0)
	}
	if cfg.PricePerVideo > 0 || cfg.PricePerSecond > 0 {
		return pt.CalculateVideoCost(modelID, 10, false)
	}
	if cfg.PricePerMinute > 0 || cfg.PricePerHour > 0 {
		return pt.CalculateAudioCost(modelID, 60)
	}
	if maxOutputTokens <= 0 {
		maxOutputTokens = 4096
	}
	return pt.CalculateCost(modelID, 0, maxOutputTokens)
}
