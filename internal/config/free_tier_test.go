package config

import "testing"

// TestFreeTierBillsNothingOnEveryHop pins the expensive-direction guarantee of
// the free series: a caller asking for openpaths/free is either served by a
// free model or gets an error — never silently upgraded onto a paid lane. A
// paid model pasted into the chain would charge them for a request they made
// under the free tier.
func TestFreeTierBillsNothingOnEveryHop(t *testing.T) {
	byID := loadAuditConfig(t)

	free := byID["openpaths/free"]
	if free == nil {
		t.Fatal("openpaths/free missing from config.yaml")
	}
	if free.Provider != "openrouter" {
		t.Errorf("openpaths/free routes to provider %q, want the OpenRouter free lane", free.Provider)
	}
	if isPriced(t, free) || len(free.FallbackProviders) != 0 {
		t.Errorf("openpaths/free is priced or has provider fallbacks: %+v", free)
	}
	if len(free.FallbackModels) == 0 {
		t.Fatal("openpaths/free has no fallback ladder")
	}
	for _, alias := range []string{"free", "openpaths-free", "free-latest", "openpaths-auto-free"} {
		if byID[alias] != free {
			t.Errorf("alias %q does not reach openpaths/free", alias)
		}
	}
	for i, hop := range free.FallbackModels {
		m := byID[hop]
		if m == nil {
			t.Errorf("free tier hop %q is not a configured model", hop)
			continue
		}
		if m.Provider != "openrouter" {
			t.Errorf("free tier hop %q uses provider %q, want the OpenRouter free lane", hop, m.Provider)
		}
		if isPriced(t, m) || len(m.FallbackModels) != 0 || len(m.FallbackProviders) != 0 {
			t.Errorf("free tier hop %q is priced or falls back further: %+v", hop, m)
		}
		if last := i == len(free.FallbackModels)-1; last != (m.ProviderModelID == "openrouter/free") {
			t.Errorf("hop %d %q: OpenRouter's own free router must be the last hop and only there", i, hop)
		}
	}
}
