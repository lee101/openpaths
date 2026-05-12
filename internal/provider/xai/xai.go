package xai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/provider/openai"
)

type XAIProvider struct {
	*openai.OpenAIProvider
	apiKey  string
	baseURL string
	client  *http.Client
}

func New(apiKey, baseURL string) *XAIProvider {
	if baseURL == "" {
		baseURL = "https://api.x.ai"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &XAIProvider{
		OpenAIProvider: openai.New(apiKey, baseURL),
		apiKey:         apiKey,
		baseURL:        baseURL,
		client:         &http.Client{Timeout: 2 * time.Minute},
	}
}

func (p *XAIProvider) Name() string { return "xai" }

func (p *XAIProvider) GenerateImage(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	payload := map[string]any{
		"model":  req.Model,
		"prompt": req.Prompt,
	}
	if req.N > 0 {
		payload["n"] = req.N
	}
	if req.AspectRatio != "" {
		payload["aspect_ratio"] = req.AspectRatio
	} else if aspectRatio := sizeToAspectRatio(req.Size); aspectRatio != "" {
		payload["aspect_ratio"] = aspectRatio
	}

	path := "/v1/images/generations"
	inputs := normalizeImageInputs(req)
	if len(inputs) == 1 {
		path = "/v1/images/edits"
		payload["image"] = inputs[0]
	} else if len(inputs) > 1 {
		path = "/v1/images/edits"
		payload["images"] = inputs
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	respBody, err := p.do(ctx, "POST", path, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var result model.ImageGenerationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &result, nil
}

func (p *XAIProvider) GenerateSpeech(ctx context.Context, req *model.SpeechRequest) (*model.SpeechResponse, error) {
	text := req.Input
	if text == "" {
		text = req.Text
	}
	voice := req.Voice
	if voice == "" {
		voice = req.VoiceID
	}
	if voice == "" {
		voice = "eve"
	}
	language := req.Language
	if language == "" {
		language = "en"
	}

	body, err := json.Marshal(map[string]any{
		"text":     text,
		"voice_id": voice,
		"language": language,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	respBody, err := p.do(ctx, "POST", "/v1/tts", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	return &model.SpeechResponse{
		Audio:      base64.StdEncoding.EncodeToString(respBody),
		Format:     "mp3",
		Characters: len(text),
	}, nil
}

func normalizeImageInputs(req *model.ImageGenerationRequest) []model.ImageInput {
	var out []model.ImageInput
	if req.Image != nil {
		out = append(out, normalizeImageInput(*req.Image))
	}
	for _, image := range req.Images {
		out = append(out, normalizeImageInput(image))
	}
	if req.ImageURL != "" {
		out = append(out, model.ImageInput{Type: "image_url", URL: req.ImageURL})
	}
	for _, url := range req.ImageURLs {
		if url != "" {
			out = append(out, model.ImageInput{Type: "image_url", URL: url})
		}
	}
	return out
}

func normalizeImageInput(input model.ImageInput) model.ImageInput {
	if input.Type == "" {
		input.Type = "image_url"
	}
	return input
}

func sizeToAspectRatio(size string) string {
	parts := strings.SplitN(strings.ToLower(size), "x", 2)
	if len(parts) != 2 {
		return ""
	}
	var w, h int
	if _, err := fmt.Sscanf(parts[0], "%d", &w); err != nil {
		return ""
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &h); err != nil {
		return ""
	}
	if w <= 0 || h <= 0 {
		return ""
	}
	g := gcd(w, h)
	return fmt.Sprintf("%d:%d", w/g, h/g)
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	return a
}

func (p *XAIProvider) Transcribe(ctx context.Context, req *model.TranscriptionRequest) (*model.TranscriptionResponse, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if req.Language != "" {
		_ = w.WriteField("format", "true")
		_ = w.WriteField("language", req.Language)
	}

	fw, err := w.CreateFormFile("file", req.Filename)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := fw.Write(req.File); err != nil {
		return nil, fmt.Errorf("write file: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	respBody, err := p.do(ctx, "POST", "/v1/stt", w.FormDataContentType(), &buf)
	if err != nil {
		return nil, err
	}

	var result model.TranscriptionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, &provider.ProviderError{
			Provider: "xai", StatusCode: 502, Message: "unmarshal: " + err.Error(), Retryable: false,
		}
	}
	return &result, nil
}

func (p *XAIProvider) do(ctx context.Context, method, path, contentType string, body io.Reader) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, body)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "xai", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "xai", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "xai", StatusCode: 502, Message: "read response: " + err.Error(), Retryable: true, Err: err,
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &provider.ProviderError{
			Provider:   "xai",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}
	return respBody, nil
}
