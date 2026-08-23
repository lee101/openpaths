package handler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/guardrails"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/modelaccess"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
	"github.com/openpaths/openpaths/internal/savedresp"
)

type AnthropicHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
	saver    *savedresp.Saver
	accessQ  *queries.AccessQueries
}

func NewAnthropicHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder) *AnthropicHandler {
	return &AnthropicHandler{router: r, billing: b, recorder: rec}
}

// SetAccess wires the optional Model IAM policy store (default-open when unset).
func (h *AnthropicHandler) SetAccess(a *queries.AccessQueries) { h.accessQ = a }

// SetResponseSaver wires the optional saved-response service (response saving feature).
func (h *AnthropicHandler) SetResponseSaver(s *savedresp.Saver) { h.saver = s }

// Anthropic API request/response types

type anthRequest struct {
	Model        string        `json:"model"`
	Messages     []anthMessage `json:"messages"`
	System       any           `json:"system,omitempty"`
	MaxTokens    int           `json:"max_tokens"`
	Stream       bool          `json:"stream,omitempty"`
	Temperature  *float64      `json:"temperature,omitempty"`
	TopP         *float64      `json:"top_p,omitempty"`
	Tools        []anthTool    `json:"tools,omitempty"`
	ToolChoice   any           `json:"tool_choice,omitempty"`
	Metadata     any           `json:"metadata,omitempty"`
	Thinking     any           `json:"thinking,omitempty"`
	OutputConfig struct {
		Effort string `json:"effort,omitempty"`
	} `json:"output_config,omitempty"`

	// Non-standard cross-provider hints — see model.ChatCompletionRequest.
	TaskTier        string `json:"task_tier,omitempty"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	RoutingStrategy string `json:"routing_strategy,omitempty"`
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
	if h.accessQ != nil {
		if ok, reason := h.accessQ.ModelAllowed(ctx, userID, originalModel); !ok {
			writeAnthError(ctx, 403, "permission_error", reason)
			return
		}
	}
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}

	chatReq := anthToInternal(&req)

	autoResult := h.router.MaybeResolveAutoWithTier(ctx, chatReq.Model, "", chatReq.TaskTier, extractChatPrompt(chatReq.Messages))
	useAutoReasoning := router.IsAutoReasoningEffort(chatReq.ReasoningEffort)
	if autoResult.ReasoningEffort != "" && (chatReq.ReasoningEffort == "" || useAutoReasoning || router.IsAutoThinkModel(originalModel) || chatReq.TaskTier == "think") {
		chatReq.ReasoningEffort = autoResult.ReasoningEffort
	} else if useAutoReasoning {
		chatReq.ReasoningEffort = h.router.MaybeResolveAutoReasoning(ctx, extractChatPrompt(chatReq.Messages))
		if chatReq.ReasoningEffort == "" {
			chatReq.ReasoningEffort = "medium"
		}
	}
	candidates, err := h.router.ResolveForRequest(originalModel, autoResult.ModelID)
	if err != nil {
		writeAnthError(ctx, 404, "not_found_error", err.Error())
		return
	}
	candidates = router.OrderCandidates(candidates, chatReq.RoutingStrategy)
	candidates, err = modelaccess.FilterCandidates(ctx, h.accessQ, userID, originalModel, candidates)
	if err != nil {
		writeAnthError(ctx, 403, "permission_error", err.Error())
		return
	}
	if providers, blocked := middleware.GuardrailProviders(ctx), middleware.GuardrailBlockedProviders(ctx); len(providers) > 0 || len(blocked) > 0 {
		candidates, err = guardrails.FilterRouteCandidates(candidates, providers, blocked)
		if err != nil {
			middleware.NotifyGuardrailAccessViolation(ctx, "OpenPaths guardrail: provider access blocked", err.Error())
			writeAnthError(ctx, 403, "permission_error", err.Error())
			return
		}
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

// enforcePrepaid blocks the request when the user lacks sufficient balance.
// The /v1/messages path always bills to OpenPaths' own key (no BYOK), and the
// BalanceCheck middleware is skipped for any user holding a BYOK key, so this
// is the hard prepay gate for this endpoint. Returns true (and writes 402) when
// rejected, which stops the fallback loop.
func (h *AnthropicHandler) enforcePrepaid(ctx *fasthttp.RequestCtx, req *model.ChatCompletionRequest, modelID string) bool {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if userID == "" {
		return false
	}
	if err := h.billing.PreCheck(ctx, userID, modelID, maxOutputTokens(req)); err != nil {
		writeAnthError(ctx, 402, "billing_error", "Insufficient credits. Please add credits to continue.")
		return true
	}
	return false
}

func (h *AnthropicHandler) tryAnthNonStream(
	ctx *fasthttp.RequestCtx,
	req *model.ChatCompletionRequest,
	modelCfg *model.ModelConfig,
	prov provider.Provider,
	userID, apiKeyID, originalModel string,
	start time.Time,
) bool {
	if h.enforcePrepaid(ctx, req, modelCfg.ID) {
		return true
	}
	app := requestAppAttribution(ctx)
	applySafetyIdentifier(req, prov, userID)
	resp, err := prov.ChatCompletion(ctx, req)
	latency := time.Since(start)

	if err != nil {
		statusCode := 502
		if pe, ok := err.(*provider.ProviderError); ok {
			statusCode = pe.StatusCode
			if statusCode == 401 || statusCode == 403 {
				h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, prov.Name(),
					int(latency.Milliseconds()), statusCode, pe.Message, false, app.ID, app.URL, app.Title, app.Categories)
				return false
			}
			if !pe.Retryable {
				h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, prov.Name(),
					int(latency.Milliseconds()), statusCode, pe.Message, false, app.ID, app.URL, app.Title, app.Categories)
				// 404 means the upstream no longer serves this model; fall through to
				// the next fallback candidate like the chat handler does.
				if statusCode == 404 {
					return false
				}
				writeAnthError(ctx, statusCode, "api_error", pe.Message)
				return true
			}
		}
		h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, prov.Name(),
			int(latency.Milliseconds()), statusCode, err.Error(), false, app.ID, app.URL, app.Title, app.Categories)
		return false
	}

	var tps float32
	var tokensIn, tokensOut, cachedTokensIn int
	if resp.Usage != nil {
		tokensIn = resp.Usage.PromptTokens
		tokensOut = resp.Usage.CompletionTokens
		cachedTokensIn = resp.Usage.CachedPromptTokens()
		if latency.Seconds() > 0 {
			tps = float32(float64(tokensOut) / latency.Seconds())
		}
	}

	cost, _ := h.billing.DeductWithCachedInput(ctx, userID, modelCfg.ID,
		tokensIn, tokensOut, cachedTokensIn, req.ReasoningEffort, "")
	h.recorder.RecordSuccessWithApp(userID, apiKeyID, originalModel, prov.Name(),
		tokensIn, tokensOut, int(latency.Milliseconds()), tps, cost, false, app.ID, app.URL, app.Title, app.Categories)

	saveTextGeneration(h.saver, userID, apiKeyID, originalModel, prov.Name(),
		req.Messages, chatResponseText(resp), tokensIn, tokensOut, cost)

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
	if h.enforcePrepaid(ctx, req, modelCfg.ID) {
		return true
	}
	app := requestAppAttribution(ctx)
	applySafetyIdentifier(req, prov, userID)
	streamCh, err := prov.ChatCompletionStream(ctx, req)
	if err != nil {
		if pe, ok := err.(*provider.ProviderError); ok {
			if pe.StatusCode == 401 || pe.StatusCode == 403 || pe.StatusCode == 404 {
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
		var outBuf strings.Builder

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
						outBuf.WriteString(text)
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
			cost, _ := h.billing.DeductWithCachedInput(ctx, userID, modelCfg.ID,
				usage.PromptTokens, usage.CompletionTokens, usage.CachedPromptTokens(), req.ReasoningEffort, "")
			h.recorder.RecordSuccessWithApp(userID, apiKeyID, originalModel, prov.Name(),
				usage.PromptTokens, usage.CompletionTokens,
				int(latency.Milliseconds()), tps, cost, true, app.ID, app.URL, app.Title, app.Categories)
			saveTextGeneration(h.saver, userID, apiKeyID, originalModel, prov.Name(),
				req.Messages, outBuf.String(), usage.PromptTokens, usage.CompletionTokens, cost)
		} else {
			// Upstream call happened (billed to us) but ended without a usage frame
			// — e.g. client disconnect. Leave a DB trace for invoice reconciliation.
			h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, prov.Name(),
				int(latency.Milliseconds()), 200, "stream_no_usage", true, app.ID, app.URL, app.Title, app.Categories)
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
		TaskTier:        req.TaskTier,
		RoutingStrategy: req.RoutingStrategy,
	}
	if req.ReasoningEffort != "" {
		chatReq.ReasoningEffort = req.ReasoningEffort
	} else if req.OutputConfig.Effort != "" {
		chatReq.ReasoningEffort = req.OutputConfig.Effort
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
			Function: &model.ToolFunction{
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
	if router.IsAutoReasoningEffort(thinkingType) {
		return "auto"
	}
	if thinkingType == "adaptive" {
		return "auto"
	}
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
