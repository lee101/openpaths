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

// Z.AI exposes two OpenAI-compatible surfaces under the same host:
//   - the standard pay-as-you-go API at /api/paas/v4 (regular API keys)
//   - the GLM Coding Plan API at /api/coding/paas/v4 (subscription keys)
//
// A coding-plan key only works against the coding path and a standard key only
// works against the standard path, so the provider keeps an ordered list of base
// paths and falls through to the next one when the upstream rejects the key as a
// wrong-surface auth error.
const (
	standardBasePath = "/api/paas/v4"
	codingBasePath   = "/api/coding/paas/v4"
)

type ZAIProvider struct {
	apiKey  string
	baseURL string
	// paths is the ordered list of base paths to try for chat completions. The
	// first 200 wins; a 401/403/404 falls through to the next path.
	paths  []string
	client *http.Client
}

func New(apiKey, baseURL string) *ZAIProvider {
	return newWithPaths(apiKey, baseURL, []string{standardBasePath})
}

// NewCoding targets the GLM Coding Plan endpoint first, falling back to the
// standard endpoint if the key turns out to be a regular API key. This is the
// constructor used for BYOK GLM keys, which are almost always coding-plan keys.
func NewCoding(apiKey, baseURL string) *ZAIProvider {
	return newWithPaths(apiKey, baseURL, []string{codingBasePath, standardBasePath})
}

func newWithPaths(apiKey, baseURL string, paths []string) *ZAIProvider {
	if baseURL == "" {
		baseURL = "https://api.z.ai"
	}
	return &ZAIProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		paths:   paths,
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *ZAIProvider) Name() string { return "zai" }

// wrongSurfaceStatus reports whether a non-200 status indicates the key is for a
// different Z.AI surface (and we should try the next base path) rather than a
// genuine error we should surface to the caller.
func wrongSurfaceStatus(code int) bool {
	return code == 401 || code == 403 || code == 404
}

// doChat sends the chat request, trying each base path until one returns 200 or
// returns a non-200 that is not a wrong-surface auth error. The returned
// response body is left open for the caller to read/stream.
func (p *ZAIProvider) doChat(ctx context.Context, body []byte) (*http.Response, error) {
	var lastResp *http.Response
	var lastNetErr error
	for i, path := range p.paths {
		httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+path+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

		resp, err := p.client.Do(httpReq)
		if err != nil {
			lastNetErr = err
			continue
		}
		if resp.StatusCode == 200 || !wrongSurfaceStatus(resp.StatusCode) || i == len(p.paths)-1 {
			return resp, nil
		}
		// Wrong-surface auth error with another path to try: drain, close, retry.
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		lastResp = resp
	}
	if lastNetErr != nil {
		return nil, &provider.ProviderError{
			Provider: "zai", StatusCode: 502, Message: lastNetErr.Error(), Retryable: true, Err: lastNetErr,
		}
	}
	if lastResp != nil {
		return nil, &provider.ProviderError{
			Provider: "zai", StatusCode: lastResp.StatusCode, Message: "z.ai rejected the key on all endpoints", Retryable: false,
		}
	}
	return nil, &provider.ProviderError{Provider: "zai", StatusCode: 502, Message: "no z.ai endpoint configured", Retryable: false}
}

// sanitizeForZAI removes cross-provider hints that Z.AI's raw chat endpoint
// does not understand, and reshapes message lists Z.AI rejects but the
// OpenAI-compatible surface we present accepts.
func sanitizeForZAI(req *model.ChatCompletionRequest) {
	req.Prefill = ""
	req.TaskTier = ""
	req.RoutingStrategy = ""
	// Every model on this surface is a GLM, so the user-turn reshaping always
	// applies here.
	model.PromoteSystemToUser(req)

	// GLM-5.3-Flash only supports enabled thinking. Keep prior reasoning blocks
	// available for long-running tool conversations, as recommended by Z.AI.
	if strings.HasPrefix(strings.ToLower(req.Model), "glm-5.3-flash") {
		if req.Thinking == nil {
			clearThinking := false
			req.Thinking = &model.ThinkingConfig{Type: "enabled", ClearThinking: &clearThinking}
		} else {
			req.Thinking.Type = "enabled"
		}
	}
}

func (p *ZAIProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	sanitizeForZAI(req)
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := p.doChat(ctx, body)
	if err != nil {
		return nil, err
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
	if len(req.Tools) > 0 {
		req.ToolStream = true
	}
	sanitizeForZAI(req)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := p.doChat(ctx, body)
	if err != nil {
		return nil, err
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

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+p.paths[0]+"/images/generations", bytes.NewReader(body))
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
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+p.paths[0]+"/models", nil)
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
