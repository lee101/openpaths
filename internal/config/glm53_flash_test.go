package config

import "testing"

func TestGLM53FlashRoute(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	var found bool
	for i := range cfg.Models {
		m := &cfg.Models[i]
		if m.ID != "glm-5.3-flash" {
			continue
		}
		found = true
		if m.Provider != "zai" || m.ProviderModelID != "glm-5.3-flash" {
			t.Fatalf("route = %s/%s, want zai/glm-5.3-flash", m.Provider, m.ProviderModelID)
		}
		if m.InputPricePer1M != .075 || m.InputCacheHitPricePer1M != .015 || m.OutputPricePer1M != .25 {
			t.Errorf("promotion pricing = %v/%v/%v, want .075/.015/.25", m.InputPricePer1M, m.InputCacheHitPricePer1M, m.OutputPricePer1M)
		}
		if m.ContextWindow != 1048576 || m.MaxOutputTokens != 131072 {
			t.Errorf("limits = %d/%d, want 1048576/131072", m.ContextWindow, m.MaxOutputTokens)
		}
		if !m.SupportsStreaming || !m.SupportsTools || !m.SupportsVision {
			t.Errorf("capabilities = streaming:%t tools:%t vision:%t", m.SupportsStreaming, m.SupportsTools, m.SupportsVision)
		}
		pricing := m.ScheduledTokenPricing
		if pricing == nil || pricing.EffectiveAt != "2026-09-09T16:00:00Z" ||
			pricing.OffPeakInputPricePer1M != .15 ||
			pricing.OffPeakInputCacheHitPer1M != .03 ||
			pricing.OffPeakOutputPricePer1M != .50 {
			t.Errorf("post-promotion pricing = %+v", pricing)
		}
	}
	if !found {
		t.Fatal("glm-5.3-flash missing from config.yaml")
	}
}
