package config

import (
	"path/filepath"
	"testing"
)

// TestFalVideoCatalogRoutes pins the Higgsfield-parity video catalog: Kling 3.0,
// Kling 2.6, Veo 3.1, Seedance 2.5, and Seedance 4K must all route through fal
// at the advertised per-second prices so the site pages never underprice us.
func TestFalVideoCatalogRoutes(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	idx := func(id string) int {
		for i := range cfg.Models {
			if cfg.Models[i].ID == id {
				return i
			}
		}
		return -1
	}

	want := []struct {
		id            string
		providerModel string
		price         float64
		vision        bool
	}{
		{"fal-ai/kling-video/v3/pro/text-to-video", "fal-ai/kling-video/v3/pro/text-to-video", 0.168, false},
		{"fal-ai/kling-video/v3/pro/image-to-video", "fal-ai/kling-video/v3/pro/image-to-video", 0.168, true},
		{"fal-ai/kling-video/v3/standard/text-to-video", "fal-ai/kling-video/v3/standard/text-to-video", 0.126, false},
		{"fal-ai/kling-video/v3/standard/image-to-video", "fal-ai/kling-video/v3/standard/image-to-video", 0.126, true},
		{"fal-ai/kling-video/v2.6/pro/text-to-video", "fal-ai/kling-video/v2.6/pro/text-to-video", 0.14, false},
		{"fal-ai/kling-video/v2.6/pro/image-to-video", "fal-ai/kling-video/v2.6/pro/image-to-video", 0.14, true},
		{"fal-ai/veo3.1", "fal-ai/veo3.1", 0.40, false},
		{"fal-ai/veo3.1/image-to-video", "fal-ai/veo3.1/image-to-video", 0.40, true},
		{"fal-ai/veo3.1/reference-to-video", "fal-ai/veo3.1/reference-to-video", 0.40, true},
		{"fal-ai/veo3.1/first-last-frame-to-video", "fal-ai/veo3.1/first-last-frame-to-video", 0.40, true},
		{"fal-ai/veo3.1/fast", "fal-ai/veo3.1/fast", 0.15, false},
		{"fal-ai/veo3.1/fast/image-to-video", "fal-ai/veo3.1/fast/image-to-video", 0.15, true},
		{"seedance-2.5-text-to-video", "bytedance/seedance-2.5/text-to-video", 0.473, false},
		{"seedance-2.5-image-to-video", "bytedance/seedance-2.5/image-to-video", 0.473, true},
		{"seedance-2.5-reference-to-video", "bytedance/seedance-2.5/reference-to-video", 0.473, true},
		{"seedance-2.0-4k-text-to-video", "bytedance/seedance-2.0/text-to-video", 1.5552, false},
	}
	for _, w := range want {
		i := idx(w.id)
		if i < 0 {
			t.Errorf("%s missing from config.yaml", w.id)
			continue
		}
		m := &cfg.Models[i]
		if m.Provider != "fal" || m.ProviderModelID != w.providerModel {
			t.Errorf("%s routes through %s/%s, want fal/%s", w.id, m.Provider, m.ProviderModelID, w.providerModel)
		}
		if m.PricePerSecond != w.price {
			t.Errorf("%s price = %v, want %v", w.id, m.PricePerSecond, w.price)
		}
		if m.SupportsVision != w.vision {
			t.Errorf("%s supports_vision = %v, want %v", w.id, m.SupportsVision, w.vision)
		}
	}

	// Resolution tiers keep higher-resolution calls from billing at the 720p rate.
	checks := []struct {
		id   string
		key  string
		rate float64
	}{
		{"seedance-2.0-text-to-video", "1080p", 0.682},
		{"seedance-2.0-image-to-video", "1080p", 0.682},
		{"seedance-2.0-reference-to-video", "1080p", 0.682},
		{"seedance-2.0-text-to-video", "4k", 1.5552},
		{"seedance-2.5-text-to-video", "480p", 0.2205},
	}
	for _, c := range checks {
		i := idx(c.id)
		if i < 0 {
			t.Errorf("%s missing from config.yaml", c.id)
			continue
		}
		if got := cfg.Models[i].PricePerSecondByResolution[c.key]; got != c.rate {
			t.Errorf("%s[%s] = %v, want %v", c.id, c.key, got, c.rate)
		}
	}
	if i := idx("seedance-2.5-reference-to-video"); i >= 0 && cfg.Models[i].PricePerSecondWithVideoInput != 0.2838 {
		t.Errorf("seedance-2.5 video-input rate = %v, want 0.2838", cfg.Models[i].PricePerSecondWithVideoInput)
	}
}
