// Package cloudflare serves Cloudflare Workers AI structured-inference models:
// Clairic (typed field extraction) and Clef / Clef-flash (typed decisions).
// They answer with typed, probability-scored fields rather than generated
// text, so they are exposed through StructuredProvider rather than chat.
//
//	POST {base_url}/@cf/cloudflare/<model>
//
// base_url is https://api.cloudflare.com/client/v4/accounts/<account>/ai/run.
// Auth is a Workers AI scoped bearer token, or the account email plus global
// API key when CLOUDFLARE_EMAIL is set.
package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

type Provider struct {
	apiKey  string
	baseURL string
	email   string
	client  *http.Client
}

func New(apiKey, baseURL string) *Provider {
	return &Provider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		email:   strings.TrimSpace(os.Getenv("CLOUDFLARE_EMAIL")),
		client:  &http.Client{Timeout: 2 * time.Minute},
	}
}

func (p *Provider) Name() string                          { return "cloudflare" }
func (p *Provider) HealthCheck(ctx context.Context) error { return nil }

func (p *Provider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	return nil, &provider.ProviderError{Provider: "cloudflare", StatusCode: 400, Message: "cloudflare structured models do not support chat; use /v1/extract or /v1/decide", Retryable: false}
}

func (p *Provider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	return nil, &provider.ProviderError{Provider: "cloudflare", StatusCode: 400, Message: "cloudflare structured models do not support chat; use /v1/extract or /v1/decide", Retryable: false}
}

// RunStructured posts body to the Workers AI model and returns the "result"
// object plus the input tokens Cloudflare billed.
func (p *Provider) RunStructured(ctx context.Context, providerModelID string, body map[string]any) (json.RawMessage, int, error) {
	if p.apiKey == "" || !strings.Contains(p.baseURL, "/accounts/") {
		return nil, 0, &provider.ProviderError{Provider: "cloudflare", StatusCode: 503, Message: "cloudflare workers ai is not configured", Retryable: false}
	}
	// The Workers AI "model" field is the short name, e.g. clef or clairic.
	short := providerModelID[strings.LastIndex(providerModelID, "/")+1:]
	payload := make(map[string]any, len(body)+1)
	for k, v := range body {
		payload[k] = v
	}
	if strings.Contains(providerModelID, "/clef") {
		payload["model"] = short
	} else {
		delete(payload, "model")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/"+providerModelID, bytes.NewReader(raw))
	if err != nil {
		return nil, 0, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.email != "" {
		httpReq.Header.Set("X-Auth-Email", p.email)
		httpReq.Header.Set("X-Auth-Key", p.apiKey)
	} else {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, 0, &provider.ProviderError{Provider: "cloudflare", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, 0, fmt.Errorf("read response: %w", err)
	}
	var env struct {
		Result  json.RawMessage `json:"result"`
		Success bool            `json:"success"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	_ = json.Unmarshal(respBody, &env)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || !env.Success || len(env.Result) == 0 {
		msg := fmt.Sprintf("workers ai returned HTTP %d", resp.StatusCode)
		if len(env.Errors) > 0 {
			msg = env.Errors[0].Message
		}
		status := resp.StatusCode
		if status < 400 {
			status = 502
		}
		// Request-shape problems are the caller's; credentials are ours.
		if status == 401 || status == 403 {
			status = 502
		}
		return nil, 0, &provider.ProviderError{Provider: "cloudflare", StatusCode: status, Message: msg, Retryable: status >= 500 || status == 429}
	}
	var usage struct {
		Usage struct {
			InputTokens int `json:"input_tokens"`
		} `json:"usage"`
	}
	_ = json.Unmarshal(env.Result, &usage)
	return env.Result, usage.Usage.InputTokens, nil
}
