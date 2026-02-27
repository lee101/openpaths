package handler

import (
	"encoding/json"

	"github.com/openpath/openpath/internal/model"
	"github.com/valyala/fasthttp"
)

func writeError(ctx *fasthttp.RequestCtx, status int, errType, message string) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	resp := model.ErrorResponse{
		Error: model.ErrorDetail{
			Message: message,
			Type:    errType,
		},
	}
	json.NewEncoder(ctx).Encode(resp)
}

func writeJSON(ctx *fasthttp.RequestCtx, status int, v any) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(v)
}
