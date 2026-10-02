package handler

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

func decodeAnth(t *testing.T, body string) *anthRequest {
	t.Helper()
	var req anthRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatal(err)
	}
	return &req
}

func TestAnthToInternal_SystemArrayWithoutCacheControlStaysText(t *testing.T) {
	req := decodeAnth(t, `{"model":"m","max_tokens":1,"system":[{"type":"text","text":"a"},{"type":"text","text":"b"}],"messages":[{"role":"user","content":"hi"}]}`)
	chat := anthToInternal(req)
	if s, ok := chat.Messages[0].Content.(string); !ok || s != "ab" {
		t.Fatalf("system = %#v, want ab", chat.Messages[0].Content)
	}
}

func TestAnthToInternal_SystemArrayKeptWhenCacheControlPresent(t *testing.T) {
	req := decodeAnth(t, `{"model":"m","max_tokens":1,"system":[{"type":"text","text":"a"},{"type":"text","text":"b","cache_control":{"type":"ephemeral","ttl":"1h"}}],"messages":[{"role":"user","content":"hi"}]}`)
	chat := anthToInternal(req)
	parts, ok := chat.Messages[0].Content.([]any)
	if !ok || len(parts) != 2 {
		t.Fatalf("system = %#v, want 2 blocks", chat.Messages[0].Content)
	}
	b := parts[1].(map[string]any)
	if b["text"] != "b" || b["cache_control"] == nil {
		t.Fatalf("block = %#v", b)
	}
	if parts[0].(map[string]any)["cache_control"] != nil {
		t.Fatal("first block got cache_control")
	}
}

func TestAnthToInternal_ToolCacheControl(t *testing.T) {
	req := decodeAnth(t, `{"model":"m","max_tokens":1,"messages":[{"role":"user","content":"hi"}],"tools":[{"name":"a","input_schema":{}},{"name":"b","input_schema":{},"cache_control":{"type":"ephemeral","ttl":"5m"}}]}`)
	chat := anthToInternal(req)
	if chat.Tools[0].CacheControl != nil {
		t.Fatal("tool a got cache_control")
	}
	cc := chat.Tools[1].CacheControl
	if cc == nil || cc.Type != "ephemeral" || cc.TTL != "5m" {
		t.Fatalf("tool b cc = %#v", cc)
	}
}

func TestAnthToInternal_MessageBlocksKeepCacheControl(t *testing.T) {
	req := decodeAnth(t, `{"model":"m","max_tokens":1,"messages":[{"role":"user","content":[{"type":"text","text":"x","cache_control":{"type":"ephemeral"}}]}]}`)
	chat := anthToInternal(req)
	parts := chat.Messages[0].Content.([]any)
	if parts[0].(map[string]any)["cache_control"] == nil {
		t.Fatal("message cache_control lost")
	}
}

func TestRequestSessionID(t *testing.T) {
	cases := []struct {
		name, header, body, want string
	}{
		{"none", "", `{"model":"m"}`, ""},
		{"body", "", `{"model":"m","session_id":"s-body"}`, "s-body"},
		{"header", "s-head", `{"model":"m"}`, "s-head"},
		{"body wins", "s-head", `{"session_id":"s-body"}`, "s-body"},
		{"non-string body", "s-head", `{"session_id":5}`, "s-head"},
		{"trimmed", "  s  ", `{}`, "s"},
		{"too long", strings.Repeat("a", 300), `{}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &fasthttp.RequestCtx{}
			if tc.header != "" {
				ctx.Request.Header.Set("x-session-id", tc.header)
			}
			if got := requestSessionID(ctx, []byte(tc.body)); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
