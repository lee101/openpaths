package handler

import (
	"errors"
	"strings"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/valyala/fasthttp"
)

const (
	kimiProviderName        = "moonshot"
	kimiDefaultBaseURL      = "https://api.moonshot.ai"
	kimiSearchProviderName  = "kimi"
	kimiProSearchName       = "kimi-pro"
	kimiFetchProviderName   = "kimi-fetch"
	kimiSearchPriceUSD      = 0.002
	kimiProSearchPriceUSD   = 0.003
	kimiFetchPriceUSD       = 0.002
	kimiSearchMaxLimit      = 20
	kimiSearchMaxSites      = 5
	kimiFetchMaxChars       = 200000
	kimiSearchTimeoutSecond = 30
)

var errKimiNoContent = errors.New("no extractable content for this URL")

func (h *SearchHandler) SetKimiProvider(cfg SearchProviderConfig) *SearchHandler {
	if cfg.BaseURL == "" {
		cfg.BaseURL = kimiDefaultBaseURL
	}
	h.kimi = cfg
	return h
}

func KimiSearchProviderConfig(providers []model.ProviderConfig) SearchProviderConfig {
	return searchProviderConfig(providers, kimiProviderName, kimiDefaultBaseURL)
}

type kimiSearchResult struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	SiteName string `json:"site_name"`
	Date     string `json:"date"`
	Snippet  string `json:"snippet"`
	Text     string `json:"text"`
	Chunks   []struct {
		Text string `json:"text"`
	} `json:"chunks"`
}

func (r kimiSearchResult) normalize() answerResult {
	text := r.Text
	if text == "" && len(r.Chunks) > 0 {
		parts := make([]string, 0, len(r.Chunks))
		for _, c := range r.Chunks {
			if c.Text != "" {
				parts = append(parts, c.Text)
			}
		}
		text = strings.Join(parts, "\n\n")
	}
	if text == "" {
		text = r.Snippet
	}
	return answerResult{Title: r.Title, URL: r.URL, Text: text, Snippet: r.Snippet, PublishedDate: r.Date}
}

func (h *SearchHandler) kimiHeaders(key string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + key}
}

func kimiDatePrefix(v any) string {
	s := stringFromPayload(v, "")
	if len(s) > 10 {
		s = s[:10]
	}
	return s
}

// handleKimiSearch serves Moonshot's Web Search Basic ($0.002) and Pro ($0.003,
// sites + time_window + content chunks). Moonshot only bills non-empty results,
// so empty result sets are not billed here either.
func (h *SearchHandler) handleKimiSearch(ctx *fasthttp.RequestCtx, payload map[string]any, provider, query string, numResults int, userID, apiKeyID string) {
	pro := provider == kimiProSearchName
	if numResults > kimiSearchMaxLimit {
		numResults = kimiSearchMaxLimit
	}
	body := map[string]any{"text_query": strings.TrimSpace(query), "limit": numResults, "timeout_seconds": kimiSearchTimeoutSecond}
	endpoint, price, modelName := "/v1/tools/search", kimiSearchPriceUSD, "kimi-search"
	if pro {
		endpoint, price, modelName = "/v1/tools/search_pro", kimiProSearchPriceUSD, "kimi-search-pro"
		sites := stringSlice(payload["includeDomains"])
		if len(sites) == 0 {
			sites = stringSlice(payload["sites"])
		}
		if len(sites) > kimiSearchMaxSites {
			sites = sites[:kimiSearchMaxSites]
		}
		if len(sites) > 0 {
			body["sites"] = sites
		}
		tw := map[string]any{}
		if s := kimiDatePrefix(payload["startPublishedDate"]); s != "" {
			tw["start"] = s
		}
		if e := kimiDatePrefix(payload["endPublishedDate"]); e != "" {
			tw["end"] = e
		}
		if len(tw) > 0 {
			body["time_window"] = tw
		}
	} else if b, ok := payload["include_content"].(bool); ok {
		body["include_content"] = b
	} else if contentsRequested(payload["contents"]) {
		body["include_content"] = true
	}
	h.runAnswerProvider(ctx, kimiProviderName, modelName, modelName, h.kimi, price, userID, apiKeyID,
		func(key string) (*answerResponse, int, error) {
			var out struct {
				SearchResults []kimiSearchResult `json:"search_results"`
			}
			status, err := h.postJSON(ctx, strings.TrimRight(h.kimi.BaseURL, "/")+endpoint, h.kimiHeaders(key), body, &out)
			if err != nil {
				return nil, status, err
			}
			res := make([]answerResult, 0, len(out.SearchResults))
			for _, r := range out.SearchResults {
				res = append(res, r.normalize())
			}
			return &answerResponse{Provider: provider, Model: modelName, Query: query, Citations: res, Results: res, free: len(res) == 0}, 200, nil
		})
}

// handleKimiFetch serves Moonshot URL Fetch ($0.002); `query` is the URL.
func (h *SearchHandler) handleKimiFetch(ctx *fasthttp.RequestCtx, query, userID, apiKeyID string) {
	pageURL := strings.TrimSpace(query)
	h.runAnswerProvider(ctx, kimiProviderName, "kimi-fetch", "kimi-fetch", h.kimi, kimiFetchPriceUSD, userID, apiKeyID,
		func(key string) (*answerResponse, int, error) {
			var out struct {
				URL      string `json:"url"`
				Title    string `json:"title"`
				Markdown string `json:"markdown"`
			}
			status, err := h.postJSON(ctx, strings.TrimRight(h.kimi.BaseURL, "/")+"/v1/tools/fetch", h.kimiHeaders(key), map[string]any{"url": pageURL}, &out)
			if err != nil {
				return nil, status, err
			}
			if strings.TrimSpace(out.Markdown) == "" {
				return nil, 404, errKimiNoContent
			}
			if out.URL == "" {
				out.URL = pageURL
			}
			r := answerResult{Title: out.Title, URL: out.URL, Text: truncateRunes(out.Markdown, kimiFetchMaxChars)}
			return &answerResponse{Provider: kimiFetchProviderName, Model: "kimi-fetch", Query: query, Answer: r.Text, Citations: []answerResult{r}, Results: []answerResult{r}}, 200, nil
		})
}
