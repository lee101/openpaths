package config

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// OpenRouter adds its platform fee on top of the advertised per-token rate, so
// a model billed at the advertised rate loses money on every call.
const openRouterFee = 0.055

var (
	syncBlockRe   = regexp.MustCompile(`(?s)# --- BEGIN openrouter-sync.*?# --- END openrouter-sync ---`)
	syncEntryRe   = regexp.MustCompile(`(?m)^(?:  # openrouter: in=([0-9.]+) out=([0-9.]+) cache=([0-9.]+) per 1M, \+([0-9.]+)% fee\n)?  - id: "([^"]+)"`)
	upstreamPrice = func(s string) float64 { f, _ := strconv.ParseFloat(s, 64); return f }
)

func syncedEntries(t *testing.T) map[string][4]float64 {
	t.Helper()
	block := syncBlockRe.FindString(readSite(t, "config.yaml"))
	if block == "" {
		t.Fatal("openrouter-sync block missing from config.yaml; run scripts/sync-openrouter-models.py")
	}
	out := map[string][4]float64{}
	for _, m := range syncEntryRe.FindAllStringSubmatch(block, -1) {
		if m[1] == "" {
			out[m[5]] = [4]float64{-1}
			continue
		}
		out[m[5]] = [4]float64{upstreamPrice(m[1]), upstreamPrice(m[2]), upstreamPrice(m[3]), upstreamPrice(m[4]) / 100}
	}
	return out
}

// TestOpenRouterSyncedPricesCoverTheFee: every paid model the sync pulls in
// bills at least OpenRouter's advertised rate plus the fee, and every
// unpriced one is a genuinely free OpenRouter lane.
func TestOpenRouterSyncedPricesCoverTheFee(t *testing.T) {
	byID := loadAuditConfig(t)
	entries := syncedEntries(t)
	if len(entries) < 3 {
		t.Fatalf("openrouter-sync block has %d entries, want the free ladder and paid models", len(entries))
	}
	for id, up := range entries {
		m := byID[id]
		if m == nil || m.ID != id {
			t.Errorf("synced entry %s did not load from config.yaml", id)
			continue
		}
		if m.Provider != "openrouter" {
			t.Errorf("%s provider = %q, want openrouter", id, m.Provider)
		}
		if up[0] < 0 {
			if isPriced(t, m) {
				t.Errorf("%s is priced but has no '# openrouter:' upstream price line", id)
			}
			if id != "openpaths/free" && !strings.HasSuffix(m.ProviderModelID, ":free") && m.ProviderModelID != "openrouter/free" {
				t.Errorf("%s bills $0 but routes to paid OpenRouter model %s", id, m.ProviderModelID)
			}
			continue
		}
		if up[3] < openRouterFee {
			t.Errorf("%s marks up %.1f%%, want at least the %.1f%% OpenRouter fee", id, up[3]*100, openRouterFee*100)
		}
		for _, f := range []struct {
			name           string
			billed, upstrm float64
		}{
			{"input", m.InputPricePer1M, up[0]},
			{"output", m.OutputPricePer1M, up[1]},
			{"cache hit", m.InputCacheHitPricePer1M, up[2]},
		} {
			if floor := f.upstrm * (1 + openRouterFee); f.billed+1e-9 < floor {
				t.Errorf("%s %s bills %v/M, below OpenRouter %v/M + fee = %v/M", id, f.name, f.billed, f.upstrm, math.Round(floor*1e4)/1e4)
			}
		}
	}
}
