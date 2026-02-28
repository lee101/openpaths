package handler

import (
	"log"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpath/openpath/internal/metrics"
	"github.com/openpath/openpath/internal/middleware"
	"github.com/openpath/openpath/internal/model"
	"github.com/openpath/openpath/internal/provider"
	"github.com/openpath/openpath/internal/router"
)

type TranscriptionHandler struct {
	providers []provider.TranscriptionProvider
	health    *router.HealthTracker
	recorder  *metrics.Recorder
}

func NewTranscriptionHandler(providers []provider.TranscriptionProvider, health *router.HealthTracker, rec *metrics.Recorder) *TranscriptionHandler {
	return &TranscriptionHandler{providers: providers, health: health, recorder: rec}
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

	for i, p := range h.providers {
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
				h.recorder.RecordError(userID, apiKeyID, "whisper", p.Name(),
					int(latency.Milliseconds()), pe.StatusCode, pe.Message, false)
				writeError(ctx, pe.StatusCode, "provider_error", pe.Message)
				return
			}
			if i < len(h.providers)-1 {
				log.Printf("transcription: falling back from %s to %s", p.Name(), h.providers[i+1].Name())
			}
			continue
		}

		h.health.MarkHealthy(key)
		log.Printf("transcription: %s ok (%dms)", p.Name(), latency.Milliseconds())
		h.recorder.RecordSuccess(userID, apiKeyID, "whisper", p.Name(),
			0, 0, int(latency.Milliseconds()), 0, 0, false)
		writeJSON(ctx, 200, resp)
		return
	}

	writeError(ctx, 502, "provider_error", "all transcription providers failed")
}
