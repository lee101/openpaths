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

	geminiProviderName   = "gemini"
	geminiDefaultBaseURL = "https://generativelanguage.googleapis.com"
	geminiDefaultModel   = "gemini-3.5-flash"
	geminiSearchPriceUSD = 0.004

	openaiSearchProviderName = "openai"
	openaiDefaultBaseURL     = "https://api.openai.com"
	openaiSearchDefaultModel = "gpt-4o"
	openaiSearchPriceUSD     = 0.01

	grokProviderName   = "grok"
	grokDefaultBaseURL = "https://api.x.ai"
	grokDefaultModel   = "grok-4.3"
	grokSearchPriceUSD = 0.01
)

type SearchProviderConfig struct {
	BaseURL string
	APIKey  string
	Enabled bool
}

type SearchHandler struct {
	exa      SearchProviderConfig
	papers   SearchProviderConfig
	gemini   SearchProviderConfig
	openai   SearchProviderConfig
	xai      SearchProviderConfig
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
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

// SetAnswerProviders wires the LLM-with-web-search answer providers (Gemini Flash,
// OpenAI web search, Grok live search). Returns the handler for fluent setup.
func (h *SearchHandler) SetAnswerProviders(gemini, openai, xai SearchProviderConfig) *SearchHandler {
	if gemini.BaseURL == "" {
		gemini.BaseURL = geminiDefaultBaseURL
	}
	if openai.BaseURL == "" {
		openai.BaseURL = openaiDefaultBaseURL
	}
	if xai.BaseURL == "" {
		xai.BaseURL = grokDefaultBaseURL
	}
	h.gemini = gemini
	h.openai = openai
	h.xai = xai
	return h
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
	switch searchProvider {
	case papersProviderName:
		h.handlePapersSearch(ctx, payload, query, numResults, userID, apiKeyID)
		return
	case geminiProviderName:
		h.handleGeminiSearch(ctx, payload, query, userID, apiKeyID)
		return
	case openaiSearchProviderName:
		h.handleOpenAISearch(ctx, payload, query, userID, apiKeyID)
		return
	case grokProviderName:
		h.handleGrokSearch(ctx, payload, query, userID, apiKeyID)
		return
	case "", exaProviderName:
		h.handleExaSearch(ctx, raw, payload, numResults, userID, apiKeyID)
		return
	default:
		writeError(ctx, 400, "invalid_request", "provider must be one of: exa, papers, gemini, openai, grok")
		return
	}
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

// answerResult is a normalized citation/source for answer providers.
type answerResult struct {
	Title string `json:"title,omitempty"`
	URL   string `json:"url,omitempty"`
	Text  string `json:"text,omitempty"`
}

// answerResponse is the normalized shape returned by the answer providers
// (gemini, openai, grok). `results` mirrors `citations` so the existing
// result-card renderer keeps working without a special case.
type answerResponse struct {
	Provider      string         `json:"provider"`
	Model         string         `json:"model"`
	Query         string         `json:"query"`
	Answer        string         `json:"answer"`
	Citations     []answerResult `json:"citations"`
	Results       []answerResult `json:"results"`
	SearchQueries []string       `json:"searchQueries,omitempty"`
}

func stringFromPayload(v any, fallback string) string {
	if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s)
	}
	return fallback
}

func fixedCostUnits(priceUSD float64) int64 {
	units := int64(math.Ceil(priceUSD * 10000))
	if units < 1 {
		return 1
	}
	return units
}

