# OpenRouter Provider Integration Plan

## Goal
1. Make OpenPaths itself a provider on OpenRouter (expose the required endpoints)
2. Enhance discovery service to scrape OpenRouter-compatible `/v1/models` endpoints from all our providers

## Part 1: OpenRouter Provider Endpoint

### Required: List Models Endpoint
OpenRouter requires a public endpoint returning all models in their format.

**New endpoint**: `GET /openrouter/models` (public, no auth required)

Response format per OpenRouter spec:
```json
{
  "data": [{
    "id": "model-id",
    "name": "Display Name",
    "created": 1690502400,
    "input_modalities": ["text", "image"],
    "output_modalities": ["text"],
    "context_length": 200000,
    "max_output_length": 64000,
    "pricing": {
      "prompt": "0.000003",   // per token
      "completion": "0.000015", // per token
      "image": "0",
      "request": "0",
      "input_cache_read": "0"
    },
    "supported_sampling_parameters": ["temperature", "top_p", ...],
    "supported_features": ["tools", "json_mode", ...],
    "description": "...",
    "datacenters": [{"country_code": "US"}]
  }]
}
```

### Implementation
1. New handler: `internal/handler/openrouter_provider.go`
   - `OpenRouterProviderHandler` with `HandleListModels`
   - Reads from router's model list, converts to OpenRouter format
   - Converts pricing from per-1M-tokens to per-token (divide by 1M)
   - Maps capabilities to OpenRouter features/sampling params

2. New route in `server.go`:
   - `GET /openrouter/models` on publicChain (no auth)

### Chat completions already work
OpenPaths already exposes `/v1/chat/completions` in OpenAI-compatible format, which is what OpenRouter calls. OpenRouter will authenticate with an API key.

## Part 2: Enhanced Provider Discovery

### Current state
Discovery service already scrapes:
- mistral, openai, openrouter, together, groq, xai, deepseek

### Enhancements
Add discovery for providers we're missing:
- **Google** (Gemini) - uses different API format
- **Anthropic** - uses `/v1/models`
- **MiniMax** - check for `/v1/models`
- **Nous Research** - OpenAI-compatible
- **Z.AI** - check for `/v1/models`
- **FAL** - custom API

Also enhance existing OpenRouter discovery to extract richer metadata (features, modalities, pricing per token).

### Implementation
Extend `internal/discovery/discovery.go`:
- Add `discoverAnthropicModels()` - Anthropic has a models list endpoint
- Add `discoverGoogle()` - Google has a models list endpoint
- Add `discoverNous()` - OpenAI compatible
- Add `discoverMiniMax()` - Check their API
- Add `discoverZAI()` - Check their API
- Enhance `discoverOpenRouter()` with richer metadata parsing
- Add `discoverFal()` - Check their API

### Enriched metadata
For OpenRouter discovery, parse additional fields:
- `top_provider` for reliability
- `per_request_limits`
- Architecture info (tokenizer, modality)
- Pricing per token
