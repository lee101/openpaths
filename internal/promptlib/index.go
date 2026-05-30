package promptlib

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

// Result is a prompt plus its similarity score for a query.
type Result struct {
	Prompt
	Score float64 `json:"score"`
}

// Status reports the health of the semantic index.
type Status struct {
	Enabled   bool   `json:"enabled"`
	Ready     bool   `json:"ready"`
	Indexing  bool   `json:"indexing"`
	Error     string `json:"error,omitempty"`
	Items     int    `json:"items"`
	IndexedAt string `json:"indexedAt,omitempty"`
}

// Index embeds the prompt catalog with a local embedder (gobed) and serves
// cosine-similarity search and related-prompt lookups. It mirrors
// internal/artindex.Service but over the in-process prompt catalog.
//
// When the embedder is nil or the index is not yet ready, callers fall back to
// the lexical/metadata helpers in data.go (Filter / RelatedMeta).
type Index struct {
	embedder provider.EmbeddingProvider

	mu        sync.RWMutex
	ready     bool
	indexing  bool
	lastErr   string
	indexedAt time.Time
	prompts   []Prompt
	vectors   [][]float32
	bySlug    map[string]int // slug -> position in prompts/vectors
}

// New builds an Index over the full prompt catalog using the given embedder.
func New(embedder provider.EmbeddingProvider) *Index {
	return &Index{embedder: embedder}
}

// Start kicks off embedding in the background. Safe to call with a nil index.
func (ix *Index) Start(ctx context.Context) {
	if ix == nil || ix.embedder == nil {
		return
	}
	go ix.Rebuild(ctx)
}

// Status returns the current index status.
func (ix *Index) Status() Status {
	if ix == nil {
		return Status{}
	}
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	s := Status{
		Enabled:  ix.embedder != nil,
		Ready:    ix.ready,
		Indexing: ix.indexing,
		Error:    ix.lastErr,
		Items:    len(ix.prompts),
	}
	if !ix.indexedAt.IsZero() {
		s.IndexedAt = ix.indexedAt.UTC().Format(time.RFC3339)
	}
	return s
}

// Ready reports whether the semantic index can serve queries.
func (ix *Index) Ready() bool {
	if ix == nil {
		return false
	}
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	return ix.ready
}

// Rebuild embeds the entire catalog. It is idempotent and concurrency-safe.
func (ix *Index) Rebuild(ctx context.Context) {
	ix.mu.Lock()
	if ix.indexing {
		ix.mu.Unlock()
		return
	}
	ix.indexing = true
	ix.lastErr = ""
	ix.mu.Unlock()

	defer func() {
		ix.mu.Lock()
		ix.indexing = false
		ix.mu.Unlock()
	}()

	if ix.embedder == nil {
		ix.setError("no embedding provider configured")
		return
	}

	prompts := All()
	if len(prompts) == 0 {
		ix.setError("prompt catalog is empty")
		return
	}

	start := time.Now()
	vectors, err := embedSearchText(ctx, ix.embedder, prompts, 128)
	if err != nil {
		ix.setError(err.Error())
		return
	}

	bySlug := make(map[string]int, len(prompts))
	for i, p := range prompts {
		bySlug[p.Slug] = i
	}

	ix.mu.Lock()
	ix.prompts = prompts
	ix.vectors = vectors
	ix.bySlug = bySlug
	ix.ready = true
	ix.indexedAt = time.Now()
	ix.lastErr = ""
	ix.mu.Unlock()

	log.Printf("prompt index: embedded %d prompts in %s", len(prompts), time.Since(start).Round(time.Millisecond))
}

