// Package manifoldgen integrates ManifoldGen (https://manifoldgen.com), our
// first-party generative-media GPU lane: K-Fold/H3 cinematic video,
// Wan-Animate character animation, H3 control restyle, transparent-video
// background removal, MiniMax-Music3 music, H3 SFX, H3 image generation and
// editing, and text-to-speech.
//
// Every capability shares one endpoint:
//
//	POST https://manifoldgen.com/api/service
//	Authorization: Bearer <api key>
//	{"service": "<name>", ...service params}
//
// Heavy GPU services answer 202 with
//
//	{"result": {"job_id": "...", "status": "queued", "status_url": "/api/video-jobs/..."}}
//
// and are polled at GET {status_url} until the job reaches a terminal state.
// Light services return the backend payload inside "result" synchronously.
package manifoldgen

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
	"github.com/openpaths/openpaths/internal/provider"
)

const DefaultBaseURL = "https://manifoldgen.com"

// pollInterval matches the server's own worker cadence; pollDeadline stays
// under OpenPaths' 15-minute durable video job budget so the caller still gets
// a structured error instead of a context cancellation.
const (
	pollInterval = 3 * time.Second
	pollDeadline = 13 * time.Minute

	// dramatizePollDeadline covers the upstream video_dramatize agent, whose
	// multi-shot runs can legitimately take ~30-60 minutes.
	dramatizePollDeadline = 55 * time.Minute
)

type Provider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func New(apiKey, baseURL string) *Provider {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Provider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *Provider) Name() string { return "manifoldgen" }

func (p *Provider) HealthCheck(ctx context.Context) error { return nil }

func (p *Provider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	return nil, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 400, Message: "manifoldgen does not support chat", Retryable: false}
}

func (p *Provider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	return nil, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 400, Message: "manifoldgen does not support chat", Retryable: false}
}

// serviceEnvelope mirrors the shared /api/service response wrapper.
type serviceEnvelope struct {
	Result        json.RawMessage `json:"result"`
	CreditsUsed   float64         `json:"credits_used"`
	CreditsRemain float64         `json:"credits_remain"`
	Error         string          `json:"error"`
}

// jobHandle is what async services place inside "result" on submission.
type jobHandle struct {
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	StatusURL string `json:"status_url"`
}

// jobStatus is the polled shape of GET {status_url}.
type jobStatus struct {
	Job struct {
		Status string          `json:"status"`
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	} `json:"job"`
	Error string `json:"error"`
}

func manifoldgenError(body []byte, status int) string {
	var e struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &e) == nil {
		if e.Error != "" {
			return e.Error
		}
		if e.Message != "" {
			return e.Message
		}
	}
	if len(body) > 0 {
		return string(body)
	}
	return "manifoldgen returned status " + strconv.Itoa(status)
}

func requestError(message string) error {
	return &provider.ProviderError{Provider: "manifoldgen", StatusCode: 400, Message: message, Retryable: false}
}

// callService posts one service request and returns the decoded envelope.
func (p *Provider) callService(ctx context.Context, service string, params map[string]any) (*serviceEnvelope, error) {
	payload := make(map[string]any, len(params)+1)
	payload["service"] = service
	for k, v := range params {
		if k == "service" {
			continue
		}
		payload[k] = v
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/service", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &provider.ProviderError{
			Provider:   "manifoldgen",
			StatusCode: resp.StatusCode,
			Message:    manifoldgenError(respBody, resp.StatusCode),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429 || resp.StatusCode == 503,
		}
	}

	var env serviceEnvelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}
	if env.Error != "" {
		return nil, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 502, Message: env.Error, Retryable: false}
	}
	return &env, nil
}

