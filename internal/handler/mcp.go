package handler

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/router"
)

// MCP server: exposes all OpenPaths models over the Model Context Protocol
// (JSON-RPC 2.0 over Streamable HTTP, stateless). Tool calls are dispatched
// to the existing HTTP handlers via an internal sub-context so routing,
// fallback, billing, metrics and saving are reused unchanged.

const (
	mcpProtocolVersion = "2025-06-18"
	mcpServerName      = "openpaths"
	mcpServerVersion   = "1.0.0"
)

type MCPHandler struct {
	router    *router.Router
	chat      *ChatHandler
	models    *ModelsHandler
	image     *ImageHandler
	embedding *EmbeddingHandler
	search    *SearchHandler
}

func NewMCPHandler(r *router.Router, chat *ChatHandler, models *ModelsHandler, image *ImageHandler, embedding *EmbeddingHandler, search *SearchHandler) *MCPHandler {
	return &MCPHandler{router: r, chat: chat, models: models, image: image, embedding: embedding, search: search}
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

func (h *MCPHandler) HandleMCP(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		ctx.SetStatusCode(405)
		ctx.SetContentType("application/json")
		ctx.SetBody([]byte(`{"jsonrpc":"2.0","error":{"code":-32000,"message":"Use POST for MCP"},"id":null}`))
		return
	}

	body := ctx.PostBody()
	trimmed := strings.TrimLeft(string(body), " \t\r\n")

	// Batch request (JSON array).
	if strings.HasPrefix(trimmed, "[") {
		var batch []rpcRequest
		if err := json.Unmarshal(body, &batch); err != nil {
			writeRPCError(ctx, nil, -32700, "Parse error")
			return
		}
		responses := make([]*rpcResponse, 0, len(batch))
		for i := range batch {
			if resp := h.handleOne(ctx, &batch[i]); resp != nil {
				responses = append(responses, resp)
			}
		}
		if len(responses) == 0 {
			ctx.SetStatusCode(202)
			return
		}
		writeJSON(ctx, 200, responses)
		return
	}

	var req rpcRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeRPCError(ctx, nil, -32700, "Parse error")
		return
	}
	resp := h.handleOne(ctx, &req)
	if resp == nil {
		ctx.SetStatusCode(202)
		return
	}
	writeJSON(ctx, 200, resp)
}

// handleOne returns nil for notifications (no id / notifications/*).
func (h *MCPHandler) handleOne(ctx *fasthttp.RequestCtx, req *rpcRequest) *rpcResponse {
	isNotification := len(req.ID) == 0 || strings.HasPrefix(req.Method, "notifications/")

	result, rpcErr := h.route(ctx, req)
	if isNotification {
		return nil
	}
	if rpcErr != nil {
		return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: rpcErr}
	}
	return &rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
}

func (h *MCPHandler) route(ctx *fasthttp.RequestCtx, req *rpcRequest) (any, *rpcError) {
	switch req.Method {
	case "initialize":
		return map[string]any{
			"protocolVersion": mcpProtocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    mcpServerName,
				"version": mcpServerVersion,
			},
			"instructions": "OpenPaths MCP server. Call `list_models` to discover models, then `chat` with any model id. Image, embedding and web_search tools are also available.",
		}, nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": mcpTools}, nil
	case "tools/call":
		return h.callTool(ctx, req.Params)
	case "resources/list":
		return map[string]any{"resources": []any{}}, nil
	case "prompts/list":
		return map[string]any{"prompts": []any{}}, nil
	case "notifications/initialized", "notifications/cancelled":
		return map[string]any{}, nil
	default:
		return nil, &rpcError{Code: -32601, Message: "Method not found: " + req.Method}
	}
}

// ---- tools ----

