package config

import "testing"

func TestGemini38LatestAndCompatibilityRoutes(t *testing.T) {
	models := loadAuditConfig(t)
	for _, id := range []string{
		"gemini-3.8-flash",
		"gemini-3.7-flash",
		"gemini-latest",
		"gemini-flash-latest",
		"flash-latest",
		"gemini-3.6-flash",
		"gemini-3.5-flash",
	} {
		m, ok := models[id]
		if !ok {
			t.Fatalf("model route %q is missing", id)
		}
		if m.Provider != "google" || m.ProviderModelID != "gemini-3.8-flash" {
			t.Errorf("%s routes to %s/%s, want google/gemini-3.8-flash", id, m.Provider, m.ProviderModelID)
		}
		wantInput, wantOutput := .75, 3.75
		if m.InputPricePer1M != wantInput || m.OutputPricePer1M != wantOutput {
			t.Errorf("%s bills %v/%v, want %v/%v", id, m.InputPricePer1M, m.OutputPricePer1M, wantInput, wantOutput)
		}
	}
}
