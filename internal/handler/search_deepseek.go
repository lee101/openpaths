package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/valyala/fasthttp"
)

const (
	deepseekSearchProviderName = "deepseek"
	deepseekDefaultBaseURL     = "https://api.deepseek.com"
	deepseekSearchPriceUSD     = 0.00075 // $0.75 / 1,000 calls
	deepseekAgentModel         = "deepseek-v4-flash"

	openpathsSearchProviderName = "openpaths"
	openpathsSearchModel        = "openpaths-search"
	openpathsSearchPriceUSD     = 0.01
	openpathsAgentMaxRounds     = 3
	openpathsMaxCitations       = 12
	openpathsFetchMaxChars      = 6000
)

// SetDeepSeekProvider wires the DeepSeek Search/Fetch API (also drives the
// openpaths agent search, which uses DeepSeek Flash for tool calling).
func (h *SearchHandler) SetDeepSeekProvider(cfg SearchProviderConfig) *SearchHandler {
	if cfg.BaseURL == "" {
		cfg.BaseURL = deepseekDefaultBaseURL
	}
	h.deepseek = cfg
	return h
}

func DeepSeekSearchProviderConfig(providers []model.ProviderConfig) SearchProviderConfig {
	return searchProviderConfig(providers, deepseekSearchProviderName, deepseekDefaultBaseURL)
}

type deepseekSearchResult struct {
	RerankScore float64 `json:"rerank_score"`
	Title       string  `json:"title"`
	URL         string  `json:"url"`
	Snippet     string  `json:"snippet"`
	Summary     string  `json:"summary"`
	Content     string  `json:"content"`
	SiteName    string  `json:"site_name"`
	Date        string  `json:"date"`
}

func (h *SearchHandler) deepseekHeaders(key string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + key}
}

// deepseekSearch calls POST /v1/tools/search and normalizes the results.
func (h *SearchHandler) deepseekSearch(ctx *fasthttp.RequestCtx, key, query string, count int, needContent bool) ([]answerResult, int, error) {
	q := []rune(strings.TrimSpace(query))
	if len(q) > 128 {
		q = q[:128]
	}
	if count < 1 {
		count = 10
	}
	if count > 50 {
		count = 50
	}
	var out struct {
		Results []deepseekSearchResult `json:"results"`
	}
	status, err := h.postJSON(ctx, strings.TrimRight(h.deepseek.BaseURL, "/")+"/v1/tools/search", h.deepseekHeaders(key),
		map[string]any{"query": string(q), "count": count, "need_content": needContent}, &out)
	if err != nil {
		return nil, status, err
	}
	res := make([]answerResult, 0, len(out.Results))
	for _, r := range out.Results {
		text := r.Content
		if text == "" {
			text = r.Summary
		}
		if text == "" {
			text = r.Snippet
		}
		res = append(res, answerResult{Title: r.Title, URL: r.URL, Text: text, Snippet: r.Snippet, PublishedDate: r.Date})
	}
	return res, 200, nil
}

// deepseekFetch calls POST /v1/tools/fetch and returns the page text.
func (h *SearchHandler) deepseekFetch(ctx *fasthttp.RequestCtx, key, pageURL string) (string, error) {
	var out map[string]any
	if _, err := h.postJSON(ctx, strings.TrimRight(h.deepseek.BaseURL, "/")+"/v1/tools/fetch", h.deepseekHeaders(key), map[string]any{"url": pageURL}, &out); err != nil {
		return "", err
	}
	for _, k := range []string{"content", "text", "markdown"} {
		if s, ok := out[k].(string); ok && s != "" {
			return truncateRunes(s, openpathsFetchMaxChars), nil
		}
	}
	b, _ := json.Marshal(out)
	return truncateRunes(string(b), openpathsFetchMaxChars), nil
}

