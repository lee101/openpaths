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
		// The OpenRouter lane adds OpenRouter's 5.5% fee to the same list price.
		in, cache, out := 0.15, 0.03, 0.47
		if m.ID == "or/qwen3.8-flash" {
			in, cache, out = 0.1583, 0.0169, 0.4959
		}
		if m.InputPricePer1M != in || m.InputCacheHitPricePer1M != cache || m.OutputPricePer1M != out {
			t.Errorf("%s pricing = %v/%v/%v, want %v/%v/%v", m.ID, m.InputPricePer1M, m.InputCacheHitPricePer1M, m.OutputPricePer1M, in, cache, out)
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
	for alias, want := range map[string]string{"qwen3.8-flash": "qwen/qwen3.8-flash", "qwen-flash": "qwen/qwen3.8-flash", "qwen3.8-omni-flash": "qwen/qwen3.8-omni-flash", "qwen-omni-flash": "qwen/qwen3.8-omni-flash", "qwen3.8-omni": "qwen/qwen3.8-omni-flash"} {
		if got := known[alias]; got != want {
			t.Errorf("alias %q resolves to %q, want %s", alias, got, want)
		}
	}
}
