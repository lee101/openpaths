package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"sort"
	"strconv"
	"strings"
	"time"

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
	video     *VideoHandler
	music     *MusicHandler
	speech    *SpeechHandler
	stt       *TranscriptionHandler
	textTo3D  *TextTo3DHandler
}

func NewMCPHandler(r *router.Router, chat *ChatHandler, models *ModelsHandler, image *ImageHandler, embedding *EmbeddingHandler, search *SearchHandler) *MCPHandler {
	return &MCPHandler{router: r, chat: chat, models: models, image: image, embedding: embedding, search: search}
}

// Optional modality handlers; tools degrade gracefully when unset.
func (h *MCPHandler) SetVideoHandler(v *VideoHandler)                 { h.video = v }
func (h *MCPHandler) SetMusicHandler(m *MusicHandler)                 { h.music = m }
func (h *MCPHandler) SetSpeechHandler(s *SpeechHandler)               { h.speech = s }
func (h *MCPHandler) SetTranscriptionHandler(t *TranscriptionHandler) { h.stt = t }
func (h *MCPHandler) SetTextTo3DHandler(t *TextTo3DHandler)           { h.textTo3D = t }

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
			"instructions": "OpenPaths MCP server exposing every OpenPaths model across all providers. Call `list_models` (supports `filter` substring and `modality` filters) to discover models; each entry's `modality` selects the tool: chat→`chat`, image→`generate_image`, video→`generate_video`, music→`generate_music`, speech→`text_to_speech`, transcription→`transcribe_audio`, 3d→`generate_3d`, embedding→`embed`. Video and 3D generations may return a job_id while rendering — poll it with `check_job`. Web search via `web_search`.",
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
		"description": "List all available OpenPaths models across every provider with modality, pricing summary, context window and capabilities. Filter by substring and/or modality.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"filter":   map[string]any{"type": "string", "description": "Case-insensitive substring matched against model id or owner/provider."},
				"modality": map[string]any{"type": "string", "enum": []string{"chat", "image", "video", "music", "speech", "transcription", "3d", "embedding", "forecasting"}, "description": "Only return models of this modality."},
			},
		},
	},
	{
		"name":        "generate_image",
		"description": "Generate an image from a text prompt and return image URL(s).",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"model":  map[string]any{"type": "string", "description": "Image model id, e.g. h3-image, gpt-image-1, flux-schnell."},
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
	{
		"name":        "generate_video",
		"description": "Generate a video from a text prompt (optionally with input image/video URLs). Returns the video URL, or a job_id to poll via check_job while rendering.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"model":           map[string]any{"type": "string", "description": "Video model id, e.g. kfold-video (ManifoldGen H3 cinematic with native audio), wan-animate / -fast / -xfast (character animation; requires both image_url and video_url), h3-control-video (restyle via control_type), remove-video-background (transparent WebM), video-dramatize (multi-shot dramatizer; duration is planned total edit seconds 8-100), sora-2, fal-ai/veo3.1."},
				"prompt":          map[string]any{"type": "string"},
				"image_url":       map[string]any{"type": "string", "description": "Optional first-frame / input image URL for image-to-video."},
				"end_image_url":   map[string]any{"type": "string", "description": "Optional last-frame image URL."},
				"video_url":       map[string]any{"type": "string", "description": "Optional input video URL for video-to-video / extension."},
				"duration":        map[string]any{"type": "string", "description": "Seconds as string, e.g. \"5\" or \"10\"."},
				"resolution":      map[string]any{"type": "string"},
				"aspect_ratio":    map[string]any{"type": "string"},
				"negative_prompt": map[string]any{"type": "string"},
				"seed":            map[string]any{"type": "integer"},
			},
			"required": []string{"model", "prompt"},
		},
	},
	{
		"name":        "check_job",
		"description": "Poll an async generation job (video or 3D) started by generate_video/generate_3d. Returns the job status and result when complete.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"kind":   map[string]any{"type": "string", "enum": []string{"video", "model3d", "text3d"}, "description": "Default video."},
				"job_id": map[string]any{"type": "string"},
			},
			"required": []string{"job_id"},
		},
	},
	{
		"name":        "generate_music",
		"description": "Generate instrumental music or a song from a prompt and optional lyrics. Returns an audio URL or base64 data URI.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"model":    map[string]any{"type": "string", "description": "e.g. mg-music (ManifoldGen MiniMax-Music3 full songs; optional duration seconds 30-300) or music-2.5. Use mg-sfx for short ~5s sound effects."},
				"prompt":   map[string]any{"type": "string", "description": "Style/genre description."},
				"lyrics":   map[string]any{"type": "string", "description": "Optional lyrics ([verse]/[chorus] sections supported)."},
				"duration": map[string]any{"type": "integer", "description": "Song length in seconds (mg-music only, 30-300)."},
			},
			"required": []string{"model"},
		},
	},
	{
		"name":        "text_to_speech",
		"description": "Synthesize speech from text. Returns an audio URL or base64 data URI.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"model":    map[string]any{"type": "string", "description": "e.g. mg-tts, xai-tts, pocket-tts, gemini-3.1-flash-tts-preview."},
				"input":    map[string]any{"type": "string", "description": "Text to speak."},
				"voice":    map[string]any{"type": "string"},
				"language": map[string]any{"type": "string"},
				"speed":    map[string]any{"type": "number"},
			},
			"required": []string{"model", "input"},
		},
	},
	{
		"name":        "transcribe_audio",
		"description": "Transcribe audio to text (Whisper-family and other STT models). Pass the audio file base64-encoded.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"audio_base64": map[string]any{"type": "string", "description": "Base64-encoded audio file bytes."},
				"filename":     map[string]any{"type": "string", "description": "File name with extension, e.g. clip.mp3. Default audio.mp3."},
				"model":        map[string]any{"type": "string", "description": "e.g. whisper-large-v3-turbo, gpt-4o-transcribe. Default server default."},
				"language":     map[string]any{"type": "string"},
				"prompt":       map[string]any{"type": "string", "description": "Optional spelling hint transcript."},
			},
			"required": []string{"audio_base64"},
		},
	},
	{
		"name":        "generate_3d",
		"description": "Generate a textured 3D model (GLB) from a text prompt or image URL. Returns the GLB asset, or a job_id to poll via check_job.",
		"inputSchema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt":       map[string]any{"type": "string", "description": "Text description (or pass image_url instead)."},
				"image_url":    map[string]any{"type": "string", "description": "Public http(s) image URL for image-to-3D."},
				"texture_size": map[string]any{"type": "integer"},
			},
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
	case "generate_video":
		return h.toolGenerateVideo(ctx, args)
	case "check_job":
		return h.toolCheckJob(ctx, args)
	case "generate_music":
		return h.toolGenerateMusic(ctx, args)
	case "text_to_speech":
		return h.toolSpeak(ctx, args)
	case "transcribe_audio":
		return h.toolTranscribe(ctx, args)
	case "generate_3d":
		return h.toolGenerate3D(ctx, args)
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
		Filter   string `json:"filter"`
		Modality string `json:"modality"`
	}
	_ = json.Unmarshal(args, &a)
	filter := strings.ToLower(strings.TrimSpace(a.Filter))
	modality := strings.ToLower(strings.TrimSpace(a.Modality))

	cfgs := h.router.ListModelConfigs()
	out := make([]map[string]any, 0, len(cfgs))
	for _, cfg := range cfgs {
		m := classifyModality(cfg.ID, cfg)
		if modality != "" && m != modality {
			continue
		}
		if filter != "" && !strings.Contains(strings.ToLower(cfg.ID), filter) && !strings.Contains(strings.ToLower(cfg.Provider), filter) {
			continue
		}
		entry := map[string]any{
			"id":             cfg.ID,
			"owned_by":       cfg.Provider,
			"modality":       m,
			"context_window": cfg.ContextWindow,
		}
		if cfg.MaxOutputTokens > 0 {
			entry["max_output_tokens"] = cfg.MaxOutputTokens
		}
		if len(cfg.Aliases) > 0 {
			entry["aliases"] = cfg.Aliases
		}
		if len(cfg.SupportedSizes) > 0 {
			entry["supported_sizes"] = cfg.SupportedSizes
		}
		if p := compactPricing(cfg); len(p) > 0 {
			entry["pricing"] = p
		}
		if cfg.Deprecated {
			entry["deprecated"] = true
			entry["deprecated_note"] = cfg.DeprecatedNote
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["id"].(string) < out[j]["id"].(string) })
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

