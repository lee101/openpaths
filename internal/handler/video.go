package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/modelaccess"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
	"github.com/openpaths/openpaths/internal/storage"
)

var activeVideoJobs int64

func ActiveVideoJobs() int64 {
	return atomic.LoadInt64(&activeVideoJobs)
}

type VideoHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
	jobs     *videoJobCache
	jobQ     *queries.VideoJobQueries
	store    storage.Store
	accessQ  modelaccess.Checker
}

func NewVideoHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder, jobQ *queries.VideoJobQueries) *VideoHandler {
	return &VideoHandler{router: r, billing: b, recorder: rec, jobs: newVideoJobCache(), jobQ: jobQ}
}

func (h *VideoHandler) SetStorage(store storage.Store) {
	h.store = store
}

func (h *VideoHandler) SetAccess(checker modelaccess.Checker) { h.accessQ = checker }

func (h *VideoHandler) HandleVideoGeneration(ctx *fasthttp.RequestCtx) {
	h.handleVideoRequest(ctx, "generation")
}

func (h *VideoHandler) HandleVideoExtension(ctx *fasthttp.RequestCtx) {
	h.handleVideoRequest(ctx, "extension")
}

func (h *VideoHandler) HandleVideoEdit(ctx *fasthttp.RequestCtx) {
	h.handleVideoRequest(ctx, "edit")
}

func (h *VideoHandler) handleVideoRequest(ctx *fasthttp.RequestCtx, operation string) {
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
	req.Prompt = req.TextPrompt()
	if req.Prompt == "" && len(req.Input) == 0 {
		writeError(ctx, 400, "invalid_request", "prompt or input is required")
		return
	}
	req.Operation = operation
	if (operation == "extension" || operation == "edit") && !req.HasVideoInput() {
		writeError(ctx, 400, "invalid_request", "video is required")
		return
	}
	if prepaidGate(ctx, h.billing, req.Model, 0) {
		return
	}
	if err := h.authorizeVideoRequest(ctx, userID, &req); err != nil {
		writeError(ctx, 403, "model_not_permitted", err.Error())
		return
	}

	async := req.Async || string(ctx.QueryArgs().Peek("async")) == "true" || string(ctx.Request.Header.Peek("Prefer")) == "respond-async"
	if h.jobQ != nil {
		h.handleDurableVideoGeneration(ctx, req, userID, apiKeyID, async)
		return
	}
	job, cached := h.jobs.getOrCreate(req)
	if !cached {
		go h.runVideoJob(job.ID, req, userID, apiKeyID, requestAppAttribution(ctx))
	}
	if async {
		writeJSON(ctx, 202, videoJobPayload(job, cached))
		return
	}
	if job.Status == videoJobCompleted && job.Result != nil {
		writeJSON(ctx, 200, job.Result)
		return
	}
	if job.Status == videoJobFailed {
		writeError(ctx, videoJobHTTPStatus(job), job.ErrorType, job.Error)
		return
	}

	waited, ok := h.jobs.wait(job.ID, 85*time.Second)
	if !ok {
		writeError(ctx, 404, "job_not_found", "video generation job not found")
		return
	}
	switch waited.Status {
	case videoJobCompleted:
		writeJSON(ctx, 200, waited.Result)
	case videoJobFailed:
		writeError(ctx, videoJobHTTPStatus(waited), waited.ErrorType, waited.Error)
	default:
		ctx.Response.Header.Set("Retry-After", "2")
		writeJSON(ctx, 202, videoJobPayload(waited, cached))
	}
}

func (h *VideoHandler) authorizeVideoRequest(ctx context.Context, userID string, req *model.VideoGenerationRequest) error {
	autoResult := h.router.MaybeResolveAuto(ctx, req.Model, "video", req.Prompt)
	candidates, err := h.router.ResolveForRequest(req.Model, autoResult.ModelID)
	if err != nil {
		return nil // preserve the existing model_not_found response path
	}
	_, err = modelaccess.FilterCandidates(ctx, h.accessQ, userID, req.Model, candidates)
	return err
}

func videoJobHTTPStatus(job *videoJob) int {
	if job.StatusCode >= 400 {
		return job.StatusCode
	}
	return 502
}

func (h *VideoHandler) HandleVideoGenerationJob(ctx *fasthttp.RequestCtx) {
	jobID, _ := ctx.UserValue("job_id").(string)
	if jobID == "" {
		writeError(ctx, 400, "invalid_request", "job_id is required")
		return
	}
	if h.jobQ != nil {
		job, err := h.jobQ.GetByID(ctx, jobID)
		if err != nil {
			writeError(ctx, 404, "not_found", "video generation job not found")
			return
		}
		writeJSON(ctx, 200, durableVideoJobPayload(job, true))
		return
	}
	job, ok := h.jobs.get(jobID)
	if !ok {
		writeError(ctx, 404, "not_found", "video generation job not found")
		return
	}
	writeJSON(ctx, 200, videoJobPayload(job, true))
}

