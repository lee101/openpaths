package config

import "testing"

// Models whose upstream is gone or permanently incompatible are not deleted --
// that would break callers using the id -- they are repointed at a working
// substitute and flagged. This guards both halves: the flag stays set, and the
// route no longer points at the dead upstream.
func TestDeprecatedModelsRouteToWorkingUpstreams(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	byID := map[string]int{}
	for i, m := range cfg.Models {
		byID[m.ID] = i
	}

	want := []struct {
		id           string
		provider     string
		providerName string
		deadUpstream string
	}{
		{"grok-4.20-multi-agent-0309", "xai", "grok-4.5", "grok-4.20-multi-agent-0309"},
		{"nvidia/gpt-oss-120b", "fireworks", "accounts/fireworks/models/gpt-oss-120b", "openai/gpt-oss-120b"},
		{"nvidia/mistral-medium-3.5", "mistral", "mistral-medium-latest", "mistralai/mistral-medium-3.5-128b"},
		// thinkingmachines/inkling was here while it had no upstream; the open
		// weights are now served by Together, so it is a live model again.
	}

	for _, w := range want {
		i, ok := byID[w.id]
		if !ok {
			t.Errorf("%s missing from config (deprecated ids must keep resolving)", w.id)
			continue
		}
		m := &cfg.Models[i]
		if !m.Deprecated {
			t.Errorf("%s: deprecated flag not set", w.id)
		}
		if m.DeprecatedNote == "" {
			t.Errorf("%s: deprecated_note is empty, callers get no migration hint", w.id)
		}
		if m.Provider != w.provider || m.ProviderModelID != w.providerName {
			t.Errorf("%s route = %s/%s, want %s/%s", w.id, m.Provider, m.ProviderModelID, w.provider, w.providerName)
		}
		if m.ProviderModelID == w.deadUpstream {
			t.Errorf("%s still points at the dead upstream %q", w.id, w.deadUpstream)
		}
	}
}

// A deprecated model must not be a fallback target for a healthy model: that
// would quietly route live traffic onto a substitute.
func TestDeprecatedModelsAreNotFallbackTargets(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	deprecated := map[string]bool{}
	for _, m := range cfg.Models {
		if m.Deprecated {
			deprecated[m.ID] = true
		}
	}
	for _, m := range cfg.Models {
		if m.Deprecated {
			continue
		}
		for _, fb := range m.FallbackModels {
			if deprecated[fb] {
				t.Errorf("%s falls back to deprecated model %s", m.ID, fb)
			}
		}
	}
}
