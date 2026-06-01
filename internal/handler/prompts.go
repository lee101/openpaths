package handler

import (
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/promptlib"
)

// PromptsHandler serves the read-only prompt library (/prompts directory).
// Semantic search and related-prompt lookups are powered by the gobed-backed
// promptlib.Index; it gracefully degrades to lexical/metadata helpers when the
// index is unavailable.
type PromptsHandler struct {
	index *promptlib.Index
}

func NewPromptsHandler(index *promptlib.Index) *PromptsHandler {
	return &PromptsHandler{index: index}
}

func parseFilters(ctx *fasthttp.RequestCtx) promptlib.Filters {
	q := ctx.QueryArgs()
	return promptlib.Filters{
		Category: strings.TrimSpace(string(q.Peek("category"))),
		Model:    strings.TrimSpace(string(q.Peek("model"))),
		Modality: strings.TrimSpace(string(q.Peek("type"))),
		Free:     string(q.Peek("free")) == "1" || string(q.Peek("free")) == "true",
		Query:    strings.TrimSpace(string(q.Peek("q"))),
	}
}

func parseLimit(ctx *fasthttp.RequestCtx, def, max int) int {
	n, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("limit")))
	if n <= 0 || n > max {
		return def
	}
	return n
}

// HandleList serves GET /v1/prompts with optional filters and search.
// Query params: q, category, model, type, free, limit, offset.
func (h *PromptsHandler) HandleList(ctx *fasthttp.RequestCtx) {
	f := parseFilters(ctx)
	limit := parseLimit(ctx, 48, 200)
	offset, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("offset")))
	if offset < 0 {
		offset = 0
	}

	var results []promptlib.Result
	semantic := false

	// Semantic search when a query is present and the index is ready.
	if f.Query != "" && h.index != nil && h.index.Ready() {
		if r, err := h.index.Search(ctx, f.Query, f, limit+offset); err == nil {
			results = r
			semantic = true
		}
	}
	if !semantic {
		for _, p := range promptlib.Filter(f) {
			results = append(results, promptlib.Result{Prompt: p})
		}
	}

	total := len(results)
	if offset > len(results) {
		offset = len(results)
	}
	results = results[offset:]
	if len(results) > limit {
		results = results[:limit]
	}

	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{
		"object":   "list",
		"query":    f.Query,
		"semantic": semantic,
		"count":    len(results),
		"total":    total,
		"offset":   offset,
		"results":  results,
	})
}

// HandleMeta serves GET /v1/prompts/meta — taxonomy + counts for filter UIs.
func (h *PromptsHandler) HandleMeta(ctx *fasthttp.RequestCtx) {
	byCategory, byModel, byModality, free := promptlib.Counts()
	var indexStatus promptlib.Status
	if h.index != nil {
		indexStatus = h.index.Status()
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{
		"total":      promptlib.Count(),
		"categories": promptlib.Categories(),
		"models":     promptlib.Models(),
		"types":      promptlib.Modalities(),
		"counts": map[string]any{
			"byCategory": byCategory,
			"byModel":    byModel,
			"byType":     byModality,
			"free":       free,
		},
		"index": indexStatus,
	})
}

// HandleGet serves GET /v1/prompts/{slug} — a single prompt plus related prompts.
func (h *PromptsHandler) HandleGet(ctx *fasthttp.RequestCtx) {
	slug := strings.TrimSpace(ctx.UserValue("slug").(string))
	if slug == "" {
		WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": "slug required"})
		return
	}
	prompt, ok := promptlib.BySlug(slug)
	if !ok {
		WriteJSONPublic(ctx, fasthttp.StatusNotFound, map[string]any{"error": "prompt not found"})
		return
	}
	limit := parseLimit(ctx, 6, 24)
	var related []promptlib.Prompt
	if h.index != nil {
		related = h.index.Related(slug, limit)
	} else {
		related = promptlib.RelatedMeta(slug, limit)
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{
		"prompt":  prompt,
		"related": related,
	})
}
