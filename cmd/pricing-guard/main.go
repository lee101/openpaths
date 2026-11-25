// Command pricing-guard checks that every model bills at or above its upstream
// cost and that the site advertises exactly what we bill.
//
//	go run ./cmd/pricing-guard                  # offline: recorded costs only
//	go run ./cmd/pricing-guard -live            # refresh costs from public endpoints
//	go run ./cmd/pricing-guard -live -update-ledger   # also rewrite upstream-costs.json
//
// Exit status: 0 clean, 1 critical findings (below cost / advertised != billed),
// 2 the check itself failed. scripts/pricing-guard/pricing_guard.sh runs it on
// a timer and emails the report.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/openpaths/openpaths/internal/config"
	"github.com/openpaths/openpaths/internal/pricingguard"
)

func main() {
	repo := flag.String("repo", ".", "openpaths checkout")
	live := flag.Bool("live", false, "fetch live upstream prices (OpenRouter public API)")
	update := flag.Bool("update-ledger", false, "with -live: write refreshed costs into scripts/pricing-guard/upstream-costs.json")
	out := flag.String("out", "", "write the markdown report here (default stdout)")
	ledgerFlag := flag.String("ledger", "", "upstream cost ledger (default <repo>/scripts/pricing-guard/upstream-costs.json); "+
		"lets a newer guard check an older prod checkout that predates the ledger")
	flag.Parse()
	if err := run(*repo, *ledgerFlag, *live, *update, *out); err != nil {
		fmt.Fprintln(os.Stderr, "pricing-guard:", err)
		os.Exit(2)
	}
}

func run(repo, ledgerOverride string, live, update bool, out string) error {
	p := func(rel string) string { return filepath.Join(repo, rel) }
	cfg, err := config.Load(p("config.yaml"))
	if err != nil {
		return err
	}
	cfgText, err := os.ReadFile(p("config.yaml"))
	if err != nil {
		return err
	}
	ledgerPath := p("scripts/pricing-guard/upstream-costs.json")
	if ledgerOverride != "" {
		ledgerPath = ledgerOverride
	}
	ledger, err := pricingguard.LoadLedger(ledgerPath)
	if err != nil {
		return err
	}
	site, err := os.ReadFile(p("src/data/models.ts"))
	if err != nil {
		return err
	}
	pj, err := os.ReadFile(p("src/data/pricingConfig.generated.json"))
	if err != nil {
		return err
	}
	pricing, err := pricingguard.ParsePricingJSON(pj)
	if err != nil {
		return err
	}
	caps, err := os.ReadFile(p("src/data/capabilities.ts"))
	if err != nil {
		return err
	}

	in := pricingguard.Inputs{
		Models:          cfg.Models,
		OpenRouterCosts: pricingguard.RecordedOpenRouterCosts(string(cfgText)),
		RouteCosts:      map[string]pricingguard.Cost{},
		Site:            pricingguard.ParseSiteModels(string(site)),
		PricingJSON:     pricing,
		FreeIDs:         pricingguard.ParseFreeModelIDs(string(caps)),
	}
	for k, e := range ledger.Routes {
		in.RouteCosts[k] = e.Cost
	}
	title := "openpaths pricing guard (recorded costs)"
	var notes []string
	if live {
		orModels := map[string]string{}
		for _, m := range cfg.Models {
			if m.Provider == "openrouter" && m.ProviderModelID != "" {
				orModels[m.ID] = m.ProviderModelID
			}
		}
		lc, err := pricingguard.FetchLive(pricingguard.HTTPFetcher(&http.Client{Timeout: 60 * time.Second}),
			orModels, in.OpenRouterCosts, ledger, 150*time.Millisecond)
		if err != nil {
			return err
		}
		// Live costs win; recorded ones stay for routes the live sources lack.
		for id, c := range lc.OpenRouter {
			in.OpenRouterCosts[id] = c
		}
		for k, c := range lc.Routes {
			in.RouteCosts[k] = c
		}
		in.Missing, in.Stale = lc.Missing, lc.Stale
		notes = lc.Errors
		title = "openpaths pricing guard (live upstream prices, " + time.Now().UTC().Format("2006-01-02") + ")"
		if update {
			n := pricingguard.UpdateLedger(ledger, lc, time.Now().UTC().Format("2006-01-02"))
			if err := ledger.Save(ledgerPath); err != nil {
				return err
			}
			notes = append(notes, fmt.Sprintf("updated %d ledger entries in %s", n, ledgerPath))
		}
	}

	res := pricingguard.Check(in)
	report := res.Markdown(title)
	if len(notes) > 0 {
		report += "\n## Notes\n\n"
		for _, n := range notes {
			report += "- " + n + "\n"
		}
	}
	if out != "" {
		if err := os.WriteFile(out, []byte(report), 0o644); err != nil {
			return err
		}
	} else {
		fmt.Print(report)
	}
	if len(res.Critical()) > 0 {
		os.Exit(1)
	}
	return nil
}
