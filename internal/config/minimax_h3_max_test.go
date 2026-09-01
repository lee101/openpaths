package config

import "testing"

func TestMiniMaxH3MaxFalRoute(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"minimax-h3-max":                "minimax/h3-max/text-to-video",
		"minimax-h3-max-image-to-video": "minimax/h3-max/image-to-video",
	}
	for id, providerModel := range want {
		found := false
		for _, m := range cfg.Models {
			if m.ID != id {
				continue
			}
			found = true
			if m.Provider != "fal" || m.ProviderModelID != providerModel {
				t.Fatalf("%s route = %s/%s", id, m.Provider, m.ProviderModelID)
			}
			if m.PricePerSecond != 0.08 || m.PricePerSecondByResolution["480p"] != 0.05 ||
				m.PricePerSecondByResolution["768p"] != 0.08 || !m.SupportsVision {
				t.Fatalf("%s pricing/capabilities = %#v", id, m)
			}
		}
		if !found {
			t.Fatalf("%s model is missing", id)
		}
	}
}
