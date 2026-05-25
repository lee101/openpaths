package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/valyala/fasthttp"
)

const (
	exaProviderName              = "exa"
	exaDefaultBaseURL            = "https://api.exa.ai"
	exaSearchModel               = "exa-search"
	exaSearchBasePriceUSD        = 0.0077
	exaSearchExtraResultPriceUSD = 0.0011
	exaContentPagePriceUSD       = 0.0011
	papersProviderName           = "papers"
	papersDefaultBaseURL         = "https://papers.app.nz"
	papersSearchModel            = "papers-search"
	papersSearchPriceUSD         = 0.001
)

type SearchProviderConfig struct {
	BaseURL string
	APIKey  string
	Enabled bool
}

type SearchHandler struct {
	exa      SearchProviderConfig
	papers   SearchProviderConfig
	billing  *billing.Engine
	recorder *metrics.Recorder
	client   *http.Client
}

func NewSearchHandler(exa, papers SearchProviderConfig, b *billing.Engine, rec *metrics.Recorder) *SearchHandler {
	if exa.BaseURL == "" {
		exa.BaseURL = exaDefaultBaseURL
	}
	if papers.BaseURL == "" {
		papers.BaseURL = papersDefaultBaseURL
	}
	return &SearchHandler{
		exa:      exa,
		papers:   papers,
		billing:  b,
		recorder: rec,
		client:   &http.Client{Timeout: 35 * time.Second},
	}
}

func (h *SearchHandler) HandleSearch(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}

	raw := ctx.PostBody()
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON: "+err.Error())
		return
	}
	query, _ := payload["query"].(string)
	if strings.TrimSpace(query) == "" {
		writeError(ctx, 400, "invalid_request", "query is required")
		return
	}
	numResults := intFromPayload(payload["numResults"], 10)
	if numResults < 1 {
		writeError(ctx, 400, "invalid_request", "numResults must be at least 1")
		return
	}
	if numResults > 100 {
		writeError(ctx, 400, "invalid_request", "numResults must be 100 or less")
		return
	}

	searchProvider, _ := payload["provider"].(string)
	if searchProvider == papersProviderName {
		h.handlePapersSearch(ctx, payload, query, numResults, userID, apiKeyID)
		return
	}
	if searchProvider != "" && searchProvider != exaProviderName {
		writeError(ctx, 400, "invalid_request", "provider must be exa or papers")
		return
	}
	h.handleExaSearch(ctx, raw, payload, numResults, userID, apiKeyID)
}

func (h *SearchHandler) handleExaSearch(ctx *fasthttp.RequestCtx, raw []byte, payload map[string]any, numResults int, userID, apiKeyID string) {
	app := requestAppAttribution(ctx)
	upstreamKey, byok := h.resolveExaKey(ctx)
	if upstreamKey == "" {
		writeError(ctx, 503, "provider_unavailable", "Exa search is not configured")
		return
	}
	cost := int64(0)
	if !byok {
		cost = estimateExaSearchCost(payload, numResults)
		if err := h.billing.PreCheckFixed(ctx, userID, cost); err != nil {
			h.recorder.RecordErrorWithApp(userID, apiKeyID, exaSearchModel, exaProviderName, 0, 402, err.Error(), false, app.ID, app.URL, app.Title, app.Categories)
			writeError(ctx, 402, "billing_error", "Insufficient credits. Please add credits to continue.")
			return
		}
	}

	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(h.exa.BaseURL, "/")+"/search", bytes.NewReader(raw))
	if err != nil {
		writeError(ctx, 500, "server_error", err.Error())
		return
	}
	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("x-api-key", upstreamKey)

	start := time.Now()
	resp, err := h.client.Do(upstreamReq)
	latency := time.Since(start)
	if err != nil {
		h.recorder.RecordErrorWithApp(userID, apiKeyID, exaSearchModel, exaProviderName, int(latency.Milliseconds()), 502, err.Error(), false, app.ID, app.URL, app.Title, app.Categories)
		writeError(ctx, 502, "provider_error", err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.recorder.RecordErrorWithApp(userID, apiKeyID, exaSearchModel, exaProviderName, int(latency.Milliseconds()), 502, err.Error(), false, app.ID, app.URL, app.Title, app.Categories)
		writeError(ctx, 502, "provider_error", err.Error())
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("Exa returned status %d", resp.StatusCode)
		}
		h.recorder.RecordErrorWithApp(userID, apiKeyID, exaSearchModel, exaProviderName, int(latency.Milliseconds()), resp.StatusCode, msg, false, app.ID, app.URL, app.Title, app.Categories)
		ctx.SetStatusCode(resp.StatusCode)
		ctx.SetContentType("application/json")
		ctx.SetBody(body)
		return
	}

	if !byok {
		if err := h.billing.DeductFixed(ctx, userID, exaSearchModel, cost, formatExaUsageDescription(payload, numResults), ""); err != nil {
			h.recorder.RecordErrorWithApp(userID, apiKeyID, exaSearchModel, exaProviderName, int(latency.Milliseconds()), 402, err.Error(), false, app.ID, app.URL, app.Title, app.Categories)
			writeError(ctx, 402, "billing_error", "Insufficient credits. Please add credits to continue.")
			return
		}
	}

	h.recorder.RecordSuccessWithApp(userID, apiKeyID, exaSearchModel, exaProviderName, 0, numResults, int(latency.Milliseconds()), 0, cost, false, app.ID, app.URL, app.Title, app.Categories)
	ctx.SetStatusCode(200)
	ctx.SetContentType("application/json")
	ctx.SetBody(body)
}

