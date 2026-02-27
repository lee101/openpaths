package anthropic

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

	"github.com/google/uuid"
	"github.com/openpath/openpath/internal/model"
	"github.com/openpath/openpath/internal/provider"
)

const anthropicVersion = "2023-06-01"

type AnthropicProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func New(apiKey, baseURL string) *AnthropicProvider {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	return &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *AnthropicProvider) Name() string { return "anthropic" }

// Anthropic API types
type anthropicRequest struct {
	Model     string             `json:"model"`
	Messages  []anthropicMessage `json:"messages"`
	System    string             `json:"system,omitempty"`
	MaxTokens int                `json:"max_tokens"`
	Stream    bool               `json:"stream,omitempty"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type anthropicTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema,omitempty"`
}

type anthropicResponse struct {
	ID         string             `json:"id"`
	Type       string             `json:"type"`
	Role       string             `json:"role"`
	Content    []anthropicContent `json:"content"`
	Model      string             `json:"model"`
	StopReason string             `json:"stop_reason"`
	Usage      anthropicUsage     `json:"usage"`
}

type anthropicContent struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// Streaming event types
type anthropicStreamEvent struct {
	Type         string            `json:"type"`
	Message      *anthropicResponse `json:"message,omitempty"`
	Index        int               `json:"index,omitempty"`
	ContentBlock *anthropicContent `json:"content_block,omitempty"`
	Delta        *anthropicDelta   `json:"delta,omitempty"`
	Usage        *anthropicUsage   `json:"usage,omitempty"`
}

type anthropicDelta struct {
	Type       string `json:"type,omitempty"`
	Text       string `json:"text,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
}

func (p *AnthropicProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	anthReq := translateRequest(req)

	body, err := json.Marshal(anthReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "anthropic", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "anthropic",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var anthResp anthropicResponse
	if err := json.Unmarshal(respBody, &anthResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return translateResponse(&anthResp, req.Model), nil
}

func (p *AnthropicProvider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	anthReq := translateRequest(req)
	anthReq.Stream = true

	body, err := json.Marshal(anthReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "anthropic", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &provider.ProviderError{
			Provider:   "anthropic",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	ch := make(chan provider.StreamEvent, 64)
	chatID := "chatcmpl-" + uuid.New().String()[:8]
	createdTime := time.Now().Unix()

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		var totalUsage model.UsageInfo

		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "event: ") {
				// Read the next data line
				if !scanner.Scan() {
					break
				}
				dataLine := scanner.Text()
				if !strings.HasPrefix(dataLine, "data: ") {
					continue
				}
				data := strings.TrimPrefix(dataLine, "data: ")

				var event anthropicStreamEvent
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					continue
				}

				switch event.Type {
				case "message_start":
					if event.Message != nil && event.Message.Usage.InputTokens > 0 {
						totalUsage.PromptTokens = event.Message.Usage.InputTokens
					}

				case "content_block_delta":
					if event.Delta != nil && event.Delta.Text != "" {
						finishReason := (*string)(nil)
						chunk := &model.ChatCompletionChunk{
							ID:      chatID,
							Object:  "chat.completion.chunk",
							Created: createdTime,
							Model:   req.Model,
							Choices: []model.ChatChoice{{
								Index: 0,
								Delta: &model.ChatMessage{
									Role:    "",
									Content: event.Delta.Text,
								},
								FinishReason: finishReason,
							}},
						}
						ch <- provider.StreamEvent{Chunk: chunk}
					}

				case "message_delta":
					if event.Usage != nil {
						totalUsage.CompletionTokens = event.Usage.OutputTokens
					}
					if event.Delta != nil && event.Delta.StopReason != "" {
						reason := mapStopReason(event.Delta.StopReason)
						chunk := &model.ChatCompletionChunk{
							ID:      chatID,
							Object:  "chat.completion.chunk",
							Created: createdTime,
							Model:   req.Model,
							Choices: []model.ChatChoice{{
								Index:        0,
								Delta:        &model.ChatMessage{},
								FinishReason: &reason,
							}},
						}
						ch <- provider.StreamEvent{Chunk: chunk}
					}

				case "message_stop":
					totalUsage.TotalTokens = totalUsage.PromptTokens + totalUsage.CompletionTokens
					ch <- provider.StreamEvent{Done: true, Usage: &totalUsage}
					return
				}
			}
		}
	}()

	return ch, nil
}

func (p *AnthropicProvider) HealthCheck(ctx context.Context) error {
	// Anthropic doesn't have a models endpoint; send a minimal request
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/v1/messages", nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	resp.Body.Close()
	// 405 Method Not Allowed means the API is reachable
	if resp.StatusCode == 405 || resp.StatusCode == 200 {
		return nil
	}
	return fmt.Errorf("health check failed: %d", resp.StatusCode)
}

func translateRequest(req *model.ChatCompletionRequest) *anthropicRequest {
	anthReq := &anthropicRequest{
		Model:     req.Model,
		MaxTokens: 4096,
	}

	if req.MaxTokens != nil {
		anthReq.MaxTokens = *req.MaxTokens
	}
	if req.MaxCompletionTokens != nil {
		anthReq.MaxTokens = *req.MaxCompletionTokens
	}

	// Extract system message
	var messages []anthropicMessage
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			if s, ok := msg.Content.(string); ok {
				anthReq.System = s
			}
			continue
		}
		messages = append(messages, anthropicMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	anthReq.Messages = messages

	// Translate tools
	for _, tool := range req.Tools {
		anthReq.Tools = append(anthReq.Tools, anthropicTool{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			InputSchema: tool.Function.Parameters,
		})
	}

	return anthReq
}

func translateResponse(resp *anthropicResponse, requestModel string) *model.ChatCompletionResponse {
	// Build content string from content blocks
	var textContent string
	var toolCalls []model.ToolCall

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			textContent += block.Text
		case "tool_use":
			inputJSON, _ := json.Marshal(block.Input)
			toolCalls = append(toolCalls, model.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: model.ToolCallFunc{
					Name:      block.Name,
					Arguments: string(inputJSON),
				},
			})
		}
	}

	finishReason := mapStopReason(resp.StopReason)

	message := &model.ChatMessage{
		Role:    "assistant",
		Content: textContent,
	}
	if len(toolCalls) > 0 {
		message.ToolCalls = toolCalls
	}

	return &model.ChatCompletionResponse{
		ID:      "chatcmpl-" + resp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   requestModel,
		Choices: []model.ChatChoice{{
			Index:        0,
			Message:      message,
			FinishReason: &finishReason,
		}},
		Usage: &model.UsageInfo{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}
}

func mapStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	default:
		return "stop"
	}
}
