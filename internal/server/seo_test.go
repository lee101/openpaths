package server

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	fasthttprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/seo"
)

const testIndex = `<!doctype html><html><head><title>Home</title><meta name="description" content="home" /><meta property="og:url" content="https://openpaths.io/" /><meta property="og:title" content="Home" /><meta property="og:description" content="home" /><meta property="og:image" content="https://openpaths.io/og-image.png" /><meta name="twitter:title" content="Home" /><meta name="twitter:description" content="home" /><meta name="twitter:image" content="x" /><link rel="canonical" href="https://openpaths.io/" /></head><body><div id="root"><div id="static-shell"><h1>Home hero</h1><div></div></div></div>
    <script>if(location.pathname!=='/'){var s=document.getElementById('static-shell');s&&s.remove()}</script>
    <script type="module" src="/src/main.tsx"></script></body></html>`

func testCatalog() *seo.Catalog {
	return seo.New([]model.ModelInfo{
		{ID: "gemini-nano-banana-2.1", Type: "image", OwnedBy: "google", Pricing: &model.ModelPricing{PerImage: 0.0336}},
		{ID: "gemini-3.1-flash-lite-image", Type: "image", OwnedBy: "google", Pricing: &model.ModelPricing{PerImage: 0.0336}},
		{ID: "fal-ai/flux-2-pro/inpaint", Type: "image", OwnedBy: "fal", Pricing: &model.ModelPricing{PerImage: 0.05}},
		{ID: "old-model", Type: "language", OwnedBy: "openai", Deprecated: true},
	})
}

func render(t *testing.T, dir, path string) string {
	t.Helper()
	m := newModelSEO(dir, testCatalog())
	out, ok := m.render([]byte(testIndex), path)
	if !ok {
		t.Fatalf("%s: not rendered", path)
	}
	return string(out)
}

func TestModelPageRenderTags(t *testing.T) {
	doc := render(t, t.TempDir(), "/models/gemini-nano-banana-2.1")
	for _, want := range []string{
		"<title>Gemini Nano Banana 2.1 API, Pricing and Model ID | OpenPaths</title>",
		`<link rel="canonical" href="https://openpaths.io/models/gemini-nano-banana-2.1" />`,
		`<meta property="og:url" content="https://openpaths.io/models/gemini-nano-banana-2.1" />`,
		`<meta property="og:title" content="Gemini Nano Banana 2.1 API, Pricing and Model ID | OpenPaths" />`,
		`<meta name="twitter:title"`,
		`<meta name="twitter:image"`,
		`"@type":"SoftwareApplication"`, `"@type":"BreadcrumbList"`, `"@type":"FAQPage"`,
		`id="model-seo"`,
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q", want)
		}
	}
	if n := strings.Count(doc, "<h1"); n != 1 {
		t.Fatalf("h1 count = %d", n)
	}
	if strings.Contains(doc, "Home hero") || strings.Contains(doc, `id="static-shell"`) {
		t.Fatal("home static shell must be replaced")
	}
	if !strings.Contains(doc, `<script>if(location.pathname`) || !strings.Contains(doc, `src="/src/main.tsx"`) {
		t.Fatal("scripts after root lost")
	}
	if m := regexp.MustCompile(`<title>`).FindAllString(doc, -1); len(m) != 1 {
		t.Fatal("duplicate title")
	}
}

func TestModelPageRenderSlashIDAndUnknown(t *testing.T) {
	doc := render(t, t.TempDir(), "/models/fal-ai/flux-2-pro/inpaint")
	if !strings.Contains(doc, `href="https://openpaths.io/models/fal-ai%2Fflux-2-pro%2Finpaint"`) {
		t.Fatalf("canonical not percent-escaped:\n%s", doc)
	}
	m := newModelSEO(t.TempDir(), testCatalog())
	for _, p := range []string{"/models", "/models/", "/models/nope", "/models/old-model", "/models/../etc/passwd", "/pricing"} {
		if _, ok := m.render([]byte(testIndex), p); ok {
			t.Errorf("%s should not render", p)
		}
	}
}

