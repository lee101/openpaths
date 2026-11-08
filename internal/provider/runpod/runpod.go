// Package runpod adds Runpod Public Endpoints (serverless video models such as
// Pruna P-Video, Seedance 2.5, WAN 2.6, Vidu Q3) on top of the OpenAI-compatible
// vLLM chat endpoint the runpod provider already serves.
package runpod

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
	"github.com/openpaths/openpaths/internal/provider/openai"
)

const defaultAPIBase = "https://api.runpod.ai/v2"

// Provider embeds the OpenAI-compatible chat provider and adds GenerateVideo.
type Provider struct {
	*openai.OpenAIProvider
	apiKey       string
	apiBase      string
	client       *http.Client
	pollInterval time.Duration
}

func New(chat *openai.OpenAIProvider, apiKey string) *Provider {
	return &Provider{
		OpenAIProvider: chat,
		apiKey:         strings.TrimSpace(apiKey),
		apiBase:        defaultAPIBase,
		client:         &http.Client{Timeout: 2 * time.Minute, Transport: baseprovider.UserAgentTransport(nil)},
		pollInterval:   3 * time.Second,
	}
}

type jobResponse struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Error  string          `json:"error"`
	Output json.RawMessage `json:"output"`
}

type jobOutput struct {
	Result string  `json:"result"`
	Cost   float64 `json:"cost"`
}

func (p *Provider) GenerateVideo(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	endpoint := strings.Trim(req.Model, "/")
	input, err := videoInput(endpoint, req)
	if err != nil {
		return nil, &baseprovider.ProviderError{Provider: "runpod", StatusCode: 400, Message: err.Error(), Err: err}
	}
	body, _ := json.Marshal(map[string]any{"input": input})
	var job jobResponse
	if err := p.call(ctx, http.MethodPost, p.apiBase+"/"+endpoint+"/run", body, &job); err != nil {
		return nil, err
	}
	if job.ID == "" {
		return nil, &baseprovider.ProviderError{Provider: "runpod", StatusCode: 502, Message: "Runpod returned no job id", Retryable: true}
	}
	for !terminal(job.Status) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(p.pollInterval):
		}
		if err := p.call(ctx, http.MethodGet, p.apiBase+"/"+endpoint+"/status/"+job.ID, nil, &job); err != nil {
			return nil, err
		}
	}
	if job.Status != "COMPLETED" {
		msg := job.Error
		if msg == "" {
			msg = "Runpod job " + strings.ToLower(job.Status)
		}
		return nil, &baseprovider.ProviderError{Provider: "runpod", StatusCode: 502, Message: msg, Retryable: job.Status != "CANCELLED"}
	}
	var out jobOutput
	if err := json.Unmarshal(job.Output, &out); err != nil || out.Result == "" {
		return nil, &baseprovider.ProviderError{Provider: "runpod", StatusCode: 502, Message: "Runpod completed without a video URL", Retryable: true}
	}
	return &model.VideoGenerationResponse{VideoURL: out.Result, OutputFormat: "mp4", Model: req.Model, BackendUsed: "runpod"}, nil
}

func terminal(s string) bool {
	switch s {
	case "COMPLETED", "FAILED", "CANCELLED", "TIMED_OUT":
		return true
	}
	return false
}

func (p *Provider) call(ctx context.Context, method, url string, body []byte, out any) error {
	httpReq, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return &baseprovider.ProviderError{Provider: "runpod", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 400 {
		return &baseprovider.ProviderError{Provider: "runpod", StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(raw)), Retryable: resp.StatusCode >= 500 || resp.StatusCode == 429}
	}
	return json.Unmarshal(raw, out)
}

