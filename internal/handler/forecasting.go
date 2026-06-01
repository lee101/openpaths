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

// ForecastingHandler serves /v1/forecasts — time-series forecasting (chronos2).
type ForecastingHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
}

func NewForecastingHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder) *ForecastingHandler {
	return &ForecastingHandler{router: r, billing: b, recorder: rec}
}

func (h *ForecastingHandler) HandleForecast(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}
	app := requestAppAttribution(ctx)

	var req model.ForecastingRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON: "+err.Error())
		return
	}
	if req.Model == "" {
		writeError(ctx, 400, "invalid_request", "model is required")
		return
	}
	if len(req.History()) == 0 {
		writeError(ctx, 400, "invalid_request", "context (historical series) is required")
		return
	}

	resp, status, errType, errMsg := h.executeForecast(ctx, &req, userID, apiKeyID, app)
	if status >= 400 || resp == nil {
		writeError(ctx, status, errType, errMsg)
		return
	}
	writeJSON(ctx, 200, resp)
}

func (h *ForecastingHandler) executeForecast(ctx context.Context, req *model.ForecastingRequest, userID, apiKeyID string, app requestApp) (*model.ForecastingResponse, int, string, string) {
	originalModel := req.Model
	candidates, err := h.router.ResolveForRequest(originalModel, originalModel)
	if err != nil {
		return nil, 404, "model_not_found", err.Error()
	}

	for i, cand := range candidates {
		fc, ok := cand.Provider.(provider.ForecastingProvider)
		if !ok {
			log.Printf("forecast: %s does not support forecasting", cand.Provider.Name())
			continue
		}

		req.Model = cand.ModelCfg.ProviderModelID
		start := time.Now()
		resp, err := fc.GenerateForecast(ctx, req)
		latency := time.Since(start)

		if err != nil {
			statusCode := 502
			errMsg := "upstream error"
			if pe, ok := err.(*provider.ProviderError); ok {
				statusCode = pe.StatusCode
				errMsg = pe.Message
				if !pe.Retryable && i == len(candidates)-1 {
					h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
						int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
					return nil, statusCode, "provider_error", errMsg
				}
			}
			h.router.MarkModelUnhealthy(cand.Provider.Name(), cand.ModelCfg.ID)
			h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
				int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
			if i < len(candidates)-1 {
				log.Printf("forecast fallback: %s/%s -> %s/%s",
					cand.Provider.Name(), cand.ModelCfg.ID,
					candidates[i+1].Provider.Name(), candidates[i+1].ModelCfg.ID)
			}
			continue
		}

		h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
		resp.Model = originalModel
		cost, _ := h.billing.DeductForecast(ctx, userID, cand.ModelCfg.ID, "")
		h.recorder.RecordSuccessWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
			0, 0, int(latency.Milliseconds()), 0, cost, false, app.ID, app.URL, app.Title, app.Categories)

		return resp, 200, "", ""
	}

	return nil, 502, "provider_error", "all providers failed for model " + originalModel
}
