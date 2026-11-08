package handler

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/valyala/fasthttp"
)

const maxSessionIDLen = 256

// requestSessionID returns the sticky-routing session key from the body
// "session_id" or the x-session-id header (body wins). Empty when absent or
// oversized.
func requestSessionID(ctx *fasthttp.RequestCtx, body []byte) string {
	if bytes.Contains(body, []byte(`"session_id"`)) {
		var probe struct {
			SessionID any `json:"session_id"`
		}
		if json.Unmarshal(body, &probe) == nil {
			if s, ok := probe.SessionID.(string); ok {
				if s = strings.TrimSpace(s); s != "" && len(s) <= maxSessionIDLen {
					return s
				}
			}
		}
	}
	s := strings.TrimSpace(string(ctx.Request.Header.Peek("x-session-id")))
	if len(s) > maxSessionIDLen {
		return ""
	}
	return s
}
