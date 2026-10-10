package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

const (
	maxImageResponseBytes = 128 << 20
	maxErrorMessageBytes  = 2048
)

type imageReference struct {
	Type     string        `json:"type"`
	ImageURL imageReferURL `json:"image_url"`
}

type imageReferURL struct {
	URL string `json:"url"`
}

type imageRequest struct {
	Model           string           `json:"model"`
	Prompt          string           `json:"prompt"`
	N               int              `json:"n,omitempty"`
	Size            string           `json:"size,omitempty"`
	Quality         string           `json:"quality,omitempty"`
	ResponseFormat  string           `json:"response_format,omitempty"`
	AspectRatio     string           `json:"aspect_ratio,omitempty"`
	InputReferences []imageReference `json:"input_references,omitempty"`
}

func (p *OpenRouterProvider) GenerateImage(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	out := imageRequest{
		Model:          req.Model,
		Prompt:         req.Prompt,
		N:              req.N,
		Size:           req.Size,
		Quality:        req.Quality,
		ResponseFormat: req.ResponseFormat,
		AspectRatio:    strings.TrimSpace(req.AspectRatio),
	}
	if out.ResponseFormat == "" {
		out.ResponseFormat = "b64_json"
	}
	for _, u := range imageReferenceURLs(req) {
		out.InputReferences = append(out.InputReferences, imageReference{Type: "image_url", ImageURL: imageReferURL{URL: u}})
	}
	body, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/images/generations", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	p.setHeaders(httpReq)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "openrouter", StatusCode: http.StatusBadGateway, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxImageResponseBytes))
	if err != nil {
		return nil, &provider.ProviderError{Provider: "openrouter", StatusCode: http.StatusBadGateway, Message: "read response: " + err.Error(), Retryable: true, Err: err}
	}
	if resp.StatusCode != http.StatusOK {
		msg := string(respBody)
		if len(msg) > maxErrorMessageBytes {
			msg = msg[:maxErrorMessageBytes]
		}
		return nil, &provider.ProviderError{Provider: "openrouter", StatusCode: resp.StatusCode, Message: msg, Retryable: resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests}
	}
	var result model.ImageGenerationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, &provider.ProviderError{Provider: "openrouter", StatusCode: http.StatusBadGateway, Message: "invalid image response", Retryable: true, Err: err}
	}
	if len(result.Data) == 0 {
		return nil, &provider.ProviderError{Provider: "openrouter", StatusCode: http.StatusBadGateway, Message: "openrouter image response contained no images", Retryable: true}
	}
	return &result, nil
}

func imageReferenceURLs(req *model.ImageGenerationRequest) []string {
	var out []string
	add := func(v string) {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	add(req.ImageURL)
	if req.Image != nil {
		add(req.Image.URL)
	}
	for _, img := range req.Images {
		add(img.URL)
	}
	for _, v := range req.ImageURLs {
		add(v)
	}
	for _, v := range req.ReferenceImageURLs {
		add(v)
	}
	return out
}
