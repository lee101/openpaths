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

	"github.com/openpath/openpath/internal/model"
	"github.com/openpath/openpath/internal/provider"
)

type FalProvider struct {
	apiKey string
	client *http.Client
}

func New(apiKey string) *FalProvider {
	return &FalProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 2 * time.Minute},
	}
}

func (p *FalProvider) Name() string { return "fal" }

type falRequest struct {
	AudioURL   string `json:"audio_url"`
	Task       string `json:"task"`
	ChunkLevel string `json:"chunk_level"`
}

type falResponse struct {
	Text   string    `json:"text"`
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
