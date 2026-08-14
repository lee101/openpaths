package billing

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	imgutil "github.com/openpaths/openpaths/internal/image"
	"github.com/openpaths/openpaths/internal/model"
)

// PricingTable holds per-model pricing loaded from config.
type PricingTable struct {
	models map[string]*model.ModelConfig
	now    func() time.Time
}

type RealtimeUsage struct {
	TextInputTokens        int
	CachedTextInputTokens  int
	TextOutputTokens       int
	AudioInputTokens       int
	CachedAudioInputTokens int
	AudioOutputTokens      int
	ImageInputTokens       int
	CachedImageInputTokens int
}

func NewPricingTable(models []model.ModelConfig) *PricingTable {
	pt := &PricingTable{models: make(map[string]*model.ModelConfig), now: time.Now}
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
	// Long-context tiered pricing: above the threshold the whole request bills at
	// the higher *Long rates.
	inputRate, cacheRate, outputRate := pt.tokenRates(cfg)
	if cfg.LongContextThreshold > 0 && inputTokens > cfg.LongContextThreshold {
		if cfg.InputPricePer1MLong > 0 {
			inputRate = cfg.InputPricePer1MLong
		}
		if cfg.InputCacheHitPricePer1MLong > 0 {
			cacheRate = cfg.InputCacheHitPricePer1MLong
		}
		if cfg.OutputPricePer1MLong > 0 {
			outputRate = cfg.OutputPricePer1MLong
		}
	}

	if cacheRate <= 0 {
		cachedInputTokens = 0
	}

	uncachedInputTokens := inputTokens - cachedInputTokens
	inputCost := float64(uncachedInputTokens)*inputRate/1_000_000.0 +
		float64(cachedInputTokens)*cacheRate/1_000_000.0
	outputCost := float64(outputTokens) * outputRate / 1_000_000.0
	totalDollars := inputCost + outputCost

	// Convert dollars to hundredths-of-a-cent: $1 = 10000 units
	totalCents := int64(totalDollars * 10000)
	if totalCents < 1 && (inputTokens > 0 || outputTokens > 0) {
		totalCents = 1 // minimum charge
	}
	return totalCents, nil
}

func (pt *PricingTable) tokenRates(cfg *model.ModelConfig) (float64, float64, float64) {
	inputRate, cacheRate, outputRate := cfg.InputPricePer1M, cfg.InputCacheHitPricePer1M, cfg.OutputPricePer1M
	schedule := cfg.ScheduledTokenPricing
	if schedule == nil {
		return inputRate, cacheRate, outputRate
	}
	nowUTC := pt.now().UTC()
	effectiveAt, err := time.Parse(time.RFC3339, schedule.EffectiveAt)
	if err != nil || nowUTC.Before(effectiveAt) {
		return inputRate, cacheRate, outputRate
	}
	minute := nowUTC.Hour()*60 + nowUTC.Minute()
	peak := false
	for _, window := range schedule.PeakWindowsUTC {
		start, startOK := utcMinute(window.Start)
		end, endOK := utcMinute(window.End)
		if !startOK || !endOK {
			continue
		}
		if (start <= end && minute >= start && minute < end) ||
			(start > end && (minute >= start || minute < end)) {
			peak = true
			break
		}
	}
	if peak {
		return schedule.PeakInputPricePer1M, schedule.PeakInputCacheHitPricePer1M, schedule.PeakOutputPricePer1M
	}
	return schedule.OffPeakInputPricePer1M, schedule.OffPeakInputCacheHitPer1M, schedule.OffPeakOutputPricePer1M
}

func utcMinute(value string) (int, bool) {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, false
	}
	hour, hourErr := strconv.Atoi(parts[0])
	minute, minuteErr := strconv.Atoi(parts[1])
	if hourErr != nil || minuteErr != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