func (h *MCPHandler) toolGenerateVideo(ctx *fasthttp.RequestCtx, args json.RawMessage) (any, *rpcError) {
	if h.video == nil {
		return toolError("video generation is not enabled on this server"), nil
	}
	var a struct {
		Model          string `json:"model"`
		Prompt         string `json:"prompt"`
		ImageURL       string `json:"image_url"`
		EndImageURL    string `json:"end_image_url"`
		VideoURL       string `json:"video_url"`
		Duration       string `json:"duration"`
		Resolution     string `json:"resolution"`
		AspectRatio    string `json:"aspect_ratio"`
		NegativePrompt string `json:"negative_prompt"`
		Seed           *int   `json:"seed"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, &rpcError{Code: -32602, Message: "Invalid arguments: " + err.Error()}
	}
	if a.Model == "" || a.Prompt == "" {
		return toolError("model and prompt are required"), nil
	}
	reqBody := model.VideoGenerationRequest{
		Model:          a.Model,
		Prompt:         a.Prompt,
		ImageURL:       a.ImageURL,
		EndImageURL:    a.EndImageURL,
		VideoURL:       a.VideoURL,
		Duration:       model.VideoDuration(a.Duration),
		Resolution:     a.Resolution,
		AspectRatio:    a.AspectRatio,
		NegativePrompt: a.NegativePrompt,
		Seed:           a.Seed,
	}
	status, body := h.dispatch(ctx, h.video.HandleVideoGeneration, "/v1/videos/generations", reqBody)
	switch {
	case status == 200:
		var resp model.VideoGenerationResponse
		if err := json.Unmarshal(body, &resp); err == nil && resp.VideoURL != "" {
			return toolText(resp.VideoURL), nil
		}
		return toolText(string(body)), nil
	case status == 202:
		return toolText(pendingJobText(body, "video")), nil
	default:
		return toolError(upstreamMessage(body)), nil
	}
}

func (h *MCPHandler) toolCheckJob(ctx *fasthttp.RequestCtx, args json.RawMessage) (any, *rpcError) {
	var a struct {
		Kind  string `json:"kind"`
		JobID string `json:"job_id"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, &rpcError{Code: -32602, Message: "Invalid arguments: " + err.Error()}
	}
	if a.JobID == "" {
		return toolError("job_id is required"), nil
	}
	var handler fasthttp.RequestHandler
	var uri string
	switch strings.ToLower(strings.TrimSpace(a.Kind)) {
	case "", "video":
		if h.video == nil {
			return toolError("video jobs are not enabled on this server"), nil
		}
		handler, uri = h.video.HandleVideoGenerationJob, "/v1/videos/generations/"+a.JobID
	case "text3d":
		if h.textTo3D == nil {
			return toolError("3D jobs are not enabled on this server"), nil
		}
		handler, uri = h.textTo3D.HandleTextTo3DGenerationJob, "/v1/3d/text-generations/"+a.JobID
	case "model3d":
		return toolError("kind=model3d is not supported; use kind=text3d or start via generate_3d"), nil
	default:
		return toolError("unknown kind: " + a.Kind), nil
	}
	status, body := h.dispatchRaw(ctx, handler, "GET", uri, "", nil)
	if status != 200 {
		return toolError(upstreamMessage(body)), nil
	}
	return toolText(string(body)), nil
}

