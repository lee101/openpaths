package router

import (
	"context"
	"testing"

	"github.com/openpath/openpath/internal/model"
	"github.com/openpath/openpath/internal/provider"
)

// mockProvider implements provider.Provider for testing.
type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) ChatCompletion(_ context.Context, _ *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	return nil, nil
}

func (m *mockProvider) ChatCompletionStream(_ context.Context, _ *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	return nil, nil
}

func (m *mockProvider) HealthCheck(_ context.Context) error {
	return nil
}

// newTestRouter creates a Router with the given models and registers mock
// providers for each provider name referenced in the model configs.
func newTestRouter(models []model.ModelConfig, providerNames ...string) *Router {
	reg := provider.NewRegistry()
	for _, name := range providerNames {
		reg.Register(&mockProvider{name: name})
	}
	return New(reg, models)
}

func TestResolve(t *testing.T) {
	models := []model.ModelConfig{
		{
			ID:                "gpt-4o",
			Provider:          "openai",
			ProviderModelID:   "gpt-4o-2024-08-06",
			Aliases:           []string{"gpt4", "gpt-4"},
			FallbackProviders: []string{"anthropic"},
		},
		{
			ID:              "claude-3-opus",
			Provider:        "anthropic",
			ProviderModelID: "claude-3-opus-20240229",
		},
	}

	tests := []struct {
		name             string
		modelName        string
		unhealthy        []string
		wantModelID      string
		wantProviderName string
		wantErr          bool
	}{
		{
			name:             "resolve by exact model ID",
			modelName:        "gpt-4o",
			wantModelID:      "gpt-4o",
			wantProviderName: "openai",
		},
		{
			name:             "resolve by alias gpt4",
			modelName:        "gpt4",
			wantModelID:      "gpt-4o",
			wantProviderName: "openai",
		},
		{
			name:             "resolve by alias gpt-4",
			modelName:        "gpt-4",
			wantModelID:      "gpt-4o",
			wantProviderName: "openai",
		},
		{
			name:             "resolve second model by ID",
			modelName:        "claude-3-opus",
			wantModelID:      "claude-3-opus",
			wantProviderName: "anthropic",
		},
		{
			name:      "error for unknown model",
			modelName: "nonexistent-model",
			wantErr:   true,
		},
		{
			name:             "fallback when primary is unhealthy",
			modelName:        "gpt-4o",
			unhealthy:        []string{"openai"},
			wantModelID:      "gpt-4o",
			wantProviderName: "anthropic",
		},
		{
			name:      "error when all providers are unhealthy",
			modelName: "gpt-4o",
			unhealthy: []string{"openai", "anthropic"},
			wantErr:   true,
		},
		{
			name:      "error when model has no fallback and primary is unhealthy",
			modelName: "claude-3-opus",
			unhealthy: []string{"anthropic"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestRouter(models, "openai", "anthropic")

			for _, name := range tt.unhealthy {
				r.MarkUnhealthy(name)
			}

			cfg, p, err := r.Resolve(tt.modelName)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("Resolve(%q) expected error, got nil", tt.modelName)
				}
				return
			}

			if err != nil {
				t.Fatalf("Resolve(%q) unexpected error: %v", tt.modelName, err)
			}

			if cfg.ID != tt.wantModelID {
				t.Errorf("Resolve(%q) model ID = %q, want %q", tt.modelName, cfg.ID, tt.wantModelID)
			}

			if p.Name() != tt.wantProviderName {
				t.Errorf("Resolve(%q) provider = %q, want %q", tt.modelName, p.Name(), tt.wantProviderName)
			}
		})
	}
}

func TestResolve_FallbackOrder(t *testing.T) {
	models := []model.ModelConfig{
		{
			ID:                "test-model",
			Provider:          "primary",
			FallbackProviders: []string{"fallback-1", "fallback-2", "fallback-3"},
		},
	}

	tests := []struct {
		name             string
		unhealthy        []string
		wantProviderName string
		wantErr          bool
	}{
		{
			name:             "uses primary when all healthy",
			wantProviderName: "primary",
		},
		{
			name:             "uses first fallback when primary is down",
			unhealthy:        []string{"primary"},
			wantProviderName: "fallback-1",
		},
		{
			name:             "skips unhealthy fallbacks",
			unhealthy:        []string{"primary", "fallback-1"},
			wantProviderName: "fallback-2",
		},
		{
			name:             "uses last fallback when others are down",
			unhealthy:        []string{"primary", "fallback-1", "fallback-2"},
			wantProviderName: "fallback-3",
		},
		{
			name:      "error when everything is down",
			unhealthy: []string{"primary", "fallback-1", "fallback-2", "fallback-3"},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestRouter(models, "primary", "fallback-1", "fallback-2", "fallback-3")

			for _, name := range tt.unhealthy {
				r.MarkUnhealthy(name)
			}

			_, p, err := r.Resolve("test-model")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if p.Name() != tt.wantProviderName {
				t.Errorf("provider = %q, want %q", p.Name(), tt.wantProviderName)
			}
		})
	}
}

func TestResolve_UnregisteredProvider(t *testing.T) {
	// The model config references "missing-provider" but the registry
	// does not have it registered. Resolve should return an error.
	models := []model.ModelConfig{
		{
			ID:       "test-model",
			Provider: "missing-provider",
		},
	}

	r := newTestRouter(models) // no providers registered

	_, _, err := r.Resolve("test-model")
	if err == nil {
		t.Fatal("expected error when provider is not registered, got nil")
	}
}

