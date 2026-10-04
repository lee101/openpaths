package config

import (
	"github.com/openpaths/openpaths/internal/model"
	"path/filepath"
	"testing"
)

func TestSonnet55RoutesAndLatest(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	byID := map[string]*model.ModelConfig{}
	for i := range cfg.Models {
		byID[cfg.Models[i].ID] = &cfg.Models[i]
	}
	s55 := byID["claude-sonnet-5-5"]
	if s55 == nil {
		t.Fatal("claude-sonnet-5-5 missing from config")
	}
	if s55.Provider != "anthropic" || s55.ProviderModelID != "claude-sonnet-5-5" {
		t.Errorf("claude-sonnet-5-5 routes to %q/%q", s55.Provider, s55.ProviderModelID)
	}
	if s55.ContextWindow != 1000000 || s55.MaxOutputTokens != 128000 {
		t.Errorf("claude-sonnet-5-5 limits = %d/%d", s55.ContextWindow, s55.MaxOutputTokens)
	}
	for _, want := range []string{"sonnet-5.5", "claude-sonnet-5.5"} {
		found := false
		for _, a := range s55.Aliases {
			found = found || a == want
		}
		if !found {
			t.Errorf("claude-sonnet-5-5 missing alias %q", want)
		}
	}
	if latest := byID["claude-sonnet-latest"]; latest == nil || latest.ProviderModelID != "claude-sonnet-5-5" {
		t.Errorf("claude-sonnet-latest does not route to claude-sonnet-5-5")
	}
}
