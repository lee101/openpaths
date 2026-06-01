package handler

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/artindex"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

// ArtHandler serves the image prompt search engine: semantic search (gobed),
// trigram exact/substring search + browse (Postgres), tag facets, and single-item lookups.
type ArtHandler struct {
	index *artindex.Service
	db    *queries.ArtImageQueries
}

func NewArtHandler(index *artindex.Service, db *queries.ArtImageQueries) *ArtHandler {
	return &ArtHandler{index: index, db: db}
}

func (h *ArtHandler) HandleStatus(ctx *fasthttp.RequestCtx) {
	if h.index == nil {
		WriteJSONPublic(ctx, fasthttp.StatusOK, artindex.Status{})
		return
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, h.index.Status())
}

func artFilters(ctx *fasthttp.RequestCtx) model.ArtImageFilters {
	f := model.ArtImageFilters{
		Aspect: normalizeAspect(string(ctx.QueryArgs().Peek("aspect"))),
		Tag:    strings.ToLower(strings.TrimSpace(string(ctx.QueryArgs().Peek("tag")))),
		Source: strings.TrimSpace(string(ctx.QueryArgs().Peek("source"))),
	}
	f.MinW, _ = strconv.Atoi(string(ctx.QueryArgs().Peek("min_w")))
	f.MinH, _ = strconv.Atoi(string(ctx.QueryArgs().Peek("min_h")))
	return f
}

func normalizeAspect(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "portrait", "tall":
		return "portrait"
	case "wide", "landscape":
		return "wide"
	case "square":
		return "square"
	default:
		return ""
	}
}

func artLimitOffset(ctx *fasthttp.RequestCtx, def, max int) (int, int) {
	limit, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("limit")))
	if limit <= 0 || limit > max {
		limit = def
	}
	offset, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("offset")))
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// HandleSearch combines the gobed semantic index with the Postgres trigram index.
// Empty query browses newest-first. Filters (aspect/tag/source/min_w/min_h) apply to both.
func (h *ArtHandler) HandleSearch(ctx *fasthttp.RequestCtx) {
	query := strings.TrimSpace(string(ctx.QueryArgs().Peek("q")))
	filters := artFilters(ctx)
	limit, offset := artLimitOffset(ctx, 48, 100)

	// Browse (no query): straight from DB, newest first.
	if query == "" {
		h.browse(ctx, filters, limit, offset)
		return
	}

	var results []artindex.Result
	mode := "trigram"

	if h.index != nil {
		if semantic, err := h.index.Search(ctx, query, limit*4); err == nil && len(semantic) > 0 {
			for _, r := range semantic {
				if matchesFilters(r.Item, filters) {
					results = append(results, r)
				}
			}
			mode = "semantic"
		}
	}

	// Fall back to / supplement with trigram when semantic is unavailable or too sparse.
	if len(results) < limit && h.db != nil {
		seen := map[string]bool{}
		for _, r := range results {
			seen[r.ID] = true
		}
		if rows, err := h.db.SearchTrigram(ctx, query, filters, limit, offset); err == nil {
			for _, row := range rows {
				if seen[row.ID] {
					continue
				}
				results = append(results, artindex.Result{Item: row})
			}
			if len(results) == 0 {
				mode = "trigram"
			} else if mode != "semantic" {
				mode = "trigram"
			}
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}
	if results == nil {
		results = []artindex.Result{}
	}

	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{
		"query":   query,
		"count":   len(results),
		"mode":    mode,
		"results": results,
		"status":  h.statusOrEmpty(),
	})
}