func (pt *PricingTable) CalculateRealtimeCost(modelID string, usage RealtimeUsage) (int64, error) {
	cfg, ok := pt.models[modelID]
	if !ok {
		return 0, fmt.Errorf("unknown model %q for pricing", modelID)
	}

	usage.TextInputTokens = max(0, usage.TextInputTokens)
	usage.CachedTextInputTokens = min(max(0, usage.CachedTextInputTokens), usage.TextInputTokens)
	usage.TextOutputTokens = max(0, usage.TextOutputTokens)
	usage.AudioInputTokens = max(0, usage.AudioInputTokens)
	usage.CachedAudioInputTokens = min(max(0, usage.CachedAudioInputTokens), usage.AudioInputTokens)
	usage.AudioOutputTokens = max(0, usage.AudioOutputTokens)
	usage.ImageInputTokens = max(0, usage.ImageInputTokens)
	usage.CachedImageInputTokens = min(max(0, usage.CachedImageInputTokens), usage.ImageInputTokens)

	dollarsPerMillion :=
		float64(usage.TextInputTokens-usage.CachedTextInputTokens)*cfg.InputPricePer1M +
			float64(usage.CachedTextInputTokens)*cfg.InputCacheHitPricePer1M +
			float64(usage.TextOutputTokens)*cfg.OutputPricePer1M +
			float64(usage.AudioInputTokens-usage.CachedAudioInputTokens)*cfg.AudioInputPricePer1M +
			float64(usage.CachedAudioInputTokens)*cfg.AudioInputCacheHitPricePer1M +
			float64(usage.AudioOutputTokens)*cfg.AudioOutputPricePer1M +
			float64(usage.ImageInputTokens-usage.CachedImageInputTokens)*cfg.ImageInputPricePer1M +
			float64(usage.CachedImageInputTokens)*cfg.ImageInputCacheHitPricePer1M

	totalTokens := usage.TextInputTokens + usage.TextOutputTokens + usage.AudioInputTokens +
		usage.AudioOutputTokens + usage.ImageInputTokens
	if dollarsPerMillion == 0 || totalTokens == 0 {
		return 0, nil
	}
	cost := int64(dollarsPerMillion / 1_000_000 * 10000)
	if cost < 1 {
		cost = 1
	}
	return cost, nil
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
		if cfg.RoundMegapixelPricing {
			megapixels = math.Ceil(float64(outputSize.W*outputSize.H) / float64(1024*1024))
		}
		outputRate := cfg.PricePerMegapixel
		if inputImageCount > 0 && cfg.PricePerMegapixelWithImageInput > 0 {
			outputRate = cfg.PricePerMegapixelWithImageInput
		}
		totalDollars := outputRate*megapixels*float64(outputImageCount) + cfg.PricePerInputImage*float64(inputImageCount)
		totalCents := int64(totalDollars * 10000)
		if totalCents < 1 && (outputImageCount > 0 || inputImageCount > 0) {
			totalCents = 1
		}
		return totalCents, nil
	}
	pricePerImage := cfg.PricePerImage
	if tierPrice, ok := resolutionPrice(cfg.PricePerImageByResolution, size); ok {
		pricePerImage = tierPrice
	}
	if pricePerImage <= 0 {
		return 0, fmt.Errorf("model %q has no per-image pricing", modelID)
	}
	totalDollars := pricePerImage*float64(outputImageCount) + cfg.PricePerInputImage*float64(inputImageCount)
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
	return pt.CalculateVideoCostWithResolution(modelID, durationSeconds, hasVideoInput, "")
}

// CalculateVideoCostWithResolution applies the output resolution tier and then
// adds any separately published video-input rate. The legacy combined
// PricePerSecondWithVideoInput field remains supported for existing models.
func (pt *PricingTable) CalculateVideoCostWithResolution(modelID string, durationSeconds int, hasVideoInput bool, resolution string) (int64, error) {
	return pt.CalculateVideoCostWithMediaInputs(modelID, durationSeconds, hasVideoInput, 0, resolution)
}

