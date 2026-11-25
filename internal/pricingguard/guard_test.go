package pricingguard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func kinds(r *Result) map[string]int {
	out := map[string]int{}
	for _, f := range r.Findings {
		out[f.Kind+" "+f.Model+" "+f.Field]++
	}
	return out
}

var now = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

func TestRecordedOpenRouterCosts(t *testing.T) {
	got := RecordedOpenRouterCosts(string(fixture(t, "config_snippet.yaml")))
	if len(got) != 2 {
		t.Fatalf("parsed %d entries, want 2: %+v", len(got), got)
	}
	if c := got["or/paid"]; c.In != 1 || c.Out != 4 || c.Cache != 0.1 || c.Fee != OpenRouterFee {
		t.Errorf("or/paid = %+v", c)
	}
	if c := got["or/peaky"]; c.LongThreshold != 200000 || c.LongIn != 4 || c.LongOut != 8 {
		t.Errorf("or/peaky long tier = %+v", c)
	}
}

func TestBelowCostIncludesFeeCacheAndLongTier(t *testing.T) {
	cost := Cost{In: 1, Out: 4, Cache: 0.1, LongThreshold: 200000, LongIn: 2, LongOut: 8, Fee: OpenRouterFee}
	models := []model.ModelConfig{
		// exactly rate + fee, long tier configured: clean
		{ID: "ok", Provider: "openrouter", InputPricePer1M: 1.055, OutputPricePer1M: 4.22, InputCacheHitPricePer1M: 0.1055,
			LongContextThreshold: 200000, InputPricePer1MLong: 2.11, OutputPricePer1MLong: 8.44},
		// billed at the listed rate: loses the fee on every field
		{ID: "at-list", Provider: "openrouter", InputPricePer1M: 1, OutputPricePer1M: 4, InputCacheHitPricePer1M: 0.1,
			LongContextThreshold: 200000, InputPricePer1MLong: 2, OutputPricePer1MLong: 8},
		// no long tier: base rates are billed on long prompts, below the long cost
		{ID: "no-long", Provider: "openrouter", InputPricePer1M: 1.055, OutputPricePer1M: 4.22},
	}
	res := Check(Inputs{Models: models, OpenRouterCosts: map[string]Cost{"ok": cost, "at-list": cost, "no-long": cost}, Now: now})
	k := kinds(res)
	for _, want := range []string{
		"BELOW_COST at-list input", "BELOW_COST at-list output", "BELOW_COST at-list cache_hit",
		"BELOW_COST at-list long_input", "BELOW_COST at-list long_output",
		"BELOW_COST no-long long_input", "BELOW_COST no-long long_output",
	} {
		if k[want] == 0 {
			t.Errorf("missing finding %q in %v", want, k)
		}
	}
	for key := range k {
		if strings.Contains(key, " ok ") {
			t.Errorf("unexpected finding for a model billed at cost+fee: %s", key)
		}
	}
	// no-long's cache falls back to its input rate (1.055), above the cache cost.
	if k["BELOW_COST no-long cache_hit"] != 0 {
		t.Errorf("unset cache rate bills at the input rate and must not be flagged")
	}
	if res.Checked != 3 {
		t.Errorf("checked = %d, want 3", res.Checked)
	}
	for _, f := range res.Findings {
		if f.Model == "at-list" && f.Field == "input" && f.Want != 1.055 {
			t.Errorf("floor = %v, want 1.055", f.Want)
		}
	}
}

func TestDirectRouteCostKeyedByUpstream(t *testing.T) {
	// Two catalogue ids routed to the same upstream share its cost.
	models := []model.ModelConfig{
		{ID: "lab-new", Provider: "lab", ProviderModelID: "m1", InputPricePer1M: 2, OutputPricePer1M: 6},
		{ID: "lab-old-alias", Provider: "lab", ProviderModelID: "m1", InputPricePer1M: 1.25, OutputPricePer1M: 2.5},
		{ID: "lab-unknown", Provider: "lab", ProviderModelID: "m2", InputPricePer1M: 1, OutputPricePer1M: 1},
		{ID: "lab-image", Provider: "lab", ProviderModelID: "img", PricePerImage: 0.01},
	}
	res := Check(Inputs{Models: models, RouteCosts: map[string]Cost{"lab/m1": {In: 2, Out: 6}}, Now: now})
	k := kinds(res)
	if k["BELOW_COST lab-old-alias input"] == 0 || k["BELOW_COST lab-old-alias output"] == 0 {
		t.Errorf("alias routed to a dearer upstream not flagged: %v", k)
	}
	if k["BELOW_COST lab-new input"] != 0 {
		t.Errorf("model billed at direct cost (no fee) flagged: %v", k)
	}
	if len(res.Unverified) != 1 || !strings.HasPrefix(res.Unverified[0], "lab-unknown") {
		t.Errorf("unverified = %v, want only lab-unknown (asset-priced models are skipped)", res.Unverified)
	}
}

