package config

import "testing"

func TestFlux3MaxVideoFalRoute(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range cfg.Models {
		if m.ID != "flux-3-max-video" {
			continue
		}
		if m.Provider != "fal" || m.ProviderModelID != "blackforestlabs/flux-3-max/text-to-video" {
			t.Fatalf("route = %s/%s", m.Provider, m.ProviderModelID)
		}
		if m.PricePerSecond != 0.17 || m.PricePerSecondByResolution["720p"] != 0.17 ||
			m.PricePerSecondByResolution["1080p"] != 0.29 || !m.SupportsVision {
			t.Fatalf("pricing/capabilities = %#v", m)
		}
		return
	}
	t.Fatal("flux-3-max-video model is missing")
}
