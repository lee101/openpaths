// Package seedream talks to BytePlus ModelArk directly (SEEDREAM_API_KEY): the
// OpenAI-compatible chat API serving DeepSeek-V4.1-Flash, and Seedream image
// generation. ModelArk lives under /api/v3 instead of /v1, so requests from the
// shared OpenAI client are rewritten on the way out.
package seedream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	baseprovider "github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/provider/openai"
)

const defaultBaseURL = "https://ark.ap-southeast.bytepluses.com/api/v3"

type Provider struct {
	*openai.OpenAIProvider
	apiKey  string
	baseURL string
	client  *http.Client

	pollInterval time.Duration
}

func New(apiKey, baseURL string) *Provider {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	// The shared client appends /v1/...; give it the host and swap in /api/v3.
	chat := openai.NewCompatible("seedream", apiKey, baseURL, sanitize)
	chat.WrapTransport(func(rt http.RoundTripper) http.RoundTripper {
		return rewriteV1{base: rt, prefix: pathPrefix(baseURL)}
	})
	return &Provider{
		OpenAIProvider: chat,
		apiKey:         apiKey,
		baseURL:        baseURL,
		client:         &http.Client{Timeout: 3 * time.Minute, Transport: baseprovider.UserAgentTransport(nil)},
		pollInterval:   5 * time.Second,
	}
}

func (p *Provider) Name() string { return "seedream" }

func pathPrefix(baseURL string) string {
	if i := strings.Index(baseURL, "://"); i >= 0 {
		baseURL = baseURL[i+3:]
	}
	if i := strings.Index(baseURL, "/"); i >= 0 {
		return baseURL[i:]
	}
	return ""
}

// rewriteV1 maps <baseURL>/v1/x onto <baseURL>/x (baseURL already holds /api/v3).
type rewriteV1 struct {
	base   http.RoundTripper
	prefix string
}

func (r rewriteV1) RoundTrip(req *http.Request) (*http.Response, error) {
	if rest, ok := strings.CutPrefix(req.URL.Path, r.prefix+"/v1/"); ok {
		req = req.Clone(req.Context())
		req.URL.Path = r.prefix + "/" + rest
		req.URL.RawPath = ""
	}
	return r.base.RoundTrip(req)
}

// sanitize strips OpenPaths-only hints. ModelArk takes thinking.type and
// reasoning_effort (minimal|low|medium|high); "none" becomes thinking disabled.
func sanitize(req *model.ChatCompletionRequest) {
	req.Prefill = ""
	req.TaskTier = ""
	req.RoutingStrategy = ""
	if req.Thinking == nil {
		switch req.ReasoningEffort {
		case "none":
			req.Thinking = &model.ThinkingConfig{Type: "disabled"}
			req.ReasoningEffort = ""
		case "xhigh", "max":
			req.ReasoningEffort = "high"
		}
	} else {
		req.ReasoningEffort = ""
	}
}

func (p *Provider) GenerateImage(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error) {
	payload := map[string]any{
		"model":           req.Model,
		"prompt":          req.Prompt,
		"response_format": "b64_json",
		"watermark":       false,
	}
	if req.ResponseFormat == "url" {
		payload["response_format"] = "url"
	}
	if req.Size != "" {
		payload["size"] = req.Size
	}
	var inputs []string
	inputs = append(inputs, req.ImageURLs...)
	inputs = append(inputs, req.ReferenceImageURLs...)
	if req.ImageURL != "" {
		inputs = append([]string{req.ImageURL}, inputs...)
	}
	if len(inputs) == 1 {
		payload["image"] = inputs[0]
	} else if len(inputs) > 1 {
		payload["image"] = inputs
	}
	if req.N > 1 {
		payload["sequential_image_generation"] = "auto"
		payload["sequential_image_generation_options"] = map[string]any{"max_images": req.N}
	}
	if req.Seed != nil {
		payload["seed"] = *req.Seed
	}
	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/images/generations", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &baseprovider.ProviderError{Provider: "seedream", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		msg := string(respBody)
		if len(msg) > 500 {
			msg = msg[:500]
		}
		return nil, &baseprovider.ProviderError{Provider: "seedream", StatusCode: resp.StatusCode, Message: msg,
			Retryable: resp.StatusCode >= 500 || resp.StatusCode == 429}
	}
	var out model.ImageGenerationResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &out, nil
}
