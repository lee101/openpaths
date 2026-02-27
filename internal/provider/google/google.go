package google

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

type GoogleProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func New(apiKey, baseURL string) *GoogleProvider {
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com"
	}
	return &GoogleProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *GoogleProvider) Name() string { return "google" }

// Gemini API types
type geminiRequest struct {
	Contents         []geminiContent        `json:"contents"`
	SystemInstruction *geminiContent        `json:"systemInstruction,omitempty"`
	GenerationConfig *geminiGenerationCfg   `json:"generationConfig,omitempty"`
	Tools            []geminiToolDecl       `json:"tools,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text         string          `json:"text,omitempty"`
	FunctionCall *geminiFuncCall `json:"functionCall,omitempty"`
	FunctionResp *geminiFuncResp `json:"functionResponse,omitempty"`
}

type geminiFuncCall struct {
	Name string `json:"name"`
	Args any    `json:"args"`
}

type geminiFuncResp struct {
	Name     string `json:"name"`
	Response any    `json:"response"`
}

type geminiGenerationCfg struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type geminiToolDecl struct {
	FunctionDeclarations []geminiFuncDecl `json:"functionDeclarations"`
}

type geminiFuncDecl struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type geminiResponse struct {
	Candidates    []geminiCandidate `json:"candidates"`
	UsageMetadata *geminiUsage      `json:"usageMetadata"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

type geminiUsage struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

func (p *GoogleProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	gemReq := translateRequest(req)

	body, err := json.Marshal(gemReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", p.baseURL, req.Model, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "google", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "google",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(respBody, &gemResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return translateResponse(&gemResp, req.Model), nil
}

func (p *GoogleProvider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	gemReq := translateRequest(req)

	body, err := json.Marshal(gemReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:streamGenerateContent?alt=sse&key=%s", p.baseURL, req.Model, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "google", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &provider.ProviderError{
			Provider:   "google",
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
		var lastUsage *model.UsageInfo

		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")

			var gemResp geminiResponse
			if err := json.Unmarshal([]byte(data), &gemResp); err != nil {
				continue
			}

			if gemResp.UsageMetadata != nil {
				lastUsage = &model.UsageInfo{
					PromptTokens:     gemResp.UsageMetadata.PromptTokenCount,
					CompletionTokens: gemResp.UsageMetadata.CandidatesTokenCount,
					TotalTokens:      gemResp.UsageMetadata.TotalTokenCount,
				}
			}

			for _, cand := range gemResp.Candidates {
				for _, part := range cand.Content.Parts {
					if part.Text != "" {
						chunk := &model.ChatCompletionChunk{
							ID:      chatID,
							Object:  "chat.completion.chunk",
							Created: createdTime,
							Model:   req.Model,
							Choices: []model.ChatChoice{{
								Index: 0,
								Delta: &model.ChatMessage{Content: part.Text},
							}},
						}
						ch <- provider.StreamEvent{Chunk: chunk}
					}
				}

				if cand.FinishReason != "" && cand.FinishReason != "FINISH_REASON_UNSPECIFIED" {
					reason := mapFinishReason(cand.FinishReason)
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
			}
		}

		ch <- provider.StreamEvent{Done: true, Usage: lastUsage}
	}()

	return ch, nil
}

func (p *GoogleProvider) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/v1beta/models?key=%s", p.baseURL, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("health check failed: %d", resp.StatusCode)
	}
	return nil
}

func translateRequest(req *model.ChatCompletionRequest) *geminiRequest {
	gemReq := &geminiRequest{
		GenerationConfig: &geminiGenerationCfg{
			Temperature:   req.Temperature,
			TopP:          req.TopP,
			StopSequences: req.Stop,
		},
	}

	if req.MaxTokens != nil {
		gemReq.GenerationConfig.MaxOutputTokens = req.MaxTokens
	}
	if req.MaxCompletionTokens != nil {
		gemReq.GenerationConfig.MaxOutputTokens = req.MaxCompletionTokens
	}

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			if s, ok := msg.Content.(string); ok {
				gemReq.SystemInstruction = &geminiContent{
					Parts: []geminiPart{{Text: s}},
				}
			}
			continue
		}

		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		var parts []geminiPart
		switch c := msg.Content.(type) {
		case string:
			parts = []geminiPart{{Text: c}}
		default:
			// For complex content, serialize to string
			b, _ := json.Marshal(c)
			parts = []geminiPart{{Text: string(b)}}
		}

		gemReq.Contents = append(gemReq.Contents, geminiContent{
			Role:  role,
			Parts: parts,
		})
	}

	// Translate tools
	if len(req.Tools) > 0 {
		var funcDecls []geminiFuncDecl
		for _, tool := range req.Tools {
			funcDecls = append(funcDecls, geminiFuncDecl{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			})
		}
		gemReq.Tools = []geminiToolDecl{{FunctionDeclarations: funcDecls}}
	}

	return gemReq
}

func translateResponse(resp *geminiResponse, requestModel string) *model.ChatCompletionResponse {
	var textContent string
	var toolCalls []model.ToolCall
	finishReason := "stop"

	if len(resp.Candidates) > 0 {
		cand := resp.Candidates[0]
		for _, part := range cand.Content.Parts {
			if part.Text != "" {
				textContent += part.Text
			}
			if part.FunctionCall != nil {
				argsJSON, _ := json.Marshal(part.FunctionCall.Args)
				toolCalls = append(toolCalls, model.ToolCall{
					ID:   "call_" + uuid.New().String()[:8],
					Type: "function",
					Function: model.ToolCallFunc{
						Name:      part.FunctionCall.Name,
						Arguments: string(argsJSON),
					},
				})
			}
		}
		finishReason = mapFinishReason(cand.FinishReason)
	}

	message := &model.ChatMessage{
		Role:    "assistant",
		Content: textContent,
	}
	if len(toolCalls) > 0 {
		message.ToolCalls = toolCalls
	}

	var usage *model.UsageInfo
	if resp.UsageMetadata != nil {
		usage = &model.UsageInfo{
			PromptTokens:     resp.UsageMetadata.PromptTokenCount,
			CompletionTokens: resp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      resp.UsageMetadata.TotalTokenCount,
		}
	}

	return &model.ChatCompletionResponse{
		ID:      "chatcmpl-" + uuid.New().String()[:8],
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   requestModel,
		Choices: []model.ChatChoice{{
			Index:        0,
			Message:      message,
			FinishReason: &finishReason,
		}},
		Usage: usage,
	}
}

func mapFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY":
		return "content_filter"
	default:
		return "stop"
	}
}
