package model

type ModelConfig struct {
	ID                string   `yaml:"id" json:"id"`
	Provider          string   `yaml:"provider" json:"provider"`
	ProviderModelID   string   `yaml:"provider_model_id" json:"provider_model_id"`
	InputPricePer1M   float64  `yaml:"input_price_per_1m" json:"input_price_per_1m"`
	OutputPricePer1M  float64  `yaml:"output_price_per_1m" json:"output_price_per_1m"`
	Aliases           []string `yaml:"aliases,omitempty" json:"aliases,omitempty"`
	ContextWindow     int      `yaml:"context_window" json:"context_window"`
	MaxOutputTokens   int      `yaml:"max_output_tokens" json:"max_output_tokens"`
	SupportsStreaming bool     `yaml:"supports_streaming" json:"supports_streaming"`
	SupportsTools     bool     `yaml:"supports_tools" json:"supports_tools"`
	SupportsVision    bool     `yaml:"supports_vision" json:"supports_vision"`
	FallbackProviders []string `yaml:"fallback_providers,omitempty" json:"fallback_providers,omitempty"`
}

type ProviderConfig struct {
	Name    string `yaml:"name" json:"name"`
	BaseURL string `yaml:"base_url" json:"base_url"`
	APIKey  string `yaml:"api_key" json:"api_key"`
	Enabled bool   `yaml:"enabled" json:"enabled"`
}
