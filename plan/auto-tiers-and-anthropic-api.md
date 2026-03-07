# Auto Task Tiers + Anthropic API Compatibility

## 1. auto-easy-task & auto-medium-task Models

New auto-routing tiers that constrain model selection to specific cost bands.

### auto-easy-task
- Maps to `easy-task` modality in autorouter
- Only routes to cheapest/fastest models: gemini-flash-lite, minimax-m2.5-highspeed, gpt-4o-mini, llama-8b
- Use case: simple lookups, formatting, summarization, basic Q&A
- Primary model in config: gemini-flash-lite (cheapest)
- Fallbacks: minimax-m2.5-highspeed, gpt-4o-mini, llama-8b

### auto-medium-task
- Maps to `medium-task` modality in autorouter
- Routes to mid-tier models: claude-sonnet-4-6, gemini-2.5-flash, deepseek-chat, gpt-4o, minimax-m2.5
- Use case: coding tasks, analysis, moderate complexity
- Primary model in config: claude-sonnet-4-6 (strong all-rounder)
- Fallbacks: gemini-2.5-flash, deepseek-chat, minimax-m2.5, gpt-4o

### Files to modify:
- `internal/router/autorouter.go` - add modalities + routing tables + autoModelMap entries
- `config.yaml` - add auto-easy-task and auto-medium-task model entries
- `src/pages/Playground.tsx` - add to frontend model list
- `src/data/models.ts` - add metadata

## 2. MiniMax Anthropic-Compatible API Support

MiniMax supports Anthropic SDK format at `https://api.minimax.io/anthropic`.
Current MiniMax provider uses OpenAI-compatible API. Add a secondary `minimax-anthropic`
provider that uses the Anthropic-compatible endpoint for models that support it.

Actually - since MiniMax already works fine via OpenAI compat, this is lower priority.
The main value is for the Anthropic API endpoint on OpenPaths (item 3).

## 3. Anthropic API Endpoint on OpenPaths (/v1/messages)

Allow users to point their Anthropic SDK at openpaths.io:
```python
client = anthropic.Anthropic(base_url="https://openpaths.io", api_key="op-...")
```

### New handler: `internal/handler/anthropic.go`
- POST /v1/messages - accepts Anthropic message format
- Translates Anthropic request -> internal ChatCompletionRequest
- Routes through existing router (supports auto, fallbacks, billing)
- Translates response back to Anthropic format
- Supports streaming with Anthropic SSE event format

### Auth
- Accept both `x-api-key` header (Anthropic style) and `Authorization: Bearer` (standard)
- Middleware already handles Bearer; add x-api-key fallback

### Request translation: Anthropic -> Internal
- model -> model (pass through, routes via our router)
- messages -> messages (Anthropic format is close to internal)
- system -> prepend as system message
- max_tokens -> max_tokens
- tools -> tools (translate input_schema -> parameters)
- temperature, top_p -> pass through
- stream -> stream

### Response translation: Internal -> Anthropic
- Non-streaming: wrap in Anthropic response envelope
- Streaming: emit Anthropic SSE events (message_start, content_block_start, content_block_delta, message_delta, message_stop)

### Files to create:
- `internal/handler/anthropic.go`

### Files to modify:
- `internal/server/server.go` - register /v1/messages route
- `internal/middleware/auth.go` - add x-api-key header support
