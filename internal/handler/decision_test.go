package handler

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/router"
)

type fakeLocalDecider struct{ got string }

func (f *fakeLocalDecider) Name() string { return "gobed" }
func (f *fakeLocalDecider) Embed(context.Context, *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	return nil, nil
}
func (f *fakeLocalDecider) Decide(_ context.Context, req *model.DecisionRequest) (*model.DecisionResponse, error) {
	f.got = req.Model
	return &model.DecisionResponse{Model: req.Model, Answers: json.RawMessage(`{"q":{"type":"noul","noul":0.9}}`), Usage: model.DecisionUsage{InputTokens: 7}}, nil
}

func TestDecisionRoutesToLocalEmbedder(t *testing.T) {
	r := router.New(provider.NewRegistry(), []model.ModelConfig{{ID: "openpaths-decision", Provider: "gobed", ProviderModelID: "openpaths-decision", Aliases: []string{"gobed-decision"}}})
	d := &fakeLocalDecider{}
	h := NewDecisionHandler(r, nil, nil, []provider.EmbeddingProvider{d})
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetBody([]byte(`{"model":"gobed-decision","state":"hi","questions":{"q":{"type":"noul","instructions":"ok?"}}}`))
	h.HandleDecision(ctx)
	body := string(ctx.Response.Body())
	if ctx.Response.StatusCode() != 200 || d.got != "openpaths-decision" || !strings.Contains(body, `"model":"openpaths-decision"`) || !strings.Contains(body, `"input_tokens":7`) {
		t.Fatalf("%d %s got=%s", ctx.Response.StatusCode(), body, d.got)
	}
}

func TestDecisionUnknownModel404(t *testing.T) {
	h := NewDecisionHandler(router.New(provider.NewRegistry(), nil), nil, nil, nil)
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetBody([]byte(`{"model":"nope","state":"hi","questions":{"q":{"type":"noul","instructions":"ok?"}}}`))
	h.HandleDecision(ctx)
	if ctx.Response.StatusCode() != 404 {
		t.Fatal(ctx.Response.StatusCode())
	}
}
