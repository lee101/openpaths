package middleware

import (
	"log"
	"runtime/debug"

	"github.com/valyala/fasthttp"
)

// Recovery recovers from panics and returns 500.
func Recovery() Middleware {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC: %v\n%s", r, debug.Stack())
					ctx.SetStatusCode(500)
					ctx.SetContentType("application/json")
					ctx.SetBodyString(`{"error":{"message":"Internal server error","type":"server_error"}}`)
				}
			}()
			next(ctx)
		}
	}
}
