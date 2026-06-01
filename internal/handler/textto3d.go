package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
)

// TextTo3DHandler runs the text/image -> image -> GLB pipeline by composing the
// existing image and 3D handlers. Each leg bills independently through the
// shared billing engine, so the user is charged the true sum (image price +
// Pixal3D price).
type TextTo3DHandler struct {
	image   *ImageHandler
	model3d *Model3DHandler
	billing *billing.Engine
	jobs    *textTo3DJobCache
}

func NewTextTo3DHandler(image *ImageHandler, model3d *Model3DHandler, b *billing.Engine) *TextTo3DHandler {
	return &TextTo3DHandler{image: image, model3d: model3d, billing: b, jobs: newTextTo3DJobCache()}
}

func (h *TextTo3DHandler) HandleTextTo3DGeneration(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}

	req, err := parseTextTo3DRequest(ctx.PostBody())
	if err != nil {
		writeError(ctx, 400, "invalid_request", err.Error())
		return
	}
	actualCost := h.estimateCost(req)

	async := req.Async || string(ctx.QueryArgs().Peek("async")) == "true" || string(ctx.Request.Header.Peek("Prefer")) == "respond-async"

	job, cached := h.jobs.getOrCreate(req)
	if !cached {
		if err := h.precheck(ctx, userID, actualCost); err != nil {
			h.jobs.complete(job.ID, textTo3DExecutionResult{StatusCode: 402, ErrorType: "billing_error", ErrorMessage: "Insufficient credits. Please add credits to continue."})
			writeError(ctx, 402, "billing_error", "Insufficient credits. Please add credits to continue.")
			return
		}
		go h.runTextTo3DJob(job.ID, req, userID, apiKeyID, requestAppAttribution(ctx))
	}
	if async {
		writeJSON(ctx, 202, textTo3DJobPayload(job, cached))
		return
	}
	if job.Status == model3DJobCompleted && job.Result != nil {
		writeJSON(ctx, 200, job.Result)
		return
	}
	if job.Status == model3DJobFailed {
		writeError(ctx, textTo3DJobHTTPStatus(job), job.ErrorType, job.Error)
		return
	}

	waited, ok := h.jobs.wait(job.ID, 85*time.Second)
	if !ok {
		writeError(ctx, 404, "job_not_found", "text-to-3D job not found")
		return
	}
	switch waited.Status {
	case model3DJobCompleted:
		writeJSON(ctx, 200, waited.Result)
	case model3DJobFailed:
		writeError(ctx, textTo3DJobHTTPStatus(waited), waited.ErrorType, waited.Error)
	default:
		ctx.Response.Header.Set("Retry-After", "2")
		writeJSON(ctx, 202, textTo3DJobPayload(waited, cached))
	}
}

func (h *TextTo3DHandler) HandleTextTo3DGenerationJob(ctx *fasthttp.RequestCtx) {
	jobID, _ := ctx.UserValue("job_id").(string)
	if jobID == "" {
		writeError(ctx, 400, "invalid_request", "job_id is required")
		return
	}
	job, ok := h.jobs.get(jobID)
	if !ok {
		writeError(ctx, 404, "not_found", "text-to-3D job not found")
		return
	}
	writeJSON(ctx, 200, textTo3DJobPayload(job, true))
}

func parseTextTo3DRequest(body []byte) (model.TextTo3DGenerationRequest, error) {
	var req model.TextTo3DGenerationRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return req, fmt.Errorf("Invalid JSON: %w", err)
	}
	if req.Model == "" {
		req.Model = "pixal3d-image-to-3d"
	}
	if req.ImageModel == "" {
		req.ImageModel = "openpaths/auto-image"
	}
	if req.Prompt == "" && req.ImageURL == "" {
		return req, fmt.Errorf("prompt is required (or pass image_url)")
	}
	if req.ImageURL != "" && !isPublicHTTPURL(req.ImageURL) {
		return req, fmt.Errorf("image_url must be a public http or https URL")
	}
	req.TextureSize = normalize3DTextureSize(req.TextureSize)
	if req.Resolution <= 0 {
		req.Resolution = 1024
	}
	return req, nil
}

