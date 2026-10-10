package seo

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/openpaths/openpaths/internal/model"
)

const BaseURL = "https://openpaths.io"

type Entry struct {
	ID            string
	Name          string
	Provider      string
	Type          string
	Summary       string
	Price         string
	Endpoint      string
	Aliases       []string
	ContextWindow int
	Streaming     bool
	Tools         bool
	Vision        bool
	Sizes         []string
}

type Catalog struct {
	entries []*Entry
	byID    map[string]*Entry
	byProv  map[string][]*Entry
}

var typeEndpoint = map[string]string{
	"language":      "/v1/chat/completions",
	"image":         "/v1/images/generations",
	"video":         "/v1/videos/generations",
	"speech":        "/v1/audio/speech",
	"transcription": "/v1/audio/transcriptions",
	"embedding":     "/v1/embeddings",
	"music":         "/v1/music/generations",
	"3d":            "/v1/3d/generations",
	"forecasting":   "/v1/forecasts",
}

var typeLabel = map[string]string{
	"language":      "language model",
	"image":         "image generation model",
	"video":         "video generation model",
	"speech":        "speech model",
	"transcription": "speech-to-text model",
	"embedding":     "embedding model",
	"music":         "music generation model",
	"3d":            "3D generation model",
	"forecasting":   "forecasting model",
}

var sectionOrder = []string{"language", "image", "video", "speech", "music", "transcription", "embedding", "3d", "forecasting"}

var sectionTitle = map[string]string{
	"language":      "Language models",
	"image":         "Image generation models",
	"video":         "Video generation models",
	"speech":        "Speech models",
	"music":         "Music models",
	"transcription": "Transcription models",
	"embedding":     "Embedding models",
	"3d":            "3D models",
	"forecasting":   "Forecasting models",
}

func New(models []model.ModelInfo) *Catalog {
	c := &Catalog{byID: map[string]*Entry{}, byProv: map[string][]*Entry{}}
	for i := range models {
		m := &models[i]
		if m.Deprecated || m.ID == "" || strings.HasPrefix(m.ID, "or/") {
			continue
		}
		e := &Entry{
			ID:            m.ID,
			Name:          DisplayName(m.ID),
			Provider:      ProviderName(m.OwnedBy),
			Type:          m.Type,
			Price:         priceSummary(m.Pricing),
			Endpoint:      typeEndpoint[m.Type],
			Aliases:       m.Aliases,
			ContextWindow: m.ContextWindow,
			Sizes:         m.SupportedSizes,
		}
		if e.Endpoint == "" {
			e.Endpoint = typeEndpoint["language"]
		}
		if m.Capabilities != nil {
			e.Streaming, e.Tools, e.Vision = m.Capabilities.Streaming, m.Capabilities.Tools, m.Capabilities.Vision
		}
		e.Summary = summary(e)
		c.entries = append(c.entries, e)
	}
	sort.Slice(c.entries, func(i, j int) bool { return c.entries[i].ID < c.entries[j].ID })
	for _, e := range c.entries {
		c.byID[e.ID] = e
		c.byProv[e.Provider] = append(c.byProv[e.Provider], e)
	}
	return c
}

func (c *Catalog) Len() int { return len(c.entries) }

func (c *Catalog) Entries() []*Entry { return c.entries }

func (c *Catalog) Lookup(id string) (*Entry, bool) {
	e, ok := c.byID[id]
	return e, ok
}

func (e *Entry) Path() string { return "/models/" + url.PathEscape(e.ID) }

func (e *Entry) URL() string { return BaseURL + e.Path() }

func (e *Entry) Title() string {
	return fmt.Sprintf("%s API, Pricing and Model ID | OpenPaths", e.Name)
}

func (e *Entry) Description() string {
	return Clip(fmt.Sprintf("%s by %s: %s. Call it as %s through the OpenAI-compatible OpenPaths API.", e.Name, e.Provider, e.Summary, e.ID), 160)
}

func (e *Entry) Image() string { return BaseURL + "/og-image.png" }

func summary(e *Entry) string {
	label := typeLabel[e.Type]
	if label == "" {
		label = "language model"
	}
	parts := []string{"A " + label}
	if e.ContextWindow > 0 {
		parts[0] += " with a " + formatTokens(e.ContextWindow) + " token context window"
	}
	if e.Price != "" {
		parts = append(parts, e.Price)
	}
	return strings.Join(parts, ", ")
}

