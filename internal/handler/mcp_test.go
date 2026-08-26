package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/router"
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
	want := map[string]bool{
		"chat": false, "list_models": false, "generate_image": false, "embed": false, "web_search": false,
		"generate_video": false, "check_job": false, "generate_music": false, "text_to_speech": false,
		"transcribe_audio": false, "generate_3d": false,
	}
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

// toolResultText extracts the text payload from a tools/call result that went
// through a JSON round-trip (so content is []any, not typed slices).
func toolResultText(t *testing.T, resp rpcResponse) string {
	t.Helper()
	res, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("result not object: %T", resp.Result)
	}
	content, ok := res["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("missing content: %#v", res)
	}
	first, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("bad content item: %T", content[0])
	}
	text, _ := first["text"].(string)
	return text
}

func TestMCPUnknownTool(t *testing.T) {
	h := &MCPHandler{}
	_, resp := mcpDo(t, h, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"nope","arguments":{}}}`)
	if resp.Error == nil || resp.Error.Code != -32602 {
		t.Fatalf("expected -32602, got %+v", resp.Error)
	}
}

func TestMCPDisabledToolsReturnIsError(t *testing.T) {
	h := &MCPHandler{}
	for _, name := range []string{"generate_video", "generate_music", "text_to_speech", "transcribe_audio", "generate_3d"} {
		body := fmt.Sprintf(`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":%q,"arguments":{"model":"m"}}}`, name)
		_, resp := mcpDo(t, h, body)
		if resp.Error != nil {
			t.Fatalf("%s: unexpected rpc error: %+v", name, resp.Error)
		}
		res, ok := resp.Result.(map[string]any)
		if !ok || res["isError"] != true {
			t.Fatalf("%s: expected isError result, got %#v", name, resp.Result)
		}
	}
}

func TestClassifyModality(t *testing.T) {
	cases := []struct {
		id   string
		cfg  *model.ModelConfig
		want string
	}{
		{"gpt-5", &model.ModelConfig{}, "chat"},
		{"mistral-embed", &model.ModelConfig{}, "embedding"},
		{"openpaths-embed", &model.ModelConfig{}, "embedding"},
		{"whisper-large-v3-turbo", &model.ModelConfig{}, "transcription"},
		{"xai-stt", &model.ModelConfig{}, "transcription"},
		{"local-whisper", &model.ModelConfig{}, "transcription"},
		{"xai-tts", &model.ModelConfig{}, "speech"},
		{"pocket-tts", &model.ModelConfig{PricePerMinute: 0.01}, "speech"},
		{"voice-x", &model.ModelConfig{PricePer1MCharacters: 10}, "speech"},
		{"music-2.5", &model.ModelConfig{PricePerImage: 0.03}, "music"},
		{"mg-music", &model.ModelConfig{}, "music"},
		{"mg-sfx", &model.ModelConfig{PricePerImage: 0.02}, "music"},
		{"mg-tts", &model.ModelConfig{}, "speech"},
		{"wan-animate-fast", &model.ModelConfig{PricePerSecond: 0.1}, "video"},
		{"h3-image-edit", &model.ModelConfig{PricePerImage: 0.03}, "image"},
		{"video-dramatize", &model.ModelConfig{PricePerSecond: 0.1}, "video"},
		{"pixal3d-image-to-3d", &model.ModelConfig{PricePerRequest: 0.3}, "3d"},
		{"sora-2", &model.ModelConfig{PricePerVideo: 0.1}, "video"},
		{"fal-ai/veo3.1", &model.ModelConfig{PricePerSecond: 0.25}, "video"},
		{"flux-3-video-draft", &model.ModelConfig{}, "video"},
		{"gpt-image-1", &model.ModelConfig{PricePerImage: 0.04}, "image"},
		{"gemini-2.5-flash-image", &model.ModelConfig{}, "image"},
		{"chronos2", &model.ModelConfig{}, "forecasting"},
	}
	for _, c := range cases {
		if got := classifyModality(c.id, c.cfg); got != c.want {
			t.Errorf("classifyModality(%q) = %q, want %q", c.id, got, c.want)
		}
	}
}

func TestMCPListModelsModalityAndFilter(t *testing.T) {
	models := []model.ModelConfig{
		{ID: "gpt-5", Provider: "openai", ContextWindow: 400000},
		{ID: "sora-2", Provider: "openai", PricePerVideo: 0.1},
		{ID: "xai-tts", Provider: "xai", PricePer1MCharacters: 30},
		{ID: "mistral-embed", Provider: "mistral"},
		{ID: "old-model", Provider: "openai", Deprecated: true, DeprecatedNote: "use gpt-5"},
	}
	h := &MCPHandler{router: router.New(nil, models)}
	_, resp := mcpDo(t, h, `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"list_models","arguments":{"modality":"video"}}}`)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %+v", resp.Error)
	}
	text := toolResultText(t, resp)
	var parsed struct {
		Count  int `json:"count"`
		Models []struct {
			ID       string         `json:"id"`
			OwnedBy  string         `json:"owned_by"`
			Modality string         `json:"modality"`
			Pricing  map[string]any `json:"pricing"`
		} `json:"models"`
	}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("unmarshal result: %v\n%s", err, text)
	}
	if parsed.Count != 1 || len(parsed.Models) != 1 {
		t.Fatalf("expected 1 video model, got %d:\n%s", parsed.Count, text)
	}
	m := parsed.Models[0]
	if m.ID != "sora-2" || m.Modality != "video" || m.OwnedBy != "openai" {
		t.Fatalf("bad entry: %+v", m)
	}
	if m.Pricing["per_video"] != 0.1 {
		t.Fatalf("expected per_video pricing, got %v", m.Pricing)
	}

	_, resp = mcpDo(t, h, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"list_models","arguments":{"filter":"xai"}}}`)
	text = toolResultText(t, resp)
	if !strings.Contains(text, `"xai-tts"`) || strings.Contains(text, `"gpt-5"`) {
		t.Fatalf("filter=xai should match only xai provider entries, got:\n%s", text)
	}

	_, resp = mcpDo(t, h, `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"list_models","arguments":{"filter":"old"}}}`)
	text = toolResultText(t, resp)
	if !strings.Contains(text, `"deprecated": true`) || !strings.Contains(text, `"deprecated_note": "use gpt-5"`) {
		t.Fatalf("deprecated flags missing, got:\n%s", text)
	}
}
