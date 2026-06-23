package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/promptcache"
	"github.com/openpaths/openpaths/internal/provider"
)

const anthropicVersion = "2023-06-01"

// extendedCacheTTLBeta is required to request 1h cache TTLs.
const extendedCacheTTLBeta = "extended-cache-ttl-2025-04-11"

type AnthropicProvider struct {
	apiKey    string
	baseURL   string
	client    *http.Client
	optimizer *promptcache.Optimizer
}

// SetCacheOptimizer attaches a prompt-cache cost optimizer. When set, the
// provider enables Anthropic prompt caching and picks the cheapest TTL per
// stable prefix. Nil-safe: with no optimizer, no cache_control is sent.
func (p *AnthropicProvider) SetCacheOptimizer(o *promptcache.Optimizer) {
	p.optimizer = o
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
	System    any                `json:"system,omitempty"`
	MaxTokens int                `json:"max_tokens"`
	Stream    bool               `json:"stream,omitempty"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
	Thinking  *anthropicThinking `json:"thinking,omitempty"`
}

// anthropicCacheControl is attached to cacheable system/content/tool blocks.
// ttl is "5m" (default) or "1h".
type anthropicCacheControl struct {
	Type string `json:"type"`          // "ephemeral"
	TTL  string `json:"ttl,omitempty"` // "5m" | "1h"
}

type anthropicTextBlock struct {
	Type         string                 `json:"type"`
	Text         string                 `json:"text"`
	CacheControl *anthropicCacheControl `json:"cache_control,omitempty"`
}

type anthropicImageBlock struct {
	Type   string               `json:"type"`
	Source anthropicImageSource `json:"source"`
}

type anthropicImageSource struct {
	Type      string `json:"type"`
	URL       string `json:"url,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Data      string `json:"data,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type anthropicTool struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	InputSchema  any                    `json:"input_schema,omitempty"`
	CacheControl *anthropicCacheControl `json:"cache_control,omitempty"`
}

type anthropicThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens,omitempty"`
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
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// promptTokens is the total billable input: uncached + cache reads + cache writes.
// Anthropic reports input_tokens excluding cached/written tokens.
func (u anthropicUsage) promptTokens() int {
	return u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens
}

// Streaming event types
type anthropicStreamEvent struct {
	Type         string             `json:"type"`
	Message      *anthropicResponse `json:"message,omitempty"`
	Index        int                `json:"index,omitempty"`
	ContentBlock *anthropicContent  `json:"content_block,omitempty"`
	Delta        *anthropicDelta    `json:"delta,omitempty"`
	Usage        *anthropicUsage    `json:"usage,omitempty"`
}

