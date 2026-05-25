package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
)

type Model3DHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
	jobs     *model3DJobCache
	jobQ     *queries.Model3DJobQueries
}

func NewModel3DHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder, jobQ *queries.Model3DJobQueries) *Model3DHandler {
	return &Model3DHandler{router: r, billing: b, recorder: rec, jobs: newModel3DJobCache(), jobQ: jobQ}
}

func (h *Model3DHandler) HandleModel3DGeneration(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}

	req, originalModel, actualCost, err := parseModel3DGenerationRequest(ctx.PostBody())
	if err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}

	async := req.Async || string(ctx.QueryArgs().Peek("async")) == "true" || string(ctx.Request.Header.Peek("Prefer")) == "respond-async"
	if h.jobQ != nil {
		h.handleDurableModel3DGeneration(ctx, req, originalModel, userID, apiKeyID, actualCost, async)
		return
	}

	job, cached := h.jobs.getOrCreate(req)
	if !cached {
		if err := h.precheckModel3DBilling(ctx, userID, actualCost); err != nil {
			h.jobs.complete(job.ID, model3DExecutionResult{StatusCode: 402, ErrorType: "billing_error", ErrorMessage: "Insufficient credits. Please add credits to continue."})
			writeError(ctx, 402, "billing_error", "Insufficient credits. Please add credits to continue.")
			return
		}
		go h.runModel3DJob(job.ID, req, userID, apiKeyID, requestAppAttribution(ctx))
	}
	if async {
		writeJSON(ctx, 202, model3DJobPayload(job, cached))
		return
	}
	if job.Status == model3DJobCompleted && job.Result != nil {
		writeJSON(ctx, 200, job.Result)
		return
	}
	if job.Status == model3DJobFailed {
		writeError(ctx, model3DJobHTTPStatus(job), job.ErrorType, job.Error)
		return
	}

	waited, ok := h.jobs.wait(job.ID, 85*time.Second)
	if !ok {
		writeError(ctx, 404, "job_not_found", "3D generation job not found")
		return
	}
	switch waited.Status {
	case model3DJobCompleted:
		writeJSON(ctx, 200, waited.Result)
	case model3DJobFailed:
		writeError(ctx, model3DJobHTTPStatus(waited), waited.ErrorType, waited.Error)
	default:
		ctx.Response.Header.Set("Retry-After", "2")
		writeJSON(ctx, 202, model3DJobPayload(waited, cached))
	}
}

func parseModel3DGenerationRequest(body []byte) (model.Model3DGenerationRequest, string, int64, error) {
	var req model.Model3DGenerationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return req, "", 0, fmt.Errorf("Invalid JSON: %w", err)
	}
	if req.Model == "" {
		req.Model = "pixal3d-image-to-3d"
	}
	if req.ImageURL == "" {
		return req, "", 0, fmt.Errorf("image_url is required")
	}
	if !isPublicHTTPURL(req.ImageURL) {
		return req, "", 0, fmt.Errorf("image_url must be a public http or https URL")
	}
	originalModel := req.Model
	req.TextureSize = normalize3DTextureSize(req.TextureSize)
	return req, originalModel, pixal3DRequestCost(req.TextureSize), nil
}

func (h *Model3DHandler) precheckModel3DBilling(ctx context.Context, userID string, actualCost int64) error {
	if h.billing == nil {
		return nil
	}
	return h.billing.PreCheckFixed(ctx, userID, actualCost)
}

func (h *Model3DHandler) HandleModel3DGenerationJob(ctx *fasthttp.RequestCtx) {
	jobID, _ := ctx.UserValue("job_id").(string)
	if jobID == "" {
		writeError(ctx, 400, "invalid_request", "job_id is required")
		return
	}
	if h.jobQ != nil {
		job, err := h.jobQ.GetByID(ctx, jobID)
		if err != nil {
			writeError(ctx, 404, "not_found", "3D generation job not found")
			return
		}
		writeJSON(ctx, 200, durableModel3DJobPayload(job, true))
		return
	}
	job, ok := h.jobs.get(jobID)
	if !ok {
		writeError(ctx, 404, "not_found", "3D generation job not found")
		return
	}
	writeJSON(ctx, 200, model3DJobPayload(job, true))
}

