package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/pricingguard"
)

// TestBilledPricesCoverRecordedUpstreamCost enforces the owner rule offline:
// every model bills at least what its upstream charges us (OpenRouter rate +
// fee from the "# openrouter:" lines, direct providers from
// scripts/pricing-guard/upstream-costs.json, every scheduled tariff and
// long-context tier included), the model cards and /pricing data advertise
// exactly what config.yaml bills, and FREE_MODEL_IDS is exactly the $0 set.
// scripts/pricing-guard/pricing_guard.sh re-checks against live prices.
func TestBilledPricesCoverRecordedUpstreamCost(t *testing.T) {
	root := filepath.Join("..", "..")
	cfg, err := Load(filepath.Join(root, "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := pricingguard.LoadLedger(filepath.Join(root, "scripts", "pricing-guard", "upstream-costs.json"))
	if err != nil {
		t.Fatal(err)
	}
	pj, err := os.ReadFile(filepath.Join(root, "src", "data", "pricingConfig.generated.json"))
	if err != nil {
		t.Fatal(err)
	}
	pricing, err := pricingguard.ParsePricingJSON(pj)
	if err != nil {
		t.Fatal(err)
	}
	routes := map[string]pricingguard.Cost{}
	for k, e := range ledger.Routes {
		if e.In <= 0 && e.Out <= 0 {
			t.Errorf("ledger route %s has no recorded cost; run go run ./cmd/pricing-guard -live -update-ledger", k)
		}
		routes[k] = e.Cost
	}
	or := pricingguard.RecordedOpenRouterCosts(readSite(t, "config.yaml"))
	if len(or) < 30 || len(routes) < 30 {
		t.Fatalf("parsed only %d OpenRouter and %d direct recorded costs; parser or ledger broke", len(or), len(routes))
	}
	res := pricingguard.Check(pricingguard.Inputs{
		Models:          cfg.Models,
		OpenRouterCosts: or,
		RouteCosts:      routes,
		Site:            pricingguard.ParseSiteModels(readSite(t, "src/data/models.ts")),
		PricingJSON:     pricing,
		FreeIDs:         pricingguard.ParseFreeModelIDs(readSite(t, "src/data/capabilities.ts")),
		Now:             time.Now(),
	})
	for _, f := range res.Critical() {
		t.Errorf("%s %s %s: billed %v, want %v (%s)", f.Kind, f.Model, f.Field, f.Billed, f.Want, f.Detail)
	}
	if res.Checked < 150 {
		t.Errorf("only %d models checked against an upstream cost", res.Checked)
	}
	t.Logf("%d models checked against recorded upstream cost; %d token-priced models unverified", res.Checked, len(res.Unverified))
}
