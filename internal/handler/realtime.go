package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	fws "github.com/fasthttp/websocket"
	gws "github.com/gorilla/websocket"
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/router"
)

type RealtimeHandler struct {
	router    *router.Router
	billing   *billing.Engine
	recorder  *metrics.Recorder
	providers []model.ProviderConfig
	upgrader  fws.FastHTTPUpgrader
	dialer    *gws.Dialer
}

func NewRealtimeHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder, providers []model.ProviderConfig) *RealtimeHandler {
	return &RealtimeHandler{
		router: r, billing: b, recorder: rec, providers: providers,
		upgrader: fws.FastHTTPUpgrader{
			ReadBufferSize: 8192, WriteBufferSize: 8192,
			CheckOrigin: func(*fasthttp.RequestCtx) bool { return true },
		},
		dialer: &gws.Dialer{HandshakeTimeout: 15 * time.Second},
	}
}

func (h *RealtimeHandler) HandleRealtime(ctx *fasthttp.RequestCtx) {
	requestedModel := strings.TrimSpace(string(ctx.QueryArgs().Peek("model")))
	cfg, ok := h.router.GetModelConfig(requestedModel)
	if !ok || cfg.Provider != "openai" || !strings.HasPrefix(cfg.ProviderModelID, "gpt-realtime-") {
		writeJSON(ctx, http.StatusBadRequest, realtimeError("invalid_model", "A supported OpenAI realtime model is required"))
		return
	}

	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKeyID := ""
	if key, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey); key != nil {
		apiKeyID = key.ID
	}
	providerCfg, found := h.openAIProvider()
	if !found {
		writeJSON(ctx, http.StatusServiceUnavailable, realtimeError("provider_unavailable", "OpenAI realtime is not configured"))
		return
	}

	upstreamKey := providerCfg.APIKey
	byok := false
	if key := getUserProviderKeys(ctx)["openai"]; key != nil && strings.TrimSpace(key.APIKey) != "" {
		upstreamKey = strings.TrimSpace(key.APIKey)
		byok = true
	}
	if upstreamKey == "" {
		writeJSON(ctx, http.StatusServiceUnavailable, realtimeError("provider_unavailable", "OpenAI realtime credentials are unavailable"))
		return
	}
	if !byok {
		estimate, err := h.billing.RealtimeCost(cfg.ID, billing.RealtimeUsage{AudioOutputTokens: cfg.MaxOutputTokens})
		if err != nil || h.billing.PreCheckFixed(ctx, userID, estimate) != nil {
			writeJSON(ctx, http.StatusPaymentRequired, realtimeError("insufficient_balance", "Insufficient balance for a realtime session"))
			return
		}
	}

	upstreamURL, err := makeRealtimeURL(providerCfg.BaseURL, cfg.ProviderModelID)
	if err != nil {
		writeJSON(ctx, http.StatusServiceUnavailable, realtimeError("provider_unavailable", "OpenAI realtime URL is invalid"))
		return
	}
	app := requestAppAttribution(ctx)
	started := time.Now()
	err = h.upgrader.Upgrade(ctx, func(client *fws.Conn) {
		h.relay(client, upstreamURL, upstreamKey, userID, apiKeyID, cfg.ID, byok, app, started)
	})
	if err != nil {
		log.Printf("realtime websocket upgrade failed: %v", err)
	}
}

