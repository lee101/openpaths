package config

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

// OpenRouter adds its platform fee on top of the advertised per-token rate, so
// a model billed at the advertised rate loses money on every call.
const openRouterFee = 0.055

var (
	syncBlockRe   = regexp.MustCompile(`(?s)# --- BEGIN openrouter-sync.*?# --- END openrouter-sync ---`)
	syncEntryRe   = regexp.MustCompile(`(?m)^(?:  # openrouter: in=([0-9.]+) out=([0-9.]+) cache=([0-9.]+)(?: long@[0-9]+ in=([0-9.]+) out=([0-9.]+) cache=([0-9.]+))? per 1M, \+([0-9.]+)% fee\n)?  - id: "([^"]+)"`)
	upstreamPrice = func(s string) float64 { f, _ := strconv.ParseFloat(s, 64); return f }
)

// upstream is OpenRouter's advertised rate for one entry: base in/out/cache,
// the long-context tier (zero when none), and the fee the price was set with.
type upstream struct {
	in, out, cache, longIn, longOut, longCache, fee float64
	known                                           bool
}

func upstreamEntries(text string) map[string]upstream {
	out := map[string]upstream{}
	for _, m := range syncEntryRe.FindAllStringSubmatch(text, -1) {
		if m[1] == "" {
			out[m[8]] = upstream{}
			continue
		}
		p := upstreamPrice
		out[m[8]] = upstream{p(m[1]), p(m[2]), p(m[3]), p(m[4]), p(m[5]), p(m[6]), p(m[7]) / 100, true}
	}
	return out
}

func syncedEntries(t *testing.T) map[string]upstream {
	t.Helper()
	block := syncBlockRe.FindString(readSite(t, "config.yaml"))
	if block == "" {
		t.Fatal("openrouter-sync block missing from config.yaml; run scripts/sync-openrouter-models.py")
	}
	return upstreamEntries(block)
}

func assertCoversFee(t *testing.T, id string, m *model.ModelConfig, up upstream) {
	t.Helper()
	if up.fee < openRouterFee {
		t.Errorf("%s marks up %.1f%%, want at least the %.1f%% OpenRouter fee", id, up.fee*100, openRouterFee*100)
	}
	// Effective rates as billing applies them: an unset cache rate bills cached
	// tokens at the input rate, an unset long rate falls back to the base one.
	or := func(v, fallback float64) float64 {
		if v > 0 {
			return v
		}
		return fallback
	}
	cache := or(m.InputCacheHitPricePer1M, m.InputPricePer1M)
	longIn := or(m.InputPricePer1MLong, m.InputPricePer1M)
	for _, f := range []struct {
		name           string
		billed, upstrm float64
	}{
		{"input", m.InputPricePer1M, up.in},
		{"output", m.OutputPricePer1M, up.out},
		{"cache hit", cache, up.cache},
		{"long input", longIn, up.longIn},
		{"long output", or(m.OutputPricePer1MLong, m.OutputPricePer1M), up.longOut},
		{"long cache hit", or(m.InputCacheHitPricePer1MLong, or(m.InputCacheHitPricePer1M, longIn)), up.longCache},
	} {
		if floor := f.upstrm * (1 + openRouterFee); f.billed+1e-9 < floor {
			t.Errorf("%s %s bills %v/M, below OpenRouter %v/M + fee = %v/M", id, f.name, f.billed, f.upstrm, math.Round(floor*1e4)/1e4)
		}
	}
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
		if !up.known {
			if isPriced(t, m) {
				t.Errorf("%s is priced but has no '# openrouter:' upstream price line", id)
			}
			if id != "openpaths/free" && !strings.HasSuffix(m.ProviderModelID, ":free") && m.ProviderModelID != "openrouter/free" {
				t.Errorf("%s bills $0 but routes to paid OpenRouter model %s", id, m.ProviderModelID)
			}
			continue
		}
		assertCoversFee(t, id, m, up)
	}
}

// TestHandWrittenOpenRouterPricesCoverTheFee: hand-written OpenRouter entries
// carry the upstream rate the sync last saw, and must bill at least that plus
// the fee - the sync raises them, this keeps a manual edit from undercutting.
func TestHandWrittenOpenRouterPricesCoverTheFee(t *testing.T) {
	byID := loadAuditConfig(t)
	text := syncBlockRe.ReplaceAllString(readSite(t, "config.yaml"), "")
	checked := 0
	for id, up := range upstreamEntries(text) {
		if !up.known {
			continue
		}
		m := byID[id]
		if m == nil || m.Provider != "openrouter" {
			t.Errorf("%s has an OpenRouter upstream price line but is not an OpenRouter entry", id)
			continue
		}
		checked++
		assertCoversFee(t, id, m, up)
	}
	if checked < 10 {
		t.Fatalf("only %d hand-written OpenRouter entries carry upstream prices; run scripts/sync-openrouter-models.py", checked)
	}
}