// runAnswerProvider centralizes billing pre-check, the upstream call, error
// handling, metrics and credit deduction for the answer providers. The caller
// supplies a `call` closure that performs the provider-specific HTTP request
// and returns the normalized response.
func (h *SearchHandler) runAnswerProvider(
	ctx *fasthttp.RequestCtx,
	providerName, modelName, billingModel string,
	cfg SearchProviderConfig,
	priceUSD float64,
	userID, apiKeyID string,
	call func(upstreamKey string) (*answerResponse, int, error),
) {
	app := requestAppAttribution(ctx)
	upstreamKey, byok := h.resolveProviderKey(ctx, providerName, cfg)
	if upstreamKey == "" {
		writeError(ctx, 503, "provider_unavailable", "This search provider is not configured")
		return
	}

	cost := int64(0)
	if !byok && h.billing != nil {
		cost = fixedCostUnits(priceUSD)
		if err := h.billing.PreCheckFixed(ctx, userID, cost); err != nil {
			h.recorder.RecordErrorWithApp(userID, apiKeyID, billingModel, providerName, 0, 402, err.Error(), false, app.ID, app.URL, app.Title, app.Categories)
			writeError(ctx, 402, "billing_error", "Insufficient credits. Please add credits to continue.")
			return
		}
	}

	start := time.Now()
	resp, status, err := call(upstreamKey)
	latency := time.Since(start)
	if err != nil {
		if status == 0 {
			status = 502
		}
		h.recorder.RecordErrorWithApp(userID, apiKeyID, billingModel, providerName, int(latency.Milliseconds()), status, err.Error(), false, app.ID, app.URL, app.Title, app.Categories)
		writeError(ctx, status, "provider_error", err.Error())
		return
	}

	if !byok && h.billing != nil {
		if err := h.billing.DeductFixed(ctx, userID, billingModel, cost, fmt.Sprintf("Search: %s, model: %s", billingModel, modelName), ""); err != nil {
			h.recorder.RecordErrorWithApp(userID, apiKeyID, billingModel, providerName, int(latency.Milliseconds()), 402, err.Error(), false, app.ID, app.URL, app.Title, app.Categories)
			writeError(ctx, 402, "billing_error", "Insufficient credits. Please add credits to continue.")
			return
		}
	}

	h.recorder.RecordSuccessWithApp(userID, apiKeyID, billingModel, providerName, 0, len(resp.Citations), int(latency.Milliseconds()), 0, cost, false, app.ID, app.URL, app.Title, app.Categories)
	writeJSON(ctx, 200, resp)
}

// postJSON performs a JSON POST to url with optional headers and decodes the
// JSON body into out. It returns the upstream status and an error describing a
// non-2xx response or transport failure.
func (h *SearchHandler) postJSON(ctx *fasthttp.RequestCtx, url string, headers map[string]string, reqBody any, out any) (int, error) {
	raw, err := json.Marshal(reqBody)
	if err != nil {
		return 500, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return 500, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return 502, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 502, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 500 {
			msg = msg[:500]
		}
		if msg == "" {
			msg = fmt.Sprintf("upstream returned status %d", resp.StatusCode)
		}
		return resp.StatusCode, fmt.Errorf("%s", msg)
	}
	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return 502, fmt.Errorf("decode upstream response: %w", err)
		}
	}
	return resp.StatusCode, nil
}

// --- Gemini Flash with Google Search grounding ---

type geminiSearchResp struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		GroundingMetadata struct {
			WebSearchQueries []string `json:"webSearchQueries"`
			GroundingChunks  []struct {
				Web struct {
					URI   string `json:"uri"`
					Title string `json:"title"`
				} `json:"web"`
			} `json:"groundingChunks"`
		} `json:"groundingMetadata"`
	} `json:"candidates"`
}

func (h *SearchHandler) handleGeminiSearch(ctx *fasthttp.RequestCtx, payload map[string]any, query, userID, apiKeyID string) {
	modelName := stringFromPayload(payload["model"], geminiDefaultModel)
	h.runAnswerProvider(ctx, "google", modelName, "gemini-search", h.gemini, geminiSearchPriceUSD, userID, apiKeyID,
		func(upstreamKey string) (*answerResponse, int, error) {
			endpoint := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", strings.TrimRight(h.gemini.BaseURL, "/"), modelName, url.QueryEscape(upstreamKey))
			reqBody := map[string]any{
				"contents": []map[string]any{{"role": "user", "parts": []map[string]any{{"text": query}}}},
				"tools":    []map[string]any{{"google_search": map[string]any{}}},
			}
			var gr geminiSearchResp
			status, err := h.postJSON(ctx, endpoint, nil, reqBody, &gr)
			if err != nil {
				return nil, status, err
			}
			out := &answerResponse{Provider: geminiProviderName, Model: modelName, Query: query}
			if len(gr.Candidates) > 0 {
				cand := gr.Candidates[0]
				var sb strings.Builder
				for _, p := range cand.Content.Parts {
					sb.WriteString(p.Text)
				}
				out.Answer = strings.TrimSpace(sb.String())
				out.SearchQueries = cand.GroundingMetadata.WebSearchQueries
				for _, ch := range cand.GroundingMetadata.GroundingChunks {
					if ch.Web.URI == "" {
						continue
					}
					out.Citations = append(out.Citations, answerResult{Title: ch.Web.Title, URL: ch.Web.URI})
				}
			}
			out.Results = out.Citations
			return out, 200, nil
		})
}

