package minimax

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/provider/openai"
)

const baseURL = "https://api.minimax.io"

type Provider struct {
	*openai.OpenAIProvider
	apiKey       string
	client       *http.Client
	baseURL      string
	pollInterval time.Duration
}

func New(apiKey string) *Provider {
	return &Provider{
		OpenAIProvider: openai.NewRequiringUserTurn("minimax", apiKey, baseURL),
		apiKey:         apiKey,
		client:         &http.Client{Timeout: 5 * time.Minute},
		baseURL:        baseURL,
		pollInterval:   5 * time.Second,
	}
}

func (p *Provider) Name() string { return "minimax" }

// --- Music Generation ---

func (p *Provider) GenerateMusic(ctx context.Context, req *model.MusicGenerationRequest) (*model.MusicGenerationResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	resp, err := p.doJSON(ctx, "POST", baseURL+"/v1/music_generation", body)
	if err != nil {
		return nil, err
	}

	var result model.MusicGenerationResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	if result.BaseResp != nil && result.BaseResp.StatusCode != 0 {
		return nil, &provider.ProviderError{
			Provider:   "minimax",
			StatusCode: mapStatusCode(result.BaseResp.StatusCode),
			Message:    result.BaseResp.StatusMsg,
			Retryable:  result.BaseResp.StatusCode == 1002,
		}
	}
	return &result, nil
}

// --- Video Generation (async: submit -> poll -> retrieve URL) ---

func (p *Provider) GenerateVideo(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	if req.Model == "MiniMax-H3" {
		return p.generateVideoV2(ctx, req)
	}
	mmReq := minimaxVideoReq{
		Model:  req.Model,
		Prompt: req.Prompt,
	}
	if req.ImageURL != "" {
		mmReq.FirstFrameImage = req.ImageURL
	}
	if req.Resolution != "" {
		mmReq.Resolution = req.Resolution
	} else {
		mmReq.Resolution = "1080P"
	}
	dur := 6
	if req.NumFrames > 0 && req.FramesPerSecond > 0 {
		dur = req.NumFrames / req.FramesPerSecond
	}
	if dur != 6 && dur != 10 {
		dur = 6
	}
	mmReq.Duration = dur

	body, err := json.Marshal(mmReq)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	resp, err := p.doJSON(ctx, "POST", p.baseURL+"/v1/video_generation", body)
	if err != nil {
		return nil, err
	}

	var submitResp videoSubmitResp
	if err := json.Unmarshal(resp, &submitResp); err != nil {
		return nil, fmt.Errorf("unmarshal submit: %w", err)
	}
	if err := checkBaseResp(submitResp.BaseResp); err != nil {
		return nil, err
	}
	if submitResp.TaskID == "" {
		return nil, &provider.ProviderError{Provider: "minimax", StatusCode: 502, Message: "no task_id returned", Retryable: true}
	}

	fileID, err := p.pollVideoTask(ctx, submitResp.TaskID)
	if err != nil {
		return nil, err
	}

	downloadURL, err := p.retrieveFile(ctx, fileID)
	if err != nil {
		return nil, err
	}

	return &model.VideoGenerationResponse{
		VideoURL: downloadURL,
		Model:    req.Model,
	}, nil
}

func (p *Provider) generateVideoV2(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	mmReq, err := buildVideoV2Request(req)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(mmReq)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	resp, err := p.doJSON(ctx, "POST", p.baseURL+"/v2/video_generation", body)
	if err != nil {
		return nil, err
	}
	var submit videoV2SubmitResp
	if err := json.Unmarshal(resp, &submit); err != nil {
		return nil, fmt.Errorf("unmarshal submit: %w", err)
	}
	if submit.TaskID == "" {
		return nil, &provider.ProviderError{Provider: "minimax", StatusCode: 502, Message: "no task_id returned", Retryable: true}
	}
	videoURL, err := p.pollVideoTaskV2(ctx, submit.TaskID)
	if err != nil {
		return nil, err
	}
	return &model.VideoGenerationResponse{VideoURL: videoURL, Model: req.Model}, nil
}

