package router

import (
	"sync"
	"time"
)

// HealthTracker monitors provider health with automatic recovery.
type HealthTracker struct {
	mu        sync.RWMutex
	unhealthy map[string]time.Time
	cooldown  time.Duration
}

func NewHealthTracker() *HealthTracker {
	return &HealthTracker{
		unhealthy: make(map[string]time.Time),
		cooldown:  30 * time.Second,
	}
}

func (h *HealthTracker) IsHealthy(providerName string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	until, ok := h.unhealthy[providerName]
	if !ok {
		return true
	}
	return time.Now().After(until)
}

func (h *HealthTracker) MarkUnhealthy(providerName string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.unhealthy[providerName] = time.Now().Add(h.cooldown)
}

func (h *HealthTracker) MarkHealthy(providerName string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.unhealthy, providerName)
}
