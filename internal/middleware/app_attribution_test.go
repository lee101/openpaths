package middleware

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func TestAppAttributionKeepsTitleOnlyCallers(t *testing.T) {
	var ctx fasthttp.RequestCtx
	ctx.Request.Header.Set("X-Title", "Example Agent")

	called := false
	h := AppAttribution(nil)(func(ctx *fasthttp.RequestCtx) {
		called = true
		if got := AppTitle(ctx); got != "Example Agent" {
			t.Fatalf("AppTitle = %q, want title-only attribution", got)
		}
		if got := AppURL(ctx); got != "" {
			t.Fatalf("AppURL = %q, want empty URL for title-only attribution", got)
		}
	})
	h(&ctx)
	if !called {
		t.Fatal("next handler was not called")
	}
}
