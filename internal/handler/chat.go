package handler

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
	"github.com/openpaths/openpaths/internal/savedresp"
)

type ChatHandler struct {
	router       *router.Router
	billing      *billing.Engine
	recorder     *metrics.Recorder
	userQ        *queries.UserQueries
	providerKeyQ *queries.ProviderKeyQueries
	saver        *savedresp.Saver
}

func NewChatHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder, userQ *queries.UserQueries, providerKeyQ *queries.ProviderKeyQueries) *ChatHandler {
	return &ChatHandler{router: r, billing: b, recorder: rec, userQ: userQ, providerKeyQ: providerKeyQ}
}

// SetResponseSaver wires the optional saved-response service (response saving feature).
func (h *ChatHandler) SetResponseSaver(s *savedresp.Saver) { h.saver = s }

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
	if req.Fusion != nil {
		h.HandleFusionCompletion(ctx, &req, userID, apiKey)
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

	var lastErr error
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
			if attemptErr != nil {
				lastErr = attemptErr
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

	if lastErr != nil {
		writeError(ctx, 502, "provider_error", "all providers failed for model "+originalModel+": "+lastErr.Error())
		return
	}
	writeError(ctx, 502, "provider_error", "all providers failed for model "+originalModel)
}

type fusionPanelResponse struct {
	Model   string `json:"model"`
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

type fusionAnalysis struct {
	Responses []fusionPanelResponse `json:"responses"`
}

func (h *ChatHandler) HandleFusionCompletion(ctx *fasthttp.RequestCtx, req *model.ChatCompletionRequest, userID string, apiKey *model.APIKey) {
	if req.Stream {
		writeError(ctx, 400, "invalid_request", "fusion does not support stream=true yet")
		return
	}
	fusionType := strings.ToLower(strings.TrimSpace(req.Fusion.Type))
	if fusionType == "" {
		fusionType = "openpaths"
	}
	if fusionType == "openrouter" {
		h.handleOpenRouterFusion(ctx, req, userID, apiKey)
		return
	}
	if fusionType != "openpaths" {
		writeError(ctx, 400, "invalid_request", "fusion.type must be openpaths or openrouter")
		return
	}

	panelModels := append([]string(nil), req.Fusion.AnalysisModels...)
	if len(panelModels) == 0 {
		panelModels = []string{"claude-opus-latest", "gpt-5.5", "gemini-latest"}
	}
	if req.Fusion.MaxPanelModels > 0 && req.Fusion.MaxPanelModels < len(panelModels) {
		panelModels = panelModels[:req.Fusion.MaxPanelModels]
	}
	if len(panelModels) > 8 {
		panelModels = panelModels[:8]
	}
	fusionCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}

	responses := make([]fusionPanelResponse, len(panelModels))
	var wg sync.WaitGroup
	for i, panelModel := range panelModels {
		wg.Add(1)
		go func(idx int, modelID string) {
			defer wg.Done()
			content, err := h.runFusionLeg(fusionCtx, req, modelID, userID, apiKeyID)
			responses[idx] = fusionPanelResponse{Model: modelID, Content: content}
			if err != nil {
				responses[idx].Error = err.Error()
			}
		}(i, panelModel)
	}
	wg.Wait()

	successes := 0
	for _, resp := range responses {
		if resp.Error == "" && strings.TrimSpace(resp.Content) != "" {
			successes++
		}
	}
	if successes == 0 {
		writeError(ctx, 502, "provider_error", "all fusion panel models failed")
		return
	}

	judgeModel := strings.TrimSpace(req.Fusion.Model)
	if judgeModel == "" {
		judgeModel = req.Model
	}
	judgeReq := *req
	judgeReq.Model = judgeModel
	judgeReq.Fusion = nil
	judgeReq.Tools = nil
	judgeReq.Messages = buildFusionJudgeMessages(req.Messages, responses, req.Fusion.Prompt)
	if req.Fusion.MaxCompletionTokens > 0 {
		judgeReq.MaxCompletionTokens = &req.Fusion.MaxCompletionTokens
	}

	resp, err := h.runFusionCompletion(fusionCtx, &judgeReq, judgeModel, userID, apiKeyID)
	if err != nil {
		writeError(ctx, 502, "provider_error", "fusion judge failed: "+err.Error())
		return
	}
	resp.Model = req.Model
	writeJSON(ctx, 200, resp)
}

func (h *ChatHandler) handleOpenRouterFusion(ctx *fasthttp.RequestCtx, req *model.ChatCompletionRequest, userID string, apiKey *model.APIKey) {
	prov, err := h.router.Provider("openrouter")
	if err != nil {
		writeError(ctx, 502, "provider_error", "openrouter provider is not configured")
		return
	}
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}
	outerModel := strings.TrimSpace(req.Fusion.Model)
	if outerModel == "" {
		outerModel = req.Model
	}
	orReq := *req
	orReq.Model = outerModel
	orReq.Fusion = nil
	orReq.Tools = []model.Tool{{
		Type: "openrouter:fusion",
		Parameters: map[string]any{
			"analysis_models": req.Fusion.AnalysisModels,
			"model":           outerModel,
		},
	}}
	if req.Fusion.Prompt != "" {
		orReq.Messages = append([]model.ChatMessage{{Role: "system", Content: req.Fusion.Prompt}}, req.Messages...)
	}

	billingModel := h.openRouterFusionBillingModel(outerModel, req.Model)
	if userID != "" && h.billing != nil {
		if err := h.billing.PreCheck(ctx, userID, billingModel, maxOutputTokens(&orReq)); err != nil {
			writeError(ctx, 402, "billing_error", "Insufficient credits. Please add credits to continue.")
			return
		}
	}
	app := requestAppAttribution(ctx)
	start := time.Now()
	resp, err := prov.ChatCompletion(ctx, &orReq)
	latency := time.Since(start)
	if err != nil {
		if h.recorder != nil {
			statusCode := 502
			errMsg := err.Error()
			if pe, ok := err.(*provider.ProviderError); ok {
				statusCode = pe.StatusCode
				errMsg = pe.Message
			}
			h.recorder.RecordErrorWithApp(userID, apiKeyID, req.Model, prov.Name(),
				int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
		}
		writeError(ctx, 502, "provider_error", err.Error())
		return
	}
	resp.Model = req.Model

	tokensIn, tokensOut, cachedTokensIn := 0, 0, 0
	if resp.Usage != nil {
		tokensIn = resp.Usage.PromptTokens
		tokensOut = resp.Usage.CompletionTokens
		cachedTokensIn = resp.Usage.CachedPromptTokens()
	}
	var cost int64
	if userID != "" && h.billing != nil {
		cost, _ = h.billing.DeductWithCachedInput(ctx, userID, billingModel, tokensIn, tokensOut, cachedTokensIn, orReq.ReasoningEffort, "")
	}
	if h.recorder != nil {
		h.recorder.RecordSuccessWithApp(userID, apiKeyID, req.Model, prov.Name(),
			tokensIn, tokensOut, int(latency.Milliseconds()), 0, cost, false, app.ID, app.URL, app.Title, app.Categories)
	}
	saveTextGeneration(h.saver, userID, apiKeyID, req.Model, prov.Name(),
		orReq.Messages, chatResponseText(resp), tokensIn, tokensOut, cost)
	writeJSON(ctx, 200, resp)
}

func (h *ChatHandler) openRouterFusionBillingModel(openRouterModel, requestedModel string) string {
	if cfg, ok := h.router.GetModelConfig(openRouterModel); ok {
		return cfg.ID
	}
	for _, cfg := range h.router.ListModelConfigs() {
		if cfg.Provider == "openrouter" && cfg.ProviderModelID == openRouterModel {
			return cfg.ID
		}
	}
	if cfg, ok := h.router.GetModelConfig(requestedModel); ok {
		return cfg.ID
	}
	return requestedModel
}

func buildFusionJudgeMessages(base []model.ChatMessage, responses []fusionPanelResponse, fusionPrompt string) []model.ChatMessage {
	if strings.TrimSpace(fusionPrompt) == "" {
		fusionPrompt = "You are the fusion judge. Compare the panel responses against the original user request. Keep consensus, resolve contradictions, preserve uniquely valuable insights, discard weak claims, and produce the single best final answer. Do not mention that you are fusing models unless it is directly useful."
	}
	var b strings.Builder
	b.WriteString(fusionPrompt)
	b.WriteString("\n\nOriginal conversation:\n")
	for _, msg := range base {
		if msg.Role == "system" || msg.Role == "user" || msg.Role == "assistant" {
			b.WriteString(strings.ToUpper(msg.Role))
			b.WriteString(": ")
			b.WriteString(fusionMessageContentText(msg.Content))
			b.WriteString("\n")
		}
	}
	b.WriteString("\nPanel responses:\n")
	modelCounts := map[string]int{}
	for _, resp := range responses {
		modelCounts[resp.Model]++
	}
	seen := map[string]int{}
	for _, resp := range responses {
		label := resp.Model
		// When the same model is run multiple times as separate panel variants,
		// number them so the judge can tell the responses apart.
		if modelCounts[resp.Model] > 1 {
			seen[resp.Model]++
			label = fmt.Sprintf("%s (variant %d)", resp.Model, seen[resp.Model])
		}
		b.WriteString("\n--- ")
		b.WriteString(label)
		b.WriteString(" ---\n")
		if resp.Error != "" {
			b.WriteString("ERROR: ")
			b.WriteString(resp.Error)
			b.WriteString("\n")
			continue
		}
		b.WriteString(resp.Content)
		b.WriteString("\n")
	}
	return []model.ChatMessage{{Role: "user", Content: b.String()}}
}

func fusionMessageContentText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var parts []string
		for _, part := range v {
			if m, ok := part.(map[string]any); ok {
				if text, ok := m["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "\n")
	default:
		data, _ := json.Marshal(v)
		return string(data)
	}
}

func (h *ChatHandler) runFusionLeg(ctx context.Context, base *model.ChatCompletionRequest, modelID, userID, apiKeyID string) (string, error) {
	req := *base
	req.Model = modelID
	req.Fusion = nil
	req.Tools = nil
	resp, err := h.runFusionCompletion(ctx, &req, modelID, userID, apiKeyID)
	if err != nil {
		return "", err
	}
	return chatResponseText(resp), nil
}

func (h *ChatHandler) runFusionCompletion(ctx context.Context, req *model.ChatCompletionRequest, requestedModel, userID, apiKeyID string) (*model.ChatCompletionResponse, error) {
	autoResult := h.router.MaybeResolveAutoWithTier(ctx, requestedModel, "", req.TaskTier, extractChatPrompt(req.Messages))
	candidates, err := h.router.ResolveForRequest(requestedModel, autoResult.ModelID)
	if err != nil {
		return nil, err
	}
	for _, cand := range candidates {
		legReq := *req
		legReq.Model = cand.ModelCfg.ProviderModelID
		if userID != "" {
			if err := h.billing.PreCheck(ctx, userID, cand.ModelCfg.ID, maxOutputTokens(&legReq)); err != nil {
				return nil, fmt.Errorf("insufficient credits")
			}
		}
		start := time.Now()
		applySafetyIdentifier(&legReq, cand.Provider, userID)
		resp, err := cand.Provider.ChatCompletion(ctx, &legReq)
		latency := time.Since(start)
		if err != nil {
			h.router.MarkModelUnhealthy(cand.Provider.Name(), cand.ModelCfg.ID)
			continue
		}
		tokensIn, tokensOut, cachedTokensIn := 0, 0, 0
		if resp.Usage != nil {
			tokensIn = resp.Usage.PromptTokens
			tokensOut = resp.Usage.CompletionTokens
			cachedTokensIn = resp.Usage.CachedPromptTokens()
		}
		cost, _ := h.billing.DeductWithCachedInput(ctx, userID, cand.ModelCfg.ID, tokensIn, tokensOut, cachedTokensIn, legReq.ReasoningEffort, "")
		h.recorder.RecordSuccessWithApp(userID, apiKeyID, requestedModel, cand.Provider.Name(), tokensIn, tokensOut, int(latency.Milliseconds()), 0, cost, false, "", "", "", nil)
		h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
		return resp, nil
	}
	return nil, fmt.Errorf("all providers failed for model %s", requestedModel)
}

// safetyIdentifier derives a stable, non-PII identifier for an openpaths end
// user so OpenAI can attribute policy violations to (and block) that individual
// user instead of penalizing the whole openpaths account. It is a one-way hash
// of the user ID, so it stays stable across requests without leaking the raw
// ID. See https://platform.openai.com/docs/guides/safety-best-practices
func safetyIdentifier(userID string) string {
	if userID == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("openpaths:" + userID))
	return "op_" + hex.EncodeToString(sum[:8])
}

// applySafetyIdentifier attaches the end-user safety identifier when the request
// is bound for OpenAI's API (the only provider that understands the field) and
// otherwise strips any value, since the request struct is reused across fallback
// candidates and OpenAI-compatible upstreams reject unknown fields.
func applySafetyIdentifier(req *model.ChatCompletionRequest, prov provider.Provider, userID string) {
	if prov.Name() != "openai" {
		req.SafetyIdentifier = ""
		return
	}
	if id := safetyIdentifier(userID); id != "" {
		req.SafetyIdentifier = id
	}
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

// enforcePrepaid blocks a request that would hit an upstream provider on
// OpenPaths' own key (non-BYOK) when the user lacks sufficient balance. BYOK
// attempts are exempt because the user pays their own provider directly. This is
// the hard prepay gate: the BalanceCheck middleware is skipped for any user with
// a BYOK key loaded, so without this a $0-balance user holding an unrelated BYOK
// key could route to our Anthropic/OpenAI keys for free. Returns true (and writes
// 402) when the request is rejected, which stops the fallback loop.
func (h *ChatHandler) enforcePrepaid(ctx *fasthttp.RequestCtx, req *model.ChatCompletionRequest, modelID string, byok bool) bool {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	if byok || userID == "" {
		return false
	}
	if err := h.billing.PreCheck(ctx, userID, modelID, maxOutputTokens(req)); err != nil {
		writeError(ctx, 402, "billing_error", "Insufficient credits. Please add credits to continue.")
		return true
	}
	return false
}

// maxOutputTokens returns the worst-case output token count the request can
// produce, so the prepay gate reserves enough balance to cover the full
// completion rather than a token or two. Falls back to a conservative default
// when the caller did not cap output.
func maxOutputTokens(req *model.ChatCompletionRequest) int {
	if req.MaxCompletionTokens != nil && *req.MaxCompletionTokens > 0 {
		return *req.MaxCompletionTokens
	}
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		return *req.MaxTokens
	}
	return 4096
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
	if h.enforcePrepaid(ctx, req, modelCfg.ID, byok) {
		return true, nil
	}
	app := requestAppAttribution(ctx)
	applySafetyIdentifier(req, prov, userID)
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
				h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, prov.Name(),
					int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
				// A 404 means the upstream no longer serves this model (deprecation
				// or outage, e.g. Fable returning "not available, use Opus 4.8"). Fall
				// through to the next fallback candidate instead of failing hard, so a
				// configured fallback_models chain can take over.
				if statusCode == 404 {
					return false, err
				}
				writeError(ctx, statusCode, "provider_error", errMsg)
				return true, err
			}
		}
		h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, prov.Name(),
			int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
		return false, err
	}

	resp.Model = originalModel

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

	var cost int64
	if !byok {
		cost, _ = h.billing.DeductWithCachedInput(ctx, userID, modelCfg.ID,
			tokensIn, tokensOut, cachedTokensIn, req.ReasoningEffort, "")
	}
	h.recorder.RecordSuccessWithApp(userID, apiKeyID, originalModel, prov.Name(),
		tokensIn, tokensOut, int(latency.Milliseconds()), tps, cost, false, app.ID, app.URL, app.Title, app.Categories)

	saveTextGeneration(h.saver, userID, apiKeyID, originalModel, prov.Name(),
		req.Messages, chatResponseText(resp), tokensIn, tokensOut, cost)

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
	if h.enforcePrepaid(ctx, req, modelCfg.ID, byok) {
		return true, nil
	}
	app := requestAppAttribution(ctx)
	applySafetyIdentifier(req, prov, userID)
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
				// See tryNonStreamingBYOK: fall through on 404 so the fallback
				// chain (e.g. Fable -> Opus during a Fable outage) can take over.
				if statusCode == 404 {
					return false, err
				}
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

	wantSave := h.saver.WantText(userID)
	ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
		var usage *model.UsageInfo
		var outBuf strings.Builder

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
			if wantSave && len(event.Chunk.Choices) > 0 && event.Chunk.Choices[0].Delta != nil {
				outBuf.WriteString(messageContentText(event.Chunk.Choices[0].Delta.Content))
			}
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
				cost, _ = h.billing.DeductWithCachedInput(ctx, userID, modelCfg.ID,
					usage.PromptTokens, usage.CompletionTokens, usage.CachedPromptTokens(), req.ReasoningEffort, "")
			}
			h.recorder.RecordSuccessWithApp(userID, apiKeyID, originalModel, prov.Name(),
				usage.PromptTokens, usage.CompletionTokens,
				int(latency.Milliseconds()), tps, cost, true, app.ID, app.URL, app.Title, app.Categories)
			if wantSave {
				saveTextGeneration(h.saver, userID, apiKeyID, originalModel, prov.Name(),
					req.Messages, outBuf.String(), usage.PromptTokens, usage.CompletionTokens, cost)
			}
		} else {
			// The upstream call happened (and was billed to us) but ended without a
			// usage frame — e.g. client disconnect mid-stream. Leave a DB trace so the
			// request can be reconciled against the provider invoice instead of
			// silently vanishing from usage_logs.
			h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, prov.Name(),
				int(latency.Milliseconds()), 200, "stream_no_usage", true, app.ID, app.URL, app.Title, app.Categories)
		}
	})
	return true, nil
}