func TestScheduledTariffs(t *testing.T) {
	sched := model.ModelConfig{ID: "sched", Provider: "lab", ProviderModelID: "s",
		InputPricePer1M: 0.1, OutputPricePer1M: 0.4,
		ScheduledTokenPricing: &model.ScheduledTokenPricing{
			EffectiveAt:         "2026-09-01T00:00:00Z",
			PeakWindowsUTC:      []model.UTCPriceWindow{{Start: "01:00", End: "04:00"}},
			PeakInputPricePer1M: 0.15, PeakOutputPricePer1M: 0.9,
			OffPeakInputPricePer1M: 0.2, OffPeakOutputPricePer1M: 0.8,
		}}
	ts := Tariffs(&sched, now)
	if len(ts) != 2 || ts[0].Label != "off_peak_" || ts[1].Label != "peak_" {
		t.Fatalf("tariffs after effective_at = %+v, want off-peak and peak only", ts)
	}
	if len(Tariffs(&sched, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC))) != 3 {
		t.Errorf("before effective_at the base tariff must be checked too")
	}
	if in, _, out := HeadlineRates(&sched, now); in != 0.2 || out != 0.8 {
		t.Errorf("headline = %v/%v, want the off-peak 0.2/0.8 in force", in, out)
	}
	// Cost 0.18/0.85: off-peak output 0.8 and peak input 0.15 are both short.
	res := Check(Inputs{Models: []model.ModelConfig{sched}, RouteCosts: map[string]Cost{"lab/s": {In: 0.18, Out: 0.85}}, Now: now})
	k := kinds(res)
	if k["BELOW_COST sched off_peak_output"] == 0 || k["BELOW_COST sched peak_input"] == 0 || len(k) != 2 {
		t.Errorf("findings = %v, want off_peak_output and peak_input", k)
	}
}

func TestAdvertisedMustEqualBilled(t *testing.T) {
	site := ParseSiteModels(string(fixture(t, "models.ts")))
	if len(site) != 3 || !site["direct-1"].HasIn || site["direct-1"].In != 0.9 {
		t.Fatalf("parsed site = %+v (both quote styles must parse)", site)
	}
	pricing, err := ParsePricingJSON([]byte(`[{"id":"or/paid","input":1.055,"output":4.22,"cacheRead":0.2}]`))
	if err != nil {
		t.Fatal(err)
	}
	models := []model.ModelConfig{
		{ID: "or/paid", Provider: "openrouter", InputPricePer1M: 1.055, OutputPricePer1M: 4.22, InputCacheHitPricePer1M: 0.1055},
		{ID: "direct-1", Provider: "lab", InputPricePer1M: 1.8, OutputPricePer1M: 4},
		{ID: "sched", Provider: "lab", InputPricePer1M: 0.1, OutputPricePer1M: 0.4, ScheduledTokenPricing: &model.ScheduledTokenPricing{
			EffectiveAt: "2026-01-01T00:00:00Z", OffPeakInputPricePer1M: 0.2, OffPeakOutputPricePer1M: 0.8}},
	}
	res := Check(Inputs{Models: models, Site: site, PricingJSON: pricing, Now: now})
	k := kinds(res)
	if k["AD_MISMATCH direct-1 input"] != 1 {
		t.Errorf("site advertising half the billed input not flagged: %v", k)
	}
	if k["AD_MISMATCH or/paid cache_hit"] != 1 {
		t.Errorf("/pricing cache rate mismatch not flagged: %v", k)
	}
	if len(k) != 2 {
		t.Errorf("findings = %v; the scheduled model's card shows the off-peak rate in force and must match", k)
	}
}

func TestFreeRoutesAndFreeList(t *testing.T) {
	models := []model.ModelConfig{
		{ID: "or/free", Provider: "openrouter", ProviderModelID: "acme/free:free"},
		{ID: "or/was-free", Provider: "openrouter", ProviderModelID: "acme/was-free"},
		{ID: "or/paid", Provider: "openrouter", ProviderModelID: "acme/paid", InputPricePer1M: 2, OutputPricePer1M: 5},
	}
	res := Check(Inputs{
		Models:          models,
		OpenRouterCosts: map[string]Cost{"or/free": {}, "or/was-free": {In: 0.5, Out: 1, Fee: OpenRouterFee}},
		FreeIDs:         ParseFreeModelIDs(string(fixture(t, "capabilities.ts"))),
		Now:             now,
	})
	k := kinds(res)
	for _, want := range []string{"FREE_ROUTE_PAID or/was-free ", "FREE_LIST or/ghost ", "FREE_LIST or/was-free "} {
		if k[want] == 0 {
			t.Errorf("missing %q in %v", want, k)
		}
	}
	if k["FREE_LIST or/free "] != 0 || k["FREE_ROUTE_PAID or/free "] != 0 {
		t.Errorf("genuinely free advertised route flagged: %v", k)
	}
	if len(res.Critical()) != len(res.Findings) {
		t.Errorf("all of these break the owner rule and must be critical")
	}
}

