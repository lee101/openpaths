package netwrck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openpath/openpath/internal/model"
	"github.com/openpath/openpath/internal/provider"
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
		return nil, &provider.ProviderError{
			Provider: "netwrck", StatusCode: 502, Message: "no image_url in response", Retryable: false,
		}
	}

	return &model.ImageGenerationResponse{
		Created: time.Now().Unix(),
		Data: []model.ImageData{
			{URL: imageURL},
		},
	}, nil
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
