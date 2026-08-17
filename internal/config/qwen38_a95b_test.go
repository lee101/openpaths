package config

import "testing"

func TestQwen38A95BTogetherModel(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	for i := range cfg.Models {
		m := &cfg.Models[i]
		if m.ID != "qwen3.8-2.4t-a95b" {
			continue
		}
		if m.Provider != "together" || m.ProviderModelID != "Qwen/Qwen3.8-2.4T-A95B" {
			t.Errorf("route = %s/%s, want together/Qwen/Qwen3.8-2.4T-A95B", m.Provider, m.ProviderModelID)
		}
		if m.InputPricePer1M != 2.50 || m.InputCacheHitPricePer1M != 0.50 || m.OutputPricePer1M != 6.25 {
			t.Errorf("pricing = %v/%v/%v, want 2.5/0.5/6.25", m.InputPricePer1M, m.InputCacheHitPricePer1M, m.OutputPricePer1M)
		}
		if m.ContextWindow != 262144 || m.MaxOutputTokens != 32768 {
			t.Errorf("limits = %d/%d, want 262144/32768", m.ContextWindow, m.MaxOutputTokens)
		}
		if !m.SupportsStreaming || !m.SupportsTools || !m.SupportsVision {
			t.Errorf("Qwen 3.8 2.4T A95B capability flags are incomplete: %+v", m)
		}
		return
	}
	t.Fatal("qwen3.8-2.4t-a95b missing from config")
}
