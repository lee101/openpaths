package sakana

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

// Sakana exposes an OpenAI-compatible surface at https://api.sakana.ai/v1. The
// flagship "fugu" / "fugu-ultra" models route a single request across an
// internal pool of providers (orchestration), so the usage they report carries
// orchestration token counts on TOP of the surface prompt/completion tokens.
// total_tokens already folds those in (total = prompt + completion +
// orchestration_input + orchestration_output). We must bill on the full usage,
// otherwise the surface prompt_tokens (which can be ~180x smaller than reality)
// would massively undercharge. normalizeUsage folds the orchestration tokens
// back into PromptTokens/CompletionTokens so downstream pricing is accurate.

type SakanaProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func New(apiKey, baseURL string) *SakanaProvider {
	if baseURL == "" {
		baseURL = "https://api.sakana.ai"
	}
	return &SakanaProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

func (p *SakanaProvider) Name() string { return "sakana" }

// sakanaUsage captures fugu's orchestration-aware usage shape.
type sakanaUsage struct {
	PromptTokens        int `json:"prompt_tokens"`
	CompletionTokens    int `json:"completion_tokens"`
	TotalTokens         int `json:"total_tokens"`
	PromptTokensDetails struct {
		CachedTokens                   int `json:"cached_tokens"`
		OrchestrationInputTokens       int `json:"orchestration_input_tokens"`
		OrchestrationInputCachedTokens int `json:"orchestration_input_cached_tokens"`
	} `json:"prompt_tokens_details"`
	CompletionTokensDetails struct {
		ReasoningTokens           int `json:"reasoning_tokens"`
		OrchestrationOutputTokens int `json:"orchestration_output_tokens"`
	} `json:"completion_tokens_details"`
}

// normalize folds orchestration tokens into the billable prompt/completion
// counts so the per-request cost reconstructed from fugu's reported usage is
// accurate.
func (u *sakanaUsage) normalize() *model.UsageInfo {
	inputBillable := u.PromptTokens + u.PromptTokensDetails.OrchestrationInputTokens
	outputBillable := u.CompletionTokens + u.CompletionTokensDetails.OrchestrationOutputTokens
	cached := u.PromptTokensDetails.CachedTokens + u.PromptTokensDetails.OrchestrationInputCachedTokens
	if cached > inputBillable {
		cached = inputBillable
	}
	out := &model.UsageInfo{
		PromptTokens:     inputBillable,
		CompletionTokens: outputBillable,
		TotalTokens:      inputBillable + outputBillable,
	}
	if cached > 0 {
		out.PromptCacheHitTokens = cached
		out.PromptTokensDetails = &model.PromptTokensDetails{CachedTokens: cached}
	}
	return out
}

// sanitize strips cross-provider hints fugu's raw endpoint does not understand.
func sanitize(req *model.ChatCompletionRequest) {
	req.Prefill = ""
	req.TaskTier = ""
	req.RoutingStrategy = ""
	req.Thinking = nil
	req.ChatTemplateKwargs = nil
}

func (p *SakanaProvider) do(ctx context.Context, body []byte) (*http.Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "sakana", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	return resp, nil
}

func (p *SakanaProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	sanitize(req)
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := p.do(ctx, body)
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
			Provider:   "sakana",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var result model.ChatCompletionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Replace surface usage with orchestration-aware billable usage.
	var probe struct {
		Usage *sakanaUsage `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &probe); err == nil && probe.Usage != nil {
		result.Usage = probe.Usage.normalize()
	}

	return &result, nil
}

func (p *SakanaProvider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	req.Stream = true
	sanitize(req)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := p.do(ctx, body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &provider.ProviderError{
			Provider:   "sakana",
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
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
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

			// Fold orchestration tokens into the usage chunk.
			var probe struct {
				Usage *sakanaUsage `json:"usage"`
			}
			if err := json.Unmarshal([]byte(data), &probe); err == nil && probe.Usage != nil {
				chunk.Usage = probe.Usage.normalize()
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

func (p *SakanaProvider) HealthCheck(ctx context.Context) error {
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
