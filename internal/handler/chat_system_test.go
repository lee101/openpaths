package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/billing"
	"github.com/openpaths/openpaths/internal/metrics"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
)

type hiProvider struct {
	t *testing.T
}

func (p *hiProvider) Name() string { return "google" }

func (p *hiProvider) ChatCompletion(_ context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	p.t.Helper()
	if req.Model != "gemini-3.5-flash" {
		p.t.Fatalf("provider model = %q, want gemini-3.5-flash", req.Model)
	}
	if len(req.Messages) != 1 || req.Messages[0].Role != "user" || req.Messages[0].Content != "say hi nothing else" {
		p.t.Fatalf("messages = %#v, want single user prompt", req.Messages)
	}
	return &model.ChatCompletionResponse{
		ID:      "chatcmpl_test_hi",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []model.ChatChoice{{
			Index: 0,
			Message: &model.ChatMessage{
				Role:    "assistant",
				Content: "hi",
			},
			FinishReason: strPtr("stop"),
		}},
		Usage: &model.UsageInfo{},
	}, nil
}

func (p *hiProvider) ChatCompletionStream(_ context.Context, _ *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	ch := make(chan provider.StreamEvent)
	close(ch)
	return ch, nil
}

func (p *hiProvider) HealthCheck(context.Context) error { return nil }

func TestChatCompletionSystem_SayHiNothingElseReturnsHi(t *testing.T) {
	reg := provider.NewRegistry()
	reg.Register(&hiProvider{t: t})

	models := []model.ModelConfig{{
		ID:              "gemini-latest",
		Provider:        "google",
		ProviderModelID: "gemini-3.5-flash",
		Aliases:         []string{"auto-hard"},
	}}
	r := router.New(reg, models)
	pricing := billing.NewPricingTable(models)
	h := NewChatHandler(
		r,
		billing.NewEngine(pricing, nil),
		metrics.NewRecorder(metrics.NewCollector(nil, time.Hour)),
		nil,
		nil,
	)

	body, err := json.Marshal(map[string]any{
		"model": "gemini-latest",
		"messages": []map[string]string{
			{"role": "user", "content": "say hi nothing else"},
		},
		"max_tokens": 8,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	var req fasthttp.Request
	req.Header.SetMethod(http.MethodPost)
	req.SetRequestURI("/v1/chat/completions")
	req.Header.SetContentType("application/json")
	req.SetBody(body)

	var ctx fasthttp.RequestCtx
	ctx.Init(&req, nil, nil)
	ctx.SetUserValue(middleware.CtxKeyUserID, "ci-user")

	h.HandleChatCompletion(&ctx)

	if got := ctx.Response.StatusCode(); got != http.StatusOK {
		t.Fatalf("status = %d, body = %s", got, ctx.Response.Body())
	}

	var resp model.ChatCompletionResponse
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, ctx.Response.Body())
	}
	if resp.Model != "gemini-latest" {
		t.Fatalf("response model = %q, want gemini-latest", resp.Model)
	}
	if len(resp.Choices) != 1 || resp.Choices[0].Message == nil {
		t.Fatalf("choices = %#v, want one assistant message", resp.Choices)
	}
	if got := resp.Choices[0].Message.Content; got != "hi" {
		t.Fatalf("content = %q, want hi", got)
	}
}

func strPtr(s string) *string {
	return &s
}
