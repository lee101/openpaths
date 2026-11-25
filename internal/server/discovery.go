package server

import (
	"encoding/json"
	"strings"

	fasthttprouter "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

const siteBase = "https://openpaths.io"

var discoveryLinkHeader = `</.well-known/api-catalog>; rel="api-catalog", </openapi.json>; rel="service-desc"; type="application/json", </docs>; rel="service-doc", </llms.txt>; rel="describedby"; type="text/plain"`

func bearerSecurity() []map[string][]string {
	return []map[string][]string{{"bearerAuth": {}}}
}

func jsonBody(desc, schemaRef string) map[string]any {
	return map[string]any{
		"required": true,
		"content":  map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + schemaRef}}},
	}
}

func okResp(desc string) map[string]any {
	return map[string]any{
		"200": map[string]any{"description": desc, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object"}}}},
		"401": map[string]any{"description": "Missing or invalid API key"},
		"403": map[string]any{"description": "insufficient_scope"},
		"429": map[string]any{"description": "Rate limited"},
	}
}

func buildOpenAPI() []byte {
	obj := func(props map[string]any, required ...string) map[string]any {
		s := map[string]any{"type": "object", "properties": props}
		if len(required) > 0 {
			s["required"] = required
		}
		return s
	}
	str := map[string]any{"type": "string"}
	doc := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "OpenPaths API",
			"version":     "1.0.0",
			"summary":     "OpenAI- and Anthropic-compatible gateway for language, image, video, speech, music, embedding and 3D models.",
			"description": "One API key and base URL for many providers with smart routing (openpaths/auto), automatic fallbacks, BYOK and near-zero markup. Model ids and prices: GET /v1/models, https://openpaths.io/llms-full.txt and https://openpaths.io/models.",
			"contact":     map[string]any{"name": "OpenPaths", "url": siteBase + "/docs"},
		},
		"servers":      []map[string]any{{"url": siteBase, "description": "Production"}},
		"externalDocs": map[string]any{"description": "API documentation", "url": siteBase + "/docs"},
		"security":     bearerSecurity(),
		"paths": map[string]any{
			"/v1/chat/completions": map[string]any{"post": map[string]any{
				"operationId": "createChatCompletion", "summary": "Create a chat completion (OpenAI schema)", "tags": []string{"Chat"},
				"requestBody": jsonBody("chat", "ChatCompletionRequest"), "responses": okResp("Chat completion or SSE stream"),
			}},
			"/v1/messages": map[string]any{"post": map[string]any{
				"operationId": "createMessage", "summary": "Create a message (Anthropic schema)", "tags": []string{"Chat"},
				"requestBody": jsonBody("messages", "MessagesRequest"), "responses": okResp("Message or SSE stream"),
			}},
			"/v1/embeddings": map[string]any{"post": map[string]any{
				"operationId": "createEmbedding", "summary": "Create embeddings", "tags": []string{"Embeddings"},
				"requestBody": jsonBody("embeddings", "EmbeddingRequest"), "responses": okResp("Embedding vectors"),
			}},
			"/v1/images/generations": map[string]any{"post": map[string]any{
				"operationId": "createImage", "summary": "Generate images", "tags": []string{"Images"},
				"requestBody": jsonBody("images", "ImageRequest"), "responses": okResp("Generated images"),
			}},
			"/v1/models": map[string]any{"get": map[string]any{
				"operationId": "listModels", "summary": "List models with pricing", "tags": []string{"Models"}, "responses": okResp("Model list"),
			}},
			"/v1/models/{model_id}": map[string]any{"get": map[string]any{
				"operationId": "getModel", "summary": "Get one model", "tags": []string{"Models"},
				"parameters": []map[string]any{{"name": "model_id", "in": "path", "required": true, "schema": str}},
				"responses":  okResp("Model"),
			}},
			"/mcp": map[string]any{"post": map[string]any{
				"operationId": "mcpJsonRpc", "summary": "Model Context Protocol JSON-RPC endpoint", "tags": []string{"MCP"},
				"requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object"}}}},
				"responses":   okResp("JSON-RPC response"),
			}},
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{"bearerAuth": map[string]any{"type": "http", "scheme": "bearer", "description": "Authorization: Bearer <key>. X-API-Key is also accepted."}},
			"schemas": map[string]any{
				"ChatCompletionRequest": obj(map[string]any{
					"model":    map[string]any{"type": "string", "description": "Model id, alias or openpaths/auto"},
					"messages": map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
					"stream":   map[string]any{"type": "boolean"},
				}, "model", "messages"),
				"MessagesRequest": obj(map[string]any{
					"model":      str,
					"max_tokens": map[string]any{"type": "integer"},
					"messages":   map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
				}, "model", "messages"),
				"EmbeddingRequest": obj(map[string]any{"model": str, "input": map[string]any{"oneOf": []map[string]any{{"type": "string"}, {"type": "array", "items": str}}}}, "model", "input"),
				"ImageRequest":     obj(map[string]any{"model": str, "prompt": str, "size": str, "n": map[string]any{"type": "integer"}}, "model", "prompt"),
			},
		},
	}
	b, _ := json.MarshalIndent(doc, "", " ")
	return b
}

