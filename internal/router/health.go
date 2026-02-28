package router

import (
	"sync"
	"time"
)

type HealthTracker struct {
	mu        sync.RWMutex
	unhealthy map[string]time.Time
	failures  map[string]int
	baseCooldown time.Duration
	maxCooldown  time.Duration
}

func NewHealthTracker() *HealthTracker {
	return &HealthTracker{
		unhealthy:    make(map[string]time.Time),
		failures:     make(map[string]int),
		baseCooldown: 30 * time.Second,
		maxCooldown:  2 * time.Minute,
	}
}

func (h *HealthTracker) IsHealthy(name string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	until, ok := h.unhealthy[name]
	if !ok {
		return true
	}
	return time.Now().After(until)
}

func (h *HealthTracker) MarkUnhealthy(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failures[name]++
	cooldown := h.baseCooldown * time.Duration(h.failures[name])
	if cooldown > h.maxCooldown {
		cooldown = h.maxCooldown
	}
	h.unhealthy[name] = time.Now().Add(cooldown)
}

func (h *HealthTracker) MarkHealthy(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.unhealthy, name)
	delete(h.failures, name)
}

func (h *HealthTracker) ModelProviderKey(providerName, modelID string) string {
	return providerName + ":" + modelID
}