// resolveResult returns the artifact payload for a service response: either the
// synchronous result itself or, when the service queued an async job, the
// terminal job result after polling.
func (p *Provider) resolveResult(ctx context.Context, env *serviceEnvelope, jobDeadline time.Duration) (json.RawMessage, error) {
	var handle jobHandle
	if err := json.Unmarshal(env.Result, &handle); err == nil && handle.JobID != "" && handle.StatusURL != "" {
		return p.waitJob(ctx, handle.StatusURL, jobDeadline)
	}
	return env.Result, nil
}

func (p *Provider) waitJob(ctx context.Context, statusURL string, jobDeadline time.Duration) (json.RawMessage, error) {
	if !strings.HasPrefix(statusURL, "http") {
		statusURL = p.baseURL + statusURL
	}
	limit := time.Now().Add(jobDeadline)
	for {
		result, done, err := p.pollOnce(ctx, statusURL)
		if err != nil || done {
			return result, err
		}
		if time.Now().After(limit) {
			return nil, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 504, Message: "job did not finish in time: " + statusURL, Retryable: true}
		}
		select {
		case <-ctx.Done():
			return nil, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 499, Message: "context cancelled while waiting for job", Err: ctx.Err()}
		case <-time.After(pollInterval):
		}
	}
}

func (p *Provider) pollOnce(ctx context.Context, statusURL string) (json.RawMessage, bool, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return nil, true, fmt.Errorf("create status request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, true, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, fmt.Errorf("read status response: %w", err)
	}

	var state jobStatus
	if err := json.Unmarshal(body, &state); err != nil {
		return nil, true, fmt.Errorf("unmarshal status response: %w", err)
	}
	if state.Job.Error != "" {
		state.Error = state.Job.Error
	}

	switch normalizeJobStatus(state.Job.Status) {
	case "completed", "succeeded":
		return state.Job.Result, true, nil
	case "failed", "error", "timed_out", "cancelled", "canceled":
		message := state.Error
		if message == "" {
			message = "job ended with status " + state.Job.Status
		}
		return nil, true, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 502, Message: message, Retryable: false}
	case "payment_required":
		message := state.Error
		if message == "" {
			message = "upstream account needs a credit top-up to release the finished artifact"
		}
		return nil, true, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 402, Message: message, Retryable: false}
	default: // queued, processing, unknown transient statuses
		return nil, false, nil
	}
}

func normalizeJobStatus(status string) string {
	return strings.ToLower(strings.TrimSpace(status))
}

func durationSeconds(d model.VideoDuration) int {
	raw := strings.TrimSpace(string(d))
	if raw == "" || raw == "auto" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return n
}

func clamp(n, low, high, fallback int) int {
	if n <= 0 {
		return fallback
	}
	if n < low {
		return low
	}
	if n > high {
		return high
	}
	return n
}

