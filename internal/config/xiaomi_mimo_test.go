package config

import (
	"path/filepath"
	"testing"
)

// Xiaomi publishes no direct API lane, so the MiMo-V2.6 family is reached
// through OpenRouter only. Each id must keep the xiaomi/ upstream slug, the
// 1M-token window OpenRouter advertises, the multimodal/tool capability flags,
// and aliases that resolve back to the entry (including the or/ prefix callers
// use to force OpenRouter).
func TestXiaomiMiMoV26OpenRouterRoutes(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]struct {
		in, cache, out float64
	}{
		"xiaomi/mimo-v2.6-pro":            {0.435, 0.0036, 0.87},
		"xiaomi/mimo-v2.6-pro-ultraspeed": {4.35, 0.036, 8.70},
		"xiaomi/mimo-v2.6-flash":          {0.14, 0.0028, 0.28},
	}
	aliases := map[string][]string{
		"xiaomi/mimo-v2.6-pro":            {"mimo-v2.6-pro", "mimo-pro", "or/mimo-v2.6-pro", "or/xiaomi/mimo-v2.6-pro"},
		"xiaomi/mimo-v2.6-pro-ultraspeed": {"mimo-v2.6-pro-ultraspeed", "mimo-v2.6-ultraspeed", "or/mimo-v2.6-pro-ultraspeed", "or/xiaomi/mimo-v2.6-pro-ultraspeed"},
		"xiaomi/mimo-v2.6-flash":          {"mimo-v2.6-flash", "mimo-flash", "or/mimo-v2.6-flash", "or/xiaomi/mimo-v2.6-flash"},
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

	seen := map[string]bool{}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		w, ok := want[m.ID]
		if !ok {
			continue
		}
		seen[m.ID] = true
		if m.Provider != "openrouter" || m.ProviderModelID != m.ID {
			t.Errorf("%s route = %s/%s, want openrouter/%s", m.ID, m.Provider, m.ProviderModelID, m.ID)
		}
		if m.InputPricePer1M != w.in || m.InputCacheHitPricePer1M != w.cache || m.OutputPricePer1M != w.out {
			t.Errorf("%s pricing = %v/%v/%v, want %v/%v/%v", m.ID, m.InputPricePer1M, m.InputCacheHitPricePer1M, m.OutputPricePer1M, w.in, w.cache, w.out)
		}
		if m.ContextWindow != 1048576 || m.MaxOutputTokens != 131072 {
			t.Errorf("%s limits = %d/%d, want 1048576/131072", m.ID, m.ContextWindow, m.MaxOutputTokens)
		}
		if !m.SupportsStreaming || !m.SupportsTools || !m.SupportsVision {
			t.Errorf("%s capability flags are incomplete: %+v", m.ID, m)
		}
	}
	for id := range want {
		if !seen[id] {
			t.Errorf("%s missing from config.yaml", id)
		}
	}
	for id, names := range aliases {
		for _, alias := range names {
			if got := known[alias]; got != id {
				t.Errorf("alias %q resolves to %q, want %q", alias, got, id)
			}
		}
	}
}
