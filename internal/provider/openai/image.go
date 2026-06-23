package openai

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

type openaiImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Style          string `json:"style,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
}

func (p *OpenAIProvider) GenerateImage(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	out := openaiImageRequest{
		Model:          req.Model,
		Prompt:         req.Prompt,
		N:              req.N,
		Size:           req.Size,
		Quality:        req.Quality,
		Style:          req.Style,
		ResponseFormat: req.ResponseFormat,
	}

	// GPT Image models return b64_json by default and do not accept response_format.
	// dall-e-3 is now served by the same images backend and also rejects
	// response_format ("Unknown parameter: 'response_format'"); only legacy
	// dall-e-2 still honours it. The handler rehosts URL responses regardless,
	// so dropping the param just lets the provider use its default format.
	m := strings.ToLower(req.Model)
	isGPTImage := strings.HasPrefix(m, "gpt-image")
	rejectsResponseFormat := isGPTImage || strings.HasPrefix(m, "dall-e-3")
	if rejectsResponseFormat {
		out.ResponseFormat = ""
	}
	if isGPTImage {
		// GPT Image models don't support "style"; they use quality only.
		out.Style = ""
	}
	if !rejectsResponseFormat && out.ResponseFormat == "" {
		out.ResponseFormat = "b64_json"
	}

	body, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/images/generations", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "openai", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "openai",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var result model.ImageGenerationResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}