func TestToCostPeakAndLong(t *testing.T) {
	cat, err := ParseCatalog(fixture(t, "catalog.json"))
	if err != nil {
		t.Fatal(err)
	}
	c := cat["acme/peaky"].ToCost(OpenRouterFee, "x")
	if c.In != 2 || c.Out != 3 {
		t.Errorf("time-of-day override must raise the base to the peak: %+v", c)
	}
	if c.LongThreshold != 200000 || c.LongIn != 4 || c.LongOut != 8 {
		t.Errorf("long tier = %+v", c)
	}
	if _, err := ParseCatalog([]byte(`{"data":[]}`)); err == nil {
		t.Errorf("empty catalog must be an error, not a clean run")
	}
}

func fakeFetcher(t *testing.T, files map[string]string) Fetcher {
	return func(u string) ([]byte, error) {
		name, ok := files[u]
		if !ok {
			return nil, fmt.Errorf("unexpected fetch %s", u)
		}
		return fixture(t, name), nil
	}
}

func TestFetchLiveAndUpdateLedger(t *testing.T) {
	get := fakeFetcher(t, map[string]string{
		CatalogURL:                               "catalog.json",
		fmt.Sprintf(EndpointsURL, "lab/model-1"): "endpoints_lab.json",
	})
	ledger := &Ledger{Routes: map[string]*LedgerEntry{
		"lab/m1":     {Cost: Cost{In: 1, Out: 4, Cache: 0.1, LongThreshold: 272000, LongIn: 2, LongOut: 6, LongCache: 0.2}, Ref: &LedgerRef{OpenRouter: "lab/model-1", Tag: "lab"}},
		"lab/stale":  {Cost: Cost{In: 0.4, Out: 1}, Ref: &LedgerRef{OpenRouter: "lab/model-1", Tag: "lab/flex"}},
		"lab/badtag": {Cost: Cost{In: 1, Out: 1}, Ref: &LedgerRef{OpenRouter: "lab/model-1", Tag: "lab/priority"}},
		"lab/manual": {Cost: Cost{In: 3, Out: 3}},
	}}
	recorded := map[string]Cost{"or/paid": {In: 0.8, Out: 4, Cache: 0.1, Fee: OpenRouterFee}}
	lc, err := FetchLive(get, map[string]string{"or/paid": "acme/paid", "or/gone": "acme/withdrawn"}, recorded, ledger, 0)
	if err != nil {
		t.Fatal(err)
	}
	if c := lc.OpenRouter["or/paid"]; c.In != 1 || c.Out != 4 || c.Fee != OpenRouterFee {
		t.Errorf("live or/paid = %+v", c)
	}
	if !lc.Missing["acme/withdrawn"] {
		t.Errorf("withdrawn model not reported missing")
	}
	if c := lc.Routes["lab/m1"]; c.In != 1 || c.LongIn != 2 || c.Fee != 0 {
		t.Errorf("live lab/m1 = %+v", c)
	}
	if len(lc.Errors) != 1 || !strings.Contains(lc.Errors[0], "lab/priority") {
		t.Errorf("errors = %v, want the unknown tag reported", lc.Errors)
	}
	stale := strings.Join(lc.Stale, "\n")
	if !strings.Contains(stale, "or/paid") || !strings.Contains(stale, "lab/stale") || strings.Contains(stale, "lab/m1") {
		t.Errorf("stale = %v", lc.Stale)
	}
	if n := UpdateLedger(ledger, lc, "2026-11-25"); n != 2 {
		t.Errorf("updated %d entries, want 2 (m1 re-verified, stale refreshed)", n)
	}
	if e := ledger.Routes["lab/stale"]; e.In != 0.5 || e.Out != 2 || e.Verified != "2026-11-25" {
		t.Errorf("stale entry after update = %+v", e)
	}
	if e := ledger.Routes["lab/manual"]; e.In != 3 || e.Verified != "" {
		t.Errorf("hand-maintained entry touched: %+v", e)
	}
}

func TestMarkdownTable(t *testing.T) {
	res := &Result{Findings: []Finding{{Severity: Critical, Kind: "BELOW_COST", Model: "m", Field: "input", Billed: 1, Want: 2, Detail: "d"}}, Checked: 1}
	md := res.Markdown("t")
	if !strings.Contains(md, "| CRITICAL | BELOW_COST | m | input | 1 | 2 | d |") || !strings.Contains(md, "Critical: 1") {
		t.Errorf("markdown = %s", md)
	}
}
