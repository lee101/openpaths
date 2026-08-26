package config

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestManifoldGenFirstPartyCatalog(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatal(err)
	}

	byID := map[string]*model.ModelConfig{}
	known := map[string]string{}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		byID[m.ID] = m
		known[m.ID] = m.ID
		for _, alias := range m.Aliases {
			if _, exists := known[alias]; !exists {
				known[alias] = m.ID
			}
		}
	}

	providers := map[string]bool{}
	for _, p := range cfg.Providers {
		providers[p.Name] = true
	}
	if !providers["manifoldgen"] {
		t.Fatal("manifoldgen provider missing from config.yaml")
	}

	want := []struct {
		id        string
		upstream  string
		perSecond float64
		perImage  float64
		perChar1M float64
		vision    bool
	}{
		{id: "kfold-video", upstream: "kfold-video", perSecond: 0.15, vision: true},
		{id: "wan-animate", upstream: "wan-animate", perSecond: 0.15, vision: true},
		{id: "wan-animate-fast", upstream: "wan-animate-fast", perSecond: 0.30, vision: true},
		{id: "wan-animate-xfast", upstream: "wan-animate-xfast", perSecond: 0.60, vision: true},
		{id: "h3-control-video", upstream: "h3-control", perSecond: 0.10, vision: true},
		{id: "remove-video-background", upstream: "remove-video-background", perSecond: 0.0051, vision: true},
		{id: "mg-music", upstream: "mg-music", perImage: 0.35},
		{id: "mg-sfx", upstream: "mg-sfx", perImage: 0.50},
		{id: "mg-tts", upstream: "mg-tts", perChar1M: 50},
		{id: "h3-image", upstream: "h3-image", perImage: 0.30},
		{id: "h3-image-edit", upstream: "h3-image-edit", perImage: 0.35, vision: true},
		{id: "video-dramatize", upstream: "video-dramatize", perSecond: 0.10, vision: true},
	}

	for _, w := range want {
		m, ok := byID[w.id]
		if !ok {
			t.Errorf("%s missing from config.yaml", w.id)
			continue
		}
		if m.Provider != "manifoldgen" {
			t.Errorf("%s provider = %q, want manifoldgen", w.id, m.Provider)
		}
		if m.ProviderModelID != w.upstream {
			t.Errorf("%s routes to %q, want %q", w.id, m.ProviderModelID, w.upstream)
		}
		if m.PricePerSecond != w.perSecond {
			t.Errorf("%s price_per_second = %v, want %v", w.id, m.PricePerSecond, w.perSecond)
		}
		if m.PricePerImage != w.perImage {
			t.Errorf("%s price_per_image = %v, want %v", w.id, m.PricePerImage, w.perImage)
		}
		if m.PricePer1MCharacters != w.perChar1M {
			t.Errorf("%s price_per_1m_characters = %v, want %v", w.id, m.PricePer1MCharacters, w.perChar1M)
		}
		if m.SupportsVision != w.vision {
			t.Errorf("%s supports_vision = %v, want %v", w.id, m.SupportsVision, w.vision)
		}
		if len(m.Aliases) == 0 {
			t.Errorf("%s has no aliases", w.id)
		}
	}

	for alias, id := range map[string]string{
		"kfold":               "kfold-video",
		"character-animation": "wan-animate",
		"music3":              "mg-music",
		"transparent-video":   "remove-video-background",
		"dramatize":           "video-dramatize",
		"video-dramatizer":    "video-dramatize",
	} {
		if got := known[alias]; got != id {
			t.Errorf("alias %q resolves to %q, want %q", alias, got, id)
		}
	}
}
