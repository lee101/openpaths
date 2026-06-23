// Package promptcache implements Anthropic prompt-cache cost optimization: it
// tracks a rolling window of request timing per stable prefix and decides, every
// minute, whether the cheapest cache breakpoint for that prefix is none, a 5m
// ephemeral cache, or a 1h ephemeral cache. It is Anthropic-only.
package promptcache

import "strings"

// ModelCost holds Anthropic per-MTok prices used for the cache TTL cost
// comparison. All current Anthropic models share the same multipliers relative
// to base input (5m write = 1.25x, 1h write = 2x, cache hit/refresh = 0.1x), but
// we store explicit prices so the optimizer stays correct if pricing ever
// diverges and so realized savings can be reported in real dollars.
type ModelCost struct {
	BaseInputPer1M float64 // uncached input
	Write5mPer1M   float64 // 5-minute cache write
	Write1hPer1M   float64 // 1-hour cache write
	CacheHitPer1M  float64 // cache hit / refresh
	OutputPer1M    float64 // output
}

// DefaultModel is the fallback used for unknown Anthropic models (latest Opus).
const DefaultModel = "claude-opus-4-8"

// costTable mirrors the published Anthropic prompt-caching pricing table.
var costTable = map[string]ModelCost{
	"claude-opus-4-8":   {BaseInputPer1M: 5, Write5mPer1M: 6.25, Write1hPer1M: 10, CacheHitPer1M: 0.50, OutputPer1M: 25},
	"claude-opus-4-7":   {BaseInputPer1M: 5, Write5mPer1M: 6.25, Write1hPer1M: 10, CacheHitPer1M: 0.50, OutputPer1M: 25},
	"claude-opus-4-6":   {BaseInputPer1M: 5, Write5mPer1M: 6.25, Write1hPer1M: 10, CacheHitPer1M: 0.50, OutputPer1M: 25},
	"claude-opus-4-5":   {BaseInputPer1M: 5, Write5mPer1M: 6.25, Write1hPer1M: 10, CacheHitPer1M: 0.50, OutputPer1M: 25},
	"claude-opus-4-1":   {BaseInputPer1M: 15, Write5mPer1M: 18.75, Write1hPer1M: 30, CacheHitPer1M: 1.50, OutputPer1M: 75},
	"claude-opus-4":     {BaseInputPer1M: 15, Write5mPer1M: 18.75, Write1hPer1M: 30, CacheHitPer1M: 1.50, OutputPer1M: 75},
	"claude-sonnet-4-6": {BaseInputPer1M: 3, Write5mPer1M: 3.75, Write1hPer1M: 6, CacheHitPer1M: 0.30, OutputPer1M: 15},
	"claude-sonnet-4-5": {BaseInputPer1M: 3, Write5mPer1M: 3.75, Write1hPer1M: 6, CacheHitPer1M: 0.30, OutputPer1M: 15},
	"claude-sonnet-4":   {BaseInputPer1M: 3, Write5mPer1M: 3.75, Write1hPer1M: 6, CacheHitPer1M: 0.30, OutputPer1M: 15},
	"claude-haiku-4-5":  {BaseInputPer1M: 1, Write5mPer1M: 1.25, Write1hPer1M: 2, CacheHitPer1M: 0.10, OutputPer1M: 5},
	"claude-haiku-3-5":  {BaseInputPer1M: 0.80, Write5mPer1M: 1, Write1hPer1M: 1.60, CacheHitPer1M: 0.08, OutputPer1M: 4},
}

// CostFor returns the cost row for an Anthropic model. It matches by exact id
// first, then by the longest known prefix (so dated ids like
// "claude-opus-4-8-20260101" resolve), and finally falls back to DefaultModel.
func CostFor(model string) ModelCost {
	m := normalize(model)
	if c, ok := costTable[m]; ok {
		return c
	}
	var best string
	for k := range costTable {
		if strings.HasPrefix(m, k) && len(k) > len(best) {
			best = k
		}
	}
	if best != "" {
		return costTable[best]
	}
	return costTable[DefaultModel]
}

func normalize(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	// strip a routing prefix like "anthropic/claude-..."
	if i := strings.LastIndex(m, "/"); i >= 0 {
		m = m[i+1:]
	}
	return m
}