func (h *MCPHandler) toolGenerateMusic(ctx *fasthttp.RequestCtx, args json.RawMessage) (any, *rpcError) {
	if h.music == nil {
		return toolError("music generation is not enabled on this server"), nil
	}
	var a struct {
		Model    string `json:"model"`
		Prompt   string `json:"prompt"`
		Lyrics   string `json:"lyrics"`
		Duration int    `json:"duration"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, &rpcError{Code: -32602, Message: "Invalid arguments: " + err.Error()}
	}
	if a.Model == "" || (a.Lyrics == "" && a.Prompt == "") {
		return toolError("model and prompt or lyrics are required"), nil
	}
	reqBody := model.MusicGenerationRequest{Model: a.Model, Prompt: a.Prompt, Lyrics: a.Lyrics, Duration: a.Duration, OutputFormat: "url"}
	status, body := h.dispatch(ctx, h.music.HandleMusicGeneration, "/v1/music/generations", reqBody)
	if status != 200 {
		return toolError(upstreamMessage(body)), nil
	}
	var resp model.MusicGenerationResponse
	if err := json.Unmarshal(body, &resp); err == nil && resp.Data != nil && resp.Data.Audio != "" {
		if audio := audioDataURI(resp.Data.Audio); audio != "" {
			return toolText(audio), nil
		}
	}
	return toolText(string(body)), nil
}

func (h *MCPHandler) toolSpeak(ctx *fasthttp.RequestCtx, args json.RawMessage) (any, *rpcError) {
	if h.speech == nil {
		return toolError("text-to-speech is not enabled on this server"), nil
	}
	var a struct {
		Model    string  `json:"model"`
		Input    string  `json:"input"`
		Voice    string  `json:"voice"`
		Language string  `json:"language"`
		Speed    float64 `json:"speed"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, &rpcError{Code: -32602, Message: "Invalid arguments: " + err.Error()}
	}
	if a.Model == "" || a.Input == "" {
		return toolError("model and input are required"), nil
	}
	reqBody := model.SpeechRequest{Model: a.Model, Input: a.Input, Voice: a.Voice, Language: a.Language, Speed: a.Speed}
	status, body := h.dispatch(ctx, h.speech.HandleSpeechGeneration, "/v1/audio/speech", reqBody)
	if status != 200 {
		return toolError(upstreamMessage(body)), nil
	}
	var resp model.SpeechResponse
	if err := json.Unmarshal(body, &resp); err == nil {
		if resp.AudioURL != "" {
			return toolText(resp.AudioURL), nil
		}
		if resp.Audio != "" {
			format := resp.Format
			if format == "" {
				format = "mp3"
			}
			return toolText("data:audio/" + format + ";base64," + resp.Audio), nil
		}
	}
	return toolText(string(body)), nil
}