// Search returns prompts ranked by semantic similarity to the query, after
// applying the given filters.
func (ix *Index) Search(ctx context.Context, query string, f Filters, limit int) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query required")
	}
	if limit <= 0 || limit > 200 {
		limit = 48
	}

	ix.mu.RLock()
	ready := ix.ready
	prompts := append([]Prompt(nil), ix.prompts...)
	vectors := append([][]float32(nil), ix.vectors...)
	ix.mu.RUnlock()

	if !ready {
		return nil, fmt.Errorf("prompt index is not ready")
	}

	resp, err := ix.embedder.Embed(ctx, &model.EmbeddingRequest{
		Model:        "gobed",
		Input:        query,
		LongTextMode: "truncate",
	})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("embed query: no embedding returned")
	}
	q := normalizeFloat64(resp.Data[0].Embedding)

	results := make([]Result, 0, len(prompts))
	for i, p := range prompts {
		if i >= len(vectors) {
			break
		}
		if !f.matches(p) {
			continue
		}
		results = append(results, Result{Prompt: p, Score: float64(dot(q, vectors[i]))})
	}
	sort.SliceStable(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// Related returns prompts most similar to the given prompt by embedding cosine
// similarity. Falls back to RelatedMeta when the index is not ready.
func (ix *Index) Related(slug string, limit int) []Prompt {
	if limit <= 0 {
		limit = 4
	}
	if ix == nil {
		return RelatedMeta(slug, limit)
	}
	ix.mu.RLock()
	ready := ix.ready
	pos, ok := ix.bySlug[slug]
	prompts := ix.prompts
	vectors := ix.vectors
	ix.mu.RUnlock()

	if !ready || !ok || pos >= len(vectors) {
		return RelatedMeta(slug, limit)
	}

	base := vectors[pos]
	type scored struct {
		p     Prompt
		score float32
	}
	ranked := make([]scored, 0, len(prompts))
	for i, p := range prompts {
		if i == pos || i >= len(vectors) {
			continue
		}
		ranked = append(ranked, scored{p, dot(base, vectors[i])})
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]Prompt, len(ranked))
	for i, r := range ranked {
		out[i] = r.p
	}
	return out
}

func (ix *Index) setError(msg string) {
	log.Printf("prompt index: %s", msg)
	ix.mu.Lock()
	ix.lastErr = msg
	ix.ready = false
	ix.mu.Unlock()
}

func embedSearchText(ctx context.Context, embedder provider.EmbeddingProvider, prompts []Prompt, batchSize int) ([][]float32, error) {
	if batchSize <= 0 {
		batchSize = 128
	}
	vectors := make([][]float32, len(prompts))
	for start := 0; start < len(prompts); start += batchSize {
		end := start + batchSize
		if end > len(prompts) {
			end = len(prompts)
		}
		inputs := make([]string, end-start)
		for i := start; i < end; i++ {
			inputs[i-start] = prompts[i].SearchText
		}
		resp, err := embedder.Embed(ctx, &model.EmbeddingRequest{
			Model:        "gobed",
			Input:        inputs,
			LongTextMode: "truncate",
		})
		if err != nil {
			return nil, fmt.Errorf("embed prompts %d-%d: %w", start, end, err)
		}
		if len(resp.Data) != len(inputs) {
			return nil, fmt.Errorf("embed prompts %d-%d: got %d embeddings for %d inputs", start, end, len(resp.Data), len(inputs))
		}
		for _, data := range resp.Data {
			if data.Index < 0 || data.Index >= len(inputs) {
				return nil, fmt.Errorf("embed prompts %d-%d: invalid embedding index %d", start, end, data.Index)
			}
			vectors[start+data.Index] = normalizeFloat64(data.Embedding)
		}
	}
	return vectors, nil
}

func normalizeFloat64(values []float64) []float32 {
	out := make([]float32, len(values))
	var sum float64
	for _, v := range values {
		sum += v * v
	}
	if sum == 0 {
		return out
	}
	inv := 1 / math.Sqrt(sum)
	for i, v := range values {
		out[i] = float32(v * inv)
	}
	return out
}

func dot(a, b []float32) float32 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var out float32
	for i := 0; i < n; i++ {
		out += a[i] * b[i]
	}
	return out
}
