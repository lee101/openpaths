package cursor

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

const defaultBaseURL = "https://api.cursor.com"

type Provider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func New(apiKey, baseURL string) *Provider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Provider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 10 * time.Minute},
	}
}

func (p *Provider) Name() string { return "cursor" }

func (p *Provider) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"/v1/me", nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(p.apiKey, "")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health check failed: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (p *Provider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	prompt := messagesToPrompt(req.Messages)
	if prompt == "" {
		return nil, &provider.ProviderError{
			Provider: p.Name(), StatusCode: 400, Message: "messages must include user content", Retryable: false,
		}
	}

	modelID, fast := parseModelSelection(req.Model)
	createBody := createAgentRequest{
		Prompt: promptPayload{Text: prompt},
		Model: modelSelection{
			ID: modelID,
		},
	}
	if fast {
		createBody.Model.Params = []modelParam{{ID: "fast", Value: "true"}}
	}

	agentID, runID, err := p.createAgent(ctx, createBody)
	if err != nil {
		return nil, err
	}
	defer p.deleteAgent(context.WithoutCancel(ctx), agentID)

	result, err := p.waitForRun(ctx, agentID, runID)
	if err != nil {
		return nil, err
	}

	promptTokens := estimateTokens(prompt)
	completionTokens := estimateTokens(result)
	finishReason := "stop"
	now := time.Now().Unix()

	return &model.ChatCompletionResponse{
		ID:      runID,
		Object:  "chat.completion",
		Created: now,
		Model:   req.Model,
		Choices: []model.ChatChoice{{
			Index: 0,
			Message: &model.ChatMessage{
				Role:    "assistant",
				Content: result,
			},
			FinishReason: &finishReason,
		}},
		Usage: &model.UsageInfo{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      promptTokens + completionTokens,
		},
	}, nil
}

func (p *Provider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	resp, err := p.ChatCompletion(ctx, req)
	if err != nil {
		return nil, err
	}

	ch := make(chan provider.StreamEvent, 2)
	go func() {
		defer close(ch)
		content, _ := resp.Choices[0].Message.Content.(string)
		ch <- provider.StreamEvent{
			Chunk: &model.ChatCompletionChunk{
				ID:      resp.ID,
				Object:  "chat.completion.chunk",
				Created: resp.Created,
				Model:   resp.Model,
				Choices: []model.ChatChoice{{
					Index: 0,
					Delta: &model.ChatMessage{Role: "assistant", Content: content},
				}},
			},
		}
		ch <- provider.StreamEvent{Usage: resp.Usage, Done: true}
	}()
	return ch, nil
}

type promptPayload struct {
	Text string `json:"text"`
}

type modelParam struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

type modelSelection struct {
	ID     string       `json:"id"`
	Params []modelParam `json:"params,omitempty"`
}

type createAgentRequest struct {
	Prompt promptPayload  `json:"prompt"`
	Model  modelSelection `json:"model"`
}

type createAgentResponse struct {
	Agent struct {
		ID string `json:"id"`
	} `json:"agent"`
	Run struct {
		ID string `json:"id"`
	} `json:"run"`
	Error *apiError `json:"error"`
}

type runResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Result string `json:"result"`
	Error  *apiError
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (p *Provider) createAgent(ctx context.Context, body createAgentRequest) (agentID, runID string, err error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return "", "", fmt.Errorf("marshal create agent: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/agents", bytes.NewReader(payload))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(p.apiKey, "")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", "", &provider.ProviderError{
			Provider: p.Name(), StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	if resp.StatusCode >= 400 {
		return "", "", p.apiError(resp.StatusCode, respBody)
	}

	var out createAgentResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", "", fmt.Errorf("unmarshal create agent: %w", err)
	}
	if out.Error != nil {
		return "", "", p.wrapAPIError(400, out.Error)
	}
	if out.Agent.ID == "" || out.Run.ID == "" {
		return "", "", &provider.ProviderError{
			Provider: p.Name(), StatusCode: 502, Message: "cursor create agent returned empty ids", Retryable: false,
		}
	}
	return out.Agent.ID, out.Run.ID, nil
}