func buildAPICatalog() []byte {
	doc := map[string]any{"linkset": []map[string]any{{
		"anchor":       siteBase + "/v1",
		"service-desc": []map[string]any{{"href": siteBase + "/openapi.json", "type": "application/json"}},
		"service-doc":  []map[string]any{{"href": siteBase + "/docs", "type": "text/html"}},
		"describedby":  []map[string]any{{"href": siteBase + "/llms.txt", "type": "text/plain"}, {"href": siteBase + "/llms-full.txt", "type": "text/plain"}},
		"status":       []map[string]any{{"href": siteBase + "/status", "type": "text/html"}},
	}}}
	b, _ := json.MarshalIndent(doc, "", " ")
	return b
}

func buildMCPCard() []byte {
	doc := map[string]any{
		"name":        "OpenPaths",
		"description": "Every OpenPaths model over the Model Context Protocol.",
		"endpoint":    siteBase + "/mcp",
		"transport":   "http",
		"auth":        map[string]any{"type": "bearer", "docs": siteBase + "/mcp"},
		"docs":        siteBase + "/mcp",
	}
	b, _ := json.MarshalIndent(doc, "", " ")
	return b
}

func registerDiscoveryRoutes(r *fasthttprouter.Router, llms func() string) {
	serve := func(ct, cc string, body func() []byte) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			ctx.SetContentType(ct)
			ctx.Response.Header.Set("Cache-Control", cc)
			ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
			ctx.SetStatusCode(200)
			ctx.SetBody(body())
		}
	}
	oa := buildOpenAPI()
	cat := buildAPICatalog()
	card := buildMCPCard()
	spec := serve("application/json; charset=utf-8", "public, max-age=3600", func() []byte { return oa })
	r.GET("/openapi.json", spec)
	r.GET("/.well-known/openapi.json", spec)
	r.GET("/.well-known/api-catalog", serve("application/linkset+json", "public, max-age=3600", func() []byte { return cat }))
	r.GET("/.well-known/mcp.json", serve("application/json; charset=utf-8", "public, max-age=3600", func() []byte { return card }))
	r.GET("/.well-known/llms.txt", func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("text/plain; charset=utf-8")
		ctx.Response.Header.Set("Cache-Control", "public, max-age=3600")
		ctx.SetStatusCode(200)
		ctx.SetBodyString(llms())
	})
}

func isDiscoveryPath(path string) bool {
	return path == "/openapi.json" || len(path) > 13 && path[:13] == "/.well-known/"
}

func staticCacheControl(path string, status int) string {
	if status != fasthttp.StatusOK || strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/auth/") || strings.HasPrefix(path, "/admin/") || strings.HasPrefix(path, "/stats/") || strings.HasPrefix(path, "/account/") {
		return ""
	}
	switch {
	case hasSuffixAny(path, ".woff2", ".woff", ".ttf"):
		return "public, max-age=2592000, immutable"
	case hasSuffixAny(path, ".webp", ".png", ".jpg", ".jpeg", ".svg", ".ico", ".gif", ".avif", ".mp4", ".webm", ".mp3"):
		return "public, max-age=86400, stale-while-revalidate=604800"
	case hasSuffixAny(path, ".js", ".css", ".json", ".txt", ".xml"):
		return "public, max-age=300, stale-while-revalidate=86400"
	}
	return ""
}

func hasSuffixAny(s string, suf ...string) bool {
	for _, x := range suf {
		if len(s) >= len(x) && s[len(s)-len(x):] == x {
			return true
		}
	}
	return false
}
