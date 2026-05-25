package textgenerator

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

type TextGeneratorProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func New(apiKey string, baseURL ...string) *TextGeneratorProvider {
	resolvedBaseURL := "https://api.text-generator.io"
	if len(baseURL) > 0 && strings.TrimSpace(baseURL[0]) != "" {
		resolvedBaseURL = strings.TrimSpace(baseURL[0])
	}

	return &TextGeneratorProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(resolvedBaseURL, "/"),
		client:  &http.Client{Timeout: 2 * time.Minute},
	}
}

func (p *TextGeneratorProvider) Name() string { return "textgenerator" }

func (p *TextGeneratorProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	return nil, &provider.ProviderError{
		Provider:   p.Name(),
		StatusCode: 400,
		Message:    "textgenerator provider does not support chat completions",
	}
}

func (p *TextGeneratorProvider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	return nil, &provider.ProviderError{
		Provider:   p.Name(),
		StatusCode: 400,
		Message:    "textgenerator provider does not support streaming chat completions",
	}
}

func (p *TextGeneratorProvider) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/liveness_check", nil)
	if err != nil {
		return fmt.Errorf("create health check request: %w", err)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}
	return nil
}

type featureRequest struct {
	Text        string `json:"text"`
	NumFeatures int    `json:"num_features"`
}

type speechRequest struct {
	Text     string  `json:"text"`
	Voice    string  `json:"voice"`
	Language string  `json:"language"`
	Speed    float64 `json:"speed"`
	Steps    int     `json:"steps"`
}

func (p *TextGeneratorProvider) Embed(ctx context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	inputs, err := normalizeInput(req.Input)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "textgenerator", StatusCode: 400, Message: err.Error(),
		}
	}

	dims := req.Dimensions
	if dims <= 0 {
		dims = 768
	}

	data := make([]model.EmbeddingData, 0, len(inputs))
	totalTokens := 0

	for i, text := range inputs {
		embedding, err := p.fetchEmbedding(ctx, text, dims)
		if err != nil {
			return nil, err
		}
		data = append(data, model.EmbeddingData{
			Object:    "embedding",
			Embedding: embedding,
			Index:     i,
		})
		totalTokens += len(text) / 4
	}

	return &model.EmbeddingResponse{
		Object: "list",
		Data:   data,
		Model:  req.Model,
		Usage: model.EmbeddingUsage{
			PromptTokens: totalTokens,
			TotalTokens:  totalTokens,
		},
	}, nil
}

func (p *TextGeneratorProvider) fetchEmbedding(ctx context.Context, text string, dims int) ([]float64, error) {
	body, _ := json.Marshal(featureRequest{Text: text, NumFeatures: dims})

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		p.baseURL+"/api/v1/feature-extraction", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("secret", p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "textgenerator", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "textgenerator",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var embedding []float64
	if err := json.Unmarshal(respBody, &embedding); err != nil {
		return nil, fmt.Errorf("unmarshal embedding: %w", err)
	}
	return embedding, nil
}

func (p *TextGeneratorProvider) GenerateSpeech(ctx context.Context, req *model.SpeechRequest) (*model.SpeechResponse, error) {
	text := req.Input
	if text == "" {
		text = req.Text
	}
	if strings.TrimSpace(text) == "" {
		return nil, &provider.ProviderError{
			Provider:   p.Name(),
			StatusCode: 400,
			Message:    "input is required",
		}
	}

	voice := req.Voice
	if voice == "" {
		voice = req.VoiceID
	}
	if voice == "" {
		voice = "M1"
	}

	language := req.Language
	if language == "" {
		language = "en"
	}

	speed := req.Speed
	if speed <= 0 {
		speed = 1
	}

	body, err := json.Marshal(speechRequest{
		Text:     text,
		Voice:    voice,
		Language: language,
		Speed:    speed,
		Steps:    4,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal speech request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/v1/generate_speech", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create speech request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("secret", p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: p.Name(), StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read speech response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &provider.ProviderError{
			Provider:   p.Name(),
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests,
		}
	}

	return &model.SpeechResponse{
		Audio:      base64.StdEncoding.EncodeToString(respBody),
		Format:     speechFormat(resp.Header.Get("Content-Type")),
		Characters: len(text),
	}, nil
}

func speechFormat(contentType string) string {
	contentType = strings.ToLower(contentType)
	switch {
	case strings.Contains(contentType, "mpeg"), strings.Contains(contentType, "mp3"):
		return "mp3"
	case strings.Contains(contentType, "ogg"):
		return "ogg"
	case strings.Contains(contentType, "webm"):
		return "webm"
	case strings.Contains(contentType, "wav"), strings.Contains(contentType, "wave"):
		return "wav"
	default:
		return "wav"
	}
}

func normalizeInput(input any) ([]string, error) {
	switch v := input.(type) {
	case string:
		return []string{v}, nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("input array must contain strings")
			}
			out = append(out, s)
		}
		return out, nil
	case []string:
		return v, nil
	default:
		return nil, fmt.Errorf("input must be a string or array of strings")
	}
}
