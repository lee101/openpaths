package handler

import (
	"encoding/json"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/audio"
	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/modelaccess"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
)

type SpeechHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
	emotions *audio.AutoEmotion
	accessQ  modelaccess.Checker
}

func NewSpeechHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder) *SpeechHandler {
	return &SpeechHandler{router: r, billing: b, recorder: rec}
}

func (h *SpeechHandler) SetAutoEmotion(marker *audio.AutoEmotion) {
	h.emotions = marker
}

func (h *SpeechHandler) SetAccess(checker modelaccess.Checker) { h.accessQ = checker }

func (h *SpeechHandler) HandleSpeechGeneration(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}
	app := requestAppAttribution(ctx)

	var req model.SpeechRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON: "+err.Error())
		return
	}
	if req.Model == "" && string(ctx.Path()) == "/v1/tts" {
		req.Model = "xai-tts"
	}
	if req.Input == "" {
		req.Input = req.Text
	}
	if req.Voice == "" {
		req.Voice = req.VoiceID
	}
	if req.Model == "" {
		writeError(ctx, 400, "invalid_request", "model is required")
		return
	}
	if req.Input == "" {
		writeError(ctx, 400, "invalid_request", "input is required")
		return
	}
	if req.AutoEmotion {
		if h.emotions == nil {
			log.Printf("speech: auto_emotion requested but not configured on this server; continuing without expressive tags")
		} else {
			marked, err := h.emotions.Markup(ctx, req.Input)
			if err != nil {
				writeError(ctx, 500, "auto_emotion_error", err.Error())
				return
			}
			req.Input = marked
			req.Text = marked
		}
	}

	if prepaidGate(ctx, h.billing, req.Model, 0) {
		return
	}

	originalModel := req.Model
	candidates, err := h.router.ResolveWithRetries(req.Model)
	if err != nil {
		writeError(ctx, 404, "model_not_found", err.Error())
		return
	}
	candidates, err = modelaccess.FilterCandidates(ctx, h.accessQ, userID, originalModel, candidates)
	if err != nil {
		writeError(ctx, 403, "model_not_permitted", err.Error())
		return
	}

	for i, cand := range candidates {
		speechProv, ok := cand.Provider.(provider.SpeechProvider)
		if !ok {
			log.Printf("speech: %s does not support speech", cand.Provider.Name())
			continue
		}

		req.Model = cand.ModelCfg.ProviderModelID
		start := time.Now()

		resp, err := speechProv.GenerateSpeech(ctx, &req)
		latency := time.Since(start)

		if err != nil {
			statusCode := 502
			errMsg := "upstream error"
			if pe, ok := err.(*provider.ProviderError); ok {
				statusCode = pe.StatusCode
				errMsg = pe.Message
				if !pe.Retryable {
					h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
						int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
					writeError(ctx, statusCode, "provider_error", errMsg)
					return
				}
			}
			h.router.MarkModelUnhealthy(cand.Provider.Name(), cand.ModelCfg.ID)
			h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
				int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
			if i < len(candidates)-1 {
				log.Printf("speech fallback: %s/%s -> %s/%s",
					cand.Provider.Name(), cand.ModelCfg.ID,
					candidates[i+1].Provider.Name(), candidates[i+1].ModelCfg.ID)
			}
			continue
		}

		h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
		characters := utf8.RuneCountInString(req.Input)
		inputTokens := resp.InputTokens
		if inputTokens == 0 {
			inputTokens = estimateSpeechInputTokens(req.Input)
		}
		outputTokens := resp.OutputTokens
		var cost int64
		if cand.ModelCfg.PricePer1MCharacters > 0 {
			cost, _ = h.billing.DeductCharacters(ctx, userID, cand.ModelCfg.ID, characters, "")
		} else {
			if inputTokens == 0 {
				inputTokens = 1
			}
			cost, _ = h.billing.Deduct(ctx, userID, cand.ModelCfg.ID, inputTokens, outputTokens, "", "")
		}
		h.recorder.RecordSuccessWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
			inputTokens, outputTokens, int(latency.Milliseconds()), 0, cost, false, app.ID, app.URL, app.Title, app.Categories)

		writeJSON(ctx, 200, resp)
		return
	}

	writeError(ctx, 502, "provider_error", "all providers failed for model "+originalModel)
}

func estimateSpeechInputTokens(input string) int {
	input = strings.TrimSpace(input)
	if input == "" {
		return 0
	}
	return max(1, (utf8.RuneCountInString(input)+3)/4)
}
