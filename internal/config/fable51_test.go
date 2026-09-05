package config

import (
	"path/filepath"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestClaudeFable51Catalog(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	models := make(map[string]*model.ModelConfig)
	for _, m := range cfg.Models {
		m := m
		models[m.ID] = &m
	}

	for _, id := range []string{"claude-fable-latest", "claude-fable-5-1"} {
		m, ok := models[id]
		if !ok {
			t.Fatalf("%s missing from config.yaml", id)
		}
		if m.ProviderModelID != "claude-fable-5-1" {
			t.Errorf("%s provider_model_id = %q, want claude-fable-5-1", id, m.ProviderModelID)
		}
		if m.InputPricePer1M != 10 || m.OutputPricePer1M != 50 {
			t.Errorf("%s pricing = %v/%v, want 10/50", id, m.InputPricePer1M, m.OutputPricePer1M)
		}
		if m.ContextWindow != 1_000_000 {
			t.Errorf("%s context window = %d, want 1000000", id, m.ContextWindow)
		}
		if len(m.FallbackModels) != 1 || m.FallbackModels[0] != "or/claude-fable" {
			t.Errorf("%s fallback_models = %v, want [or/claude-fable]", id, m.FallbackModels)
		}
	}

	fallback := models["or/claude-fable"]
	if fallback == nil {
		t.Fatal("or/claude-fable missing from config.yaml")
	}
	if fallback.Provider != "openrouter" || fallback.ProviderModelID != "anthropic/claude-fable-5.1" {
		t.Errorf("OpenRouter fallback = %s/%s, want openrouter/anthropic/claude-fable-5.1",
			fallback.Provider, fallback.ProviderModelID)
	}
	if fallback.InputCacheHitPricePer1M != 0.25 {
		t.Errorf("OpenRouter fallback cache-hit pricing = %v, want 0.25", fallback.InputCacheHitPricePer1M)
	}
}
