package config

import "testing"

func TestRelightConfig(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]float64{"480p": .055, "768p": .088, "1080p": .176, "2k": .352}
	for _, m := range cfg.Models {
		if m.ID != "minimax-h3-max-relight" {
			continue
		}
		if m.ProviderModelID != "minimax/h3-max/relight" || m.Provider != "fal" || m.PricePerSecond != .352 || len(m.PricePerSecondByResolution) != len(want) {
			t.Fatalf("unexpected relight config: %#v", m)
		}
		for k, v := range want {
			if m.PricePerSecondByResolution[k] != v {
				t.Fatalf("%s price %v, want %v", k, m.PricePerSecondByResolution[k], v)
			}
		}
		return
	}
	t.Fatal("missing relight")
}
