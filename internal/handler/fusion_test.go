package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
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

// fusionProvider serves the panel models and the judge model from a single
// provider, recording which provider-model IDs were called and capturing the
// judge's incoming prompt so tests can assert the panel answers were fused in.
type fusionProvider struct {
	mu          sync.Mutex
	calls       []string
	judgePrompt string
	failModel   string // provider-model ID that should return an error
}

func (p *fusionProvider) Name() string { return "test" }

func (p *fusionProvider) ChatCompletion(_ context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	p.mu.Lock()
	p.calls = append(p.calls, req.Model)
	p.mu.Unlock()

	if p.failModel != "" && req.Model == p.failModel {
		return nil, &provider.ProviderError{Provider: "test", StatusCode: 500, Message: "boom", Retryable: false}
	}

	var content string
	switch req.Model {
	case "opus":
		content = "OPUS_ANSWER"
	case "gpt":
		content = "GPT_ANSWER"
	case "gemini": // judge
		p.mu.Lock()
		p.judgePrompt = fusionMessageContentText(req.Messages[len(req.Messages)-1].Content)
		p.mu.Unlock()
		content = "FUSED_RESULT"
	default:
		content = "OTHER"
	}

	return &model.ChatCompletionResponse{
		ID:      "chatcmpl_" + req.Model,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []model.ChatChoice{{
			Index:        0,
			Message:      &model.ChatMessage{Role: "assistant", Content: content},
			FinishReason: strPtr("stop"),
		}},
		Usage: &model.UsageInfo{PromptTokens: 5, CompletionTokens: 5},
	}, nil
}

func (p *fusionProvider) ChatCompletionStream(_ context.Context, _ *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	ch := make(chan provider.StreamEvent)
	close(ch)
	return ch, nil
}

func (p *fusionProvider) HealthCheck(context.Context) error { return nil }

type openRouterFusionProvider struct {
	mu          sync.Mutex
	calledModel string
	tools       []model.Tool
}

func (p *openRouterFusionProvider) Name() string { return "openrouter" }

func (p *openRouterFusionProvider) ChatCompletion(_ context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	p.mu.Lock()
	p.calledModel = req.Model
	p.tools = append([]model.Tool(nil), req.Tools...)
	p.mu.Unlock()
	return &model.ChatCompletionResponse{
		ID:      "chatcmpl_or",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []model.ChatChoice{{
			Index:        0,
			Message:      &model.ChatMessage{Role: "assistant", Content: "OPENROUTER_FUSED"},
			FinishReason: strPtr("stop"),
		}},
		Usage: &model.UsageInfo{PromptTokens: 7, CompletionTokens: 3},
	}, nil
}

func (p *openRouterFusionProvider) ChatCompletionStream(_ context.Context, _ *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	ch := make(chan provider.StreamEvent)
	close(ch)
	return ch, nil
}

func (p *openRouterFusionProvider) HealthCheck(context.Context) error { return nil }

func fusionModels() []model.ModelConfig {
	return []model.ModelConfig{
		{ID: "claude-opus-latest", Provider: "test", ProviderModelID: "opus"},
		{ID: "gpt-5.5", Provider: "test", ProviderModelID: "gpt"},
		{ID: "gemini-latest", Provider: "test", ProviderModelID: "gemini"},
	}
}

func newFusionHandler(prov provider.Provider) *ChatHandler {
	reg := provider.NewRegistry()
	reg.Register(prov)
	models := fusionModels()
	r := router.New(reg, models)
	pricing := billing.NewPricingTable(models)
	return NewChatHandler(
		r,
		billing.NewEngine(pricing, nil),
		metrics.NewRecorder(metrics.NewCollector(nil, time.Hour)),
		nil,
		nil,
	)
}

func postFusion(t *testing.T, h *ChatHandler, body map[string]any) *fasthttp.RequestCtx {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var req fasthttp.Request
	req.Header.SetMethod(http.MethodPost)
	req.SetRequestURI("/v1/chat/completions")
	req.Header.SetContentType("application/json")
	req.SetBody(raw)

	ctx := &fasthttp.RequestCtx{}
	ctx.Init(&req, nil, nil)
	ctx.SetUserValue(middleware.CtxKeyUserID, "ci-user")
	h.HandleChatCompletion(ctx)
	return ctx
}