func (h *Model3DHandler) handleDurableModel3DGeneration(ctx *fasthttp.RequestCtx, req model.Model3DGenerationRequest, originalModel, userID, apiKeyID string, actualCost int64, async bool) {
	req.Async = false
	requestJSON, _ := json.Marshal(req)
	job, created, err := h.jobQ.GetOrCreate(ctx, &queries.Model3DJob{
		ID:          newModel3DJobID(),
		RequestHash: model3DRequestCacheKey(req),
		UserID:      userID,
		APIKeyID:    apiKeyID,
		Model:       originalModel,
		Status:      string(model3DJobQueued),
		RequestJSON: requestJSON,
	})
	if err != nil {
		writeError(ctx, 500, "server_error", "failed to create 3D generation job")
		return
	}
	if created {
		if err := h.precheckModel3DBilling(ctx, userID, actualCost); err != nil {
			_ = h.jobQ.Fail(context.Background(), job.ID, "billing_error", "Insufficient credits. Please add credits to continue.")
			writeError(ctx, 402, "billing_error", "Insufficient credits. Please add credits to continue.")
			return
		}
		go h.runDurableModel3DJob(job.ID, req, userID, apiKeyID, requestAppAttribution(ctx))
	}
	if async {
		writeJSON(ctx, 202, durableModel3DJobPayload(job, !created))
		return
	}
	switch job.Status {
	case string(model3DJobCompleted):
		resp, err := durableModel3DJobResult(job)
		if err != nil {
			writeError(ctx, 502, "provider_error", "stored 3D result is invalid")
			return
		}
		writeJSON(ctx, 200, resp)
		return
	case string(model3DJobFailed):
		writeError(ctx, 502, stringPtrValue(job.ErrorType, "provider_error"), stringPtrValue(job.ErrorMessage, "3D generation failed"))
		return
	}

	waited, ok := h.waitDurableModel3DJob(job.ID, 85*time.Second)
	if !ok {
		writeError(ctx, 404, "job_not_found", "3D generation job not found")
		return
	}
	switch waited.Status {
	case string(model3DJobCompleted):
		resp, err := durableModel3DJobResult(waited)
		if err != nil {
			writeError(ctx, 502, "provider_error", "stored 3D result is invalid")
			return
		}
		writeJSON(ctx, 200, resp)
	case string(model3DJobFailed):
		writeError(ctx, 502, stringPtrValue(waited.ErrorType, "provider_error"), stringPtrValue(waited.ErrorMessage, "3D generation failed"))
	default:
		ctx.Response.Header.Set("Retry-After", "2")
		writeJSON(ctx, 202, durableModel3DJobPayload(waited, !created))
	}
}

func (h *Model3DHandler) waitDurableModel3DJob(jobID string, timeout time.Duration) (*queries.Model3DJob, bool) {
	deadline := time.Now().Add(timeout)
	for {
		job, err := h.jobQ.GetByID(context.Background(), jobID)
		if err != nil {
			return nil, false
		}
		if job.Status == string(model3DJobCompleted) || job.Status == string(model3DJobFailed) || time.Now().After(deadline) {
			return job, true
		}
		time.Sleep(2 * time.Second)
	}
}

func (h *Model3DHandler) runDurableModel3DJob(jobID string, req model.Model3DGenerationRequest, userID, apiKeyID string, app requestApp) {
	bg, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	_ = h.jobQ.MarkRunning(bg, jobID)
	result := h.executeModel3DGeneration(bg, nil, req, userID, apiKeyID, app)
	if result.Response != nil && result.StatusCode < 400 {
		body, _ := json.Marshal(result.Response)
		_ = h.jobQ.Complete(context.Background(), jobID, body)
		return
	}
	errType := result.ErrorType
	if errType == "" {
		errType = "provider_error"
	}
	_ = h.jobQ.Fail(context.Background(), jobID, errType, result.ErrorMessage)
}

func (h *Model3DHandler) runModel3DJob(jobID string, req model.Model3DGenerationRequest, userID, apiKeyID string, app requestApp) {
	h.jobs.markRunning(jobID)
	bg, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	h.jobs.complete(jobID, h.executeModel3DGeneration(bg, nil, req, userID, apiKeyID, app))
}

