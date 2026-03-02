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

func (h *ModelsHandler) HandleListModels(ctx *fasthttp.RequestCtx) {
	models := h.router.ListModels()
	writeJSON(ctx, 200, map[string]any{
		"object": "list",
		"data":   models,
	})
}

func (h *ModelsHandler) HandleGetModel(ctx *fasthttp.RequestCtx) {
	modelID, ok := ctx.UserValue("model_id").(string)
	if !ok || modelID == "" {
		writeJSON(ctx, 400, map[string]any{"error": map[string]string{"message": "model_id required"}})
		return
	}

	info, found := h.router.GetModelInfo(modelID)
	if !found {
		writeJSON(ctx, 404, map[string]any{"error": map[string]string{"message": "model not found", "type": "invalid_request_error"}})
		return
	}
	writeJSON(ctx, 200, info)
}
