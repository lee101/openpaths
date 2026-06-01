package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/valyala/fasthttp"
)

func TestSearchHandlerForwardsExaSearchWithBYOK(t *testing.T) {
	var gotKey string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Fatalf("path = %s, want /search", r.URL.Path)
		}
		gotKey = r.Header.Get("x-api-key")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode upstream body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"requestId": "req_test",
			"resolvedSearchType": "auto",
			"results": [{
				"id": "https://example.test/a",
				"title": "Example result",
				"url": "https://example.test/a",
				"highlights": ["Relevant excerpt"]
			}]
		}`))
	}))
	defer upstream.Close()

	h := NewSearchHandler(
		SearchProviderConfig{BaseURL: upstream.URL, APIKey: "platform-key", Enabled: true},
		SearchProviderConfig{},
		nil,
		metrics.NewRecorder(metrics.NewCollector(nil, time.Hour)),
	)

	ctx := &fasthttp.RequestCtx{}
	var req fasthttp.Request
	req.Header.SetMethod(http.MethodPost)
	req.SetRequestURI("/v1/search")
	req.Header.SetContentType("application/json")
	req.SetBodyString(`{
		"query": "Latest news on Nvidia",
		"numResults": 10,
		"type": "auto",
		"contents": { "highlights": true }
	}`)
	ctx.Init(&req, nil, nil)
	ctx.SetUserValue(middleware.CtxKeyUserID, "u1")
	ctx.SetUserValue(middleware.CtxKeyUserProviderKeys, map[string]*queries.UserProviderKey{
		"exa": {Provider: "exa", APIKey: "user-exa-key"},
	})

	h.HandleSearch(ctx)

	if ctx.Response.StatusCode() != 200 {
		t.Fatalf("status = %d, body = %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if gotKey != "user-exa-key" {
		t.Fatalf("x-api-key = %q, want user-exa-key", gotKey)
	}
	if gotBody["query"] != "Latest news on Nvidia" {
		t.Fatalf("query = %#v", gotBody["query"])
	}
	if gotBody["type"] != "auto" {
		t.Fatalf("type = %#v", gotBody["type"])
	}

	var resp struct {
		RequestID string `json:"requestId"`
		Results   []struct {
			Title      string   `json:"title"`
			Highlights []string `json:"highlights"`
		} `json:"results"`
	}
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.RequestID != "req_test" || len(resp.Results) != 1 || resp.Results[0].Title != "Example result" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestSearchHandlerValidatesQuery(t *testing.T) {
	h := NewSearchHandler(
		SearchProviderConfig{BaseURL: "https://api.exa.ai", APIKey: "platform-key", Enabled: true},
		SearchProviderConfig{},
		nil,
		metrics.NewRecorder(metrics.NewCollector(nil, time.Hour)),
	)
	ctx := &fasthttp.RequestCtx{}
	var req fasthttp.Request
	req.Header.SetMethod(http.MethodPost)
	req.SetRequestURI("/v1/search")
	req.Header.SetContentType("application/json")
	req.SetBodyString(`{"numResults": 10}`)
	ctx.Init(&req, nil, nil)

	h.HandleSearch(ctx)

	if ctx.Response.StatusCode() != 400 {
		t.Fatalf("status = %d, body = %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
}

func TestSearchHandlerForwardsPapersSearchWithBYOK(t *testing.T) {
	var gotKey string
	var gotQuery = make(map[string]string)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/search" {
			t.Fatalf("path = %s, want /api/search", r.URL.Path)
		}
		gotKey = r.Header.Get("Authorization")
		for key, value := range r.URL.Query() {
			if len(value) > 0 {
				gotQuery[key] = value[0]
			}
		}
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		_, _ = w.Write([]byte("# Papers Search\n\n- Query: diffusion\n"))
	}))
	defer upstream.Close()

	h := NewSearchHandler(
		SearchProviderConfig{},
		SearchProviderConfig{BaseURL: upstream.URL, APIKey: "platform-papers-key", Enabled: true},
		nil,
		metrics.NewRecorder(metrics.NewCollector(nil, time.Hour)),
	)

	ctx := &fasthttp.RequestCtx{}
	var req fasthttp.Request
	req.Header.SetMethod(http.MethodPost)
	req.SetRequestURI("/v1/search")
	req.Header.SetContentType("application/json")
	req.SetBodyString(`{
		"provider": "papers",
		"query": "diffusion",
		"numResults": 5,
		"type": "papers",
		"format": "markdown",
		"hasCode": true
	}`)
	ctx.Init(&req, nil, nil)
	ctx.SetUserValue(middleware.CtxKeyUserID, "u1")
	ctx.SetUserValue(middleware.CtxKeyUserProviderKeys, map[string]*queries.UserProviderKey{
		"papers": {Provider: "papers", APIKey: "user-papers-key"},
	})

	h.HandleSearch(ctx)

	if ctx.Response.StatusCode() != 200 {
		t.Fatalf("status = %d, body = %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if gotKey != "Bearer user-papers-key" {
		t.Fatalf("authorization = %q, want Bearer user-papers-key", gotKey)
	}
	if gotQuery["q"] != "diffusion" || gotQuery["type"] != "papers" || gotQuery["limit"] != "5" || gotQuery["format"] != "markdown" || gotQuery["has_code"] != "true" {
		t.Fatalf("unexpected query params: %#v", gotQuery)
	}
	if !strings.Contains(string(ctx.Response.Body()), "Papers Search") {
		t.Fatalf("unexpected body: %s", ctx.Response.Body())
	}
}

func newAnswerTestCtx(t *testing.T, body string, byok map[string]*queries.UserProviderKey) *fasthttp.RequestCtx {
	t.Helper()
	ctx := &fasthttp.RequestCtx{}
	var req fasthttp.Request
	req.Header.SetMethod(http.MethodPost)
	req.SetRequestURI("/v1/search")
	req.Header.SetContentType("application/json")
	req.SetBodyString(body)
	ctx.Init(&req, nil, nil)
	ctx.SetUserValue(middleware.CtxKeyUserID, "u1")
	if byok != nil {
		ctx.SetUserValue(middleware.CtxKeyUserProviderKeys, byok)
	}
	return ctx
}

func TestSearchHandlerGeminiSearch(t *testing.T) {
	var gotPath, gotKey string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.URL.Query().Get("key")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"candidates": [{
				"content": { "parts": [{ "text": "Nvidia released a new GPU." }] },
				"groundingMetadata": {
					"webSearchQueries": ["nvidia news"],
					"groundingChunks": [
						{ "web": { "uri": "https://example.test/nvidia", "title": "Nvidia News" } }
					]
				}
			}]
		}`))
	}))
	defer upstream.Close()

	h := NewSearchHandler(SearchProviderConfig{}, SearchProviderConfig{}, nil,
		metrics.NewRecorder(metrics.NewCollector(nil, time.Hour))).
		SetAnswerProviders(SearchProviderConfig{BaseURL: upstream.URL, APIKey: "platform-gemini", Enabled: true}, SearchProviderConfig{}, SearchProviderConfig{})

	ctx := newAnswerTestCtx(t, `{"provider":"gemini","query":"Latest news on Nvidia"}`, nil)
	h.HandleSearch(ctx)

	if ctx.Response.StatusCode() != 200 {
		t.Fatalf("status = %d, body = %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if !strings.Contains(gotPath, "generateContent") {
		t.Fatalf("path = %s", gotPath)
	}
	if gotKey != "platform-gemini" {
		t.Fatalf("key = %q", gotKey)
	}
	if _, ok := gotBody["tools"]; !ok {
		t.Fatalf("expected google_search tools in body: %#v", gotBody)
	}
	var resp answerResponse
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Provider != "gemini" || resp.Answer != "Nvidia released a new GPU." {
		t.Fatalf("unexpected answer: %+v", resp)
	}
	if len(resp.Citations) != 1 || resp.Citations[0].URL != "https://example.test/nvidia" {
		t.Fatalf("unexpected citations: %+v", resp.Citations)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results should mirror citations: %+v", resp.Results)
	}
}

