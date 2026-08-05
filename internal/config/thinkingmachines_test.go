package config

import "testing"

// TINKER_API_KEY is still unset, so both Inkling ids are served from the open
// weights on Together (probed working 2026-07-31) with OpenRouter as the
// fallback hop. Repoint these if a Tinker key is ever configured.
func TestThinkingMachinesInklingConfig(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	var providerFound bool
	for _, p := range cfg.Providers {
		if p.Name != "thinkingmachines" {
			continue
		}
		providerFound = true
		if p.BaseURL != "https://tinker.thinkingmachines.dev/services/tinker-prod/oai/api" {
			t.Errorf("thinkingmachines base URL = %q", p.BaseURL)
		}
	}
	if !providerFound {
		t.Fatal("thinkingmachines provider missing")
	}

	byID := map[string]int{}
	for i, m := range cfg.Models {
		byID[m.ID] = i
	}

	want := []struct {
		id        string
		upstream  string
		in        float64
		out       float64
		cacheHit  float64
		fallbacks []string
	}{
		{"thinkingmachines/inkling", "thinkingmachines/Inkling", 1.00, 4.05, 0.17, []string{"or/inkling"}},
		{"inkling-small", "thinkingmachines/Inkling-Small", 0.50, 1.20, 0.10, []string{"thinkingmachines/inkling"}},
	}

	for _, w := range want {
		i, ok := byID[w.id]
		if !ok {
			t.Errorf("%s missing from config", w.id)
			continue
		}
		m := &cfg.Models[i]
		if m.Deprecated {
			t.Errorf("%s: deprecated flag set, but the Together route serves it", w.id)
		}
		if m.Provider != "together" || m.ProviderModelID != w.upstream {
			t.Errorf("%s route = %s/%s, want together/%s", w.id, m.Provider, m.ProviderModelID, w.upstream)
		}
		if m.InputPricePer1M != w.in || m.OutputPricePer1M != w.out || m.InputCacheHitPricePer1M != w.cacheHit {
			t.Errorf("%s pricing = %v/%v (cache %v), want %v/%v (cache %v)",
				w.id, m.InputPricePer1M, m.OutputPricePer1M, m.InputCacheHitPricePer1M, w.in, w.out, w.cacheHit)
		}
		if m.ContextWindow != 524288 {
			t.Errorf("%s context window = %d, want 524288 (Together serves 512K of the 1M window)", w.id, m.ContextWindow)
		}
		if !m.SupportsStreaming || !m.SupportsTools || !m.SupportsVision {
			t.Errorf("%s capability flags are incomplete", w.id)
		}
		if len(m.FallbackModels) != len(w.fallbacks) {
			t.Errorf("%s fallbacks = %v, want %v", w.id, m.FallbackModels, w.fallbacks)
			continue
		}
		for j, f := range w.fallbacks {
			if m.FallbackModels[j] != f {
				t.Errorf("%s fallback[%d] = %q, want %q", w.id, j, m.FallbackModels[j], f)
			}
		}
	}

	i, ok := byID["or/inkling"]
	if !ok {
		t.Fatal("or/inkling missing: thinkingmachines/inkling has no fallback hop")
	}
	if m := &cfg.Models[i]; m.Provider != "openrouter" || m.ProviderModelID != "thinkingmachines/inkling" {
		t.Errorf("or/inkling route = %s/%s", m.Provider, m.ProviderModelID)
	}
}