func (h *VideoHandler) handleDurableVideoGeneration(ctx *fasthttp.RequestCtx, req model.VideoGenerationRequest, userID, apiKeyID string, async bool) {
	req.Async = false
	requestJSON, _ := json.Marshal(req)
	job, created, err := h.jobQ.GetOrCreate(ctx, &queries.VideoJob{
		ID:          newVideoJobID(),
		RequestHash: videoRequestCacheKey(req),
		UserID:      userID,
		APIKeyID:    apiKeyID,
		Model:       req.Model,
		Status:      string(videoJobQueued),
		RequestJSON: requestJSON,
	})
	if err != nil {
		writeError(ctx, 500, "server_error", "failed to create video generation job")
		return
	}
	if created {
		go h.runDurableVideoJob(job.ID, req, userID, apiKeyID, requestAppAttribution(ctx))
	}
	if async {
		writeJSON(ctx, 202, durableVideoJobPayload(job, !created))
		return
	}
	switch job.Status {
	case string(videoJobCompleted):
		resp, err := durableVideoJobResult(job)
		if err != nil {
			writeError(ctx, 502, "provider_error", "stored video result is invalid")
			return
		}
		writeJSON(ctx, 200, resp)
		return
	case string(videoJobFailed):
		writeError(ctx, 502, stringPtrValue(job.ErrorType, "provider_error"), stringPtrValue(job.ErrorMessage, "video generation failed"))
		return
	}

	waited, ok := h.waitDurableVideoJob(job.ID, 85*time.Second)
	if !ok {
		writeError(ctx, 404, "job_not_found", "video generation job not found")
		return
	}
	switch waited.Status {
	case string(videoJobCompleted):
		resp, err := durableVideoJobResult(waited)
		if err != nil {
			writeError(ctx, 502, "provider_error", "stored video result is invalid")
			return
		}
		writeJSON(ctx, 200, resp)
	case string(videoJobFailed):
		writeError(ctx, 502, stringPtrValue(waited.ErrorType, "provider_error"), stringPtrValue(waited.ErrorMessage, "video generation failed"))
	default:
		ctx.Response.Header.Set("Retry-After", "2")
		writeJSON(ctx, 202, durableVideoJobPayload(waited, !created))
	}
}

func (h *VideoHandler) waitDurableVideoJob(jobID string, timeout time.Duration) (*queries.VideoJob, bool) {
	deadline := time.Now().Add(timeout)
	for {
		job, err := h.jobQ.GetByID(context.Background(), jobID)
		if err != nil {
			return nil, false
		}
		if job.Status == string(videoJobCompleted) || job.Status == string(videoJobFailed) || time.Now().After(deadline) {
			return job, true
		}
		time.Sleep(2 * time.Second)
	}
}

