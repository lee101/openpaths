package handler

import (
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/artindex"
)

type ArtHandler struct {
	index *artindex.Service
}

func NewArtHandler(index *artindex.Service) *ArtHandler {
	return &ArtHandler{index: index}
}

func (h *ArtHandler) HandleStatus(ctx *fasthttp.RequestCtx) {
	if h.index == nil {
		WriteJSONPublic(ctx, fasthttp.StatusOK, artindex.Status{})
		return
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, h.index.Status())
}

func (h *ArtHandler) HandleSearch(ctx *fasthttp.RequestCtx) {
	if h.index == nil {
		WriteJSONPublic(ctx, fasthttp.StatusServiceUnavailable, map[string]any{
			"error": "art index is not configured",
		})
		return
	}
	query := strings.TrimSpace(string(ctx.QueryArgs().Peek("q")))
	if query == "" {
		WriteJSONPublic(ctx, fasthttp.StatusBadRequest, map[string]any{
			"error": "q parameter required",
		})
		return
	}
	limit, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("limit")))
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	results, err := h.index.Search(ctx, query, limit)
	if err != nil {
		WriteJSONPublic(ctx, fasthttp.StatusServiceUnavailable, map[string]any{
			"error":  err.Error(),
			"status": h.index.Status(),
		})
		return
	}
	WriteJSONPublic(ctx, fasthttp.StatusOK, map[string]any{
		"query":   query,
		"count":   len(results),
		"results": results,
		"status":  h.index.Status(),
	})
}
