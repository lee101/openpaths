package google

import (
	"context"
	"strings"
	"testing"
	"time"
)

func largePrefixReq() *geminiRequest {
	return &geminiRequest{
		SystemInstruction: &geminiContent{
			Parts: []geminiPart{{Text: strings.Repeat("system context ", 1000)}}, // ~15k chars
		},
	}
}

func TestGeminiCacheDisabledIsNoOp(t *testing.T) {
	m := NewGeminiCacheManager(false, "key", "")
	name, key := m.Ensure(context.Background(), "gemini-2.5-pro", largePrefixReq())
	if name != "" || key != "" {
		t.Errorf("disabled manager should no-op, got name=%q key=%q", name, key)
	}
}

func TestGeminiCachePrefixKeyStable(t *testing.T) {
	r1 := largePrefixReq()
	r2 := largePrefixReq()
	if cachePrefixKey("gemini-2.5-pro", r1) != cachePrefixKey("gemini-2.5-pro", r2) {
		t.Errorf("identical prefixes should hash equal")
	}
	if cachePrefixKey("gemini-2.5-pro", r1) == cachePrefixKey("gemini-2.5-flash", r1) {
		t.Errorf("different models should hash differently")
	}
}

func TestGeminiShouldCreateGating(t *testing.T) {
	base := time.Unix(1_700_000_000, 0)
	m := NewGeminiCacheManager(true, "key", "")
	m.now = func() time.Time { return base }

	// Small prefix never qualifies regardless of hits.
	small := &geminiRequest{SystemInstruction: &geminiContent{Parts: []geminiPart{{Text: "hi"}}}}
	smallKey := cachePrefixKey("gemini-2.5-pro", small)
	for i := 0; i < 5; i++ {
		m.stats[smallKey] = append(m.stats[smallKey], base.UnixNano())
	}
	if m.shouldCreateLocked(smallKey, small, base.UnixNano()) {
		t.Errorf("small prefix should not be cached")
	}

	// Large prefix: below minHits -> no; at minHits -> yes.
	large := largePrefixReq()
	largeKey := cachePrefixKey("gemini-2.5-pro", large)
	m.stats[largeKey] = []int64{base.UnixNano(), base.UnixNano()} // 2 < minHits(3)
	if m.shouldCreateLocked(largeKey, large, base.UnixNano()) {
		t.Errorf("large prefix below minHits should not be cached yet")
	}
	m.stats[largeKey] = append(m.stats[largeKey], base.UnixNano()) // now 3
	if !m.shouldCreateLocked(largeKey, large, base.UnixNano()) {
		t.Errorf("large prefix at minHits should be cached")
	}
}

func TestGeminiInvalidate(t *testing.T) {
	m := NewGeminiCacheManager(true, "key", "")
	m.caches["k"] = cacheEntry{name: "cachedContents/x", expiresAt: 1 << 62}
	m.Invalidate("k")
	if _, ok := m.caches["k"]; ok {
		t.Errorf("Invalidate should remove the cache entry")
	}
}