func firstString(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

// --- Video ---

// GenerateVideo routes the configured provider_model_id onto the matching
// ManifoldGen service: kfold-video -> h3_video, wan-animate* ->
// character_animation, h3-control -> video_restyle, remove-video-background ->
// video_background_removal.
func (p *Provider) GenerateVideo(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	switch req.Model {
	case "wan-animate", "wan-animate-fast", "wan-animate-xfast":
		return p.generateCharacterAnimation(ctx, req)
	case "h3-control":
		return p.generateRestyle(ctx, req)
	case "remove-video-background":
		return p.generateBackgroundRemoval(ctx, req)
	case "video-dramatize":
		return p.generateDramatize(ctx, req)
	default:
		return p.generateKFoldVideo(ctx, req)
	}
}

// generateKFoldVideo drives h3_video — K-Fold/H3 cinematic generation with
// native audio, keyframes, and looping.
func (p *Provider) generateKFoldVideo(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	params := map[string]any{
		"prompt":       req.Prompt,
		"aspect_ratio": firstString(req.AspectRatio, "16:9"),
		"duration":     clamp(durationSeconds(req.Duration), 4, 60, 5),
	}
	if req.EndImageURL != "" {
		params["last_frame"] = req.EndImageURL
	}
	if req.ImageURL != "" {
		params["first_frame"] = req.ImageURL
	} else if req.EndImageURL != "" {
		params["first_frame"] = req.EndImageURL
	}
	if len(req.ImageURLs) > 0 {
		delete(params, "first_frame")
		params["keyframes"] = req.ImageURLs
	}
	if req.AudioURL != "" {
		params["audio_url"] = req.AudioURL
	}
	if req.Seed != nil && *req.Seed != 0 {
		params["seed"] = *req.Seed
	}
	if req.NumInferenceSteps > 0 {
		params["num_steps"] = clamp(req.NumInferenceSteps, 8, 30, 20)
	}
	if req.GenerateAudio != nil {
		params["include_audio"] = *req.GenerateAudio
	}
	switch strings.ToLower(req.OutputFormat) {
	case "mp4", "mp4-h264", "h264":
		params["output_format"] = "mp4-h264"
	case "webm", "webm-av1", "av1":
		params["output_format"] = "webm-av1"
	}

	result, err := p.submitAndResolve(ctx, "h3_video", params)
	if err != nil {
		return nil, err
	}
	videoURL := stringField(result, "video_url")
	if videoURL == "" {
		videoURL = stringField(result, "audio_url")
	}
	if videoURL == "" {
		return nil, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 502, Message: "no video_url in result: " + string(result), Retryable: false}
	}
	return &model.VideoGenerationResponse{VideoURL: videoURL, Model: req.Model, BackendUsed: p.Name()}, nil
}

// generateCharacterAnimation drives character_animation — Wan-Animate-2
// transfers motion from a driving video onto a reference character image.
func (p *Provider) generateCharacterAnimation(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	tier := "standard"
	switch req.Model {
	case "wan-animate-fast":
		tier = "fast"
	case "wan-animate-xfast":
		tier = "xfast"
	}

	imageURL := firstString(req.ImageURL)
	drivingVideo := firstString(req.VideoURL, videoInputURL(req.Video))
	if imageURL == "" {
		return nil, requestError("image_url is required: wan-animate animates a reference character image")
	}
	if drivingVideo == "" {
		return nil, requestError("video_url is required: wan-animate needs a driving performance video")
	}

	params := map[string]any{
		"prompt":       req.Prompt,
		"image_url":    imageURL,
		"video_url":    drivingVideo,
		"duration":     clamp(durationSeconds(req.Duration), 1, 8, 5),
		"service_tier": tier,
	}

	result, err := p.submitAndResolve(ctx, "character_animation", params)
	if err != nil {
		return nil, err
	}
	videoURL := stringField(result, "video_url")
	if videoURL == "" {
		return nil, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 502, Message: "no video_url in result: " + string(result), Retryable: false}
	}
	return &model.VideoGenerationResponse{VideoURL: videoURL, Model: req.Model, BackendUsed: p.Name()}, nil
}

// generateRestyle drives video_restyle with the H3-control lane: pose, depth,
// canny, hed, mlsd, or inpaint conditioning applied over a driving clip.
func (p *Provider) generateRestyle(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	controlType := strings.ToLower(firstString(req.ControlType))
	if controlType == "" {
		controlType = "pose"
	}
	drivingVideo := firstString(req.VideoURL, videoInputURL(req.Video))
	if drivingVideo == "" {
		return nil, requestError("video_url is required for h3-control restyle")
	}
	if controlType == "inpaint" && firstString(req.MaskVideoURL) == "" {
		return nil, requestError("mask_video_url is required for inpaint control")
	}

	params := map[string]any{
		"prompt":            req.Prompt,
		"model":             "h3-control",
		"video_url":         drivingVideo,
		"control_type":      controlType,
		"accept_h3_license": true,
		"duration":          clamp(durationSeconds(req.Duration), 1, 15, 5),
	}
	if r := strings.ToLower(req.Resolution); r != "" {
		params["resolution"] = r
	}
	if len(req.ImageURLs) > 0 {
		params["reference_image_urls"] = req.ImageURLs
	} else if req.ImageURL != "" {
		params["reference_image_urls"] = []string{req.ImageURL}
	}
	if req.MaskVideoURL != "" {
		params["mask_video_url"] = req.MaskVideoURL
	}
	if req.ControlScale != nil {
		params["control_scale"] = *req.ControlScale
	}
	if req.NumInferenceSteps > 0 {
		params["num_steps"] = clamp(req.NumInferenceSteps, 20, 50, 20)
	}

	result, err := p.submitAndResolve(ctx, "video_restyle", params)
	if err != nil {
		return nil, err
	}
	videoURL := stringField(result, "video_url")
	if videoURL == "" {
		return nil, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 502, Message: "no video_url in result: " + string(result), Retryable: false}
	}
	return &model.VideoGenerationResponse{VideoURL: videoURL, Model: req.Model, BackendUsed: p.Name()}, nil
}