func (h *Model3DHandler) executeModel3DGeneration(ctx context.Context, byokCtx *fasthttp.RequestCtx, req model.Model3DGenerationRequest, userID, apiKeyID string, app requestApp) model3DExecutionResult {
	originalModel := req.Model
	textureSize := normalize3DTextureSize(req.TextureSize)
	actualCost := pixal3DRequestCost(textureSize)

	candidates, err := h.router.ResolveForRequest(originalModel, originalModel)
	if err != nil {
		return model3DExecutionResult{StatusCode: 404, ErrorType: "model_not_found", ErrorMessage: err.Error()}
	}

	for i, cand := range candidates {
		req.Model = cand.ModelCfg.ProviderModelID
		attempts := []provider.Provider{cand.Provider}
		if byokCtx != nil {
			if byokProvider, ok := getBYOKProvider(byokCtx, cand.Provider.Name()); ok {
				attempts = []provider.Provider{byokProvider, cand.Provider}
			}
		}

		for _, attempt := range attempts {
			model3DProv, ok := attempt.(provider.Model3DProvider)
			if !ok {
				log.Printf("3d: %s does not support 3D generation", attempt.Name())
				continue
			}

			start := time.Now()
			resp, err := model3DProv.Generate3D(ctx, &req)
			latency := time.Since(start)
			if err != nil {
				statusCode := 502
				errMsg := "upstream error"
				if pe, ok := err.(*provider.ProviderError); ok {
					statusCode = pe.StatusCode
					errMsg = pe.Message
					if !pe.Retryable {
						if h.recorder != nil {
							h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, attempt.Name(),
								int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
						}
						return model3DExecutionResult{StatusCode: statusCode, ErrorType: "provider_error", ErrorMessage: errMsg}
					}
				}
				if h.recorder != nil {
					h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, attempt.Name(),
						int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
				}
				continue
			}

			h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
			resp.Model = originalModel
			resp.BackendUsed = attempt.Name()
			resp.CreditsCharged = float64(actualCost) / 10000
			if resp.Billing == nil {
				resp.Billing = &model.Model3DBilling{
					TextureSize:       textureSize,
					ExternalCostUSD:   float64(pixal3DExternalCostCents(textureSize)) / 100,
					ExternalCostCents: pixal3DExternalCostCents(textureSize),
				}
			}
			if h.billing != nil {
				if err := h.billing.DeductFixed(ctx, userID, cand.ModelCfg.ID, actualCost, fmt.Sprintf("3D model: %s, texture: %d", cand.ModelCfg.ID, textureSize), ""); err != nil {
					return model3DExecutionResult{StatusCode: 402, ErrorType: "billing_error", ErrorMessage: "Insufficient credits. Please add credits to continue."}
				}
			}
			if h.recorder != nil {
				h.recorder.RecordSuccessWithApp(userID, apiKeyID, originalModel, attempt.Name(),
					0, 1, int(latency.Milliseconds()), 0, actualCost, false, app.ID, app.URL, app.Title, app.Categories)
			}

			return model3DExecutionResult{Response: resp, StatusCode: 200}
		}

		h.router.MarkModelUnhealthy(cand.Provider.Name(), cand.ModelCfg.ID)
		if i < len(candidates)-1 {
			log.Printf("3d fallback: %s/%s -> %s/%s",
				cand.Provider.Name(), cand.ModelCfg.ID,
				candidates[i+1].Provider.Name(), candidates[i+1].ModelCfg.ID)
		}
	}

	return model3DExecutionResult{StatusCode: 502, ErrorType: "provider_error", ErrorMessage: "all providers failed for model " + originalModel}
}

func isPublicHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func normalize3DTextureSize(value int) int {
	switch value {
	case 2048, 4096:
		return value
	default:
		return 1024
	}
}

func pixal3DExternalCostCents(textureSize int) int {
	if normalize3DTextureSize(textureSize) == 1024 {
		return 30
	}
	return 42
}

func pixal3DRequestCost(textureSize int) int64 {
	return int64(pixal3DExternalCostCents(textureSize) * 100)
}
