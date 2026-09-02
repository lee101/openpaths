package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/router"
)

const (
	defaultResearchModel = "deepseek-v4-pro"
	cheapResearchModel   = "deepseek-v4-flash"
)

// researchEngine orchestrates the deep-research pipeline. The default models are
// all DeepSeek (cheap reasoning); verify/synthesize may be pointed at a premium
// model per request for the "premium adjudication" tier.
type researchEngine struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
	search   *researchSearchClient
	jobQ     *queries.ResearchJobQueries

	plannerModel string
	extractModel string
	verifyModel  string
	synthModel   string
}

type researchTuning struct {
	maxQueries int
	maxSources int
	maxFetches int
	maxClaims  int
}

func tuningForDepth(depth string) researchTuning {
	switch depth {
	case "deep":
		return researchTuning{maxQueries: 6, maxSources: 18, maxFetches: 8, maxClaims: 30}
	case "quick":
		return researchTuning{maxQueries: 2, maxSources: 6, maxFetches: 2, maxClaims: 10}
	default: // standard
		return researchTuning{maxQueries: 4, maxSources: 10, maxFetches: 4, maxClaims: 18}
	}
}

// run executes the full pipeline, persisting each step. It always finalizes the
// job (Complete or Fail).
func (e *researchEngine) run(ctx context.Context, jobID, userID, query, depth string) {
	_ = e.jobQ.MarkRunning(ctx, jobID)
	var cost int64
	if err := e.pipeline(ctx, jobID, userID, query, depth, &cost); err != nil {
		_ = e.jobQ.Fail(context.Background(), jobID, "research_error", err.Error())
	}
}

func (e *researchEngine) pipeline(ctx context.Context, jobID, userID, query, depth string, cost *int64) error {
	t := tuningForDepth(depth)

	// 1. PLAN
	plan, err := e.plan(ctx, userID, query, t, cost)
	if err != nil {
		return fmt.Errorf("plan: %w", err)
	}
	e.step(ctx, jobID, "plan", "Planned research", fmt.Sprintf("%d sub-questions, %d search queries", len(plan.SubQuestions), len(plan.Queries)), plan)

	// 2. SEARCH
	sources := e.gather(ctx, plan.Queries, t)
	e.step(ctx, jobID, "search", "Collected sources", fmt.Sprintf("%d unique sources from %d queries", len(sources), len(plan.Queries)), sources)
	if len(sources) == 0 {
		return fmt.Errorf("no sources found")
	}

	// 3. READ — enrich sources lacking text
	read := e.read(ctx, sources, t)
	e.step(ctx, jobID, "read", "Read sources", fmt.Sprintf("fetched full text for %d sources", read), nil)

	// 4. EXTRACT
	claims, err := e.extract(ctx, userID, query, sources, t, cost)
	if err != nil {
		return fmt.Errorf("extract: %w", err)
	}
	e.step(ctx, jobID, "extract", "Extracted claims", fmt.Sprintf("%d candidate claims", len(claims)), claims)

	// 5. VERIFY (independent critique) + mechanical GATE
	verified := e.verifyAndGate(ctx, userID, sources, claims, cost)
	e.step(ctx, jobID, "verify", "Verified claims", fmt.Sprintf("%d supported, %d dropped", len(verified), len(claims)-len(verified)), verified)
	if len(verified) == 0 {
		return fmt.Errorf("no claims survived verification")
	}

	// 6. SYNTHESIZE
	report, citations := e.synthesize(ctx, userID, query, verified, sources, cost)
	e.step(ctx, jobID, "synthesize", "Wrote report", fmt.Sprintf("%d citations", len(citations)), nil)

	cj, _ := json.Marshal(citations)
	return e.jobQ.Complete(context.Background(), jobID, report, cj, *cost)
}

// ---- phases ----

type researchPlan struct {
	SubQuestions []string `json:"sub_questions"`
	Queries      []string `json:"queries"`
}

