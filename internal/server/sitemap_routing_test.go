package server

import (
	"testing"

	fasthttprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

// The sitemap index (handler.ArtSitemapIndexXML) advertises paged art sitemaps at
// /sitemap-art-1.xml, /sitemap-art-2.xml, … Googlebot fetches those exact URLs, so the
// router MUST resolve them to the image-sitemap handler. The catch is that they share a
// prefix with the static /sitemap-art-tags.xml and /sitemap-art.xml routes; a careless
// pattern (or a router that can't do mid-segment params) makes /sitemap-art-1.xml fall
// through, nginx then serves the SPA, and a URL the index claims is XML returns
// text/html. This pins the exact route set registered in New() (see server.go).
func TestSitemapRoutesResolve(t *testing.T) {
	r := fasthttprouter.New()
	mark := func(name string) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			ctx.SetUserValue("route", name)
			ctx.SetStatusCode(fasthttp.StatusOK)
		}
	}
	// Mirror the registration order used in New().
	r.GET("/sitemap.xml", mark("index"))
	r.GET("/sitemap-pages.xml", mark("pages"))
	r.GET("/sitemap-art-tags.xml", mark("tags"))
	r.GET("/sitemap-art-{page}.xml", mark("images"))
	r.GET("/sitemap-art.xml", mark("images"))

	cases := []struct{ path, want string }{
		{"/sitemap.xml", "index"},
		{"/sitemap-pages.xml", "pages"},
		{"/sitemap-art-tags.xml", "tags"},
		{"/sitemap-art.xml", "images"},
		{"/sitemap-art-1.xml", "images"},
		{"/sitemap-art-2.xml", "images"},
		{"/sitemap-art-42.xml", "images"},
	}
	for _, tc := range cases {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.Header.SetMethod(fasthttp.MethodGet)
		ctx.Request.SetRequestURI(tc.path)
		r.Handler(ctx)
		if got := ctx.Response.StatusCode(); got != fasthttp.StatusOK {
			t.Errorf("%s: status %d, want 200 (route not matched -> would serve SPA html)", tc.path, got)
			continue
		}
		if got, _ := ctx.UserValue("route").(string); got != tc.want {
			t.Errorf("%s: routed to %q, want %q", tc.path, got, tc.want)
		}
	}
}

// Crawlers and sitemap validators frequently issue a HEAD before a GET. fasthttp/router
// does not auto-answer HEAD for GET-only routes, so without explicit HEAD handlers a probe
// gets 405 Method Not Allowed -- which Google Search Console surfaces as "sitemap could not
// be read". This pins that every sitemap route answers HEAD with 200.
func TestSitemapRoutesAnswerHEAD(t *testing.T) {
	r := fasthttprouter.New()
	head := func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("text/xml; charset=utf-8")
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
	r.HEAD("/sitemap.xml", head)
	r.HEAD("/sitemap-pages.xml", head)
	r.HEAD("/sitemap-art-tags.xml", head)
	r.HEAD("/sitemap-art-{page}.xml", head)
	r.HEAD("/sitemap-art.xml", head)

	for _, path := range []string{
		"/sitemap.xml", "/sitemap-pages.xml", "/sitemap-art-tags.xml",
		"/sitemap-art.xml", "/sitemap-art-1.xml", "/sitemap-art-11.xml",
	} {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.Header.SetMethod(fasthttp.MethodHead)
		ctx.Request.SetRequestURI(path)
		r.Handler(ctx)
		if got := ctx.Response.StatusCode(); got != fasthttp.StatusOK {
			t.Errorf("HEAD %s: status %d, want 200 (405 -> validators report 'could not be read')", path, got)
		}
		if ct := string(ctx.Response.Header.ContentType()); ct != "text/xml; charset=utf-8" {
			t.Errorf("HEAD %s: content-type %q, want text/xml; charset=utf-8", path, ct)
		}
	}
}
