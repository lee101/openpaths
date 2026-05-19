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
				h.baseCooldown = 1 * time.Millisecond
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
				h.baseCooldown = 1 * time.Hour
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
	h := NewHealthTracker()

	h.MarkUnhealthy("openai")
	if h.IsHealthy("openai") {
		t.Errorf("expected provider to be unhealthy after MarkUnhealthy")
	}

	h.MarkUnhealthy("openai")
	if h.IsHealthy("openai") {
		t.Errorf("expected provider to still be unhealthy after second MarkUnhealthy")
	}
}

func TestHealthTracker_CooldownRecovery(t *testing.T) {
	h := NewHealthTracker()
	h.baseCooldown = 10 * time.Millisecond

	h.MarkUnhealthy("openai")

	if h.IsHealthy("openai") {
		t.Fatal("expected provider to be unhealthy immediately after MarkUnhealthy")
	}

	time.Sleep(20 * time.Millisecond)

	if !h.IsHealthy("openai") {
		t.Fatal("expected provider to recover after cooldown period")
	}
}

func TestHealthTracker_DefaultCooldown(t *testing.T) {
	h := NewHealthTracker()
	if h.baseCooldown != 30*time.Second {
		t.Errorf("default cooldown = %v, want %v", h.baseCooldown, 30*time.Second)
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

func TestHealthTracker_EscalatingCooldown(t *testing.T) {
	h := NewHealthTracker()
	h.baseCooldown = 10 * time.Millisecond
	h.maxCooldown = 40 * time.Millisecond

	h.MarkUnhealthy("openai")
	if h.failures["openai"] != 1 {
		t.Errorf("failures = %d, want 1", h.failures["openai"])
	}

	h.MarkUnhealthy("openai")
	if h.failures["openai"] != 2 {
		t.Errorf("failures = %d, want 2", h.failures["openai"])
	}

	h.MarkHealthy("openai")
	if h.failures["openai"] != 0 {
		t.Errorf("failures after MarkHealthy = %d, want 0", h.failures["openai"])
	}
}

func TestHealthTracker_ModelProviderKey(t *testing.T) {
	h := NewHealthTracker()
	key := h.ModelProviderKey("google", "gemini-3.5-flash")
	if key != "google:gemini-3.5-flash" {
		t.Errorf("key = %q, want %q", key, "google:gemini-3.5-flash")
	}
}
