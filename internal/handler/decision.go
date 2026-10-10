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

// DecisionHandler serves POST /v1/systemone, wire-compatible with the TypeSafe
// (Jev) API so the typesafe SDKs work with baseURL set to OpenPaths.
const defaultDecisionModel = "decision-model-preview"

type DecisionHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
}

func NewDecisionHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder) *DecisionHandler {
	return &DecisionHandler{router: r, billing: b, recorder: rec}
}

func (h *DecisionHandler) HandleDecision(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKeyID := ""
	if key, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey); key != nil {
		apiKeyID = key.ID
	}
	var req model.DecisionRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 422, "invalid_request", "Invalid JSON: "+err.Error())
		return
	}
	if req.Model == "" {
		req.Model = defaultDecisionModel
	}
	if err := req.Validate(); err != nil {
		writeError(ctx, 422, "invalid_request", err.Error())
		return
	}
	originalModel := req.Model
	candidates, err := h.router.ResolveWithRetries(req.Model)
	if err != nil {
		writeError(ctx, 404, "model_not_found", err.Error())
		return
	}
	app := requestAppAttribution(ctx)
	for _, cand := range candidates {
		decider, ok := cand.Provider.(provider.DecisionProvider)
		if !ok {
			continue
		}
		if prepaidGate(ctx, h.billing, cand.ModelCfg.ID, 1) {
			return
		}
		req.Model = cand.ModelCfg.ProviderModelID
		start := time.Now()
		resp, err := decider.Decide(ctx, &req)
		latency := int(time.Since(start).Milliseconds())
		if err != nil {
			pe, isProviderErr := err.(*provider.ProviderError)
			if isProviderErr && !pe.Retryable {
				h.recordError(userID, apiKeyID, originalModel, cand.Provider.Name(), latency, pe.StatusCode, pe.Message, app)
				writeError(ctx, pe.StatusCode, "provider_error", pe.Message)
				return
			}
			h.router.MarkModelUnhealthy(cand.Provider.Name(), cand.ModelCfg.ID)
			h.recordError(userID, apiKeyID, originalModel, cand.Provider.Name(), latency, 502, err.Error(), app)
			log.Printf("decision: %s/%s failed (%dms): %v", cand.Provider.Name(), cand.ModelCfg.ID, latency, err)
			continue
		}
		h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
		// Report the model that actually answered; TypeSafe SDK defaults to jev-latest.
		resp.Model = cand.ModelCfg.ID
		var cost int64
		if h.billing != nil {
			cost, _ = h.billing.Deduct(ctx, userID, cand.ModelCfg.ID, resp.Usage.InputTokens, resp.Usage.OutputTokens, "", "")
		}
		if h.recorder != nil {
			h.recorder.RecordSuccessWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
				resp.Usage.InputTokens, resp.Usage.OutputTokens, latency, 0, cost, false, app.ID, app.URL, app.Title, app.Categories)
		}
		writeJSON(ctx, 200, resp)
		return
	}
	writeError(ctx, 502, "provider_error", "no decision provider available for model "+originalModel)
}

func (h *DecisionHandler) recordError(userID, apiKeyID, modelID, providerName string, latency, status int, message string, app requestApp) {
	if h.recorder != nil {
		h.recorder.RecordErrorWithApp(userID, apiKeyID, modelID, providerName, latency, status, message, false, app.ID, app.URL, app.Title, app.Categories)
	}
}
