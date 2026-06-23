package handler

import (
	"encoding/json"
	"testing"

	"github.com/valyala/fasthttp"
)

func mcpDo(t *testing.T, h *MCPHandler, body string) (*fasthttp.RequestCtx, rpcResponse) {
	t.Helper()
	var ctx fasthttp.RequestCtx
	var req fasthttp.Request
	req.Header.SetMethod("POST")
	req.SetRequestURI("/mcp")
	req.SetBody([]byte(body))
	ctx.Init(&req, nil, nil)
	h.HandleMCP(&ctx)
	var resp rpcResponse
	_ = json.Unmarshal(ctx.Response.Body(), &resp)
	return &ctx, resp
}

func TestMCPInitialize(t *testing.T) {
	h := &MCPHandler{}
	_, resp := mcpDo(t, h, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	m, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result not object: %T", resp.Result)
	}
	if m["protocolVersion"] != mcpProtocolVersion {
		t.Fatalf("bad protocolVersion: %v", m["protocolVersion"])
	}
	si, _ := m["serverInfo"].(map[string]any)
	if si["name"] != mcpServerName {
		t.Fatalf("bad serverInfo: %v", m["serverInfo"])
	}
}

func TestMCPToolsList(t *testing.T) {
	h := &MCPHandler{}
	_, resp := mcpDo(t, h, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	m := resp.Result.(map[string]any)
	tools := m["tools"].([]any)
	if len(tools) != len(mcpTools) || len(tools) == 0 {
		t.Fatalf("expected %d tools, got %d", len(mcpTools), len(tools))
	}
	want := map[string]bool{"chat": false, "list_models": false, "generate_image": false, "embed": false, "web_search": false}
	for _, tl := range tools {
		name := tl.(map[string]any)["name"].(string)
		if _, ok := want[name]; ok {
			want[name] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Fatalf("missing tool %s", name)
		}
	}
}

func TestMCPNotificationNoResponse(t *testing.T) {
	h := &MCPHandler{}
	ctx, _ := mcpDo(t, h, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if ctx.Response.StatusCode() != 202 {
		t.Fatalf("expected 202 for notification, got %d", ctx.Response.StatusCode())
	}
	if len(ctx.Response.Body()) != 0 {
		t.Fatalf("expected empty body, got %q", ctx.Response.Body())
	}
}

func TestMCPUnknownMethod(t *testing.T) {
	h := &MCPHandler{}
	_, resp := mcpDo(t, h, `{"jsonrpc":"2.0","id":3,"method":"does/not/exist"}`)
	if resp.Error == nil || resp.Error.Code != -32601 {
		t.Fatalf("expected -32601, got %+v", resp.Error)
	}
}

func TestMCPParseError(t *testing.T) {
	h := &MCPHandler{}
	_, resp := mcpDo(t, h, `{not json`)
	if resp.Error == nil || resp.Error.Code != -32700 {
		t.Fatalf("expected -32700, got %+v", resp.Error)
	}
}

func TestMCPBatch(t *testing.T) {
	h := &MCPHandler{}
	var ctx fasthttp.RequestCtx
	var req fasthttp.Request
	req.Header.SetMethod("POST")
	req.SetRequestURI("/mcp")
	req.SetBody([]byte(`[{"jsonrpc":"2.0","id":1,"method":"ping"},{"jsonrpc":"2.0","method":"notifications/initialized"},{"jsonrpc":"2.0","id":2,"method":"tools/list"}]`))
	ctx.Init(&req, nil, nil)
	h.HandleMCP(&ctx)
	var responses []rpcResponse
	if err := json.Unmarshal(ctx.Response.Body(), &responses); err != nil {
		t.Fatalf("unmarshal batch: %v", err)
	}
	if len(responses) != 2 {
		t.Fatalf("expected 2 responses (notification excluded), got %d", len(responses))
	}
}

func TestMCPGetRejected(t *testing.T) {
	h := &MCPHandler{}
	var ctx fasthttp.RequestCtx
	var req fasthttp.Request
	req.Header.SetMethod("GET")
	req.SetRequestURI("/mcp")
	ctx.Init(&req, nil, nil)
	h.HandleMCP(&ctx)
	if ctx.Response.StatusCode() != 405 {
		t.Fatalf("expected 405 for GET, got %d", ctx.Response.StatusCode())
	}
}