func (h *MCPHandler) toolTranscribe(ctx *fasthttp.RequestCtx, args json.RawMessage) (any, *rpcError) {
	if h.stt == nil {
		return toolError("transcription is not enabled on this server"), nil
	}
	var a struct {
		AudioBase64 string `json:"audio_base64"`
		Filename    string `json:"filename"`
		Model       string `json:"model"`
		Language    string `json:"language"`
		Prompt      string `json:"prompt"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, &rpcError{Code: -32602, Message: "Invalid arguments: " + err.Error()}
	}
	audio, err := base64.StdEncoding.DecodeString(a.AudioBase64)
	if err != nil {
		audio, err = base64.RawStdEncoding.DecodeString(a.AudioBase64)
	}
	if err != nil || len(audio) == 0 {
		return toolError("audio_base64 must be valid base64-encoded audio"), nil
	}
	filename := a.Filename
	if filename == "" {
		filename = "audio.mp3"
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, ferr := mw.CreateFormFile("file", filename)
	if ferr != nil {
		return toolError("failed to encode audio upload"), nil
	}
	if _, ferr := fw.Write(audio); ferr != nil {
		return toolError("failed to encode audio upload"), nil
	}
	for k, v := range map[string]string{"model": a.Model, "language": a.Language, "prompt": a.Prompt} {
		if v != "" {
			mw.WriteField(k, v)
		}
	}
	mw.Close()

	uri := "/v1/audio/transcriptions"
	if a.Model == "" {
		uri = "/v1/stt" // handler applies its default STT model on this path
	}
	status, body := h.dispatchRaw(ctx, h.stt.HandleTranscription, "POST", uri, mw.FormDataContentType(), buf.Bytes())
	if status != 200 {
		return toolError(upstreamMessage(body)), nil
	}
	var tr struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &tr); err == nil && tr.Text != "" {
		return toolText(tr.Text), nil
	}
	return toolText(string(body)), nil
}

func (h *MCPHandler) toolGenerate3D(ctx *fasthttp.RequestCtx, args json.RawMessage) (any, *rpcError) {
	if h.textTo3D == nil {
		return toolError("3D generation is not enabled on this server"), nil
	}
	var a struct {
		Prompt      string `json:"prompt"`
		ImageURL    string `json:"image_url"`
		TextureSize int    `json:"texture_size"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		return nil, &rpcError{Code: -32602, Message: "Invalid arguments: " + err.Error()}
	}
	if a.Prompt == "" && a.ImageURL == "" {
		return toolError("prompt or image_url is required"), nil
	}
	reqBody := model.TextTo3DGenerationRequest{Prompt: a.Prompt, ImageURL: a.ImageURL, TextureSize: a.TextureSize}
	status, body := h.dispatch(ctx, h.textTo3D.HandleTextTo3DGeneration, "/v1/3d/text-generations", reqBody)
	switch {
	case status == 200:
		var resp model.TextTo3DGenerationResponse
		if err := json.Unmarshal(body, &resp); err == nil && resp.ModelGLB.URL != "" {
			return toolText(resp.ModelGLB.URL), nil
		}
		return toolText(string(body)), nil
	case status == 202:
		return toolText(pendingJobText(body, "text3d")), nil
	default:
		return toolError(upstreamMessage(body)), nil
	}
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
	return h.dispatchRaw(parent, handler, "POST", uri, "application/json", body)
}

