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
	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/router"
	"github.com/valyala/fasthttp"
)

const realtimeMaxMessageBytes = 1 << 20
const realtimeSetupTimeout = 15 * time.Second
const realtimeIdleTimeout = 90 * time.Second
const realtimeWriteTimeout = 10 * time.Second

type RealtimeHandler struct {
	router    *router.Router
	billing   *billing.Engine
	recorder  *metrics.Recorder
	providers []model.ProviderConfig
	upgrader  fws.FastHTTPUpgrader
	dialer    *gws.Dialer
}

func NewRealtimeHandler(r *router.Router, b *billing.Engine, rec *metrics.Recorder, providers []model.ProviderConfig) *RealtimeHandler {
	return &RealtimeHandler{router: r, billing: b, recorder: rec, providers: providers,
		upgrader: fws.FastHTTPUpgrader{ReadBufferSize: 8192, WriteBufferSize: 8192, Subprotocols: middleware.RealtimeSubprotocols(), CheckOrigin: realtimeSameOrigin},
		dialer:   &gws.Dialer{HandshakeTimeout: 15 * time.Second},
	}
}

func realtimeSameOrigin(ctx *fasthttp.RequestCtx) bool {
	origin := string(ctx.Request.Header.Peek("Origin"))
	if origin == "" {
		return true
	} // Non-browser API clients authenticate by header.
	u, err := url.Parse(origin)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && strings.EqualFold(u.Host, string(ctx.Host()))
}

func (h *RealtimeHandler) providerByName(name string) (model.ProviderConfig, bool) {
	for _, cfg := range h.providers {
		if cfg.Name == name && cfg.Enabled {
			return cfg, true
		}
	}
	return model.ProviderConfig{}, false
}

func (h *RealtimeHandler) HandleRealtime(ctx *fasthttp.RequestCtx) {
	cfg, ok := h.router.GetModelConfig(strings.TrimSpace(string(ctx.QueryArgs().Peek("model"))))
	if !ok || !supportedRealtimeModel(cfg) {
		writeJSON(ctx, http.StatusBadRequest, realtimeError("invalid_model", "Choose a supported OpenAI or Gemini live model"))
		return
	}
	if !realtimeSameOrigin(ctx) {
		writeJSON(ctx, http.StatusForbidden, realtimeError("origin_rejected", "Cross-origin realtime connections are not allowed"))
		return
	}
	providerCfg, found := h.providerByName(cfg.Provider)
	if !found {
		writeJSON(ctx, http.StatusServiceUnavailable, realtimeError("provider_unavailable", "The voice provider is not configured"))
		return
	}
	userID, _ := ctx.UserValue(middleware.CtxKeyUserID).(string)
	apiKeyID := ""
	if key, _ := ctx.UserValue(middleware.CtxKeyAPIKey).(*model.APIKey); key != nil {
		apiKeyID = key.ID
	}
	upstreamKey := providerCfg.APIKey
	byok := false
	if key := getUserProviderKeys(ctx)[cfg.Provider]; key != nil && strings.TrimSpace(key.APIKey) != "" {
		upstreamKey = strings.TrimSpace(key.APIKey)
		byok = true
	}
	if upstreamKey == "" {
		writeJSON(ctx, http.StatusServiceUnavailable, realtimeError("provider_unavailable", "The voice provider credentials are unavailable"))
		return
	}
	if !byok {
		estimate, err := h.billing.RealtimeCost(cfg.ID, billing.RealtimeUsage{AudioOutputTokens: 1024})
		if err != nil || h.billing.PreCheckFixed(ctx, userID, estimate) != nil {
			writeJSON(ctx, http.StatusPaymentRequired, realtimeError("insufficient_balance", "Insufficient balance for a voice session"))
			return
		}
	}
	var upstreamURL string
	var err error
	if cfg.Provider == "google" {
		upstreamURL, err = makeGeminiRealtimeURL(providerCfg.BaseURL)
	} else {
		upstreamURL, err = makeRealtimeURL(providerCfg.BaseURL, cfg.ProviderModelID)
	}
	if err != nil {
		writeJSON(ctx, http.StatusServiceUnavailable, realtimeError("provider_unavailable", "The voice provider URL is invalid"))
		return
	}
	app := requestAppAttribution(ctx)
	started := time.Now()
	err = h.upgrader.Upgrade(ctx, func(client *fws.Conn) {
		h.relay(client, upstreamURL, upstreamKey, userID, apiKeyID, cfg, byok, app, started)
	})
	if err != nil {
		log.Printf("realtime websocket upgrade failed: %v", err)
	}
}

