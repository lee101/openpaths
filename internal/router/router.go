package router

import (
	"fmt"
	"sync"
	"time"

	"github.com/openpath/openpath/internal/model"
	"github.com/openpath/openpath/internal/provider"
)

type RouteCandidate struct {
	ModelCfg *model.ModelConfig
	Provider provider.Provider
}

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

	if r.health.IsHealthy(cfg.Provider) {
		p, err := r.registry.Get(cfg.Provider)
		if err == nil {
			return cfg, p, nil
		}
	}

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

func (r *Router) ResolveWithRetries(modelName string) ([]RouteCandidate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	canonical := modelName
	if mapped, ok := r.aliases[modelName]; ok {
		canonical = mapped
	}

	cfg, ok := r.models[canonical]
	if !ok {
		return nil, fmt.Errorf("model %q not found", modelName)
	}

	var candidates []RouteCandidate

	key := r.health.ModelProviderKey(cfg.Provider, cfg.ID)
	if r.health.IsHealthy(key) {
		if p, err := r.registry.Get(cfg.Provider); err == nil {
			candidates = append(candidates, RouteCandidate{ModelCfg: cfg, Provider: p})
		}
	}

	for _, fb := range cfg.FallbackProviders {
		fbKey := r.health.ModelProviderKey(fb, cfg.ID)
		if r.health.IsHealthy(fbKey) {
			if p, err := r.registry.Get(fb); err == nil {
				candidates = append(candidates, RouteCandidate{ModelCfg: cfg, Provider: p})
			}
		}
	}

	for _, fbModelID := range cfg.FallbackModels {
		fbCfg, ok := r.models[fbModelID]
		if !ok {
			continue
		}
		fbKey := r.health.ModelProviderKey(fbCfg.Provider, fbCfg.ID)
		if r.health.IsHealthy(fbKey) {
			if p, err := r.registry.Get(fbCfg.Provider); err == nil {
				candidates = append(candidates, RouteCandidate{ModelCfg: fbCfg, Provider: p})
			}
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no healthy provider available for model %q", modelName)
	}

	return candidates, nil
}

func (r *Router) HealthTracker() *HealthTracker {
	return r.health
}

func (r *Router) MarkUnhealthy(providerName string) {
	r.health.MarkUnhealthy(providerName)
}

func (r *Router) MarkModelUnhealthy(providerName, modelID string) {
	key := r.health.ModelProviderKey(providerName, modelID)
	r.health.MarkUnhealthy(key)
}

func (r *Router) MarkModelHealthy(providerName, modelID string) {
	key := r.health.ModelProviderKey(providerName, modelID)
	r.health.MarkHealthy(key)
}

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
