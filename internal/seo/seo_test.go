package seo

import (
	"encoding/json"
	"encoding/xml"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/config"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
)

func realCatalog(t *testing.T) *Catalog {
	t.Helper()
	cfg, err := config.Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return New(router.New(provider.NewRegistry(), cfg.Models).ListModels())
}

func TestCatalogFromConfigIncludesGeminiImage(t *testing.T) {
	c := realCatalog(t)
	for _, id := range []string{"gemini-nano-banana-2.1", "gemini-3.1-flash-lite-image"} {
		e, ok := c.Lookup(id)
		if !ok {
			t.Fatalf("%s missing from catalog", id)
		}
		if e.Type != "image" || e.Endpoint != "/v1/images/generations" {
			t.Fatalf("%s type=%q endpoint=%q", id, e.Type, e.Endpoint)
		}
		if !strings.Contains(e.Price, "$0.0336 per image") {
			t.Fatalf("%s price %q", id, e.Price)
		}
	}
	if _, ok := c.Lookup("or/gemini-3.1-flash-image"); ok {
		t.Fatal("or/ routing alias must not get its own page")
	}
}

func TestCatalogSkipsDeprecated(t *testing.T) {
	c := realCatalog(t)
	cfg, _ := config.Load(filepath.Join("..", "..", "config.yaml"))
	skipped := 0
	for _, m := range cfg.Models {
		if m.Deprecated {
			skipped++
			if _, ok := c.Lookup(m.ID); ok {
				t.Fatalf("deprecated %s listed", m.ID)
			}
		}
	}
	t.Logf("catalog=%d deprecated=%d", c.Len(), skipped)
}

func TestSitemapWellFormedAndCoversEveryModel(t *testing.T) {
	c := realCatalog(t)
	var doc struct {
		URLs []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal([]byte(c.SitemapXML()), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.URLs) != c.Len() || c.Len() < 100 {
		t.Fatalf("urls=%d catalog=%d", len(doc.URLs), c.Len())
	}
	seen := map[string]bool{}
	for _, u := range doc.URLs {
		if !strings.HasPrefix(u.Loc, "https://openpaths.io/models/") || seen[u.Loc] {
			t.Fatalf("bad or duplicate loc %q", u.Loc)
		}
		seen[u.Loc] = true
	}
	if !seen["https://openpaths.io/models/gemini-nano-banana-2.1"] {
		t.Fatal("nano banana missing from sitemap")
	}
	if !seen["https://openpaths.io/models/openpaths%2Fauto"] && !seen["https://openpaths.io/models/openpaths%2Fauto-image"] {
		t.Fatal("slash ids must be percent-escaped")
	}
}

func TestLLMsTxt(t *testing.T) {
	c := realCatalog(t)
	txt := c.LLMsTxt()
	for _, want := range []string{"# OpenPaths", "\n> ", "## Image generation models", "(https://openpaths.io/models/gemini-nano-banana-2.1)", "`gemini-nano-banana-2.1`", "/docs/image-generation"} {
		if !strings.Contains(txt, want) {
			t.Errorf("llms.txt missing %q", want)
		}
	}
	if strings.Count(txt, "\n# ") != 0 || !strings.HasPrefix(txt, "# OpenPaths\n") {
		t.Error("llms.txt must have exactly one H1 at the top")
	}
	full := c.LLMsFullTxt()
	if !strings.Contains(full, "Endpoint: `POST https://openpaths.io/v1/images/generations`") || !strings.Contains(full, "$0.0336 per image") {
		t.Error("llms-full.txt missing endpoint or price for image models")
	}
}

func TestNewModelAutoIncluded(t *testing.T) {
	cfg, err := config.Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	extra := cfg.Models[0]
	extra.ID = "brand-new-image-model"
	extra.Deprecated = false
	extra.PricePerImage = 0.02
	c := New(router.New(provider.NewRegistry(), append(cfg.Models, extra)).ListModels())
	e, ok := c.Lookup("brand-new-image-model")
	if !ok {
		t.Fatal("new model not in catalog")
	}
	for name, body := range map[string]string{"sitemap": c.SitemapXML(), "llms": c.LLMsTxt(), "llms-full": c.LLMsFullTxt()} {
		if !strings.Contains(body, "brand-new-image-model") {
			t.Errorf("%s does not list new model", name)
		}
	}
	if e.URL() != "https://openpaths.io/models/brand-new-image-model" {
		t.Fatalf("url %s", e.URL())
	}
}

func TestModelPageMetaAndJSONLD(t *testing.T) {
	c := realCatalog(t)
	e, _ := c.Lookup("gemini-nano-banana-2.1")
	if len(e.Description()) > 165 || e.Description() == "" || !strings.Contains(e.Title(), "Gemini Nano Banana 2.1") {
		t.Fatalf("title=%q desc=%q", e.Title(), e.Description())
	}
	var graph []map[string]any
	if err := json.Unmarshal([]byte(e.JSONLD()), &graph); err != nil {
		t.Fatal(err)
	}
	types := map[string]bool{}
	for _, n := range graph {
		types[n["@type"].(string)] = true
	}
	for _, want := range []string{"SoftwareApplication", "BreadcrumbList", "FAQPage"} {
		if !types[want] {
			t.Errorf("missing %s", want)
		}
	}
	body := c.BodyHTML(e)
	if strings.Count(body, "<h1>") != 1 || !strings.Contains(body, `href="/models"`) || !strings.Contains(body, `href="/docs"`) {
		t.Fatalf("body h1/links wrong: %s", body)
	}
}

func TestJSONLDEscapesScriptBreakout(t *testing.T) {
	c := New(nil)
	e := &Entry{ID: "x</script><b>", Name: "x", Provider: "p", Type: "language", Endpoint: "/v1/chat/completions"}
	if strings.Contains(e.JSONLD(), "</script>") || strings.Contains(e.ClientJSON(), "</script>") {
		t.Fatal("script breakout not escaped")
	}
	if strings.Contains(c.BodyHTML(e), "<b>") {
		t.Fatal("body not escaped")
	}
}

func TestDisplayName(t *testing.T) {
	for id, want := range map[string]string{
		"gpt-image-2":            "GPT Image 2",
		"gemini-nano-banana-2.1": "Gemini Nano Banana 2.1",
		"openpaths/auto-image":   "Auto Image",
	} {
		if got := DisplayName(id); got != want {
			t.Errorf("DisplayName(%q)=%q want %q", id, got, want)
		}
	}
}

func TestTitlesAndDescriptionsUnique(t *testing.T) {
	c := realCatalog(t)
	titles, descs := map[string]string{}, map[string]string{}
	for _, e := range c.Entries() {
		if o, ok := titles[e.Title()]; ok {
			t.Errorf("duplicate title %q: %s and %s", e.Title(), o, e.ID)
		}
		if o, ok := descs[e.Description()]; ok {
			t.Errorf("duplicate description for %s and %s", o, e.ID)
		}
		titles[e.Title()], descs[e.Description()] = e.ID, e.ID
	}
}