func (h *MCPHandler) dispatchRaw(parent *fasthttp.RequestCtx, handler fasthttp.RequestHandler, method, uri, contentType string, body []byte) (int, []byte) {
	var req fasthttp.Request
	req.Header.SetMethod(method)
	req.SetRequestURI(uri)
	if contentType != "" {
		req.Header.SetContentType(contentType)
	}
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

// classifyModality maps a model config to the MCP tool domain that serves it.
// Order matters: id markers beat pricing hints so e.g. pocket-tts (billed per
// minute like STT) still classifies as speech.
func classifyModality(id string, cfg *model.ModelConfig) string {
	idl := strings.ToLower(id)
	switch {
	case strings.Contains(idl, "embed"):
		return "embedding"
	case strings.Contains(idl, "whisper"), strings.Contains(idl, "stt"),
		strings.Contains(idl, "transcribe"), strings.Contains(idl, "transcription"):
		return "transcription"
	case strings.Contains(idl, "tts"), cfg.PricePer1MCharacters > 0:
		return "speech"
	case strings.Contains(idl, "music"), strings.Contains(idl, "sfx"):
		return "music"
	case strings.Contains(idl, "to-3d"), strings.Contains(idl, "3d"):
		return "3d"
	case cfg.PricePerVideo > 0, cfg.PricePerSecond > 0, cfg.PricePerSecondWithVideoInput > 0,
		len(cfg.PricePerSecondByResolution) > 0,
		strings.Contains(idl, "video"), strings.Contains(idl, "sora"), strings.Contains(idl, "veo"):
		return "video"
	case cfg.PricePerImage > 0, len(cfg.PricePerImageByResolution) > 0,
		cfg.PricePerMegapixel > 0, cfg.PriceFirstMegapixel > 0, cfg.PriceExtraMegapixel > 0,
		strings.Contains(idl, "image"):
		return "image"
	case strings.Contains(idl, "chronos"), strings.Contains(idl, "forecast"):
		return "forecasting"
	default:
		return "chat"
	}
}

// compactPricing returns only the non-zero rates so list_models stays compact
// across 250+ models.
func compactPricing(cfg *model.ModelConfig) map[string]any {
	inputRate, cacheRate, outputRate := cfg.TokenRatesAt(time.Now())
	p := make(map[string]any)
	add := func(key string, v float64) {
		if v > 0 {
			p[key] = v
		}
	}
	add("input_per_1m_tokens", inputRate)
	add("input_cache_hit_per_1m_tokens", cacheRate)
	add("output_per_1m_tokens", outputRate)
	add("per_1m_characters", cfg.PricePer1MCharacters)
	add("per_request", cfg.PricePerRequest)
	add("per_image", cfg.PricePerImage)
	add("per_megapixel", cfg.PricePerMegapixel)
	add("per_input_image", cfg.PricePerInputImage)
	add("per_video", cfg.PricePerVideo)
	add("per_second", cfg.PricePerSecond)
	add("per_minute", cfg.PricePerMinute)
	add("per_hour", cfg.PricePerHour)
	if len(cfg.PricePerImageByResolution) > 0 {
		p["per_image_by_resolution"] = cfg.PricePerImageByResolution
	}
	if len(cfg.PricePerSecondByResolution) > 0 {
		p["per_second_by_resolution"] = cfg.PricePerSecondByResolution
	}
	return p
}

// audioDataURI converts a provider audio payload (hex bytes, base64 or plain
// URL) into a single referenceable value for MCP text content.
func audioDataURI(payload string) string {
	if strings.HasPrefix(payload, "http://") || strings.HasPrefix(payload, "https://") {
		return payload
	}
	if raw, err := hex.DecodeString(payload); err == nil && len(raw) > 16 {
		format := "mpeg"
		if len(raw) > 4 && string(raw[:4]) == "RIFF" {
			format = "wav"
		}
		return "data:audio/" + format + ";base64," + base64.StdEncoding.EncodeToString(raw)
	}
	if _, err := base64.StdEncoding.DecodeString(payload); err == nil && len(payload) > 32 {
		return "data:audio/mpeg;base64," + payload
	}
	return ""
}

func pendingJobText(body []byte, kind string) string {
	var job struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &job); err != nil || job.ID == "" {
		return string(body)
	}
	return fmt.Sprintf("Job %s is %s. Poll it with the check_job tool using kind=%q and job_id=%q.", job.ID, job.Status, kind, job.ID)
}
