package handler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
)

type AnthropicHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
}

func NewAnthropicHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder) *AnthropicHandler {
	return &AnthropicHandler{router: r, billing: b, recorder: rec}
}

// Anthropic API request/response types

type anthRequest struct {
	Model       string        `json:"model"`
	Messages    []anthMessage `json:"messages"`
	System      any           `json:"system,omitempty"`
	MaxTokens   int           `json:"max_tokens"`
	Stream      bool          `json:"stream,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	TopP        *float64      `json:"top_p,omitempty"`
	Tools       []anthTool    `json:"tools,omitempty"`
	ToolChoice  any           `json:"tool_choice,omitempty"`
	Metadata    any           `json:"metadata,omitempty"`
	Thinking    any           `json:"thinking,omitempty"`
}

type anthMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type anthTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema,omitempty"`
}

type anthResponse struct {
	ID         string        `json:"id"`
	Type       string        `json:"type"`
	Role       string        `json:"role"`
	Content    []anthContent `json:"content"`
	Model      string        `json:"model"`
	StopReason string        `json:"stop_reason"`
	Usage      anthUsage     `json:"usage"`
}

type anthContent struct {
	Type  string `json:"type"`
	Text  string `json:"text,omitempty"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Input any    `json:"input,omitempty"`
}

type anthUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthErrorResponse struct {
	Type  string        `json:"type"`
	Error anthErrorBody `json:"error"`
}

type anthErrorBody struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (h *AnthropicHandler) HandleMessages(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)

	var req anthRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeAnthError(ctx, 400, "invalid_request_error", "Invalid JSON: "+err.Error())
		return
	}

	if req.Model == "" {
		writeAnthError(ctx, 400, "invalid_request_error", "model is required")
		return
	}
	if req.MaxTokens == 0 {
		writeAnthError(ctx, 400, "invalid_request_error", "max_tokens is required")
		return
	}

	originalModel := req.Model
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}

	chatReq := anthToInternal(&req)

	autoResult := h.router.MaybeResolveAuto(ctx, chatReq.Model, "", extractChatPrompt(chatReq.Messages))
	if autoResult.ReasoningEffort != "" && chatReq.ReasoningEffort == "" {
		chatReq.ReasoningEffort = autoResult.ReasoningEffort
	}
	candidates, err := h.router.ResolveForRequest(originalModel, autoResult.ModelID)
	if err != nil {
		writeAnthError(ctx, 404, "not_found_error", err.Error())
		return
	}

	for i, cand := range candidates {
		chatReq.Model = cand.ModelCfg.ProviderModelID
		start := time.Now()

		if req.Stream {
			if h.tryAnthStream(ctx, chatReq, cand.ModelCfg, cand.Provider, userID, apiKeyID, originalModel, start) {
				h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
				return
			}
		} else {
			if h.tryAnthNonStream(ctx, chatReq, cand.ModelCfg, cand.Provider, userID, apiKeyID, originalModel, start) {
				h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
				return
			}
		}

		h.router.MarkModelUnhealthy(cand.Provider.Name(), cand.ModelCfg.ID)
		if i < len(candidates)-1 {
			log.Printf("anthropic-api fallback: %s/%s failed, trying %s/%s",
				cand.Provider.Name(), cand.ModelCfg.ID,
				candidates[i+1].Provider.Name(), candidates[i+1].ModelCfg.ID)
		}
	}

	writeAnthError(ctx, 502, "api_error", "all providers failed for model "+originalModel)
}

func (h *AnthropicHandler) tryAnthNonStream(
	ctx *fasthttp.RequestCtx,
	req *model.ChatCompletionRequest,
	modelCfg *model.ModelConfig,
	prov provider.Provider,
	userID, apiKeyID, originalModel string,
	start time.Time,
) bool {
	resp, err := prov.ChatCompletion(ctx, req)
	latency := time.Since(start)

	if err != nil {
		statusCode := 502
		if pe, ok := err.(*provider.ProviderError); ok {
			statusCode = pe.StatusCode
			if statusCode == 401 || statusCode == 403 {
				h.recorder.RecordError(userID, apiKeyID, originalModel, prov.Name(),
					int(latency.Milliseconds()), statusCode, pe.Message, false)
				return false
			}
			if !pe.Retryable {
				h.recorder.RecordError(userID, apiKeyID, originalModel, prov.Name(),
					int(latency.Milliseconds()), statusCode, pe.Message, false)
				writeAnthError(ctx, statusCode, "api_error", pe.Message)
				return true
			}
		}
		h.recorder.RecordError(userID, apiKeyID, originalModel, prov.Name(),
			int(latency.Milliseconds()), statusCode, err.Error(), false)
		return false
	}

	var tps float32
	var tokensIn, tokensOut int
	if resp.Usage != nil {
		tokensIn = resp.Usage.PromptTokens
		tokensOut = resp.Usage.CompletionTokens
		if latency.Seconds() > 0 {
			tps = float32(float64(tokensOut) / latency.Seconds())
		}
	}

	cost, _ := h.billing.Deduct(ctx, userID, modelCfg.ID, tokensIn, tokensOut, req.ReasoningEffort, "")
	h.recorder.RecordSuccess(userID, apiKeyID, originalModel, prov.Name(),
		tokensIn, tokensOut, int(latency.Milliseconds()), tps, cost, false)

	anthResp := internalToAnth(resp, originalModel)
	writeJSON(ctx, 200, anthResp)
	return true
}

func (h *AnthropicHandler) tryAnthStream(
	ctx *fasthttp.RequestCtx,
	req *model.ChatCompletionRequest,
	modelCfg *model.ModelConfig,
	prov provider.Provider,
	userID, apiKeyID, originalModel string,
	start time.Time,
) bool {
	streamCh, err := prov.ChatCompletionStream(ctx, req)
	if err != nil {
		if pe, ok := err.(*provider.ProviderError); ok {
			if pe.StatusCode == 401 || pe.StatusCode == 403 {
				return false
			}
			if !pe.Retryable {
				writeAnthError(ctx, pe.StatusCode, "api_error", pe.Message)
				return true
			}
		}
		return false
	}

	ctx.SetContentType("text/event-stream")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.Response.Header.Set("X-Accel-Buffering", "no")

	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		msgID := "msg_" + uuid.New().String()[:12]
		contentIdx := 0

		// message_start
		startMsg := map[string]any{
			"type": "message_start",
			"message": map[string]any{
				"id":          msgID,
				"type":        "message",
				"role":        "assistant",
				"content":     []any{},
				"model":       originalModel,
				"stop_reason": nil,
				"usage": map[string]int{
					"input_tokens":  0,
					"output_tokens": 0,
				},
			},
		}
		writeSSE(w, "message_start", startMsg)

		// content_block_start
		blockStart := map[string]any{
			"type":  "content_block_start",
			"index": contentIdx,
			"content_block": map[string]string{
				"type": "text",
				"text": "",
			},
		}
		writeSSE(w, "content_block_start", blockStart)

		var usage *model.UsageInfo

		for event := range streamCh {
			if event.Err != nil {
				errEvt := map[string]any{
					"type": "error",
					"error": map[string]string{
						"type":    "api_error",
						"message": event.Err.Error(),
					},
				}
				writeSSE(w, "error", errEvt)
				break
			}
			if event.Done {
				usage = event.Usage
				break
			}

			if event.Chunk != nil && len(event.Chunk.Choices) > 0 {
				delta := event.Chunk.Choices[0].Delta
				if delta != nil {
					text := ""
					if s, ok := delta.Content.(string); ok {
						text = s
					}
					if text != "" {
						blockDelta := map[string]any{
							"type":  "content_block_delta",
							"index": contentIdx,
							"delta": map[string]string{
								"type": "text_delta",
								"text": text,
							},
						}
						writeSSE(w, "content_block_delta", blockDelta)
					}
				}
			}
		}

		// content_block_stop
		blockStop := map[string]any{
			"type":  "content_block_stop",
			"index": contentIdx,
		}
		writeSSE(w, "content_block_stop", blockStop)

		// message_delta
		outTokens := 0
		if usage != nil {
			outTokens = usage.CompletionTokens
		}
		msgDelta := map[string]any{
			"type": "message_delta",
			"delta": map[string]string{
				"stop_reason": "end_turn",
			},
			"usage": map[string]int{
				"output_tokens": outTokens,
			},
		}
		writeSSE(w, "message_delta", msgDelta)

		// message_stop
		writeSSE(w, "message_stop", map[string]string{"type": "message_stop"})

		latency := time.Since(start)
		if usage != nil {
			var tps float32
			if latency.Seconds() > 0 {
				tps = float32(float64(usage.CompletionTokens) / latency.Seconds())
			}
			cost, _ := h.billing.Deduct(ctx, userID, modelCfg.ID,
				usage.PromptTokens, usage.CompletionTokens, req.ReasoningEffort, "")
			h.recorder.RecordSuccess(userID, apiKeyID, originalModel, prov.Name(),
				usage.PromptTokens, usage.CompletionTokens,
				int(latency.Milliseconds()), tps, cost, true)
		}
	})
	return true
}

func writeSSE(w *bufio.Writer, event string, data any) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
	w.Flush()
}

func writeAnthError(ctx *fasthttp.RequestCtx, status int, errType, message string) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	resp := anthErrorResponse{
		Type: "error",
		Error: anthErrorBody{
			Type:    errType,
			Message: message,
		},
	}
	data, _ := json.Marshal(resp)
	ctx.SetBody(data)
}

func anthToInternal(req *anthRequest) *model.ChatCompletionRequest {
	chatReq := &model.ChatCompletionRequest{
		Model:           req.Model,
		Stream:          req.Stream,
		Temperature:     req.Temperature,
		TopP:            req.TopP,
		MaxTokens:       &req.MaxTokens,
		ReasoningEffort: parseAnthropicThinking(req.Thinking),
	}

	// System message
	switch s := req.System.(type) {
	case string:
		if s != "" {
			chatReq.Messages = append(chatReq.Messages, model.ChatMessage{
				Role:    "system",
				Content: s,
			})
		}
	case []any:
		// Array of content blocks
		var systemText string
		for _, block := range s {
			if m, ok := block.(map[string]any); ok {
				if t, ok := m["text"].(string); ok {
					systemText += t
				}
			}
		}
		if systemText != "" {
			chatReq.Messages = append(chatReq.Messages, model.ChatMessage{
				Role:    "system",
				Content: systemText,
			})
		}
	}

	for _, msg := range req.Messages {
		chatReq.Messages = append(chatReq.Messages, model.ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	for _, tool := range req.Tools {
		chatReq.Tools = append(chatReq.Tools, model.Tool{
			Type: "function",
			Function: model.ToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	return chatReq
}

func parseAnthropicThinking(thinking any) string {
	m, ok := thinking.(map[string]any)
	if !ok {
		return ""
	}

	thinkingType, _ := m["type"].(string)
	if thinkingType == "disabled" {
		return "none"
	}
	if thinkingType != "" && thinkingType != "enabled" {
		return ""
	}

	budget, ok := numberToInt(m["budget_tokens"])
	if !ok || budget <= 0 {
		return ""
	}
	switch {
	case budget < 2048:
		return "low"
	case budget < 12288:
		return "medium"
	default:
		return "high"
	}
}

func numberToInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float32:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

func internalToAnth(resp *model.ChatCompletionResponse, originalModel string) *anthResponse {
	ar := &anthResponse{
		ID:    "msg_" + uuid.New().String()[:12],
		Type:  "message",
		Role:  "assistant",
		Model: originalModel,
	}

	if resp.Usage != nil {
		ar.Usage = anthUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		}
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		if choice.FinishReason != nil {
			ar.StopReason = mapFinishToAnth(*choice.FinishReason)
		} else {
			ar.StopReason = "end_turn"
		}

		if choice.Message != nil {
			if s, ok := choice.Message.Content.(string); ok && s != "" {
				ar.Content = append(ar.Content, anthContent{
					Type: "text",
					Text: s,
				})
			}
			for _, tc := range choice.Message.ToolCalls {
				var input any
				json.Unmarshal([]byte(tc.Function.Arguments), &input)
				ar.Content = append(ar.Content, anthContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}
		}
	}

	if len(ar.Content) == 0 {
		ar.Content = []anthContent{{Type: "text", Text: ""}}
	}

	return ar
}

func mapFinishToAnth(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	default:
		return "end_turn"
	}
}
