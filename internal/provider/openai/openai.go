package openai

import (
	"bufio"
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
)

type OpenAIProvider struct {
	apiKey       string
	baseURL      string
	providerName string
	sanitizeChat func(*model.ChatCompletionRequest)
	client       *http.Client
}

func New(apiKey, baseURL string) *OpenAIProvider {
	return NewCompatible("openai", apiKey, baseURL, sanitizeForOpenAI)
}

func NewCompatible(providerName, apiKey, baseURL string, sanitizeChat func(*model.ChatCompletionRequest)) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com"
	}
	if providerName == "" {
		providerName = "openai"
	}
	return &OpenAIProvider{
		apiKey:       apiKey,
		baseURL:      strings.TrimRight(baseURL, "/"),
		providerName: providerName,
		sanitizeChat: sanitizeChat,
		client:       &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *OpenAIProvider) Name() string { return p.providerName }

// sanitizeForOpenAI removes cross-provider hints that OpenAI's API does not
// recognize. OpenAI has no native "prefill" or "task_tier" fields and may
// reject unknown properties in strict mode. Callers should set these fields
// and let the Anthropic/Google providers translate them; here we drop them.
func sanitizeForOpenAI(req *model.ChatCompletionRequest) {
	req.Prefill = ""
	req.TaskTier = ""
	req.Thinking = nil
	req.ChatTemplateKwargs = nil
}

// normalizeMaxTokens converts max_tokens to max_completion_tokens for newer
// OpenAI models (o-series, gpt-5.*) that reject the legacy parameter.
func normalizeMaxTokens(req *model.ChatCompletionRequest) {
	if req.MaxTokens != nil && req.MaxCompletionTokens == nil {
		m := req.Model
		if strings.HasPrefix(m, "o1") || strings.HasPrefix(m, "o3") || strings.HasPrefix(m, "o4") ||
			strings.HasPrefix(m, "gpt-5.") || strings.HasPrefix(m, "gpt-5-codex") {
			req.MaxCompletionTokens = req.MaxTokens
			req.MaxTokens = nil
		}
	}
}

func (p *OpenAIProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	outReq := *req
	outReq.ChatTemplateKwargs = cloneMap(req.ChatTemplateKwargs)
	normalizeMaxTokens(&outReq)
	if p.sanitizeChat != nil {
		p.sanitizeChat(&outReq)
	}
	body, err := json.Marshal(&outReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: p.providerName, StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   p.providerName,
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

func (p *OpenAIProvider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	outReq := *req
	outReq.ChatTemplateKwargs = cloneMap(req.ChatTemplateKwargs)
	outReq.Stream = true
	if outReq.StreamOptions == nil {
		outReq.StreamOptions = &model.StreamOptions{IncludeUsage: true}
	}
	normalizeMaxTokens(&outReq)
	if p.sanitizeChat != nil {
		p.sanitizeChat(&outReq)
	}

	body, err := json.Marshal(&outReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: p.providerName, StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &provider.ProviderError{
			Provider:   p.providerName,
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	ch := make(chan provider.StreamEvent, 64)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		var lastUsage *model.UsageInfo

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")

			if data == "[DONE]" {
				ch <- provider.StreamEvent{Done: true, Usage: lastUsage}
				return
			}

			var chunk model.ChatCompletionChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				ch <- provider.StreamEvent{Err: fmt.Errorf("unmarshal chunk: %w", err)}
				return
			}

			if chunk.Usage != nil {
				lastUsage = chunk.Usage
			}

			ch <- provider.StreamEvent{Chunk: &chunk}
		}

		if err := scanner.Err(); err != nil {
			ch <- provider.StreamEvent{Err: err}
		}
	}()

	return ch, nil
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

func (p *OpenAIProvider) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/v1/models", nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("health check failed: %d", resp.StatusCode)
	}
	return nil
}