func TestModelPageUsesPrerenderedBaseAndOgImage(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "models", "gemini-nano-banana-2.1"), 0o755)
	os.MkdirAll(filepath.Join(dir, "og", "models"), 0o755)
	os.WriteFile(filepath.Join(dir, "og", "models", "gemini-3.1-flash-lite-image.png"), []byte("x"), 0o644)
	curated := strings.Replace(testIndex, "<title>Home</title>", "<title>Curated Title</title>", 1)
	os.WriteFile(filepath.Join(dir, "models", "gemini-nano-banana-2.1", "index.html"), []byte(curated), 0o644)

	doc := render(t, dir, "/models/gemini-nano-banana-2.1")
	if !strings.Contains(doc, "<title>Curated Title</title>") || !strings.Contains(doc, `"@type":"FAQPage"`) {
		t.Fatal("prerendered meta must be kept and JSON-LD added")
	}
	if !strings.Contains(doc, `<link rel="canonical" href="https://openpaths.io/models/gemini-nano-banana-2.1" />`) {
		t.Fatal("canonical not set on prerendered base")
	}
	og := render(t, dir, "/models/gemini-3.1-flash-lite-image")
	if !strings.Contains(og, `content="https://openpaths.io/og/models/gemini-3.1-flash-lite-image.png"`) {
		t.Fatal("per-model og image not used")
	}
}

func TestSEORoutes(t *testing.T) {
	r := fasthttprouter.New()
	registerSEORoutes(r, testCatalog())
	cases := map[string]string{
		"/llms.txt":           "text/plain; charset=utf-8",
		"/llms-full.txt":      "text/plain; charset=utf-8",
		"/sitemap-models.xml": "text/xml; charset=utf-8",
	}
	for path, ct := range cases {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.Header.SetMethod(fasthttp.MethodGet)
		ctx.Request.SetRequestURI(path)
		r.Handler(ctx)
		if ctx.Response.StatusCode() != 200 || string(ctx.Response.Header.ContentType()) != ct {
			t.Errorf("%s: status=%d ct=%q", path, ctx.Response.StatusCode(), ctx.Response.Header.ContentType())
		}
		if !strings.Contains(string(ctx.Response.Body()), "gemini-nano-banana-2.1") {
			t.Errorf("%s missing model", path)
		}
		if strings.Contains(string(ctx.Response.Body()), "old-model") {
			t.Errorf("%s lists deprecated model", path)
		}
	}
}

func TestSPAHandlerServesModelPage(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte(testIndex), 0o644)
	var apiPaths []string
	api := func(ctx *fasthttp.RequestCtx) {
		apiPaths = append(apiPaths, string(ctx.Path()))
		ctx.SetStatusCode(200)
	}
	h := spaHandler(dir, api, nil, nil, nil, nil, testCatalog())

	get := func(path string) *fasthttp.RequestCtx {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.Header.SetMethod(fasthttp.MethodGet)
		ctx.Request.SetRequestURI(path)
		h(ctx)
		return ctx
	}
	ctx := get("/models/gemini-3.1-flash-lite-image")
	if ctx.Response.StatusCode() != 200 || !strings.Contains(string(ctx.Response.Body()), "<title>Gemini 3.1 Flash Lite Image API") {
		t.Fatalf("model page not injected: %d %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	get("/llms.txt")
	get("/llms-full.txt")
	get("/sitemap-models.xml")
	if len(apiPaths) != 3 {
		t.Fatalf("seo routes must reach the API router: %v", apiPaths)
	}
	ctx = get("/models/unknown-model")
	if ctx.Response.StatusCode() != 200 || strings.Contains(string(ctx.Response.Body()), "model-seo") {
		t.Fatal("unknown model must fall back to the SPA shell")
	}
}

func TestSPAHandlerPrefersPrerenderedPage(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "index.html"), []byte(testIndex), 0o644)
	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	os.WriteFile(filepath.Join(dir, "docs", "index.html"), []byte(strings.Replace(testIndex, "<title>Home</title>", "<title>Docs Page</title>", 1)), 0o644)
	h := spaHandler(dir, func(ctx *fasthttp.RequestCtx) {}, nil, nil, nil, nil, testCatalog())
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/docs")
	h(ctx)
	if !strings.Contains(string(ctx.Response.Body()), "<title>Docs Page</title>") {
		t.Fatalf("prerendered page not served: %s", ctx.Response.Body())
	}
}
