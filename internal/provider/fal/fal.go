package fal

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	imgutil "github.com/openpaths/openpaths/internal/image"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

// outpaintMaxEdge bounds the longest input edge sent to fal outpaint models.
// Larger inputs are downscaled aspect-preserving to control cost and latency.
const outpaintMaxEdge = 2048

type FalProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func New(apiKey string) *FalProvider {
	return &FalProvider{
		apiKey:  apiKey,
		baseURL: "https://fal.run",
		client:  &http.Client{Timeout: 2 * time.Minute},
	}
}

func (p *FalProvider) Name() string { return "fal" }

func (p *FalProvider) HealthCheck(ctx context.Context) error { return nil }

func (p *FalProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	return nil, &provider.ProviderError{Provider: "fal", StatusCode: 400, Message: "fal does not support chat", Retryable: false}
}

func (p *FalProvider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	return nil, &provider.ProviderError{Provider: "fal", StatusCode: 400, Message: "fal does not support chat", Retryable: false}
}

type falRequest struct {
	AudioURL   string `json:"audio_url"`
	Task       string `json:"task"`
	ChunkLevel string `json:"chunk_level"`
}

type falResponse struct {
	Text   string     `json:"text"`
	Chunks []falChunk `json:"chunks,omitempty"`
}

type falChunk struct {
	Text      string    `json:"text"`
	Timestamp []float64 `json:"timestamp"`
}

func mimeType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".m4a":
		return "audio/mp4"
	case ".webm":
		return "audio/webm"
	case ".flac":
		return "audio/flac"
	default:
		return "audio/mpeg"
	}
}

func (p *FalProvider) GenerateImage(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	if strings.Contains(strings.ToLower(req.Model), "smart-resize") {
		return p.smartResize(ctx, req)
	}

	falReq := falImageRequest(req)
	body, err := json.Marshal(falReq)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	if falImageUsesQueue(req.Model) {
		resultBody, err := p.submitFalImageQueue(ctx, req.Model, body)
		if err != nil {
			return nil, err
		}
		return parseFalImageResult(resultBody)
	}

	endpoint := strings.TrimRight(p.baseURL, "/") + "/" + req.Model
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Key "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "fal", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "fal",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	return parseFalImageResult(respBody)
}