// generateBackgroundRemoval drives video_background_removal: alpha-transparent
// WebM output keyed off the input clip.
func (p *Provider) generateBackgroundRemoval(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	drivingVideo := firstString(req.VideoURL, videoInputURL(req.Video))
	if drivingVideo == "" {
		return nil, requestError("video_url is required for background removal")
	}

	params := map[string]any{
		"video_url":      drivingVideo,
		"duration":       clamp(durationSeconds(req.Duration), 1, 30, 5),
		"output_format":  "webm_vp9",
		"preserve_audio": req.GenerateAudio == nil || *req.GenerateAudio,
	}
	if bg := strings.TrimSpace(req.BackgroundColor); bg != "" {
		params["background_color"] = bg
	} else {
		params["background_color"] = "transparent"
	}
	if req.MaskVideoURL != "" {
		params["mask_video_url"] = req.MaskVideoURL
	}

	result, err := p.submitAndResolve(ctx, "video_background_removal", params)
	if err != nil {
		return nil, err
	}
	videoURL := stringField(result, "video_url")
	if videoURL == "" {
		return nil, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 502, Message: "no video_url in result: " + string(result), Retryable: false}
	}
	resp := &model.VideoGenerationResponse{VideoURL: videoURL, Model: req.Model, BackendUsed: p.Name()}
	if ct := stringField(result, "content_type"); ct != "" {
		resp.OutputFormat = ct
	}
	return resp, nil
}

// generateDramatize drives video_dramatize — a long-running agent that turns
// one source clip plus a creative brief into a finished multi-shot edit.
// req.Duration is the planned total edit seconds the client pays for.
func (p *Provider) generateDramatize(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, requestError("prompt is required: video-dramatize plans shots from a creative brief")
	}
	drivingVideo := firstString(req.VideoURL, videoInputURL(req.Video))
	if drivingVideo == "" {
		return nil, requestError("video_url is required: video-dramatize edits a source clip")
	}
	if req.Duration == "" {
		return nil, requestError("duration is required: video-dramatize bills planned total edit seconds, 8..100")
	}

	total := clamp(durationSeconds(req.Duration), 8, 100, 30)
	maxShots := clamp((total+4)/5, 2, 10, 6)
	secondsPerShot := 5
	if len(req.GenerationConfig) > 0 {
		var cfg struct {
			MaxShots       int `json:"max_shots"`
			SecondsPerShot int `json:"seconds_per_shot"`
		}
		if err := json.Unmarshal(req.GenerationConfig, &cfg); err == nil {
			if cfg.MaxShots > 0 {
				maxShots = clamp(cfg.MaxShots, 2, 10, 6)
			}
			if cfg.SecondsPerShot > 0 {
				secondsPerShot = clamp(cfg.SecondsPerShot, 2, 10, 5)
			}
		}
	}

	params := map[string]any{
		"prompt":    req.Prompt,
		"video_url": drivingVideo,
		"max_shots": maxShots,
		"seconds":   secondsPerShot,
	}

	result, err := p.submitAndResolveDeadline(ctx, "video_dramatize", params, dramatizePollDeadline)
	if err != nil {
		return nil, err
	}
	videoURL := stringField(result, "video_url")
	if videoURL == "" {
		return nil, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 502, Message: "no video_url in result: " + string(result), Retryable: false}
	}
	return &model.VideoGenerationResponse{VideoURL: videoURL, Model: req.Model, BackendUsed: p.Name()}, nil
}

