package netwrck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

type NetwrckProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func New(apiKey, baseURL string) *NetwrckProvider {
	if baseURL == "" {
		baseURL = "https://netwrck.com"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &NetwrckProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *NetwrckProvider) Name() string { return "netwrck" }

func (p *NetwrckProvider) HealthCheck(ctx context.Context) error {
	return nil
}

// ChatCompletion - netwrck is not a chat provider, stub to satisfy registry.
func (p *NetwrckProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	return nil, &provider.ProviderError{
		Provider: "netwrck", StatusCode: 400, Message: "netwrck does not support chat", Retryable: false,
	}
}

func (p *NetwrckProvider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	return nil, &provider.ProviderError{
		Provider: "netwrck", StatusCode: 400, Message: "netwrck does not support chat", Retryable: false,
	}
}

// GenerateImage handles ra1 and zimage endpoints.
func (p *NetwrckProvider) GenerateImage(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	endpoint := "/api/" + req.Model

	netwrckReq := map[string]any{
		"api_key": p.apiKey,
		"prompt":  req.Prompt,
	}
	if req.Size != "" {
		netwrckReq["size"] = req.Size
	} else {
		netwrckReq["size"] = "1024x1024"
	}

	body, err := json.Marshal(netwrckReq)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "netwrck", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		if fallbackURL, fallbackErr := p.generateViaFalFallback(ctx, req); fallbackErr == nil {
			return imageResponse(fallbackURL), nil
		}
		return nil, &provider.ProviderError{
			Provider:   "netwrck",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	imageURL, _ := raw["image_url"].(string)
	if imageURL == "" {
		if fallbackURL, fallbackErr := p.generateViaFalFallback(ctx, req); fallbackErr == nil {
			return imageResponse(fallbackURL), nil
		}
		return nil, &provider.ProviderError{
			Provider: "netwrck", StatusCode: 502, Message: "no image_url in response", Retryable: false,
		}
	}

	return imageResponse(imageURL), nil
}

func imageResponse(imageURL string) *model.ImageGenerationResponse {
	return &model.ImageGenerationResponse{
		Created: time.Now().Unix(),
		Data: []model.ImageData{
			{URL: imageURL},
		},
	}
}

func (p *NetwrckProvider) generateViaFalFallback(ctx context.Context, req *model.ImageGenerationRequest) (string, error) {
	key := os.Getenv("FAL_KEY")
	if key == "" {
		key = os.Getenv("FAL_API_KEY")
	}
	if key == "" {
		return "", fmt.Errorf("fal fallback key not configured")
	}

	modelID := strings.ToLower(req.Model)
	endpoint := "fal-ai/flux/dev"
	payload := map[string]any{
		"prompt":                "Ultra realistic, " + req.Prompt,
		"image_size":            falImageSize(req.Size),
		"num_images":            imageCount(req.N),
		"num_inference_steps":   28,
		"guidance_scale":        3.5,
		"enable_safety_checker": false,
		"output_format":         "png",
	}
	if modelID == "zimage" || modelID == "netwrck-anime" || modelID == "anime-image" {
		endpoint = "fal-ai/z-image/turbo"
		payload = map[string]any{
			"prompt":                req.Prompt,
			"image_size":            falImageSize(req.Size),
			"num_images":            imageCount(req.N),
			"num_inference_steps":   8,
			"enable_safety_checker": false,
			"output_format":         "webp",
			"acceleration":          "regular",
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://fal.run/"+endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Key "+key)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fal fallback status %d: %s", resp.StatusCode, string(respBody))
	}

	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return "", err
	}
	if status, _ := raw["status"].(string); status == "IN_QUEUE" {
		responseURL, _ := raw["response_url"].(string)
		if responseURL == "" {
			return "", fmt.Errorf("fal fallback missing response_url")
		}
		for range 60 {
			time.Sleep(2 * time.Second)
			pollReq, err := http.NewRequestWithContext(ctx, "GET", responseURL, nil)
			if err != nil {
				return "", err
			}
			pollReq.Header.Set("Authorization", "Key "+key)
			pollResp, err := p.client.Do(pollReq)
			if err != nil {
				continue
			}
			pollBody, _ := io.ReadAll(pollResp.Body)
			pollResp.Body.Close()
			if pollResp.StatusCode == http.StatusAccepted {
				continue
			}
			if pollResp.StatusCode != http.StatusOK {
				return "", fmt.Errorf("fal fallback poll status %d: %s", pollResp.StatusCode, string(pollBody))
			}
			if err := json.Unmarshal(pollBody, &raw); err != nil {
				return "", err
			}
			break
		}
	}

	images, _ := raw["images"].([]any)
	for _, item := range images {
		img, _ := item.(map[string]any)
		if url, _ := img["url"].(string); url != "" {
			return url, nil
		}
	}
	return "", fmt.Errorf("fal fallback returned no image url")
}

func imageCount(n int) int {
	if n > 0 {
		return n
	}
	return 1
}

func falImageSize(size string) string {
	switch size {
	case "1360x768", "1152x768", "1024x576":
		return "landscape_16_9"
	case "768x1360", "768x1152", "576x1024":
		return "portrait_16_9"
	default:
		return "square_hd"
	}
}

// GenerateVideo handles wan, ltx-video-v097, ltx-2, ra2v endpoints.
func (p *NetwrckProvider) GenerateVideo(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	endpoint := "/api/" + req.Model

	netwrckReq := map[string]any{
		"api_key": p.apiKey,
		"prompt":  req.Prompt,
	}
	if req.ImageURL != "" {
		netwrckReq["image_url"] = req.ImageURL
	}
	if req.NumFrames > 0 {
		netwrckReq["num_frames"] = req.NumFrames
	}
	if req.FramesPerSecond > 0 {
		netwrckReq["frames_per_second"] = req.FramesPerSecond
	}
	if req.Resolution != "" {
		netwrckReq["resolution"] = req.Resolution
	}
	if req.AspectRatio != "" {
		netwrckReq["aspect_ratio"] = req.AspectRatio
	}
	if req.NegativePrompt != "" {
		netwrckReq["negative_prompt"] = req.NegativePrompt
	}
	if req.Seed != nil {
		netwrckReq["seed"] = *req.Seed
	}
	if req.NumInferenceSteps > 0 {
		netwrckReq["num_inference_steps"] = req.NumInferenceSteps
	}
	if req.GuidanceScale != nil {
		netwrckReq["guidance_scale"] = *req.GuidanceScale
	}
	if req.EnableSafetyChecker != nil {
		netwrckReq["enable_safety_checker"] = *req.EnableSafetyChecker
	}

	body, err := json.Marshal(netwrckReq)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "netwrck", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "netwrck",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var raw map[string]any
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	result := &model.VideoGenerationResponse{}

	// Different endpoints return video URL in different fields
	if videoURL, ok := raw["video_url"].(string); ok {
		result.VideoURL = videoURL
	} else if video, ok := raw["video"].(string); ok {
		result.VideoURL = video
	} else if res, ok := raw["result"].(map[string]any); ok {
		if vid, ok := res["video"].(map[string]any); ok {
			result.VideoURL, _ = vid["url"].(string)
		}
		if seed, ok := res["seed"].(float64); ok {
			s := int(seed)
			result.Seed = &s
		}
	}

	if backendUsed, ok := raw["backend_used"].(string); ok {
		result.BackendUsed = backendUsed
	}
	if credits, ok := raw["credits_charged"].(float64); ok {
		result.CreditsCharged = credits
	}

	if result.VideoURL == "" {
		return nil, &provider.ProviderError{
			Provider: "netwrck", StatusCode: 502, Message: "no video_url in response: " + string(respBody), Retryable: false,
		}
	}

	return result, nil
}