func (e *researchEngine) plan(ctx context.Context, userID, query string, t researchTuning, cost *int64) (researchPlan, error) {
	sys := "You are a meticulous research planner. Decompose the user's question into focused sub-questions and concrete web search queries. " +
		"Respond ONLY with JSON: {\"sub_questions\":[string],\"queries\":[string]}. " +
		fmt.Sprintf("Provide at most %d queries, each a short standalone search string.", t.maxQueries)
	out, err := e.chat(ctx, userID, e.plannerModel, []model.ChatMessage{
		{Role: "system", Content: sys},
		{Role: "user", Content: query},
	}, true, cost)
	if err != nil {
		return researchPlan{}, err
	}
	var p researchPlan
	if err := decodeJSON(out, &p); err != nil {
		// fall back to the raw query as a single search
		return researchPlan{SubQuestions: []string{query}, Queries: []string{query}}, nil
	}
	if len(p.Queries) == 0 {
		p.Queries = []string{query}
	}
	if len(p.Queries) > t.maxQueries {
		p.Queries = p.Queries[:t.maxQueries]
	}
	return p, nil
}

func (e *researchEngine) gather(ctx context.Context, queries []string, t researchTuning) []ResearchSource {
	seen := map[string]bool{}
	var out []ResearchSource
	add := func(srcs []ResearchSource) {
		for _, s := range srcs {
			key := normalizeURL(s.URL)
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, s)
		}
	}
	for _, q := range queries {
		if len(out) >= t.maxSources {
			break
		}
		if web, err := e.search.webSearch(ctx, q, 5); err == nil {
			add(web)
		}
		if pap, err := e.search.papersSearch(ctx, q, 3); err == nil {
			add(pap)
		}
	}
	if len(out) > t.maxSources {
		out = out[:t.maxSources]
	}
	return out
}

func (e *researchEngine) read(ctx context.Context, sources []ResearchSource, t researchTuning) int {
	fetched := 0
	for i := range sources {
		if fetched >= t.maxFetches {
			break
		}
		if strings.TrimSpace(sources[i].Text) != "" {
			continue
		}
		if txt, err := e.search.fetchURL(ctx, sources[i].URL); err == nil && strings.TrimSpace(txt) != "" {
			sources[i].Text = txt
			fetched++
		}
	}
	return fetched
}

type researchClaim struct {
	Claim  string `json:"claim"`
	Source string `json:"source"`
	Quote  string `json:"quote,omitempty"`
}

func (e *researchEngine) extract(ctx context.Context, userID, query string, sources []ResearchSource, t researchTuning, cost *int64) ([]researchClaim, error) {
	sys := "You extract factual claims relevant to the research question from the provided sources. " +
		"Each claim must be grounded in a specific source URL with a short supporting quote. " +
		"Respond ONLY with JSON: {\"claims\":[{\"claim\":string,\"source\":string(url),\"quote\":string}]}. " +
		fmt.Sprintf("Extract at most %d of the most relevant, verifiable claims.", t.maxClaims)
	user := "QUESTION:\n" + query + "\n\nSOURCES:\n" + sourcesDigest(sources, 1800)
	out, err := e.chat(ctx, userID, e.extractModel, []model.ChatMessage{
		{Role: "system", Content: sys},
		{Role: "user", Content: user},
	}, true, cost)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Claims []researchClaim `json:"claims"`
	}
	if err := decodeJSON(out, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Claims) > t.maxClaims {
		parsed.Claims = parsed.Claims[:t.maxClaims]
	}
	return parsed.Claims, nil
}

