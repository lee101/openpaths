package handler

import (
	"bufio"
	"encoding/json"
	"log"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
)

type ChatHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
}

func NewChatHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder) *ChatHandler {
	return &ChatHandler{router: r, billing: b, recorder: rec}
}

func (h *ChatHandler) HandleChatCompletion(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)

	var req model.ChatCompletionRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON: "+err.Error())
		return
	}

	if req.Model == "" {
		writeError(ctx, 400, "invalid_request", "model is required")
		return
	}

	originalModel := req.Model
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}

	autoResult := h.router.MaybeResolveAuto(ctx, req.Model, "text", extractChatPrompt(req.Messages))
	if autoResult.ReasoningEffort != "" && req.ReasoningEffort == "" {
		req.ReasoningEffort = autoResult.ReasoningEffort
	}
	candidates, err := h.router.ResolveWithRetries(autoResult.ModelID)
	if err != nil {
		writeError(ctx, 404, "model_not_found", err.Error())
		return
	}

	for i, cand := range candidates {
		req.Model = cand.ModelCfg.ProviderModelID
		start := time.Now()

		if req.Stream {
			if h.tryStreaming(ctx, &req, cand.ModelCfg, cand.Provider, userID, apiKeyID, originalModel, start) {
				h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
				return
			}
		} else {
			if h.tryNonStreaming(ctx, &req, cand.ModelCfg, cand.Provider, userID, apiKeyID, originalModel, start) {
				h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
				return
			}
		}

		h.router.MarkModelUnhealthy(cand.Provider.Name(), cand.ModelCfg.ID)
		if i < len(candidates)-1 {
			log.Printf("fallback: %s/%s failed, trying %s/%s",
				cand.Provider.Name(), cand.ModelCfg.ID,
				candidates[i+1].Provider.Name(), candidates[i+1].ModelCfg.ID)
		}
	}

	writeError(ctx, 502, "provider_error", "all providers failed for model "+originalModel)
}

func extractChatPrompt(messages []model.ChatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			if s, ok := messages[i].Content.(string); ok {
				return s
			}
		}
	}
	return ""
}

func (h *ChatHandler) tryNonStreaming(
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
		errMsg := "Upstream error"
		if pe, ok := err.(*provider.ProviderError); ok {
			statusCode = pe.StatusCode
			errMsg = pe.Message
			if statusCode == 401 || statusCode == 403 {
				h.recorder.RecordError(userID, apiKeyID, originalModel, prov.Name(),
					int(latency.Milliseconds()), statusCode, errMsg, false)
				return false
			}
			if !pe.Retryable {
				h.recorder.RecordError(userID, apiKeyID, originalModel, prov.Name(),
					int(latency.Milliseconds()), statusCode, errMsg, false)
				writeError(ctx, statusCode, "provider_error", errMsg)
				return true
			}
		}
		h.recorder.RecordError(userID, apiKeyID, originalModel, prov.Name(),
			int(latency.Milliseconds()), statusCode, errMsg, false)
		return false
	}

	resp.Model = originalModel

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

	writeJSON(ctx, 200, resp)
	return true
}

func (h *ChatHandler) tryStreaming(
	ctx *fasthttp.RequestCtx,
	req *model.ChatCompletionRequest,
	modelCfg *model.ModelConfig,
	prov provider.Provider,
	userID, apiKeyID, originalModel string,
	start time.Time,
) bool {
	streamCh, err := prov.ChatCompletionStream(ctx, req)
	if err != nil {
		statusCode := 502
		errMsg := "Stream initialization failed"
		if pe, ok := err.(*provider.ProviderError); ok {
			statusCode = pe.StatusCode
			errMsg = pe.Message
			if statusCode == 401 || statusCode == 403 {
				return false
			}
			if !pe.Retryable {
				writeError(ctx, statusCode, "provider_error", errMsg)
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
		var usage *model.UsageInfo

		for event := range streamCh {
			if event.Err != nil {
				errData, _ := json.Marshal(model.ErrorResponse{
					Error: model.ErrorDetail{Message: event.Err.Error(), Type: "provider_error"},
				})
				w.WriteString("data: ")
				w.Write(errData)
				w.WriteString("\n\n")
				w.Flush()
				break
			}
			if event.Done {
				usage = event.Usage
				w.WriteString("data: [DONE]\n\n")
				w.Flush()
				break
			}

			event.Chunk.Model = originalModel
			data, _ := json.Marshal(event.Chunk)
			w.WriteString("data: ")
			w.Write(data)
			w.WriteString("\n\n")
			w.Flush()
		}

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
