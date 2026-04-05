package handler

import (
	"encoding/json"
	"fmt"

	"github.com/openpaths/openpaths/internal/model"
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
	data, _ := json.Marshal(resp)
	ctx.SetBody(data)
}

func writeJSON(ctx *fasthttp.RequestCtx, status int, v any) {
	ctx.SetStatusCode(status)
	ctx.SetContentType("application/json")
	data, _ := json.Marshal(v)
	ctx.SetBody(data)
}

func WriteJSONPublic(ctx *fasthttp.RequestCtx, status int, v any) {
	writeJSON(ctx, status, v)
}

func formatUSDExact(amountUnits int64) string {
	sign := ""
	if amountUnits < 0 {
		sign = "-"
		amountUnits = -amountUnits
	}

	return fmt.Sprintf("%s%d.%04d", sign, amountUnits/10000, amountUnits%10000)
}