func supportedRealtimeModel(cfg *model.ModelConfig) bool {
	return (cfg.Provider == "openai" && strings.HasPrefix(cfg.ProviderModelID, "gpt-realtime-")) ||
		(cfg.Provider == "google" && cfg.ProviderModelID == "gemini-3.8-live-extended-thinking")
}

// validateRealtimeClientMessage runs on every frame so a second setup/update
// cannot silently change the provider model while retaining the original price.
func validateRealtimeClientMessage(payload []byte, cfg *model.ModelConfig, first bool) error {
	var event map[string]json.RawMessage
	if json.Unmarshal(payload, &event) != nil {
		return errors.New("Voice messages must be JSON objects")
	}
	if cfg.Provider == "google" {
		raw, setup := event["setup"]
		if first && !setup {
			return errors.New("The first Gemini message must contain setup")
		}
		if setup {
			if !first {
				return errors.New("A voice session can only be configured once")
			}
			var body struct {
				Model string                       `json:"model"`
				Tools []map[string]json.RawMessage `json:"tools"`
			}
			if json.Unmarshal(raw, &body) != nil || strings.TrimPrefix(body.Model, "models/") != cfg.ProviderModelID {
				return errors.New("The setup model must match the selected voice model")
			}
			// Built-in search has separate charges not covered by token billing.
			for _, tool := range body.Tools {
				for name := range tool {
					if name != "functionDeclarations" {
						return errors.New("Only client function declarations are supported in live sessions")
					}
				}
			}
		}
	} else {
		var body struct {
			Session struct {
				Model string `json:"model"`
			} `json:"session"`
		}
		if json.Unmarshal(payload, &body) != nil {
			return errors.New("Invalid voice message")
		}
		if body.Session.Model != "" && body.Session.Model != cfg.ProviderModelID {
			return errors.New("session.update cannot change the selected voice model")
		}
	}
	return nil
}

