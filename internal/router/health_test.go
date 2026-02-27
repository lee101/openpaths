package router

import (
	"testing"
	"time"
)

func TestHealthTracker_IsHealthy(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(h *HealthTracker)
		providerName string
		want         bool
	}{
		{
			name:         "unknown provider is healthy by default",
			setup:        func(h *HealthTracker) {},
			providerName: "never-seen-before",
			want:         true,
		},
		{
			name: "provider marked unhealthy is not healthy",
			setup: func(h *HealthTracker) {
				h.MarkUnhealthy("openai")
			},
			providerName: "openai",
			want:         false,
		},
		{
			name: "other provider is still healthy when one is marked unhealthy",
			setup: func(h *HealthTracker) {
				h.MarkUnhealthy("openai")
			},
			providerName: "anthropic",
			want:         true,
		},
		{
			name: "provider recovers after cooldown expires",
			setup: func(h *HealthTracker) {
				// Set cooldown to a very short duration so it expires immediately.
				h.cooldown = 1 * time.Millisecond
				h.MarkUnhealthy("openai")
				time.Sleep(5 * time.Millisecond)
			},
			providerName: "openai",
			want:         true,
		},
		{
			name: "MarkHealthy immediately recovers provider",
			setup: func(h *HealthTracker) {
				h.MarkUnhealthy("openai")
				h.MarkHealthy("openai")
			},
			providerName: "openai",
			want:         true,
		},
		{
			name: "MarkHealthy on already healthy provider is a no-op",
			setup: func(h *HealthTracker) {
				h.MarkHealthy("openai")
			},
			providerName: "openai",
			want:         true,
		},
		{
			name: "provider remains unhealthy before cooldown expires",
			setup: func(h *HealthTracker) {
				h.cooldown = 1 * time.Hour
				h.MarkUnhealthy("openai")
			},
			providerName: "openai",
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHealthTracker()
			tt.setup(h)

			got := h.IsHealthy(tt.providerName)
			if got != tt.want {
				t.Errorf("IsHealthy(%q) = %v, want %v", tt.providerName, got, tt.want)
			}
		})
	}
}

func TestHealthTracker_MarkUnhealthy(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
	}{
		{
			name:         "marks a new provider as unhealthy",
			providerName: "openai",
		},
		{
			name:         "marks an already unhealthy provider again",
			providerName: "openai",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHealthTracker()

			h.MarkUnhealthy(tt.providerName)

			if h.IsHealthy(tt.providerName) {
				t.Errorf("expected provider %q to be unhealthy after MarkUnhealthy", tt.providerName)
			}

			// Mark it again; should not panic or change behavior.
			h.MarkUnhealthy(tt.providerName)

			if h.IsHealthy(tt.providerName) {
				t.Errorf("expected provider %q to still be unhealthy after second MarkUnhealthy", tt.providerName)
			}
		})
	}
}

func TestHealthTracker_CooldownRecovery(t *testing.T) {
	h := NewHealthTracker()
	h.cooldown = 10 * time.Millisecond

	h.MarkUnhealthy("openai")

	if h.IsHealthy("openai") {
		t.Fatal("expected provider to be unhealthy immediately after MarkUnhealthy")
	}

	// Wait for cooldown to expire.
	time.Sleep(20 * time.Millisecond)

	if !h.IsHealthy("openai") {
		t.Fatal("expected provider to recover after cooldown period")
	}
}

func TestHealthTracker_DefaultCooldown(t *testing.T) {
	h := NewHealthTracker()
	if h.cooldown != 30*time.Second {
		t.Errorf("default cooldown = %v, want %v", h.cooldown, 30*time.Second)
	}
}

func TestHealthTracker_MultipleProviders(t *testing.T) {
	h := NewHealthTracker()

	h.MarkUnhealthy("openai")
	h.MarkUnhealthy("anthropic")

	if h.IsHealthy("openai") {
		t.Error("expected openai to be unhealthy")
	}
	if h.IsHealthy("anthropic") {
		t.Error("expected anthropic to be unhealthy")
	}
	if !h.IsHealthy("google") {
		t.Error("expected google (never marked) to be healthy")
	}

	h.MarkHealthy("openai")

	if !h.IsHealthy("openai") {
		t.Error("expected openai to be healthy after MarkHealthy")
	}
	if h.IsHealthy("anthropic") {
		t.Error("expected anthropic to still be unhealthy")
	}
}