type anthropicDelta struct {
	Type       string `json:"type,omitempty"`
	Text       string `json:"text,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
}

func (p *AnthropicProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	anthReq := translateRequest(req)
	p.applyCacheControl(anthReq)

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
	setCacheBeta(httpReq, anthReq)

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

	if p.optimizer != nil {
		p.optimizer.RecordResult(anthReq.Model, anthResp.Usage.CacheReadInputTokens, anthResp.Usage.CacheCreationInputTokens)
	}
	return translateResponse(&anthResp, req.Model), nil
}

func (p *AnthropicProvider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	anthReq := translateRequest(req)
	anthReq.Stream = true
	p.applyCacheControl(anthReq)

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
	setCacheBeta(httpReq, anthReq)

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
					if event.Message != nil {
						// Includes cached read+write tokens; cache discount accrues to
						// us, not the customer (see translateResponse).
						totalUsage.PromptTokens = event.Message.Usage.promptTokens()
						totalUsage.CacheReadTokens = event.Message.Usage.CacheReadInputTokens
						totalUsage.CacheWriteTokens = event.Message.Usage.CacheCreationInputTokens
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
					if p.optimizer != nil {
						p.optimizer.RecordResult(anthReq.Model, totalUsage.CacheReadTokens, totalUsage.CacheWriteTokens)
					}
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
	if thinking := reasoningToThinking(req.ReasoningEffort, anthReq.MaxTokens); thinking != nil {
		anthReq.Thinking = thinking
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
			Content: translateContentBlocks(msg.Content),
		})
	}
	// Cross-provider prefill: Anthropic accepts a trailing assistant message to
	// force the response to start with a specific string. Only append if the
	// caller hasn't already supplied one.
	if req.Prefill != "" {
		if len(messages) == 0 || messages[len(messages)-1].Role != "assistant" {
			messages = append(messages, anthropicMessage{
				Role:    "assistant",
				Content: req.Prefill,
			})
		}
	}
	anthReq.Messages = messages

	// Translate tools
	for _, tool := range req.Tools {
		if tool.Function == nil {
			continue
		}
		anthReq.Tools = append(anthReq.Tools, anthropicTool{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			InputSchema: tool.Function.Parameters,
		})
	}

	return anthReq
}

// applyCacheControl records the request's prefix timing and sets the cheapest
// cache_control TTL for it, as decided by the optimizer. No-op when no optimizer
// is attached.
func (p *AnthropicProvider) applyCacheControl(anthReq *anthropicRequest) {
	if p.optimizer == nil {
		return
	}
	// Anthropic only caches prefixes above a minimum size (~1024 tokens for
	// Opus/Sonnet). Below that, cache_control is ignored upstream, so skip it
	// rather than track a prefix that can never produce a cache write/hit.
	if cacheablePrefixChars(anthReq) < minCacheablePrefixChars {
		return
	}
	key := prefixKey(anthReq)
	p.optimizer.Observe(key, anthReq.Model, time.Now())
	switch p.optimizer.Decide(key) {
	case promptcache.TTL5m:
		attachCacheControl(anthReq, &anthropicCacheControl{Type: "ephemeral", TTL: "5m"})
	case promptcache.TTL1h:
		attachCacheControl(anthReq, &anthropicCacheControl{Type: "ephemeral", TTL: "1h"})
	}
}

func attachCacheControl(anthReq *anthropicRequest, cc *anthropicCacheControl) {
	if cc == nil {
		return
	}
	if len(anthReq.Tools) > 0 {
		anthReq.Tools[len(anthReq.Tools)-1].CacheControl = cc
		return
	}
	if anthReq.System == nil {
		return
	}
	switch sys := anthReq.System.(type) {
	case string:
		anthReq.System = []anthropicTextBlock{{Type: "text", Text: sys, CacheControl: cc}}
	case []anthropicTextBlock:
		if len(sys) == 0 {
			anthReq.System = []anthropicTextBlock{{Type: "text", Text: "", CacheControl: cc}}
			return
		}
		sys[len(sys)-1].CacheControl = cc
		anthReq.System = sys
	case []any:
		if len(sys) > 0 {
			if block, ok := sys[len(sys)-1].(map[string]any); ok {
				block["cache_control"] = cc
				sys[len(sys)-1] = block
			}
		}
		anthReq.System = sys
	}
}

// minCacheablePrefixChars approximates Anthropic's ~1024-token minimum cacheable
// prefix at ~4 chars/token. Below this we don't bother requesting a cache.
const minCacheablePrefixChars = 4096

// cacheablePrefixChars estimates the size of the stable cacheable head
// (system + tools) in characters.
func cacheablePrefixChars(anthReq *anthropicRequest) int {
	n := systemTextLen(anthReq.System)
	if len(anthReq.Tools) > 0 {
		if tb, err := json.Marshal(anthReq.Tools); err == nil {
			n += len(tb)
		}
	}
	return n
}

// prefixKey identifies the stable cacheable head of a request (model + system +
// tools), matching how Anthropic prefix-matches caches across turns.
func prefixKey(anthReq *anthropicRequest) string {
	h := sha256.New()
	io.WriteString(h, anthReq.Model)
	h.Write([]byte{0})
	io.WriteString(h, systemText(anthReq.System))
	h.Write([]byte{0})
	if len(anthReq.Tools) > 0 {
		if tb, err := json.Marshal(anthReq.Tools); err == nil {
			h.Write(tb)
		}
	}
	return hex.EncodeToString(h.Sum(nil)[:16])
}

// setCacheBeta adds the beta header required for 1h cache TTLs.
func setCacheBeta(httpReq *http.Request, anthReq *anthropicRequest) {
	if requestCacheControlTTL(anthReq) == "1h" {
		httpReq.Header.Set("anthropic-beta", extendedCacheTTLBeta)
	}
}

func requestCacheControlTTL(anthReq *anthropicRequest) string {
	for _, tool := range anthReq.Tools {
		if tool.CacheControl != nil && tool.CacheControl.TTL != "" {
			return tool.CacheControl.TTL
		}
	}
	for _, block := range systemTextBlocks(anthReq.System) {
		if block.CacheControl != nil && block.CacheControl.TTL != "" {
			return block.CacheControl.TTL
		}
	}
	for _, msg := range anthReq.Messages {
		for _, block := range contentTextBlocks(msg.Content) {
			if block.CacheControl != nil && block.CacheControl.TTL != "" {
				return block.CacheControl.TTL
			}
		}
	}
	return ""
}

func systemTextLen(system any) int {
	return len(systemText(system))
}

func systemText(system any) string {
	switch s := system.(type) {
	case string:
		return s
	case []anthropicTextBlock:
		var b strings.Builder
		for _, block := range s {
			b.WriteString(block.Text)
		}
		return b.String()
	default:
		if system == nil {
			return ""
		}
		data, _ := json.Marshal(system)
		return string(data)
	}
}

func systemTextBlocks(system any) []anthropicTextBlock {
	if blocks, ok := system.([]anthropicTextBlock); ok {
		return blocks
	}
	return nil
}

func contentTextBlocks(content any) []anthropicTextBlock {
	if blocks, ok := content.([]anthropicTextBlock); ok {
		return blocks
	}
	return nil
}

func translateContentBlocks(content any) any {
	switch c := content.(type) {
	case []any:
		return translateContentBlockSlice(c)
	case []map[string]any:
		parts := make([]any, 0, len(c))
		for _, part := range c {
			parts = append(parts, part)
		}
		return translateContentBlockSlice(parts)
	default:
		return content
	}
}

func translateContentBlockSlice(parts []any) any {
	translated := make([]any, 0, len(parts))
	for _, part := range parts {
		block, ok := part.(map[string]any)
		if !ok {
			translated = append(translated, part)
			continue
		}
		switch blockType, _ := block["type"].(string); blockType {
		case "text":
			if text, ok := block["text"].(string); ok {
				translated = append(translated, anthropicTextBlock{Type: "text", Text: text})
				continue
			}
		case "image_url":
			if img, ok := anthropicImageBlockFromOpenAI(block); ok {
				translated = append(translated, img)
				continue
			}
		}
		translated = append(translated, part)
	}
	return translated
}

func anthropicImageBlockFromOpenAI(block map[string]any) (anthropicImageBlock, bool) {
	var rawURL string
	switch v := block["image_url"].(type) {
	case string:
		rawURL = v
	case map[string]any:
		rawURL, _ = v["url"].(string)
	}
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return anthropicImageBlock{}, false
	}
	if strings.HasPrefix(rawURL, "data:") {
		mediaType, data, ok := parseDataImageURL(rawURL)
		if !ok {
			return anthropicImageBlock{}, false
		}
		return anthropicImageBlock{
			Type: "image",
			Source: anthropicImageSource{
				Type:      "base64",
				MediaType: mediaType,
				Data:      data,
			},
		}, true
	}
	return anthropicImageBlock{
		Type: "image",
		Source: anthropicImageSource{
			Type: "url",
			URL:  rawURL,
		},
	}, true
}

func parseDataImageURL(rawURL string) (string, string, bool) {
	header, data, ok := strings.Cut(rawURL, ",")
	if !ok || !strings.Contains(header, ";base64") {
		return "", "", false
	}
	mediaType := strings.TrimPrefix(strings.TrimSuffix(header, ";base64"), "data:")
	if mediaType == "" {
		mediaType = "image/jpeg"
	}
	if _, err := base64.StdEncoding.DecodeString(data); err != nil {
		return "", "", false
	}
	return mediaType, data, true
}

func reasoningToThinking(effort string, maxTokens int) *anthropicThinking {
	switch effort {
	case "", "none":
		return nil
	}

	maxBudget := maxTokens - 1
	if maxBudget < 1024 {
		return nil
	}

	budget := 1024
	switch effort {
	case "low":
		budget = 1024
	case "medium":
		budget = 4096
	case "high":
		budget = 16384
	default:
		return nil
	}

	if budget > maxBudget {
		budget = maxBudget
	}
	if budget < 1024 {
		return nil
	}

	return &anthropicThinking{
		Type:         "enabled",
		BudgetTokens: budget,
	}
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
		// PromptTokens includes cached read+write tokens (Anthropic excludes them
		// from input_tokens) so customers are billed for the full prompt. We do not
		// set PromptCacheHitTokens: caching is a cost optimization for our upstream
		// spend, so the cache discount accrues to us rather than being passed
		// through. To pass savings to customers instead, set PromptCacheHitTokens =
		// CacheReadInputTokens and configure input_cache_hit_price_per_1m.
		Usage: &model.UsageInfo{
			PromptTokens:     resp.Usage.promptTokens(),
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.promptTokens() + resp.Usage.OutputTokens,
			CacheReadTokens:  resp.Usage.CacheReadInputTokens,
			CacheWriteTokens: resp.Usage.CacheCreationInputTokens,
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
