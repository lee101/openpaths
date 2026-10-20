package handler

import (
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

type EmbeddingHandler struct {
	router   *router.Router
	billing  *billing.Engine
	recorder *metrics.Recorder
	fallback []provider.EmbeddingProvider
}

func NewEmbeddingHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder, fallback []provider.EmbeddingProvider) *EmbeddingHandler {
	return &EmbeddingHandler{router: r, billing: b, recorder: rec, fallback: fallback}
}

func (h *EmbeddingHandler) HandleEmbedding(ctx *fasthttp.RequestCtx) {
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKey, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey)
	apiKeyID := ""
	if apiKey != nil {
		apiKeyID = apiKey.ID
	}

	var req model.EmbeddingRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "invalid_request", "Invalid JSON: "+err.Error())
		return
	}
	if req.Input == nil {
		writeError(ctx, 400, "invalid_request", "input is required")
		return
	}

	originalModel := req.Model
	app := requestAppAttribution(ctx)

	// Try router-resolved providers first (OpenRouter embedding models etc)
	if req.Model != "" {
		candidates, err := h.router.ResolveWithRetries(req.Model)
		if err == nil {
			for i, cand := range candidates {
				embProv, ok := cand.Provider.(provider.EmbeddingProvider)
				if !ok {
					continue
				}
				req.Model = cand.ModelCfg.ProviderModelID
				if prepaidGate(ctx, h.billing, cand.ModelCfg.ID, 0) {
					return
				}
				if !validEncodingFormat(req.EncodingFormat) {
					writeError(ctx, 400, "invalid_request", "encoding_format must be float, base64 or int8")
					return
				}
				start := time.Now()
				var resp *model.EmbeddingResponse
				var raw *model.RawEmbeddings
				var err error
				if re, ok := embProv.(rawEmbedder); ok {
					raw, err = re.EmbedRaw(ctx, &req)
				} else {
					resp, err = embProv.Embed(ctx, &req)
				}
				latency := time.Since(start)

				if err != nil {
					if pe, ok := err.(*provider.ProviderError); ok && !pe.Retryable {
						h.recorder.RecordErrorWithApp(userID, apiKeyID, originalModel, cand.Provider.Name(),
							int(latency.Milliseconds()), pe.StatusCode, pe.Message, false, app.ID, app.URL, app.Title, app.Categories)
						if pe.StatusCode == 429 || pe.StatusCode == 503 {
							ctx.Response.Header.Set("Retry-After", "1")
						}
						writeError(ctx, pe.StatusCode, "provider_error", pe.Message)
						return
					}
					h.router.MarkModelUnhealthy(cand.Provider.Name(), cand.ModelCfg.ID)
					log.Printf("embedding: %s failed (%dms): %v", cand.Provider.Name(), latency.Milliseconds(), err)
					if i < len(candidates)-1 {
						continue
					}
				} else {
					h.router.MarkModelHealthy(cand.Provider.Name(), cand.ModelCfg.ID)
					count, tokens := 0, 0
					if raw != nil {
						count, tokens = raw.N, raw.Tokens
					} else {
						resp.Model = originalModel
						count, tokens = len(resp.Data), resp.Usage.TotalTokens
					}
					cost, handled, _ := h.billing.DeductEmbeddings(ctx, userID, cand.ModelCfg.ID, count)
					if !handled {
						cost, _ = h.billing.Deduct(ctx, userID, cand.ModelCfg.ID, tokens, 0, "", "")
					}
					h.recorder.RecordSuccessCoalesced(userID, apiKeyID, originalModel, cand.Provider.Name(),
						tokens, int(latency.Milliseconds()), cost, app.ID, app.URL, app.Title, app.Categories)
					if raw != nil {
						writeRawEmbeddings(ctx, raw, originalModel, req.EncodingFormat)
						raw.Release()
					} else {
						writeEmbeddingJSON(ctx, resp)
					}
					return
				}
			}
		}
	}

	// Fall back to dedicated embedding providers (text-generator etc)
	for _, ep := range h.fallback {
		start := time.Now()
		resp, err := ep.Embed(ctx, &req)
		latency := time.Since(start)

		if err != nil {
			log.Printf("embedding: %s failed (%dms): %v", ep.Name(), latency.Milliseconds(), err)
			continue
		}

		resp.Model = originalModel
		var cost int64
		if originalModel != "" && h.billing != nil {
			var handled bool
			cost, handled, _ = h.billing.DeductEmbeddings(ctx, userID, originalModel, len(resp.Data))
			if !handled {
				cost, _ = h.billing.Deduct(ctx, userID, originalModel, resp.Usage.TotalTokens, 0, "", "")
			}
		}
		h.recorder.RecordSuccessCoalesced(userID, apiKeyID, originalModel, ep.Name(),
			resp.Usage.TotalTokens, int(latency.Milliseconds()), cost, app.ID, app.URL, app.Title, app.Categories)
		writeEmbeddingJSON(ctx, resp)
		return
	}

	writeError(ctx, 502, "provider_error", "all embedding providers failed")
}

func writeEmbeddingJSON(ctx *fasthttp.RequestCtx, resp *model.EmbeddingResponse) {
	ctx.SetStatusCode(200)
	ctx.SetContentType("application/json")
	ctx.Response.SetBodyRaw(resp.AppendJSON(make([]byte, 0, 256+len(resp.Data)*4096)))
}