func buildVideoV2Request(req *model.VideoGenerationRequest) (*minimaxVideoV2Req, error) {
	content := append([]model.VideoContentItem(nil), req.Content...)
	if len(content) == 0 {
		content = append(content, model.VideoContentItem{Type: "text", Text: req.Prompt})
		if req.ImageURL != "" {
			content = append(content, videoImageItem(req.ImageURL, "first_frame"))
		}
		if req.EndImageURL != "" {
			content = append(content, videoImageItem(req.EndImageURL, "last_frame"))
		}
		for _, url := range req.ImageURLs {
			content = append(content, videoImageItem(url, "reference_image"))
		}
		videoURLs := append([]string(nil), req.VideoURLs...)
		if req.VideoURL != "" {
			videoURLs = append([]string{req.VideoURL}, videoURLs...)
		} else if req.Video != nil && req.Video.URL != "" {
			videoURLs = append([]string{req.Video.URL}, videoURLs...)
		}
		for _, url := range videoURLs {
			content = append(content, model.VideoContentItem{Type: "video_url", VideoURL: &model.VideoMediaURL{URL: url}, Role: "reference_video"})
		}
		audioURLs := append([]string(nil), req.AudioURLs...)
		if req.AudioURL != "" {
			audioURLs = append([]string{req.AudioURL}, audioURLs...)
		}
		for _, url := range audioURLs {
			content = append(content, model.VideoContentItem{Type: "audio_url", AudioURL: &model.VideoMediaURL{URL: url}, Role: "reference_audio"})
		}
	}

	mode, err := validateVideoV2Content(content)
	if err != nil {
		return nil, videoV2RequestError(err.Error())
	}
	duration, err := videoV2Duration(req)
	if err != nil {
		return nil, videoV2RequestError(err.Error())
	}
	if duration < 4 || duration > 15 {
		return nil, videoV2RequestError("duration must be between 4 and 15 seconds")
	}
	req.Duration = model.VideoDuration(strconv.Itoa(duration))
	resolution := strings.ToUpper(strings.TrimSpace(req.Resolution))
	if resolution == "" {
		resolution = "2K"
	}
	if resolution != "768P" && resolution != "2K" {
		return nil, videoV2RequestError("resolution must be 768P or 2K")
	}
	req.Resolution = resolution
	ratioValue := req.AspectRatio
	if ratioValue == "" {
		ratioValue = req.Ratio
	}
	ratio := strings.ToLower(strings.TrimSpace(ratioValue))
	if ratio == "auto" || ratio == "" {
		ratio = "adaptive"
	}
	if mode == "image" {
		ratio = "adaptive"
	} else if mode == "text" && ratio == "adaptive" {
		ratio = "16:9"
	}
	if !validVideoV2Ratio(ratio) {
		return nil, videoV2RequestError("aspect_ratio must be adaptive, 21:9, 16:9, 4:3, 1:1, 3:4, or 9:16")
	}

	return &minimaxVideoV2Req{Model: req.Model, Content: content, Resolution: resolution, Duration: duration, Ratio: ratio}, nil
}

func videoImageItem(url, role string) model.VideoContentItem {
	return model.VideoContentItem{Type: "image_url", ImageURL: &model.VideoMediaURL{URL: url}, Role: role}
}

func videoV2Duration(req *model.VideoGenerationRequest) (int, error) {
	if req.Duration != "" && req.Duration != "auto" {
		if n, err := strconv.Atoi(string(req.Duration)); err == nil {
			return n, nil
		}
		return 0, fmt.Errorf("duration must be an integer between 4 and 15")
	}
	if req.NumFrames > 0 && req.FramesPerSecond > 0 {
		return req.NumFrames / req.FramesPerSecond, nil
	}
	return 5, nil
}

func validateVideoV2Content(content []model.VideoContentItem) (string, error) {
	textCount, firstCount, lastCount, refImages, refVideos, refAudio := 0, 0, 0, 0, 0, 0
	for _, item := range content {
		switch item.Type {
		case "text":
			if strings.TrimSpace(item.Text) != "" {
				textCount++
			}
		case "image_url":
			if item.ImageURL == nil || item.ImageURL.URL == "" {
				return "", fmt.Errorf("image_url content requires image_url.url")
			}
			switch item.Role {
			case "", "first_frame":
				firstCount++
			case "last_frame":
				lastCount++
			case "reference_image":
				refImages++
			default:
				return "", fmt.Errorf("invalid role %q for image_url content", item.Role)
			}
		case "video_url":
			if item.VideoURL == nil || item.VideoURL.URL == "" || item.Role != "reference_video" {
				return "", fmt.Errorf("video_url content requires video_url.url and role=reference_video")
			}
			refVideos++
		case "audio_url":
			if item.AudioURL == nil || item.AudioURL.URL == "" || item.Role != "reference_audio" {
				return "", fmt.Errorf("audio_url content requires audio_url.url and role=reference_audio")
			}
			refAudio++
		default:
			return "", fmt.Errorf("unsupported content type %q", item.Type)
		}
	}
	if textCount == 0 {
		return "", fmt.Errorf("content must include a non-empty text item")
	}
	if textCount > 1 {
		return "", fmt.Errorf("content may include only one text item")
	}
	if firstCount > 1 || lastCount > 1 || refImages > 9 || refVideos > 3 || refAudio > 3 {
		return "", fmt.Errorf("content exceeds MiniMax-H3 media count limits")
	}
	hasFrames := firstCount+lastCount > 0
	hasReferences := refImages+refVideos+refAudio > 0
	if hasFrames && hasReferences {
		return "", fmt.Errorf("frame and reference inputs cannot be mixed")
	}
	if refAudio > 0 && refImages+refVideos == 0 {
		return "", fmt.Errorf("reference audio requires at least one reference image or video")
	}
	if hasFrames {
		return "image", nil
	}
	if hasReferences {
		return "reference", nil
	}
	return "text", nil
}

