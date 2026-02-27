package middleware

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestChainOrder(t *testing.T) {
	var order []int

	mw1 := func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			order = append(order, 1)
			next(ctx)
		}
	}

	mw2 := func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			order = append(order, 2)
			next(ctx)
		}
	}

	mw3 := func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			order = append(order, 3)
			next(ctx)
		}
	}

	chain := Chain(mw1, mw2, mw3)
	handler := chain(func(ctx *fasthttp.RequestCtx) {
		order = append(order, 4)
	})

	handler(&fasthttp.RequestCtx{})

	expected := []int{1, 2, 3, 4}
	if len(order) != len(expected) {
		t.Fatalf("got %d calls, want %d", len(order), len(expected))
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %d, want %d", i, order[i], v)
		}
	}
}

func TestChainShortCircuit(t *testing.T) {
	handlerCalled := false

	blocker := func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			ctx.SetStatusCode(403)
			// Don't call next - short circuit
		}
	}

	chain := Chain(blocker)
	handler := chain(func(ctx *fasthttp.RequestCtx) {
		handlerCalled = true
	})

	ctx := &fasthttp.RequestCtx{}
	handler(ctx)

	if handlerCalled {
		t.Error("handler should not have been called")
	}
	if ctx.Response.StatusCode() != 403 {
		t.Errorf("got status %d, want 403", ctx.Response.StatusCode())
	}
}
