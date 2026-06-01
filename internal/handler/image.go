package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
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

// imageExecutionResult is the outcome of running the image-generation candidate
// loop. It lets the HTTP handler and internal pipelines (e.g. text-to-3D) share
// the same generation, billing, and metrics path.
type imageExecutionResult struct {
	Response     *model.ImageGenerationResponse
	ModelID      string // resolved config model id that produced the image
	ProviderName string
	Cost         int64
	StatusCode   int
	ErrorType    string
	ErrorMessage string
}

func (h *ImageHandler) HandleImageGeneration(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}
	app := requestAppAttribution(ctx)

	var req model.ImageGenerationRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON: "+err.Error())
		return
	}
	if req.Model == "" {
		writeError(ctx, 400, "invalid_request", "model is required")
		return
	}
	if req.Prompt == "" && !isPromptlessImageModel(req.Model) {
		writeError(ctx, 400, "invalid_request", "prompt is required")
		return
	}
	if isPromptlessImageModel(req.Model) && !hasImageInput(req) {
		writeError(ctx, 400, "invalid_request", "image_url is required")
		return
	}

	result := h.executeImageGeneration(ctx, &req, userID, apiKeyID, app)
	if result.StatusCode >= 400 || result.Response == nil {
		writeError(ctx, result.StatusCode, result.ErrorType, result.ErrorMessage)
		return
	}
	writeJSON(ctx, 200, result.Response)
}

// executeImageGeneration runs auto-routing, the provider candidate loop, image
// resizing/rehosting, billing, and metrics. It mutates req.Model/req.Size in
// place (matching the original handler behaviour) and never writes to ctx, so
// it can be reused by internal pipelines.
func (h *ImageHandler) executeImageGeneration(ctx context.Context, req *model.ImageGenerationRequest, userID, apiKeyID string, app requestApp) imageExecutionResult {
	if req.N <= 0 && req.NumImages > 0 {
		req.N = req.NumImages
	}
	if req.N <= 0 {
		req.N = 1
	}

	originalModel := req.Model
	autoResult := h.router.MaybeResolveAuto(ctx, req.Model, "image", req.Prompt)
	candidates, err := h.router.ResolveForRequest(originalModel, autoResult.ModelID)
	if err != nil {
		return imageExecutionResult{StatusCode: 404, ErrorType: "model_not_found", ErrorMessage: err.Error()}
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

		resp, err := imgProv.GenerateImage(ctx, req)
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
					return imageExecutionResult{StatusCode: statusCode, ErrorType: "provider_error", ErrorMessage: errMsg}
				}
			}
			h.router.MarkModelUnhealthy(cand.Provider.Name(), cand.ModelCfg.ID)
			h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
				int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
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
		inputImageCount := countInputImages(req)
		cost, _ := h.deductImageCost(ctx, userID, cand.ModelCfg.ID, req, resp, imageCount, inputImageCount)
		h.recorder.RecordSuccessWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
			0, imageCount, int(latency.Milliseconds()), 0, cost, false, app.ID, app.URL, app.Title, app.Categories)

		return imageExecutionResult{
			Response:     resp,
			ModelID:      cand.ModelCfg.ID,
			ProviderName: cand.Provider.Name(),
			Cost:         cost,
		}
	}

	return imageExecutionResult{StatusCode: 502, ErrorType: "provider_error", ErrorMessage: "all providers failed for model " + originalModel}
}

// ensurePublicImageURL returns a public http(s) URL for the first image in the
// response, uploading inline base64 bytes to storage when the provider only
// returned data. Returns "" when no usable image is present. Used by the
// text-to-3D pipeline, where the downstream 3D provider must fetch the image.
func (h *ImageHandler) ensurePublicImageURL(ctx context.Context, resp *model.ImageGenerationResponse) string {
	if resp == nil || len(resp.Data) == 0 {
		return ""
	}
	d := resp.Data[0]
	if isPublicHTTPURL(d.URL) {
		return d.URL
	}
	if d.B64JSON != "" && h.store != nil {
		data, err := base64.StdEncoding.DecodeString(d.B64JSON)
		if err != nil {
			log.Printf("text-to-3d: decode b64 image: %v", err)
			return d.URL
		}
		contentType := http.DetectContentType(data)
		ext := extFromContentType(contentType)
		if ext == "" {
			ext = ".png"
		}
		url, err := h.store.Upload(ctx, fmt.Sprintf("text-to-3d-input%s", ext), contentType, bytes.NewReader(data))
		if err != nil {
			log.Printf("text-to-3d: upload b64 image: %v", err)
			return d.URL
		}
		return url
	}
	return d.URL
}

func (h *ImageHandler) deductImageCost(ctx context.Context, userID, modelID string, req *model.ImageGenerationRequest, resp *model.ImageGenerationResponse, imageCount, inputImageCount int) (int64, error) {
	if isOutpaintModel(modelID) && len(resp.Data) > 0 {
		inputW, inputH := h.firstInputImageDimensions(ctx, req)
		outputW, outputH := resp.Data[0].Width, resp.Data[0].Height
		return h.billing.DeductOutpaint(ctx, userID, modelID, inputW, inputH, outputW, outputH, imageCount, "")
	}
	return h.billing.DeductImageWithInputsAndSize(ctx, userID, modelID, imageCount, inputImageCount, req.Size, "")
}

