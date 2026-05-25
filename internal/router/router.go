package router

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

type RouteCandidate struct {
	ModelCfg *model.ModelConfig
	Provider provider.Provider
}

type Router struct {
	mu         sync.RWMutex
	models     map[string]*model.ModelConfig
	aliases    map[string]string
	registry   *provider.Registry
	health     *HealthTracker
	autoRouter *AutoRouter
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
	r.setDerivedAliases()
	return r
}

func (r *Router) setDerivedAliases() {
	if latest := latestGrokModelID(r.models); latest != "" {
		r.aliases["grok"] = latest
		r.aliases["grok-latest"] = latest
		if cfg := r.models[latest]; cfg != nil {
			cfg.Aliases = appendUniqueStrings(cfg.Aliases, "grok", "grok-latest")
		}
	}
}

func appendUniqueStrings(items []string, values ...string) []string {
	seen := make(map[string]struct{}, len(items)+len(values))
	for _, item := range items {
		seen[item] = struct{}{}
	}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		items = append(items, value)
		seen[value] = struct{}{}
	}
	return items
}

func latestGrokModelID(models map[string]*model.ModelConfig) string {
	var latest string
	var latestVersion grokVersion
	for id, cfg := range models {
		if cfg.Provider != "xai" {
			continue
		}
		if strings.Contains(strings.ToLower(id), "non-reasoning") {
			continue
		}
		version, ok := parseGrokVersion(id)
		if !ok {
			continue
		}
		if latest == "" || version.greaterThan(latestVersion) {
			latest = id
			latestVersion = version
		}
	}
	return latest
}

type grokVersion struct {
	major int
	minor int
}

func (v grokVersion) greaterThan(other grokVersion) bool {
	if v.major != other.major {
		return v.major > other.major
	}
	return v.minor > other.minor
}

func parseGrokVersion(id string) (grokVersion, bool) {
	rest, ok := strings.CutPrefix(strings.ToLower(id), "grok-")
	if !ok {
		return grokVersion{}, false
	}

	parts := strings.FieldsFunc(rest, func(r rune) bool {
		return r == '-' || r == '.'
	})
	if len(parts) == 0 {
		return grokVersion{}, false
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return grokVersion{}, false
	}

	minor := 0
	if len(parts) > 1 && len(parts[1]) <= 2 {
		if parsed, err := strconv.Atoi(parts[1]); err == nil {
			minor = parsed
		}
	}

	return grokVersion{major: major, minor: minor}, true
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

	canonical := r.canonicalModelNameLocked(modelName)
	return r.resolveWithRetriesLocked(modelName, canonical)
}

// ResolveForRequest prefers the routed model chosen by the autorouter, then
// falls back to the original requested model chain if the routed target is
// unavailable or unhealthy.
func (r *Router) ResolveForRequest(requestedModel, routedModel string) ([]RouteCandidate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	requestedCanonical := r.canonicalModelNameLocked(requestedModel)
	routedCanonical := r.canonicalModelNameLocked(routedModel)
	if routedCanonical == "" {
		routedCanonical = requestedCanonical
	}

	if requestedCanonical == routedCanonical {
		return r.resolveWithRetriesLocked(requestedModel, requestedCanonical)
	}

	var candidates []RouteCandidate
	seen := make(map[string]struct{})

	appendUnique := func(items []RouteCandidate) {
		for _, cand := range items {
			key := cand.Provider.Name() + "::" + cand.ModelCfg.ID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			candidates = append(candidates, cand)
		}
	}

	var routedErr error
	if routedCanonical != "" {
		routed, err := r.resolveWithRetriesLocked(routedModel, routedCanonical)
		if err != nil {
			routedErr = err
		} else {
			appendUnique(routed)
		}
	}

	requested, requestedErr := r.resolveWithRetriesLocked(requestedModel, requestedCanonical)
	if requestedErr == nil {
		appendUnique(requested)
	}

	if len(candidates) > 0 {
		return candidates, nil
	}
	if routedErr != nil {
		return nil, routedErr
	}
	return nil, requestedErr
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

func (r *Router) ListModelConfigs() []*model.ModelConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var configs []*model.ModelConfig
	for _, cfg := range r.models {
		configs = append(configs, cfg)
	}
	return configs
}