// TestFusion_OpenPaths_HappyPath runs a two-model panel through a judge and
// asserts the judge received both panel answers and the final response is
// reported under the originally requested model id.
func TestFusion_OpenPaths_HappyPath(t *testing.T) {
	prov := &fusionProvider{}
	h := newFusionHandler(prov)

	ctx := postFusion(t, h, map[string]any{
		"model":    "gemini-latest",
		"messages": []map[string]string{{"role": "user", "content": "carbon tax pros and cons?"}},
		"fusion": map[string]any{
			"type":            "openpaths",
			"analysis_models": []string{"claude-opus-latest", "gpt-5.5"},
			"model":           "gemini-latest",
			"prompt":          "MERGE_THE_PANEL",
		},
	})

	if got := ctx.Response.StatusCode(); got != http.StatusOK {
		t.Fatalf("status = %d, body = %s", got, ctx.Response.Body())
	}

	var resp model.ChatCompletionResponse
	if err := json.Unmarshal(ctx.Response.Body(), &resp); err != nil {
		t.Fatalf("decode: %v; body=%s", err, ctx.Response.Body())
	}
	if resp.Model != "gemini-latest" {
		t.Errorf("response model = %q, want gemini-latest", resp.Model)
	}
	if len(resp.Choices) != 1 || resp.Choices[0].Message == nil || resp.Choices[0].Message.Content != "FUSED_RESULT" {
		t.Fatalf("choices = %#v, want single FUSED_RESULT message", resp.Choices)
	}

	// Panel + judge all called.
	prov.mu.Lock()
	calls := append([]string(nil), prov.calls...)
	judgePrompt := prov.judgePrompt
	prov.mu.Unlock()

	wantCalled := map[string]bool{"opus": false, "gpt": false, "gemini": false}
	for _, c := range calls {
		if _, ok := wantCalled[c]; ok {
			wantCalled[c] = true
		}
	}
	for m, called := range wantCalled {
		if !called {
			t.Errorf("expected provider model %q to be called; calls=%v", m, calls)
		}
	}

	// Judge prompt must contain the fusion instruction and both panel answers.
	for _, needle := range []string{"MERGE_THE_PANEL", "OPUS_ANSWER", "GPT_ANSWER", "claude-opus-latest", "gpt-5.5"} {
		if !strings.Contains(judgePrompt, needle) {
			t.Errorf("judge prompt missing %q\nprompt=%s", needle, judgePrompt)
		}
	}
}

// TestFusion_DefaultsJudgeToRequestModel verifies that omitting fusion.model
// falls back to the top-level request model as the judge.
func TestFusion_DefaultsJudgeToRequestModel(t *testing.T) {
	prov := &fusionProvider{}
	h := newFusionHandler(prov)

	ctx := postFusion(t, h, map[string]any{
		"model":    "gemini-latest",
		"messages": []map[string]string{{"role": "user", "content": "q"}},
		"fusion": map[string]any{
			"analysis_models": []string{"claude-opus-latest", "gpt-5.5"},
		},
	})
	if got := ctx.Response.StatusCode(); got != http.StatusOK {
		t.Fatalf("status = %d, body = %s", got, ctx.Response.Body())
	}
	prov.mu.Lock()
	defer prov.mu.Unlock()
	called := false
	for _, c := range prov.calls {
		if c == "gemini" {
			called = true
		}
	}
	if !called {
		t.Errorf("judge (gemini) not called; calls=%v", prov.calls)
	}
}

