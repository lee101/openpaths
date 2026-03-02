# Sentence Compaction via Embeddings

## Problem
When endpoints error or token count estimates show content won't fit a model's context window, we need to intelligently shed sentences while preserving the most semantically relevant content.

## Algorithm
1. Split text into sentences (period/newline/semicolon boundaries)
2. Batch-embed all sentences on GPU using gobed
3. Compute reference embedding (either from a query string, or mean of all sentence embeddings)
4. Rank sentences by cosine similarity to reference
5. Greedily select top-N sentences such that total estimated tokens <= budget
6. Re-order selected sentences by original position to maintain coherence
7. Return compacted text

## Token Estimation
Fast heuristic: `len(text) / 3.5` (avg English chars per token ~3.5 for most tokenizers). No need for exact counts - we're already approximating.

## Implementation (in gobed ~/code/gobed)

### New Files
- `compact.go` - Core compaction API
  - `CompactConfig` - settings (max tokens, query, preserve order)
  - `SentenceSplit(text) []string` - fast sentence splitter
  - `EstimateTokens(text) int` - fast token count heuristic
  - `Compact(model, text, maxTokens, query) (string, error)` - main entry point
  - `CompactSentences(model, sentences, maxTokens, query) ([]string, error)` - granular API

- `compact_test.go` - comprehensive tests
  - Sentence splitting edge cases
  - Token estimation accuracy
  - Compaction preserves order
  - Compaction respects token budget
  - Empty/single sentence handling
  - GPU batch path validation

### Integration Points
- Uses `EmbedInt8` for fast sentence embeddings (already int8 quantized)
- Uses SIMD `Cosine512` for similarity computation
- Batch processing leverages existing GPU pipeline when available
- Zero-allocation paths where possible using existing buffer pools

## Usage from OpenPath
When a provider returns a context-length error or pre-request token estimation exceeds model's `ContextWindow`:
1. Extract message content
2. Call `gobed.Compact(model, content, remainingTokenBudget, userQuery)`
3. Replace message content with compacted version
4. Retry the request
