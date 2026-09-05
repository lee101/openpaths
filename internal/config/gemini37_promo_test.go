package config

import "testing"

func TestGemini38IntroductoryPricing(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	wanted := map[string]bool{"gemini-3.8-flash": false, "gemini-3.7-flash": false, "gemini-latest": false}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		if _, ok := wanted[m.ID]; !ok {
			continue
		}
		wanted[m.ID] = true
		if m.Provider != "google" || m.ProviderModelID != "gemini-3.8-flash" {
			t.Errorf("%s standard route = %s/%s, want google/gemini-3.8-flash", m.ID, m.Provider, m.ProviderModelID)
		}
		if m.InputPricePer1M != .75 || m.InputCacheHitPricePer1M != .075 || m.OutputPricePer1M != 3.75 {
			t.Errorf("%s introductory pricing = %v/%v/%v, want .75/.075/3.75", m.ID, m.InputPricePer1M, m.InputCacheHitPricePer1M, m.OutputPricePer1M)
		}
		if m.TemporaryProviderRoute != nil {
			t.Errorf("%s still has expired temporary route = %+v", m.ID, m.TemporaryProviderRoute)
		}
		if m.ScheduledTokenPricing != nil {
			t.Errorf("%s still has expired scheduled pricing = %+v", m.ID, m.ScheduledTokenPricing)
		}
	}

	for id, found := range wanted {
		if !found {
			t.Errorf("%s missing from config", id)
		}
	}
}