// TestFusion_PartialPanelFailureStillFuses ensures the judge still runs when
// some — but not all — panel legs fail.
func TestFusion_PartialPanelFailureStillFuses(t *testing.T) {
	prov := &fusionProvider{failModel: "gpt"}
	h := newFusionHandler(prov)

	ctx := postFusion(t, h, map[string]any{
		"model":    "gemini-latest",
		"messages": []map[string]string{{"role": "user", "content": "q"}},
		"fusion": map[string]any{
			"analysis_models": []string{"claude-opus-latest", "gpt-5.5"},
			"model":           "gemini-latest",
		},
	})
	if got := ctx.Response.StatusCode(); got != http.StatusOK {
		t.Fatalf("status = %d, body = %s", got, ctx.Response.Body())
	}
	prov.mu.Lock()
	judgePrompt := prov.judgePrompt
	prov.mu.Unlock()
	if !strings.Contains(judgePrompt, "OPUS_ANSWER") {
		t.Errorf("judge prompt should include the surviving panel answer; got %s", judgePrompt)
	}
	if !strings.Contains(judgePrompt, "ERROR") {
		t.Errorf("judge prompt should note the failed leg; got %s", judgePrompt)
	}
}

// TestFusion_AllPanelFail returns 502 when every panel leg fails.
func TestFusion_AllPanelFail(t *testing.T) {
	prov := &fusionProvider{}
	// Fail every panel model: route both opus and gpt to errors by failing on a
	// shared sentinel is not possible with one field, so register a provider
	// that fails everything except the judge.
	prov.failModel = "*"
	h := newFusionHandler(&allFailExceptJudge{fusionProvider: prov})

	ctx := postFusion(t, h, map[string]any{
		"model":    "gemini-latest",
		"messages": []map[string]string{{"role": "user", "content": "q"}},
		"fusion": map[string]any{
			"analysis_models": []string{"claude-opus-latest", "gpt-5.5"},
			"model":           "gemini-latest",
		},
	})
	if got := ctx.Response.StatusCode(); got != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body = %s", got, ctx.Response.Body())
	}
}

type allFailExceptJudge struct{ *fusionProvider }

func (p *allFailExceptJudge) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	if req.Model != "gemini" {
		return nil, &provider.ProviderError{Provider: "test", StatusCode: 500, Message: "boom", Retryable: false}
	}
	return p.fusionProvider.ChatCompletion(ctx, req)
}

// TestFusion_StreamRejected confirms streaming fusion is rejected with 400.
func TestFusion_StreamRejected(t *testing.T) {
	prov := &fusionProvider{}
	h := newFusionHandler(prov)
	ctx := postFusion(t, h, map[string]any{
		"model":    "gemini-latest",
		"stream":   true,
		"messages": []map[string]string{{"role": "user", "content": "q"}},
		"fusion": map[string]any{
			"analysis_models": []string{"claude-opus-latest"},
		},
	})
	if got := ctx.Response.StatusCode(); got != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", got, ctx.Response.Body())
	}
}

// TestFusion_InvalidType rejects an unknown fusion.type.
func TestFusion_InvalidType(t *testing.T) {
	prov := &fusionProvider{}
	h := newFusionHandler(prov)
	ctx := postFusion(t, h, map[string]any{
		"model":    "gemini-latest",
		"messages": []map[string]string{{"role": "user", "content": "q"}},
		"fusion": map[string]any{
			"type":            "bogus",
			"analysis_models": []string{"claude-opus-latest"},
		},
	})
	if got := ctx.Response.StatusCode(); got != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", got, ctx.Response.Body())
	}
}

// TestFusion_OpenRouterMissingProvider returns 502 when the openrouter provider
// is not configured (the registry only has the test provider).
func TestFusion_OpenRouterMissingProvider(t *testing.T) {
	prov := &fusionProvider{}
	h := newFusionHandler(prov)
	ctx := postFusion(t, h, map[string]any{
		"model":    "gemini-latest",
		"messages": []map[string]string{{"role": "user", "content": "q"}},
		"fusion": map[string]any{
			"type":            "openrouter",
			"analysis_models": []string{"x-ai/grok-4"},
		},
	})
	if got := ctx.Response.StatusCode(); got != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502; body = %s", got, ctx.Response.Body())
	}
}

