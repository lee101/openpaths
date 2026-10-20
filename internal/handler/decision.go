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
	local    map[string]provider.DecisionProvider
}

type decisionRoute struct {
	cfg     *model.ModelConfig
	decider provider.DecisionProvider
}

// NewDecisionHandler also serves models whose provider is an in-process embedder
// (gobed) that implements DecisionProvider; those are not in the router registry.
func NewDecisionHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder, embedders []provider.EmbeddingProvider) *DecisionHandler {
	local := map[string]provider.DecisionProvider{}
	for _, e := range embedders {
		if d, ok := e.(provider.DecisionProvider); ok {
			local[d.Name()] = d
		}
	}
	return &DecisionHandler{router: r, billing: b, recorder: rec, local: local}
}

func (h *DecisionHandler) routes(name string) ([]decisionRoute, error) {
	if cfg, ok := h.router.GetModelConfig(name); ok {
		if d := h.local[cfg.Provider]; d != nil {
			return []decisionRoute{{cfg, d}}, nil
		}
	}
	candidates, err := h.router.ResolveWithRetries(name)
	if err != nil {
		return nil, err
	}
	out := make([]decisionRoute, 0, len(candidates))
	for _, cand := range candidates {
		if d, ok := cand.Provider.(provider.DecisionProvider); ok {
			out = append(out, decisionRoute{cand.ModelCfg, d})
		}
	}
	return out, nil
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
	routes, err := h.routes(req.Model)
	if err != nil {
		writeError(ctx, 404, "model_not_found", err.Error())
		return
	}
	app := requestAppAttribution(ctx)
	for _, rt := range routes {
		cfg, decider := rt.cfg, rt.decider
		if prepaidGate(ctx, h.billing, cfg.ID, 1) {
			return
		}
		req.Model = cfg.ProviderModelID
		start := time.Now()
		resp, err := decider.Decide(ctx, &req)
		latency := int(time.Since(start).Milliseconds())
		if err != nil {
			pe, isProviderErr := err.(*provider.ProviderError)
			if isProviderErr && !pe.Retryable {
				h.recordError(userID, apiKeyID, originalModel, decider.Name(), latency, pe.StatusCode, pe.Message, app)
				writeError(ctx, pe.StatusCode, "provider_error", pe.Message)
				return
			}
			h.router.MarkModelUnhealthy(decider.Name(), cfg.ID)
			h.recordError(userID, apiKeyID, originalModel, decider.Name(), latency, 502, err.Error(), app)
			log.Printf("decision: %s/%s failed (%dms): %v", decider.Name(), cfg.ID, latency, err)
			continue
		}
		h.router.MarkModelHealthy(decider.Name(), cfg.ID)
		// Report the model that actually answered; TypeSafe SDK defaults to jev-latest.
		resp.Model = cfg.ID
		var cost int64
		if h.billing != nil {
			cost, _ = h.billing.Deduct(ctx, userID, cfg.ID, resp.Usage.InputTokens, resp.Usage.OutputTokens, "", "")
		}
		if h.recorder != nil {
			h.recorder.RecordSuccessWithApp(userID, apiKeyID, originalModel, decider.Name(),
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