func (h *ArtHandler) browse(ctx *fasthttp.RequestCtx, filters model.ArtImageFilters, limit, offset int) {
	if h.db == nil {
		WriteJSONPublic(ctx, fasthttp.StatusServiceUnavailable, map[string]any{"error": "art database not configured"})
		return
	}
	rows, err := h.db.List(ctx, filters, limit, offset)
	if err != nil {
		WriteJSONPublic(ctx, fasthttp.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	results := make([]artindex.Result, 0, len(rows))
	for _, row := range rows {
		results = append(results, artindex.Result{Item: row})
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{
		"query":   "",
		"count":   len(results),
		"mode":    "browse",
		"results": results,
		"status":  h.statusOrEmpty(),
	})
}

// HandleList is the browse endpoint with totals + aspect facets for the gallery UI.
func (h *ArtHandler) HandleList(ctx *fasthttp.RequestCtx) {
	if h.db == nil {
		WriteJSONPublic(ctx, fasthttp.StatusServiceUnavailable, map[string]any{"error": "art database not configured"})
		return
	}
	filters := artFilters(ctx)
	limit, offset := artLimitOffset(ctx, 48, 100)
	rows, err := h.db.List(ctx, filters, limit, offset)
	if err != nil {
		WriteJSONPublic(ctx, fasthttp.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	total, _ := h.db.Count(ctx, filters)
	aspects, _ := h.db.AspectCounts(ctx)
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{
		"count":   len(rows),
		"total":   total,
		"offset":  offset,
		"limit":   limit,
		"aspects": aspects,
		"results": rows,
	})
}

// HandleItem returns a single image by slug plus related images (semantic neighbors).
func (h *ArtHandler) HandleItem(ctx *fasthttp.RequestCtx) {
	if h.db == nil {
		WriteJSONPublic(ctx, fasthttp.StatusServiceUnavailable, map[string]any{"error": "art database not configured"})
		return
	}
	slug := strings.TrimSpace(string(ctx.QueryArgs().Peek("slug")))
	if slug == "" {
		WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": "slug parameter required"})
		return
	}
	item, err := h.db.GetBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteJSONPublic(ctx, fasthttp.StatusNotFound, map[string]any{"error": "not found"})
			return
		}
		WriteJSONPublic(ctx, fasthttp.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	related := h.related(ctx, item)
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{
		"item":    item,
		"related": related,
	})
}

func (h *ArtHandler) related(ctx *fasthttp.RequestCtx, item *model.ArtImage) []model.ArtImage {
	out := make([]model.ArtImage, 0, 12)
	seen := map[string]bool{item.ID: true}
	// Prefer semantic neighbors.
	if h.index != nil {
		if res, err := h.index.Search(ctx, item.Prompt, 16); err == nil {
			for _, r := range res {
				if seen[r.ID] {
					continue
				}
				seen[r.ID] = true
				out = append(out, r.Item)
				if len(out) >= 12 {
					return out
				}
			}
		}
	}
	// Supplement with same-tag images.
	if len(out) < 12 && len(item.Tags) > 0 {
		rows, err := h.db.List(ctx, model.ArtImageFilters{Tag: item.Tags[0]}, 16, 0)
		if err == nil {
			for _, r := range rows {
				if seen[r.ID] {
					continue
				}
				seen[r.ID] = true
				out = append(out, r)
				if len(out) >= 12 {
					break
				}
			}
		}
	}
	return out
}

// HandleTags returns top-tag facets for the subtag UI and tag pages.
func (h *ArtHandler) HandleTags(ctx *fasthttp.RequestCtx) {
	limit, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("limit")))
	if limit <= 0 || limit > 300 {
		limit = 60
	}
	var tags []queries.TagCount
	if h.index != nil {
		tags = h.index.Tags()
	}
	if len(tags) == 0 && h.db != nil {
		tags, _ = h.db.TopTags(ctx, limit)
	}
	if len(tags) > limit {
		tags = tags[:limit]
	}
	if tags == nil {
		tags = []queries.TagCount{}
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{
		"count": len(tags),
		"tags":  tags,
	})
}

const (
	artSiteBase      = "https://openpaths.io"
	artSitemapPageSz = 45000
)

// ArtSitemapPages returns how many image sitemap pages exist (for the sitemap index).
func (h *ArtHandler) ArtSitemapPages(ctx *fasthttp.RequestCtx) int {
	if h.db == nil {
		return 0
	}
	total, err := h.db.Count(ctx, model.ArtImageFilters{})
	if err != nil || total <= 0 {
		return 0
	}
	pages := total / artSitemapPageSz
	if total%artSitemapPageSz != 0 {
		pages++
	}
	return pages
}

// ArtSitemapIndexXML builds the sitemap index. Art image sitemaps use a clean
// path scheme (/sitemap-art-1.xml, /sitemap-art-2.xml, …) rather than a query
// string (?p=1) so crawlers like Googlebot treat each as a distinct, cacheable
// sitemap with no ambiguity.
func ArtSitemapIndexXML(base string, artPages int) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	fmt.Fprintf(&b, "  <sitemap>\n    <loc>%s/sitemap-pages.xml</loc>\n  </sitemap>\n", base)
	fmt.Fprintf(&b, "  <sitemap>\n    <loc>%s/sitemap-art-tags.xml</loc>\n  </sitemap>\n", base)
	for p := 1; p <= artPages; p++ {
		fmt.Fprintf(&b, "  <sitemap>\n    <loc>%s/sitemap-art-%d.xml</loc>\n  </sitemap>\n", base, p)
	}
	b.WriteString("</sitemapindex>\n")
	return b.String()
}