// --- Music / SFX ---

// GenerateMusic drives music_generation (MiniMax-Music3 full songs with
// vocals/lyrics) and sfx_generation (H3 text-to-audio). Both are queued jobs
// whose terminal result carries an audio_url.
func (p *Provider) GenerateMusic(ctx context.Context, req *model.MusicGenerationRequest) (*model.MusicGenerationResponse, error) {
	if req.Model == "mg-sfx" {
		return p.generateSFX(ctx, req)
	}

	prompt := strings.TrimSpace(req.Prompt)
	if len([]rune(prompt)) < 10 {
		return nil, requestError("manifoldgen music requires a prompt of at least 10 characters describing the track")
	}
	params := map[string]any{"prompt": prompt}
	if lyrics := strings.TrimSpace(req.Lyrics); lyrics != "" {
		params["lyrics"] = lyrics
	}
	if req.Duration > 0 {
		params["duration"] = clamp(req.Duration, 30, 300, 60)
	}

	result, err := p.submitAndResolve(ctx, "music_generation", params)
	if err != nil {
		return nil, err
	}
	audioURL := stringField(result, "audio_url")
	if audioURL == "" {
		audioURL = stringField(result, "video_url")
	}
	if audioURL == "" {
		return nil, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 502, Message: "no audio_url in result: " + string(result), Retryable: false}
	}
	resp := &model.MusicGenerationResponse{Data: &model.MusicData{Audio: audioURL}}
	if d := intValue(result, "duration_seconds"); d > 0 {
		resp.ExtraInfo = &model.MusicExtraInfo{Duration: d}
	}
	return resp, nil
}

func (p *Provider) generateSFX(ctx context.Context, req *model.MusicGenerationRequest) (*model.MusicGenerationResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, requestError("prompt is required for sound effects")
	}
	params := map[string]any{
		"prompt": prompt,
		"size":   "audio",
		"kind":   "sfx",
	}
	if req.AudioSetting != nil && req.AudioSetting.Format != "" {
		params["output_format"] = req.AudioSetting.Format
	}

	result, err := p.submitAndResolve(ctx, "sfx_generation", params)
	if err != nil {
		return nil, err
	}
	audioURL := stringField(result, "audio_url")
	if audioURL == "" {
		audioURL = stringField(result, "video_url")
	}
	if audioURL == "" {
		return nil, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 502, Message: "no audio_url in result: " + string(result), Retryable: false}
	}
	resp := &model.MusicGenerationResponse{Data: &model.MusicData{Audio: audioURL}}
	if d := intValue(result, "duration_seconds"); d > 0 {
		resp.ExtraInfo = &model.MusicExtraInfo{Duration: d}
	}
	return resp, nil
}

// --- Speech ---

// GenerateSpeech drives the tts service. The upstream responds synchronously
// with base64 audio inside result.audio_base64.
func (p *Provider) GenerateSpeech(ctx context.Context, req *model.SpeechRequest) (*model.SpeechResponse, error) {
	text := firstString(req.Input, req.Text)
	if text == "" {
		return nil, requestError("input is required for speech")
	}
	params := map[string]any{
		"text":     text,
		"language": firstString(req.Language, "en"),
	}
	if voice := firstString(req.Voice, req.VoiceID); voice != "" {
		params["voice"] = voice
	}
	if req.Speed > 0 {
		params["speed"] = req.Speed
	}

	env, err := p.callService(ctx, "tts", params)
	if err != nil {
		return nil, err
	}
	audio := stringField(env.Result, "audio_base64")
	if audio == "" {
		if url := stringField(env.Result, "audio_url"); url != "" {
			return &model.SpeechResponse{AudioURL: url, Format: stringField(env.Result, "format")}, nil
		}
		return nil, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 502, Message: "no audio in tts result: " + string(env.Result), Retryable: false}
	}
	format := stringField(env.Result, "format")
	if format == "" {
		format = "wav"
	}
	return &model.SpeechResponse{Audio: audio, Format: format}, nil
}

