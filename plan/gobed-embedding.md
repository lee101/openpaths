# Gobed Embedding - First-Party Embedding Model

## Overview
OpenPaths.io offers a first-party embedding model powered by gobed (static-retrieval-mrl-en-v1).
Local GPU inference at ~1ms per embedding makes this extremely cheap to operate.

## Model Details
- Model: static-retrieval-mrl-en-v1 (int8 quantized, 512 dimensions)
- Latency: ~0.15ms per embedding on CPU, faster on GPU
- Memory: ~15MB model size
- Provider name: `gobed`
- Model ID: `openpaths-embed`
- Aliases: `gobed`, `gobed-embed`, `op-embed`

## Pricing
- Input: $0.001 per 1M tokens ($1 per billion), output $0
- `sub_unit_billing: true` in config.yaml: cost accrues per user+model in memory (billing.Engine.accrue), whole units ($0.0001) are charged once the carry crosses 1. Before this every request rounded up to the 1-unit minimum, so $1/B was unrepresentable (a 256-token request costs 0.0000026 units... 1 unit = 100k tokens)
- A restart forgives under 1 unit per user (customer-favorable)
- Compute cost: ~5M tokens/s per core measured (below), so 1B tokens is ~200 core-seconds; at $1/B revenue the margin is dominated by HTTP/auth/DB per-request overhead, not embedding

## Implementation
- `internal/provider/gobed/gobed.go` - EmbeddingProvider wrapping gobed's SimpleInt8Model512
- Loaded at startup before other providers, becomes first embedder in fallback chain
- Also used by AutoRouter for model selection (free internal embedding)
- No API key needed - runs in-process

## Files Changed
- `internal/provider/gobed/gobed.go` - New provider
- `cmd/openpaths/main.go` - Gobed init + import
- `config.yaml` - Model config entry
- `go.mod` - Added github.com/lee101/gobed dependency

## Optimization (2026-10)
- `internal/provider/gobed/tokenizer.go`: allocation-free ASCII WordPiece (parity-tested vs gobed SimpleTokenize on 20k random strings; non-ASCII falls back to the lib path; words over 100 bytes become [UNK], BERT-style, removing the O(L^2) blowup)
- Early exit at 256 tokens in truncate mode (huge inputs no longer tokenized fully)
- AVX2 accumulation straight from the model table (`simd.EmbedAccum`), pooled token/accumulator scratch
- Batches split across goroutines (8 inputs/worker grain, max GOMAXPROCS)
- `dimensions` (matryoshka, 1-512, truncated + L2-renormalized) and `encoding_format=base64` supported; `model.EmbeddingData.MarshalJSON` emits float32-shortest floats (about half the bytes of float64 formatting)
- Input cap 4 MiB per string

Bench (Ryzen 9 5950X, ~70-token input), `GOBED_MODEL_DIR=<dir containing model/> go test ./internal/provider/gobed -bench .`:

| bench | before | after |
|---|---|---|
| single | 91.1 us, 85 allocs | 14.1 us, 6 allocs |
| batch 64 | 11.26 ms | 0.80 ms |
| parallel (32 thr) | 12.5 us/op | 2.5 us/op |

Remaining per-request cost is outside the provider: PreCheck balance lookup, usage recording, JSON, auth.

## Overflow (RunPod) design
Static embeddings are table lookup + mean: no GPU needed. A CPU pod (or the same Go binary as `cmd/embedworker` exposing /v1/embeddings behind a shared key) beats a GPU on cost; only consider a small GPU for >100k-token batches.
- Tier 0: in-process gobed (default)
- Tier 1 overflow: when in-flight embed requests > N (or p95 > 25 ms) for 10 s, route a fraction to a remote `gobed-remote` provider (EmbeddingProvider over HTTP, same fallback chain slot after gobed)
- Tier 2: sustained overflow for 15 min -> start a RunPod CPU pod via existing provisioning/ scripts; scale down after 30 idle min
- Billing unchanged (billed at the gateway on usage tokens)
- Status: not implemented

## End-to-end hot path (2026-10)
- billing.Engine: sub-unit models skip the DB balance read when a cached balance (3s TTL, >= 1000 units) exists; cache is decremented on charge
- Charges only hit the DB when a user's carry crosses 1 unit (about 100k tokens at $1/B)
- queries caches (internal/ttlcache): API key validation 10s (last_used_at write throttled to 1/min), "no BYOK keys" 10s, guardrail policy resolve 5s; invalidated on revoke/upsert/guardrail edits. Disabling a user takes up to 10s to take effect
- middleware.requestModel stops decoding at the `model` key
- /v1/embeddings has its own rate bucket with limits x50 (EMBED_RATE_LIMIT_MULT)
- model.EmbeddingResponse.AppendJSON writes the response directly
- metrics.Collector buffer 4096 -> 65536, batch 256 -> 1024
- gobed lib (~/code/gobed, uncommitted): default alloc-free tokenizer, drops unknown tokens, TokenizeInto/EmbedTextInto; Embed 96us -> 10.7us. Provider still uses its local tokenizer until a gobed tag is cut
- TODO: per-request usage_logs rows still 1/request (needs request_count column + coalescing), overflow tier, HTTP e2e benchmark
