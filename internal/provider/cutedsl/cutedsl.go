// Package cutedsl integrates CuteDSL (https://cutedsl.cc), a Triton-accelerated
// model API that exposes image generation, time-series forecasting (chronos2),
// TTS, STT, chat, and image captioning behind a single unified endpoint:
//
//	POST https://cutedsl.cc/api/service
//	Authorization: Bearer <api_key>
//	{"service": "<name>", ...service-specific params}
//
// Every response shares an envelope:
//
//	{"result": {...}, "credits_used": N, "credits_remain": N, "usd_equivalent": F}
//
// The per-service result field names are not documented, so image/forecast
// parsing is deliberately defensive (see imageURLFromResult / forecastFromResult).
package cutedsl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

const defaultBaseURL = "https://cutedsl.cc"

type CuteDSLProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func New(apiKey, baseURL string) *CuteDSLProvider {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &CuteDSLProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

func (p *CuteDSLProvider) Name() string { return "cutedsl" }

func (p *CuteDSLProvider) HealthCheck(ctx context.Context) error { return nil }

// ChatCompletion / ChatCompletionStream — CuteDSL has a Gemma4 chat service, but
// it is not OpenAI-chat-shaped, so we don't expose it via the chat router yet.
func (p *CuteDSLProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	return nil, &provider.ProviderError{Provider: "cutedsl", StatusCode: 400, Message: "cutedsl does not support chat", Retryable: false}
}

func (p *CuteDSLProvider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	return nil, &provider.ProviderError{Provider: "cutedsl", StatusCode: 400, Message: "cutedsl does not support chat", Retryable: false}
}

// serviceEnvelope is the shared response wrapper for every CuteDSL service.
type serviceEnvelope struct {
	Result        json.RawMessage `json:"result"`
	CreditsUsed   float64         `json:"credits_used"`
	CreditsRemain float64         `json:"credits_remain"`
	USDEquivalent float64         `json:"usd_equivalent"`
	Error         string          `json:"error"`
}

// callService posts a service request to the unified endpoint and returns the
// decoded envelope. params is merged with {"service": service}; a "service" key
// inside params is ignored in favour of the explicit argument.
func (p *CuteDSLProvider) callService(ctx context.Context, service string, params map[string]any) (*serviceEnvelope, error) {
	payload := map[string]any{"service": service}
	for k, v := range params {
		if k == "service" {
			continue
		}
		payload[k] = v
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	endpoint := p.baseURL + "/api/service"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "cutedsl", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &provider.ProviderError{
			Provider:   "cutedsl",
			StatusCode: resp.StatusCode,
			Message:    cutedslErrorMessage(respBody, resp.StatusCode),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var env serviceEnvelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}
	if env.Error != "" {
		return nil, &provider.ProviderError{Provider: "cutedsl", StatusCode: 502, Message: env.Error, Retryable: false}
	}
	return &env, nil
}

func cutedslErrorMessage(body []byte, status int) string {
	var e struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &e) == nil {
		if e.Error != "" {
			return e.Error
		}
		if e.Message != "" {
			return e.Message
		}
	}
	if len(body) > 0 {
		return string(body)
	}
	return "cutedsl returned status " + strconv.Itoa(status)
}

// GenerateImage drives the z-image-turbo / flux image services. req.Model holds
// the CuteDSL service name (set from the model config's provider_model_id).
func (p *CuteDSLProvider) GenerateImage(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	service := req.Model
	if service == "" {
		service = "z-image-turbo"
	}

	params := map[string]any{"prompt": req.Prompt}
	if w, h, ok := parseSize(req.Size); ok {
		params["width"] = w
		params["height"] = h
	}
	if req.NumInferenceSteps > 0 {
		params["steps"] = req.NumInferenceSteps
	}
	if req.Seed != nil {
		params["seed"] = *req.Seed
	}

	env, err := p.callService(ctx, service, params)
	if err != nil {
		return nil, err
	}

	images := imagesFromResult(env.Result)
	if len(images) == 0 {
		return nil, &provider.ProviderError{Provider: "cutedsl", StatusCode: 502, Message: "no image in response: " + string(env.Result), Retryable: false}
	}
	return &model.ImageGenerationResponse{Created: time.Now().Unix(), Data: images}, nil
}

// GenerateForecast drives the chronos2 time-series service.
func (p *CuteDSLProvider) GenerateForecast(ctx context.Context, req *model.ForecastingRequest) (*model.ForecastingResponse, error) {
	service := req.Model
	if service == "" {
		service = "chronos2"
	}
	history := req.History()
	if len(history) == 0 {
		return nil, &provider.ProviderError{Provider: "cutedsl", StatusCode: 400, Message: "context (historical series) is required", Retryable: false}
	}

	params := map[string]any{
		"context":           history,
		"prediction_length": req.Steps(),
	}
	if len(req.Quantiles) > 0 {
		params["quantiles"] = req.Quantiles
	}
	if req.Freq != "" {
		params["freq"] = req.Freq
	}

	env, err := p.callService(ctx, service, params)
	if err != nil {
		return nil, err
	}

	forecast, quantiles := forecastFromResult(env.Result)
	if len(forecast) == 0 {
		return nil, &provider.ProviderError{Provider: "cutedsl", StatusCode: 502, Message: "no forecast in response: " + string(env.Result), Retryable: false}
	}
	return &model.ForecastingResponse{
		Created:   time.Now().Unix(),
		Model:     service,
		Forecast:  forecast,
		Quantiles: quantiles,
	}, nil
}

// imagesFromResult extracts image outputs from the (undocumented) result object.
// It handles a bare URL string, a single image_url/url/image field, an images
// array, and inline base64 / data-URI payloads.
func imagesFromResult(raw json.RawMessage) []model.ImageData {
	if len(raw) == 0 {
		return nil
	}

	// result may be a bare URL string.
	var asString string
	if json.Unmarshal(raw, &asString) == nil && asString != "" {
		if img, ok := imageDataFromValue(asString); ok {
			return []model.ImageData{img}
		}
	}

	var obj map[string]any
	if json.Unmarshal(raw, &obj) != nil {
		return nil
	}

	var out []model.ImageData
	appendFromAny := func(v any) {
		if img, ok := imageDataFromValue(v); ok {
			out = append(out, img)
		}
	}

	for _, key := range []string{"image_url", "url", "image", "output", "b64_json", "image_base64"} {
		if v, ok := obj[key]; ok {
			appendFromAny(v)
		}
	}
	for _, key := range []string{"images", "outputs", "data"} {
		if items, ok := obj[key].([]any); ok {
			for _, item := range items {
				switch t := item.(type) {
				case string:
					appendFromAny(t)
				case map[string]any:
					for _, k := range []string{"url", "image_url", "image", "b64_json"} {
						if v, ok := t[k]; ok {
							appendFromAny(v)
							break
						}
					}
				}
			}
		}
	}
	return out
}

func imageDataFromValue(v any) (model.ImageData, bool) {
	s, ok := v.(string)
	if !ok || s == "" {
		return model.ImageData{}, false
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return model.ImageData{URL: s}, true
	}
	if strings.HasPrefix(s, "data:image/") {
		if idx := strings.Index(s, ";base64,"); idx != -1 {
			return model.ImageData{B64JSON: s[idx+len(";base64,"):]}, true
		}
		return model.ImageData{}, false
	}
	// Heuristic: a long token with no spaces is raw base64.
	if len(s) > 100 && !strings.ContainsAny(s, " \n\t") {
		return model.ImageData{B64JSON: s}, true
	}
	return model.ImageData{}, false
}

// forecastFromResult extracts the point forecast and optional quantiles from the
// (undocumented) result object.
func forecastFromResult(raw json.RawMessage) ([]float64, map[string][]float64) {
	if len(raw) == 0 {
		return nil, nil
	}

	// result may be a bare array of numbers.
	var asArray []float64
	if json.Unmarshal(raw, &asArray) == nil && len(asArray) > 0 {
		return asArray, nil
	}

	var obj map[string]json.RawMessage
	if json.Unmarshal(raw, &obj) != nil {
		return nil, nil
	}

	var forecast []float64
	for _, key := range []string{"forecast", "mean", "predictions", "prediction", "values", "result", "median"} {
		if rawVal, ok := obj[key]; ok {
			if vals, ok := floatSlice(rawVal); ok && len(vals) > 0 {
				forecast = vals
				break
			}
		}
	}

	var quantiles map[string][]float64
	if rawQ, ok := obj["quantiles"]; ok {
		var qobj map[string]json.RawMessage
		if json.Unmarshal(rawQ, &qobj) == nil {
			for k, rv := range qobj {
				if vals, ok := floatSlice(rv); ok {
					if quantiles == nil {
						quantiles = make(map[string][]float64)
					}
					quantiles[k] = vals
				}
			}
		}
	}
	return forecast, quantiles
}

// floatSlice decodes a JSON value into []float64, tolerating both a flat array
// and a single-row nested array ([[...]]).
func floatSlice(raw json.RawMessage) ([]float64, bool) {
	var flat []float64
	if json.Unmarshal(raw, &flat) == nil && len(flat) > 0 {
		return flat, true
	}
	var nested [][]float64
	if json.Unmarshal(raw, &nested) == nil && len(nested) > 0 {
		return nested[0], true
	}
	return nil, false
}

func parseSize(size string) (int, int, bool) {
	if size == "" {
		return 0, 0, false
	}
	parts := strings.SplitN(strings.ToLower(size), "x", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	w, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || w <= 0 || h <= 0 {
		return 0, 0, false
	}
	return w, h, true
}
