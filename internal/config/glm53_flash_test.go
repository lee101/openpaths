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
		// Z.AI list price; the 50% launch promotion ended 2026-09-09.
		if m.InputPricePer1M != .15 || m.InputCacheHitPricePer1M != .03 || m.OutputPricePer1M != .50 {
			t.Errorf("pricing = %v/%v/%v, want .15/.03/.50", m.InputPricePer1M, m.InputCacheHitPricePer1M, m.OutputPricePer1M)
		}
		if m.ContextWindow != 1048576 || m.MaxOutputTokens != 131072 {
			t.Errorf("limits = %d/%d, want 1048576/131072", m.ContextWindow, m.MaxOutputTokens)
		}
		if !m.SupportsStreaming || !m.SupportsTools || !m.SupportsVision {
			t.Errorf("capabilities = streaming:%t tools:%t vision:%t", m.SupportsStreaming, m.SupportsTools, m.SupportsVision)
		}
		if m.ScheduledTokenPricing != nil {
			t.Errorf("expired promotion schedule still configured: %+v", m.ScheduledTokenPricing)
		}
	}
	if !found {
		t.Fatal("glm-5.3-flash missing from config.yaml")
	}
}
