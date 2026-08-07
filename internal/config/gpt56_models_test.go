package config

import "testing"

// GPT-5.6 ships as three capability tiers (Sol/Terra/Luna). Each must exist in
// config.yaml with openai provider, correct pricing, and fallbacks that resolve.
func TestGPT56TierModels(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	byID := map[string]int{}
	known := map[string]bool{}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		byID[m.ID] = i
		known[m.ID] = true
		for _, a := range m.Aliases {
			known[a] = true
		}
	}
	want := []struct {
		id      string
		in, out float64
		alias   string
	}{
		{"gpt-5.6-sol", 5.00, 30.00, "gpt5.6-sol"},
		{"gpt-5.6-terra", 2.00, 12.00, "gpt5.6-terra"},
		{"gpt-5.6-luna", 0.20, 1.20, "gpt5.6-luna"},
	}
	for _, w := range want {
		i, ok := byID[w.id]
		if !ok {
			t.Errorf("model %q missing from config", w.id)
			continue
		}
		m := &cfg.Models[i]
		if m.Provider != "openai" {
			t.Errorf("%s provider = %q, want openai", w.id, m.Provider)
		}
		if m.ProviderModelID != w.id {
			t.Errorf("%s provider_model_id = %q, want %q", w.id, m.ProviderModelID, w.id)
		}
		if m.InputPricePer1M != w.in || m.OutputPricePer1M != w.out {
			t.Errorf("%s pricing = %v/%v, want %v/%v", w.id, m.InputPricePer1M, m.OutputPricePer1M, w.in, w.out)
		}
		if !known[w.alias] {
			t.Errorf("%s alias %q not registered", w.id, w.alias)
		}
		if len(m.FallbackModels) == 0 {
			t.Errorf("%s has no fallback_models", w.id)
		}
		for _, fb := range m.FallbackModels {
			if !known[fb] {
				t.Errorf("%s fallback %q does not resolve to any model id or alias", w.id, fb)
			}
		}
	}
}

func TestGPT56OpenRouterMirrors(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	byID := map[string]*struct {
		providerModelID               string
		input, cache, output          float64
		longInput, longCache, longOut float64
	}{}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		if m.ID != "or/gpt-5.6-sol" && m.ID != "or/gpt-5.6-terra" && m.ID != "or/gpt-5.6-luna" {
			continue
		}
		byID[m.ID] = &struct {
			providerModelID               string
			input, cache, output          float64
			longInput, longCache, longOut float64
		}{
			providerModelID: m.ProviderModelID,
			input:           m.InputPricePer1M, cache: m.InputCacheHitPricePer1M, output: m.OutputPricePer1M,
			longInput: m.InputPricePer1MLong, longCache: m.InputCacheHitPricePer1MLong, longOut: m.OutputPricePer1MLong,
		}
		if m.Provider != "openrouter" || m.ContextWindow != 1050000 || m.LongContextThreshold != 272000 {
			t.Errorf("%s OpenRouter metadata is incomplete: %+v", m.ID, m)
		}
	}
	want := map[string]struct {
		providerModelID               string
		input, cache, output          float64
		longInput, longCache, longOut float64
	}{
		"or/gpt-5.6-sol":   {"openai/gpt-5.6-sol", 5, .5, 30, 10, 1, 45},
		"or/gpt-5.6-terra": {"openai/gpt-5.6-terra", 1, .1, 6, 2, .2, 9},
		"or/gpt-5.6-luna":  {"openai/gpt-5.6-luna", .1, .01, .6, .2, .02, .9},
	}
	for id, expected := range want {
		got := byID[id]
		if got == nil {
			t.Errorf("%s missing from config", id)
			continue
		}
		if *got != expected {
			t.Errorf("%s = %+v, want %+v", id, *got, expected)
		}
	}
}
