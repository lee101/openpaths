package config

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestXAIGrok46FallsBackToOpenRouter(t *testing.T) {
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

	grok, ok := models["grok-4.6"]
	if !ok {
		t.Fatal("grok-4.6 is missing")
	}
	if grok.provider != "xai" || grok.providerModel != "grok-4.6" {
		t.Fatalf("grok-4.6 route = %s/%s, want xai/grok-4.6", grok.provider, grok.providerModel)
	}
	if !slices.Contains(grok.fallbacks, "or/grok-4") {
		t.Fatalf("grok-4.6 fallbacks = %v, want or/grok-4", grok.fallbacks)
	}

	fallback, ok := models["or/grok-4"]
	if !ok {
		t.Fatal("or/grok-4 fallback is missing")
	}
	if fallback.provider != "openrouter" || fallback.providerModel != "x-ai/grok-4.6" {
		t.Fatalf("or/grok-4 route = %s/%s, want openrouter/x-ai/grok-4.6", fallback.provider, fallback.providerModel)
	}
}
