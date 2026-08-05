package config

import "testing"

func TestQwen38MaxOpenRouterModel(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	known := map[string]string{}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		known[m.ID] = m.ID
		for _, alias := range m.Aliases {
			if _, exists := known[alias]; !exists {
				known[alias] = m.ID
			}
		}
	}
	var found bool
	for i := range cfg.Models {
		m := &cfg.Models[i]
		if m.ID != "or/qwen3.8-max" {
			continue
		}
		found = true
		if m.Provider != "openrouter" || m.ProviderModelID != "qwen/qwen3.8-max" {
			t.Errorf("route = %s/%s, want openrouter/qwen/qwen3.8-max", m.Provider, m.ProviderModelID)
		}
		if m.InputPricePer1M != 2 || m.InputCacheHitPricePer1M != .25 || m.OutputPricePer1M != 6 {
			t.Errorf("pricing = %v/%v/%v, want 2/.25/6", m.InputPricePer1M, m.InputCacheHitPricePer1M, m.OutputPricePer1M)
		}
		if m.ContextWindow != 1000000 || m.MaxOutputTokens != 131072 {
			t.Errorf("limits = %d/%d, want 1000000/131072", m.ContextWindow, m.MaxOutputTokens)
		}
		if !m.SupportsStreaming || !m.SupportsTools || !m.SupportsVision {
			t.Errorf("Qwen 3.8 Max capability flags are incomplete: %+v", m)
		}
	}
	if !found {
		t.Fatal("or/qwen3.8-max missing from config")
	}
	for _, alias := range []string{"qwen3.8-max", "qwen3.8", "qwen-max", "qwen-latest", "qwen/qwen3.8-max"} {
		if got := known[alias]; got != "or/qwen3.8-max" {
			t.Errorf("alias %q resolves to %q, want or/qwen3.8-max", alias, got)
		}
	}
}
