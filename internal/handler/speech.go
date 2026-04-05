package handler

import (
	"encoding/json"
	"log"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
)

type SpeechHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
}

func NewSpeechHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder) *SpeechHandler {
	return &SpeechHandler{router: r, billing: b, recorder: rec}
}

func (h *SpeechHandler) HandleSpeechGeneration(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}

	var req model.SpeechRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON: "+err.Error())
		return
	}
	if req.Model == "" {
		writeError(ctx, 400, "invalid_request", "model is required")
		return
	}
	if req.Input == "" {
		writeError(ctx, 400, "invalid_request", "input is required")
		return
	}

	originalModel := req.Model
	candidates, err := h.router.ResolveWithRetries(req.Model)
	if err != nil {
		writeError(ctx, 404, "model_not_found", err.Error())
		return
	}

	for i, cand := range candidates {
		speechProv, ok := cand.Provider.(provider.SpeechProvider)
		if !ok {
			log.Printf("speech: %s does not support speech", cand.Provider.Name())
			continue
		}

		req.Model = cand.ModelCfg.ProviderModelID
		start := time.Now()

		resp, err := speechProv.GenerateSpeech(ctx, &req)
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
				log.Printf("speech fallback: %s/%s -> %s/%s",
					cand.Provider.Name(), cand.ModelCfg.ID,
					candidates[i+1].Provider.Name(), candidates[i+1].ModelCfg.ID)
			}
			continue
		}

		h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
		chars := resp.Characters
		if chars == 0 {
			chars = len(req.Input)
		}
		cost, _ := h.billing.Deduct(ctx, userID, cand.ModelCfg.ID, chars, 0, "", "")
		h.recorder.RecordSuccess(userID, apiKeyID, originalModel, cand.Provider.Name(),
			chars, 0, int(latency.Milliseconds()), 0, cost, false)

		writeJSON(ctx, 200, resp)
		return
	}

	writeError(ctx, 502, "provider_error", "all providers failed for model "+originalModel)
}