func (h *VideoHandler) runDurableVideoJob(jobID string, req model.VideoGenerationRequest, userID, apiKeyID string, app requestApp) {
	atomic.AddInt64(&activeVideoJobs, 1)
	defer atomic.AddInt64(&activeVideoJobs, -1)
	bg, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	_ = h.jobQ.MarkRunning(bg, jobID)
	result := h.executeVideoGeneration(bg, req, userID, apiKeyID, app)
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

func (h *VideoHandler) runVideoJob(jobID string, req model.VideoGenerationRequest, userID, apiKeyID string, app requestApp) {
	atomic.AddInt64(&activeVideoJobs, 1)
	defer atomic.AddInt64(&activeVideoJobs, -1)
	h.jobs.markRunning(jobID)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	h.jobs.complete(jobID, h.executeVideoGeneration(ctx, req, userID, apiKeyID, app))
}

func (h *VideoHandler) executeVideoGeneration(ctx context.Context, req model.VideoGenerationRequest, userID, apiKeyID string, app requestApp) videoExecutionResult {
	originalModel := req.Model
	autoResult := h.router.MaybeResolveAuto(ctx, req.Model, "video", req.Prompt)
	candidates, err := h.router.ResolveForRequest(originalModel, autoResult.ModelID)
	if err != nil {
		return videoExecutionResult{StatusCode: 404, ErrorType: "model_not_found", ErrorMessage: err.Error()}
	}
	candidates, err = modelaccess.FilterCandidates(ctx, h.accessQ, userID, originalModel, candidates)
	if err != nil {
		return videoExecutionResult{StatusCode: 403, ErrorType: "model_not_permitted", ErrorMessage: err.Error()}
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
			log.Printf("video provider error: provider=%s model=%s status_latency_ms=%d err=%v",
				cand.Provider.Name(), cand.ModelCfg.ID, int(latency.Milliseconds()), err)
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				if h.recorder != nil {
					h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
						int(latency.Milliseconds()), 499, err.Error(), false, app.ID, app.URL, app.Title, app.Categories)
				}
				return videoExecutionResult{StatusCode: 499, ErrorType: "request_canceled", ErrorMessage: "request canceled"}
			}
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
					return videoExecutionResult{StatusCode: statusCode, ErrorType: "provider_error", ErrorMessage: errMsg}
				}
			}
			h.router.MarkModelUnhealthy(cand.Provider.Name(), cand.ModelCfg.ID)
			if h.recorder != nil {
				h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
					int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
			}
			if i < len(candidates)-1 {
				log.Printf("video fallback: %s/%s -> %s/%s",
					cand.Provider.Name(), cand.ModelCfg.ID,
					candidates[i+1].Provider.Name(), candidates[i+1].ModelCfg.ID)
			}
			continue
		}

		h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
		h.ensurePublicVideoURL(ctx, resp, originalModel)
		if wantsVideoWebM(req.OutputFormat) {
			optimized, err := h.reencodeVideoWebM(ctx, resp.VideoURL, originalModel)
			if err != nil {
				log.Printf("video reencode error: provider=%s model=%s err=%v", cand.Provider.Name(), cand.ModelCfg.ID, err)
				return videoExecutionResult{StatusCode: 502, ErrorType: "video_reencode_failed", ErrorMessage: err.Error()}
			}
			resp.OriginalVideoURL = resp.VideoURL
			resp.VideoURL = optimized.URL
			resp.OutputFormat = "webm"
			resp.Bytes = optimized.Bytes
			resp.OriginalBytes = optimized.OriginalBytes
		}
		resp.Model = originalModel
		var cost int64
		if h.billing != nil {
			cost, _ = h.billing.DeductVideoWithMediaInputs(ctx, userID, cand.ModelCfg.ID, videoDurationSeconds(string(req.Duration)), req.HasVideoInput(), req.InputImageCount(), req.Resolution, "")
		}
		if h.recorder != nil {
			h.recorder.RecordSuccessWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
				0, 1, int(latency.Milliseconds()), 0, cost, false, app.ID, app.URL, app.Title, app.Categories)
		}

		return videoExecutionResult{Response: resp, StatusCode: 200}
	}

	return videoExecutionResult{StatusCode: 502, ErrorType: "provider_error", ErrorMessage: "all providers failed for model " + originalModel}
}

func (h *VideoHandler) ensurePublicVideoURL(ctx context.Context, resp *model.VideoGenerationResponse, modelID string) {
	if resp == nil || h.store == nil || !strings.HasPrefix(resp.VideoURL, "data:video/") {
		return
	}
	contentType, data, ok := decodeVideoDataURL(resp.VideoURL)
	if !ok {
		return
	}
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if !strings.HasPrefix(strings.ToLower(contentType), "video/") {
		contentType = "video/mp4"
	}
	ext := videoExtFromContentType(contentType)
	safeModel := strings.NewReplacer("/", "-", " ", "-").Replace(modelID)
	url, err := h.store.Upload(ctx, fmt.Sprintf("%s%s", safeModel, ext), contentType, bytes.NewReader(data))
	if err != nil {
		log.Printf("video upload inline data: %v", err)
		return
	}
	resp.VideoURL = url
	resp.Bytes = int64(len(data))
	resp.OutputFormat = strings.TrimPrefix(ext, ".")
}

func decodeVideoDataURL(raw string) (string, []byte, bool) {
	header, encoded, ok := strings.Cut(raw, ",")
	if !ok || !strings.HasPrefix(header, "data:") || !strings.Contains(header, ";base64") {
		return "", nil, false
	}
	contentType := strings.TrimPrefix(strings.Split(header, ";")[0], "data:")
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, false
	}
	return contentType, data, true
}

func videoExtFromContentType(contentType string) string {
	switch {
	case strings.Contains(contentType, "webm"):
		return ".webm"
	case strings.Contains(contentType, "quicktime"), strings.Contains(contentType, "mov"):
		return ".mov"
	case strings.Contains(contentType, "m4v"):
		return ".m4v"
	default:
		return ".mp4"
	}
}

func wantsVideoWebM(format string) bool {
	format = strings.ToLower(strings.TrimSpace(format))
	return format == "webm" || format == "video/webm"
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