// Two independent pumps preserve low audio latency. The upstream pump bills
// only provider usage events; PCM frames don't wait on a database operation.
func (h *RealtimeHandler) relay(client *fws.Conn, upstreamURL, upstreamKey, userID, apiKeyID string, cfg *model.ModelConfig, byok bool, app requestApp, started time.Time) {
	defer client.Close()
	headers := http.Header{}
	if cfg.Provider == "google" {
		headers.Set("x-goog-api-key", upstreamKey)
	} else {
		headers.Set("Authorization", "Bearer "+upstreamKey)
		headers.Set("OpenAI-Safety-Identifier", safetyIdentifier(userID))
	}
	upstream, resp, err := h.dialer.Dial(upstreamURL, headers)
	if err != nil {
		message := "Unable to connect to the voice provider"
		if resp != nil {
			message = fmt.Sprintf("Voice provider connection failed with status %d", resp.StatusCode)
			if resp.Body != nil {
				resp.Body.Close()
			}
		}
		_ = client.SetWriteDeadline(time.Now().Add(realtimeWriteTimeout))
		_ = client.WriteJSON(realtimeError("upstream_connection_error", message))
		h.recordRealtimeError(userID, apiKeyID, cfg.ID, cfg.Provider, app, started, http.StatusBadGateway, message)
		return
	}
	defer upstream.Close()
	client.SetReadLimit(realtimeMaxMessageBytes)
	upstream.SetReadLimit(realtimeMaxMessageBytes)
	_ = client.SetReadDeadline(time.Now().Add(realtimeSetupTimeout))
	_ = upstream.SetReadDeadline(time.Now().Add(realtimeIdleTimeout))
	client.SetPongHandler(func(string) error { return client.SetReadDeadline(time.Now().Add(realtimeIdleTimeout)) })
	upstream.SetPongHandler(func(string) error { return upstream.SetReadDeadline(time.Now().Add(realtimeIdleTimeout)) })
	var clientWriteMu sync.Mutex
	writeClient := func(kind int, payload []byte) error {
		clientWriteMu.Lock()
		defer clientWriteMu.Unlock()
		_ = client.SetWriteDeadline(time.Now().Add(realtimeWriteTimeout))
		return client.WriteMessage(kind, payload)
	}
	sendError := func(code, message string) {
		_ = writeClient(fws.TextMessage, marshalRealtimeJSON(realtimeError(code, message)))
	}
	done := make(chan error, 2)
	go func() {
		first := true
		for {
			kind, payload, err := client.ReadMessage()
			if err != nil {
				done <- err
				return
			}
			if err = validateRealtimeClientMessage(payload, cfg, first); err != nil {
				sendError("invalid_session", err.Error())
				done <- err
				return
			}
			first = false
			_ = client.SetReadDeadline(time.Now().Add(realtimeIdleTimeout))
			_ = upstream.SetWriteDeadline(time.Now().Add(realtimeWriteTimeout))
			if err = upstream.WriteMessage(kind, payload); err != nil {
				done <- err
				return
			}
		}
	}()
	var totalUsage billing.RealtimeUsage
	var totalCost int64
	go func() {
		seenResponses := make(map[string]bool)
		for {
			kind, payload, err := upstream.ReadMessage()
			if err != nil {
				done <- err
				return
			}
			_ = upstream.SetReadDeadline(time.Now().Add(realtimeIdleTimeout))
			if kind == gws.TextMessage || kind == gws.BinaryMessage {
				var usage billing.RealtimeUsage
				var present bool
				var responseID string
				var parseErr error
				if cfg.Provider == "google" {
					usage, present = parseGeminiUsage(payload)
				} else {
					responseID, usage, present, parseErr = parseRealtimeUsage(payload)
				}
				if parseErr != nil && !byok {
					sendError("usage_unavailable", parseErr.Error())
					done <- parseErr
					return
				}
				if present && (responseID == "" || !seenResponses[responseID]) {
					if responseID != "" {
						seenResponses[responseID] = true
					}
					var cost int64
					if !byok {
						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						cost, err = h.billing.DeductRealtime(ctx, userID, cfg.ID, usage, "")
						cancel()
						if err != nil {
							sendError("billing_failed", "Voice session ended because usage could not be billed")
							done <- err
							return
						}
					}
					addRealtimeUsage(&totalUsage, usage)
					totalCost += cost
				}
			}
			if err = writeClient(kind, payload); err != nil {
				done <- err
				return
			}
		}
	}()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	lifetime := time.NewTimer(30 * time.Minute)
	defer lifetime.Stop()
	var relayErr error
	completedPumps := 0
loop:
	for {
		select {
		case relayErr = <-done:
			completedPumps++
			break loop
		case <-lifetime.C:
			sendError("session_expired", "Start a new call to continue after 30 minutes")
			break loop
		case <-ticker.C:
			if !byok {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := h.billing.PreCheckFixed(ctx, userID, 1)
				cancel()
				if err != nil {
					sendError("insufficient_balance", "Add credits to continue the voice session")
					relayErr = err
					break loop
				}
			}
			deadline := time.Now().Add(realtimeWriteTimeout)
			if err := client.WriteControl(fws.PingMessage, nil, deadline); err != nil {
				relayErr = err
				break loop
			}
			if err := upstream.WriteControl(gws.PingMessage, nil, deadline); err != nil {
				relayErr = err
				break loop
			}
		}
	}
	// Closing both sockets releases both pumps, even if the upstream failed while
	// the browser was silent. Wait for accounting before reading the totals.
	_ = client.WriteControl(fws.CloseMessage, fws.FormatCloseMessage(fws.CloseNormalClosure, "Voice session ended"), time.Now().Add(time.Second))
	_ = client.Close()
	_ = upstream.Close()
	for completedPumps < 2 {
		<-done
		completedPumps++
	}
	if totalRealtimeUsageTokens(totalUsage) > 0 && h.recorder != nil {
		h.recorder.RecordSuccessWithApp(userID, apiKeyID, cfg.ID, cfg.Provider, realtimeInputTokens(totalUsage), realtimeOutputTokens(totalUsage), int(time.Since(started).Milliseconds()), 0, totalCost, true, app.ID, app.URL, app.Title, app.Categories)
	} else if relayErr != nil && !gws.IsCloseError(relayErr, gws.CloseNormalClosure, gws.CloseGoingAway) && !fws.IsCloseError(relayErr, fws.CloseNormalClosure, fws.CloseGoingAway) {
		h.recordRealtimeError(userID, apiKeyID, cfg.ID, cfg.Provider, app, started, http.StatusBadGateway, "Voice connection ended before usage was reported")
	}
}

func (h *RealtimeHandler) recordRealtimeError(userID, apiKeyID, modelID, provider string, app requestApp, started time.Time, status int, message string) {
	if h.recorder != nil {
		h.recorder.RecordErrorWithApp(userID, apiKeyID, modelID, provider, int(time.Since(started).Milliseconds()), status, message, true, app.ID, app.URL, app.Title, app.Categories)
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
