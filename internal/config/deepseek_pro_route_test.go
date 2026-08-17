package config

import (
	"reflect"
	"testing"
)

func TestDeepSeekV4ProRoutesToFlash(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	for _, m := range cfg.Models {
		if m.ID != "deepseek-v4-pro" {
			continue
		}
		if m.Provider != "deepseek" || m.ProviderModelID != "deepseek-v4-flash" {
			t.Fatalf("deepseek-v4-pro route = %s/%s, want deepseek/deepseek-v4-flash", m.Provider, m.ProviderModelID)
		}
		wantFallbacks := []string{"nvidia/deepseek-v4-flash", "deepseek-reasoner", "nvidia/deepseek-v3.2", "together/deepseek-v3.1"}
		if !reflect.DeepEqual(m.FallbackModels, wantFallbacks) {
			t.Fatalf("deepseek-v4-pro fallbacks = %v, want %v", m.FallbackModels, wantFallbacks)
		}
		if m.InputPricePer1M != 0.435 || m.OutputPricePer1M != 0.87 {
			t.Fatalf("deepseek-v4-pro public prices changed: input=%v output=%v", m.InputPricePer1M, m.OutputPricePer1M)
		}
		if m.ScheduledTokenPricing == nil || m.ScheduledTokenPricing.EffectiveAt != "2026-08-16T16:00:00Z" ||
			m.ScheduledTokenPricing.PeakInputPricePer1M != 1.32 || m.ScheduledTokenPricing.OffPeakOutputPricePer1M != 1.98 {
			t.Fatalf("deepseek-v4-pro scheduled pricing not loaded correctly: %+v", m.ScheduledTokenPricing)
		}
		return
	}

	t.Fatal("deepseek-v4-pro missing from config")
}
