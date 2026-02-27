package middleware

import "github.com/valyala/fasthttp"

// Middleware wraps a fasthttp handler.
type Middleware func(fasthttp.RequestHandler) fasthttp.RequestHandler

// Chain applies middlewares in order: first middleware is outermost.
func Chain(middlewares ...Middleware) Middleware {
	return func(final fasthttp.RequestHandler) fasthttp.RequestHandler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}
