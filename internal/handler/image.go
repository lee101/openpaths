package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	imgutil "github.com/openpaths/openpaths/internal/image"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
	"github.com/openpaths/openpaths/internal/storage"
)

type ImageHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
	store    storage.Store
}

func NewImageHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder) *ImageHandler {
	return &ImageHandler{router: r, billing: b, recorder: rec}
}

func (h *ImageHandler) SetStorage(store storage.Store) {
	h.store = store
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
		h.rehostImageURLs(ctx, resp)

		h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
		imageCount := len(resp.Data)
		inputImageCount := countInputImages(&req)
		cost, _ := h.billing.DeductImageWithInputsAndSize(ctx, userID, cand.ModelCfg.ID, imageCount, inputImageCount, req.Size, "")
		h.recorder.RecordSuccess(userID, apiKeyID, originalModel, cand.Provider.Name(),
			0, imageCount, int(latency.Milliseconds()), 0, cost, false)

		writeJSON(ctx, 200, resp)
		return
	}

	writeError(ctx, 502, "provider_error", "all providers failed for model "+originalModel)
}

func (h *ImageHandler) rehostImageURLs(ctx context.Context, resp *model.ImageGenerationResponse) {
	if h.store == nil || resp == nil {
		return
	}
	for i := range resp.Data {
		if resp.Data[i].URL == "" {
			continue
		}
		url, err := h.rehostImageURL(ctx, resp.Data[i].URL, i)
		if err != nil {
			log.Printf("image rehost [%d]: %v, keeping upstream URL", i, err)
			continue
		}
		resp.Data[i].URL = url
	}
}

func (h *ImageHandler) rehostImageURL(ctx context.Context, sourceURL string, index int) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	contentType := resp.Header.Get("Content-Type")
	ext := extFromContentType(contentType)
	if ext == "" {
		ext = filepath.Ext(strings.Split(sourceURL, "?")[0])
	}
	if ext == "" {
		ext = ".png"
	}
	return h.store.Upload(ctx, fmt.Sprintf("generated-%d%s", index+1, ext), contentType, bytes.NewReader(body))
}

func extFromContentType(contentType string) string {
	if idx := strings.Index(contentType, ";"); idx != -1 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}

func countInputImages(req *model.ImageGenerationRequest) int {
	count := len(req.ReferenceImageURLs) + len(req.Images) + len(req.ImageURLs)
	if req.Image != nil {
		count++
	}
	if req.ImageURL != "" {
		count++
	}
	return count
}
