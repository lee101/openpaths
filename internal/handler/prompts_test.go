package handler

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/valyala/fasthttp"
)

// newGetCtx builds a fasthttp request context for a GET against the given URI,
// optionally setting a {slug} user value for param routes.
func newGetCtx(uri, slug string) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	var req fasthttp.Request
	req.Header.SetMethod(http.MethodGet)
	req.SetRequestURI(uri)
	ctx.Init(&req, nil, nil)
	if slug != "" {
		ctx.SetUserValue("slug", slug)
	}
	return ctx
}

func decodeBody(t *testing.T, ctx *fasthttp.RequestCtx) map[string]any {
	t.Helper()
	if ctx.Response.StatusCode() != 200 {
		t.Fatalf("status = %d, body = %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	var out map[string]any
	if err := json.Unmarshal(ctx.Response.Body(), &out); err != nil {
		t.Fatalf("decode body: %v (%s)", err, ctx.Response.Body())
	}
	return out
}

func TestPromptsHandlerList(t *testing.T) {
	h := NewPromptsHandler(nil) // nil index -> lexical path

	// Plain list.
	ctx := newGetCtx("/v1/prompts?limit=10", "")
	h.HandleList(ctx)
	body := decodeBody(t, ctx)
	results, ok := body["results"].([]any)
	if !ok || len(results) == 0 {
		t.Fatalf("expected results, got %#v", body["results"])
	}
	if body["semantic"] != false {
		t.Fatalf("expected semantic=false with nil index, got %#v", body["semantic"])
	}

	// Lexical search.
	ctx = newGetCtx("/v1/prompts?q=pytorch", "")
	h.HandleList(ctx)
	body = decodeBody(t, ctx)
	results = body["results"].([]any)
	if len(results) == 0 {
		t.Fatal("expected search results for pytorch")
	}
}

func TestPromptsHandlerMeta(t *testing.T) {
	h := NewPromptsHandler(nil)
	ctx := newGetCtx("/v1/prompts/meta", "")
	h.HandleMeta(ctx)
	body := decodeBody(t, ctx)
	for _, key := range []string{"total", "categories", "models", "types", "counts"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("meta missing %q", key)
		}
	}
	cats, ok := body["categories"].([]any)
	if !ok || len(cats) == 0 {
		t.Fatal("expected categories")
	}
}

func TestPromptsHandlerGet(t *testing.T) {
	h := NewPromptsHandler(nil)

	ctx := newGetCtx("/v1/prompts/autocomplete-pytorch-coding", "autocomplete-pytorch-coding")
	h.HandleGet(ctx)
	body := decodeBody(t, ctx)
	prompt, ok := body["prompt"].(map[string]any)
	if !ok {
		t.Fatalf("expected prompt object, got %#v", body["prompt"])
	}
	if prompt["slug"] != "autocomplete-pytorch-coding" {
		t.Fatalf("slug = %#v", prompt["slug"])
	}
	if _, ok := body["related"]; !ok {
		t.Fatal("expected related field")
	}

	// Unknown slug -> 404.
	ctx = newGetCtx("/v1/prompts/does-not-exist", "does-not-exist")
	h.HandleGet(ctx)
	if ctx.Response.StatusCode() != 404 {
		t.Fatalf("status = %d, want 404", ctx.Response.StatusCode())
	}
}
