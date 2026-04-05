package router

import (
	"path/filepath"
	"testing"

	"github.com/openpaths/openpaths/internal/config"
)

func loadRepoConfig(t *testing.T) *config.Config {
	t.Helper()

	cfg, err := config.Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	return cfg
}

func providerSet(candidates []RouteCandidate) map[string]bool {
	out := make(map[string]bool)
	for _, cand := range candidates {
		out[cand.Provider.Name()] = true
	}
	return out
}

func TestAutoTierConfig_PrimaryModels(t *testing.T) {
	cfg := loadRepoConfig(t)

	models := map[string]string{}
	for _, m := range cfg.Models {
		models[m.ID] = m.ProviderModelID
	}

	if got := models["auto-medium-task"]; got != "gpt-5.4-mini" {
		t.Fatalf("auto-medium-task provider_model_id = %q, want %q", got, "gpt-5.4-mini")
	}
	if got := models["auto-think"]; got != "gpt-5.4-mini" {
		t.Fatalf("auto-think provider_model_id = %q, want %q", got, "gpt-5.4-mini")
	}
	if got := models["auto-easy-task"]; got != "gemini-flash-lite-latest" {
		t.Fatalf("auto-easy-task provider_model_id = %q, want %q", got, "gemini-flash-lite-latest")
	}
}

func TestResolveWithRetries_GPT54NanoHasThreeProviderCoverage(t *testing.T) {
	cfg := loadRepoConfig(t)
	r := newTestRouter(cfg.Models, "openai", "google", "anthropic", "openrouter")

	candidates, err := r.ResolveWithRetries("gpt-5.4-nano")
	if err != nil {
		t.Fatalf("ResolveWithRetries() error = %v", err)
	}

	providers := providerSet(candidates)
	for _, want := range []string{"openai", "google", "anthropic"} {
		if !providers[want] {
			t.Fatalf("gpt-5.4-nano candidates missing provider %q", want)
		}
	}
	if len(providers) < 3 {
		t.Fatalf("gpt-5.4-nano providers = %d, want at least 3", len(providers))
	}
}

func TestResolveWithRetries_GPT54MiniHasThreeProviderCoverage(t *testing.T) {
	cfg := loadRepoConfig(t)
	r := newTestRouter(cfg.Models, "openai", "google", "anthropic", "deepseek", "openrouter")

	candidates, err := r.ResolveWithRetries("gpt-5.4-mini")
	if err != nil {
		t.Fatalf("ResolveWithRetries() error = %v", err)
	}

	providers := providerSet(candidates)
	for _, want := range []string{"openai", "anthropic", "google"} {
		if !providers[want] {
			t.Fatalf("gpt-5.4-mini candidates missing provider %q", want)
		}
	}
	if len(providers) < 3 {
		t.Fatalf("gpt-5.4-mini providers = %d, want at least 3", len(providers))
	}
}

func TestResolveWithRetries_GPT54HasThreeProviderCoverage(t *testing.T) {
	cfg := loadRepoConfig(t)
	r := newTestRouter(cfg.Models, "openai", "anthropic", "google", "deepseek", "openrouter")

	candidates, err := r.ResolveWithRetries("gpt-5.4")
	if err != nil {
		t.Fatalf("ResolveWithRetries() error = %v", err)
	}

	providers := providerSet(candidates)
	for _, want := range []string{"openai", "anthropic", "google"} {
		if !providers[want] {
			t.Fatalf("gpt-5.4 candidates missing provider %q", want)
		}
	}
	if len(providers) < 3 {
		t.Fatalf("gpt-5.4 providers = %d, want at least 3", len(providers))
	}
}
