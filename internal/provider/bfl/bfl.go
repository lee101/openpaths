package bfl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	imgutil "github.com/openpaths/openpaths/internal/image"
	"github.com/openpaths/openpaths/internal/model"
	baseprovider "github.com/openpaths/openpaths/internal/provider"
)

const defaultBaseURL = "https://api.us2.bfl.ai"

type Provider struct {
	apiKey       string
	baseURL      string
	client       *http.Client
	pollInterval time.Duration
}

func New(apiKey, baseURL string) *Provider {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	return &Provider{apiKey: strings.TrimSpace(apiKey), baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: 2 * time.Minute}, pollInterval: 2 * time.Second}
}

func (p *Provider) Name() string { return "bfl" }

func (p *Provider) ChatCompletion(context.Context, *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 400, Message: "BFL does not support chat", Retryable: false}
}

func (p *Provider) ChatCompletionStream(context.Context, *model.ChatCompletionRequest) (<-chan baseprovider.StreamEvent, error) {
	return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 400, Message: "BFL does not support chat", Retryable: false}
}

func (p *Provider) HealthCheck(ctx context.Context) error {
	status, body, err := p.do(ctx, http.MethodGet, p.baseURL+"/v1/credits", nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("BFL health check: HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	return nil
}

type asyncResponse struct {
	ID         string `json:"id"`
	PollingURL string `json:"polling_url"`
}

type resultResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Result *struct {
		Sample  string   `json:"sample"`
		Samples []string `json:"samples"`
	} `json:"result"`
	Details any `json:"details"`
}

func (p *Provider) GenerateImage(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	payload, err := bflImagePayload(req)
	if err != nil {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 400, Message: err.Error(), Retryable: false, Err: err}
	}
	count := req.N
	if count <= 0 {
		count = req.NumImages
	}
	if count <= 0 {
		count = 1
	}

	response := &model.ImageGenerationResponse{Created: time.Now().Unix(), Data: make([]model.ImageData, 0, count)}
	for i := 0; i < count; i++ {
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		var submitted asyncResponse
		if err := p.submitWithRetry(ctx, "/v1/flux-2-pro-preview", body, &submitted); err != nil {
			return nil, err
		}
		pollURL, err := p.resolvePollingURL(submitted.PollingURL)
		if err != nil || submitted.ID == "" {
			return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: "BFL returned an invalid image task id or polling URL", Retryable: true, Err: err}
		}
		result, err := p.waitForResult(ctx, pollURL)
		if err != nil {
			return nil, err
		}
		imageURL := ""
		if result.Result != nil {
			imageURL = result.Result.Sample
			if imageURL == "" && len(result.Result.Samples) > 0 {
				imageURL = result.Result.Samples[0]
			}
		}
		if imageURL == "" {
			return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: "BFL completed without an image URL", Retryable: true}
		}
		response.Data = append(response.Data, model.ImageData{URL: imageURL, Width: payload["width"].(int), Height: payload["height"].(int)})
	}
	return response, nil
}

func bflImagePayload(req *model.ImageGenerationRequest) (map[string]any, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	width, height := 1024, 1024
	if strings.TrimSpace(req.Size) != "" {
		size, ok := imgutil.ParseSize(req.Size)
		if !ok || size.W < 64 || size.H < 64 || size.W%16 != 0 || size.H%16 != 0 || size.W*size.H > 2048*2048 {
			return nil, fmt.Errorf("size must use multiples of 16 from 64x64 through 4 MP")
		}
		width, height = size.W, size.H
	}
	format := strings.ToLower(strings.TrimSpace(req.OutputFormat))
	if format == "jpg" {
		format = "jpeg"
	}
	if format != "png" && format != "jpeg" && format != "webp" {
		format = "webp"
	}
	disablePUP := false
	if req.DisablePUP != nil {
		disablePUP = *req.DisablePUP
	}
	payload := map[string]any{
		"prompt": prompt, "safety_tolerance": 5, "width": width, "height": height,
		"output_format": format, "disable_pup": disablePUP,
	}
	if req.Seed != nil {
		payload["seed"] = *req.Seed
	}
	for index, imageURL := range bflInputImages(req) {
		if index >= 8 {
			break
		}
		key := "input_image"
		if index > 0 {
			key = fmt.Sprintf("input_image_%d", index+1)
		}
		payload[key] = imageURL
	}
	return payload, nil
}

