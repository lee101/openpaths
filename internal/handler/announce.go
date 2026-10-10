package handler

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/discovery"
	"github.com/openpaths/openpaths/internal/email"
)

type AnnounceHandler struct {
	announcer *email.Announcer
	disc      *discovery.Service
}

func NewAnnounceHandler(a *email.Announcer, disc *discovery.Service) *AnnounceHandler {
	return &AnnounceHandler{announcer: a, disc: disc}
}

func announceFail(ctx *fasthttp.RequestCtx, err error) {
	var ae *email.AnnounceError
	if errors.As(err, &ae) {
		writeError(ctx, ae.Code, "announce_error", ae.Message)
		return
	}
	writeError(ctx, 500, "server_error", err.Error())
}

func (h *AnnounceHandler) HandleAnnounce(ctx *fasthttp.RequestCtx) {
	var req email.AnnounceRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON body")
		return
	}
	res, err := h.announcer.Announce(ctx, req)
	if err != nil {
		announceFail(ctx, err)
		return
	}
	status := 200
	if res.Started {
		status = 202
	}
	writeJSON(ctx, status, res)
}

func (h *AnnounceHandler) HandleStatus(ctx *fasthttp.RequestCtx) {
	id, _ := ctx.UserValue("model_id").(string)
	if u, err := url.PathUnescape(id); err == nil {
		id = u
	}
	id = strings.Trim(id, "/")
	if id == "" {
		writeError(ctx, 400, "invalid_request", "model_id required")
		return
	}
	st, err := h.announcer.Status(ctx, id)
	if err != nil {
		announceFail(ctx, err)
		return
	}
	writeJSON(ctx, 200, st)
}

func (h *AnnounceHandler) HandleCandidates(ctx *fasthttp.RequestCtx) {
	pending, err := h.announcer.Candidates(ctx)
	if err != nil {
		announceFail(ctx, err)
		return
	}
	discovered := []discovery.NewModel{}
	if h.disc != nil {
		discovered = h.disc.NewlyDiscovered()
	}
	writeJSON(ctx, 200, map[string]any{
		"catalog_pending": pending,
		"discovered":      discovered,
	})
}