func TestFusion_OpenRouterUsesConfiguredModelForBillingPath(t *testing.T) {
	prov := &openRouterFusionProvider{}
	reg := provider.NewRegistry()
	reg.Register(prov)
	models := []model.ModelConfig{
		{ID: "gemini-latest", Provider: "openrouter", ProviderModelID: "google/gemini-pro"},
		{ID: "or/test-fusion", Provider: "openrouter", ProviderModelID: "openai/test-fusion"},
	}
	r := router.New(reg, models)
	h := NewChatHandler(
		r,
		billing.NewEngine(billing.NewPricingTable(models), nil),
		metrics.NewRecorder(metrics.NewCollector(nil, time.Hour)),
		nil,
		nil,
	)

	ctx := postFusion(t, h, map[string]any{
		"model":    "gemini-latest",
		"messages": []map[string]string{{"role": "user", "content": "q"}},
		"fusion": map[string]any{
			"type":            "openrouter",
			"analysis_models": []string{"anthropic/claude-sonnet-4"},
			"model":           "openai/test-fusion",
		},
	})
	if got := ctx.Response.StatusCode(); got != http.StatusOK {
		t.Fatalf("status = %d, body = %s", got, ctx.Response.Body())
	}
	if got := h.openRouterFusionBillingModel("openai/test-fusion", "gemini-latest"); got != "or/test-fusion" {
		t.Fatalf("billing model = %q, want or/test-fusion", got)
	}
	prov.mu.Lock()
	calledModel := prov.calledModel
	tools := append([]model.Tool(nil), prov.tools...)
	prov.mu.Unlock()
	if calledModel != "openai/test-fusion" {
		t.Fatalf("provider model = %q, want raw OpenRouter model", calledModel)
	}
	if len(tools) != 1 || tools[0].Type != "openrouter:fusion" {
		t.Fatalf("tools = %#v, want OpenRouter fusion tool", tools)
	}
}

// TestFusion_MaxPanelModelsCaps trims the panel to MaxPanelModels.
func TestFusion_MaxPanelModelsCaps(t *testing.T) {
	prov := &fusionProvider{}
	h := newFusionHandler(prov)
	ctx := postFusion(t, h, map[string]any{
		"model":    "gemini-latest",
		"messages": []map[string]string{{"role": "user", "content": "q"}},
		"fusion": map[string]any{
			"analysis_models":  []string{"claude-opus-latest", "gpt-5.5"},
			"max_panel_models": 1,
			"model":            "gemini-latest",
		},
	})
	if got := ctx.Response.StatusCode(); got != http.StatusOK {
		t.Fatalf("status = %d, body = %s", got, ctx.Response.Body())
	}
	prov.mu.Lock()
	defer prov.mu.Unlock()
	gptCalled := false
	for _, c := range prov.calls {
		if c == "gpt" {
			gptCalled = true
		}
	}
	if gptCalled {
		t.Errorf("max_panel_models=1 should have dropped gpt; calls=%v", prov.calls)
	}
}

// TestFusion_DuplicatePanelModels verifies the same model can be listed multiple
// times as separate panel variants: every leg runs and the judge sees distinct,
// numbered entries so it can tell the variants apart.
func TestFusion_DuplicatePanelModels(t *testing.T) {
	prov := &fusionProvider{}
	h := newFusionHandler(prov)

	ctx := postFusion(t, h, map[string]any{
		"model":    "gemini-latest",
		"messages": []map[string]string{{"role": "user", "content": "q"}},
		"fusion": map[string]any{
			"type":            "openpaths",
			"analysis_models": []string{"claude-opus-latest", "claude-opus-latest", "claude-opus-latest"},
			"model":           "gemini-latest",
			"prompt":          "MERGE",
		},
	})
	if got := ctx.Response.StatusCode(); got != http.StatusOK {
		t.Fatalf("status = %d, body = %s", got, ctx.Response.Body())
	}

	prov.mu.Lock()
	calls := append([]string(nil), prov.calls...)
	judgePrompt := prov.judgePrompt
	prov.mu.Unlock()

	opusCalls := 0
	for _, c := range calls {
		if c == "opus" {
			opusCalls++
		}
	}
	if opusCalls != 3 {
		t.Errorf("expected 3 opus panel legs, got %d; calls=%v", opusCalls, calls)
	}

	for _, needle := range []string{"(variant 1)", "(variant 2)", "(variant 3)"} {
		if !strings.Contains(judgePrompt, needle) {
			t.Errorf("judge prompt missing %q; prompt=%s", needle, judgePrompt)
		}
	}
}