func falImageRequest(req *model.ImageGenerationRequest) map[string]any {
	if falImageIsOutpaint(req.Model) {
		falReq := map[string]any{}
		if req.ImageURL != "" {
			falReq["image_url"] = req.ImageURL
		} else if len(req.ImageURLs) > 0 {
			falReq["image_url"] = req.ImageURLs[0]
		} else if req.Image != nil && req.Image.URL != "" {
			falReq["image_url"] = req.Image.URL
		} else if len(req.Images) > 0 && req.Images[0].URL != "" {
			falReq["image_url"] = req.Images[0].URL
		} else if len(req.ReferenceImageURLs) > 0 {
			falReq["image_url"] = req.ReferenceImageURLs[0]
		}
		if req.ExpandTop != nil {
			falReq["expand_top"] = *req.ExpandTop
		}
		if req.ExpandBottom != nil {
			falReq["expand_bottom"] = *req.ExpandBottom
		}
		if req.ExpandLeft != nil {
			falReq["expand_left"] = *req.ExpandLeft
		}
		if req.ExpandRight != nil {
			falReq["expand_right"] = *req.ExpandRight
		}
		if req.ZoomOutPercentage != nil {
			falReq["zoom_out_percentage"] = *req.ZoomOutPercentage
		}
		if strings.TrimSpace(req.Prompt) != "" {
			falReq["prompt"] = req.Prompt
		}
		if n := req.N; n > 0 {
			falReq["num_images"] = n
		} else if req.NumImages > 0 {
			falReq["num_images"] = req.NumImages
		}
		if req.Seed != nil {
			falReq["seed"] = *req.Seed
		}
		if req.AutoCrop != nil {
			falReq["auto_crop"] = *req.AutoCrop
		}
		if req.EnableSafetyChecker != nil {
			falReq["enable_safety_checker"] = *req.EnableSafetyChecker
		} else {
			falReq["enable_safety_checker"] = true
		}
		if req.OutputFormat != "" {
			falReq["output_format"] = strings.ToLower(strings.TrimSpace(req.OutputFormat))
		}
		downscaleOutpaintInput(falReq)
		return falReq
	}

	falReq := map[string]any{
		"prompt": req.Prompt,
	}
	if req.Quality != "" {
		falReq["quality"] = req.Quality
	}
	if req.ImageSize != nil {
		falReq["image_size"] = req.ImageSize
	} else if req.Size != "" {
		parts := strings.SplitN(req.Size, "x", 2)
		if len(parts) == 2 {
			var w, h int
			fmt.Sscanf(parts[0], "%d", &w)
			fmt.Sscanf(parts[1], "%d", &h)
			if w > 0 && h > 0 {
				falReq["image_size"] = map[string]int{"width": w, "height": h}
			}
		}
	}
	if req.N > 0 {
		falReq["num_images"] = req.N
	}
	if req.NumInferenceSteps > 0 {
		falReq["num_inference_steps"] = req.NumInferenceSteps
	}
	if req.GuidanceScale != nil {
		falReq["guidance_scale"] = *req.GuidanceScale
	}
	if req.Seed != nil {
		falReq["seed"] = *req.Seed
	}
	if req.KeepOriginalAspect != nil {
		falReq["keep_original_aspect"] = *req.KeepOriginalAspect
	}
	if refs := falReferenceImageURLs(req); len(refs) > 0 {
		if falImageUsesQueue(req.Model) && strings.Contains(req.Model, "hidream-o1-image") {
			falReq["reference_image_urls"] = refs
		} else {
			falReq["image_urls"] = refs
		}
	}
	if req.EnableSafetyChecker != nil {
		falReq["enable_safety_checker"] = *req.EnableSafetyChecker
	} else if strings.Contains(strings.ToLower(req.Model), "hidream-o1-image") {
		falReq["enable_safety_checker"] = false
	}
	if outputFormat, syncMode := falOutputMode(req); outputFormat != "" {
		falReq["output_format"] = outputFormat
		if syncMode {
			falReq["sync_mode"] = true
		}
	}
	return falReq
}

func falReferenceImageURLs(req *model.ImageGenerationRequest) []string {
	var refs []string
	refs = append(refs, req.ReferenceImageURLs...)
	if req.ImageURL != "" {
		refs = append(refs, req.ImageURL)
	}
	refs = append(refs, req.ImageURLs...)
	if req.Image != nil && req.Image.URL != "" {
		refs = append(refs, req.Image.URL)
	}
	for _, img := range req.Images {
		if img.URL != "" {
			refs = append(refs, img.URL)
		}
	}
	return refs
}

func falImageUsesQueue(modelID string) bool {
	modelName := strings.ToLower(modelID)
	return strings.Contains(modelName, "hidream-o1-image") || falImageIsOutpaint(modelName)
}

func falImageIsOutpaint(modelID string) bool {
	return strings.Contains(strings.ToLower(modelID), "outpaint")
}

// downscaleOutpaintInput shrinks the source image to outpaintMaxEdge on its
// longest side (aspect-preserving) before outpainting, replacing image_url
// with a data URI. Best-effort: on any failure the original URL is kept.
func downscaleOutpaintInput(falReq map[string]any) {
	src, _ := falReq["image_url"].(string)
	if src == "" {
		return
	}
	var data []byte
	switch {
	case strings.HasPrefix(src, "data:"):
		b64, err := dataURIToBase64(src)
		if err != nil {
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return
		}
		data = decoded
	case strings.HasPrefix(src, "http://"), strings.HasPrefix(src, "https://"):
		fetched, err := imgutil.FetchImage(src)
		if err != nil {
			return
		}
		data = fetched
	default:
		return
	}
	out, format, changed, err := imgutil.DownscaleToMaxEdge(data, outpaintMaxEdge)
	if err != nil || !changed {
		return
	}
	mime := "image/jpeg"
	if format == "png" {
		mime = "image/png"
	}
	falReq["image_url"] = "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(out)
}

func (p *FalProvider) Generate3D(ctx context.Context, req *model.Model3DGenerationRequest) (*model.Model3DGenerationResponse, error) {
	falReq := fal3DRequest(req)
	body, err := json.Marshal(falReq)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	resultBody, err := p.submitFalImageQueue(ctx, req.Model, body)
	if err != nil {
		return nil, err
	}
	return parseFal3DResult(resultBody, normalizePixalTextureSize(req.TextureSize))
}

