package handler

import (
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/router"
)

type ModelsHandler struct {
	router *router.Router
	cache  ttlBodyCache
}

type ttlBodyEntry struct {
	body    []byte
	expires time.Time
}

type ttlBodyCache struct {
	v atomic.Pointer[ttlBodyEntry]
}

const modelsListTTL = 30 * time.Second

func (c *ttlBodyCache) serve(ctx *fasthttp.RequestCtx, build func() any) {
	e := c.v.Load()
	if e == nil || time.Now().After(e.expires) {
		body, err := json.Marshal(build())
		if err != nil {
			writeJSON(ctx, 500, map[string]any{"error": map[string]string{"message": "internal error"}})
			return
		}
		e = &ttlBodyEntry{body: body, expires: time.Now().Add(modelsListTTL)}
		c.v.Store(e)
	}
	ctx.SetStatusCode(200)
	ctx.SetContentType("application/json")
	ctx.SetBody(e.body)
}

func NewModelsHandler(r *router.Router) *ModelsHandler {
	return &ModelsHandler{router: r}
}

func (h *ModelsHandler) HandleListModels(ctx *fasthttp.RequestCtx) {
	h.cache.serve(ctx, func() any {
		return map[string]any{
			"object": "list",
			"data":   h.router.ListModels(),
		}
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