// --- Images ---

func videoInputURL(video *model.VideoInput) string {
	if video == nil {
		return ""
	}
	return video.URL
}

// GenerateImage drives h3_image (H3 diffusion stills) and h3_image_edit
// (source-image edits with up to 8 reference images). Both are queued jobs
// whose terminal result carries an image_url.
func (p *Provider) GenerateImage(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	service := "h3_image"
	params := map[string]any{"prompt": req.Prompt}

	if req.Model == "h3-image-edit" {
		service = "h3_image_edit"
		source := firstString(req.ImageURL)
		if source == "" && len(req.ImageURLs) > 0 {
			source = req.ImageURLs[0]
		}
		if source == "" && req.Image != nil {
			source = req.Image.URL
		}
		if source == "" {
			return nil, requestError("image_url is required for h3-image-edit")
		}
		params["image_url"] = source
		if refs := nonEmpty(req.ReferenceImageURLs); len(refs) > 0 {
			params["reference_image_urls"] = refs
		}
	} else if req.ImageURL != "" {
		// h3_image accepts an optional init/reference image through image_url.
		params["image_url"] = req.ImageURL
	}

	if w, h, ok := parseSize(req.Size); ok {
		params["width"] = w
		params["height"] = h
	}
	if req.NumInferenceSteps > 0 {
		params["num_steps"] = req.NumInferenceSteps
	}
	if req.Seed != nil {
		params["seed"] = *req.Seed
	}

	result, err := p.submitAndResolve(ctx, service, params)
	if err != nil {
		return nil, err
	}
	imageURL := stringField(result, "image_url")
	if imageURL == "" {
		imageURL = stringField(result, "url")
	}
	if imageURL == "" {
		return nil, &provider.ProviderError{Provider: "manifoldgen", StatusCode: 502, Message: "no image_url in result: " + string(result), Retryable: false}
	}
	return &model.ImageGenerationResponse{
		Created: time.Now().Unix(),
		Data:    []model.ImageData{{URL: imageURL}},
	}, nil
}

// submitAndResolve posts one service call and returns its final artifact
// payload, following async job handles to completion.
func (p *Provider) submitAndResolve(ctx context.Context, service string, params map[string]any) (json.RawMessage, error) {
	env, err := p.callService(ctx, service, params)
	if err != nil {
		return nil, err
	}
	return p.resolveResult(ctx, env, pollDeadline)
}

// submitAndResolveDeadline is submitAndResolve with a custom async poll
// budget for long-running services.
func (p *Provider) submitAndResolveDeadline(ctx context.Context, service string, params map[string]any, jobDeadline time.Duration) (json.RawMessage, error) {
	env, err := p.callService(ctx, service, params)
	if err != nil {
		return nil, err
	}
	return p.resolveResult(ctx, env, jobDeadline)
}

func stringField(raw json.RawMessage, key string) string {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func intValue(raw json.RawMessage, key string) int {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return 0
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func nonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// parseSize accepts "WxH" size strings; anything else falls back to upstream
// defaults by omitting width/height entirely.
func parseSize(size string) (int, int, bool) {
	size = strings.ToLower(strings.TrimSpace(size))
	parts := strings.SplitN(size, "x", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, false
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, false
	}
	if w <= 0 || h <= 0 {
		return 0, 0, false
	}
	return w, h, true
}