// verifyAndGate runs an INDEPENDENT verifier over the claims (skeptical system
// prompt, separate call) then mechanically drops any claim that the verifier
// did not support or whose source is not in our source set.
func (e *researchEngine) verifyAndGate(ctx context.Context, userID string, sources []ResearchSource, claims []researchClaim, cost *int64) []researchClaim {
	if len(claims) == 0 {
		return nil
	}
	srcByURL := map[string]ResearchSource{}
	for _, s := range sources {
		srcByURL[normalizeURL(s.URL)] = s
	}
	sys := "You are a skeptical fact-checker. For each numbered claim, decide whether the cited source text actually supports it. " +
		"Be strict: if the source does not clearly support the claim, mark supported=false. " +
		"Respond ONLY with JSON: {\"verdicts\":[{\"index\":int,\"supported\":bool,\"reason\":string}]}."
	var sb strings.Builder
	for i, c := range claims {
		src := srcByURL[normalizeURL(c.Source)]
		fmt.Fprintf(&sb, "[%d] CLAIM: %s\nSOURCE URL: %s\nSOURCE TEXT: %s\n\n", i, c.Claim, c.Source, truncate(firstNonEmpty(src.Text, src.Snippet), 1200))
	}
	out, err := e.chat(ctx, userID, e.verifyModel, []model.ChatMessage{
		{Role: "system", Content: sys},
		{Role: "user", Content: sb.String()},
	}, true, cost)
	if err != nil {
		// verifier unavailable: fail safe — keep only claims with a known source
		return gateBySourceOnly(claims, srcByURL)
	}
	var parsed struct {
		Verdicts []struct {
			Index     int  `json:"index"`
			Supported bool `json:"supported"`
		} `json:"verdicts"`
	}
	if err := decodeJSON(out, &parsed); err != nil {
		return gateBySourceOnly(claims, srcByURL)
	}
	supported := map[int]bool{}
	for _, v := range parsed.Verdicts {
		supported[v.Index] = v.Supported
	}
	var kept []researchClaim
	for i, c := range claims {
		if !supported[i] {
			continue
		}
		if _, ok := srcByURL[normalizeURL(c.Source)]; !ok {
			continue
		}
		kept = append(kept, c)
	}
	return kept
}

func gateBySourceOnly(claims []researchClaim, srcByURL map[string]ResearchSource) []researchClaim {
	var kept []researchClaim
	for _, c := range claims {
		if _, ok := srcByURL[normalizeURL(c.Source)]; ok {
			kept = append(kept, c)
		}
	}
	return kept
}

