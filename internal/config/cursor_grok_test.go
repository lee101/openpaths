package config

import (
	"path/filepath"
	"testing"
)

func TestCursorGrokModels(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		providerModel string
		input         float64
		cache         float64
		output        float64
	}{
		"cursor-grok-4.5":      {"grok-4.5", 2, 0.30, 6},
		"cursor-grok-4.5-fast": {"grok-4.5-fast", 4, 0.60, 18},
		"cursor-grok-4.6":      {"grok-4.6", 2, 0.50, 6},
		"cursor-grok-4.6-fast": {"grok-4.6-fast", 4, 1.00, 12},
	}
	seen := map[string]bool{}
	for _, m := range cfg.Models {
		w, ok := want[m.ID]
		if !ok {
			continue
		}
		seen[m.ID] = true
		if m.Provider != "cursor" || m.ProviderModelID != w.providerModel {
			t.Errorf("%s routes through %s/%s, want cursor/%s", m.ID, m.Provider, m.ProviderModelID, w.providerModel)
		}
		if m.InputPricePer1M != w.input || m.InputCacheHitPricePer1M != w.cache || m.OutputPricePer1M != w.output {
			t.Errorf("%s prices = %v/%v/%v, want %v/%v/%v", m.ID, m.InputPricePer1M, m.InputCacheHitPricePer1M, m.OutputPricePer1M, w.input, w.cache, w.output)
		}
		if m.ContextWindow != 256000 || m.MaxOutputTokens != 16384 || !m.SupportsStreaming {
			t.Errorf("%s capability metadata = context %d max_output %d streaming %v", m.ID, m.ContextWindow, m.MaxOutputTokens, m.SupportsStreaming)
		}
	}
	for id := range want {
		if !seen[id] {
			t.Errorf("%s missing from config.yaml", id)
		}
	}
}
