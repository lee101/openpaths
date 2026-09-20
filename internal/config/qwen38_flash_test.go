package config

import "testing"

func TestQwen38FlashModels(t *testing.T) {
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
	var direct, or bool
	for i := range cfg.Models {
		m := &cfg.Models[i]
		switch m.ID {
		case "qwen/qwen3.8-flash":
			direct = true
			if m.Provider != "qwen" || m.ProviderModelID != "qwen3.8-flash" {
				t.Errorf("route = %s/%s, want qwen/qwen3.8-flash", m.Provider, m.ProviderModelID)
			}
			if len(m.FallbackModels) != 1 || m.FallbackModels[0] != "or/qwen3.8-flash" {
				t.Errorf("fallbacks = %v, want [or/qwen3.8-flash]", m.FallbackModels)
			}
		case "or/qwen3.8-flash":
			or = true
			if m.Provider != "openrouter" || m.ProviderModelID != "qwen/qwen3.8-flash" {
				t.Errorf("route = %s/%s, want openrouter/qwen/qwen3.8-flash", m.Provider, m.ProviderModelID)
			}
		default:
			continue
		}
		if m.InputPricePer1M != 0.15 || m.InputCacheHitPricePer1M != 0.016 || m.OutputPricePer1M != 0.47 {
			t.Errorf("%s pricing = %v/%v/%v, want 0.15/0.016/0.47", m.ID, m.InputPricePer1M, m.InputCacheHitPricePer1M, m.OutputPricePer1M)
		}
		if m.ContextWindow != 1000000 || m.MaxOutputTokens != 131072 {
			t.Errorf("%s limits = %d/%d, want 1000000/131072", m.ID, m.ContextWindow, m.MaxOutputTokens)
		}
		if !m.SupportsStreaming || !m.SupportsTools || !m.SupportsVision {
			t.Errorf("%s capability flags are incomplete: %+v", m.ID, m)
		}
	}
	if !direct {
		t.Error("qwen/qwen3.8-flash missing from config")
	}
	if !or {
		t.Error("or/qwen3.8-flash missing from config")
	}
	for _, alias := range []string{"qwen3.8-flash", "qwen3.8-omni-flash", "qwen-omni-flash", "qwen3.8-omni", "qwen-flash"} {
		if got := known[alias]; got != "qwen/qwen3.8-flash" {
			t.Errorf("alias %q resolves to %q, want qwen/qwen3.8-flash", alias, got)
		}
	}
}
