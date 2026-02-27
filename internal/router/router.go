package router

import (
	"fmt"
	"sync"
	"time"

	"github.com/openpath/openpath/internal/model"
	"github.com/openpath/openpath/internal/provider"
)

// Router maps model names to providers and handles fallback logic.
type Router struct {
	mu       sync.RWMutex
	models   map[string]*model.ModelConfig
	aliases  map[string]string
	registry *provider.Registry
	health   *HealthTracker
}

func New(registry *provider.Registry, models []model.ModelConfig) *Router {
	r := &Router{
		models:   make(map[string]*model.ModelConfig),
		aliases:  make(map[string]string),
		registry: registry,
		health:   NewHealthTracker(),
	}
	for i := range models {
		m := &models[i]
		r.models[m.ID] = m
		for _, alias := range m.Aliases {
			r.aliases[alias] = m.ID
		}
	}
	return r
}

// Resolve returns the ModelConfig and Provider for a given model name.
func (r *Router) Resolve(modelName string) (*model.ModelConfig, provider.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	canonical := modelName
	if mapped, ok := r.aliases[modelName]; ok {
		canonical = mapped
	}

	cfg, ok := r.models[canonical]
	if !ok {
		return nil, nil, fmt.Errorf("model %q not found", modelName)
	}

	// Try primary provider
	if r.health.IsHealthy(cfg.Provider) {
		p, err := r.registry.Get(cfg.Provider)
		if err == nil {
			return cfg, p, nil
		}
	}

	// Try fallbacks
	for _, fb := range cfg.FallbackProviders {
		if r.health.IsHealthy(fb) {
			p, err := r.registry.Get(fb)
			if err == nil {
				return cfg, p, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("no healthy provider available for model %q", modelName)
}

// MarkUnhealthy marks a provider as unhealthy for the cooldown period.
func (r *Router) MarkUnhealthy(providerName string) {
	r.health.MarkUnhealthy(providerName)
}

// ListModels returns all available models in OpenAI format.
func (r *Router) ListModels() []model.ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var models []model.ModelInfo
	for _, cfg := range r.models {
		models = append(models, model.ModelInfo{
			ID:      cfg.ID,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: cfg.Provider,
		})
	}
	return models
}

// GetModelConfig returns the config for a model by name or alias.
func (r *Router) GetModelConfig(modelName string) (*model.ModelConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	canonical := modelName
	if mapped, ok := r.aliases[modelName]; ok {
		canonical = mapped
	}
	cfg, ok := r.models[canonical]
	return cfg, ok
}