func (pt *PricingTable) CalculateVideoCostWithMediaInputs(modelID string, durationSeconds int, hasVideoInput bool, inputImageCount int, resolution string) (int64, error) {
	cfg, ok := pt.models[modelID]
	if !ok {
		return 0, fmt.Errorf("unknown model %q for pricing", modelID)
	}
	if durationSeconds <= 0 {
		durationSeconds = 10
	}
	outputRate := cfg.PricePerSecond
	if tierPrice, ok := resolutionPrice(cfg.PricePerSecondByResolution, resolution); ok {
		outputRate = tierPrice
	}
	inputVideoRate := cfg.PricePerInputVideoSecond
	if tierPrice, ok := resolutionPrice(cfg.PricePerInputVideoSecondByResolution, resolution); ok {
		inputVideoRate = tierPrice
	}
	var totalDollars float64
	switch {
	case outputRate > 0:
		totalDollars = outputRate * float64(durationSeconds)
		if hasVideoInput && inputVideoRate > 0 {
			totalDollars += inputVideoRate * float64(durationSeconds)
		} else if hasVideoInput && cfg.PricePerSecondWithVideoInput > 0 {
			totalDollars = cfg.PricePerSecondWithVideoInput * float64(durationSeconds)
		}
	case hasVideoInput && cfg.PricePerSecondWithVideoInput > 0:
		totalDollars = cfg.PricePerSecondWithVideoInput * float64(durationSeconds)
	case cfg.PricePerVideo > 0:
		totalDollars = cfg.PricePerVideo
	default:
		return 0, fmt.Errorf("model %q has no per-video pricing", modelID)
	}
	billableImages := max(0, inputImageCount-cfg.FreeInputImageCount)
	if billableImages > 0 && cfg.PricePerInputImage > 0 {
		totalDollars += cfg.PricePerInputImage * float64(billableImages)
	}
	totalCents := int64(totalDollars * 10000)
	if totalCents < 1 {
		totalCents = 1
	}
	return totalCents, nil
}

func resolutionPrice(prices map[string]float64, resolution string) (float64, bool) {
	if len(prices) == 0 || resolution == "" {
		return 0, false
	}
	key := strings.ToLower(strings.TrimSpace(resolution))
	price, ok := prices[key]
	return price, ok
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

// CalculateCharacterCost bills APIs such as xAI TTS that publish prices per
// Unicode input character rather than per language-model token.
func (pt *PricingTable) CalculateCharacterCost(modelID string, characters int) (int64, error) {
	cfg, ok := pt.models[modelID]
	if !ok {
		return 0, fmt.Errorf("unknown model %q for pricing", modelID)
	}
	if cfg.PricePer1MCharacters <= 0 {
		return 0, fmt.Errorf("model %q has no per-character pricing", modelID)
	}
	if characters <= 0 {
		return 0, nil
	}
	cost := int64(float64(characters) * cfg.PricePer1MCharacters / 1_000_000 * 10000)
	if cost < 1 {
		cost = 1
	}
	return cost, nil
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

// CalculateForecastCost returns cost in hundredths-of-a-cent for a single
// time-series forecast request. It prefers an explicit per-forecast price and
// falls back to the generic per-request price.
func (pt *PricingTable) CalculateForecastCost(modelID string) (int64, error) {
	cfg, ok := pt.models[modelID]
	if !ok {
		return 0, fmt.Errorf("unknown model %q for pricing", modelID)
	}
	price := cfg.PricePerForecast
	if price <= 0 {
		price = cfg.PricePerRequest
	}
	if price <= 0 {
		return 0, fmt.Errorf("model %q has no per-forecast pricing", modelID)
	}
	totalCents := int64(price * 10000)
	if totalCents < 1 {
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
	if cfg.PricePerForecast > 0 {
		return pt.CalculateForecastCost(modelID)
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
	if cfg.PricePerVideo > 0 || cfg.PricePerSecond > 0 || len(cfg.PricePerSecondByResolution) > 0 {
		return pt.CalculateVideoCost(modelID, 10, false)
	}
	if cfg.PricePerMinute > 0 || cfg.PricePerHour > 0 {
		return pt.CalculateAudioCost(modelID, 60)
	}
	if cfg.PricePer1MCharacters > 0 {
		return pt.CalculateCharacterCost(modelID, 1000)
	}
	if maxOutputTokens <= 0 {
		maxOutputTokens = 4096
	}
	return pt.CalculateCost(modelID, 0, maxOutputTokens)
}
