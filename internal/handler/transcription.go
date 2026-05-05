package handler

import (
	"log"
	"strings"
	"time"

	"github.com/valyala/fasthttp"

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
	"whisper-1":              "openai",
	"gpt-4o-transcribe":      "openai",
	"gpt-4o-mini-transcribe": "openai",
	// Fireworks
	"whisper-v3-large":       "fireworks",
	"whisper-v3-large-turbo": "fireworks",
	// Fal
	"fal-ai/whisper": "fal",
}

type TranscriptionHandler struct {
	providers   []provider.TranscriptionProvider
	providerMap map[string]provider.TranscriptionProvider
	health      *router.HealthTracker
	recorder    *metrics.Recorder
}

func NewTranscriptionHandler(providers []provider.TranscriptionProvider, health *router.HealthTracker, rec *metrics.Recorder) *TranscriptionHandler {
	pm := make(map[string]provider.TranscriptionProvider, len(providers))
	for _, p := range providers {
		pm[p.Name()] = p
	}
	return &TranscriptionHandler{providers: providers, providerMap: pm, health: health, recorder: rec}
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

	modelName := strings.ToLower(reqModel)

	// Build ordered provider list: preferred provider first (if model-matched), then default chain
	ordered := h.buildProviderOrder(modelName)

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
				h.recorder.RecordError(userID, apiKeyID, reqModel, p.Name(),
					int(latency.Milliseconds()), pe.StatusCode, pe.Message, false)
				writeError(ctx, pe.StatusCode, "provider_error", pe.Message)
				return
			}
			if i < len(ordered)-1 {
				log.Printf("transcription: falling back from %s to %s", p.Name(), ordered[i+1].Name())
			}
			continue
		}

		h.health.MarkHealthy(key)
		log.Printf("transcription: %s model=%s ok (%dms)", p.Name(), reqModel, latency.Milliseconds())
		h.recorder.RecordSuccess(userID, apiKeyID, reqModel, p.Name(),
			0, 0, int(latency.Milliseconds()), 0, 0, false)
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
