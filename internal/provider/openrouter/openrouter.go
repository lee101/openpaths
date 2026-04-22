package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	oai "github.com/openpaths/openpaths/internal/provider/openai"
)

type OpenRouterProvider struct {
	*oai.OpenAIProvider
	apiKey  string
	baseURL string
	client  *http.Client
}

func New(apiKey, baseURL string) *OpenRouterProvider {
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &OpenRouterProvider{
		OpenAIProvider: oai.New(apiKey, baseURL),
		apiKey:         apiKey,
		baseURL:        baseURL,
		client:         &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *OpenRouterProvider) Name() string { return "openrouter" }

// sanitizeForOpenRouter removes cross-provider hints that OpenRouter does not
// understand on the raw chat request payload.
func sanitizeForOpenRouter(req *model.ChatCompletionRequest) {
	req.Prefill = ""
	req.TaskTier = ""
}

func (p *OpenRouterProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	sanitizeForOpenRouter(req)
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "openrouter", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "openrouter",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var result model.ChatCompletionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &result, nil
}

func (p *OpenRouterProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("HTTP-Referer", "https://openpaths.io")
	req.Header.Set("X-Title", "OpenPaths")
}