func validVideoV2Ratio(ratio string) bool {
	switch ratio {
	case "adaptive", "21:9", "16:9", "4:3", "1:1", "3:4", "9:16":
		return true
	default:
		return false
	}
}

func videoV2RequestError(message string) error {
	return &provider.ProviderError{Provider: "minimax", StatusCode: 400, Message: message, Retryable: false}
}

func (p *Provider) pollVideoTaskV2(ctx context.Context, taskID string) (string, error) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()
	timeout := time.NewTimer(10 * time.Minute)
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout.C:
			return "", &provider.ProviderError{Provider: "minimax", StatusCode: 504, Message: "video generation timed out", Retryable: false}
		case <-ticker.C:
			url := fmt.Sprintf("%s/v2/query/video_generation/%s", p.baseURL, taskID)
			resp, err := p.doGET(ctx, url)
			if err != nil {
				log.Printf("minimax: V2 poll error for task %s: %v", taskID, err)
				continue
			}
			var result videoV2QueryResp
			if err := json.Unmarshal(resp, &result); err != nil {
				continue
			}
			switch result.Task.Status {
			case "succeeded":
				if result.Task.Content.URL == "" {
					return "", &provider.ProviderError{Provider: "minimax", StatusCode: 502, Message: "success but no video URL", Retryable: true}
				}
				return result.Task.Content.URL, nil
			case "failed", "cancelled":
				message := result.Task.Error.Message
				if message == "" {
					message = "video generation " + result.Task.Status
				}
				return "", &provider.ProviderError{Provider: "minimax", StatusCode: 502, Message: message, Retryable: false}
			}
		}
	}
}

func (p *Provider) pollVideoTask(ctx context.Context, taskID string) (string, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	timeout := time.After(10 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", &provider.ProviderError{Provider: "minimax", StatusCode: 504, Message: "video generation timed out", Retryable: false}
		case <-ticker.C:
			url := fmt.Sprintf("%s/v1/query/video_generation?task_id=%s", p.baseURL, taskID)
			resp, err := p.doGET(ctx, url)
			if err != nil {
				log.Printf("minimax: poll error for task %s: %v", taskID, err)
				continue
			}

			var status videoQueryResp
			if err := json.Unmarshal(resp, &status); err != nil {
				continue
			}

			switch status.Status {
			case "Success":
				if status.FileID == "" {
					return "", &provider.ProviderError{Provider: "minimax", StatusCode: 502, Message: "success but no file_id", Retryable: true}
				}
				return status.FileID, nil
			case "Fail":
				msg := "video generation failed"
				if status.BaseResp != nil {
					msg = status.BaseResp.StatusMsg
				}
				return "", &provider.ProviderError{Provider: "minimax", StatusCode: 502, Message: msg, Retryable: false}
			}
		}
	}
}

func (p *Provider) retrieveFile(ctx context.Context, fileID string) (string, error) {
	url := fmt.Sprintf("%s/v1/files/retrieve?file_id=%s", p.baseURL, fileID)
	resp, err := p.doGET(ctx, url)
	if err != nil {
		return "", err
	}

	var result fileRetrieveResp
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("unmarshal file: %w", err)
	}
	if result.File.DownloadURL == "" {
		return "", &provider.ProviderError{Provider: "minimax", StatusCode: 502, Message: "no download_url", Retryable: true}
	}
	return result.File.DownloadURL, nil
}

// --- Speech/TTS ---

