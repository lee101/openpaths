package handler

import (
	"context"
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

// StructuredHandler serves /v1/extract (Clairic typed field extraction) and
// /v1/decide (Clef decision questions). The request body is forwarded as-is
// apart from the model, and the typed answers come back unchanged.
type StructuredHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
}

func NewStructuredHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder) *StructuredHandler {
	return &StructuredHandler{router: r, billing: b, recorder: rec}
}

func (h *StructuredHandler) HandleExtract(ctx *fasthttp.RequestCtx) {
	h.handle(ctx, "clairic", "fields")
}

func (h *StructuredHandler) HandleDecide(ctx *fasthttp.RequestCtx) {
	h.handle(ctx, "clef", "questions")
}

func (h *StructuredHandler) handle(ctx *fasthttp.RequestCtx, defaultModel, requiredKey string) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}
	app := requestAppAttribution(ctx)

	var body map[string]any
	if err := json.Unmarshal(ctx.PostBody(), &body); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON: "+err.Error())
		return
	}
	modelID, _ := body["model"].(string)
	if modelID == "" {
		modelID = defaultModel
	}
	if fields, ok := body[requiredKey].(map[string]any); !ok || len(fields) == 0 {
		writeError(ctx, 400, "invalid_request", requiredKey+" must be a non-empty object")
		return
	}
	if body["state"] == nil && body["images"] == nil {
		writeError(ctx, 400, "invalid_request", "state (or images) is required")
		return
	}
	if prepaidGate(ctx, h.billing, modelID, 0) {
		return
	}

	result, status, errType, errMsg := h.execute(ctx, modelID, body, userID, apiKeyID, app)
	if result == nil {
		writeError(ctx, status, errType, errMsg)
		return
	}
	ctx.SetStatusCode(200)
	ctx.SetContentType("application/json")
	ctx.SetBody(result)
}

func (h *StructuredHandler) execute(ctx context.Context, originalModel string, body map[string]any, userID, apiKeyID string, app requestApp) (json.RawMessage, int, string, string) {
	candidates, err := h.router.ResolveForRequest(originalModel, originalModel)
	if err != nil {
		return nil, 404, "model_not_found", err.Error()
	}
	for i, cand := range candidates {
		sp, ok := cand.Provider.(provider.StructuredProvider)
		if !ok {
			log.Printf("structured: %s does not support structured models", cand.Provider.Name())
			continue
		}
		start := time.Now()
		result, inputTokens, err := sp.RunStructured(ctx, cand.ModelCfg.ProviderModelID, body)
		latency := time.Since(start)
		if err != nil {
			statusCode, errMsg := 502, "upstream error"
			if pe, ok := err.(*provider.ProviderError); ok {
				statusCode, errMsg = pe.StatusCode, pe.Message
				if !pe.Retryable && i == len(candidates)-1 {
					h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
						int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
					return nil, statusCode, "provider_error", errMsg
				}
			}
			h.router.MarkModelUnhealthy(cand.Provider.Name(), cand.ModelCfg.ID)
			h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
				int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
			continue
		}
		h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
		cost, _ := h.billing.Deduct(ctx, userID, cand.ModelCfg.ID, inputTokens, 0, "", "")
		h.recorder.RecordSuccessWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
			inputTokens, 0, int(latency.Milliseconds()), 0, cost, false, app.ID, app.URL, app.Title, app.Categories)
		return result, 200, "", ""
	}
	return nil, 502, "provider_error", "all providers failed for model " + originalModel
}
