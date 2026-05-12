package handler

import (
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
)

type VideoHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
}

func NewVideoHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder) *VideoHandler {
	return &VideoHandler{router: r, billing: b, recorder: rec}
}

func (h *VideoHandler) HandleVideoGeneration(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}

	var req model.VideoGenerationRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON: "+err.Error())
		return
	}
	if req.Model == "" {
		writeError(ctx, 400, "invalid_request", "model is required")
		return
	}
	if req.Prompt == "" {
		writeError(ctx, 400, "invalid_request", "prompt is required")
		return
	}

	originalModel := req.Model
	autoResult := h.router.MaybeResolveAuto(ctx, req.Model, "video", req.Prompt)
	candidates, err := h.router.ResolveForRequest(originalModel, autoResult.ModelID)
	if err != nil {
		writeError(ctx, 404, "model_not_found", err.Error())
		return
	}

	for i, cand := range candidates {
		vidProv, ok := cand.Provider.(provider.VideoProvider)
		if !ok {
			log.Printf("video: %s does not support video generation", cand.Provider.Name())
			continue
		}

		req.Model = cand.ModelCfg.ProviderModelID
		start := time.Now()

		resp, err := vidProv.GenerateVideo(ctx, &req)
		latency := time.Since(start)

		if err != nil {
			statusCode := 502
			errMsg := "upstream error"
			if pe, ok := err.(*provider.ProviderError); ok {
				statusCode = pe.StatusCode
				errMsg = pe.Message
				if !pe.Retryable {
					h.recorder.RecordError(userID, apiKeyID, originalModel, cand.Provider.Name(),
						int(latency.Milliseconds()), statusCode, errMsg, false)
					writeError(ctx, statusCode, "provider_error", errMsg)
					return
				}
			}
			h.router.MarkModelUnhealthy(cand.Provider.Name(), cand.ModelCfg.ID)
			h.recorder.RecordError(userID, apiKeyID, originalModel, cand.Provider.Name(),
				int(latency.Milliseconds()), statusCode, errMsg, false)
			if i < len(candidates)-1 {
				log.Printf("video fallback: %s/%s -> %s/%s",
					cand.Provider.Name(), cand.ModelCfg.ID,
					candidates[i+1].Provider.Name(), candidates[i+1].ModelCfg.ID)
			}
			continue
		}

		h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
		resp.Model = originalModel
		cost, _ := h.billing.DeductVideo(ctx, userID, cand.ModelCfg.ID, videoDurationSeconds(req.Duration), len(req.VideoURLs) > 0, "")
		h.recorder.RecordSuccess(userID, apiKeyID, originalModel, cand.Provider.Name(),
			0, 1, int(latency.Milliseconds()), 0, cost, false)

		writeJSON(ctx, 200, resp)
		return
	}

	writeError(ctx, 502, "provider_error", "all providers failed for model "+originalModel)
}

func videoDurationSeconds(duration string) int {
	if duration == "" || duration == "auto" {
		return 10
	}
	n, err := strconv.Atoi(duration)
	if err != nil || n <= 0 {
		return 10
	}
	return n
}
