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

	if got := models["openpaths/auto-code"]; got != "gpt-5.5" {
		t.Fatalf("openpaths/auto-code provider_model_id = %q, want %q", got, "gpt-5.5")
	}
	if got := models["openpaths/auto-reasoning"]; got != "gpt-5.4-mini" {
		t.Fatalf("openpaths/auto-reasoning provider_model_id = %q, want %q", got, "gpt-5.4-mini")
	}
	if got := models["openpaths/auto-cheap"]; got != "gpt-5.6-luna" {
		t.Fatalf("openpaths/auto-cheap provider_model_id = %q, want %q", got, "gpt-5.6-luna")
	}
	if got := models["openpaths/auto-fast"]; got != "deepseek-v4-flash" {
		t.Fatalf("openpaths/auto-fast provider_model_id = %q, want %q", got, "deepseek-v4-flash")
	}
	if got := models["openpaths/auto"]; got != "gemini-3.7-flash" {
		t.Fatalf("openpaths/auto provider_model_id = %q, want %q", got, "gemini-3.7-flash")
	}
	if got := models["openpaths/auto-image"]; got != "gpt-image-2" {
		t.Fatalf("openpaths/auto-image provider_model_id = %q, want %q", got, "gpt-image-2")
	}
	if got := models["gpt-5.5"]; got != "gpt-5.5" {
		t.Fatalf("gpt-5.5 provider_model_id = %q, want %q", got, "gpt-5.5")
	}
}

func TestGPTImage2FallsBackToOpenRouterGoogleImage(t *testing.T) {
	cfg := loadRepoConfig(t)
	for _, m := range cfg.Models {
		if m.ID != "gpt-image-2" {
			continue
		}
		if len(m.FallbackModels) < 1 || m.FallbackModels[0] != "or/gemini-3.1-flash-image" {
			t.Fatalf("gpt-image-2 fallbacks = %v, want OpenRouter Google image first", m.FallbackModels)
		}
		return
	}
	t.Fatal("gpt-image-2 model missing")
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

func TestResolveWithRetries_GPT55HasThreeProviderCoverage(t *testing.T) {
	cfg := loadRepoConfig(t)
	r := newTestRouter(cfg.Models, "openai", "anthropic", "google", "deepseek", "openrouter")

	candidates, err := r.ResolveWithRetries("gpt-5.5")
	if err != nil {
		t.Fatalf("ResolveWithRetries() error = %v", err)
	}

	providers := providerSet(candidates)
	for _, want := range []string{"openai", "anthropic", "google"} {
		if !providers[want] {
			t.Fatalf("gpt-5.5 candidates missing provider %q", want)
		}
	}
	if len(providers) < 3 {
		t.Fatalf("gpt-5.5 providers = %d, want at least 3", len(providers))
	}
}
