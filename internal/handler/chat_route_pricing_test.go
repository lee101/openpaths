package handler

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestChatRoutePricingUsesServedModel(t *testing.T) {
	cfg := &model.ModelConfig{ID: "paid-fallback", InputPricePer1M: 0.11, InputCacheHitPricePer1M: 0.01, OutputPricePer1M: 0.51}
	got := chatRoutePricing(cfg, "nous", nil)
	if got.Model != cfg.ID || got.Provider != "nous" || got.Currency != "USD" || got.InputPricePer1M != 0.11 || got.InputCacheHitPricePer1M != 0.01 || got.OutputPricePer1M != 0.51 {
		t.Fatalf("wrong served-model pricing: %+v", got)
	}
	cfg.LongContextThreshold = 100
	cfg.InputPricePer1MLong = 0.22
	cfg.OutputPricePer1MLong = 1.02
	got = chatRoutePricing(cfg, "nous", &model.UsageInfo{PromptTokens: 101})
	if got.InputPricePer1M != 0.22 || got.OutputPricePer1M != 1.02 {
		t.Fatalf("wrong long-context pricing: %+v", got)
	}
	cfg = &model.ModelConfig{ID: "missingno"}
	got = chatRoutePricing(cfg, "nous", nil)
	if got.InputPricePer1M != 0 || got.InputCacheHitPricePer1M != 0 || got.OutputPricePer1M != 0 {
		t.Fatalf("free route has a charge: %+v", got)
	}
}