func (h *SearchHandler) handleDeepSeekSearch(ctx *fasthttp.RequestCtx, payload map[string]any, query string, numResults int, userID, apiKeyID string) {
	needContent := false
	if b, ok := payload["need_content"].(bool); ok {
		needContent = b
	} else if contentsRequested(payload["contents"]) {
		needContent = true
	}
	h.runAnswerProvider(ctx, deepseekSearchProviderName, "deepseek-search", "deepseek-search", h.deepseek, deepseekSearchPriceUSD, userID, apiKeyID,
		func(key string) (*answerResponse, int, error) {
			res, status, err := h.deepseekSearch(ctx, key, query, numResults, needContent)
			if err != nil {
				return nil, status, err
			}
			return &answerResponse{Provider: deepseekSearchProviderName, Model: "deepseek-search", Query: query, Citations: res, Results: res}, 200, nil
		})
}

// papersSearchRaw hits the papers.app.nz API and flattens hits into results.
func (h *SearchHandler) papersSearchRaw(ctx *fasthttp.RequestCtx, key, query string, limit int) ([]answerResult, error) {
	u := strings.TrimRight(h.papers.BaseURL, "/") + "/api/search?" + papersSearchQuery(nil, query, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("papers returned status %d", resp.StatusCode)
	}
	var parsed struct {
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	out := make([]answerResult, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		title, _ := r["title"].(string)
		link, _ := r["url"].(string)
		text, _ := r["abstract"].(string)
		if text == "" {
			text, _ = r["summary"].(string)
		}
		out = append(out, answerResult{Title: title, URL: link, Text: truncateRunes(text, 800)})
	}
	return out, nil
}

var openpathsAgentTools = []map[string]any{
	agentFn("web_search", "Search the live web. Returns results with title, url and summary. Supports operators like site:, \"phrase\", after:2026.", map[string]any{
		"query": map[string]any{"type": "string"},
		"count": map[string]any{"type": "integer", "description": "1-10, default 5"},
	}, []string{"query"}),
	agentFn("papers_search", "Search academic papers (arXiv etc). Returns title, url, abstract.", map[string]any{
		"query": map[string]any{"type": "string"},
	}, []string{"query"}),
	agentFn("fetch_page", "Fetch a web page and dump its readable text (like `lynx --dump`).", map[string]any{
		"url": map[string]any{"type": "string"},
	}, []string{"url"}),
}

func agentFn(name, desc string, props map[string]any, required []string) map[string]any {
	return map[string]any{"type": "function", "function": map[string]any{
		"name": name, "description": desc,
		"parameters": map[string]any{"type": "object", "properties": props, "required": required},
	}}
}

const openpathsAgentSystem = "You are a research agent. Use web_search, papers_search and fetch_page to gather evidence, " +
	"then answer concisely with inline [n] citations matching the order of sources you used. " +
	"Prefer primary sources, fetch pages when snippets are insufficient, and say so when evidence is thin."

// handleOpenPathsSearch runs a small tool-use loop: DeepSeek Flash decides
// when to call web search, paper search and page fetch, then writes a cited answer.
func (h *SearchHandler) handleOpenPathsSearch(ctx *fasthttp.RequestCtx, payload map[string]any, query, userID, apiKeyID string) {
	modelName := stringFromPayload(payload["model"], deepseekAgentModel)
	h.runAnswerProvider(ctx, deepseekSearchProviderName, modelName, openpathsSearchModel, h.deepseek, openpathsSearchPriceUSD, userID, apiKeyID,
		func(key string) (*answerResponse, int, error) {
			papersKey, _ := h.resolveProviderKey(ctx, papersProviderName, h.papers)
			out := &answerResponse{Provider: openpathsSearchProviderName, Model: modelName, Query: query}
			seen := map[string]bool{}
			addCites := func(rs []answerResult) {
				for _, r := range rs {
					if r.URL != "" && !seen[r.URL] {
						seen[r.URL] = true
						r.Text = ""
						out.Citations = append(out.Citations, r)
					}
				}
			}
			msgs := []map[string]any{
				{"role": "system", "content": openpathsAgentSystem},
				{"role": "user", "content": query},
			}
			for round := 0; round <= openpathsAgentMaxRounds; round++ {
				req := map[string]any{"model": modelName, "messages": msgs}
				if round < openpathsAgentMaxRounds {
					req["tools"] = openpathsAgentTools
				} else {
					msgs = append(msgs, map[string]any{"role": "user", "content": "Tool budget exhausted. Do not call tools. Write the final cited answer now from the evidence gathered."})
					req["messages"] = msgs
				}
				var cr struct {
					Choices []struct {
						Message struct {
							Content   string `json:"content"`
							ToolCalls []struct {
								ID       string `json:"id"`
								Function struct {
									Name      string `json:"name"`
									Arguments string `json:"arguments"`
								} `json:"function"`
							} `json:"tool_calls"`
						} `json:"message"`
					} `json:"choices"`
				}
				status, err := h.postJSON(ctx, strings.TrimRight(h.deepseek.BaseURL, "/")+"/chat/completions", h.deepseekHeaders(key), req, &cr)
				if err != nil {
					return nil, status, err
				}
				if len(cr.Choices) == 0 {
					return nil, 502, fmt.Errorf("empty response from %s", modelName)
				}
				m := cr.Choices[0].Message
				if len(m.ToolCalls) == 0 {
					out.Answer = strings.TrimSpace(m.Content)
					if strings.Contains(out.Answer, "DSML") {
						return nil, 502, fmt.Errorf("agent did not produce a final answer")
					}
					if len(out.Citations) > openpathsMaxCitations {
						out.Citations = out.Citations[:openpathsMaxCitations]
					}
					out.Results = out.Citations
					return out, 200, nil
				}
				calls := make([]map[string]any, 0, len(m.ToolCalls))
				for _, tc := range m.ToolCalls {
					calls = append(calls, map[string]any{"id": tc.ID, "type": "function",
						"function": map[string]any{"name": tc.Function.Name, "arguments": tc.Function.Arguments}})
				}
				msgs = append(msgs, map[string]any{"role": "assistant", "content": m.Content, "tool_calls": calls})
				for _, tc := range m.ToolCalls {
					var args map[string]any
					_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
					result := h.runAgentTool(ctx, key, papersKey, tc.Function.Name, args, out, addCites)
					msgs = append(msgs, map[string]any{"role": "tool", "tool_call_id": tc.ID, "content": result})
				}
			}
			return nil, 502, fmt.Errorf("agent did not produce a final answer")
		})
}

func (h *SearchHandler) runAgentTool(ctx *fasthttp.RequestCtx, key, papersKey, name string, args map[string]any, out *answerResponse, addCites func([]answerResult)) string {
	q := stringFromPayload(args["query"], "")
	var rs []answerResult
	var err error
	switch name {
	case "web_search":
		out.SearchQueries = append(out.SearchQueries, q)
		n := intFromPayload(args["count"], 5)
		if n > 10 {
			n = 10
		}
		rs, _, err = h.deepseekSearch(ctx, key, q, n, false)
	case "papers_search":
		if papersKey == "" {
			return "papers search unavailable"
		}
		out.SearchQueries = append(out.SearchQueries, "papers: "+q)
		rs, err = h.papersSearchRaw(ctx, papersKey, q, 5)
	case "fetch_page":
		u := stringFromPayload(args["url"], "")
		if pu, perr := url.Parse(u); perr != nil || (pu.Scheme != "http" && pu.Scheme != "https") {
			return "error: invalid url"
		}
		txt, ferr := h.deepseekFetch(ctx, key, u)
		if ferr != nil {
			return "error: " + ferr.Error()
		}
		addCites([]answerResult{{URL: u, Title: citationTitleFromURL(u)}})
		return txt
	default:
		return "error: unknown tool"
	}
	if err != nil {
		return "error: " + err.Error()
	}
	addCites(rs)
	var buf bytes.Buffer
	for i, r := range rs {
		fmt.Fprintf(&buf, "[%d] %s\n%s\n%s\n\n", i+1, r.Title, r.URL, truncateRunes(r.Text, 700))
	}
	if buf.Len() == 0 {
		return "no results"
	}
	return buf.String()
}
