package handler

import (
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func chatRoutePricing(cfg *model.ModelConfig, provider string, usage *model.UsageInfo) *model.ChatRouting {
	input, cache, output := cfg.TokenRatesAt(time.Now())
	if usage != nil && cfg.LongContextThreshold > 0 && usage.PromptTokens > cfg.LongContextThreshold {
		if cfg.InputPricePer1MLong > 0 {
			input = cfg.InputPricePer1MLong
		}
		if cfg.InputCacheHitPricePer1MLong > 0 {
			cache = cfg.InputCacheHitPricePer1MLong
		}
		if cfg.OutputPricePer1MLong > 0 {
			output = cfg.OutputPricePer1MLong
		}
	}
	if cache <= 0 {
		cache = input
	}
	return &model.ChatRouting{
		Model: cfg.ID, Provider: provider, Currency: "USD",
		InputPricePer1M: input, InputCacheHitPricePer1M: cache, OutputPricePer1M: output,
	}
}
