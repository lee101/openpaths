package config

import "testing"

func TestMiniMaxH3FalRoute(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range cfg.Models {
		if m.ID != "minimax-h3" {
			continue
		}
		if m.Provider != "fal" || m.ProviderModelID != "minimax/h3/text-to-video" {
			t.Fatalf("route = %s/%s", m.Provider, m.ProviderModelID)
		}
		if m.PricePerSecond != 0.13 || !m.SupportsVision {
			t.Fatalf("pricing/capabilities = %#v", m)
		}
		return
	}
	t.Fatal("minimax-h3 model is missing")
}
