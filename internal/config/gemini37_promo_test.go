package config

import "testing"

func TestGemini37OpenRouterPromotion(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	wanted := map[string]bool{"gemini-3.7-flash": false, "gemini-latest": false}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		if _, ok := wanted[m.ID]; !ok {
			continue
		}
		wanted[m.ID] = true
		if m.Provider != "google" || m.ProviderModelID != "gemini-3.7-flash" {
			t.Errorf("%s standard route = %s/%s, want google/gemini-3.7-flash", m.ID, m.Provider, m.ProviderModelID)
		}
		if m.InputPricePer1M != .375 || m.InputCacheHitPricePer1M != .0375 || m.OutputPricePer1M != 1.875 {
			t.Errorf("%s promo pricing = %v/%v/%v, want .375/.0375/1.875", m.ID, m.InputPricePer1M, m.InputCacheHitPricePer1M, m.OutputPricePer1M)
		}
		route := m.TemporaryProviderRoute
		if route == nil || route.Provider != "openrouter" || route.ProviderModelID != "google/gemini-3.7-flash" || route.ExpiresAt != "2026-08-28T00:00:00Z" {
			t.Errorf("%s temporary route = %+v", m.ID, route)
		}
		pricing := m.ScheduledTokenPricing
		if pricing == nil || pricing.EffectiveAt != "2026-08-28T00:00:00Z" || pricing.OffPeakInputPricePer1M != .75 || pricing.OffPeakInputCacheHitPer1M != .075 || pricing.OffPeakOutputPricePer1M != 3.75 {
			t.Errorf("%s post-promo pricing = %+v", m.ID, pricing)
		}
	}

	for id, found := range wanted {
		if !found {
			t.Errorf("%s missing from config", id)
		}
	}
}
