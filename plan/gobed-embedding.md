# Gobed Embedding - First-Party Embedding Model

## Overview
OpenPaths.io offers a first-party embedding model powered by gobed (static-retrieval-mrl-en-v1).
Local GPU inference at ~1ms per embedding makes this extremely cheap to operate.

## Model Details
- Model: static-retrieval-mrl-en-v1 (int8 quantized, 512 dimensions)
- Latency: ~0.15ms per embedding on CPU, faster on GPU
- Memory: ~15MB model size
- Provider name: `gobed`
- Model ID: `openpath-embed`
- Aliases: `gobed`, `gobed-embed`, `op-embed`

## Pricing
- Input: $0.002 per 1M tokens (50x cheaper than OpenAI text-embedding-3-small)
- Output: $0 (embeddings have no output tokens)
- Rationale: near-zero compute cost, just covering infrastructure

## Implementation
- `internal/provider/gobed/gobed.go` - EmbeddingProvider wrapping gobed's SimpleInt8Model512
- Loaded at startup before other providers, becomes first embedder in fallback chain
- Also used by AutoRouter for model selection (free internal embedding)
- No API key needed - runs in-process

## Files Changed
- `internal/provider/gobed/gobed.go` - New provider
- `cmd/openpath/main.go` - Gobed init + import
- `config.yaml` - Model config entry
- `go.mod` - Added github.com/lee101/gobed dependency