// --- OpenAI web search (Responses API) ---

type openAIResponsesResp struct {
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			Annotations []struct {
				Type  string `json:"type"`
				URL   string `json:"url"`
				Title string `json:"title"`
			} `json:"annotations"`
		} `json:"content"`
	} `json:"output"`
}

func (h *SearchHandler) handleOpenAISearch(ctx *fasthttp.RequestCtx, payload map[string]any, query, userID, apiKeyID string) {
	modelName := stringFromPayload(payload["model"], openaiSearchDefaultModel)
	h.runAnswerProvider(ctx, openaiSearchProviderName, modelName, "openai-search", h.openai, openaiSearchPriceUSD, userID, apiKeyID,
		func(upstreamKey string) (*answerResponse, int, error) {
			endpoint := strings.TrimRight(h.openai.BaseURL, "/") + "/v1/responses"
			reqBody := map[string]any{
				"model": modelName,
				"input": query,
				"tools": []map[string]any{{"type": "web_search"}},
			}
			var or openAIResponsesResp
			status, err := h.postJSON(ctx, endpoint, map[string]string{"Authorization": "Bearer " + upstreamKey}, reqBody, &or)
			if err != nil {
				return nil, status, err
			}
			out := &answerResponse{Provider: openaiSearchProviderName, Model: modelName, Query: query}
			seen := map[string]bool{}
			var sb strings.Builder
			for _, item := range or.Output {
				if item.Type != "message" {
					continue
				}
				for _, c := range item.Content {
					sb.WriteString(c.Text)
					for _, a := range c.Annotations {
						if a.URL == "" || seen[a.URL] {
							continue
						}
						seen[a.URL] = true
						out.Citations = append(out.Citations, answerResult{Title: a.Title, URL: a.URL})
					}
				}
			}
			out.Answer = strings.TrimSpace(sb.String())
			out.Results = out.Citations
			return out, 200, nil
		})
}

// --- Grok live search (xAI chat completions) ---

type grokChatResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Citations []string `json:"citations"`
}

func (h *SearchHandler) handleGrokSearch(ctx *fasthttp.RequestCtx, payload map[string]any, query, userID, apiKeyID string) {
	modelName := stringFromPayload(payload["model"], grokDefaultModel)
	h.runAnswerProvider(ctx, "xai", modelName, "grok-search", h.xai, grokSearchPriceUSD, userID, apiKeyID,
		func(upstreamKey string) (*answerResponse, int, error) {
			endpoint := strings.TrimRight(h.xai.BaseURL, "/") + "/v1/chat/completions"
			reqBody := map[string]any{
				"model":    modelName,
				"messages": []map[string]any{{"role": "user", "content": query}},
				"search_parameters": map[string]any{
					"mode":             "auto",
					"return_citations": true,
				},
			}
			var cr grokChatResp
			status, err := h.postJSON(ctx, endpoint, map[string]string{"Authorization": "Bearer " + upstreamKey}, reqBody, &cr)
			if err != nil {
				return nil, status, err
			}
			out := &answerResponse{Provider: grokProviderName, Model: modelName, Query: query}
			if len(cr.Choices) > 0 {
				out.Answer = strings.TrimSpace(cr.Choices[0].Message.Content)
			}
			for _, c := range cr.Citations {
				if strings.TrimSpace(c) == "" {
					continue
				}
				out.Citations = append(out.Citations, answerResult{URL: c, Title: citationTitleFromURL(c)})
			}
			out.Results = out.Citations
			return out, 200, nil
		})
}

func citationTitleFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	return strings.TrimPrefix(u.Host, "www.")
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

// GeminiSearchProviderConfig resolves the Google provider config (used for
// Gemini Flash with Google Search grounding). Looks up the "google" provider.
func GeminiSearchProviderConfig(providers []model.ProviderConfig) SearchProviderConfig {
	return searchProviderConfig(providers, "google", geminiDefaultBaseURL)
}

func OpenAISearchProviderConfig(providers []model.ProviderConfig) SearchProviderConfig {
	return searchProviderConfig(providers, openaiSearchProviderName, openaiDefaultBaseURL)
}

func GrokSearchProviderConfig(providers []model.ProviderConfig) SearchProviderConfig {
	return searchProviderConfig(providers, "xai", grokDefaultBaseURL)
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
