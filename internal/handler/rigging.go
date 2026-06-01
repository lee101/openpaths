package handler

import (
	"context"
	"encoding/json"
	"fmt"
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

type MeshRiggingHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
	jobs     *riggingJobCache
}

func NewMeshRiggingHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder) *MeshRiggingHandler {
	return &MeshRiggingHandler{router: r, billing: b, recorder: rec, jobs: newRiggingJobCache()}
}

func (h *MeshRiggingHandler) HandleMeshRigging(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}

	req, actualCost, err := parseMeshRiggingRequest(ctx.PostBody())
	if err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}

	async := req.Async || string(ctx.QueryArgs().Peek("async")) == "true" || string(ctx.Request.Header.Peek("Prefer")) == "respond-async"

	job, cached := h.jobs.getOrCreate(req)
	if !cached {
		if err := h.precheckBilling(ctx, userID, actualCost); err != nil {
			h.jobs.complete(job.ID, riggingExecutionResult{StatusCode: 402, ErrorType: "billing_error", ErrorMessage: "Insufficient credits. Please add credits to continue."})
			writeError(ctx, 402, "billing_error", "Insufficient credits. Please add credits to continue.")
			return
		}
		go h.runRiggingJob(job.ID, req, userID, apiKeyID, requestAppAttribution(ctx))
	}
	if async {
		writeJSON(ctx, 202, riggingJobPayload(job, cached))
		return
	}
	if job.Status == model3DJobCompleted && job.Result != nil {
		writeJSON(ctx, 200, job.Result)
		return
	}
	if job.Status == model3DJobFailed {
		writeError(ctx, riggingJobHTTPStatus(job), job.ErrorType, job.Error)
		return
	}

	waited, ok := h.jobs.wait(job.ID, 85*time.Second)
	if !ok {
		writeError(ctx, 404, "job_not_found", "rigging job not found")
		return
	}
	switch waited.Status {
	case model3DJobCompleted:
		writeJSON(ctx, 200, waited.Result)
	case model3DJobFailed:
		writeError(ctx, riggingJobHTTPStatus(waited), waited.ErrorType, waited.Error)
	default:
		ctx.Response.Header.Set("Retry-After", "2")
		writeJSON(ctx, 202, riggingJobPayload(waited, cached))
	}
}

func (h *MeshRiggingHandler) HandleMeshRiggingJob(ctx *fasthttp.RequestCtx) {
	jobID, _ := ctx.UserValue("job_id").(string)
	if jobID == "" {
		writeError(ctx, 400, "invalid_request", "job_id is required")
		return
	}
	job, ok := h.jobs.get(jobID)
	if !ok {
		writeError(ctx, 404, "not_found", "rigging job not found")
		return
	}
	writeJSON(ctx, 200, riggingJobPayload(job, true))
}

func parseMeshRiggingRequest(body []byte) (model.MeshRiggingRequest, int64, error) {
	var req model.MeshRiggingRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return req, 0, fmt.Errorf("Invalid JSON: %w", err)
	}
	if req.Model == "" {
		req.Model = "meshy-rigging"
	}
	if req.ModelURL == "" {
		return req, 0, fmt.Errorf("model_url is required")
	}
	if !isPublicHTTPURL(req.ModelURL) {
		return req, 0, fmt.Errorf("model_url must be a public http or https URL")
	}
	return req, riggingRequestCost(req.EnableAnimation), nil
}

func (h *MeshRiggingHandler) precheckBilling(ctx context.Context, userID string, actualCost int64) error {
	if h.billing == nil {
		return nil
	}
	return h.billing.PreCheckFixed(ctx, userID, actualCost)
}

func (h *MeshRiggingHandler) runRiggingJob(jobID string, req model.MeshRiggingRequest, userID, apiKeyID string, app requestApp) {
	h.jobs.markRunning(jobID)
	bg, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	h.jobs.complete(jobID, h.executeMeshRigging(bg, req, userID, apiKeyID, app))
}

func (h *MeshRiggingHandler) executeMeshRigging(ctx context.Context, req model.MeshRiggingRequest, userID, apiKeyID string, app requestApp) riggingExecutionResult {
	originalModel := req.Model
	actualCost := riggingRequestCost(req.EnableAnimation)

	candidates, err := h.router.ResolveForRequest(originalModel, originalModel)
	if err != nil {
		return riggingExecutionResult{StatusCode: 404, ErrorType: "model_not_found", ErrorMessage: err.Error()}
	}

	for i, cand := range candidates {
		req.Model = cand.ModelCfg.ProviderModelID
		riggingProv, ok := cand.Provider.(provider.MeshRiggingProvider)
		if !ok {
			log.Printf("rigging: %s does not support mesh rigging", cand.Provider.Name())
			continue
		}

		start := time.Now()
		resp, err := riggingProv.RigMesh(ctx, &req)
		latency := time.Since(start)
		if err != nil {
			statusCode := 502
			errMsg := "upstream error"
			if pe, ok := err.(*provider.ProviderError); ok {
				statusCode = pe.StatusCode
				errMsg = pe.Message
				if !pe.Retryable {
					if h.recorder != nil {
						h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
							int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
					}
					return riggingExecutionResult{StatusCode: statusCode, ErrorType: "provider_error", ErrorMessage: errMsg}
				}
			}
			if h.recorder != nil {
				h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
					int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
			}
			h.router.MarkModelUnhealthy(cand.Provider.Name(), cand.ModelCfg.ID)
			if i < len(candidates)-1 {
				continue
			}
			return riggingExecutionResult{StatusCode: statusCode, ErrorType: "provider_error", ErrorMessage: errMsg}
		}

		h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
		resp.Model = originalModel
		resp.BackendUsed = cand.Provider.Name()
		resp.CreditsCharged = float64(actualCost) / 10000
		if h.billing != nil {
			if err := h.billing.DeductFixed(ctx, userID, cand.ModelCfg.ID, actualCost, fmt.Sprintf("mesh rigging: %s, animation: %t", cand.ModelCfg.ID, req.EnableAnimation), ""); err != nil {
				return riggingExecutionResult{StatusCode: 402, ErrorType: "billing_error", ErrorMessage: "Insufficient credits. Please add credits to continue."}
			}
		}
		if h.recorder != nil {
			h.recorder.RecordSuccessWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
				0, 1, int(latency.Milliseconds()), 0, actualCost, false, app.ID, app.URL, app.Title, app.Categories)
		}
		return riggingExecutionResult{Response: resp, StatusCode: 200}
	}

	return riggingExecutionResult{StatusCode: 502, ErrorType: "provider_error", ErrorMessage: "all providers failed for model " + originalModel}
}

// riggingRequestCost returns the fixed charge in credit-units ($1 = 10000):
// $0.20 base, $0.32 with an animation preset.
func riggingRequestCost(animation bool) int64 {
	if animation {
		return 3200
	}
	return 2000
}