var mcpTools = []map[string]any{
	{
		"name":        "chat",
		"description": "Run a chat completion against any OpenPaths model (OpenAI, Anthropic, Google, xAI, and more). Returns the assistant's text reply.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"model":  map[string]any{"type": "string", "description": "Model id, e.g. gpt-5, claude-opus-5, gemini-2.5-pro, or 'auto'."},
				"prompt": map[string]any{"type": "string", "description": "User prompt (shorthand for a single user message)."},
				"system": map[string]any{"type": "string", "description": "Optional system instruction."},
				"messages": map[string]any{
					"type":        "array",
					"description": "Full message list. Each item: {role, content}. Overrides prompt/system if set.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"role":    map[string]any{"type": "string"},
							"content": map[string]any{"type": "string"},
						},
						"required": []string{"role", "content"},
					},
				},
				"max_tokens":       map[string]any{"type": "integer"},
				"temperature":      map[string]any{"type": "number"},
				"reasoning_effort": map[string]any{"type": "string", "enum": []string{"auto", "none", "minimal", "low", "medium", "high", "xhigh", "max"}, "description": "Reasoning depth. 'auto' lets OpenPaths pick per prompt."},
			},
			"required": []string{"model"},
		},
	},
	{
		"name":        "list_models",
		"description": "List available OpenPaths models with owner and context window. Optional case-insensitive substring filter.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filter": map[string]any{"type": "string", "description": "Substring to match against model id or owner."},
			},
		},
	},
	{
		"name":        "generate_image",
		"description": "Generate an image from a text prompt and return image URL(s).",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"model":  map[string]any{"type": "string", "description": "Image model id, e.g. gpt-image-1, flux-schnell."},
				"prompt": map[string]any{"type": "string"},
				"size":   map[string]any{"type": "string", "description": "e.g. 1024x1024."},
				"n":      map[string]any{"type": "integer", "description": "Number of images (default 1)."},
			},
			"required": []string{"model", "prompt"},
		},
	},
	{
		"name":        "embed",
		"description": "Create embedding vectors for text input.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"model": map[string]any{"type": "string"},
				"input": map[string]any{"type": "string", "description": "Text to embed."},
			},
			"required": []string{"model", "input"},
		},
	},
	{
		"name":        "web_search",
		"description": "Search the web and return ranked results.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query":      map[string]any{"type": "string"},
				"numResults": map[string]any{"type": "integer", "description": "Default 10."},
			},
			"required": []string{"query"},
		},
	},
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (h *MCPHandler) callTool(ctx *fasthttp.RequestCtx, params json.RawMessage) (any, *rpcError) {
	var p toolCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: -32602, Message: "Invalid params: " + err.Error()}
	}
	args := p.Arguments
	if len(args) == 0 {
		args = []byte("{}")
	}

	switch p.Name {
	case "chat":
		return h.toolChat(ctx, args)
	case "list_models":
		return h.toolListModels(args)
	case "generate_image":
		return h.toolGenerateImage(ctx, args)
	case "embed":
		return h.toolEmbed(ctx, args)
	case "web_search":
		return h.toolWebSearch(ctx, args)
	default:
		return nil, &rpcError{Code: -32602, Message: "Unknown tool: " + p.Name}
	}
}

func (h *MCPHandler) toolChat(ctx *fasthttp.RequestCtx, args json.RawMessage) (any, *rpcError) {
	var a struct {
		Model           string              `json:"model"`
		Prompt          string              `json:"prompt"`
		System          string              `json:"system"`
		Messages        []model.ChatMessage `json:"messages"`
		MaxTokens       *int                `json:"max_tokens"`
		Temperature     *float64            `json:"temperature"`
		ReasoningEffort string              `json:"reasoning_effort"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, &rpcError{Code: -32602, Message: "Invalid arguments: " + err.Error()}
	}
	if a.Model == "" {
		return toolError("model is required"), nil
	}

	messages := a.Messages
	if len(messages) == 0 {
		if a.System != "" {
			messages = append(messages, model.ChatMessage{Role: "system", Content: a.System})
		}
		messages = append(messages, model.ChatMessage{Role: "user", Content: a.Prompt})
	}

	reqBody := model.ChatCompletionRequest{
		Model:           a.Model,
		Messages:        messages,
		MaxTokens:       a.MaxTokens,
		Temperature:     a.Temperature,
		ReasoningEffort: a.ReasoningEffort,
	}
	status, body := h.dispatch(ctx, h.chat.HandleChatCompletion, "/v1/chat/completions", reqBody)
	if status != 200 {
		return toolError(upstreamMessage(body)), nil
	}
	var resp model.ChatCompletionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return toolText(string(body)), nil
	}
	if len(resp.Choices) > 0 && resp.Choices[0].Message != nil {
		return toolText(contentString(resp.Choices[0].Message.Content)), nil
	}
	return toolText(string(body)), nil
}

func (h *MCPHandler) toolListModels(args json.RawMessage) (any, *rpcError) {
	var a struct {
		Filter string `json:"filter"`
	}
	_ = json.Unmarshal(args, &a)
	filter := strings.ToLower(strings.TrimSpace(a.Filter))

	infos := h.router.ListModels()
	out := make([]map[string]any, 0, len(infos))
	for _, m := range infos {
		if filter != "" && !strings.Contains(strings.ToLower(m.ID), filter) && !strings.Contains(strings.ToLower(m.OwnedBy), filter) {
			continue
		}
		out = append(out, map[string]any{
			"id":             m.ID,
			"owned_by":       m.OwnedBy,
			"context_window": m.ContextWindow,
		})
	}
	data, _ := json.MarshalIndent(map[string]any{"count": len(out), "models": out}, "", "  ")
	return toolText(string(data)), nil
}

func (h *MCPHandler) toolGenerateImage(ctx *fasthttp.RequestCtx, args json.RawMessage) (any, *rpcError) {
	var a struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		Size   string `json:"size"`
		N      int    `json:"n"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, &rpcError{Code: -32602, Message: "Invalid arguments: " + err.Error()}
	}
	if a.Model == "" || a.Prompt == "" {
		return toolError("model and prompt are required"), nil
	}
	reqBody := model.ImageGenerationRequest{Model: a.Model, Prompt: a.Prompt, Size: a.Size, N: a.N}
	status, body := h.dispatch(ctx, h.image.HandleImageGeneration, "/v1/images/generations", reqBody)
	if status != 200 {
		return toolError(upstreamMessage(body)), nil
	}
	var resp struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err == nil && len(resp.Data) > 0 {
		urls := make([]string, 0, len(resp.Data))
		for _, d := range resp.Data {
			if d.URL != "" {
				urls = append(urls, d.URL)
			} else if d.B64JSON != "" {
				urls = append(urls, "(base64 image, "+strconv.Itoa(len(d.B64JSON))+" bytes)")
			}
		}
		if len(urls) > 0 {
			return toolText(strings.Join(urls, "\n")), nil
		}
	}
	return toolText(string(body)), nil
}

