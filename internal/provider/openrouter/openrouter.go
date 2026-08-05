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
	apiKey        string
	baseURL       string
	appReferer    string
	appTitle      string
	appCategories string
	client        *http.Client
}

func New(apiKey, baseURL string) *OpenRouterProvider {
	return NewWithAttribution(apiKey, baseURL, "https://openpaths.io", "OpenPaths", "programming-app")
}

func NewWithAttribution(apiKey, baseURL, appReferer, appTitle, appCategories string) *OpenRouterProvider {
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if appReferer == "" {
		appReferer = "https://openpaths.io"
	}
	if appTitle == "" {
		appTitle = "OpenPaths"
	}
	return &OpenRouterProvider{
		OpenAIProvider: oai.New(apiKey, baseURL),
		apiKey:         apiKey,
		baseURL:        baseURL,
		appReferer:     appReferer,
		appTitle:       appTitle,
		appCategories:  appCategories,
		client:         &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *OpenRouterProvider) Name() string { return "openrouter" }

// sanitizeForOpenRouter removes cross-provider hints that OpenRouter does not
// understand on the raw chat request payload.
func sanitizeForOpenRouter(req *model.ChatCompletionRequest) {
	req.Prefill = ""
	req.TaskTier = ""
	req.RoutingStrategy = ""
	req.Thinking = nil
}

func (p *OpenRouterProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	outReq := *req
	outReq.ChatTemplateKwargs = cloneMap(req.ChatTemplateKwargs)
	sanitizeForOpenRouter(&outReq)
	body, err := json.Marshal(&outReq)
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

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (p *OpenRouterProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("HTTP-Referer", p.appReferer)
	req.Header.Set("X-OpenRouter-Title", p.appTitle)
	req.Header.Set("X-Title", p.appTitle)
	if p.appCategories != "" {
		req.Header.Set("X-OpenRouter-Categories", p.appCategories)
	}
}