func (h *RealtimeHandler) relay(client *fws.Conn, upstreamURL, upstreamKey, userID, apiKeyID, modelID string, byok bool, app requestApp, started time.Time) {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+upstreamKey)
	headers.Set("OpenAI-Safety-Identifier", safetyIdentifier(userID))
	upstream, resp, err := h.dialer.Dial(upstreamURL, headers)
	if err != nil {
		message := "Unable to connect to OpenAI realtime"
		if resp != nil {
			message = fmt.Sprintf("OpenAI realtime connection failed with status %d", resp.StatusCode)
		}
		_ = client.WriteJSON(realtimeError("upstream_connection_error", message))
		h.recordRealtimeError(userID, apiKeyID, modelID, app, started, http.StatusBadGateway, message)
		return
	}
	defer upstream.Close()
	defer client.Close()

	var clientWriteMu sync.Mutex
	writeClient := func(messageType int, payload []byte) error {
		clientWriteMu.Lock()
		defer clientWriteMu.Unlock()
		return client.WriteMessage(messageType, payload)
	}
	done := make(chan error, 2)
	go func() {
		for {
			messageType, payload, readErr := client.ReadMessage()
			if readErr != nil {
				done <- readErr
				return
			}
			if writeErr := upstream.WriteMessage(messageType, payload); writeErr != nil {
				done <- writeErr
				return
			}
		}
	}()

	var totalUsage billing.RealtimeUsage
	var totalCost int64
	seenResponses := make(map[string]bool)
	go func() {
		for {
			messageType, payload, readErr := upstream.ReadMessage()
			if readErr != nil {
				done <- readErr
				return
			}
			if messageType == gws.TextMessage {
				responseID, usage, completed, parseErr := parseRealtimeUsage(payload)
				if parseErr != nil {
					if !byok {
						_ = writeClient(fws.TextMessage, marshalRealtimeJSON(realtimeError("usage_unavailable", parseErr.Error())))
						done <- parseErr
						return
					}
					completed = false
				}
				if completed && (responseID == "" || !seenResponses[responseID]) {
					if responseID != "" {
						seenResponses[responseID] = true
					}
					cost := int64(0)
					if !byok {
						cost, err = h.billing.DeductRealtime(context.Background(), userID, modelID, usage, "")
						if err != nil {
							billingErr := errors.New("realtime billing failed")
							_ = writeClient(fws.TextMessage, marshalRealtimeJSON(realtimeError("billing_failed", "The realtime session ended because usage could not be billed")))
							done <- billingErr
							return
						}
					}
					addRealtimeUsage(&totalUsage, usage)
					totalCost += cost
				}
			}
			if writeErr := writeClient(messageType, payload); writeErr != nil {
				done <- writeErr
				return
			}
		}
	}()

	relayErr := <-done
	_ = client.Close()
	_ = upstream.Close()
	<-done
	latency := int(time.Since(started).Milliseconds())
	if totalRealtimeUsageTokens(totalUsage) > 0 && h.recorder != nil {
		h.recorder.RecordSuccessWithApp(userID, apiKeyID, modelID, "openai", realtimeInputTokens(totalUsage), realtimeOutputTokens(totalUsage), latency, 0, totalCost, true, app.ID, app.URL, app.Title, app.Categories)
	} else if !gws.IsCloseError(relayErr, gws.CloseNormalClosure, gws.CloseGoingAway) && !fws.IsCloseError(relayErr, fws.CloseNormalClosure, fws.CloseGoingAway) {
		h.recordRealtimeError(userID, apiKeyID, modelID, app, started, http.StatusBadGateway, relayErr.Error())
	}
}

func (h *RealtimeHandler) openAIProvider() (model.ProviderConfig, bool) {
	for _, cfg := range h.providers {
		if cfg.Name == "openai" && cfg.Enabled {
			return cfg, true
		}
	}
	return model.ProviderConfig{}, false
}

func (h *RealtimeHandler) recordRealtimeError(userID, apiKeyID, modelID string, app requestApp, started time.Time, status int, message string) {
	if h.recorder != nil {
		h.recorder.RecordErrorWithApp(userID, apiKeyID, modelID, "openai", int(time.Since(started).Milliseconds()), status, message, true, app.ID, app.URL, app.Title, app.Categories)
	}
}

type realtimeUsageEnvelope struct {
	Type     string `json:"type"`
	Response struct {
		ID    string `json:"id"`
		Usage *struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
			InputDetails struct {
				CachedTokens  int `json:"cached_tokens"`
				TextTokens    int `json:"text_tokens"`
				AudioTokens   int `json:"audio_tokens"`
				ImageTokens   int `json:"image_tokens"`
				CachedDetails struct {
					TextTokens  int `json:"text_tokens"`
					AudioTokens int `json:"audio_tokens"`
					ImageTokens int `json:"image_tokens"`
				} `json:"cached_tokens_details"`
			} `json:"input_token_details"`
			OutputDetails struct {
				TextTokens  int `json:"text_tokens"`
				AudioTokens int `json:"audio_tokens"`
			} `json:"output_token_details"`
		} `json:"usage"`
	} `json:"response"`
}