func TestListModels(t *testing.T) {
	tests := []struct {
		name       string
		models     []model.ModelConfig
		wantCount  int
		wantIDs    map[string]bool
		wantOwnedBy map[string]string
	}{
		{
			name:      "empty model list",
			models:    nil,
			wantCount: 0,
		},
		{
			name: "single model",
			models: []model.ModelConfig{
				{ID: "gpt-4o", Provider: "openai"},
			},
			wantCount:   1,
			wantIDs:     map[string]bool{"gpt-4o": true},
			wantOwnedBy: map[string]string{"gpt-4o": "openai"},
		},
		{
			name: "multiple models",
			models: []model.ModelConfig{
				{ID: "gpt-4o", Provider: "openai"},
				{ID: "claude-3-opus", Provider: "anthropic"},
				{ID: "gemini-pro", Provider: "google"},
			},
			wantCount: 3,
			wantIDs: map[string]bool{
				"gpt-4o":       true,
				"claude-3-opus": true,
				"gemini-pro":   true,
			},
			wantOwnedBy: map[string]string{
				"gpt-4o":       "openai",
				"claude-3-opus": "anthropic",
				"gemini-pro":   "google",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestRouter(tt.models)

			infos := r.ListModels()

			if len(infos) != tt.wantCount {
				t.Fatalf("ListModels() returned %d models, want %d", len(infos), tt.wantCount)
			}

			for _, info := range infos {
				if info.Object != "model" {
					t.Errorf("model %q has Object = %q, want %q", info.ID, info.Object, "model")
				}
				if info.Created == 0 {
					t.Errorf("model %q has Created = 0, want nonzero timestamp", info.ID)
				}
				if !tt.wantIDs[info.ID] {
					t.Errorf("unexpected model ID %q in ListModels output", info.ID)
				}
				if want, ok := tt.wantOwnedBy[info.ID]; ok && info.OwnedBy != want {
					t.Errorf("model %q OwnedBy = %q, want %q", info.ID, info.OwnedBy, want)
				}
			}
		})
	}
}

func TestGetModelConfig(t *testing.T) {
	models := []model.ModelConfig{
		{
			ID:              "gpt-4o",
			Provider:        "openai",
			ProviderModelID: "gpt-4o-2024-08-06",
			Aliases:         []string{"gpt4"},
			ContextWindow:   128000,
			MaxOutputTokens: 4096,
		},
		{
			ID:              "claude-3-opus",
			Provider:        "anthropic",
			ProviderModelID: "claude-3-opus-20240229",
			ContextWindow:   200000,
		},
	}

	tests := []struct {
		name            string
		modelName       string
		wantFound       bool
		wantID          string
		wantProvider    string
		wantContext     int
	}{
		{
			name:         "find by exact ID",
			modelName:    "gpt-4o",
			wantFound:    true,
			wantID:       "gpt-4o",
			wantProvider: "openai",
			wantContext:  128000,
		},
		{
			name:         "find by alias",
			modelName:    "gpt4",
			wantFound:    true,
			wantID:       "gpt-4o",
			wantProvider: "openai",
			wantContext:  128000,
		},
		{
			name:      "not found for unknown model",
			modelName: "nonexistent-model",
			wantFound: false,
		},
		{
			name:         "find second model",
			modelName:    "claude-3-opus",
			wantFound:    true,
			wantID:       "claude-3-opus",
			wantProvider: "anthropic",
			wantContext:  200000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newTestRouter(models, "openai", "anthropic")

			cfg, found := r.GetModelConfig(tt.modelName)

			if found != tt.wantFound {
				t.Fatalf("GetModelConfig(%q) found = %v, want %v", tt.modelName, found, tt.wantFound)
			}

			if !tt.wantFound {
				return
			}

			if cfg.ID != tt.wantID {
				t.Errorf("GetModelConfig(%q) ID = %q, want %q", tt.modelName, cfg.ID, tt.wantID)
			}
			if cfg.Provider != tt.wantProvider {
				t.Errorf("GetModelConfig(%q) Provider = %q, want %q", tt.modelName, cfg.Provider, tt.wantProvider)
			}
			if cfg.ContextWindow != tt.wantContext {
				t.Errorf("GetModelConfig(%q) ContextWindow = %d, want %d", tt.modelName, cfg.ContextWindow, tt.wantContext)
			}
		})
	}
}

func TestNew_AliasMapping(t *testing.T) {
	models := []model.ModelConfig{
		{
			ID:      "model-a",
			Provider: "provider-a",
			Aliases: []string{"alias-1", "alias-2", "alias-3"},
		},
	}

	r := newTestRouter(models, "provider-a")

	// All aliases should resolve to the same model config.
	for _, name := range []string{"model-a", "alias-1", "alias-2", "alias-3"} {
		cfg, found := r.GetModelConfig(name)
		if !found {
			t.Errorf("GetModelConfig(%q) not found", name)
			continue
		}
		if cfg.ID != "model-a" {
			t.Errorf("GetModelConfig(%q) ID = %q, want %q", name, cfg.ID, "model-a")
		}
	}
}

func TestNew_EmptyModels(t *testing.T) {
	r := newTestRouter(nil)

	infos := r.ListModels()
	if len(infos) != 0 {
		t.Errorf("ListModels() returned %d models for empty router, want 0", len(infos))
	}

	_, _, err := r.Resolve("anything")
	if err == nil {
		t.Error("Resolve on empty router should return error, got nil")
	}

	_, found := r.GetModelConfig("anything")
	if found {
		t.Error("GetModelConfig on empty router should return false, got true")
	}
}
