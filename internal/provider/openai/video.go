package openai

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
)

type soraCreateReq struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Size    string `json:"size,omitempty"`
	Seconds string `json:"seconds,omitempty"`
}

type soraVideoResp struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// GenerateVideo submits a Sora video job and polls until complete.
// It returns the signed content URL (requires the OpenAI bearer token to fetch).
func (p *OpenAIProvider) GenerateVideo(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error) {
	size := req.Resolution
	if size == "" {
		size = "1280x720"
	}
	seconds := ""
	if req.NumFrames > 0 && req.FramesPerSecond > 0 {
		secs := req.NumFrames / req.FramesPerSecond
		if secs > 0 {
			seconds = fmt.Sprintf("%d", secs)
		}
	}

	payload := soraCreateReq{
		Model:   req.Model,
		Prompt:  req.Prompt,
		Size:    size,
		Seconds: seconds,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/videos", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "openai", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, &provider.ProviderError{
			Provider:   "openai",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var submit soraVideoResp
	if err := json.Unmarshal(respBody, &submit); err != nil {
		return nil, fmt.Errorf("unmarshal submit: %w", err)
	}
	if submit.ID == "" {
		return nil, &provider.ProviderError{Provider: "openai", StatusCode: 502, Message: "no video id returned", Retryable: true}
	}

	videoID, err := p.pollSoraVideo(ctx, submit.ID)
	if err != nil {
		return nil, err
	}

	contentURL := fmt.Sprintf("%s/v1/videos/%s/content", p.baseURL, videoID)
	return &model.VideoGenerationResponse{
		VideoURL: contentURL,
		Model:    req.Model,
	}, nil
}

func (p *OpenAIProvider) pollSoraVideo(ctx context.Context, videoID string) (string, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	timeout := time.After(15 * time.Minute)

	url := fmt.Sprintf("%s/v1/videos/%s", p.baseURL, videoID)
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", &provider.ProviderError{Provider: "openai", StatusCode: 504, Message: "sora timed out", Retryable: false}
		case <-ticker.C:
			httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return "", err
			}
			httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

			resp, err := p.client.Do(httpReq)
			if err != nil {
				log.Printf("openai: sora poll error: %v", err)
				continue
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			var status soraVideoResp
			if err := json.Unmarshal(body, &status); err != nil {
				continue
			}
			switch status.Status {
			case "completed", "succeeded":
				return videoID, nil
			case "failed", "cancelled":
				msg := "sora generation failed"
				if status.Error != nil && status.Error.Message != "" {
					msg = status.Error.Message
				}
				return "", &provider.ProviderError{Provider: "openai", StatusCode: 502, Message: msg, Retryable: false}
			}
		}
	}
}
