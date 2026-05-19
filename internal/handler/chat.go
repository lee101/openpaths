package handler

import (
	"bufio"
	"encoding/json"
	"log"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
)

type ChatHandler struct {
	router       *router.Router
	billing      *billing.Engine
	recorder     *metrics.Recorder
	userQ        *queries.UserQueries
	providerKeyQ *queries.ProviderKeyQueries
}

func NewChatHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder, userQ *queries.UserQueries, providerKeyQ *queries.ProviderKeyQueries) *ChatHandler {
	return &ChatHandler{router: r, billing: b, recorder: rec, userQ: userQ, providerKeyQ: providerKeyQ}
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

	autoResult := h.router.MaybeResolveAutoWithTier(ctx, req.Model, "", req.TaskTier, extractChatPrompt(req.Messages))
	useAutoReasoning := router.IsAutoReasoningEffort(req.ReasoningEffort) ||
		(req.Thinking != nil && router.IsAutoReasoningEffort(req.Thinking.Type))
	if autoResult.ReasoningEffort != "" && (req.ReasoningEffort == "" || useAutoReasoning || router.IsAutoThinkModel(originalModel) || req.TaskTier == "think") {
		req.ReasoningEffort = autoResult.ReasoningEffort
	} else if useAutoReasoning {
		req.ReasoningEffort = h.router.MaybeResolveAutoReasoning(ctx, extractChatPrompt(req.Messages))
		if req.ReasoningEffort == "" {
			req.ReasoningEffort = "medium"
		}
	}
	if req.Thinking != nil && router.IsAutoReasoningEffort(req.Thinking.Type) {
		req.Thinking = nil
	}
	candidates, err := h.router.ResolveForRequest(originalModel, autoResult.ModelID)
	if err != nil {
		writeError(ctx, 404, "model_not_found", err.Error())
		return
	}

	for i, cand := range candidates {
		req.Model = cand.ModelCfg.ProviderModelID
		attempts := getProviderAttempts(ctx, userID, cand.Provider.Name())
		if len(attempts) > 0 && cand.Provider.Name() == "openai" {
			attempts = append(attempts, selectedProvider{provider: cand.Provider})
		}
		if len(attempts) == 0 {
			attempts = []selectedProvider{{provider: cand.Provider}}
		}

		for j, attempt := range attempts {
			start := time.Now()
			var handled bool
			var attemptErr error
			if req.Stream {
				handled, attemptErr = h.tryStreamingBYOK(ctx, &req, cand.ModelCfg, attempt.provider, userID, apiKeyID, originalModel, start, attempt.byok)
			} else {
				handled, attemptErr = h.tryNonStreamingBYOK(ctx, &req, cand.ModelCfg, attempt.provider, userID, apiKeyID, originalModel, start, attempt.byok)
			}
			if handled {
				if attemptErr == nil && attempt.cred != nil {
					markOpenAIMaxPlanCredentialHealthy(attempt.cred.ID)
				}
				h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
				return
			}
			if attempt.cred != nil {
				h.handleOpenAIMaxPlanCredentialFailure(ctx, userID, attempt.cred, attemptErr)
				if j < len(attempts)-1 {
					log.Printf("openai max plan credential fallback: %s failed, trying next credential/key", attempt.cred.Label)
				}
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
	ok, _ := h.tryNonStreamingBYOK(ctx, req, modelCfg, prov, userID, apiKeyID, originalModel, start, false)
	return ok
}

func (h *ChatHandler) tryNonStreamingBYOK(
	ctx *fasthttp.RequestCtx,
	req *model.ChatCompletionRequest,
	modelCfg *model.ModelConfig,
	prov provider.Provider,
	userID, apiKeyID, originalModel string,
	start time.Time,
	byok bool,
) (bool, error) {
	resp, err := prov.ChatCompletion(ctx, req)
	latency := time.Since(start)

	if err != nil {
		statusCode := 502
		errMsg := "Upstream error"
		if pe, ok := err.(*provider.ProviderError); ok {
			statusCode = pe.StatusCode
			errMsg = pe.Message
			if statusCode == 401 || statusCode == 403 {
				return false, err
			}
			if pe.Retryable {
				return false, err
			}
			if !pe.Retryable {
				h.recorder.RecordError(userID, apiKeyID, originalModel, prov.Name(),
					int(latency.Milliseconds()), statusCode, errMsg, false)
				writeError(ctx, statusCode, "provider_error", errMsg)
				return true, err
			}
		}
		h.recorder.RecordError(userID, apiKeyID, originalModel, prov.Name(),
			int(latency.Milliseconds()), statusCode, errMsg, false)
		return false, err
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

	var cost int64
	if !byok {
		cost, _ = h.billing.Deduct(ctx, userID, modelCfg.ID, tokensIn, tokensOut, req.ReasoningEffort, "")
	}
	h.recorder.RecordSuccess(userID, apiKeyID, originalModel, prov.Name(),
		tokensIn, tokensOut, int(latency.Milliseconds()), tps, cost, false)

	writeJSON(ctx, 200, resp)
	return true, nil
}

func (h *ChatHandler) tryStreaming(
	ctx *fasthttp.RequestCtx,
	req *model.ChatCompletionRequest,
	modelCfg *model.ModelConfig,
	prov provider.Provider,
	userID, apiKeyID, originalModel string,
	start time.Time,
) bool {
	ok, _ := h.tryStreamingBYOK(ctx, req, modelCfg, prov, userID, apiKeyID, originalModel, start, false)
	return ok
}

func (h *ChatHandler) tryStreamingBYOK(
	ctx *fasthttp.RequestCtx,
	req *model.ChatCompletionRequest,
	modelCfg *model.ModelConfig,
	prov provider.Provider,
	userID, apiKeyID, originalModel string,
	start time.Time,
	byok bool,
) (bool, error) {
	streamCh, err := prov.ChatCompletionStream(ctx, req)
	if err != nil {
		statusCode := 502
		errMsg := "Stream initialization failed"
		if pe, ok := err.(*provider.ProviderError); ok {
			statusCode = pe.StatusCode
			errMsg = pe.Message
			if statusCode == 401 || statusCode == 403 {
				return false, err
			}
			if pe.Retryable {
				return false, err
			}
			if !pe.Retryable {
				writeError(ctx, statusCode, "provider_error", errMsg)
				return true, err
			}
		}
		return false, err
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
			var cost int64
			if !byok {
				cost, _ = h.billing.Deduct(ctx, userID, modelCfg.ID,
					usage.PromptTokens, usage.CompletionTokens, req.ReasoningEffort, "")
			}
			h.recorder.RecordSuccess(userID, apiKeyID, originalModel, prov.Name(),
				usage.PromptTokens, usage.CompletionTokens,
				int(latency.Milliseconds()), tps, cost, true)
		}
	})
	return true, nil
}
