package middleware

import (
	"encoding/json"
	"testing"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/model"
)

func TestAuthorizeKey(t *testing.T) {
	key := &model.APIKey{Scopes: []string{model.ScopeImages}}
	run := func(declare bool, scope string) *fasthttp.RequestCtx {
		ctx := &fasthttp.RequestCtx{}
		ok := false
		h := func(ctx *fasthttp.RequestCtx) { ok = authorizeKey(ctx, key) }
		if declare {
			h = RequireScope(scope, h)
		}
		h(ctx)
		if ok != (ctx.Response.StatusCode() == 200) {
			t.Fatalf("decision and status disagree: %v %d", ok, ctx.Response.StatusCode())
		}
		return ctx
	}

	if ctx := run(true, model.ScopeImages); ctx.Response.StatusCode() != 200 {
		t.Fatalf("allowed scope rejected: %d", ctx.Response.StatusCode())
	}
	if ctx := run(true, model.ScopeNone); ctx.Response.StatusCode() != 200 {
		t.Fatal("unscoped self route rejected")
	}

	ctx := run(true, model.ScopeChat)
	var body struct {
		Error model.ErrorDetail `json:"error"`
	}
	json.Unmarshal(ctx.Response.Body(), &body)
	if ctx.Response.StatusCode() != 403 || body.Error.RequiredScope != model.ScopeChat || body.Error.Code == nil || *body.Error.Code != "insufficient_scope" || body.Error.Hint == "" {
		t.Fatalf("deny body: %d %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if got := string(ctx.Response.Header.Peek("WWW-Authenticate")); got != `Bearer error="insufficient_scope", scope="chat.completions"` {
		t.Fatalf("WWW-Authenticate %q", got)
	}

	if ctx := run(false, ""); ctx.Response.StatusCode() != 403 {
		t.Fatal("undeclared route must fail closed")
	}
}

func TestAuthErrorChallenge(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	writeAuthError(ctx, "Missing API key", "missing_api_key")
	if ctx.Response.StatusCode() != 401 || string(ctx.Response.Header.Peek("WWW-Authenticate")) != "Bearer" {
		t.Fatal("missing challenge")
	}
	var body struct {
		Error model.ErrorDetail `json:"error"`
	}
	json.Unmarshal(ctx.Response.Body(), &body)
	if body.Error.Hint == "" || body.Error.Type != "auth_error" {
		t.Fatalf("body %s", ctx.Response.Body())
	}
}

func TestSessionTokenAcceptsXAPIKey(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.Set("X-API-Key", "sk-op-abc")
	if peekSessionToken(ctx) != "sk-op-abc" {
		t.Fatal("X-API-Key ignored")
	}
}
