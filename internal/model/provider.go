package model

type ModelConfig struct {
	ID                           string   `yaml:"id" json:"id"`
	Provider                     string   `yaml:"provider" json:"provider"`
	ProviderModelID              string   `yaml:"provider_model_id" json:"provider_model_id"`
	InputPricePer1M              float64  `yaml:"input_price_per_1m" json:"input_price_per_1m"`
	InputCacheHitPricePer1M      float64  `yaml:"input_cache_hit_price_per_1m,omitempty" json:"input_cache_hit_price_per_1m,omitempty"`
	OutputPricePer1M             float64  `yaml:"output_price_per_1m" json:"output_price_per_1m"`
	// Long-context tiered pricing: when set and the request's input tokens exceed
	// LongContextThreshold, the *Long rates apply to the whole request (e.g.
	// Sakana Fugu Ultra charges higher rates above 272K-token contexts).
	LongContextThreshold         int      `yaml:"long_context_threshold,omitempty" json:"long_context_threshold,omitempty"`
	InputPricePer1MLong          float64  `yaml:"input_price_per_1m_long,omitempty" json:"input_price_per_1m_long,omitempty"`
	InputCacheHitPricePer1MLong  float64  `yaml:"input_cache_hit_price_per_1m_long,omitempty" json:"input_cache_hit_price_per_1m_long,omitempty"`
	OutputPricePer1MLong         float64  `yaml:"output_price_per_1m_long,omitempty" json:"output_price_per_1m_long,omitempty"`
	Aliases                      []string `yaml:"aliases,omitempty" json:"aliases,omitempty"`
	ContextWindow                int      `yaml:"context_window" json:"context_window"`
	MaxOutputTokens              int      `yaml:"max_output_tokens" json:"max_output_tokens"`
	SupportsStreaming            bool     `yaml:"supports_streaming" json:"supports_streaming"`
	SupportsTools                bool     `yaml:"supports_tools" json:"supports_tools"`
	SupportsVision               bool     `yaml:"supports_vision" json:"supports_vision"`
	PricePerRequest              float64  `yaml:"price_per_request,omitempty" json:"price_per_request,omitempty"`
	PricePerImage                float64  `yaml:"price_per_image,omitempty" json:"price_per_image,omitempty"`
	PricePerMegapixel            float64  `yaml:"price_per_megapixel,omitempty" json:"price_per_megapixel,omitempty"`
	PriceFirstMegapixel          float64  `yaml:"price_first_megapixel,omitempty" json:"price_first_megapixel,omitempty"`
	PriceExtraMegapixel          float64  `yaml:"price_extra_megapixel,omitempty" json:"price_extra_megapixel,omitempty"`
	PricePerInputImage           float64  `yaml:"price_per_input_image,omitempty" json:"price_per_input_image,omitempty"`
	PricePerVideo                float64  `yaml:"price_per_video,omitempty" json:"price_per_video,omitempty"`
	PricePerSecond               float64  `yaml:"price_per_second,omitempty" json:"price_per_second,omitempty"`
	PricePerSecondWithVideoInput float64  `yaml:"price_per_second_with_video_input,omitempty" json:"price_per_second_with_video_input,omitempty"`
	PricePerMinute               float64  `yaml:"price_per_minute,omitempty" json:"price_per_minute,omitempty"`
	PricePerHour                 float64  `yaml:"price_per_hour,omitempty" json:"price_per_hour,omitempty"`
	PricePerForecast             float64  `yaml:"price_per_forecast,omitempty" json:"price_per_forecast,omitempty"`
	FallbackProviders            []string `yaml:"fallback_providers,omitempty" json:"fallback_providers,omitempty"`
	FallbackModels               []string `yaml:"fallback_models,omitempty" json:"fallback_models,omitempty"`
	SupportedSizes               []string `yaml:"supported_sizes,omitempty" json:"supported_sizes,omitempty"`
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