func (p *Provider) waitForRun(ctx context.Context, agentID, runID string) (string, error) {
	url := fmt.Sprintf("%s/v1/agents/%s/runs/%s", p.baseURL, agentID, runID)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		req.SetBasicAuth(p.apiKey, "")

		resp, err := p.client.Do(req)
		if err != nil {
			return "", &provider.ProviderError{
				Provider: p.Name(), StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
			}
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return "", readErr
		}
		if resp.StatusCode >= 400 {
			return "", p.apiError(resp.StatusCode, body)
		}

		var run runResponse
		if err := json.Unmarshal(body, &run); err != nil {
			return "", fmt.Errorf("unmarshal run: %w", err)
		}

		switch strings.ToUpper(run.Status) {
		case "FINISHED":
			return run.Result, nil
		case "ERROR":
			msg := "cursor agent run failed"
			if run.Error != nil && run.Error.Message != "" {
				msg = run.Error.Message
			}
			return "", &provider.ProviderError{
				Provider: p.Name(), StatusCode: 502, Message: msg, Retryable: false,
			}
		case "CANCELLED", "EXPIRED":
			return "", &provider.ProviderError{
				Provider: p.Name(), StatusCode: 502, Message: "cursor agent run " + strings.ToLower(run.Status), Retryable: false,
			}
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
		}
	}
}

func (p *Provider) deleteAgent(ctx context.Context, agentID string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, p.baseURL+"/v1/agents/"+agentID, nil)
	if err != nil {
		return
	}
	req.SetBasicAuth(p.apiKey, "")
	resp, err := p.client.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

func parseModelSelection(providerModelID string) (id string, fast bool) {
	id = strings.TrimSpace(providerModelID)
	fast = strings.HasSuffix(id, "-fast")
	if fast {
		id = strings.TrimSuffix(id, "-fast")
	}
	if id == "" {
		id = "composer-2.5"
	}
	return id, fast
}

func messagesToPrompt(messages []model.ChatMessage) string {
	var b strings.Builder
	for _, msg := range messages {
		text := messageText(msg)
		if text == "" {
			continue
		}
		switch msg.Role {
		case "system":
			fmt.Fprintf(&b, "System: %s\n\n", text)
		case "assistant":
			fmt.Fprintf(&b, "Assistant: %s\n\n", text)
		case "tool":
			fmt.Fprintf(&b, "Tool: %s\n\n", text)
		default:
			fmt.Fprintf(&b, "User: %s\n\n", text)
		}
	}
	return strings.TrimSpace(b.String())
}

func messageText(msg model.ChatMessage) string {
	switch v := msg.Content.(type) {
	case string:
		return strings.TrimSpace(v)
	case []any:
		var parts []string
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if t, ok := m["text"].(string); ok && strings.TrimSpace(t) != "" {
				parts = append(parts, strings.TrimSpace(t))
			}
		}
		return strings.Join(parts, "\n")
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func estimateTokens(text string) int {
	if text == "" {
		return 0
	}
	n := len([]rune(text)) / 4
	if n < 1 {
		return 1
	}
	return n
}

func (p *Provider) apiError(statusCode int, body []byte) error {
	var wrapped struct {
		Error apiError `json:"error"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Error.Message != "" {
		return p.wrapAPIError(statusCode, &wrapped.Error)
	}
	return &provider.ProviderError{
		Provider: p.Name(), StatusCode: statusCode, Message: strings.TrimSpace(string(body)), Retryable: statusCode >= 500,
	}
}

func (p *Provider) wrapAPIError(statusCode int, apiErr *apiError) error {
	if apiErr == nil {
		return &provider.ProviderError{Provider: p.Name(), StatusCode: statusCode, Message: "cursor api error", Retryable: false}
	}
	retryable := statusCode >= 500 || apiErr.Code == "rate_limit_exceeded"
	return &provider.ProviderError{
		Provider: p.Name(), StatusCode: statusCode, Message: apiErr.Message, Retryable: retryable,
	}
}

// BasicAuthHeader returns the Authorization header value for tests/integration.
func BasicAuthHeader(apiKey string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(apiKey+":"))
}
