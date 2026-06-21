package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ResearchSource is one normalized search hit / fetched document used by the
// deep-research engine.
type ResearchSource struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Snippet  string `json:"snippet,omitempty"`
	Text     string `json:"text,omitempty"`
	Provider string `json:"provider"`
}

// researchSearchClient wraps the exa + papers backends with the server-side
// keys (no BYOK) for internal tool use by the research engine.
type researchSearchClient struct {
	exa    SearchProviderConfig
	papers SearchProviderConfig
	client *http.Client
}

func newResearchSearchClient(exa, papers SearchProviderConfig) *researchSearchClient {
	if exa.BaseURL == "" {
		exa.BaseURL = exaDefaultBaseURL
	}
	if papers.BaseURL == "" {
		papers.BaseURL = papersDefaultBaseURL
	}
	return &researchSearchClient{exa: exa, papers: papers, client: &http.Client{Timeout: 45 * time.Second}}
}

// webSearch runs an exa search with page text included.
func (c *researchSearchClient) webSearch(ctx context.Context, query string, n int) ([]ResearchSource, error) {
	if c.exa.APIKey == "" {
		return nil, fmt.Errorf("exa not configured")
	}
	if n < 1 {
		n = 6
	}
	body, _ := json.Marshal(map[string]any{
		"query":      query,
		"numResults": n,
		"type":       "auto",
		"contents":   map[string]any{"text": map[string]any{"maxCharacters": 2400}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.exa.BaseURL, "/")+"/search", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.exa.APIKey)
	raw, status, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("exa status %d", status)
	}
	var parsed struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Text    string `json:"text"`
			Snippet string `json:"snippet"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	out := make([]ResearchSource, 0, len(parsed.Results))
	for _, r := range parsed.Results {
		if strings.TrimSpace(r.URL) == "" {
			continue
		}
		out = append(out, ResearchSource{
			Title: strings.TrimSpace(r.Title), URL: r.URL,
			Snippet: truncate(firstNonEmpty(r.Snippet, r.Text), 400),
			Text:    truncate(r.Text, 2400), Provider: "exa",
		})
	}
	return out, nil
}

// papersSearch queries papers.app.nz for academic sources.
func (c *researchSearchClient) papersSearch(ctx context.Context, query string, n int) ([]ResearchSource, error) {
	if c.papers.APIKey == "" {
		return nil, fmt.Errorf("papers not configured")
	}
	if n < 1 {
		n = 6
	}
	u := fmt.Sprintf("%s/api/search?q=%s&limit=%d&type=papers", strings.TrimRight(c.papers.BaseURL, "/"), urlQueryEscape(query), n)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.papers.APIKey)
	raw, status, err := c.do(req)
	if err != nil {
		return nil, err
	}
	if status < 200 || status >= 300 {
		return nil, fmt.Errorf("papers status %d", status)
	}
	return parsePapersResults(raw), nil
}

// fetchURL returns readable text for a URL, preferring `lynx --dump` and
// falling back to a plain HTTP GET with naive HTML stripping.
func (c *researchSearchClient) fetchURL(ctx context.Context, target string) (string, error) {
	if txt := lynxDump(ctx, target); strings.TrimSpace(txt) != "" {
		return truncate(txt, 6000), nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; OpenPathsResearch/1.0)")
	raw, status, err := c.do(req)
	if err != nil {
		return "", err
	}
	if status < 200 || status >= 300 {
		return "", fmt.Errorf("fetch status %d", status)
	}
	return truncate(stripHTML(string(raw)), 6000), nil
}

func (c *researchSearchClient) do(req *http.Request) ([]byte, int, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return raw, resp.StatusCode, err
}

func lynxDump(ctx context.Context, target string) string {
	if _, err := exec.LookPath("lynx"); err != nil {
		return ""
	}
	cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "lynx", "-dump", "-nolist", "-width=120", target)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// parsePapersResults defensively extracts sources from the papers.app.nz
// response, which may nest the list under several keys.
func parsePapersResults(raw []byte) []ResearchSource {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return nil
	}
	var list []map[string]any
	for _, key := range []string{"results", "papers", "data", "items", "hits"} {
		if v, ok := top[key]; ok {
			_ = json.Unmarshal(v, &list)
			if len(list) > 0 {
				break
			}
		}
	}
	out := make([]ResearchSource, 0, len(list))
	for _, item := range list {
		url := firstStr(item, "url", "link", "pdf_url", "arxiv_url", "abs_url")
		title := firstStr(item, "title", "name")
		if url == "" && title == "" {
			continue
		}
		out = append(out, ResearchSource{
			Title:    strings.TrimSpace(title),
			URL:      url,
			Snippet:  truncate(firstStr(item, "abstract", "summary", "snippet", "description"), 600),
			Provider: "papers",
		})
	}
	return out
}

func firstStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

var htmlTagRe = regexp.MustCompile(`(?s)<(script|style)[^>]*>.*?</(script|style)>|<[^>]+>`)
var wsRe = regexp.MustCompile(`[ \t]*\n[ \t\n]*`)

func stripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = wsRe.ReplaceAllString(s, "\n")
	return strings.TrimSpace(s)
}

func urlQueryEscape(s string) string {
	return strings.NewReplacer(" ", "%20", "&", "%26", "?", "%3F", "#", "%23", "+", "%2B").Replace(s)
}
