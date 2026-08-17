package handler

import (
	"log"
	"strings"
	"time"

	"github.com/valyala/fasthttp"

	audioinfo "github.com/openpaths/openpaths/internal/audio"
	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
)

var modelToProvider = map[string]string{
	// Groq
	"whisper-large-v3":           "groq",
	"whisper-large-v3-turbo":     "groq",
	"distil-whisper-large-v3-en": "groq",
	// OpenAI
	"gpt-transcribe":         "openai",
	"whisper-1":              "openai",
	"gpt-4o-transcribe":      "openai",
	"gpt-4o-mini-transcribe": "openai",
	// Fireworks
	"whisper-v3-large":       "fireworks",
	"whisper-v3-large-turbo": "fireworks",
	// Fal
	"fal-ai/whisper": "fal",
	// Local
	"local-whisper": "local-whisper",
}

type TranscriptionHandler struct {
	router      *router.Router
	billing     *billing.Engine
	providers   []provider.TranscriptionProvider
	providerMap map[string]provider.TranscriptionProvider
	health      *router.HealthTracker
	recorder    *metrics.Recorder
}

func NewTranscriptionHandler(r *router.Router, b *billing.Engine, providers []provider.TranscriptionProvider, rec *metrics.Recorder) *TranscriptionHandler {
	pm := make(map[string]provider.TranscriptionProvider, len(providers))
	for _, p := range providers {
		pm[p.Name()] = p
	}
	return &TranscriptionHandler{
		router:      r,
		billing:     b,
		providers:   providers,
		providerMap: pm,
		health:      r.HealthTracker(),
		recorder:    rec,
	}
}

func (h *TranscriptionHandler) HandleTranscription(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		writeError(ctx, 400, "invalid_request", "multipart form required")
		return
	}

	files := form.File["file"]
	if len(files) == 0 {
		writeError(ctx, 400, "invalid_request", "file is required")
		return
	}

	fh := files[0]
	f, err := fh.Open()
	if err != nil {
		writeError(ctx, 400, "invalid_request", "cannot read file")
		return
	}
	defer f.Close()

	fileData := make([]byte, fh.Size)
	if _, err := f.Read(fileData); err != nil {
		writeError(ctx, 400, "invalid_request", "cannot read file data")
		return
	}

	reqModel := ""
	if vals, ok := form.Value["model"]; ok && len(vals) > 0 {
		reqModel = vals[0]
	}
	if reqModel == "" && string(ctx.Path()) == "/v1/stt" {
		reqModel = "xai-stt"
	}
	lang := ""
	if vals, ok := form.Value["language"]; ok && len(vals) > 0 {
		lang = vals[0]
	}
	prompt := ""
	if vals, ok := form.Value["prompt"]; ok && len(vals) > 0 {
		prompt = vals[0]
	}
	respFmt := "json"
	if vals, ok := form.Value["response_format"]; ok && len(vals) > 0 {
		respFmt = vals[0]
	}

	req := &model.TranscriptionRequest{
		File:     fileData,
		Filename: fh.Filename,
		Model:    reqModel,
		Language: lang,
		Prompt:   prompt,
		Format:   respFmt,
	}

	originalModel := reqModel
	if originalModel == "" || originalModel == "auto" {
		originalModel = "whisper"
	}
	if prepaidGate(ctx, h.billing, originalModel, 0) {
		return
	}
	if reqModel != "" && reqModel != "auto" {
		if h.handleRoutedTranscription(ctx, userID, apiKeyID, reqModel, req) {
			return
		}
	}

	h.handleFallbackTranscription(ctx, userID, apiKeyID, req, originalModel)
}

