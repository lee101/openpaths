package config

import (
	"path/filepath"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestHaiku55RoutesAndLatest(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	byID := map[string]*model.ModelConfig{}
	for i := range cfg.Models {
		byID[cfg.Models[i].ID] = &cfg.Models[i]
	}
	h := byID["claude-haiku-5-5"]
	if h == nil {
		t.Fatal("claude-haiku-5-5 missing from config")
	}
	if h.Provider != "anthropic" || h.ProviderModelID != "claude-haiku-5-5" {
		t.Errorf("routes to %q/%q", h.Provider, h.ProviderModelID)
	}
	if h.InputPricePer1M != 0.10 || h.OutputPricePer1M != 0.50 {
		t.Errorf("price = %v/%v", h.InputPricePer1M, h.OutputPricePer1M)
	}
	if h.ContextWindow != 1000000 || h.MaxOutputTokens != 128000 {
		t.Errorf("limits = %d/%d", h.ContextWindow, h.MaxOutputTokens)
	}
	if h.LongContextThreshold != 100000 || h.InputPricePer1MLong != 0.50 || h.OutputPricePer1MLong != 2.50 || h.InputCacheHitPricePer1M != 0.01 {
		t.Errorf("tier = %d/%v/%v cache %v", h.LongContextThreshold, h.InputPricePer1MLong, h.OutputPricePer1MLong, h.InputCacheHitPricePer1M)
	}
	if latest := byID["claude-haiku-latest"]; latest == nil || latest.ProviderModelID != "claude-haiku-5-5" {
		t.Errorf("claude-haiku-latest does not route to claude-haiku-5-5")
	}
}
