// Package pricingguard checks the owner rule for openpaths pricing: what we
// bill for a model is at least what the upstream provider charges us, and what
// the site advertises equals what we bill.
//
// Inputs are all in-repo so the check runs offline in `make test`:
//   - billed prices: config.yaml (model.ModelConfig)
//   - advertised prices: src/data/models.ts, src/data/pricingConfig.generated.json,
//     and FREE_MODEL_IDS in src/data/capabilities.ts
//   - recorded upstream costs: the "# openrouter: in=.. out=.. cache=.." comment
//     the OpenRouter sync writes above each OpenRouter entry (charged plus
//     OpenRouter's fee), and scripts/pricing-guard/upstream-costs.json for
//     direct providers, keyed by "<provider>/<provider_model_id>".
//
// live.go refreshes those costs from public endpoints (OpenRouter's catalog and
// per-model endpoint listings) for the recurring monitor.
package pricingguard

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

// OpenRouterFee is the platform fee OpenRouter adds on top of a model's listed
// per-token rate; a model billed at the listed rate loses money on every call.
const OpenRouterFee = 0.055

const eps = 1e-9

// Cost is what one upstream route charges us, per 1M tokens, before any fee.
type Cost struct {
	In            float64 `json:"in"`
	Out           float64 `json:"out"`
	Cache         float64 `json:"cache,omitempty"`
	LongThreshold int     `json:"long_threshold,omitempty"`
	LongIn        float64 `json:"long_in,omitempty"`
	LongOut       float64 `json:"long_out,omitempty"`
	LongCache     float64 `json:"long_cache,omitempty"`
	// Fee is a fractional surcharge on top of the rates (OpenRouter: 0.055).
	Fee    float64 `json:"fee,omitempty"`
	Source string  `json:"source,omitempty"`
}

func (c Cost) equal(o Cost) bool {
	return near(c.In, o.In) && near(c.Out, o.Out) && near(c.Cache, o.Cache) &&
		near(c.LongIn, o.LongIn) && near(c.LongOut, o.LongOut) && near(c.LongCache, o.LongCache)
}

func near(a, b float64) bool { return math.Abs(a-b) <= 1e-9+1e-6*math.Max(math.Abs(a), math.Abs(b)) }

// LedgerRef says where the monitor can re-read a direct route's price live:
// the OpenRouter model id and the endpoint tag of the serving provider's own
// standard tier (e.g. "anthropic", "xai", "google-ai-studio"), from
// GET https://openrouter.ai/api/v1/models/<id>/endpoints.
type LedgerRef struct {
	OpenRouter string `json:"openrouter"`
	Tag        string `json:"tag"`
}

// LedgerEntry is one recorded direct-provider cost.
type LedgerEntry struct {
	Cost
	Verified string     `json:"verified,omitempty"`
	Ref      *LedgerRef `json:"ref,omitempty"`
	Note     string     `json:"note,omitempty"`
}

// Ledger is scripts/pricing-guard/upstream-costs.json.
type Ledger struct {
	Comment []string                `json:"_comment,omitempty"`
	Routes  map[string]*LedgerEntry `json:"routes"`
}

func LoadLedger(path string) (*Ledger, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var l Ledger
	if err := json.Unmarshal(raw, &l); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if l.Routes == nil {
		l.Routes = map[string]*LedgerEntry{}
	}
	return &l, nil
}

