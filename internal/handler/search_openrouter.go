package handler

import (
	"fmt"
	"strings"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/valyala/fasthttp"
)

const (
	openrouterSearchProviderName = "openrouter"
	openrouterDefaultBaseURL     = "https://openrouter.ai/api"
	openrouterSearchModel        = "openrouter-search"
	openrouterSearchDefaultModel = "minimax/minimax-m3.1-flash"
	openrouterSearchDefaultN     = 5
	openrouterSearchMaxN         = 25
	openrouterModelTokensUSD     = 0.003
)

var openrouterSearchEngines = map[string]bool{"auto": true, "native": true, "exa": true, "firecrawl": true, "parallel": true, "perplexity": true}

func (h *SearchHandler) SetOpenRouterProvider(cfg SearchProviderConfig) *SearchHandler {
	if cfg.BaseURL == "" {
		cfg.BaseURL = openrouterDefaultBaseURL
	}
	h.openrouter = cfg
	return h
}

func OpenRouterSearchProviderConfig(providers []model.ProviderConfig) SearchProviderConfig {
	return searchProviderConfig(providers, openrouterSearchProviderName, openrouterDefaultBaseURL)
}

// openrouterEngineFor maps a /v1/search provider value to an OpenRouter web_search engine.
func openrouterEngineFor(provider string, payload map[string]any) string {
	switch provider {
	case "firecrawl", "parallel", "perplexity":
		return provider
	}
	if e := strings.ToLower(stringFromPayload(payload["engine"], "")); e != "" {
		return e
	}
	return "auto"
}

// openrouterSearchPriceUSD is the OpenRouter engine fee plus the 10% fee and a flat allowance for the synthesis model.
func openrouterSearchPriceUSD(engine, mode string, n int) float64 {
	var fee float64
	switch engine {
	case "firecrawl":
		fee = (0.005 + 0.0125*float64(n)) * 1.1
	case "parallel":
		fee = 0.0055
		if mode == "turbo" || mode == "fast" {
			fee = 0.0011
		}
		if n > 10 {
			fee += float64(n-10) * 0.0011
		}
	case "perplexity":
		fee = 0.0055
	case "native":
		fee = 0.01
	default:
		fee = exaSearchBasePriceUSD
		if mode == "deep" || mode == "deep-lite" {
			fee = 0.0132
		} else if mode == "deep-reasoning" {
			fee = 0.0165
		}
	}
	return fee + openrouterModelTokensUSD
}

type openrouterChatResp struct {
	Choices []struct {
		Message struct {
			Content     any `json:"content"`
			Annotations []struct {
				Type        string `json:"type"`
				URLCitation struct {
					URL     string `json:"url"`
					Title   string `json:"title"`
					Content string `json:"content"`
				} `json:"url_citation"`
			} `json:"annotations"`
		} `json:"message"`
	} `json:"choices"`
}

func stringSlice(v any) []string {
	var out []string
	switch t := v.(type) {
	case []any:
		for _, x := range t {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
	case string:
		if strings.TrimSpace(t) != "" {
			out = append(out, strings.TrimSpace(t))
		}
	}
	return out
}

// handleOpenRouterSearch grounds a cheap model with OpenRouter's openrouter:web_search
// server tool so every OpenRouter engine (exa, parallel, firecrawl, perplexity, native)
// is reachable behind one /v1/search call.
func (h *SearchHandler) handleOpenRouterSearch(ctx *fasthttp.RequestCtx, payload map[string]any, provider, query string, numResults int, userID, apiKeyID string) {
	engine := openrouterEngineFor(provider, payload)
	if !openrouterSearchEngines[engine] {
		writeError(ctx, 400, "invalid_request", "engine must be one of: auto, native, exa, firecrawl, parallel, perplexity")
		return
	}
	if _, set := payload["numResults"]; !set {
		numResults = openrouterSearchDefaultN
	}
	if numResults > openrouterSearchMaxN {
		numResults = openrouterSearchMaxN
	}
	allowed, excluded := stringSlice(payload["includeDomains"]), stringSlice(payload["excludeDomains"])
	if len(allowed) == 0 {
		allowed = stringSlice(payload["allowed_domains"])
	}
	if len(excluded) == 0 {
		excluded = stringSlice(payload["excluded_domains"])
	}
	if len(allowed) > 0 && len(excluded) > 0 && engine != "exa" && engine != "auto" {
		writeError(ctx, 400, "invalid_request", engine+" does not allow includeDomains and excludeDomains together")
		return
	}
	mode := stringFromPayload(payload["mode"], "")
	if mode == "" {
		mode = stringFromPayload(payload["type"], "")
	}
	modelName := stringFromPayload(payload["model"], openrouterSearchDefaultModel)

	params := map[string]any{"max_results": numResults, "max_total_results": numResults}
	if engine != "auto" {
		params["engine"] = engine
	}
	if mode != "" && (engine == "exa" || engine == "parallel" || engine == "auto") {
		params["mode"] = mode
	}
	if len(allowed) > 0 {
		params["allowed_domains"] = allowed
	}
	if len(excluded) > 0 {
		params["excluded_domains"] = excluded
	}
	if s := stringFromPayload(payload["search_context_size"], ""); s != "" {
		params["search_context_size"] = s
	}
	if n := intFromPayload(payload["max_characters"], 0); n > 0 {
		params["max_characters"] = n
	}
	if loc, ok := payload["user_location"].(map[string]any); ok {
		params["user_location"] = loc
	}

	h.runAnswerProvider(ctx, openrouterSearchProviderName, modelName, openrouterSearchModel, h.openrouter, openrouterSearchPriceUSD(engine, mode, numResults), userID, apiKeyID,
		func(key string) (*answerResponse, int, error) {
			reqBody := map[string]any{
				"model": modelName,
				"messages": []map[string]any{
					{"role": "system", "content": "Search the web for the user's query, then answer concisely with inline [n] citations."},
					{"role": "user", "content": query},
				},
				"tools": []map[string]any{{"type": "openrouter:web_search", "parameters": params}},
			}
			var cr openrouterChatResp
			status, err := h.postJSON(ctx, strings.TrimRight(h.openrouter.BaseURL, "/")+"/v1/chat/completions", map[string]string{"Authorization": "Bearer " + key, "X-Title": "OpenPaths Search"}, reqBody, &cr)
			if err != nil {
				return nil, status, err
			}
			if len(cr.Choices) == 0 {
				return nil, 502, fmt.Errorf("empty response from %s", modelName)
			}
			m := cr.Choices[0].Message
			out := &answerResponse{Provider: provider, Model: modelName, Query: query}
			if provider == "" {
				out.Provider = openrouterSearchProviderName
			}
			if s, ok := m.Content.(string); ok {
				out.Answer = strings.TrimSpace(s)
			}
			seen := map[string]bool{}
			for _, a := range m.Annotations {
				c := a.URLCitation
				if c.URL == "" || seen[c.URL] {
					continue
				}
				seen[c.URL] = true
				out.Citations = append(out.Citations, answerResult{Title: c.Title, URL: c.URL, Text: truncateRunes(c.Content, 1200)})
			}
			out.Results = out.Citations
			return out, 200, nil
		})
}