func fal3DRequest(req *model.Model3DGenerationRequest) map[string]any {
	textureSize := normalizePixalTextureSize(req.TextureSize)
	falReq := map[string]any{
		"image_url":                    req.ImageURL,
		"resolution":                   normalizePixalResolution(req.Resolution),
		"ss_guidance_strength":         floatOrDefault(req.SSGuidanceStrength, 7.5),
		"ss_guidance_rescale":          floatOrDefault(req.SSGuidanceRescale, 0.7),
		"ss_sampling_steps":            intOrDefault(req.SSSamplingSteps, 12),
		"ss_rescale_t":                 floatOrDefault(req.SSRescaleT, 5),
		"shape_slat_guidance_strength": floatOrDefault(req.ShapeSlatGuidanceStrength, 7.5),
		"shape_slat_guidance_rescale":  floatOrDefault(req.ShapeSlatGuidanceRescale, 0.5),
		"shape_slat_sampling_steps":    intOrDefault(req.ShapeSlatSamplingSteps, 12),
		"shape_slat_rescale_t":         floatOrDefault(req.ShapeSlatRescaleT, 3),
		"tex_slat_guidance_strength":   floatOrDefault(req.TexSlatGuidanceStrength, 1),
		"tex_slat_sampling_steps":      intOrDefault(req.TexSlatSamplingSteps, 12),
		"tex_slat_rescale_t":           floatOrDefault(req.TexSlatRescaleT, 3),
		"mesh_scale":                   floatOrDefault(req.MeshScale, 1),
		"max_num_tokens":               intOrDefault(req.MaxNumTokens, 49152),
		"decimation_target":            intOrDefault(req.DecimationTarget, 200000),
		"texture_size":                 textureSize,
		"remesh":                       boolOrDefault(req.Remesh, true),
	}
	if req.Seed != nil {
		falReq["seed"] = *req.Seed
	}
	if req.TexSlatGuidanceRescale != nil && *req.TexSlatGuidanceRescale > 0 {
		falReq["tex_slat_guidance_rescale"] = *req.TexSlatGuidanceRescale
	}
	return falReq
}

func normalizePixalResolution(value int) int {
	if value == 1536 {
		return 1536
	}
	return 1024
}

func normalizePixalTextureSize(value int) int {
	switch value {
	case 2048, 4096:
		return value
	default:
		return 1024
	}
}

func pixalExternalCostCents(textureSize int) int {
	if normalizePixalTextureSize(textureSize) == 1024 {
		return 30
	}
	return 42
}

func floatOrDefault(value *float64, fallback float64) float64 {
	if value != nil && *value > 0 {
		return *value
	}
	return fallback
}

