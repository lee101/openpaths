package handler

import (
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/router"
)

type ModelsHandler struct {
	router *router.Router
}

func NewModelsHandler(r *router.Router) *ModelsHandler {
	return &ModelsHandler{router: r}
}

// HandleListModels handles GET /v1/models.
func (h *ModelsHandler) HandleListModels(ctx *fasthttp.RequestCtx) {
	models := h.router.ListModels()
	writeJSON(ctx, 200, map[string]any{
		"object": "list",
		"data":   models,
	})
}