// HandleSitemapTags emits a urlset of the art landing page + every tag page.
func (h *ArtHandler) HandleSitemapTags(ctx *fasthttp.RequestCtx) {
	var tags []queries.TagCount
	if h.index != nil {
		tags = h.index.Tags()
	}
	if len(tags) == 0 && h.db != nil {
		tags, _ = h.db.TopTags(ctx, 300)
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	writeSitemapURL(&b, artSiteBase+"/art", "daily", "0.7")
	for _, t := range tags {
		writeSitemapURL(&b, artSiteBase+"/art/tag/"+url.PathEscape(t.Tag), "weekly", "0.6")
	}
	b.WriteString("</urlset>\n")
	ctx.SetContentType("text/xml; charset=utf-8")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(b.String())
}

// HandleSitemapImages emits one page of image detail URLs with the image sitemap extension.
// Page is selected via ?p=N (1-based).
func (h *ArtHandler) HandleSitemapImages(ctx *fasthttp.RequestCtx) {
	if h.db == nil {
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		return
	}
	// Page comes from the clean path scheme (/sitemap-art-{page}.xml) with a
	// fallback to the legacy ?p=N query string so already-indexed URLs keep working.
	page := 0
	if s, ok := ctx.UserValue("page").(string); ok {
		page, _ = strconv.Atoi(s)
	}
	if page < 1 {
		page, _ = strconv.Atoi(string(ctx.QueryArgs().Peek("p")))
	}
	if page < 1 {
		page = 1
	}
	rows, err := h.db.ListForSitemap(ctx, artSitemapPageSz, (page-1)*artSitemapPageSz)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:image="http://www.google.com/schemas/sitemap-image/1.1">` + "\n")
	for _, r := range rows {
		loc := artSiteBase + "/art/i/" + url.PathEscape(r.Slug)
		b.WriteString("  <url>\n    <loc>")
		b.WriteString(xmlEscape(loc))
		b.WriteString("</loc>\n    <changefreq>monthly</changefreq>\n")
		img := r.ImageURL
		if r.ThumbURL != "" {
			img = r.ThumbURL
		}
		if img != "" {
			b.WriteString("    <image:image>\n      <image:loc>")
			b.WriteString(xmlEscape(img))
			b.WriteString("</image:loc>\n      <image:caption>")
			b.WriteString(xmlEscape(artPromptCaption(r.Prompt, 200)))
			b.WriteString("</image:caption>\n    </image:image>\n")
		}
		b.WriteString("  </url>\n")
	}
	b.WriteString("</urlset>\n")
	ctx.SetContentType("text/xml; charset=utf-8")
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString(b.String())
}

func writeSitemapURL(b *strings.Builder, loc, freq, priority string) {
	b.WriteString("  <url>\n    <loc>")
	b.WriteString(xmlEscape(loc))
	b.WriteString("</loc>\n    <changefreq>")
	b.WriteString(freq)
	b.WriteString("</changefreq>\n    <priority>")
	b.WriteString(priority)
	b.WriteString("</priority>\n  </url>\n")
}

func xmlEscape(s string) string {
	s = stripInvalidXMLChars(s)
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&apos;")
	return r.Replace(s)
}

// isValidXMLChar reports whether r is allowed in a well-formed XML 1.0 document.
// XML 1.0 permits #x9, #xA, #xD, #x20-#xD7FF, #xE000-#xFFFD, #x10000-#x10FFFF.
// AI-generated prompts occasionally contain stray control characters (e.g. \x08),
// which make the sitemap not well-formed even after entity-escaping and cause
// crawlers like Googlebot to reject it.
func isValidXMLChar(r rune) bool {
	switch {
	case r == 0x09, r == 0x0A, r == 0x0D:
		return true
	case r >= 0x20 && r <= 0xD7FF:
		return true
	case r >= 0xE000 && r <= 0xFFFD:
		return true
	case r >= 0x10000 && r <= 0x10FFFF:
		return true
	default:
		return false
	}
}

// stripInvalidXMLChars drops any code point XML 1.0 forbids. It returns the
// input unchanged (no allocation) when everything is already valid.
func stripInvalidXMLChars(s string) string {
	if strings.IndexFunc(s, func(r rune) bool { return !isValidXMLChar(r) }) < 0 {
		return s
	}
	return strings.Map(func(r rune) rune {
		if isValidXMLChar(r) {
			return r
		}
		return -1
	}, s)
}

func artPromptCaption(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func (h *ArtHandler) statusOrEmpty() artindex.Status {
	if h.index == nil {
		return artindex.Status{}
	}
	return h.index.Status()
}

func matchesFilters(item model.ArtImage, f model.ArtImageFilters) bool {
	if f.Aspect != "" && item.Aspect != f.Aspect {
		return false
	}
	if f.Source != "" && item.Source != f.Source {
		return false
	}
	if f.MinW > 0 && item.Width < f.MinW {
		return false
	}
	if f.MinH > 0 && item.Height < f.MinH {
		return false
	}
	if f.Tag != "" {
		found := false
		for _, t := range item.Tags {
			if strings.EqualFold(t, f.Tag) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
