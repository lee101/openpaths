package handler

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/savedresp"
)

// UsageHandler serves the private "saved responses" search + settings endpoints.
type UsageHandler struct {
	saver *savedresp.Saver
	userQ *queries.UserQueries
}

func NewUsageHandler(saver *savedresp.Saver, userQ *queries.UserQueries) *UsageHandler {
	return &UsageHandler{saver: saver, userQ: userQ}
}

func (h *UsageHandler) HandleGetSettings(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	u, err := h.userQ.GetByID(ctx, userID)
	if err != nil {
		writeError(ctx, 500, "server_error", "Failed to get user")
		return
	}
	writeJSON(ctx, 200, map[string]any{
		"text_enabled":  u.SaveResponsesText,
		"image_enabled": u.SaveResponsesImages,
		"available":     h.saver != nil,
	})
}

type usageSettingsRequest struct {
	TextEnabled  bool `json:"text_enabled"`
	ImageEnabled bool `json:"image_enabled"`
}

func (h *UsageHandler) HandleUpdateSettings(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	var req usageSettingsRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON")
		return
	}
	if err := h.userQ.SetResponseSaving(ctx, userID, req.TextEnabled, req.ImageEnabled); err != nil {
		writeError(ctx, 500, "server_error", "Failed to update settings")
		return
	}
	if h.saver != nil {
		h.saver.Invalidate(userID)
	}
	writeJSON(ctx, 200, map[string]any{
		"text_enabled":  req.TextEnabled,
		"image_enabled": req.ImageEnabled,
	})
}

// HandleSearch serves GET /account/usage/responses?kind=&q=&limit=&offset=
func (h *UsageHandler) HandleSearch(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if h.saver == nil {
		writeError(ctx, 503, "unavailable", "Response saving is not enabled on this server")
		return
	}
	kind := normalizeKind(string(ctx.QueryArgs().Peek("kind")))
	q := string(ctx.QueryArgs().Peek("q"))
	limit, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("limit")))
	offset, _ := strconv.Atoi(string(ctx.QueryArgs().Peek("offset")))

	res, err := h.saver.Search(ctx, userID, kind, q, limit, offset)
	if err != nil {
		writeError(ctx, 500, "server_error", err.Error())
		return
	}
	writeJSON(ctx, 200, res)
}

// HandleItem serves GET /account/usage/responses/{id}
func (h *UsageHandler) HandleItem(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if h.saver == nil {
		writeError(ctx, 503, "unavailable", "Response saving is not enabled on this server")
		return
	}
	id, _ := ctx.UserValue("id").(string)
	item, err := h.saver.Get(ctx, userID, id)
	if err != nil {
		writeError(ctx, 404, "not_found", "Saved response not found")
		return
	}
	similar, _ := h.saver.Similar(ctx, userID, id, 12)
	writeJSON(ctx, 200, map[string]any{
		"item":    item,
		"similar": similar,
	})
}

func normalizeKind(kind string) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind == savedresp.KindImage || kind == "images" || kind == "image" {
		return savedresp.KindImage
	}
	return savedresp.KindText
}
