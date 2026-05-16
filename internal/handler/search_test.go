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
