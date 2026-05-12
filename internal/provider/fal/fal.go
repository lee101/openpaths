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

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

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
	falReq := map[string]any{
		"prompt": req.Prompt,
	}
	if req.Quality != "" {
		falReq["quality"] = req.Quality
	}
	if req.Size != "" {
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
	return strings.Contains(strings.ToLower(modelID), "hidream-o1-image")
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

	var images []model.ImageData
	if imgs, ok := raw["images"].([]any); ok {
		for _, img := range imgs {
			if m, ok := img.(map[string]any); ok {
				if url, ok := m["url"].(string); ok {
					if strings.HasPrefix(url, "data:image/") {
						if b64, err := dataURIToBase64(url); err == nil {
							images = append(images, model.ImageData{B64JSON: b64})
							continue
						}
					}
					images = append(images, model.ImageData{URL: url})
				}
			}
		}
	}
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

func (p *FalProvider) GenerateVideo(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	falReq := map[string]any{
		"prompt": req.Prompt,
	}
	if req.Resolution != "" {
		falReq["resolution"] = req.Resolution
	}
	if req.Duration != "" {
		falReq["duration"] = req.Duration
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
	if strings.Contains(modelID, "fal-ai/hidream-o1-image/") {
		bases = append(bases, queueBase+"/fal-ai/hidream-o1-image/requests/"+requestID)
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
