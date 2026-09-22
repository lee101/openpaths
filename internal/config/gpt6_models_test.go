package config

import (
	"path/filepath"
	"testing"
)

// GPT-6 ships as capability tiers priced per token, and the rate card splits at
// 272K input tokens: above the threshold the *Long column prices the whole
// request. Sol and Luna land on both the direct OpenAI lane and the OpenRouter
// fallback lane, so every id, cache rate, threshold and fallback target has to
// resolve. Rates: developers.openai.com/api/docs/pricing and
// openrouter.ai/api/v1/models, verified 2026-09-22.
func TestGPT6TierModels(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "config.yaml"))
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
		id, provider, providerModel, alias string
		in, cache, out, long               float64
		longThreshold                      int
		// OpenRouter lanes are fallback targets themselves and carry no chain.
		requireFallbacks bool
	}{
		{"gpt-6-sol", "openai", "gpt-6-sol", "gpt6-sol", 2.00, 0.20, 10.00, 4.00, 272000, true},
		{"gpt-6-luna", "openai", "gpt-6-luna", "gpt6-luna", 0.10, 0.01, 0.50, 0.20, 272000, true},
		{"or/gpt-6-sol", "openrouter", "openai/gpt-6-sol", "openai/gpt-6-sol", 2.00, 0.20, 10.00, 4.00, 272000, false},
		{"or/gpt-6-luna", "openrouter", "openai/gpt-6-luna", "openai/gpt-6-luna", 0.10, 0.01, 0.50, 0.20, 272000, false},
	}
	for _, w := range want {
		i, ok := byID[w.id]
		if !ok {
			t.Errorf("model %q missing from config", w.id)
			continue
		}
		m := &cfg.Models[i]
		if m.Provider != w.provider || m.ProviderModelID != w.providerModel {
			t.Errorf("%s provider = %q/%q, want %q/%q", w.id, m.Provider, m.ProviderModelID, w.provider, w.providerModel)
		}
		if m.InputPricePer1M != w.in || m.InputCacheHitPricePer1M != w.cache || m.OutputPricePer1M != w.out {
			t.Errorf("%s pricing = %v/%v/%v, want %v/%v/%v",
				w.id, m.InputPricePer1M, m.InputCacheHitPricePer1M, m.OutputPricePer1M, w.in, w.cache, w.out)
		}
		if m.LongContextThreshold != w.longThreshold {
			t.Errorf("%s long_context_threshold = %d, want %d", w.id, m.LongContextThreshold, w.longThreshold)
		}
		// Above the threshold input and cache double; output is 1.5x.
		if m.InputPricePer1MLong != w.long || m.OutputPricePer1MLong != w.out*1.5 {
			t.Errorf("%s long-context pricing = %v/%v, want %v/%v",
				w.id, m.InputPricePer1MLong, m.OutputPricePer1MLong, w.long, w.out*1.5)
		}
		if m.InputCacheHitPricePer1MLong != w.cache*2 {
			t.Errorf("%s long-context cache rate = %v, want %v", w.id, m.InputCacheHitPricePer1MLong, w.cache*2)
		}
		if m.ContextWindow != 1050000 || m.MaxOutputTokens != 128000 {
			t.Errorf("%s limits = %d/%d, want 1050000/128000", w.id, m.ContextWindow, m.MaxOutputTokens)
		}
		if !m.SupportsStreaming || !m.SupportsTools || !m.SupportsVision {
			t.Errorf("%s is missing streaming, tool, or vision support", w.id)
		}
		if !known[w.alias] {
			t.Errorf("%s alias %q not registered", w.id, w.alias)
		}
		if w.requireFallbacks && len(m.FallbackModels) == 0 {
			t.Errorf("%s has no fallback_models", w.id)
		}
		for _, fb := range m.FallbackModels {
			if !known[fb] {
				t.Errorf("%s fallback %q does not resolve to any model id or alias", w.id, fb)
			}
			if fb == w.id {
				t.Errorf("%s falls back to itself", w.id)
			}
		}
	}
}