func (p *Provider) GenerateSpeech(ctx context.Context, req *model.SpeechRequest) (*model.SpeechResponse, error) {
	mmReq := minimaxTTSReq{
		Model: req.Model,
		Text:  req.Input,
		VoiceSetting: &voiceSetting{
			VoiceID: req.Voice,
			Speed:   req.Speed,
		},
		AudioSetting: &audioSetting{
			Format: "mp3",
		},
		OutputFormat: "url",
	}
	if mmReq.VoiceSetting.VoiceID == "" {
		mmReq.VoiceSetting.VoiceID = "male-qn-qingse"
	}
	if mmReq.VoiceSetting.Speed <= 0 {
		mmReq.VoiceSetting.Speed = 1.0
	}

	body, err := json.Marshal(mmReq)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	resp, err := p.doJSON(ctx, "POST", baseURL+"/v1/t2a_v2", body)
	if err != nil {
		return nil, err
	}

	var result minimaxTTSResp
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	if err := checkBaseResp(result.BaseResp); err != nil {
		return nil, err
	}

	return &model.SpeechResponse{
		Audio:      result.Data.Audio,
		AudioURL:   result.Data.AudioURL,
		Format:     "mp3",
		DurationMs: result.ExtraInfo.AudioLength,
		Characters: result.ExtraInfo.UsageChars,
	}, nil
}

// --- HTTP helpers ---

func (p *Provider) doJSON(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "minimax", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "minimax",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}
	return respBody, nil
}

func (p *Provider) doGET(ctx context.Context, url string) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "minimax", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "minimax",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}
	return respBody, nil
}

// --- Internal types ---

type minimaxBaseResp struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

func checkBaseResp(br *minimaxBaseResp) error {
	if br == nil || br.StatusCode == 0 {
		return nil
	}
	return &provider.ProviderError{
		Provider:   "minimax",
		StatusCode: mapStatusCode(br.StatusCode),
		Message:    br.StatusMsg,
		Retryable:  br.StatusCode == 1002,
	}
}

func mapStatusCode(code int) int {
	switch code {
	case 1002:
		return 429
	case 1004, 2049:
		return 401
	case 1008:
		return 402
	case 1026:
		return 451
	case 2013:
		return 400
	default:
		return 502
	}
}

// video types

type minimaxVideoReq struct {
	Model           string `json:"model"`
	Prompt          string `json:"prompt"`
	FirstFrameImage string `json:"first_frame_image,omitempty"`
	Duration        int    `json:"duration,omitempty"`
	Resolution      string `json:"resolution,omitempty"`
}

type minimaxVideoV2Req struct {
	Model      string                   `json:"model"`
	Content    []model.VideoContentItem `json:"content"`
	Resolution string                   `json:"resolution"`
	Duration   int                      `json:"duration"`
	Ratio      string                   `json:"ratio"`
}

type videoV2SubmitResp struct {
	TaskID string `json:"task_id"`
}

type videoV2QueryResp struct {
	Task struct {
		Status  string `json:"status"`
		Content struct {
			URL string `json:"url"`
		} `json:"content"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"task"`
}

type videoSubmitResp struct {
	TaskID   string           `json:"task_id"`
	BaseResp *minimaxBaseResp `json:"base_resp"`
}

type videoQueryResp struct {
	TaskID   string           `json:"task_id"`
	Status   string           `json:"status"`
	FileID   string           `json:"file_id"`
	BaseResp *minimaxBaseResp `json:"base_resp"`
}

type fileRetrieveResp struct {
	File struct {
		FileID      int64  `json:"file_id"`
		DownloadURL string `json:"download_url"`
	} `json:"file"`
	BaseResp *minimaxBaseResp `json:"base_resp"`
}

// TTS types

type voiceSetting struct {
	VoiceID string  `json:"voice_id"`
	Speed   float64 `json:"speed,omitempty"`
}

type audioSetting struct {
	SampleRate int    `json:"sample_rate,omitempty"`
	Bitrate    int    `json:"bitrate,omitempty"`
	Format     string `json:"format,omitempty"`
}

type minimaxTTSReq struct {
	Model        string        `json:"model"`
	Text         string        `json:"text"`
	VoiceSetting *voiceSetting `json:"voice_setting"`
	AudioSetting *audioSetting `json:"audio_setting,omitempty"`
	OutputFormat string        `json:"output_format,omitempty"`
}

type minimaxTTSResp struct {
	Data struct {
		Audio    string `json:"audio"`
		AudioURL string `json:"audio_url"`
	} `json:"data"`
	ExtraInfo struct {
		AudioLength int `json:"audio_length"`
		UsageChars  int `json:"usage_characters"`
	} `json:"extra_info"`
	BaseResp *minimaxBaseResp `json:"base_resp"`
}
