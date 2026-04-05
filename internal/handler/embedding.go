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

type EmbeddingHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
	fallback []provider.EmbeddingProvider
}

func NewEmbeddingHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder, fallback []provider.EmbeddingProvider) *EmbeddingHandler {
	return &EmbeddingHandler{router: r, billing: b, recorder: rec, fallback: fallback}
}

func (h *EmbeddingHandler) HandleEmbedding(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}

	var req model.EmbeddingRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON: "+err.Error())
		return
	}
	if req.Input == nil {
		writeError(ctx, 400, "invalid_request", "input is required")
		return
	}

	originalModel := req.Model

	// Try router-resolved providers first (OpenRouter embedding models etc)
	if req.Model != "" {
		candidates, err := h.router.ResolveWithRetries(req.Model)
		if err == nil {
			for i, cand := range candidates {
				embProv, ok := cand.Provider.(provider.EmbeddingProvider)
				if !ok {
					continue
				}
				req.Model = cand.ModelCfg.ProviderModelID
				start := time.Now()
				resp, err := embProv.Embed(ctx, &req)
				latency := time.Since(start)

				if err != nil {
					if pe, ok := err.(*provider.ProviderError); ok && !pe.Retryable {
						h.recorder.RecordError(userID, apiKeyID, originalModel, cand.Provider.Name(),
							int(latency.Milliseconds()), pe.StatusCode, pe.Message, false)
						writeError(ctx, pe.StatusCode, "provider_error", pe.Message)
						return
					}
					h.router.MarkModelUnhealthy(cand.Provider.Name(), cand.ModelCfg.ID)
					log.Printf("embedding: %s failed (%dms): %v", cand.Provider.Name(), latency.Milliseconds(), err)
					if i < len(candidates)-1 {
						continue
					}
				} else {
					h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
					resp.Model = originalModel
					cost, _ := h.billing.Deduct(ctx, userID, cand.ModelCfg.ID, resp.Usage.TotalTokens, 0, "", "")
					h.recorder.RecordSuccess(userID, apiKeyID, originalModel, cand.Provider.Name(),
						resp.Usage.TotalTokens, 0, int(latency.Milliseconds()), 0, cost, false)
					writeJSON(ctx, 200, resp)
					return
				}
			}
		}
	}

	// Fall back to dedicated embedding providers (text-generator etc)
	for _, ep := range h.fallback {
		start := time.Now()
		resp, err := ep.Embed(ctx, &req)
		latency := time.Since(start)

		if err != nil {
			log.Printf("embedding: %s failed (%dms): %v", ep.Name(), latency.Milliseconds(), err)
			continue
		}

		resp.Model = originalModel
		log.Printf("embedding: %s ok (%dms)", ep.Name(), latency.Milliseconds())
		h.recorder.RecordSuccess(userID, apiKeyID, originalModel, ep.Name(),
			resp.Usage.TotalTokens, 0, int(latency.Milliseconds()), 0, 0, false)
		writeJSON(ctx, 200, resp)
		return
	}

	writeError(ctx, 502, "provider_error", "all embedding providers failed")
}
