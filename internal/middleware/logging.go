package middleware

import (
	"log"
	"time"

	"github.com/valyala/fasthttp"
)

// Logging logs request method, path, status, and duration.
func Logging() Middleware {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			start := time.Now()
			next(ctx)
			log.Printf("%s %s %d %s",
				ctx.Method(), ctx.Path(), ctx.Response.StatusCode(), time.Since(start))
		}
	}
}
