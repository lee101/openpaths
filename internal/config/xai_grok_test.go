package config

import (
	"path/filepath"
	"slices"
	"testing"
)

// grok-4.7 ships at grok-4.6's 2.00/6.00 rate with the same 500k context and
// 200k long-context threshold, so every grok id priced at or above that rate
// compat-routes to grok-4.7 and keeps its own price metadata. Ids on the
// cheaper 1.25/2.50 tier keep their own upstream and rate card.
func TestXAIGrokSamePriceRoutesToLatest(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	models := make(map[string]struct {
		provider      string
		providerModel string
		fallbacks     []string
	}, len(cfg.Models))
	for _, model := range cfg.Models {
		models[model.ID] = struct {
			provider      string
			providerModel string
			fallbacks     []string
		}{model.Provider, model.ProviderModelID, model.FallbackModels}
	}

	latest, ok := models["grok-4.7"]
	if !ok {
		t.Fatal("grok-4.7 is missing")
	}
	if latest.provider != "xai" || latest.providerModel != "grok-4.7" {
		t.Fatalf("grok-4.7 route = %s/%s, want xai/grok-4.7", latest.provider, latest.providerModel)
	}
	if !slices.Contains(latest.fallbacks, "or/grok-4") {
		t.Fatalf("grok-4.7 fallbacks = %v, want or/grok-4", latest.fallbacks)
	}

	for _, id := range []string{"grok-4.6", "grok-4.5", "grok-4.20-multi-agent-0309"} {
		model, ok := models[id]
		if !ok {
			t.Errorf("%s is missing", id)
			continue
		}
		if model.provider != "xai" || model.providerModel != "grok-4.7" {
			t.Errorf("%s route = %s/%s, want xai/grok-4.7", id, model.provider, model.providerModel)
		}
	}

	for _, id := range []string{"grok-4.3", "grok-4.20-0309-reasoning", "grok-4.20-0309-non-reasoning", "grok-build-0.1"} {
		model, ok := models[id]
		if !ok {
			t.Errorf("%s is missing", id)
			continue
		}
		if model.providerModel != id {
			t.Errorf("%s route = %s/%s, want its own upstream id", id, model.provider, model.providerModel)
		}
	}

	fallback, ok := models["or/grok-4"]
	if !ok {
		t.Fatal("or/grok-4 fallback is missing")
	}
	if fallback.provider != "openrouter" || fallback.providerModel != "x-ai/grok-4.7" {
		t.Fatalf("or/grok-4 route = %s/%s, want openrouter/x-ai/grok-4.7", fallback.provider, fallback.providerModel)
	}
}
