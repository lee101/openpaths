package config

import "testing"

func TestCerebrasQwen3827BModel(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	for i := range cfg.Models {
		m := &cfg.Models[i]
		if m.ID != "cerebras/qwen-3.8-27b" {
			continue
		}
		if m.Provider != "cerebras" || m.ProviderModelID != "qwen-3.8-27b" {
			t.Errorf("route = %s/%s, want cerebras/qwen-3.8-27b", m.Provider, m.ProviderModelID)
		}
		if m.InputPricePer1M != 0.99 || m.OutputPricePer1M != 1.49 {
			t.Errorf("pricing = %v/%v, want 0.99/1.49", m.InputPricePer1M, m.OutputPricePer1M)
		}
		if m.ContextWindow != 131072 || m.MaxOutputTokens != 40960 {
			t.Errorf("limits = %d/%d, want 131072/40960", m.ContextWindow, m.MaxOutputTokens)
		}
		if !m.SupportsStreaming || !m.SupportsTools || !m.SupportsVision {
			t.Errorf("Qwen 3.8 27B capability flags are incomplete: %+v", m)
		}
		return
	}
	t.Fatal("cerebras/qwen-3.8-27b missing from config")
}
