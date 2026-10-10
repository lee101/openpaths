package server

import (
	"bytes"
	"html"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	fasthttprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/seo"
)

func registerSEORoutes(r *fasthttprouter.Router, cat *seo.Catalog) {
	text := func(body func() string) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			ctx.SetContentType("text/plain; charset=utf-8")
			ctx.Response.Header.Set("Cache-Control", "public, max-age=3600")
			ctx.SetStatusCode(200)
			ctx.SetBodyString(body())
		}
	}
	r.GET("/llms.txt", text(cat.LLMsTxt))
	r.GET("/llms-full.txt", text(cat.LLMsFullTxt))
	r.GET("/sitemap-models.xml", func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("text/xml; charset=utf-8")
		ctx.Response.Header.Set("Cache-Control", "public, max-age=3600")
		ctx.SetStatusCode(200)
		ctx.SetBodyString(cat.SitemapXML())
	})
}

var modelOgSlugRe = regexp.MustCompile(`[^a-z0-9.]+`)

func modelOgSlug(id string) string {
	return strings.Trim(modelOgSlugRe.ReplaceAllString(strings.ToLower(id), "-"), "-")
}

type modelSEO struct {
	dir string
	cat *seo.Catalog
	og  map[string]bool
}

func newModelSEO(dir string, cat *seo.Catalog) *modelSEO {
	m := &modelSEO{dir: dir, cat: cat, og: map[string]bool{}}
	if files, err := os.ReadDir(filepath.Join(dir, "og", "models")); err == nil {
		for _, f := range files {
			m.og[f.Name()] = true
		}
	}
	return m
}

func (m *modelSEO) render(index []byte, path string) ([]byte, bool) {
	if m == nil || m.cat == nil || index == nil {
		return nil, false
	}
	rest := strings.TrimPrefix(path, "/models/")
	if rest == path {
		return nil, false
	}
	id, err := url.PathUnescape(strings.Trim(rest, "/"))
	if err != nil {
		return nil, false
	}
	e, ok := m.cat.Lookup(id)
	if !ok {
		return nil, false
	}
	base := index
	meta := pageMeta{Title: e.Title(), Description: e.Description(), URL: e.URL(), Image: e.Image()}
	if m.og[modelOgSlug(id)+".png"] {
		meta.Image = seo.BaseURL + "/og/models/" + modelOgSlug(id) + ".png"
	}
	if !strings.Contains(id, "..") {
		if data, err := os.ReadFile(filepath.Join(m.dir, "models", filepath.FromSlash(id), "index.html")); err == nil {
			base = data
			meta = pageMeta{}
		}
	}
	doc := injectPageMeta(base, meta)
	if meta.Title == "" {
		doc = replaceCanonical(doc, e.URL())
	}
	doc = bytes.Replace(doc, []byte("</head>"), []byte(`<script type="application/ld+json" id="seo-jsonld">`+e.JSONLD()+`</script><script type="application/json" id="model-seo">`+e.ClientJSON()+`</script></head>`), 1)
	return setRootBody(doc, m.cat.BodyHTML(e)), true
}

func setRootBody(doc []byte, body string) []byte {
	start := bytes.Index(doc, []byte(`<div id="root">`))
	end := bytes.Index(doc, []byte(`<script>if(location.pathname`))
	if start < 0 {
		return doc
	}
	if end < start {
		return bytes.Replace(doc, []byte(`<div id="root">`), []byte(`<div id="root">`+body), 1)
	}
	var out bytes.Buffer
	out.Grow(len(doc) + len(body))
	out.Write(doc[:start])
	out.WriteString(`<div id="root">` + body + "</div>\n    ")
	out.Write(doc[end:])
	return out.Bytes()
}

var canonicalRe = regexp.MustCompile(`(?is)<link\s+rel=["']canonical["']\s+href=["'][^"']*["']\s*/?>`)

func replaceCanonical(doc []byte, u string) []byte {
	return canonicalRe.ReplaceAll(doc, []byte(`<link rel="canonical" href="`+html.EscapeString(u)+`" />`))
}

var prerenderMu sync.RWMutex
var prerenderCache = map[string][]byte{}

func readPrerendered(dir, path string) []byte {
	if path == "/" || path == "" || strings.Contains(path, "..") || !isSPARoute(path) {
		return nil
	}
	key := dir + "\x00" + path
	prerenderMu.RLock()
	data, ok := prerenderCache[key]
	prerenderMu.RUnlock()
	if ok {
		return data
	}
	file := filepath.Join(dir, filepath.FromSlash(strings.Trim(path, "/")), "index.html")
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	prerenderMu.Lock()
	prerenderCache[key] = data
	prerenderMu.Unlock()
	return data
}