func (l *Ledger) Save(path string) error {
	raw, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

// RouteKey identifies an upstream route: the cost depends on where a request is
// sent, not on our catalogue id (claude-opus-4-8 routes to claude-opus-5-5).
func RouteKey(m *model.ModelConfig) string { return m.Provider + "/" + m.ProviderModelID }

var (
	syncBlockRe = regexp.MustCompile(`(?s)# --- BEGIN openrouter-sync.*?# --- END openrouter-sync ---`)
	recordedRe  = regexp.MustCompile(`(?m)^  # openrouter: in=([0-9.]+) out=([0-9.]+) cache=([0-9.]+)(?: long@([0-9]+) in=([0-9.]+) out=([0-9.]+) cache=([0-9.]+))? per 1M, \+([0-9.]+)% fee\n  - id: "?([^"\s]+)"?`)
	syncIDRe    = regexp.MustCompile(`(?m)^  - id: "?([^"\s]+)"?`)
)

func atof(s string) float64 { f, _ := strconv.ParseFloat(s, 64); return f }

// RecordedOpenRouterCosts parses the "# openrouter:" lines from config.yaml
// text: model id -> OpenRouter's listed rate and the fee.
func RecordedOpenRouterCosts(configText string) map[string]Cost {
	out := map[string]Cost{}
	for _, m := range recordedRe.FindAllStringSubmatch(configText, -1) {
		out[m[9]] = Cost{
			In: atof(m[1]), Out: atof(m[2]), Cache: atof(m[3]),
			LongThreshold: int(atof(m[4])), LongIn: atof(m[5]), LongOut: atof(m[6]), LongCache: atof(m[7]),
			Fee: OpenRouterFee, Source: "config.yaml # openrouter: line",
		}
	}
	return out
}

// SyncedIDs lists the ids inside config.yaml's generated openrouter-sync block.
func SyncedIDs(configText string) map[string]bool {
	out := map[string]bool{}
	for _, m := range syncIDRe.FindAllStringSubmatch(syncBlockRe.FindString(configText), -1) {
		out[m[1]] = true
	}
	return out
}

// Advertised is what the site shows for one model.
type Advertised struct {
	In, Out    float64
	HasIn      bool
	HasOut     bool
	Cache      float64
	HasCache   bool
	SourceFile string
}

var (
	siteEntryRe = regexp.MustCompile(`(?s)\n  \{\s*\n\s*["']?id["']?: ["']([^"']+)["'],(.*?)\n  \}`)
	siteInRe    = regexp.MustCompile(`["']?priceInput["']?: (-?[0-9.]+)`)
	siteOutRe   = regexp.MustCompile(`["']?priceOutput["']?: (-?[0-9.]+)`)
	freeIDsRe   = regexp.MustCompile(`FREE_MODEL_IDS = \[([^\]]*)\]`)
	quotedRe    = regexp.MustCompile(`'([^']+)'`)
)

// ParseSiteModels reads priceInput/priceOutput per id from src/data/models.ts.
func ParseSiteModels(text string) map[string]Advertised {
	out := map[string]Advertised{}
	for _, e := range siteEntryRe.FindAllStringSubmatch(text, -1) {
		a := Advertised{SourceFile: "src/data/models.ts"}
		if m := siteInRe.FindStringSubmatch(e[2]); m != nil {
			a.In, a.HasIn = atof(m[1]), true
		}
		if m := siteOutRe.FindStringSubmatch(e[2]); m != nil {
			a.Out, a.HasOut = atof(m[1]), true
		}
		if _, dup := out[e[1]]; !dup {
			out[e[1]] = a
		}
	}
	return out
}

// ParsePricingJSON reads src/data/pricingConfig.generated.json (the /pricing
// calculator's data).
func ParsePricingJSON(raw []byte) (map[string]Advertised, error) {
	var rows []struct {
		ID        string   `json:"id"`
		Input     float64  `json:"input"`
		Output    float64  `json:"output"`
		CacheRead *float64 `json:"cacheRead"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	out := map[string]Advertised{}
	for _, r := range rows {
		a := Advertised{In: r.Input, Out: r.Output, HasIn: true, HasOut: true, SourceFile: "src/data/pricingConfig.generated.json"}
		if r.CacheRead != nil {
			a.Cache, a.HasCache = *r.CacheRead, true
		}
		out[r.ID] = a
	}
	return out, nil
}

// ParseFreeModelIDs reads FREE_MODEL_IDS from src/data/capabilities.ts.
func ParseFreeModelIDs(text string) []string {
	m := freeIDsRe.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	var ids []string
	for _, q := range quotedRe.FindAllStringSubmatch(m[1], -1) {
		ids = append(ids, q[1])
	}
	return ids
}

// Inputs is everything one check run needs.
type Inputs struct {
	Models []model.ModelConfig
	// OpenRouterCosts: model id -> cost for provider: openrouter entries.
	OpenRouterCosts map[string]Cost
	// RouteCosts: RouteKey -> cost for direct providers.
	RouteCosts map[string]Cost
	// Site / PricingJSON: advertised prices; nil skips that source.
	Site        map[string]Advertised
	PricingJSON map[string]Advertised
	// FreeIDs is FREE_MODEL_IDS; nil skips the free-list check.
	FreeIDs []string
	// Missing: OpenRouter model ids absent from the live catalog (live runs only).
	Missing map[string]bool
	// Stale: notes about recorded costs that differ from live (live runs only).
	Stale []string
	// Now decides which scheduled tariffs are in force.
	Now time.Time
}

// Severity of a finding. Critical findings break the owner rule and alert.
type Severity string

const (
	Critical Severity = "CRITICAL"
	Warning  Severity = "WARN"
)

type Finding struct {
	Severity Severity
	Kind     string // BELOW_COST, FREE_ROUTE_PAID, AD_MISMATCH, FREE_LIST, UPSTREAM_GONE, RECORD_STALE
	Model    string
	Field    string
	Billed   float64
	Want     float64 // cost floor (BELOW_COST) or advertised price (AD_MISMATCH)
	Detail   string
}

type Result struct {
	Findings   []Finding
	Checked    int      // models compared against a known upstream cost
	Unverified []string // token-priced models with no recorded upstream cost
}

func (r *Result) Critical() []Finding {
	var out []Finding
	for _, f := range r.Findings {
		if f.Severity == Critical {
			out = append(out, f)
		}
	}
	return out
}

// AssetPriced reports models billed per image/second/request/etc.; their token
// fields (if any) are not comparable with per-token rate cards.
func AssetPriced(m *model.ModelConfig) bool {
	sum := m.PricePerRequest + m.PricePer1MCharacters + m.PricePerImage + m.PricePerMegapixel +
		m.PriceFirstMegapixel + m.PriceExtraMegapixel + m.PricePerInputImage +
		m.PricePerVideo + m.PricePerSecond + m.PricePerSecondWithVideoInput + m.PricePerInputVideoSecond +
		m.PricePerMinute + m.PricePerHour + m.PricePerForecast
	return sum > 0 || len(m.PricePerImageByResolution) > 0 || len(m.PricePerSecondByResolution) > 0
}

// IsPriced reports whether any price field of m is non-zero.
func IsPriced(m *model.ModelConfig) bool {
	raw, _ := json.Marshal(m)
	var fields map[string]any
	_ = json.Unmarshal(raw, &fields)
	for k, v := range fields {
		if !strings.Contains(k, "price") {
			continue
		}
		switch x := v.(type) {
		case float64:
			if x != 0 {
				return true
			}
		case map[string]any:
			for _, e := range x {
				if f, ok := e.(float64); ok && f != 0 {
					return true
				}
			}
		}
	}
	return false
}

func orElse(v, fallback float64) float64 {
	if v > 0 {
		return v
	}
	return fallback
}

func round4(x float64) float64 { return math.Ceil(x*1e4-1e-6) / 1e4 }

// belowCost compares billed rates as billing applies them (unset cache rate
// bills cached tokens at the input rate; unset long rates fall back to base)
// with the upstream cost plus fee.
func belowCost(id string, m *model.ModelConfig, c Cost, t Tariff) []Finding {
	cache := orElse(t.Cache, t.In)
	longIn := orElse(m.InputPricePer1MLong, t.In)
	rows := []struct {
		field          string
		billed, upstrm float64
	}{
		{t.Label + "input", t.In, c.In},
		{t.Label + "output", t.Out, c.Out},
		{t.Label + "cache_hit", cache, c.Cache},
		{"long_input", longIn, c.LongIn},
		{"long_output", orElse(m.OutputPricePer1MLong, t.Out), c.LongOut},
		{"long_cache_hit", orElse(m.InputCacheHitPricePer1MLong, orElse(t.Cache, longIn)), c.LongCache},
	}
	if t.Label != "" {
		rows = rows[:3] // long tiers do not change with the schedule; checked once
	}
	var out []Finding
	for _, r := range rows {
		if r.upstrm <= 0 {
			continue
		}
		floor := r.upstrm * (1 + c.Fee)
		if r.billed+eps < floor {
			out = append(out, Finding{
				Severity: Critical, Kind: "BELOW_COST", Model: id, Field: r.field,
				Billed: r.billed, Want: round4(floor),
				Detail: fmt.Sprintf("upstream %g/M%s via %s", r.upstrm, feeNote(c.Fee), c.Source),
			})
		}
	}
	return out
}

func feeNote(fee float64) string {
	if fee == 0 {
		return ""
	}
	return fmt.Sprintf(" +%g%% fee", fee*100)
}

// Tariff is one set of base token rates billing can apply to a model.
type Tariff struct {
	Label          string // "" base, "off_peak_", "peak_"
	In, Cache, Out float64
}

// Tariffs lists every token tariff billing applies to m at or after now: the
// base rates until a scheduled change takes effect, then its off-peak rates
// and, when peak windows exist, its peak rates. A future schedule is included
// too so a coming change is checked before it bills.
func Tariffs(m *model.ModelConfig, now time.Time) []Tariff {
	s := m.ScheduledTokenPricing
	base := Tariff{"", m.InputPricePer1M, m.InputCacheHitPricePer1M, m.OutputPricePer1M}
	if s == nil {
		return []Tariff{base}
	}
	var out []Tariff
	if at, err := time.Parse(time.RFC3339, s.EffectiveAt); err != nil || now.Before(at) {
		out = append(out, base)
	}
	out = append(out, Tariff{"off_peak_", s.OffPeakInputPricePer1M, s.OffPeakInputCacheHitPer1M, s.OffPeakOutputPricePer1M})
	if len(s.PeakWindowsUTC) > 0 {
		out = append(out, Tariff{"peak_", s.PeakInputPricePer1M, s.PeakInputCacheHitPricePer1M, s.PeakOutputPricePer1M})
	}
	return out
}

// HeadlineRates is the single per-token price the model card should show: the
// base rates, or once a schedule is in force its off-peak ("from") rates.
func HeadlineRates(m *model.ModelConfig, now time.Time) (in, cache, out float64) {
	t := Tariffs(m, now)[0]
	return t.In, t.Cache, t.Out
}

func tokenPriced(m *model.ModelConfig) bool {
	return !AssetPriced(m) && (m.InputPricePer1M != 0 || m.OutputPricePer1M != 0)
}

// Check runs every comparison and returns findings sorted critical-first.
func Check(in Inputs) *Result {
	if in.Now.IsZero() {
		in.Now = time.Now()
	}
	res := &Result{}
	seen := map[string]bool{}
	for i := range in.Models {
		m := &in.Models[i]
		if seen[m.ID] {
			continue
		}
		seen[m.ID] = true

		var cost Cost
		var known bool
		if m.Provider == "openrouter" {
			cost, known = in.OpenRouterCosts[m.ID]
			if known && !IsPriced(m) && cost.In+cost.Out > 0 {
				res.Findings = append(res.Findings, Finding{Severity: Critical, Kind: "FREE_ROUTE_PAID", Model: m.ID,
					Detail: fmt.Sprintf("bills $0 but OpenRouter lists %s at %g/%g per 1M", m.ProviderModelID, cost.In, cost.Out)})
				res.Checked++
				continue
			}
			if in.Missing[m.ProviderModelID] {
				res.Findings = append(res.Findings, Finding{Severity: Warning, Kind: "UPSTREAM_GONE", Model: m.ID,
					Detail: fmt.Sprintf("%s is no longer in OpenRouter's catalog", m.ProviderModelID)})
			}
		} else {
			cost, known = in.RouteCosts[RouteKey(m)]
		}
		if !IsPriced(m) || AssetPriced(m) {
			continue
		}
		if !known {
			if tokenPriced(m) {
				res.Unverified = append(res.Unverified, m.ID+" ("+RouteKey(m)+")")
			}
			continue
		}
		res.Checked++
		for _, t := range Tariffs(m, in.Now) {
			res.Findings = append(res.Findings, belowCost(m.ID, m, cost, t)...)
		}
	}

	billedByID := map[string]*model.ModelConfig{}
	for i := range in.Models {
		if _, ok := billedByID[in.Models[i].ID]; !ok {
			billedByID[in.Models[i].ID] = &in.Models[i]
		}
	}
	for si, src := range []map[string]Advertised{in.Site, in.PricingJSON} {
		for id, a := range src {
			m := billedByID[id]
			if m == nil || !tokenPriced(m) {
				continue
			}
			// The model card shows one price, so it must be the one in force;
			// the /pricing data carries the schedule itself next to base rates.
			bIn, bCache, bOut := m.InputPricePer1M, m.InputCacheHitPricePer1M, m.OutputPricePer1M
			if si == 0 {
				bIn, bCache, bOut = HeadlineRates(m, in.Now)
			}
			for _, f := range []struct {
				field    string
				has      bool
				ad, bill float64
			}{
				{"input", a.HasIn, a.In, bIn},
				{"output", a.HasOut, a.Out, bOut},
				{"cache_hit", a.HasCache, a.Cache, bCache},
			} {
				if f.has && math.Abs(f.ad-f.bill) > eps {
					res.Findings = append(res.Findings, Finding{Severity: Critical, Kind: "AD_MISMATCH", Model: id, Field: f.field,
						Billed: f.bill, Want: f.ad, Detail: a.SourceFile + " advertises a different price than config.yaml bills"})
				}
			}
		}
	}

	if in.FreeIDs != nil {
		adv := map[string]bool{}
		for _, id := range in.FreeIDs {
			adv[id] = true
			m := billedByID[id]
			switch {
			case m == nil:
				res.Findings = append(res.Findings, Finding{Severity: Critical, Kind: "FREE_LIST", Model: id,
					Detail: "advertised free (FREE_MODEL_IDS) but missing from config.yaml"})
			case IsPriced(m):
				res.Findings = append(res.Findings, Finding{Severity: Critical, Kind: "FREE_LIST", Model: id,
					Detail: "advertised free (FREE_MODEL_IDS) but config.yaml bills it"})
			}
		}
		for id, m := range billedByID {
			if !IsPriced(m) && !adv[id] {
				res.Findings = append(res.Findings, Finding{Severity: Critical, Kind: "FREE_LIST", Model: id,
					Detail: "bills $0 in config.yaml but is not in FREE_MODEL_IDS"})
			}
		}
	}

	for _, s := range in.Stale {
		res.Findings = append(res.Findings, Finding{Severity: Warning, Kind: "RECORD_STALE", Detail: s})
	}

	sort.SliceStable(res.Findings, func(i, j int) bool {
		a, b := res.Findings[i], res.Findings[j]
		if a.Severity != b.Severity {
			return a.Severity == Critical
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Model != b.Model {
			return a.Model < b.Model
		}
		return a.Field < b.Field
	})
	sort.Strings(res.Unverified)
	return res
}

// Markdown renders the report the monitor writes and emails.
func (r *Result) Markdown(title string) string {
	var b strings.Builder
	crit := r.Critical()
	fmt.Fprintf(&b, "# %s\n\n", title)
	fmt.Fprintf(&b, "Critical: %d   Warnings: %d   Models checked against upstream cost: %d   Unverified token-priced models: %d\n\n",
		len(crit), len(r.Findings)-len(crit), r.Checked, len(r.Unverified))
	if len(r.Findings) > 0 {
		b.WriteString("| Severity | Kind | Model | Field | Billed /1M | Floor or advertised /1M | Detail |\n")
		b.WriteString("|---|---|---|---|---|---|---|\n")
		for _, f := range r.Findings {
			billed, want := "", ""
			if f.Field != "" {
				billed, want = fmt.Sprintf("%g", f.Billed), fmt.Sprintf("%g", f.Want)
			}
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n", f.Severity, f.Kind, f.Model, f.Field, billed, want, f.Detail)
		}
		b.WriteString("\n")
	} else {
		b.WriteString("No findings: every checked model bills at or above upstream cost and the site matches billing.\n\n")
	}
	if len(r.Unverified) > 0 {
		b.WriteString("## Unverified (no recorded upstream cost; add to scripts/pricing-guard/upstream-costs.json)\n\n")
		for _, u := range r.Unverified {
			fmt.Fprintf(&b, "- %s\n", u)
		}
	}
	return b.String()
}