func intOrDefault(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func intOrDefaultString(value string, fallback int) int {
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err == nil && parsed > 0 {
		return parsed
	}
	return fallback
}

func boolOrDefault(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func parseFal3DResult(body []byte, textureSize int) (*model.Model3DGenerationResponse, error) {
	var raw struct {
		ModelGLB *model.FileAsset `json:"model_glb"`
		Seed     *int             `json:"seed,omitempty"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}
	if raw.ModelGLB == nil || strings.TrimSpace(raw.ModelGLB.URL) == "" {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: 502, Message: "no model_glb url in response: " + string(body), Retryable: false}
	}
	cents := pixalExternalCostCents(textureSize)
	return &model.Model3DGenerationResponse{
		ModelGLB: *raw.ModelGLB,
		Seed:     raw.Seed,
		Billing: &model.Model3DBilling{
			TextureSize:       normalizePixalTextureSize(textureSize),
			ExternalCostUSD:   float64(cents) / 100,
			ExternalCostCents: cents,
		},
	}, nil
}

func (p *FalProvider) submitFalImageQueue(ctx context.Context, modelID string, body []byte) ([]byte, error) {
	endpoint := strings.TrimRight(p.queueBaseURL(), "/") + "/" + strings.TrimLeft(modelID, "/")
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Key "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: resp.StatusCode, Message: string(respBody), Retryable: resp.StatusCode >= 500 || resp.StatusCode == 429}
	}

	var submit struct {
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(respBody, &submit); err != nil {
		return nil, fmt.Errorf("unmarshal submit: %w", err)
	}
	if submit.RequestID == "" {
		return respBody, nil
	}
	return p.waitFalQueueResult(ctx, modelID, submit.RequestID)
}

func parseFalImageResult(respBody []byte) (*model.ImageGenerationResponse, error) {
	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	images := falImageData(raw)
	if len(images) == 0 {
		return nil, &provider.ProviderError{
			Provider: "fal", StatusCode: 502, Message: "no images in response", Retryable: false,
		}
	}

	return &model.ImageGenerationResponse{
		Created: time.Now().Unix(),
		Data:    images,
	}, nil
}

func (p *FalProvider) smartResize(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	if req.ImageURL == "" {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: 400, Message: "image_url is required", Retryable: false}
	}
	if len(req.TargetSizes) == 0 {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: 400, Message: "target_sizes is required", Retryable: false}
	}

	payload := map[string]any{
		"image_url":    req.ImageURL,
		"target_sizes": req.TargetSizes,
		"prompt":       req.Prompt,
	}
	if req.NumImagesPerSize > 0 {
		payload["num_images_per_size"] = req.NumImagesPerSize
	}
	if req.Resolution != "" {
		payload["resolution"] = req.Resolution
	}
	if req.OutputFormat != "" {
		payload["output_format"] = req.OutputFormat
	}
	if req.SafetyTolerance != "" {
		payload["safety_tolerance"] = req.SafetyTolerance
	}
	if req.Seed != nil {
		payload["seed"] = *req.Seed
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	endpoint := "https://queue.fal.run/" + req.Model
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Key "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 300 {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: resp.StatusCode, Message: string(respBody), Retryable: resp.StatusCode >= 500 || resp.StatusCode == 429}
	}

	var handle struct {
		ResponseURL string `json:"response_url"`
		StatusURL   string `json:"status_url"`
	}
	if err := json.Unmarshal(respBody, &handle); err != nil {
		return nil, fmt.Errorf("unmarshal queue response: %w", err)
	}
	if handle.ResponseURL == "" || handle.StatusURL == "" {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: 502, Message: "missing fal queue urls", Retryable: true}
	}

	for range 90 {
		statusReq, err := http.NewRequestWithContext(ctx, "GET", handle.StatusURL, nil)
		if err != nil {
			return nil, err
		}
		statusReq.Header.Set("Authorization", "Key "+p.apiKey)
		statusResp, err := p.client.Do(statusReq)
		if err != nil {
			return nil, &provider.ProviderError{Provider: "fal", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
		}
		var status struct {
			Status string `json:"status"`
			Error  any    `json:"error"`
		}
		decodeErr := json.NewDecoder(statusResp.Body).Decode(&status)
		statusResp.Body.Close()
		if decodeErr != nil {
			return nil, decodeErr
		}
		if status.Status == "COMPLETED" {
			break
		}
		if status.Status == "FAILED" {
			return nil, &provider.ProviderError{Provider: "fal", StatusCode: 502, Message: fmt.Sprintf("smart resize failed: %v", status.Error), Retryable: false}
		}
		time.Sleep(2 * time.Second)
	}

	resultReq, err := http.NewRequestWithContext(ctx, "GET", handle.ResponseURL, nil)
	if err != nil {
		return nil, err
	}
	resultReq.Header.Set("Authorization", "Key "+p.apiKey)
	resultResp, err := p.client.Do(resultReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resultResp.Body.Close()
	resultBody, err := io.ReadAll(resultResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read result: %w", err)
	}
	if resultResp.StatusCode >= 300 {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: resultResp.StatusCode, Message: string(resultBody), Retryable: resultResp.StatusCode >= 500 || resultResp.StatusCode == 429}
	}
	var raw map[string]any
	if err := json.Unmarshal(resultBody, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}

	images := falImageData(raw)
	if len(images) == 0 {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: 502, Message: "no images in response", Retryable: false}
	}
	return &model.ImageGenerationResponse{Created: time.Now().Unix(), Data: images}, nil
}

func falImageData(raw map[string]any) []model.ImageData {
	var images []model.ImageData
	appendImages := func(items []any) {
		for _, img := range items {
			if m, ok := img.(map[string]any); ok {
				if url, ok := m["url"].(string); ok && url != "" {
					if strings.HasPrefix(url, "data:image/") {
						if b64, err := dataURIToBase64(url); err == nil {
							images = append(images, model.ImageData{B64JSON: b64})
							continue
						}
					}
					images = append(images, model.ImageData{URL: url, Width: intFromAny(m["width"]), Height: intFromAny(m["height"])})
				}
			}
		}
	}
	if items, ok := raw["images"].([]any); ok {
		appendImages(items)
	}
	if results, ok := raw["results"].([]any); ok {
		for _, item := range results {
			if m, ok := item.(map[string]any); ok {
				if items, ok := m["images"].([]any); ok {
					appendImages(items)
				}
			}
		}
	}
	return images
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func (p *FalProvider) GenerateVideo(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	falReq := map[string]any{
		"prompt": req.Prompt,
	}
	if req.Resolution != "" {
		falReq["resolution"] = req.Resolution
	}
	if req.FramesPerSecond > 0 {
		falReq["fps"] = req.FramesPerSecond
	}
	if req.Duration != "" {
		if strings.Contains(strings.ToLower(req.Model), "alibaba/happy-horse/") {
			falReq["duration"] = intOrDefaultString(string(req.Duration), 5)
		} else {
			falReq["duration"] = string(req.Duration)
		}
	}
	if req.AspectRatio != "" {
		falReq["aspect_ratio"] = req.AspectRatio
	}
	if req.GenerateAudio != nil {
		falReq["generate_audio"] = *req.GenerateAudio
	}
	if req.Seed != nil {
		falReq["seed"] = *req.Seed
	}
	if req.EndUserID != "" {
		falReq["end_user_id"] = req.EndUserID
	}
	if req.ImageURL != "" {
		falReq["image_url"] = req.ImageURL
	}
	if req.EndImageURL != "" {
		falReq["end_image_url"] = req.EndImageURL
	}
	if len(req.ImageURLs) > 0 {
		falReq["image_urls"] = req.ImageURLs
	}
	if len(req.VideoURLs) > 0 {
		falReq["video_urls"] = req.VideoURLs
	}
	if len(req.AudioURLs) > 0 {
		falReq["audio_urls"] = req.AudioURLs
	}
	if req.EnableSafetyChecker != nil {
		falReq["enable_safety_checker"] = *req.EnableSafetyChecker
	}

	body, err := json.Marshal(falReq)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	endpoint := strings.TrimRight(p.queueBaseURL(), "/") + "/" + strings.TrimLeft(req.Model, "/")
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Key "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: resp.StatusCode, Message: string(respBody), Retryable: resp.StatusCode >= 500 || resp.StatusCode == 429}
	}

	var submit struct {
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(respBody, &submit); err != nil {
		return nil, fmt.Errorf("unmarshal submit: %w", err)
	}
	if submit.RequestID == "" {
		return parseFalVideoResult(respBody)
	}

	resultBody, err := p.waitFalQueueResult(ctx, req.Model, submit.RequestID)
	if err != nil {
		return nil, err
	}
	return parseFalVideoResult(resultBody)
}

func (p *FalProvider) queueBaseURL() string {
	if strings.Contains(p.baseURL, "queue.fal.run") {
		return p.baseURL
	}
	if p.baseURL != "" && !strings.Contains(p.baseURL, "fal.run") {
		return p.baseURL
	}
	return "https://queue.fal.run"
}

func (p *FalProvider) waitFalQueueResult(ctx context.Context, modelID, requestID string) ([]byte, error) {
	bases := p.falQueueRequestBases(modelID, requestID)
	baseIndex := 0
	deadline := time.Now().Add(8 * time.Minute)
	for {
		if time.Now().After(deadline) {
			return nil, &provider.ProviderError{Provider: "fal", StatusCode: 504, Message: "fal queue request timed out", Retryable: false}
		}
		base := bases[baseIndex]
		statusBody, err := p.falQueueGET(ctx, base+"/status")
		if err != nil {
			if pe, ok := err.(*provider.ProviderError); ok && (pe.StatusCode == http.StatusNotFound || pe.StatusCode == http.StatusMethodNotAllowed) && baseIndex < len(bases)-1 {
				baseIndex++
				continue
			}
			return nil, err
		}
		var status struct {
			Status string `json:"status"`
		}
		_ = json.Unmarshal(statusBody, &status)
		switch strings.ToUpper(status.Status) {
		case "COMPLETED":
			return p.falQueueGET(ctx, base)
		case "FAILED", "ERROR":
			return nil, &provider.ProviderError{Provider: "fal", StatusCode: 502, Message: string(statusBody), Retryable: false}
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func (p *FalProvider) falQueueRequestBases(modelID, requestID string) []string {
	queueBase := strings.TrimRight(p.queueBaseURL(), "/")
	bases := []string{queueBase + "/" + strings.TrimLeft(modelID, "/") + "/requests/" + requestID}
	if strings.Contains(modelID, "bytedance/seedance-2.0") {
		bases = append(bases, queueBase+"/bytedance/seedance-2.0/requests/"+requestID)
	}
	if strings.Contains(modelID, "alibaba/happy-horse/") {
		bases = append(bases, queueBase+"/alibaba/happy-horse/requests/"+requestID)
	}
	if strings.Contains(modelID, "fal-ai/ltx-2.3/") {
		bases = append(bases, queueBase+"/fal-ai/ltx-2.3/requests/"+requestID)
	}
	if strings.Contains(modelID, "fal-ai/hidream-o1-image/") {
		bases = append(bases, queueBase+"/fal-ai/hidream-o1-image/requests/"+requestID)
	}
	if strings.Contains(modelID, "fal-ai/flux-2-pro/outpaint") {
		bases = append(bases, queueBase+"/fal-ai/flux-2-pro/requests/"+requestID)
	}
	if strings.Contains(modelID, "fal-ai/image-apps-v2/") {
		bases = append(bases, queueBase+"/fal-ai/image-apps-v2/requests/"+requestID)
	}
	return bases
}

func (p *FalProvider) falQueueGET(ctx context.Context, url string) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Key "+p.apiKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: resp.StatusCode, Message: string(body), Retryable: resp.StatusCode >= 500 || resp.StatusCode == 429}
	}
	return body, nil
}

func parseFalVideoResult(body []byte) (*model.VideoGenerationResponse, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}
	result := &model.VideoGenerationResponse{}
	if video, ok := raw["video"].(map[string]any); ok {
		result.VideoURL, _ = video["url"].(string)
	} else if videoURL, ok := raw["video_url"].(string); ok {
		result.VideoURL = videoURL
	}
	if seed, ok := raw["seed"].(float64); ok {
		s := int(seed)
		result.Seed = &s
	}
	if result.VideoURL == "" {
		return nil, &provider.ProviderError{Provider: "fal", StatusCode: 502, Message: "no video url in response: " + string(body), Retryable: false}
	}
	return result, nil
}

func falOutputMode(req *model.ImageGenerationRequest) (outputFormat string, syncMode bool) {
	respFormat := strings.ToLower(strings.TrimSpace(req.ResponseFormat))
	if req.OutputFormat != "" {
		return strings.ToLower(strings.TrimSpace(req.OutputFormat)), respFormat == "b64_json"
	}
	modelName := strings.ToLower(req.Model)

	switch respFormat {
	case "b64_json":
		return "png", true
	case "url", "":
		if strings.Contains(modelName, "gpt-image") && respFormat == "" {
			return "png", true
		}
		return "", false
	default:
		return "", false
	}
}

func dataURIToBase64(uri string) (string, error) {
	const marker = ";base64,"
	idx := strings.Index(uri, marker)
	if idx == -1 {
		return "", fmt.Errorf("data URI missing base64 marker")
	}
	return uri[idx+len(marker):], nil
}

func (p *FalProvider) Transcribe(ctx context.Context, req *model.TranscriptionRequest) (*model.TranscriptionResponse, error) {
	mime := mimeType(req.Filename)
	b64 := base64.StdEncoding.EncodeToString(req.File)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mime, b64)

	falReq := falRequest{
		AudioURL:   dataURL,
		Task:       "transcribe",
		ChunkLevel: "segment",
	}

	body, err := json.Marshal(falReq)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://fal.run/fal-ai/whisper", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Key "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "fal", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "fal",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var falResp falResponse
	if err := json.Unmarshal(respBody, &falResp); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return &model.TranscriptionResponse{Text: falResp.Text}, nil
}