func (r *Router) ListModels() []model.ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now().Unix()
	var models []model.ModelInfo
	for _, cfg := range r.models {
		info := model.ModelInfo{
			ID:              cfg.ID,
			Object:          "model",
			Created:         now,
			OwnedBy:         cfg.Provider,
			ContextWindow:   cfg.ContextWindow,
			MaxOutputTokens: cfg.MaxOutputTokens,
			Aliases:         cfg.Aliases,
			SupportedSizes:  cfg.SupportedSizes,
			Pricing: &model.ModelPricing{
				InputPer1M:              cfg.InputPricePer1M,
				InputCacheHitPer1M:      cfg.InputCacheHitPricePer1M,
				OutputPer1M:             cfg.OutputPricePer1M,
				PerRequest:              cfg.PricePerRequest,
				PerImage:                cfg.PricePerImage,
				PerMegapixel:            cfg.PricePerMegapixel,
				FirstMegapixel:          cfg.PriceFirstMegapixel,
				ExtraMegapixel:          cfg.PriceExtraMegapixel,
				PerInputImage:           cfg.PricePerInputImage,
				PerVideo:                cfg.PricePerVideo,
				PerSecond:               cfg.PricePerSecond,
				PerSecondWithVideoInput: cfg.PricePerSecondWithVideoInput,
			},
			Capabilities: &model.ModelCapabilities{
				Streaming: cfg.SupportsStreaming,
				Tools:     cfg.SupportsTools,
				Vision:    cfg.SupportsVision,
			},
		}
		models = append(models, info)
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

func (r *Router) GetModelInfo(modelName string) (model.ModelInfo, bool) {
	cfg, ok := r.GetModelConfig(modelName)
	if !ok {
		return model.ModelInfo{}, false
	}
	return model.ModelInfo{
		ID:              cfg.ID,
		Object:          "model",
		Created:         time.Now().Unix(),
		OwnedBy:         cfg.Provider,
		ContextWindow:   cfg.ContextWindow,
		MaxOutputTokens: cfg.MaxOutputTokens,
		Aliases:         cfg.Aliases,
		SupportedSizes:  cfg.SupportedSizes,
		Pricing: &model.ModelPricing{
			InputPer1M:              cfg.InputPricePer1M,
			InputCacheHitPer1M:      cfg.InputCacheHitPricePer1M,
			OutputPer1M:             cfg.OutputPricePer1M,
			PerRequest:              cfg.PricePerRequest,
			PerImage:                cfg.PricePerImage,
			PerMegapixel:            cfg.PricePerMegapixel,
			FirstMegapixel:          cfg.PriceFirstMegapixel,
			ExtraMegapixel:          cfg.PriceExtraMegapixel,
			PerInputImage:           cfg.PricePerInputImage,
			PerVideo:                cfg.PricePerVideo,
			PerSecond:               cfg.PricePerSecond,
			PerSecondWithVideoInput: cfg.PricePerSecondWithVideoInput,
		},
		Capabilities: &model.ModelCapabilities{
			Streaming: cfg.SupportsStreaming,
			Tools:     cfg.SupportsTools,
			Vision:    cfg.SupportsVision,
		},
	}, true
}

func (r *Router) SetAutoRouter(ar *AutoRouter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.autoRouter = ar
}

func (r *Router) MaybeResolveAuto(ctx context.Context, modelName, modality, prompt string) AutoRouteResult {
	return r.MaybeResolveAutoWithTier(ctx, modelName, modality, "", prompt)
}

func (r *Router) MaybeResolveAutoReasoning(ctx context.Context, prompt string) string {
	if r.autoRouter == nil {
		return ""
	}

	result, err := r.autoRouter.ResolveAuto(ctx, "think-task", prompt)
	if err != nil {
		log.Printf("autorouter: %v, using default reasoning", err)
		return ""
	}
	return result.ReasoningEffort
}

// MaybeResolveAutoWithTier is MaybeResolveAuto extended with the cross-provider
// `task_tier` hint. When set (e.g. "easy" or "hard"), it promotes the request
// into the matching autorouter modality even if the caller did not use an
// `auto-*` model name, so clients can opt into tiered routing without changing
// their model string.
func (r *Router) MaybeResolveAutoWithTier(ctx context.Context, modelName, modality, taskTier, prompt string) AutoRouteResult {
	fallback := AutoRouteResult{ModelID: modelName}

	if r.autoRouter == nil {
		return fallback
	}

	mod, isAuto := IsAutoModel(modelName)

	// task_tier overrides any auto-* modality: an explicit hint from the client
	// is more specific than the model-name convention.
	if tierMod, ok := TaskTierToModality(taskTier); ok {
		mod = tierMod
		isAuto = true
	}

	if !isAuto {
		return fallback
	}

	// Only override modality for generic auto models (text/image/video).
	// Task-specific modalities (easy-task, medium-task, hard-task) use their
	// own routing table and ignore the caller-supplied modality hint.
	if modality != "" && mod == "text" {
		mod = modality
	}

	result, err := r.autoRouter.ResolveAuto(ctx, mod, prompt)
	if err != nil {
		log.Printf("autorouter: %v, using default", err)
		return fallback
	}

	return result
}

func (r *Router) canonicalModelNameLocked(modelName string) string {
	if mapped, ok := r.aliases[modelName]; ok {
		return mapped
	}
	return modelName
}

func (r *Router) resolveWithRetriesLocked(originalName, canonical string) ([]RouteCandidate, error) {
	cfg, ok := r.models[canonical]
	if !ok {
		return nil, fmt.Errorf("model %q not found", originalName)
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
		return nil, fmt.Errorf("no healthy provider available for model %q", originalName)
	}

	return candidates, nil
}
