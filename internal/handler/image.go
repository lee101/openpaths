package handler

import (
	"encoding/json"
	"log"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	imgutil "github.com/openpaths/openpaths/internal/image"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
)

type ImageHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
}

func NewImageHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder) *ImageHandler {
	return &ImageHandler{router: r, billing: b, recorder: rec}
}

func (h *ImageHandler) HandleImageGeneration(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}

	var req model.ImageGenerationRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON: "+err.Error())
		return
	}
	if req.Model == "" {
		writeError(ctx, 400, "invalid_request", "model is required")
		return
	}
	if req.Prompt == "" {
		writeError(ctx, 400, "invalid_request", "prompt is required")
		return
	}
	if req.N <= 0 {
		req.N = 1
	}

	originalModel := req.Model
	autoResult := h.router.MaybeResolveAuto(ctx, req.Model, "image", req.Prompt)
	candidates, err := h.router.ResolveForRequest(originalModel, autoResult.ModelID)
	if err != nil {
		writeError(ctx, 404, "model_not_found", err.Error())
		return
	}

	for i, cand := range candidates {
		imgProv, ok := cand.Provider.(provider.ImageProvider)
		if !ok {
			log.Printf("image: %s does not support image generation", cand.Provider.Name())
			continue
		}

		var requestedSize imgutil.Size
		needsResize := false

		if req.Size != "" && len(cand.ModelCfg.SupportedSizes) > 0 {
			if rs, ok := imgutil.ParseSize(req.Size); ok {
				supported := imgutil.ParseSizes(cand.ModelCfg.SupportedSizes)
				matched := imgutil.MatchSize(rs, supported)
				if imgutil.NeedsResize(rs, matched) {
					requestedSize = rs
					needsResize = true
					log.Printf("image size snap: %s -> %s", req.Size, imgutil.FormatSize(matched))
				}
				req.Size = imgutil.FormatSize(matched)
			}
		}

		req.Model = cand.ModelCfg.ProviderModelID
		start := time.Now()

		resp, err := imgProv.GenerateImage(ctx, &req)
		latency := time.Since(start)

		if err != nil {
			statusCode := 502
			errMsg := "upstream error"
			if pe, ok := err.(*provider.ProviderError); ok {
				statusCode = pe.StatusCode
				errMsg = pe.Message
				if !pe.Retryable && i == len(candidates)-1 {
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
				log.Printf("image fallback: %s/%s -> %s/%s",
					cand.Provider.Name(), cand.ModelCfg.ID,
					candidates[i+1].Provider.Name(), candidates[i+1].ModelCfg.ID)
			}
			continue
		}

		if needsResize && len(resp.Data) > 0 {
			for j := range resp.Data {
				if resp.Data[j].B64JSON != "" {
					resized, err := imgutil.ProcessB64(resp.Data[j].B64JSON, requestedSize)
					if err != nil {
						log.Printf("image resize b64 [%d]: %v", j, err)
						continue
					}
					resp.Data[j].B64JSON = resized
				} else if resp.Data[j].URL != "" {
					resized, err := imgutil.ProcessImageURL(resp.Data[j].URL, requestedSize)
					if err != nil {
						log.Printf("image resize url [%d]: %v, keeping original", j, err)
						continue
					}
					resp.Data[j].B64JSON = resized
					resp.Data[j].URL = ""
				}
			}
		}

		h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
		imageCount := len(resp.Data)
		cost, _ := h.billing.DeductImage(ctx, userID, cand.ModelCfg.ID, imageCount, "")
		h.recorder.RecordSuccess(userID, apiKeyID, originalModel, cand.Provider.Name(),
			0, imageCount, int(latency.Milliseconds()), 0, cost, false)

		writeJSON(ctx, 200, resp)
		return
	}

	writeError(ctx, 502, "provider_error", "all providers failed for model "+originalModel)
}