func bflInputImages(req *model.ImageGenerationRequest) []string {
	values := make([]string, 0, 8)
	add := func(value string) {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	add(req.ImageURL)
	if req.Image != nil {
		add(req.Image.URL)
	}
	for _, image := range req.Images {
		add(image.URL)
	}
	for _, value := range req.ImageURLs {
		add(value)
	}
	for _, value := range req.ReferenceImageURLs {
		add(value)
	}
	return values
}

func (p *Provider) GenerateVideo(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	payload, err := videoPayload(req)
	if err != nil {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 400, Message: err.Error(), Retryable: false, Err: err}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var submitted asyncResponse
	if err := p.submitWithRetry(ctx, "/v1/flux-3-video", body, &submitted); err != nil {
		return nil, err
	}
	pollURL, err := p.resolvePollingURL(submitted.PollingURL)
	if err != nil || submitted.ID == "" {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: "BFL returned an invalid task id or polling URL", Retryable: true, Err: err}
	}

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()
	for {
		result, err := p.poll(ctx, pollURL)
		if err != nil {
			if providerErr, ok := err.(*baseprovider.ProviderError); !ok || !providerErr.Retryable {
				return nil, err
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-ticker.C:
				continue
			}
		}
		switch strings.ToLower(result.Status) {
		case "ready":
			videoURL := ""
			if result.Result != nil {
				videoURL = result.Result.Sample
				if videoURL == "" && len(result.Result.Samples) > 0 {
					videoURL = result.Result.Samples[0]
				}
			}
			if videoURL == "" {
				return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: "BFL completed without a video URL", Retryable: true}
			}
			return &model.VideoGenerationResponse{VideoURL: videoURL, Model: req.Model, BackendUsed: p.Name()}, nil
		case "error", "failed", "request moderated", "content moderated":
			details, _ := json.Marshal(result.Details)
			return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 422, Message: fmt.Sprintf("BFL generation %s: %s", result.Status, details), Retryable: false}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func videoPayload(req *model.VideoGenerationRequest) (map[string]any, error) {
	prompt := strings.TrimSpace(req.TextPrompt())
	if prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	draft := strings.Contains(strings.ToLower(req.Model), "draft")
	payload := map[string]any{
		"prompt": prompt, "mode": "t2v", "aspect_ratio": valueOr(req.AspectRatio, "auto"),
		"duration": durationValue(req.Duration), "resolution": resolutionValue(req.Resolution),
		"generate_audio": req.GenerateAudio == nil || *req.GenerateAudio, "safety_tolerance": 4, "draft": draft,
	}
	if draft {
		payload["resolution"] = "hd"
	}
	if video := firstVideo(req); video != "" {
		if draft {
			return nil, fmt.Errorf("FLUX 3 Video Draft does not support video-to-video")
		}
		payload["mode"], payload["start_video"] = "v2v", video
		return payload, nil
	}
	if frames := imageFrames(req); len(frames) > 0 {
		payload["mode"], payload["keyframes"] = "i2v", frames
	}
	return payload, nil
}

func durationValue(duration model.VideoDuration) any {
	value := strings.TrimSpace(string(duration))
	if value == "" || strings.EqualFold(value, "auto") {
		return "auto"
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 5 || seconds > 20 {
		return "auto"
	}
	return seconds
}

func resolutionValue(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "fhd", "full hd", "1080p":
		return "fhd"
	default:
		return "hd"
	}
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func imageFrames(req *model.VideoGenerationRequest) []string {
	frames := make([]string, 0, len(req.ImageURLs)+2)
	add := func(value string) {
		if value = strings.TrimSpace(value); value != "" {
			frames = append(frames, value)
		}
	}
	add(req.ImageURL)
	for _, value := range req.ImageURLs {
		add(value)
	}
	for _, item := range req.Content {
		if item.Type == "image_url" && item.ImageURL != nil {
			add(item.ImageURL.URL)
		}
	}
	add(req.EndImageURL)
	return frames
}

func firstVideo(req *model.VideoGenerationRequest) string {
	if value := strings.TrimSpace(req.VideoURL); value != "" {
		return value
	}
	if req.Video != nil && strings.TrimSpace(req.Video.URL) != "" {
		return strings.TrimSpace(req.Video.URL)
	}
	for _, value := range req.VideoURLs {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	for _, item := range req.Content {
		if item.Type == "video_url" && item.VideoURL != nil && strings.TrimSpace(item.VideoURL.URL) != "" {
			return strings.TrimSpace(item.VideoURL.URL)
		}
	}
	return ""
}

func (p *Provider) submitWithRetry(ctx context.Context, endpoint string, body []byte, out *asyncResponse) error {
	var last error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(1<<attempt) * time.Second):
			}
		}
		status, responseBody, err := p.do(ctx, http.MethodPost, p.baseURL+endpoint, body)
		if err != nil {
			last = err
			continue
		}
		if status == http.StatusOK {
			return json.Unmarshal(responseBody, out)
		}
		retryable := status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable || status >= 500
		last = &baseprovider.ProviderError{Provider: p.Name(), StatusCode: status, Message: strings.TrimSpace(string(responseBody)), Retryable: retryable}
		if !retryable {
			return last
		}
	}
	return last
}

func (p *Provider) waitForResult(ctx context.Context, pollURL string) (*resultResponse, error) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()
	for {
		result, err := p.poll(ctx, pollURL)
		if err == nil {
			switch strings.ToLower(result.Status) {
			case "ready":
				return result, nil
			case "error", "failed", "request moderated", "content moderated":
				details, _ := json.Marshal(result.Details)
				return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 422, Message: fmt.Sprintf("BFL generation %s: %s", result.Status, details), Retryable: false}
			}
		} else if providerErr, ok := err.(*baseprovider.ProviderError); !ok || !providerErr.Retryable {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (p *Provider) poll(ctx context.Context, pollURL string) (*resultResponse, error) {
	status, body, err := p.do(ctx, http.MethodGet, pollURL, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: status, Message: strings.TrimSpace(string(body)), Retryable: status == 429 || status >= 500}
	}
	var result resultResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode BFL result: %w", err)
	}
	return &result, nil
}

func (p *Provider) do(ctx context.Context, method, endpoint string, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-key", p.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, responseBody, err
}

func (p *Provider) resolvePollingURL(value string) (string, error) {
	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	if parsed.IsAbs() {
		if parsed.Scheme != "https" && parsed.Scheme != "http" {
			return "", fmt.Errorf("unsupported polling URL scheme %q", parsed.Scheme)
		}
		return parsed.String(), nil
	}
	base, err := url.Parse(p.baseURL)
	if err != nil {
		return "", err
	}
	return base.ResolveReference(parsed).String(), nil
}
