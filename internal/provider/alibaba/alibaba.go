package alibaba

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	baseprovider "github.com/openpaths/openpaths/internal/provider"
)

const defaultBaseURL = "https://dashscope-intl.aliyuncs.com"

// Provider talks to Alibaba Cloud Model Studio (DashScope) async video synthesis.
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
	return &Provider{apiKey: strings.TrimSpace(apiKey), baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: 2 * time.Minute}, pollInterval: 10 * time.Second}
}

func (p *Provider) Name() string { return "alibaba" }

func (p *Provider) ChatCompletion(context.Context, *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 400, Message: "alibaba provider only supports video generation", Retryable: false}
}

func (p *Provider) ChatCompletionStream(context.Context, *model.ChatCompletionRequest) (<-chan baseprovider.StreamEvent, error) {
	return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 400, Message: "alibaba provider only supports video generation", Retryable: false}
}

func (p *Provider) HealthCheck(ctx context.Context) error {
	status, body, err := p.do(ctx, http.MethodGet, p.baseURL+"/api/v1/tasks/00000000-0000-0000-0000-000000000000", nil)
	if err != nil {
		return err
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden || status >= 500 {
		return fmt.Errorf("alibaba health check: HTTP %d: %s", status, strings.TrimSpace(string(body)))
	}
	return nil
}

type media struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type taskResponse struct {
	RequestID string `json:"request_id"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Output    struct {
		TaskID     string `json:"task_id"`
		TaskStatus string `json:"task_status"`
		VideoURL   string `json:"video_url"`
		Code       string `json:"code"`
		Message    string `json:"message"`
	} `json:"output"`
	Usage *struct {
		Duration            float64 `json:"duration"`
		InputVideoDuration  float64 `json:"input_video_duration"`
		OutputVideoDuration float64 `json:"output_video_duration"`
		SR                  int     `json:"SR"`
	} `json:"usage"`
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
	submitted, err := p.submit(ctx, body)
	if err != nil {
		return nil, err
	}
	taskURL := p.baseURL + "/api/v1/tasks/" + submitted.Output.TaskID
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
		task, err := p.poll(ctx, taskURL)
		if err != nil {
			if providerErr, ok := err.(*baseprovider.ProviderError); !ok || !providerErr.Retryable {
				return nil, err
			}
			continue
		}
		switch strings.ToUpper(task.Output.TaskStatus) {
		case "SUCCEEDED":
			if task.Output.VideoURL == "" {
				return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: "Wan completed without a video URL", Retryable: true}
			}
			resp := &model.VideoGenerationResponse{VideoURL: task.Output.VideoURL, Model: req.Model, BackendUsed: p.Name()}
			// Bill what Alibaba billed: total seconds (input + output video) at the delivered tier.
			req.Resolution = resolutionValue(req.Resolution)
			if task.Usage != nil {
				resp.DurationSeconds = task.Usage.Duration
				if task.Usage.SR > 0 {
					req.Resolution = strconv.Itoa(task.Usage.SR) + "P"
				}
			}
			return resp, nil
		case "FAILED", "CANCELED", "UNKNOWN":
			code, msg := firstNonEmpty(task.Output.Code, task.Code), firstNonEmpty(task.Output.Message, task.Message)
			return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 422, Message: fmt.Sprintf("Wan task %s: %s %s", strings.ToLower(task.Output.TaskStatus), code, msg), Retryable: false}
		}
	}
}

func videoPayload(req *model.VideoGenerationRequest) (map[string]any, error) {
	native := nativeInput(req.Input)
	prompt := strings.TrimSpace(req.TextPrompt())
	if prompt == "" {
		prompt = strings.TrimSpace(native.Prompt)
	}
	items, err := mediaItems(req, native.Media)
	if err != nil {
		return nil, err
	}
	if prompt == "" && len(items) == 0 {
		return nil, fmt.Errorf("prompt or media is required")
	}
	input := map[string]any{}
	if prompt != "" {
		input["prompt"] = prompt
	}
	if len(items) > 0 {
		input["media"] = items
	}
	params := map[string]any{
		"resolution": resolutionValue(req.Resolution),
		"ratio":      ratioValue(firstNonEmpty(req.Ratio, req.AspectRatio)),
		"duration":   durationValue(req.Duration),
		"watermark":  false,
	}
	if req.GenerateAudio != nil {
		params["audio"] = *req.GenerateAudio
	}
	if req.Seed != nil {
		params["seed"] = *req.Seed
	}
	switch strings.ToLower(strings.TrimSpace(req.PromptExpansionMode)) {
	case "off", "false", "disabled", "none":
		params["prompt_extend"] = false
	}
	for _, item := range items {
		if item.Type == "file" || item.Type == "link" {
			params["prompt_extend"] = true
		}
	}
	return map[string]any{"model": req.Model, "input": input, "parameters": params}, nil
}

type nativeShape struct {
	Prompt string  `json:"prompt"`
	Media  []media `json:"media"`
}

func nativeInput(raw json.RawMessage) nativeShape {
	var out nativeShape
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return out
}

var mediaLimits = map[string]int{"first_frame": 1, "last_frame": 1, "reference_image": 10, "reference_video": 5, "reference_audio": 5, "file": 1, "link": 1}

func mediaItems(req *model.VideoGenerationRequest, native []media) ([]media, error) {
	var refImages, refVideos, refAudios []string
	var first, last string
	var extra []media
	add := func(list *[]string, value string) {
		if value = strings.TrimSpace(value); value != "" {
			*list = append(*list, value)
		}
	}
	for _, value := range req.ReferenceImageURLs {
		add(&refImages, value)
	}
	for _, value := range req.ImageURLs {
		add(&refImages, value)
	}
	add(&refVideos, req.VideoURL)
	if req.Video != nil {
		add(&refVideos, req.Video.URL)
	}
	for _, value := range req.VideoURLs {
		add(&refVideos, value)
	}
	add(&refAudios, req.AudioURL)
	for _, value := range req.AudioURLs {
		add(&refAudios, value)
	}
	for _, item := range req.Content {
		switch {
		case item.Type == "image_url" && item.ImageURL != nil:
			switch item.Role {
			case "first_frame":
				first = strings.TrimSpace(item.ImageURL.URL)
			case "last_frame":
				last = strings.TrimSpace(item.ImageURL.URL)
			default:
				add(&refImages, item.ImageURL.URL)
			}
		case item.Type == "video_url" && item.VideoURL != nil:
			add(&refVideos, item.VideoURL.URL)
		case item.Type == "audio_url" && item.AudioURL != nil:
			add(&refAudios, item.AudioURL.URL)
		}
	}
	for _, item := range native {
		item.Type, item.URL = strings.TrimSpace(item.Type), strings.TrimSpace(item.URL)
		if item.URL == "" {
			continue
		}
		switch item.Type {
		case "first_frame":
			first = item.URL
		case "last_frame":
			last = item.URL
		case "reference_image":
			refImages = append(refImages, item.URL)
		case "reference_video":
			refVideos = append(refVideos, item.URL)
		case "reference_audio":
			refAudios = append(refAudios, item.URL)
		case "file", "link":
			extra = append(extra, item)
		default:
			return nil, fmt.Errorf("unsupported media type %q", item.Type)
		}
	}
	hasRefs := len(refImages)+len(refVideos)+len(refAudios)+len(extra) > 0
	// A lone image_url is a first frame; alongside references it is one more reference.
	if image := strings.TrimSpace(req.ImageURL); image != "" {
		if hasRefs {
			refImages = append([]string{image}, refImages...)
		} else if first == "" {
			first = image
		}
	}
	if end := strings.TrimSpace(req.EndImageURL); end != "" && last == "" {
		last = end
	}
	if (first != "" || last != "") && hasRefs {
		return nil, fmt.Errorf("first/last frame cannot be combined with reference images, videos, audio, files or links")
	}
	items := make([]media, 0, len(refImages)+len(refVideos)+len(refAudios)+len(extra)+2)
	if first != "" {
		items = append(items, media{"first_frame", first})
	}
	if last != "" {
		items = append(items, media{"last_frame", last})
	}
	for _, value := range refImages {
		items = append(items, media{"reference_image", value})
	}
	for _, value := range refVideos {
		items = append(items, media{"reference_video", value})
	}
	for _, value := range refAudios {
		items = append(items, media{"reference_audio", value})
	}
	items = append(items, extra...)
	counts := map[string]int{}
	for _, item := range items {
		counts[item.Type]++
		if counts[item.Type] > mediaLimits[item.Type] {
			return nil, fmt.Errorf("at most %d %s inputs are allowed", mediaLimits[item.Type], item.Type)
		}
	}
	if counts["file"] > 0 && counts["link"] > 0 {
		return nil, fmt.Errorf("file and link inputs cannot be combined")
	}
	return items, nil
}

func resolutionValue(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "480P", "480", "SD":
		return "480P"
	case "720P", "720", "HD":
		return "720P"
	default:
		return "1080P"
	}
}

func ratioValue(value string) string {
	switch value = strings.TrimSpace(value); value {
	case "21:9", "16:9", "4:3", "1:1", "3:4", "9:16":
		return value
	default:
		return "adaptive"
	}
}

func durationValue(duration model.VideoDuration) int {
	value := strings.TrimSpace(string(duration))
	if value == "" {
		return 5
	}
	if strings.EqualFold(value, "auto") || value == "-1" {
		return -1
	}
	seconds, err := strconv.Atoi(value)
	if err != nil {
		return 5
	}
	return min(max(seconds, 2), 30)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func (p *Provider) submit(ctx context.Context, body []byte) (*taskResponse, error) {
	var last error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(1<<attempt) * time.Second):
			}
		}
		status, responseBody, err := p.do(ctx, http.MethodPost, p.baseURL+"/api/v1/services/aigc/video-generation/video-synthesis", body)
		if err != nil {
			last = err
			continue
		}
		var out taskResponse
		_ = json.Unmarshal(responseBody, &out)
		if status == http.StatusOK && out.Output.TaskID != "" {
			return &out, nil
		}
		retryable := status == http.StatusTooManyRequests || status >= 500
		msg := strings.TrimSpace(firstNonEmpty(strings.TrimSpace(out.Code+" "+out.Message), string(responseBody)))
		last = &baseprovider.ProviderError{Provider: p.Name(), StatusCode: status, Message: msg, Retryable: retryable}
		if !retryable {
			return nil, last
		}
	}
	return nil, last
}

func (p *Provider) poll(ctx context.Context, taskURL string) (*taskResponse, error) {
	status, body, err := p.do(ctx, http.MethodGet, taskURL, nil)
	if err != nil {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	if status != http.StatusOK {
		return nil, &baseprovider.ProviderError{Provider: p.Name(), StatusCode: status, Message: strings.TrimSpace(string(body)), Retryable: status == 429 || status >= 500}
	}
	var out taskResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode Wan task: %w", err)
	}
	return &out, nil
}

func (p *Provider) do(ctx context.Context, method, endpoint string, body []byte, async ...bool) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		if len(async) == 0 || async[0] {
			req.Header.Set("X-DashScope-Async", "enable")
		}
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, responseBody, err
}
