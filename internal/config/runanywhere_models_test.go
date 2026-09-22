package config

import (
	"path/filepath"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

// TestRunAnywhereLanes guards the Wally Cloud (RunAnywhere) onboarding: the
// provider block must stay enabled and keyed off RUNANYWHERE_API_KEY, and both
// hosted lanes must keep the upstream ids and limits the provider publishes for
// our key.
//
// Pricing derivation: the console quotes "cost to send/answer roughly 750,000
// words", and Wally bills per million tokens. 1M tokens is about 750k words, so
// the quoted figure is already the per-1M-token rate and is resold at list
// price like the other gateway lanes. Re-scaling that conversion either loses
// money (billing under the $0.10/$0.35 and $0.20/$2.50 cost) or prices the
// lanes out of the catalogue, so it is pinned here.
func TestRunAnywhereLanes(t *testing.T) {
	t.Setenv("RUNANYWHERE_API_KEY", "test-runanywhere-key")

	cfg, err := Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	cfgProvider := (*model.ProviderConfig)(nil)
	for i := range cfg.Providers {
		if cfg.Providers[i].Name == "runanywhere" {
			cfgProvider = &cfg.Providers[i]
		}
	}
	if cfgProvider == nil {
		t.Fatal("runanywhere provider missing from config.yaml")
	}
	if !cfgProvider.Enabled {
		t.Error("runanywhere provider is disabled")
	}
	// The OpenAI-compatible client appends /v1 itself, so the base URL must not
	// carry it or every chat call and health probe doubles the path.
	if cfgProvider.BaseURL != "https://inference.runanywhere.ai" {
		t.Errorf("base_url = %q, want https://inference.runanywhere.ai", cfgProvider.BaseURL)
	}
	if cfgProvider.APIKey != "test-runanywhere-key" {
		t.Errorf("api_key = %q, want it wired to RUNANYWHERE_API_KEY", cfgProvider.APIKey)
	}

	want := map[string]struct {
		upstream     string
		in, out      float64
		ctx, maxOut  int
		vision, tool bool
	}{
		"runanywhere/glm-5.3-flash": {"glm-5.3-flash", 0.10, 0.35, 1048567, 131072, true, true},
		"runanywhere/qwen3.8-27b":   {"qwen3.8-27b", 0.20, 2.50, 262137, 40960, true, true},
	}

	found := map[string]bool{}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		w, ok := want[m.ID]
		if !ok {
			continue
		}
		found[m.ID] = true
		if m.Provider != "runanywhere" {
			t.Errorf("%s provider = %s, want runanywhere", m.ID, m.Provider)
		}
		if m.ProviderModelID != w.upstream {
			t.Errorf("%s upstream id = %s, want %s", m.ID, m.ProviderModelID, w.upstream)
		}
		if m.InputPricePer1M != w.in || m.OutputPricePer1M != w.out {
			t.Errorf("%s pricing = %v/%v, want %v/%v", m.ID, m.InputPricePer1M, m.OutputPricePer1M, w.in, w.out)
		}
		if m.ContextWindow != w.ctx || m.MaxOutputTokens != w.maxOut {
			t.Errorf("%s limits = %d/%d, want %d/%d", m.ID, m.ContextWindow, m.MaxOutputTokens, w.ctx, w.maxOut)
		}
		if m.SupportsVision != w.vision || m.SupportsTools != w.tool || !m.SupportsStreaming {
			t.Errorf("%s capabilities = streaming:%t tools:%t vision:%t", m.ID, m.SupportsStreaming, m.SupportsTools, m.SupportsVision)
		}
	}
	for id := range want {
		if !found[id] {
			t.Errorf("%s missing from config.yaml", id)
		}
	}
}