func (h *SearchHandler) handlePapersSearch(ctx *fasthttp.RequestCtx, payload map[string]any, query string, numResults int, userID, apiKeyID string) {
	app := requestAppAttribution(ctx)
	upstreamKey, byok := h.resolveProviderKey(ctx, papersProviderName, h.papers)
	if upstreamKey == "" {
		writeError(ctx, 503, "provider_unavailable", "Papers search is not configured")
		return
	}

	cost := int64(0)
	if !byok {
		cost = estimatePapersSearchCost()
		if err := h.billing.PreCheckFixed(ctx, userID, cost); err != nil {
			h.recorder.RecordErrorWithApp(userID, apiKeyID, papersSearchModel, papersProviderName, 0, 402, err.Error(), false, app.ID, app.URL, app.Title, app.Categories)
			writeError(ctx, 402, "billing_error", "Insufficient credits. Please add credits to continue.")
			return
		}
	}

	upstreamURL := strings.TrimRight(h.papers.BaseURL, "/") + "/api/search?" + papersSearchQuery(payload, query, numResults)
	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodGet, upstreamURL, nil)
	if err != nil {
		writeError(ctx, 500, "server_error", err.Error())
		return
	}
	upstreamReq.Header.Set("Authorization", "Bearer "+upstreamKey)

	start := time.Now()
	resp, err := h.client.Do(upstreamReq)
	latency := time.Since(start)
	if err != nil {
		h.recorder.RecordErrorWithApp(userID, apiKeyID, papersSearchModel, papersProviderName, int(latency.Milliseconds()), 502, err.Error(), false, app.ID, app.URL, app.Title, app.Categories)
		writeError(ctx, 502, "provider_error", err.Error())
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.recorder.RecordErrorWithApp(userID, apiKeyID, papersSearchModel, papersProviderName, int(latency.Milliseconds()), 502, err.Error(), false, app.ID, app.URL, app.Title, app.Categories)
		writeError(ctx, 502, "provider_error", err.Error())
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("Papers returned status %d", resp.StatusCode)
		}
		h.recorder.RecordErrorWithApp(userID, apiKeyID, papersSearchModel, papersProviderName, int(latency.Milliseconds()), resp.StatusCode, msg, false, app.ID, app.URL, app.Title, app.Categories)
		ctx.SetStatusCode(resp.StatusCode)
		ctx.SetContentType(contentTypeOrDefault(resp.Header.Get("Content-Type")))
		ctx.SetBody(body)
		return
	}

	if !byok {
		if err := h.billing.DeductFixed(ctx, userID, papersSearchModel, cost, formatPapersUsageDescription(payload, numResults), ""); err != nil {
			h.recorder.RecordErrorWithApp(userID, apiKeyID, papersSearchModel, papersProviderName, int(latency.Milliseconds()), 402, err.Error(), false, app.ID, app.URL, app.Title, app.Categories)
			writeError(ctx, 402, "billing_error", "Insufficient credits. Please add credits to continue.")
			return
		}
	}

	h.recorder.RecordSuccessWithApp(userID, apiKeyID, papersSearchModel, papersProviderName, 0, numResults, int(latency.Milliseconds()), 0, cost, false, app.ID, app.URL, app.Title, app.Categories)
	ctx.SetStatusCode(200)
	ctx.SetContentType(contentTypeOrDefault(resp.Header.Get("Content-Type")))
	ctx.SetBody(body)
}

func (h *SearchHandler) resolveExaKey(ctx *fasthttp.RequestCtx) (string, bool) {
	return h.resolveProviderKey(ctx, exaProviderName, h.exa)
}

