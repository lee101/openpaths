package config

import "testing"

func TestRecastConfig(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range cfg.Models {
		if m.ID == "minimax-h3-max-recast" {
			if m.ProviderModelID != "minimax/h3-max/recast" || m.Provider != "fal" || m.PricePerSecond != .45 || m.PricePerSecondByResolution["768p"] != .30 || m.PricePerSecondByResolution["1080p"] != .45 || m.PricePerInputImage != 0 {
				t.Fatalf("unexpected recast config: %#v", m)
			}
			return
		}
	}
	t.Fatal("missing recast")
}