func isPromptlessImageModel(modelID string) bool {
	m := strings.ToLower(modelID)
	return strings.Contains(m, "outpaint") || strings.Contains(m, "extend")
}

func hasImageInput(req model.ImageGenerationRequest) bool {
	if req.ImageURL != "" || len(req.ImageURLs) > 0 || len(req.ReferenceImageURLs) > 0 {
		return true
	}
	if req.Image != nil && req.Image.URL != "" {
		return true
	}
	if len(req.Images) > 0 && req.Images[0].URL != "" {
		return true
	}
	return false
}

func isOutpaintModel(modelID string) bool {
	return strings.Contains(strings.ToLower(modelID), "flux-2-pro/outpaint")
}

func (h *ImageHandler) firstInputImageDimensions(ctx context.Context, req *model.ImageGenerationRequest) (int, int) {
	for _, url := range inputImageURLs(req) {
		if width, height := downloadImageDimensions(ctx, url); width > 0 && height > 0 {
			return width, height
		}
	}
	return 0, 0
}

func inputImageURLs(req *model.ImageGenerationRequest) []string {
	var urls []string
	if req.ImageURL != "" {
		urls = append(urls, req.ImageURL)
	}
	urls = append(urls, req.ReferenceImageURLs...)
	urls = append(urls, req.ImageURLs...)
	if req.Image != nil && req.Image.URL != "" {
		urls = append(urls, req.Image.URL)
	}
	for _, img := range req.Images {
		if img.URL != "" {
			urls = append(urls, img.URL)
		}
	}
	return urls
}

func downloadImageDimensions(ctx context.Context, sourceURL string) (int, int) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return 0, 0
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, 0
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, 0
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0
	}
	return imageDimensions(body)
}

func (h *ImageHandler) rehostImageURLs(ctx context.Context, resp *model.ImageGenerationResponse) {
	if h.store == nil || resp == nil {
		return
	}
	for i := range resp.Data {
		if resp.Data[i].URL == "" {
			continue
		}
		url, width, height, err := h.rehostImageURL(ctx, resp.Data[i].URL, i)
		if err != nil {
			log.Printf("image rehost [%d]: %v, keeping upstream URL", i, err)
			continue
		}
		resp.Data[i].URL = url
		if resp.Data[i].Width == 0 {
			resp.Data[i].Width = width
		}
		if resp.Data[i].Height == 0 {
			resp.Data[i].Height = height
		}
	}
}

func (h *ImageHandler) rehostImageURL(ctx context.Context, sourceURL string, index int) (string, int, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return "", 0, 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", 0, 0, fmt.Errorf("download status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, 0, err
	}
	width, height := imageDimensions(body)
	contentType := resp.Header.Get("Content-Type")
	ext := extFromContentType(contentType)
	if ext == "" {
		ext = filepath.Ext(strings.Split(sourceURL, "?")[0])
	}
	if ext == "" {
		ext = ".png"
	}
	url, err := h.store.Upload(ctx, fmt.Sprintf("generated-%d%s", index+1, ext), contentType, bytes.NewReader(body))
	return url, width, height, err
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

func imageDimensions(body []byte) (int, int) {
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(body)); err == nil {
		return cfg.Width, cfg.Height
	}
	return webpDimensions(body)
}

func webpDimensions(body []byte) (int, int) {
	if len(body) < 30 || string(body[:4]) != "RIFF" || string(body[8:12]) != "WEBP" {
		return 0, 0
	}
	for offset := 12; offset+8 <= len(body); {
		chunkType := string(body[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(body[offset+4 : offset+8]))
		data := offset + 8
		if data+chunkSize > len(body) {
			return 0, 0
		}
		switch chunkType {
		case "VP8X":
			if chunkSize >= 10 {
				w := 1 + int(body[data+4]) + int(body[data+5])<<8 + int(body[data+6])<<16
				h := 1 + int(body[data+7]) + int(body[data+8])<<8 + int(body[data+9])<<16
				return w, h
			}
		case "VP8 ":
			if chunkSize >= 10 {
				w := int(binary.LittleEndian.Uint16(body[data+6:data+8])) & 0x3fff
				h := int(binary.LittleEndian.Uint16(body[data+8:data+10])) & 0x3fff
				return w, h
			}
		case "VP8L":
			if chunkSize >= 5 {
				b0, b1, b2, b3 := uint32(body[data+1]), uint32(body[data+2]), uint32(body[data+3]), uint32(body[data+4])
				w := int(1 + (((b1 & 0x3f) << 8) | b0))
				h := int(1 + (((b3 & 0x0f) << 10) | (b2 << 2) | ((b1 & 0xc0) >> 6)))
				return w, h
			}
		}
		offset = data + chunkSize + chunkSize%2
	}
	return 0, 0
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
