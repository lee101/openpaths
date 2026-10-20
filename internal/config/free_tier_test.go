package config

import "testing"

// TestFreeTierBillsNothingOnEveryHop pins the expensive-direction guarantee of
// the free series: a caller asking for openpaths-free is either served by a
// free model or gets an error — never silently upgraded onto a paid lane. A
// paid model pasted into the chain would charge them for a request they made
// under the free tier.
func TestFreeTierBillsNothingOnEveryHop(t *testing.T) {
	t.Skip("pending config: openpaths/free and the or/*-free ladder lanes; prod WIP test ahead of prod config")
	byID := loadAuditConfig(t)

	free := byID["openpaths/free"]
	if free == nil {
		t.Fatal("openpaths/free missing from config.yaml")
	}
	if free.Provider != "openrouter" || free.ProviderModelID != "stealth/space-bunny-alpha" {
		t.Errorf("openpaths/free routes to %s/%s, want openrouter/stealth/space-bunny-alpha",
			free.Provider, free.ProviderModelID)
	}
	if free.InputPricePer1M != 0 || free.OutputPricePer1M != 0 {
		t.Errorf("openpaths/free bills %v/%v, want 0/0", free.InputPricePer1M, free.OutputPricePer1M)
	}
	if len(free.FallbackModels) == 0 {
		t.Fatal("openpaths/free has no fallback ladder")
	}
	for _, hop := range free.FallbackModels {
		m := byID[hop]
		if m == nil {
			t.Errorf("free tier hop %q is not a configured model", hop)
			continue
		}
		if m.Provider != "openrouter" {
			t.Errorf("free tier hop %q uses provider %q, want the OpenRouter free lane", hop, m.Provider)
		}
		if m.InputPricePer1M != 0 || m.OutputPricePer1M != 0 {
			t.Errorf("free tier hop %q bills %v/%v, want 0/0", hop, m.InputPricePer1M, m.OutputPricePer1M)
		}
	}
}

// The free preview lane is also selectable on its own, including through the
// OpenRouter-style alias callers already use for the other stealth previews.
func TestSpaceBunnyFreePreviewIsSelectable(t *testing.T) {
	t.Skip("pending config: or/space-bunny and openrouter/space-bunny-alpha aliases; prod WIP test ahead of prod config")
	byName := loadAuditConfig(t)

	for _, id := range []string{
		"or/space-bunny",
		"space-bunny",
		"openrouter/space-bunny-alpha",
		"stealth/space-bunny-alpha",
		"openpaths/space-bunny-alpha",
		"openpaths/stealth/space-bunny-alpha",
	} {
		m := byName[id]
		if m == nil {
			t.Errorf("Space Bunny entry %q is missing", id)
			continue
		}
		if m.ProviderModelID != "stealth/space-bunny-alpha" {
			t.Errorf("%s routes to %s, want stealth/space-bunny-alpha", id, m.ProviderModelID)
		}
		if m.InputPricePer1M != 0 || m.OutputPricePer1M != 0 {
			t.Errorf("%s bills %v/%v, want 0/0", id, m.InputPricePer1M, m.OutputPricePer1M)
		}
		if m.ContextWindow != 1000000 || !m.SupportsTools || !m.SupportsVision {
			t.Errorf("%s = ctx %d tools %t vision %t, want the published 1M/true/true",
				id, m.ContextWindow, m.SupportsTools, m.SupportsVision)
		}
	}
}