func (h *TextTo3DHandler) estimateCost(req model.TextTo3DGenerationRequest) int64 {
	cost := pixal3DRequestCost(req.TextureSize)
	if req.ImageURL == "" && h.billing != nil {
		if imgCost, err := h.billing.ImageCost(req.ImageModel, 1, 0, req.Size); err == nil {
			cost += imgCost
		}
	}
	return cost
}

func (h *TextTo3DHandler) precheck(ctx context.Context, userID string, cost int64) error {
	if h.billing == nil {
		return nil
	}
	return h.billing.PreCheckFixed(ctx, userID, cost)
}

func (h *TextTo3DHandler) runTextTo3DJob(jobID string, req model.TextTo3DGenerationRequest, userID, apiKeyID string, app requestApp) {
	h.jobs.markRunning(jobID)
	bg, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	h.jobs.complete(jobID, h.executeTextTo3DGeneration(bg, req, userID, apiKeyID, app))
}

func (h *TextTo3DHandler) executeTextTo3DGeneration(ctx context.Context, req model.TextTo3DGenerationRequest, userID, apiKeyID string, app requestApp) textTo3DExecutionResult {
	textureSize := normalize3DTextureSize(req.TextureSize)
	resolution := req.Resolution
	if resolution <= 0 {
		resolution = 1024
	}

	imageURL := req.ImageURL
	imageModel := ""
	var imageCost int64

	// Step 1: text -> image (skipped when an image_url is supplied).
	if imageURL == "" {
		imgReq := &model.ImageGenerationRequest{
			Model:  req.ImageModel,
			Prompt: req.Prompt,
			Size:   req.Size,
			N:      1,
			Seed:   req.Seed,
		}
		imgRes := h.image.executeImageGeneration(ctx, imgReq, userID, apiKeyID, app)
		if imgRes.StatusCode >= 400 || imgRes.Response == nil {
			return textTo3DExecutionResult{StatusCode: imgRes.StatusCode, ErrorType: imgRes.ErrorType, ErrorMessage: imgRes.ErrorMessage}
		}
		imageModel = imgRes.ModelID
		imageCost = imgRes.Cost
		imageURL = h.image.ensurePublicImageURL(ctx, imgRes.Response)
		if !isPublicHTTPURL(imageURL) {
			return textTo3DExecutionResult{StatusCode: 502, ErrorType: "server_error", ErrorMessage: "generated image is not reachable at a public URL; configure object storage or pass image_url"}
		}
	}

	// Step 2: image -> GLB. This deducts the 3D charge on success.
	m3dReq := model.Model3DGenerationRequest{
		Model:       req.Model,
		ImageURL:    imageURL,
		Resolution:  resolution,
		TextureSize: textureSize,
		Remesh:      req.Remesh,
		Seed:        req.Seed,
	}
	m3dRes := h.model3d.executeModel3DGeneration(ctx, nil, m3dReq, userID, apiKeyID, app)
	if m3dRes.StatusCode >= 400 || m3dRes.Response == nil {
		return textTo3DExecutionResult{StatusCode: m3dRes.StatusCode, ErrorType: m3dRes.ErrorType, ErrorMessage: m3dRes.ErrorMessage}
	}

	imageCostUSD := float64(imageCost) / 10000
	model3DCostUSD := float64(pixal3DExternalCostCents(textureSize)) / 100
	totalUSD := imageCostUSD + model3DCostUSD

	resp := &model.TextTo3DGenerationResponse{
		ModelGLB:    m3dRes.Response.ModelGLB,
		Prompt:      req.Prompt,
		Seed:        m3dRes.Response.Seed,
		BackendUsed: m3dRes.Response.BackendUsed,
		Model:       "text-to-3d",
		Image: &model.TextTo3DImageInfo{
			URL:   imageURL,
			Model: imageModel,
		},
		CreditsCharged: totalUSD,
		Billing: &model.TextTo3DBilling{
			TextureSize:    textureSize,
			ImageModel:     imageModel,
			ImageCostUSD:   imageCostUSD,
			Model3DCostUSD: model3DCostUSD,
			TotalCostUSD:   totalUSD,
		},
	}
	return textTo3DExecutionResult{Response: resp, StatusCode: 200}
}
