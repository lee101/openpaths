package handler

import (
	"context"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/skillindex"
)

// skillSearcher is the semantic index seam (satisfied by *skillindex.Service);
// kept as an interface so the handler is unit-testable with a stub.
type skillSearcher interface {
	Ready() bool
	Status() skillindex.Status
	Search(ctx context.Context, query string, f model.SkillFilters, limit int) ([]skillindex.Result, error)
}

// skillStore is the DB seam (satisfied by *queries.SkillQueries).
type skillStore interface {
	List(ctx context.Context, f model.SkillFilters, limit, offset int) ([]model.Skill, error)
	SearchILIKE(ctx context.Context, query string, f model.SkillFilters, limit int) ([]model.Skill, error)
	GetBySlug(ctx context.Context, slug string) (*model.Skill, error)
	Count(ctx context.Context) (int, error)
	SourceCounts(ctx context.Context) ([]queries.Facet, error)
	CategoryCounts(ctx context.Context) ([]queries.Facet, error)
}

// SkillsHandler serves the public read-only agent-skill library. Semantic search
// is powered by the gobed-backed skillindex; it degrades to pg_trgm ILIKE search
// whenever the index is unavailable so the endpoint always answers.
type SkillsHandler struct {
	index skillSearcher
	q     skillStore
}

func NewSkillsHandler(index *skillindex.Service, q *queries.SkillQueries) *SkillsHandler {
	h := &SkillsHandler{}
	if index != nil {
		h.index = index
	}
	if q != nil {
		h.q = q
	}
	return h
}

func skillFilters(ctx *fasthttp.RequestCtx) model.SkillFilters {
	qa := ctx.QueryArgs()
	return model.SkillFilters{
		Source:   strings.TrimSpace(string(qa.Peek("source"))),
		Category: strings.TrimSpace(string(qa.Peek("category"))),
	}
}

// HandleList serves GET /v1/skills (browse + optional ?q= semantic search).
func (h *SkillsHandler) HandleList(ctx *fasthttp.RequestCtx) {
	if h.q == nil {
		WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{"object": "list", "count": 0, "skills": []any{}})
		return
	}
	f := skillFilters(ctx)
	limit := parseLimit(ctx, 200, 500)
	skills, err := h.q.List(ctx, f, limit, 0)
	if err != nil {
		WriteJSONPublic(ctx, fasthttp.StatusInternalServerError, map[string]any{"error": "could not list skills"})
		return
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{"object": "list", "count": len(skills), "skills": skills})
}

// HandleSearch serves GET /v1/skills/search?q=&k= — semantic when the index is
// ready, else pg_trgm ILIKE.
func (h *SkillsHandler) HandleSearch(ctx *fasthttp.RequestCtx) {
	query := strings.TrimSpace(string(ctx.QueryArgs().Peek("q")))
	f := skillFilters(ctx)
	k := parseLimit(ctx, 20, 50)
	if n, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("k"))); n > 0 && n <= 50 {
		k = n
	}
	if query == "" {
		WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{"object": "list", "query": "", "semantic": false, "count": 0, "results": []any{}})
		return
	}

	if h.index != nil && h.index.Ready() {
		if results, err := h.index.Search(ctx, query, f, k); err == nil {
			WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{
				"object": "list", "query": query, "semantic": true, "count": len(results), "results": results,
			})
			return
		}
	}

	// Fallback: pg_trgm ILIKE.
	var results []model.Skill
	if h.q != nil {
		results, _ = h.q.SearchILIKE(ctx, query, f, k)
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{
		"object": "list", "query": query, "semantic": false, "count": len(results), "results": results,
	})
}

// HandleGet serves GET /v1/skills/{slug} — a single skill with its full body and
// a copy-paste-ready markdown payload (Setup preamble + body).
func (h *SkillsHandler) HandleGet(ctx *fasthttp.RequestCtx) {
	slug, _ := ctx.UserValue("slug").(string)
	slug = strings.TrimSpace(slug)
	if slug == "" || h.q == nil {
		WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{"error": "slug required"})
		return
	}
	skill, err := h.q.GetBySlug(ctx, slug)
	if err != nil || skill == nil {
		WriteJSONPublic(ctx, fasthttp.StatusNotFound, map[string]any{"error": "skill not found"})
		return
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{
		"object":   "skill",
		"skill":    skill,
		"markdown": skill.Markdown(),
	})
}

// HandleMeta serves GET /v1/skills/meta — sources, categories, total, index status.
func (h *SkillsHandler) HandleMeta(ctx *fasthttp.RequestCtx) {
	out := map[string]any{"object": "meta", "total": 0, "sources": []any{}, "categories": []any{}}
	if h.q != nil {
		if total, err := h.q.Count(ctx); err == nil {
			out["total"] = total
		}
		if sources, err := h.q.SourceCounts(ctx); err == nil {
			out["sources"] = sources
		}
		if cats, err := h.q.CategoryCounts(ctx); err == nil {
			out["categories"] = cats
		}
	}
	if h.index != nil {
		out["index"] = h.index.Status()
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, out)
}
