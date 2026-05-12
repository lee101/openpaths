package zai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

type ZAIProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func New(apiKey, baseURL string) *ZAIProvider {
	if baseURL == "" {
		baseURL = "https://api.z.ai"
	}
	return &ZAIProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *ZAIProvider) Name() string { return "zai" }

func (p *ZAIProvider) chatURL() string {
	return p.baseURL + "/api/paas/v4/chat/completions"
}

// sanitizeForZAI removes cross-provider hints that Z.AI's raw chat endpoint
// does not understand.
func sanitizeForZAI(req *model.ChatCompletionRequest) {
	req.Prefill = ""
	req.TaskTier = ""
	req.Thinking = nil
}

func (p *ZAIProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	sanitizeForZAI(req)
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.chatURL(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "zai", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "zai",
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

func (p *ZAIProvider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	req.Stream = true
	sanitizeForZAI(req)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.chatURL(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "zai", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &provider.ProviderError{
			Provider:   "zai",
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

func (p *ZAIProvider) GenerateImage(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	payload := map[string]any{
		"model":  req.Model,
		"prompt": req.Prompt,
	}
	if req.Size != "" {
		payload["size"] = req.Size
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/paas/v4/images/generations", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "zai", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "zai",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var raw struct {
		Created int64 `json:"created"`
		Data    []struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(raw.Data) == 0 {
		log.Printf("zai glm-image: raw response: %s", string(respBody))
		return nil, &provider.ProviderError{
			Provider: "zai", StatusCode: 502, Message: "no image data in response", Retryable: false,
		}
	}

	result := &model.ImageGenerationResponse{
		Created: raw.Created,
	}
	for _, d := range raw.Data {
		result.Data = append(result.Data, model.ImageData{URL: d.URL})
	}

	return result, nil
}

func (p *ZAIProvider) HealthCheck(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/paas/v4/models", nil)
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