func TestSearchHandlerOpenAISearchWithBYOK(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"output": [{
				"type": "message",
				"content": [{
					"type": "output_text",
					"text": "OpenAI answer.",
					"annotations": [{ "type": "url_citation", "url": "https://example.test/a", "title": "Source A" }]
				}]
			}]
		}`))
	}))
	defer upstream.Close()

	h := NewSearchHandler(SearchProviderConfig{}, SearchProviderConfig{}, nil,
		metrics.NewRecorder(metrics.NewCollector(nil, time.Hour))).
		SetAnswerProviders(SearchProviderConfig{}, SearchProviderConfig{BaseURL: upstream.URL, APIKey: "platform-openai", Enabled: true}, SearchProviderConfig{})

	ctx := newAnswerTestCtx(t, `{"provider":"openai","query":"what is rag"}`, map[string]*queries.UserProviderKey{
		"openai": {Provider: "openai", APIKey: "user-openai-key"},
	})
	h.HandleSearch(ctx)

	if ctx.Response.StatusCode() != 200 {
		t.Fatalf("status = %d, body = %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if gotPath != "/v1/responses" {
		t.Fatalf("path = %s", gotPath)
	}
	if gotAuth != "Bearer user-openai-key" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if _, ok := gotBody["tools"]; !ok {
		t.Fatalf("expected web_search tools: %#v", gotBody)
	}
	var resp answerResponse
	_ = json.Unmarshal(ctx.Response.Body(), &resp)
	if resp.Answer != "OpenAI answer." || len(resp.Citations) != 1 || resp.Citations[0].URL != "https://example.test/a" {
		t.Fatalf("unexpected resp: %+v", resp)
	}
}

func TestSearchHandlerGrokSearch(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{ "message": { "content": "Grok answer." } }],
			"citations": ["https://www.example.test/x"]
		}`))
	}))
	defer upstream.Close()

	h := NewSearchHandler(SearchProviderConfig{}, SearchProviderConfig{}, nil,
		metrics.NewRecorder(metrics.NewCollector(nil, time.Hour))).
		SetAnswerProviders(SearchProviderConfig{}, SearchProviderConfig{}, SearchProviderConfig{BaseURL: upstream.URL, APIKey: "platform-xai", Enabled: true})

	ctx := newAnswerTestCtx(t, `{"provider":"grok","query":"latest ai"}`, nil)
	h.HandleSearch(ctx)

	if ctx.Response.StatusCode() != 200 {
		t.Fatalf("status = %d, body = %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("path = %s", gotPath)
	}
	if _, ok := gotBody["search_parameters"]; !ok {
		t.Fatalf("expected search_parameters: %#v", gotBody)
	}
	var resp answerResponse
	_ = json.Unmarshal(ctx.Response.Body(), &resp)
	if resp.Answer != "Grok answer." || len(resp.Citations) != 1 {
		t.Fatalf("unexpected resp: %+v", resp)
	}
	if resp.Citations[0].Title != "example.test" {
		t.Fatalf("expected derived title, got %q", resp.Citations[0].Title)
	}
}

func TestSearchHandlerUnknownProvider(t *testing.T) {
	h := NewSearchHandler(SearchProviderConfig{}, SearchProviderConfig{}, nil,
		metrics.NewRecorder(metrics.NewCollector(nil, time.Hour)))
	ctx := newAnswerTestCtx(t, `{"provider":"bing","query":"x"}`, nil)
	h.HandleSearch(ctx)
	if ctx.Response.StatusCode() != 400 {
		t.Fatalf("status = %d", ctx.Response.StatusCode())
	}
}
