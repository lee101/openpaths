package handler

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/valyala/fasthttp"
)

const maxErrorMessageBytes = 8192

func clipErrorMessage(message string) string {
	if len(message) <= maxErrorMessageBytes {
		return message
	}
	n := maxErrorMessageBytes
	for n > 0 && !utf8.RuneStart(message[n]) {
		n--
	}
	return message[:n]
}

func upstreamClientStatus(status int) int {
	switch status {
	case 401, 403:
		return 502
	}
	return status
}

func writeError(ctx *fasthttp.RequestCtx, status int, errType, message string) {
	message = clipErrorMessage(message)
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
