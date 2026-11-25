package pricingguard

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// OpenRouter's public catalog needs no API key.
const (
	CatalogURL   = "https://openrouter.ai/api/v1/models"
	EndpointsURL = "https://openrouter.ai/api/v1/models/%s/endpoints"
)

// ORPricing is OpenRouter's per-token pricing object (strings, USD per token).
type ORPricing struct {
	Prompt         string `json:"prompt"`
	Completion     string `json:"completion"`
	InputCacheRead string `json:"input_cache_read"`
	Overrides      []struct {
		MinPromptTokens int    `json:"min_prompt_tokens"`
		Prompt          string `json:"prompt"`
		Completion      string `json:"completion"`
		InputCacheRead  string `json:"input_cache_read"`
	} `json:"overrides"`
}

func perM(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return round9(f * 1e6)
}

func round9(x float64) float64 {
	v, _ := strconv.ParseFloat(strconv.FormatFloat(x, 'f', 9, 64), 64)
	return v
}

// ToCost converts OpenRouter pricing to per-1M rates the way the sync script
// does: time-of-day overrides raise the base to the peak (one flat price must
// cover it), a min_prompt_tokens override is the long-context tier.
func (p ORPricing) ToCost(fee float64, source string) Cost {
	c := Cost{In: perM(p.Prompt), Out: perM(p.Completion), Cache: perM(p.InputCacheRead), Fee: fee, Source: source}
	for _, o := range p.Overrides {
		if o.MinPromptTokens > 0 {
			if c.LongThreshold == 0 {
				c.LongThreshold, c.LongIn, c.LongOut, c.LongCache = o.MinPromptTokens, perM(o.Prompt), perM(o.Completion), perM(o.InputCacheRead)
			}
			continue
		}
		c.In, c.Out, c.Cache = max(c.In, perM(o.Prompt)), max(c.Out, perM(o.Completion)), max(c.Cache, perM(o.InputCacheRead))
	}
	return c
}

// Catalog is OpenRouter id -> pricing.
type Catalog map[string]ORPricing

func ParseCatalog(raw []byte) (Catalog, error) {
	var body struct {
		Data []struct {
			ID      string    `json:"id"`
			Pricing ORPricing `json:"pricing"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	if len(body.Data) == 0 {
		return nil, fmt.Errorf("empty OpenRouter catalog")
	}
	out := Catalog{}
	for _, d := range body.Data {
		out[d.ID] = d.Pricing
	}
	return out, nil
}

// ParseEndpoints returns tag -> pricing from a /models/<id>/endpoints body.
func ParseEndpoints(raw []byte) (map[string]ORPricing, error) {
	var body struct {
		Data struct {
			Endpoints []struct {
				Tag     string    `json:"tag"`
				Pricing ORPricing `json:"pricing"`
			} `json:"endpoints"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	out := map[string]ORPricing{}
	for _, e := range body.Data.Endpoints {
		out[e.Tag] = e.Pricing
	}
	return out, nil
}

// Fetcher gets a URL body; tests substitute fixtures.
type Fetcher func(url string) ([]byte, error)

func HTTPFetcher(client *http.Client) Fetcher {
	return func(u string) ([]byte, error) {
		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "openpaths-pricing-guard")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GET %s: HTTP %d", u, resp.StatusCode)
		}
		return body, nil
	}
}

// LiveCosts is the result of refreshing upstream costs from public endpoints.
type LiveCosts struct {
	OpenRouter map[string]Cost // model id -> live cost (openrouter entries)
	Routes     map[string]Cost // RouteKey -> live cost (ledger entries with a ref)
	Missing    map[string]bool // OpenRouter ids not in the catalog
	Stale      []string        // recorded != live
	Errors     []string        // refs that could not be resolved
}

// FetchLive reads OpenRouter's catalog for every provider: openrouter model and
// each ledger ref's endpoint listing. orModels maps our id -> OpenRouter id.
func FetchLive(get Fetcher, orModels map[string]string, recorded map[string]Cost, ledger *Ledger, pause time.Duration) (*LiveCosts, error) {
	raw, err := get(CatalogURL)
	if err != nil {
		return nil, err
	}
	cat, err := ParseCatalog(raw)
	if err != nil {
		return nil, err
	}
	lc := &LiveCosts{OpenRouter: map[string]Cost{}, Routes: map[string]Cost{}, Missing: map[string]bool{}}
	ids := make([]string, 0, len(orModels))
	for id := range orModels {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		pmid := orModels[id]
		p, ok := cat[pmid]
		if !ok {
			lc.Missing[pmid] = true
			continue
		}
		c := p.ToCost(OpenRouterFee, "openrouter.ai/api/v1/models "+pmid)
		lc.OpenRouter[id] = c
		if r, ok := recorded[id]; ok && !r.equal(c) {
			lc.Stale = append(lc.Stale, fmt.Sprintf("%s: recorded OpenRouter rate %g/%g/%g, live %g/%g/%g (run make sync-openrouter)", id, r.In, r.Out, r.Cache, c.In, c.Out, c.Cache))
		}
	}

	keys := make([]string, 0, len(ledger.Routes))
	for k := range ledger.Routes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	endpoints := map[string]map[string]ORPricing{}
	for _, k := range keys {
		e := ledger.Routes[k]
		if e.Ref == nil || e.Ref.OpenRouter == "" {
			continue
		}
		eps, ok := endpoints[e.Ref.OpenRouter]
		if !ok {
			if pause > 0 {
				time.Sleep(pause)
			}
			body, err := get(fmt.Sprintf(EndpointsURL, (&url.URL{Path: e.Ref.OpenRouter}).EscapedPath()))
			if err == nil {
				eps, err = ParseEndpoints(body)
			}
			if err != nil {
				lc.Errors = append(lc.Errors, fmt.Sprintf("%s: %v", k, err))
				endpoints[e.Ref.OpenRouter] = nil
				continue
			}
			endpoints[e.Ref.OpenRouter] = eps
		}
		p, ok := eps[e.Ref.Tag]
		if !ok {
			if eps != nil {
				var tags []string
				for t := range eps {
					tags = append(tags, t)
				}
				sort.Strings(tags)
				lc.Errors = append(lc.Errors, fmt.Sprintf("%s: endpoint tag %q not listed for %s (have %s)", k, e.Ref.Tag, e.Ref.OpenRouter, strings.Join(tags, ", ")))
			}
			continue
		}
		c := p.ToCost(0, fmt.Sprintf("openrouter endpoints %s [%s]", e.Ref.OpenRouter, e.Ref.Tag))
		lc.Routes[k] = c
		if !e.Cost.equal(c) {
			lc.Stale = append(lc.Stale, fmt.Sprintf("%s: recorded %g/%g/%g, live %g/%g/%g (refresh with -update-ledger)", k, e.In, e.Out, e.Cache, c.In, c.Out, c.Cache))
		}
	}
	return lc, nil
}

// UpdateLedger writes live costs into the ledger entries that have refs.
func UpdateLedger(l *Ledger, lc *LiveCosts, today string) int {
	n := 0
	for k, c := range lc.Routes {
		e := l.Routes[k]
		if e == nil {
			continue
		}
		if !e.Cost.equal(c) || e.Verified != today {
			src := c.Source
			e.Cost = c
			e.Cost.Source = src
			e.Verified = today
			n++
		}
	}
	return n
}
