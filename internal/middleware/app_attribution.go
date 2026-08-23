package middleware

import (
	"log"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
)

const (
	CtxKeyAppID         = "app_id"
	CtxKeyAppURL        = "app_url"
	CtxKeyAppTitle      = "app_title"
	CtxKeyAppCategories = "app_categories"
)

func AppAttribution(appQ *queries.AppQueries) Middleware {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			appURL := firstHeader(ctx, "HTTP-Referer", "Referer")
			title := firstHeader(ctx, "X-OpenRouter-Title", "X-Title")
			categories := parseCategories(firstHeader(ctx, "X-OpenRouter-Categories"))

			// Keep title-only callers visible in private account/admin analytics as
			// well. A URL is still required to create a public app record, but
			// OpenRouter-compatible clients commonly provide X-Title on its own.
			if appURL != "" || title != "" {
				ctx.SetUserValue(CtxKeyAppURL, appURL)
				ctx.SetUserValue(CtxKeyAppTitle, title)
				ctx.SetUserValue(CtxKeyAppCategories, categories)
				if appQ != nil && appURL != "" {
					appID, err := appQ.UpsertAttribution(ctx, appURL, title, categories)
					if err != nil {
						log.Printf("app-attribution: upsert %q: %v", appURL, err)
					} else if appID != "" {
						ctx.SetUserValue(CtxKeyAppID, appID)
					}
				}
			}

			next(ctx)
		}
	}
}

func AppID(ctx *fasthttp.RequestCtx) string {
	v, _ := ctx.UserValue(CtxKeyAppID).(string)
	return v
}

func AppURL(ctx *fasthttp.RequestCtx) string {
	v, _ := ctx.UserValue(CtxKeyAppURL).(string)
	return v
}

func AppTitle(ctx *fasthttp.RequestCtx) string {
	v, _ := ctx.UserValue(CtxKeyAppTitle).(string)
	return v
}

func AppCategories(ctx *fasthttp.RequestCtx) []string {
	v, _ := ctx.UserValue(CtxKeyAppCategories).([]string)
	return v
}

func firstHeader(ctx *fasthttp.RequestCtx, names ...string) string {
	for _, name := range names {
		if v := strings.TrimSpace(string(ctx.Request.Header.Peek(name))); v != "" {
			return v
		}
		if v := strings.TrimSpace(string(ctx.Request.Header.Peek(strings.ToLower(name)))); v != "" {
			return v
		}
	}
	return ""
}

func parseCategories(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		out = append(out, part)
		if len(out) >= 10 {
			break
		}
	}
	return out
}
