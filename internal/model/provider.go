package model

import (
	"strconv"
	"strings"
	"time"
)

// UTCPriceWindow is a half-open daily interval [start, end) expressed as
// zero-padded UTC times (for example, 01:00 through 04:00).
type UTCPriceWindow struct {
	Start string `yaml:"start" json:"start"`
	End   string `yaml:"end" json:"end"`
}

// ScheduledTokenPricing changes token rates at EffectiveAt and can apply
// distinct recurring peak and off-peak tariffs thereafter.
type ScheduledTokenPricing struct {
	EffectiveAt                 string           `yaml:"effective_at" json:"effective_at"`
	PeakWindowsUTC              []UTCPriceWindow `yaml:"peak_windows_utc" json:"peak_windows_utc"`
	PeakInputPricePer1M         float64          `yaml:"peak_input_price_per_1m" json:"peak_input_price_per_1m"`
	PeakInputCacheHitPricePer1M float64          `yaml:"peak_input_cache_hit_price_per_1m" json:"peak_input_cache_hit_price_per_1m"`
	PeakOutputPricePer1M        float64          `yaml:"peak_output_price_per_1m" json:"peak_output_price_per_1m"`
	OffPeakInputPricePer1M      float64          `yaml:"off_peak_input_price_per_1m" json:"off_peak_input_price_per_1m"`
	OffPeakInputCacheHitPer1M   float64          `yaml:"off_peak_input_cache_hit_price_per_1m" json:"off_peak_input_cache_hit_price_per_1m"`
	OffPeakOutputPricePer1M     float64          `yaml:"off_peak_output_price_per_1m" json:"off_peak_output_price_per_1m"`
}

// TemporaryProviderRoute overrides a model's normal upstream until ExpiresAt.
// It is intended for fixed-duration provider promotions while keeping the
// public OpenPaths model ID stable.
type TemporaryProviderRoute struct {
	Provider        string `yaml:"provider" json:"provider"`
	ProviderModelID string `yaml:"provider_model_id" json:"provider_model_id"`
	ExpiresAt       string `yaml:"expires_at" json:"expires_at"`
}