func (h *TranscriptionHandler) handleRoutedTranscription(ctx *fasthttp.RequestCtx, userID, apiKeyID, originalModel string, req *model.TranscriptionRequest) bool {
	app := requestAppAttribution(ctx)
	candidates, err := h.router.ResolveWithRetries(originalModel)
	if err != nil {
		return false
	}

	for i, cand := range candidates {
		transcriptionProv, ok := cand.Provider.(provider.TranscriptionProvider)
		if !ok {
			transcriptionProv, ok = h.transcriberByName(cand.Provider.Name())
			if !ok {
				log.Printf("transcription: %s does not support transcription", cand.Provider.Name())
				continue
			}
		}

		req.Model = cand.ModelCfg.ProviderModelID
		start := time.Now()
		resp, err := transcriptionProv.Transcribe(ctx, req)
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
					return true
				}
			}
			h.router.MarkModelUnhealthy(cand.Provider.Name(), cand.ModelCfg.ID)
			h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
				int(latency.Milliseconds()), statusCode, errMsg, false, app.ID, app.URL, app.Title, app.Categories)
			if i < len(candidates)-1 {
				log.Printf("transcription fallback: %s/%s -> %s/%s",
					cand.Provider.Name(), cand.ModelCfg.ID,
					candidates[i+1].Provider.Name(), candidates[i+1].ModelCfg.ID)
			}
			continue
		}

		h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
		durationSeconds := audioinfo.EstimateDurationSeconds(req.Filename, req.File)
		cost, _ := h.billing.DeductAudio(ctx, userID, cand.ModelCfg.ID, durationSeconds, "")
		h.recorder.RecordSuccessWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
			durationSeconds, 0, int(latency.Milliseconds()), 0, cost, false, app.ID, app.URL, app.Title, app.Categories)
		writeJSON(ctx, 200, resp)
		return true
	}

	writeError(ctx, 502, "provider_error", "all providers failed for model "+originalModel)
	return true
}

func (h *TranscriptionHandler) transcriberByName(name string) (provider.TranscriptionProvider, bool) {
	p, ok := h.providerMap[name]
	return p, ok
}

func (h *TranscriptionHandler) handleFallbackTranscription(ctx *fasthttp.RequestCtx, userID, apiKeyID string, req *model.TranscriptionRequest, originalModel string) {
	app := requestAppAttribution(ctx)
	ordered := h.buildProviderOrder(strings.ToLower(req.Model))

	for i, p := range ordered {
		key := "transcription:" + p.Name()
		if !h.health.IsHealthy(key) {
			continue
		}

		start := time.Now()
		resp, err := p.Transcribe(ctx, req)
		latency := time.Since(start)

		if err != nil {
			h.health.MarkUnhealthy(key)
			log.Printf("transcription: %s failed (%dms): %v", p.Name(), latency.Milliseconds(), err)
			if pe, ok := err.(*provider.ProviderError); ok && !pe.Retryable {
				h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, p.Name(),
					int(latency.Milliseconds()), pe.StatusCode, pe.Message, false, app.ID, app.URL, app.Title, app.Categories)
				writeError(ctx, pe.StatusCode, "provider_error", pe.Message)
				return
			}
			if i < len(ordered)-1 {
				log.Printf("transcription: falling back from %s to %s", p.Name(), ordered[i+1].Name())
			}
			continue
		}

		h.health.MarkHealthy(key)
		log.Printf("transcription: %s ok (%dms)", p.Name(), latency.Milliseconds())
		durationSeconds := audioinfo.EstimateDurationSeconds(req.Filename, req.File)
		billingModel := defaultTranscriptionModel(p.Name())
		cost, _ := h.billing.DeductAudio(ctx, userID, billingModel, durationSeconds, "")
		h.recorder.RecordSuccessWithApp(userID, apiKeyID, originalModel, p.Name(),
			durationSeconds, 0, int(latency.Milliseconds()), 0, cost, false, app.ID, app.URL, app.Title, app.Categories)
		writeJSON(ctx, 200, resp)
		return
	}

	writeError(ctx, 502, "provider_error", "all transcription providers failed")
}

func (h *TranscriptionHandler) buildProviderOrder(modelName string) []provider.TranscriptionProvider {
	if modelName == "" || modelName == "auto" {
		return h.providers
	}

	targetProvider, ok := modelToProvider[modelName]
	if !ok {
		return h.providers
	}

	preferred, exists := h.providerMap[targetProvider]
	if !exists {
		return h.providers
	}

	ordered := make([]provider.TranscriptionProvider, 0, len(h.providers))
	ordered = append(ordered, preferred)
	for _, p := range h.providers {
		if p.Name() != targetProvider {
			ordered = append(ordered, p)
		}
	}
	return ordered
}

func defaultTranscriptionModel(providerName string) string {
	switch providerName {
	case "groq":
		return "whisper-large-v3-turbo"
	case "openai":
		return "whisper-1"
	case "xai":
		return "xai-stt"
	case "fireworks":
		return "whisper-v3-large-turbo"
	default:
		return providerName
	}
}