func (h *MCPHandler) toolEmbed(ctx *fasthttp.RequestCtx, args json.RawMessage) (any, *rpcError) {
	var a struct {
		Model string `json:"model"`
		Input any    `json:"input"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, &rpcError{Code: -32602, Message: "Invalid arguments: " + err.Error()}
	}
	if a.Model == "" {
		return toolError("model is required"), nil
	}
	reqBody := model.EmbeddingRequest{Model: a.Model, Input: a.Input}
	status, body := h.dispatch(ctx, h.embedding.HandleEmbedding, "/v1/embeddings", reqBody)
	if status != 200 {
		return toolError(upstreamMessage(body)), nil
	}
	return toolText(string(body)), nil
}

func (h *MCPHandler) toolWebSearch(ctx *fasthttp.RequestCtx, args json.RawMessage) (any, *rpcError) {
	if h.search == nil {
		return toolError("web_search is not enabled on this server"), nil
	}
	var a map[string]any
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, &rpcError{Code: -32602, Message: "Invalid arguments: " + err.Error()}
	}
	if q, _ := a["query"].(string); strings.TrimSpace(q) == "" {
		return toolError("query is required"), nil
	}
	status, body := h.dispatch(ctx, h.search.HandleSearch, "/v1/search", a)
	if status != 200 {
		return toolError(upstreamMessage(body)), nil
	}
	return toolText(string(body)), nil
}

// ---- internal dispatch ----

// dispatch invokes an existing bare handler against a fresh sub-context that
// inherits the authenticated user/byok values from the parent request, then
// returns the response status and body. This reuses all routing/billing logic.
func (h *MCPHandler) dispatch(parent *fasthttp.RequestCtx, handler fasthttp.RequestHandler, uri string, payload any) (int, []byte) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 500, []byte(`{"error":{"message":"failed to encode request"}}`)
	}

	var req fasthttp.Request
	req.Header.SetMethod("POST")
	req.SetRequestURI(uri)
	req.Header.SetContentType("application/json")
	req.SetBody(body)

	var sub fasthttp.RequestCtx
	sub.Init(&req, parent.RemoteAddr(), nil)
	parent.VisitUserValuesAll(func(k, v any) {
		sub.SetUserValue(k, v)
	})

	handler(&sub)
	return sub.Response.StatusCode(), append([]byte(nil), sub.Response.Body()...)
}

// ---- helpers ----

func writeRPCError(ctx *fasthttp.RequestCtx, id json.RawMessage, code int, msg string) {
	writeJSON(ctx, 200, &rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}})
}

func toolText(text string) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
	}
}

func toolError(text string) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": true,
	}
}

func contentString(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var b strings.Builder
		for _, part := range v {
			if m, ok := part.(map[string]any); ok {
				if t, _ := m["text"].(string); t != "" {
					b.WriteString(t)
				}
			}
		}
		return b.String()
	default:
		data, _ := json.Marshal(content)
		return string(data)
	}
}

func upstreamMessage(body []byte) string {
	var e model.ErrorResponse
	if err := json.Unmarshal(body, &e); err == nil && e.Error.Message != "" {
		return e.Error.Message
	}
	return string(body)
}