type ModelConfig struct {
	ID                           string                  `yaml:"id" json:"id"`
	Provider                     string                  `yaml:"provider" json:"provider"`
	ProviderModelID              string                  `yaml:"provider_model_id" json:"provider_model_id"`
	InputPricePer1M              float64                 `yaml:"input_price_per_1m" json:"input_price_per_1m"`
	InputCacheHitPricePer1M      float64                 `yaml:"input_cache_hit_price_per_1m,omitempty" json:"input_cache_hit_price_per_1m,omitempty"`
	OutputPricePer1M             float64                 `yaml:"output_price_per_1m" json:"output_price_per_1m"`
	ScheduledTokenPricing        *ScheduledTokenPricing  `yaml:"scheduled_token_pricing,omitempty" json:"scheduled_token_pricing,omitempty"`
	TemporaryProviderRoute       *TemporaryProviderRoute `yaml:"temporary_provider_route,omitempty" json:"temporary_provider_route,omitempty"`
	AudioInputPricePer1M         float64                 `yaml:"audio_input_price_per_1m,omitempty" json:"audio_input_price_per_1m,omitempty"`
	AudioInputCacheHitPricePer1M float64                 `yaml:"audio_input_cache_hit_price_per_1m,omitempty" json:"audio_input_cache_hit_price_per_1m,omitempty"`
	AudioOutputPricePer1M        float64                 `yaml:"audio_output_price_per_1m,omitempty" json:"audio_output_price_per_1m,omitempty"`
	ImageInputPricePer1M         float64                 `yaml:"image_input_price_per_1m,omitempty" json:"image_input_price_per_1m,omitempty"`
	ImageInputCacheHitPricePer1M float64                 `yaml:"image_input_cache_hit_price_per_1m,omitempty" json:"image_input_cache_hit_price_per_1m,omitempty"`
	PricePer1MCharacters         float64                 `yaml:"price_per_1m_characters,omitempty" json:"price_per_1m_characters,omitempty"`
	// Long-context tiered pricing: when set and the request's input tokens exceed
	// LongContextThreshold, the *Long rates apply to the whole request (e.g.
	// Sakana Fugu Ultra charges higher rates above 272K-token contexts).
	LongContextThreshold                 int                `yaml:"long_context_threshold,omitempty" json:"long_context_threshold,omitempty"`
	InputPricePer1MLong                  float64            `yaml:"input_price_per_1m_long,omitempty" json:"input_price_per_1m_long,omitempty"`
	InputCacheHitPricePer1MLong          float64            `yaml:"input_cache_hit_price_per_1m_long,omitempty" json:"input_cache_hit_price_per_1m_long,omitempty"`
	OutputPricePer1MLong                 float64            `yaml:"output_price_per_1m_long,omitempty" json:"output_price_per_1m_long,omitempty"`
	Aliases                              []string           `yaml:"aliases,omitempty" json:"aliases,omitempty"`
	ContextWindow                        int                `yaml:"context_window" json:"context_window"`
	MaxOutputTokens                      int                `yaml:"max_output_tokens" json:"max_output_tokens"`
	SupportsStreaming                    bool               `yaml:"supports_streaming" json:"supports_streaming"`
	SupportsTools                        bool               `yaml:"supports_tools" json:"supports_tools"`
	SupportsVision                       bool               `yaml:"supports_vision" json:"supports_vision"`
	DefaultReasoningEffort               string             `yaml:"default_reasoning_effort,omitempty" json:"default_reasoning_effort,omitempty"`
	PricePerRequest                      float64            `yaml:"price_per_request,omitempty" json:"price_per_request,omitempty"`
	PricePerImage                        float64            `yaml:"price_per_image,omitempty" json:"price_per_image,omitempty"`
	PricePerImageByResolution            map[string]float64 `yaml:"price_per_image_by_resolution,omitempty" json:"price_per_image_by_resolution,omitempty"`
	PricePerMegapixel                    float64            `yaml:"price_per_megapixel,omitempty" json:"price_per_megapixel,omitempty"`
	PricePerMegapixelWithImageInput      float64            `yaml:"price_per_megapixel_with_image_input,omitempty" json:"price_per_megapixel_with_image_input,omitempty"`
	RoundMegapixelPricing                bool               `yaml:"round_megapixel_pricing,omitempty" json:"round_megapixel_pricing,omitempty"`
	PriceFirstMegapixel                  float64            `yaml:"price_first_megapixel,omitempty" json:"price_first_megapixel,omitempty"`
	PriceExtraMegapixel                  float64            `yaml:"price_extra_megapixel,omitempty" json:"price_extra_megapixel,omitempty"`
	PricePerInputImage                   float64            `yaml:"price_per_input_image,omitempty" json:"price_per_input_image,omitempty"`
	FreeInputImageCount                  int                `yaml:"free_input_image_count,omitempty" json:"free_input_image_count,omitempty"`
	PricePerVideo                        float64            `yaml:"price_per_video,omitempty" json:"price_per_video,omitempty"`
	PricePerSecond                       float64            `yaml:"price_per_second,omitempty" json:"price_per_second,omitempty"`
	PricePerSecondByResolution           map[string]float64 `yaml:"price_per_second_by_resolution,omitempty" json:"price_per_second_by_resolution,omitempty"`
	PricePerSecondWithVideoInput         float64            `yaml:"price_per_second_with_video_input,omitempty" json:"price_per_second_with_video_input,omitempty"`
	PricePerInputVideoSecond             float64            `yaml:"price_per_input_video_second,omitempty" json:"price_per_input_video_second,omitempty"`
	PricePerInputVideoSecondByResolution map[string]float64 `yaml:"price_per_input_video_second_by_resolution,omitempty" json:"price_per_input_video_second_by_resolution,omitempty"`
	PricePerMinute                       float64            `yaml:"price_per_minute,omitempty" json:"price_per_minute,omitempty"`
	PricePerHour                         float64            `yaml:"price_per_hour,omitempty" json:"price_per_hour,omitempty"`
	PricePerForecast                     float64            `yaml:"price_per_forecast,omitempty" json:"price_per_forecast,omitempty"`
	FallbackProviders                    []string           `yaml:"fallback_providers,omitempty" json:"fallback_providers,omitempty"`
	FallbackModels                       []string           `yaml:"fallback_models,omitempty" json:"fallback_models,omitempty"`
	SupportedSizes                       []string           `yaml:"supported_sizes,omitempty" json:"supported_sizes,omitempty"`

	// Deprecated marks a model id whose upstream is gone or permanently
	// incompatible. The id keeps working -- provider_model_id points at the
	// replacement named in DeprecatedNote -- so existing callers do not break,
	// but it is delisted from the marketing catalog and flagged in /v1/models.
	Deprecated     bool   `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
	DeprecatedNote string `yaml:"deprecated_note,omitempty" json:"deprecated_note,omitempty"`
}

// TokenRatesAt returns the token rates effective at the supplied instant.
func (m *ModelConfig) TokenRatesAt(at time.Time) (float64, float64, float64) {
	inputRate, cacheRate, outputRate := m.InputPricePer1M, m.InputCacheHitPricePer1M, m.OutputPricePer1M
	schedule := m.ScheduledTokenPricing
	if schedule == nil {
		return inputRate, cacheRate, outputRate
	}
	effectiveAt, err := time.Parse(time.RFC3339, schedule.EffectiveAt)
	if err != nil || at.UTC().Before(effectiveAt) {
		return inputRate, cacheRate, outputRate
	}

	minute := at.UTC().Hour()*60 + at.UTC().Minute()
	for _, window := range schedule.PeakWindowsUTC {
		start, startOK := utcMinute(window.Start)
		end, endOK := utcMinute(window.End)
		if !startOK || !endOK {
			continue
		}
		if (start <= end && minute >= start && minute < end) ||
			(start > end && (minute >= start || minute < end)) {
			return schedule.PeakInputPricePer1M, schedule.PeakInputCacheHitPricePer1M, schedule.PeakOutputPricePer1M
		}
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

// ProviderRouteAt returns a copy with any active temporary route applied.
func (m *ModelConfig) ProviderRouteAt(at time.Time) *ModelConfig {
	route := m.TemporaryProviderRoute
	if route == nil || route.Provider == "" || route.ProviderModelID == "" {
		return m
	}
	expiresAt, err := time.Parse(time.RFC3339, route.ExpiresAt)
	if err != nil || !at.UTC().Before(expiresAt) {
		return m
	}
	routed := *m
	routed.Provider = route.Provider
	routed.ProviderModelID = route.ProviderModelID
	return &routed
}

type ProviderConfig struct {
	Name          string `yaml:"name" json:"name"`
	BaseURL       string `yaml:"base_url" json:"base_url"`
	APIKey        string `yaml:"api_key" json:"api_key"`
	Enabled       bool   `yaml:"enabled" json:"enabled"`
	AppReferer    string `yaml:"app_referer,omitempty" json:"app_referer,omitempty"`
	AppTitle      string `yaml:"app_title,omitempty" json:"app_title,omitempty"`
	AppCategories string `yaml:"app_categories,omitempty" json:"app_categories,omitempty"`
}