func formatTokens(n int) string {
	switch {
	case n >= 1_000_000 && n%1_000_000 == 0:
		return strconv.Itoa(n/1_000_000) + "M"
	case n >= 1_000_000:
		return strconv.FormatFloat(float64(n)/1e6, 'f', 2, 64) + "M"
	case n >= 1000:
		return strconv.Itoa(n/1000) + "K"
	}
	return strconv.Itoa(n)
}

func money(v float64) string {
	s := strconv.FormatFloat(v, 'f', 4, 64)
	s = strings.TrimRight(s, "0")
	if strings.HasSuffix(s, ".") {
		s += "00"
	} else if i := strings.IndexByte(s, '.'); len(s)-i-1 < 2 {
		s += "0"
	}
	return "$" + s
}

func priceSummary(p *model.ModelPricing) string {
	if p == nil {
		return ""
	}
	var out []string
	if p.InputPer1M > 0 || p.OutputPer1M > 0 {
		out = append(out, fmt.Sprintf("%s per 1M input and %s per 1M output tokens", money(p.InputPer1M), money(p.OutputPer1M)))
	}
	if len(p.PerImageByResolution) > 0 {
		keys := make([]string, 0, len(p.PerImageByResolution))
		for k := range p.PerImageByResolution {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var parts []string
		for _, k := range keys {
			parts = append(parts, k+" "+money(p.PerImageByResolution[k]))
		}
		out = append(out, "per image: "+strings.Join(parts, ", "))
	} else if p.PerImage > 0 {
		out = append(out, money(p.PerImage)+" per image")
	}
	if p.PerMegapixel > 0 {
		out = append(out, money(p.PerMegapixel)+" per megapixel")
	}
	if len(p.PerSecondByResolution) > 0 {
		keys := make([]string, 0, len(p.PerSecondByResolution))
		for k := range p.PerSecondByResolution {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var parts []string
		for _, k := range keys {
			parts = append(parts, k+" "+money(p.PerSecondByResolution[k]))
		}
		out = append(out, "per second: "+strings.Join(parts, ", "))
	} else if p.PerSecond > 0 {
		out = append(out, money(p.PerSecond)+" per second")
	}
	if p.PerVideo > 0 {
		out = append(out, money(p.PerVideo)+" per video")
	}
	if p.PerRequest > 0 {
		out = append(out, money(p.PerRequest)+" per request")
	}
	if p.PerMinute > 0 {
		out = append(out, money(p.PerMinute)+" per minute")
	}
	if p.PerHour > 0 {
		out = append(out, money(p.PerHour)+" per hour")
	}
	if p.Per1MCharacters > 0 {
		out = append(out, money(p.Per1MCharacters)+" per 1M characters")
	}
	return strings.Join(out, "; ")
}

var acronyms = map[string]string{
	"gpt": "GPT", "ai": "AI", "tts": "TTS", "stt": "STT", "llm": "LLM", "ra1": "RA1", "ra2": "RA2",
	"glm": "GLM", "xai": "xAI", "3d": "3D", "dall": "DALL", "e3": "E3", "qwq": "QwQ", "ltx": "LTX",
	"gemini": "Gemini", "claude": "Claude", "openai": "OpenAI", "grok": "Grok", "bfl": "BFL",
}

func DisplayName(id string) string {
	id = strings.TrimPrefix(id, "openpaths/")
	words := strings.FieldsFunc(id, func(r rune) bool { return r == '-' || r == '_' || r == '/' })
	for i, w := range words {
		lw := strings.ToLower(w)
		if a, ok := acronyms[lw]; ok {
			words[i] = a
			continue
		}
		if w == "" {
			continue
		}
		r := []rune(w)
		if unicode.IsLetter(r[0]) {
			r[0] = unicode.ToUpper(r[0])
		}
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}

var providerNames = map[string]string{
	"openai": "OpenAI", "anthropic": "Anthropic", "google": "Google", "xai": "xAI", "bfl": "Black Forest Labs",
	"deepseek": "DeepSeek", "mistral": "Mistral AI", "groq": "Groq", "minimax": "MiniMax", "together": "Together AI",
	"zai": "Z.ai", "fal": "Fal", "meta": "Meta", "openrouter": "OpenRouter", "openpaths": "OpenPaths",
	"netwrck": "Netwrck", "manifoldgen": "ManifoldGen", "cutedsl": "CuteDSL",
}

func ProviderName(p string) string {
	if n, ok := providerNames[strings.ToLower(p)]; ok {
		return n
	}
	if p == "" {
		return "OpenPaths"
	}
	return DisplayName(p)
}

func Clip(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	cut := string(r[:max-1])
	if i := strings.LastIndexByte(cut, ' '); i > max-30 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,;:.") + "…"
}

var xmlReplacer = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")

func xmlEscape(s string) string { return xmlReplacer.Replace(s) }

func (c *Catalog) SitemapXML() string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, e := range c.entries {
		fmt.Fprintf(&b, "  <url>\n    <loc>%s</loc>\n    <changefreq>weekly</changefreq>\n    <priority>0.6</priority>\n  </url>\n", xmlEscape(e.URL()))
	}
	b.WriteString("</urlset>\n")
	return b.String()
}

func (c *Catalog) grouped() map[string][]*Entry {
	g := map[string][]*Entry{}
	for _, e := range c.entries {
		t := e.Type
		if _, ok := sectionTitle[t]; !ok {
			t = "language"
		}
		g[t] = append(g[t], e)
	}
	return g
}

func (c *Catalog) LLMsTxt() string {
	var b strings.Builder
	b.WriteString("# OpenPaths\n\n")
	b.WriteString("> OpenPaths is an OpenAI- and Anthropic-compatible AI gateway: one API key and base URL (" + BaseURL + "/v1) for language, image, video, speech, music, embedding and 3D models from many providers, with smart routing and near-zero markup.\n\n")
	b.WriteString("## Docs\n\n")
	for _, l := range [][2]string{
		{"/docs", "API docs: chat completions, images, video, speech, embeddings"},
		{"/docs/image-generation", "Image generation API incl. Gemini image models (Nano Banana), GPT Image, Flux"},
		{"/models", "Model directory with pricing"},
		{"/pricing", "Pricing"},
		{"/providers", "Providers"},
		{"/tools", "Hands-on tools for every generation endpoint"},
		{"/text-to-image", "Text to image tool"},
		{"/llms-full.txt", "Full model catalog with pricing and model IDs"},
	} {
		fmt.Fprintf(&b, "- [%s](%s%s): %s\n", linkLabel(l[0]), BaseURL, l[0], l[1])
	}
	g := c.grouped()
	for _, t := range sectionOrder {
		list := g[t]
		if len(list) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s\n\n", sectionTitle[t])
		for _, e := range list {
			fmt.Fprintf(&b, "- [%s](%s): %s, model ID `%s`\n", e.Name, e.URL(), e.Provider, e.ID)
		}
	}
	return b.String()
}

func linkLabel(path string) string {
	if path == "/llms-full.txt" {
		return "Full catalog"
	}
	return DisplayName(strings.Trim(path, "/"))
}

func (c *Catalog) LLMsFullTxt() string {
	var b strings.Builder
	b.WriteString("# OpenPaths model catalog\n\n")
	fmt.Fprintf(&b, "> %d models. Base URL %s/v1, OpenAI-compatible. Use the model ID in the `model` field.\n", len(c.entries), BaseURL)
	g := c.grouped()
	for _, t := range sectionOrder {
		list := g[t]
		if len(list) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n## %s\n", sectionTitle[t])
		for _, e := range list {
			fmt.Fprintf(&b, "\n### %s\n\n", e.Name)
			fmt.Fprintf(&b, "- URL: %s\n- Model ID: `%s`\n- Provider: %s\n- Endpoint: `POST %s/v1%s`\n", e.URL(), e.ID, e.Provider, BaseURL, strings.TrimPrefix(e.Endpoint, "/v1"))
			if e.Price != "" {
				fmt.Fprintf(&b, "- Pricing: %s\n", e.Price)
			}
			if e.ContextWindow > 0 {
				fmt.Fprintf(&b, "- Context window: %d tokens\n", e.ContextWindow)
			}
			if len(e.Aliases) > 0 {
				fmt.Fprintf(&b, "- Aliases: %s\n", strings.Join(e.Aliases, ", "))
			}
			if len(e.Sizes) > 0 {
				fmt.Fprintf(&b, "- Sizes: %s\n", strings.Join(e.Sizes, ", "))
			}
		}
	}
	return b.String()
}

func (e *Entry) faq() [][2]string {
	qs := [][2]string{
		{fmt.Sprintf("What is the model ID for %s on OpenPaths?", e.Name), fmt.Sprintf("Use `%s` as the model field when calling POST %s%s.", e.ID, BaseURL, e.Endpoint)},
	}
	if e.Price != "" {
		qs = append(qs, [2]string{fmt.Sprintf("How much does %s cost?", e.Name), e.Price + ". OpenPaths bills the provider price with near-zero markup."})
	}
	qs = append(qs, [2]string{fmt.Sprintf("Is %s compatible with the OpenAI SDK?", e.Name), "Yes. Point the OpenAI SDK base URL at " + BaseURL + "/v1 and use your OpenPaths API key."})
	return qs
}

func (e *Entry) JSONLD() string {
	faq := make([]map[string]any, 0, 3)
	for _, q := range e.faq() {
		faq = append(faq, map[string]any{
			"@type": "Question", "name": q[0],
			"acceptedAnswer": map[string]any{"@type": "Answer", "text": strings.ReplaceAll(q[1], "`", "")},
		})
	}
	app := map[string]any{
		"@context":            "https://schema.org",
		"@type":               "SoftwareApplication",
		"name":                e.Name,
		"description":         e.Description(),
		"url":                 e.URL(),
		"applicationCategory": "DeveloperApplication",
		"operatingSystem":     "Web",
		"identifier":          e.ID,
		"publisher":           map[string]any{"@type": "Organization", "name": e.Provider},
	}
	graph := []any{
		app,
		map[string]any{
			"@context": "https://schema.org", "@type": "BreadcrumbList",
			"itemListElement": []any{
				map[string]any{"@type": "ListItem", "position": 1, "name": "OpenPaths", "item": BaseURL + "/"},
				map[string]any{"@type": "ListItem", "position": 2, "name": "Models", "item": BaseURL + "/models"},
				map[string]any{"@type": "ListItem", "position": 3, "name": e.Name, "item": e.URL()},
			},
		},
		map[string]any{"@context": "https://schema.org", "@type": "FAQPage", "mainEntity": faq},
	}
	out, _ := json.Marshal(graph)
	return strings.ReplaceAll(string(out), "</", `<\/`)
}

func (c *Catalog) related(e *Entry, n int) []*Entry {
	var out []*Entry
	for _, o := range c.byProv[e.Provider] {
		if o.ID != e.ID && o.Type == e.Type {
			out = append(out, o)
			if len(out) == n {
				break
			}
		}
	}
	return out
}

func (c *Catalog) BodyHTML(e *Entry) string {
	var b strings.Builder
	esc := html.EscapeString
	b.WriteString(`<main id="seo-static" data-model-id="` + esc(e.ID) + `">`)
	b.WriteString(`<nav aria-label="Breadcrumb"><a href="/">OpenPaths</a> / <a href="/models">Models</a> / <span>` + esc(e.Name) + `</span></nav>`)
	b.WriteString(`<h1>` + esc(e.Name) + ` API</h1>`)
	b.WriteString(`<p>` + esc(e.Description()) + `</p>`)
	b.WriteString(`<dl><dt>Model ID</dt><dd><code>` + esc(e.ID) + `</code></dd><dt>Provider</dt><dd>` + esc(e.Provider) + `</dd>`)
	if e.Price != "" {
		b.WriteString(`<dt>Pricing</dt><dd>` + esc(e.Price) + `</dd>`)
	}
	if e.ContextWindow > 0 {
		b.WriteString(`<dt>Context window</dt><dd>` + strconv.Itoa(e.ContextWindow) + ` tokens</dd>`)
	}
	b.WriteString(`<dt>Endpoint</dt><dd><code>POST ` + esc(BaseURL+e.Endpoint) + `</code></dd></dl>`)
	b.WriteString(`<h2>Frequently asked questions</h2>`)
	for _, q := range e.faq() {
		b.WriteString(`<h3>` + esc(q[0]) + `</h3><p>` + esc(strings.ReplaceAll(q[1], "`", "")) + `</p>`)
	}
	if rel := c.related(e, 6); len(rel) > 0 {
		b.WriteString(`<h2>More ` + esc(e.Provider) + ` models</h2><ul>`)
		for _, o := range rel {
			b.WriteString(`<li><a href="` + esc(o.Path()) + `">` + esc(o.Name) + `</a></li>`)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`<p><a href="/docs">API docs</a> · <a href="/pricing">Pricing</a> · <a href="/models">All models</a></p>`)
	b.WriteString(`</main>`)
	return b.String()
}

func (e *Entry) ClientJSON() string {
	out, _ := json.Marshal(map[string]any{
		"id": e.ID, "name": e.Name, "provider": e.Provider, "type": e.Type, "price": e.Price,
		"endpoint": e.Endpoint, "summary": e.Description(), "contextWindow": e.ContextWindow,
		"aliases": e.Aliases, "sizes": e.Sizes,
	})
	return strings.ReplaceAll(string(out), "</", `<\/`)
}