// videoInput maps the generic request onto each Runpod endpoint's schema.
func videoInput(endpoint string, req *model.VideoGenerationRequest) (map[string]any, error) {
	in := map[string]any{"prompt": req.TextPrompt()}
	if d := strings.TrimSpace(string(req.Duration)); d != "" && d != "auto" {
		n, err := strconv.Atoi(d)
		if err != nil {
			return nil, fmt.Errorf("duration must be an integer number of seconds")
		}
		in["duration"] = n
	}
	if req.Seed != nil {
		in["seed"] = *req.Seed
	}
	image := req.ImageURL
	if image == "" && len(req.ImageURLs) > 0 {
		image = req.ImageURLs[0]
	}
	video := req.VideoURL
	if video == "" && req.Video != nil {
		video = req.Video.URL
	}
	if video == "" && len(req.VideoURLs) > 0 {
		video = req.VideoURLs[0]
	}
	setIf := func(k, v string) {
		if v != "" {
			in[k] = v
		}
	}
	switch {
	case endpoint == "p-video":
		setIf("image", image)
		setIf("audio", req.AudioURL)
		setIf("resolution", req.Resolution)
		if image == "" {
			setIf("aspect_ratio", req.AspectRatio)
		}
		if req.FramesPerSecond > 0 {
			in["fps"] = req.FramesPerSecond
		}
		if req.GenerateAudio != nil {
			in["save_audio"] = *req.GenerateAudio
		}
	case endpoint == "p-video-edit":
		if video == "" {
			return nil, fmt.Errorf("p-video-edit requires video_url")
		}
		in["video"] = video
		delete(in, "duration")
		if refs := req.ReferenceImageURLs; len(refs) > 0 {
			in["images"] = refs[:min(len(refs), 4)]
		} else if len(req.ImageURLs) > 0 {
			in["images"] = req.ImageURLs[:min(len(req.ImageURLs), 4)]
		}
		setIf("resolution", req.Resolution)
	case strings.HasPrefix(endpoint, "seedance-2-5") || endpoint == "16jsvx5ncmm24x" || endpoint == "tdq07exynr7zux":
		setIf("image", image)
		if len(req.ImageURLs) > 1 {
			setIf("last_image", req.ImageURLs[1])
		}
		setIf("last_image", req.EndImageURL)
		setIf("resolution", req.Resolution)
		setIf("aspect_ratio", req.AspectRatio)
		if req.GenerateAudio != nil {
			in["generate_audio"] = *req.GenerateAudio
		}
		if len(req.ReferenceImageURLs) > 0 {
			in["reference_images"] = req.ReferenceImageURLs
		}
		if len(req.VideoURLs) > 0 {
			in["reference_videos"] = req.VideoURLs
		}
		if len(req.AudioURLs) > 0 {
			in["reference_audio"] = req.AudioURLs
		}
	case strings.HasPrefix(endpoint, "wan-2-6"):
		setIf("image", image)
		setIf("audio", req.AudioURL)
		setIf("negative_prompt", req.NegativePrompt)
		if size := wanSize(req.Resolution, req.AspectRatio); size != "" {
			in["size"] = size
		}
		if req.EnableSafetyChecker != nil {
			in["enable_safety_checker"] = *req.EnableSafetyChecker
		}
	case strings.HasPrefix(endpoint, "vidu-q3"):
		setIf("image", image)
		setIf("resolution", req.Resolution)
		if image == "" {
			setIf("aspect_ratio", req.AspectRatio)
		}
		if req.GenerateAudio != nil {
			in["generate_audio"] = *req.GenerateAudio
		}
	case endpoint == "kling-video-o1-r2v":
		refs := req.ReferenceImageURLs
		if len(refs) == 0 {
			refs = req.ImageURLs
		}
		if len(refs) == 0 && image != "" {
			refs = []string{image}
		}
		if len(refs) == 0 {
			return nil, fmt.Errorf("kling-video-o1-r2v requires reference images")
		}
		in["images"] = refs
		setIf("video", video)
		setIf("negative_prompt", req.NegativePrompt)
		setIf("aspect_ratio", req.AspectRatio)
	default:
		return nil, fmt.Errorf("unsupported Runpod video endpoint %q", endpoint)
	}
	return in, nil
}

func wanSize(resolution, aspect string) string {
	portrait := aspect == "9:16"
	switch strings.ToLower(resolution) {
	case "1080p":
		if portrait {
			return "1080*1920"
		}
		return "1920*1080"
	case "720p", "":
		if resolution == "" && aspect == "" {
			return ""
		}
		if portrait {
			return "720*1280"
		}
		return "1280*720"
	}
	return ""
}