type researchCitation struct {
	N     int    `json:"n"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

func (e *researchEngine) synthesize(ctx context.Context, userID, query string, claims []researchClaim, sources []ResearchSource, cost *int64) (string, []researchCitation) {
	// Number the sources that back surviving claims.
	citations, indexByURL := buildCitations(claims, sources)
	sys := "You are a research analyst. Write a clear, well-structured markdown report answering the question using ONLY the verified claims. " +
		"Cite sources inline with [n] matching the provided citation list. Note disagreements and gaps. Do not invent facts."
	var sb strings.Builder
	sb.WriteString("QUESTION:\n" + query + "\n\nVERIFIED CLAIMS:\n")
	for _, c := range claims {
		n := indexByURL[normalizeURL(c.Source)]
		fmt.Fprintf(&sb, "- %s [%d]\n", c.Claim, n)
	}
	sb.WriteString("\nCITATIONS:\n")
	for _, c := range citations {
		fmt.Fprintf(&sb, "[%d] %s - %s\n", c.N, firstNonEmpty(c.Title, c.URL), c.URL)
	}
	out, err := e.chat(ctx, userID, e.synthModel, []model.ChatMessage{
		{Role: "system", Content: sys},
		{Role: "user", Content: sb.String()},
	}, false, cost)
	if err != nil {
		out = "Report generation failed: " + err.Error()
	}
	return out, citations
}

func buildCitations(claims []researchClaim, sources []ResearchSource) ([]researchCitation, map[string]int) {
	srcByURL := map[string]ResearchSource{}
	for _, s := range sources {
		srcByURL[normalizeURL(s.URL)] = s
	}
	indexByURL := map[string]int{}
	var cites []researchCitation
	n := 0
	for _, c := range claims {
		key := normalizeURL(c.Source)
		if key == "" || indexByURL[key] != 0 {
			continue
		}
		n++
		indexByURL[key] = n
		s := srcByURL[key]
		cites = append(cites, researchCitation{N: n, Title: s.Title, URL: firstNonEmpty(s.URL, c.Source)})
	}
	sort.Slice(cites, func(i, j int) bool { return cites[i].N < cites[j].N })
	return cites, indexByURL
}

// ---- LLM seam ----

func (e *researchEngine) chat(ctx context.Context, userID, modelID string, msgs []model.ChatMessage, jsonMode bool, cost *int64) (string, error) {
	if modelID == "" {
		modelID = defaultResearchModel
	}
	candidates, err := e.router.ResolveForRequest(modelID, "")
	if err != nil || len(candidates) == 0 {
		return "", fmt.Errorf("resolve model %q: %v", modelID, err)
	}
	maxTok := 4096
	var lastErr error
	for _, cand := range candidates {
		req := &model.ChatCompletionRequest{
			Model:     cand.ModelCfg.ProviderModelID,
			Messages:  msgs,
			MaxTokens: &maxTok,
		}
		if jsonMode {
			req.ResponseFormat = &model.ResponseFormat{Type: "json_object"}
		}
		resp, err := cand.Provider.ChatCompletion(ctx, req)
		if err != nil {
			lastErr = err
			continue
		}
		if len(resp.Choices) == 0 || resp.Choices[0].Message == nil {
			lastErr = fmt.Errorf("empty response from %s", cand.ModelCfg.ID)
			continue
		}
		if e.billing != nil && resp.Usage != nil {
			c, _ := e.billing.Deduct(ctx, userID, cand.ModelCfg.ID, resp.Usage.PromptTokens, resp.Usage.CompletionTokens, "", "")
			*cost += c
		}
		if e.recorder != nil && resp.Usage != nil {
			e.recorder.RecordSuccess(userID, "", "research/"+cand.ModelCfg.ID, cand.Provider.Name(),
				resp.Usage.PromptTokens, resp.Usage.CompletionTokens, 0, 0, *cost, false)
		}
		return contentToString(resp.Choices[0].Message.Content), nil
	}
	return "", fmt.Errorf("all candidates failed for %q: %v", modelID, lastErr)
}

func (e *researchEngine) step(ctx context.Context, jobID, phase, title, detail string, data any) {
	st := map[string]any{
		"phase":  phase,
		"title":  title,
		"detail": detail,
		"status": "done",
		"ts":     time.Now().UTC().Format(time.RFC3339),
	}
	if data != nil {
		st["data"] = data
	}
	b, err := json.Marshal([]any{st})
	if err != nil {
		return
	}
	_ = e.jobQ.AppendStep(context.Background(), jobID, b)
}

// ---- helpers ----

func sourcesDigest(sources []ResearchSource, perSource int) string {
	var sb strings.Builder
	for _, s := range sources {
		fmt.Fprintf(&sb, "URL: %s\nTITLE: %s\nTEXT: %s\n\n", s.URL, s.Title, truncate(firstNonEmpty(s.Text, s.Snippet), perSource))
	}
	return sb.String()
}

func contentToString(v any) string {
	switch c := v.(type) {
	case string:
		return c
	case []any:
		var sb strings.Builder
		for _, part := range c {
			if m, ok := part.(map[string]any); ok {
				if txt, ok := m["text"].(string); ok {
					sb.WriteString(txt)
				}
			}
		}
		return sb.String()
	default:
		return ""
	}
}

func normalizeURL(u string) string {
	u = strings.TrimSpace(strings.ToLower(u))
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	u = strings.TrimPrefix(u, "www.")
	return strings.TrimRight(u, "/")
}

// decodeJSON robustly extracts a JSON object from an LLM response (handles
// fenced code blocks and surrounding prose) and unmarshals it into v.
func decodeJSON(s string, v any) error {
	raw := extractJSONObject(s)
	if raw == "" {
		return fmt.Errorf("no JSON object found")
	}
	if err := json.Unmarshal([]byte(raw), v); err == nil {
		return nil
	}
	return json.Unmarshal([]byte(repairJSON(raw)), v)
}

func extractJSONObject(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "```"); i >= 0 {
		rest := s[i+3:]
		rest = strings.TrimPrefix(rest, "json")
		rest = strings.TrimPrefix(rest, "JSON")
		if j := strings.Index(rest, "```"); j >= 0 {
			s = strings.TrimSpace(rest[:j])
		}
	}
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	depth, inStr, esc := 0, false, false
	for i := start; i < len(s); i++ {
		ch := s[i]
		switch {
		case esc:
			esc = false
		case ch == '\\':
			esc = true
		case ch == '"':
			inStr = !inStr
		case inStr:
		case ch == '{':
			depth++
		case ch == '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

func repairJSON(s string) string {
	// drop trailing commas before } or ]
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			j := i + 1
			for j < len(s) && (s[j] == ' ' || s[j] == '\n' || s[j] == '\t' || s[j] == '\r') {
				j++
			}
			if j < len(s) && (s[j] == '}' || s[j] == ']') {
				continue
			}
		}
		sb.WriteByte(s[i])
	}
	return sb.String()
}
