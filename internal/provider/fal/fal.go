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
	endpoint := strings.TrimRight(p.baseURL, "/") + "/" + req.Model

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
	if outputFormat, syncMode := falOutputMode(req); outputFormat != "" {
		falReq["output_format"] = outputFormat
		if syncMode {
			falReq["sync_mode"] = true
		}
	}

	body, err := json.Marshal(falReq)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

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

func falOutputMode(req *model.ImageGenerationRequest) (outputFormat string, syncMode bool) {
	respFormat := strings.ToLower(strings.TrimSpace(req.ResponseFormat))
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
