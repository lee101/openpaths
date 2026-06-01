package handler

import (
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/middleware"
)

type requestApp struct {
	ID         string
	URL        string
	Title      string
	Categories []string
}

func requestAppAttribution(ctx *fasthttp.RequestCtx) requestApp {
	return requestApp{
		ID:         middleware.AppID(ctx),
		URL:        middleware.AppURL(ctx),
		Title:      middleware.AppTitle(ctx),
		Categories: middleware.AppCategories(ctx),
	}
}