func parseRealtimeUsage(payload []byte) (string, billing.RealtimeUsage, bool, error) {
	var event realtimeUsageEnvelope
	if err := json.Unmarshal(payload, &event); err != nil || event.Type != "response.done" {
		return "", billing.RealtimeUsage{}, false, nil
	}
	if event.Response.Usage == nil {
		return event.Response.ID, billing.RealtimeUsage{}, false, errors.New("OpenAI response.done did not include usage")
	}
	u := event.Response.Usage
	textIn := max(u.InputDetails.TextTokens, u.InputTokens-u.InputDetails.AudioTokens-u.InputDetails.ImageTokens)
	if textIn < 0 {
		textIn = 0
	}
	textOut := max(u.OutputDetails.TextTokens, u.OutputTokens-u.OutputDetails.AudioTokens)
	if textOut < 0 {
		textOut = 0
	}
	usage := billing.RealtimeUsage{
		TextInputTokens: textIn, CachedTextInputTokens: u.InputDetails.CachedDetails.TextTokens,
		TextOutputTokens: textOut, AudioInputTokens: u.InputDetails.AudioTokens,
		CachedAudioInputTokens: u.InputDetails.CachedDetails.AudioTokens, AudioOutputTokens: u.OutputDetails.AudioTokens,
		ImageInputTokens: u.InputDetails.ImageTokens, CachedImageInputTokens: u.InputDetails.CachedDetails.ImageTokens,
	}
	if u.InputDetails.CachedTokens > 0 && usage.CachedTextInputTokens+usage.CachedAudioInputTokens+usage.CachedImageInputTokens == 0 {
		remaining := u.InputDetails.CachedTokens
		usage.CachedTextInputTokens = min(remaining, usage.TextInputTokens)
		remaining -= usage.CachedTextInputTokens
		usage.CachedAudioInputTokens = min(remaining, usage.AudioInputTokens)
		remaining -= usage.CachedAudioInputTokens
		usage.CachedImageInputTokens = min(remaining, usage.ImageInputTokens)
	}
	return event.Response.ID, usage, true, nil
}

func addRealtimeUsage(total *billing.RealtimeUsage, usage billing.RealtimeUsage) {
	total.TextInputTokens += usage.TextInputTokens
	total.CachedTextInputTokens += usage.CachedTextInputTokens
	total.TextOutputTokens += usage.TextOutputTokens
	total.AudioInputTokens += usage.AudioInputTokens
	total.CachedAudioInputTokens += usage.CachedAudioInputTokens
	total.AudioOutputTokens += usage.AudioOutputTokens
	total.ImageInputTokens += usage.ImageInputTokens
	total.CachedImageInputTokens += usage.CachedImageInputTokens
}

func realtimeInputTokens(usage billing.RealtimeUsage) int {
	return usage.TextInputTokens + usage.AudioInputTokens + usage.ImageInputTokens
}

func realtimeOutputTokens(usage billing.RealtimeUsage) int {
	return usage.TextOutputTokens + usage.AudioOutputTokens
}

func totalRealtimeUsageTokens(usage billing.RealtimeUsage) int {
	return realtimeInputTokens(usage) + realtimeOutputTokens(usage)
}

func makeRealtimeURL(baseURL, modelID string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return "", errors.New("invalid provider URL")
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	case "wss", "ws":
	default:
		return "", errors.New("unsupported provider URL scheme")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(u.Path, "/v1") {
		u.Path += "/v1"
	}
	u.Path += "/realtime"
	query := u.Query()
	query.Set("model", modelID)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func realtimeError(code, message string) map[string]any {
	return map[string]any{"type": "error", "error": map[string]any{"type": "invalid_request_error", "code": code, "message": message}}
}

func marshalRealtimeJSON(value any) []byte {
	payload, _ := json.Marshal(value)
	return payload
}
