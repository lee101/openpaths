package config

import (
	"path/filepath"
	"testing"
)

func TestGemini4ArgonDirectWithOpenRouterFallback(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]int{}
	for i, m := range cfg.Models {
		by[m.ID] = i
	}
	d, ok := by["gemini-4-argon"]
	if !ok {
		t.Fatal("gemini-4-argon missing")
	}
	m := cfg.Models[d]
	if m.Provider != "google" || m.ProviderModelID != "gemini-4-argon" {
		t.Errorf("route = %s/%s", m.Provider, m.ProviderModelID)
	}
	if m.InputPricePer1M != 2 || m.OutputPricePer1M != 10 || m.InputCacheHitPricePer1M != 0.1 {
		t.Errorf("price = %v/%v/%v", m.InputPricePer1M, m.InputCacheHitPricePer1M, m.OutputPricePer1M)
	}
	if len(m.FallbackModels) != 1 || m.FallbackModels[0] != "or/gemini-4-argon" {
		t.Fatalf("fallback = %v", m.FallbackModels)
	}
	f := cfg.Models[by["or/gemini-4-argon"]]
	if f.Provider != "openrouter" || f.ProviderModelID != "google/gemini-4-argon" {
		t.Errorf("fallback route = %s/%s", f.Provider, f.ProviderModelID)
	}
	if f.InputPricePer1M != m.InputPricePer1M || f.OutputPricePer1M != m.OutputPricePer1M {
		t.Error("fallback price differs from primary")
	}
}
