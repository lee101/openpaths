package config

import "testing"

func TestMiniMaxH3MaxFalRoute(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range cfg.Models {
		if m.ID != "minimax-h3-max" {
			continue
		}
		if m.Provider != "fal" || m.ProviderModelID != "minimax/h3-max/text-to-video" {
			t.Fatalf("route = %s/%s", m.Provider, m.ProviderModelID)
		}
		if m.PricePerSecond != 0.08 || m.PricePerSecondByResolution["480p"] != 0.05 ||
			m.PricePerSecondByResolution["768p"] != 0.08 || !m.SupportsVision {
			t.Fatalf("pricing/capabilities = %#v", m)
		}
		return
	}
	t.Fatal("minimax-h3-max model is missing")
}
