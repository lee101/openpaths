package config

import (
	"path/filepath"
	"testing"
)

// Claude Opus 5.5 is the current flagship: $4/$20 per MTok with cache hits at
// 0.05x ($0.20, half the usual 0.1x), the full 1M context billed at standard
// rates, 128K max output. Sources: platform.claude.com/docs/en/about-claude/pricing
// and .../models/opus-5-5, cross checked against GET api.anthropic.com/v1/models
// and openrouter.ai/api/v1/models on 2026-09-22. Every other Opus id keeps its
// own id, alias and rate card but sends claude-opus-5-5 upstream; Claude Opus 4.1
// and Claude Opus 4 are retired on the Claude API (absent from GET /v1/models),
// so those two would hard-fail without the repoint.
func TestOpus55RoutesTheWholeOpusFamily(t *testing.T) {
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

	// Ids that a caller may still name: each must resolve, stay on the anthropic
	// provider, keep its own price, and point at the 5.5 checkpoint.
	family := []struct {
		id        string
		in, out   float64
		fallbacks bool
	}{
		{"claude-opus-latest", 5.00, 25.00, true},
		{"claude-opus-5", 5.00, 25.00, true},
		{"claude-opus-4-8", 5.00, 25.00, true},
		{"claude-opus-4-7", 5.00, 25.00, false},
		{"claude-opus-4-6", 5.00, 25.00, false},
		{"claude-opus-4-5-20251101", 5.00, 25.00, false},
		// Retired upstream, so these two only work because of the repoint.
		{"claude-opus-4-1-20250805", 15.00, 75.00, false},
		{"claude-opus-4-20250514", 15.00, 75.00, false},
	}
	for _, w := range family {
		i, ok := byID[w.id]
		if !ok {
			t.Errorf("model %q missing from config", w.id)
			continue
		}
		m := &cfg.Models[i]
		if m.Provider != "anthropic" || m.ProviderModelID != "claude-opus-5-5" {
			t.Errorf("%s routes to %q/%q, want anthropic/claude-opus-5-5", w.id, m.Provider, m.ProviderModelID)
		}
		if m.InputPricePer1M != w.in || m.OutputPricePer1M != w.out {
			t.Errorf("%s pricing = %v/%v, want %v/%v", w.id, m.InputPricePer1M, m.OutputPricePer1M, w.in, w.out)
		}
		if len(m.Aliases) == 0 {
			t.Errorf("%s lost its aliases", w.id)
		}
		if w.fallbacks && len(m.FallbackModels) == 0 {
			t.Errorf("%s has no fallback_models", w.id)
		}
		for _, fb := range m.FallbackModels {
			if !known[fb] {
				t.Errorf("%s fallback %q does not resolve", w.id, fb)
			}
			if fb == w.id {
				t.Errorf("%s falls back to itself", w.id)
			}
		}
	}

	i, ok := byID["claude-opus-5-5"]
	if !ok {
		t.Fatal("claude-opus-5-5 missing from config")
	}
	opus55 := &cfg.Models[i]
	if opus55.Provider != "anthropic" || opus55.ProviderModelID != "claude-opus-5-5" {
		t.Errorf("claude-opus-5-5 routes to %q/%q", opus55.Provider, opus55.ProviderModelID)
	}
	if opus55.InputPricePer1M != 4.00 || opus55.OutputPricePer1M != 20.00 {
		t.Errorf("claude-opus-5-5 pricing = %v/%v, want 4/20", opus55.InputPricePer1M, opus55.OutputPricePer1M)
	}
	// Opus 5.5 cache hits bill at 0.05x base, not the usual 0.1x.
	if opus55.InputCacheHitPricePer1M != 0.20 {
		t.Errorf("claude-opus-5-5 cache-hit rate = %v, want 0.2", opus55.InputCacheHitPricePer1M)
	}
	if opus55.ContextWindow != 1000000 || opus55.MaxOutputTokens != 128000 {
		t.Errorf("claude-opus-5-5 limits = %d/%d, want 1000000/128000", opus55.ContextWindow, opus55.MaxOutputTokens)
	}
	// 5.5 bills the whole window at standard rates, so no long-context tier.
	if opus55.LongContextThreshold != 0 {
		t.Errorf("claude-opus-5-5 has a long-context tier (%d) but Anthropic bills 5.5 at standard rates", opus55.LongContextThreshold)
	}
	for _, alias := range []string{"opus-5.5", "claude-opus-5.5"} {
		if !known[alias] {
			t.Errorf("claude-opus-5-5 alias %q not registered", alias)
		}
	}
	if len(opus55.FallbackModels) == 0 {
		t.Error("claude-opus-5-5 has no fallback_models")
	}

	// The short OpenRouter Opus lane follows the flagship and its list price too.
	oi, ok := byID["or/claude-opus"]
	if !ok {
		t.Fatal("or/claude-opus missing from config")
	}
	lane := &cfg.Models[oi]
	if lane.Provider != "openrouter" || lane.ProviderModelID != "anthropic/claude-opus-5.5" {
		t.Errorf("or/claude-opus routes to %q/%q, want openrouter/anthropic/claude-opus-5.5", lane.Provider, lane.ProviderModelID)
	}
	if lane.InputPricePer1M != 4.00 || lane.OutputPricePer1M != 20.00 {
		t.Errorf("or/claude-opus pricing = %v/%v, want 4/20", lane.InputPricePer1M, lane.OutputPricePer1M)
	}
}
