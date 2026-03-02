package minimax

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/provider/openai"
)

const baseURL = "https://api.minimax.io"

type Provider struct {
	*openai.OpenAIProvider
	apiKey string
	client *http.Client
}

func New(apiKey string) *Provider {
	return &Provider{
		OpenAIProvider: openai.New(apiKey, baseURL),
		apiKey:         apiKey,
		client:         &http.Client{Timeout: 5 * time.Minute},
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

	resp, err := p.doJSON(ctx, "POST", baseURL+"/v1/video_generation", body)
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
			url := fmt.Sprintf("%s/v1/query/video_generation?task_id=%s", baseURL, taskID)
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
	url := fmt.Sprintf("%s/v1/files/retrieve?file_id=%s", baseURL, fileID)
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