func (h *SearchHandler) resolveProviderKey(ctx *fasthttp.RequestCtx, providerName string, cfg SearchProviderConfig) (string, bool) {
	keys, _ := ctx.UserValue(middleware.CtxKeyUserProviderKeys).(map[string]*queries.UserProviderKey)
	if keys != nil {
		if k := keys[providerName]; k != nil && strings.TrimSpace(k.APIKey) != "" {
			return strings.TrimSpace(k.APIKey), true
		}
	}
	if cfg.Enabled && strings.TrimSpace(cfg.APIKey) != "" {
		return strings.TrimSpace(cfg.APIKey), false
	}
	return "", false
}

func intFromPayload(v any, fallback int) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return fallback
	}
}

func estimateExaSearchCost(payload map[string]any, numResults int) int64 {
	requestCost := exaSearchBasePriceUSD
	if numResults > 10 {
		requestCost += float64(numResults-10) * exaSearchExtraResultPriceUSD
	}
	if contentsRequested(payload["contents"]) {
		requestCost += float64(numResults) * exaContentPagePriceUSD
	}
	units := int64(math.Ceil(requestCost * 10000))
	if units < 1 {
		return 1
	}
	return units
}

func estimatePapersSearchCost() int64 {
	units := int64(math.Ceil(papersSearchPriceUSD * 10000))
	if units < 1 {
		return 1
	}
	return units
}

func contentsRequested(v any) bool {
	m, ok := v.(map[string]any)
	if !ok || len(m) == 0 {
		return false
	}
	for _, enabled := range m {
		switch x := enabled.(type) {
		case bool:
			if x {
				return true
			}
		case map[string]any:
			return true
		}
	}
	return false
}

func formatExaUsageDescription(payload map[string]any, numResults int) string {
	searchType, _ := payload["type"].(string)
	if searchType == "" {
		searchType = "auto"
	}
	parts := []string{
		"Search: exa-search",
		fmt.Sprintf("type: %s", searchType),
		fmt.Sprintf("results: %d", numResults),
	}
	if contentsRequested(payload["contents"]) {
		parts = append(parts, "contents: true")
	}
	return strings.Join(parts, ", ")
}

func papersSearchQuery(payload map[string]any, query string, numResults int) string {
	values := url.Values{}
	values.Set("q", query)
	values.Set("limit", fmt.Sprintf("%d", numResults))
	searchType, _ := payload["type"].(string)
	if searchType == "" || searchType == "auto" || searchType == "fast" || searchType == "instant" || searchType == "deep" {
		searchType = "papers"
	}
	values.Set("type", searchType)
	for _, key := range []string{"sort", "has_code", "include_github_code", "format"} {
		if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
			values.Set(key, strings.TrimSpace(value))
		}
	}
	if value, ok := payload["hasCode"].(bool); ok {
		values.Set("has_code", fmt.Sprintf("%t", value))
	}
	if value, ok := payload["includeGithubCode"].(bool); ok {
		values.Set("include_github_code", fmt.Sprintf("%t", value))
	}
	return values.Encode()
}

func formatPapersUsageDescription(payload map[string]any, numResults int) string {
	searchType, _ := payload["type"].(string)
	if searchType == "" {
		searchType = "papers"
	}
	return strings.Join([]string{
		"Search: papers-search",
		fmt.Sprintf("type: %s", searchType),
		fmt.Sprintf("limit: %d", numResults),
	}, ", ")
}

func contentTypeOrDefault(value string) string {
	if strings.TrimSpace(value) == "" {
		return "application/json"
	}
	return value
}

func ExaSearchProviderConfig(providers []model.ProviderConfig) SearchProviderConfig {
	return searchProviderConfig(providers, exaProviderName, exaDefaultBaseURL)
}

func PapersSearchProviderConfig(providers []model.ProviderConfig) SearchProviderConfig {
	return searchProviderConfig(providers, papersProviderName, papersDefaultBaseURL)
}

func searchProviderConfig(providers []model.ProviderConfig, providerName, defaultBaseURL string) SearchProviderConfig {
	for _, p := range providers {
		if p.Name == providerName {
			return SearchProviderConfig{
				BaseURL: p.BaseURL,
				APIKey:  p.APIKey,
				Enabled: p.Enabled,
			}
		}
	}
	return SearchProviderConfig{BaseURL: defaultBaseURL}
}

func LogExaSearchPricing() {
	log.Printf("Exa search pricing enabled: $%.4f/request (1-10 results), $%.4f/extra result, $%.4f/content page",
		exaSearchBasePriceUSD, exaSearchExtraResultPriceUSD, exaContentPagePriceUSD)
	log.Printf("Papers search pricing enabled: $%.4f/request", papersSearchPriceUSD)
}
