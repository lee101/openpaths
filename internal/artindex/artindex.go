package artindex

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

const DefaultManifestURL = "https://openpathsstatic.openpaths.io/static/data/zimage-art/manifest.json"

// Item is a single indexed image. Aliased to model.ArtImage so the DB layer,
// the manifest JSON, and the /v1/art API all share one shape.
type Item = model.ArtImage

type Manifest struct {
	Version        int     `json:"version"`
	Kind           string  `json:"kind"`
	Model          string  `json:"model"`
	Count          int     `json:"count"`
	GeneratedCount int     `json:"generatedCount"`
	PublicBase     string  `json:"publicBase"`
	Chunks         []Chunk `json:"chunks"`
	GeneratedAt    string  `json:"generatedAt,omitempty"`
}

type Chunk struct {
	Path  string `json:"path"`
	URL   string `json:"url,omitempty"`
	Count int    `json:"count"`
}

type Result struct {
	Item
	Score float64 `json:"score"`
}

type Status struct {
	Enabled   bool   `json:"enabled"`
	Ready     bool   `json:"ready"`
	Indexing  bool   `json:"indexing"`
	Error     string `json:"error,omitempty"`
	Items     int    `json:"items"`
	Source    string `json:"source,omitempty"` // "db" | "manifest"
	IndexedAt string `json:"indexedAt,omitempty"`
	SourceURL string `json:"sourceUrl,omitempty"`
}

// ArtSource supplies items for the semantic index (the art_images DB table).
type ArtSource interface {
	IterForIndex(ctx context.Context, cap int, fn func(model.ArtImage) error) (int, error)
	TopTags(ctx context.Context, limit int) ([]queries.TagCount, error)
}

type Service struct {
	manifestURL string
	embedder    provider.EmbeddingProvider
	db          ArtSource
	client      *http.Client
	cacheDir    string
	indexCap    int

	mu         sync.RWMutex
	ready      bool
	indexing   bool
	lastErr    string
	sourceKind string
	indexedAt  time.Time
	items      []Item
	vectors    [][]float32
	tags       []queries.TagCount
}

func New(manifestURL string, embedder provider.EmbeddingProvider) *Service {
	if strings.TrimSpace(manifestURL) == "" {
		manifestURL = DefaultManifestURL
	}
	indexCap := 0
	if v := strings.TrimSpace(os.Getenv("OPENPATHS_ART_INDEX_CAP")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			indexCap = n
		}
	}
	cacheDir := strings.TrimSpace(os.Getenv("OPENPATHS_ART_INDEX_CACHE_DIR"))
	if cacheDir == "" {
		cacheDir = "tmp"
	}
	return &Service{
		manifestURL: manifestURL,
		embedder:    embedder,
		client:      &http.Client{Timeout: 60 * time.Second},
		cacheDir:    cacheDir,
		indexCap:    indexCap,
	}
}

// SetSource attaches the art_images DB as the index source (preferred over the manifest).
func (s *Service) SetSource(db ArtSource) {
	if s == nil {
		return
	}
	s.db = db
}

func (s *Service) Start(ctx context.Context) {
	if s == nil || s.embedder == nil {
		return
	}
	go s.Rebuild(ctx)
}

func (s *Service) Status() Status {
	if s == nil {
		return Status{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	status := Status{
		Enabled:   s.embedder != nil,
		Ready:     s.ready,
		Indexing:  s.indexing,
		Error:     s.lastErr,
		Items:     len(s.items),
		Source:    s.sourceKind,
		SourceURL: s.manifestURL,
	}
	if !s.indexedAt.IsZero() {
		status.IndexedAt = s.indexedAt.UTC().Format(time.RFC3339)
	}
	return status
}

// Tags returns the cached top-tag facets (refreshed on each Rebuild).
func (s *Service) Tags() []queries.TagCount {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]queries.TagCount(nil), s.tags...)
}

func (s *Service) Rebuild(ctx context.Context) {
	s.mu.Lock()
	if s.indexing {
		s.mu.Unlock()
		return
	}
	s.indexing = true
	s.lastErr = ""
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.indexing = false
		s.mu.Unlock()
	}()

	if s.embedder == nil {
		s.setError("no embedding provider configured")
		return
	}

	start := time.Now()
	items, kind, err := s.loadItems(ctx)
	if err != nil {
		s.setError(err.Error())
		return
	}
	if len(items) == 0 {
		s.setError("art index source contained no items")
		return
	}

	vectors, cached, err := s.loadOrEmbed(ctx, items)
	if err != nil {
		s.setError(err.Error())
		return
	}

	var tags []queries.TagCount
	if s.db != nil {
		if t, terr := s.db.TopTags(ctx, 80); terr != nil {
			log.Printf("zimage art index: top tags: %v", terr)
		} else {
			tags = t
		}
	}
	if len(tags) == 0 {
		tags = deriveTags(items, 80)
	}

	s.mu.Lock()
	s.items = items
	s.vectors = vectors
	s.tags = tags
	s.ready = true
	s.sourceKind = kind
	s.indexedAt = time.Now()
	s.lastErr = ""
	s.mu.Unlock()

	log.Printf("zimage art index: indexed %d prompts from %s (vectors cached=%v) in %s", len(items), kind, cached, time.Since(start).Round(time.Millisecond))
}

func (s *Service) Search(ctx context.Context, query string, limit int) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query required")
	}
	if limit <= 0 || limit > 100 {
		limit = 24
	}

	s.mu.RLock()
	ready := s.ready
	items := s.items
	vectors := s.vectors
	s.mu.RUnlock()

	if !ready {
		return nil, fmt.Errorf("art index is not ready")
	}

	resp, err := s.embedder.Embed(ctx, &model.EmbeddingRequest{
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
	queryVec := normalizeFloat64(resp.Data[0].Embedding)
	results := make([]Result, 0, len(vectors))
	for i, vec := range vectors {
		if i >= len(items) {
			break
		}
		score := dot(queryVec, vec)
		results = append(results, Result{Item: items[i], Score: float64(score)})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// loadItems prefers the DB source; falls back to the static manifest.
func (s *Service) loadItems(ctx context.Context) ([]Item, string, error) {
	if s.db != nil {
		items := make([]Item, 0, 1<<16)
		_, err := s.db.IterForIndex(ctx, s.indexCap, func(a model.ArtImage) error {
			items = append(items, a)
			return nil
		})
		if err != nil {
			return nil, "db", fmt.Errorf("load art images from db: %w", err)
		}
		if len(items) > 0 {
			return items, "db", nil
		}
		log.Printf("zimage art index: db empty, falling back to manifest %s", s.manifestURL)
	}
	items, err := s.fetchItems(ctx)
	return items, "manifest", err
}

func (s *Service) fetchItems(ctx context.Context) ([]Item, error) {
	var manifest Manifest
	if err := s.fetchJSON(ctx, s.manifestURL, &manifest); err != nil {
		return nil, fmt.Errorf("fetch art manifest: %w", err)
	}
	if len(manifest.Chunks) == 0 {
		return nil, fmt.Errorf("manifest has no chunks")
	}
	items := make([]Item, 0, manifest.Count)
	for _, chunk := range manifest.Chunks {
		chunkURL, err := resolveChunkURL(s.manifestURL, manifest.PublicBase, chunk)
		if err != nil {
			return nil, err
		}
		var batch []Item
		if err := s.fetchJSON(ctx, chunkURL, &batch); err != nil {
			return nil, fmt.Errorf("fetch art chunk %s: %w", chunkURL, err)
		}
		items = append(items, batch...)
	}
	return items, nil
}

func (s *Service) fetchJSON(ctx context.Context, resource string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resource, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		limited, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(limited)))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *Service) setError(message string) {
	log.Printf("zimage art index: %s", message)
	s.mu.Lock()
	s.lastErr = message
	s.ready = false
	s.mu.Unlock()
}

// ---- vector cache ----

const vectorCacheMagic uint32 = 0x5A494152 // "ZIAR"

// loadOrEmbed returns embedding vectors, using an on-disk cache keyed by the
// ordered set of item IDs so reboots don't re-embed hundreds of thousands of prompts.
func (s *Service) loadOrEmbed(ctx context.Context, items []Item) ([][]float32, bool, error) {
	key := idHash(items)
	cachePath := filepath.Join(s.cacheDir, "art-index-vectors.bin")
	if vecs, ok := readVectorCache(cachePath, len(items), key); ok {
		return vecs, true, nil
	}
	vecs, err := embedPrompts(ctx, s.embedder, items, 256)
	if err != nil {
		return nil, false, err
	}
	if err := writeVectorCache(cachePath, key, vecs); err != nil {
		log.Printf("zimage art index: vector cache write failed: %v", err)
	}
	return vecs, false, nil
}

func idHash(items []Item) uint64 {
	h := fnv.New64a()
	for i := range items {
		_, _ = h.Write([]byte(items[i].ID))
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64()
}

func readVectorCache(path string, count int, key uint64) ([][]float32, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	var magic, version uint32
	var fcount, dim int64
	var fkey uint64
	r := func(v any) bool { return binary.Read(f, binary.LittleEndian, v) == nil }
	if !r(&magic) || !r(&version) || !r(&fcount) || !r(&dim) || !r(&fkey) {
		return nil, false
	}
	if magic != vectorCacheMagic || int(fcount) != count || fkey != key || dim <= 0 || dim > 1<<16 {
		return nil, false
	}
	flat := make([]float32, int(fcount)*int(dim))
	if err := binary.Read(f, binary.LittleEndian, flat); err != nil {
		return nil, false
	}
	vecs := make([][]float32, count)
	for i := 0; i < count; i++ {
		vecs[i] = flat[i*int(dim) : (i+1)*int(dim)]
	}
	return vecs, true
}

func writeVectorCache(path string, key uint64, vecs [][]float32) error {
	if len(vecs) == 0 {
		return nil
	}
	dim := len(vecs[0])
	if dim == 0 {
		return fmt.Errorf("empty vectors")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := func(v any) error { return binary.Write(f, binary.LittleEndian, v) }
	if err := w(vectorCacheMagic); err != nil {
		f.Close()
		return err
	}
	_ = w(uint32(1))
	_ = w(int64(len(vecs)))
	_ = w(int64(dim))
	_ = w(key)
	for _, v := range vecs {
		if len(v) != dim {
			padded := make([]float32, dim)
			copy(padded, v)
			v = padded
		}
		if err := w(v); err != nil {
			f.Close()
			os.Remove(tmp)
			return err
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func embedPrompts(ctx context.Context, embedder provider.EmbeddingProvider, items []Item, batchSize int) ([][]float32, error) {
	if batchSize <= 0 {
		batchSize = 128
	}
	vectors := make([][]float32, len(items))
	for start := 0; start < len(items); start += batchSize {
		end := start + batchSize
		if end > len(items) {
			end = len(items)
		}
		inputs := make([]string, end-start)
		for i := start; i < end; i++ {
			inputs[i-start] = items[i].Prompt
		}
		resp, err := embedder.Embed(ctx, &model.EmbeddingRequest{
			Model:        "gobed",
			Input:        inputs,
			LongTextMode: "truncate",
		})
		if err != nil {
			return nil, fmt.Errorf("embed art prompts %d-%d: %w", start, end, err)
		}
		if len(resp.Data) != len(inputs) {
			return nil, fmt.Errorf("embed art prompts %d-%d: got %d embeddings for %d inputs", start, end, len(resp.Data), len(inputs))
		}
		for _, data := range resp.Data {
			if data.Index < 0 || data.Index >= len(inputs) {
				return nil, fmt.Errorf("embed art prompts %d-%d: invalid embedding index %d", start, end, data.Index)
			}
			vectors[start+data.Index] = normalizeFloat64(data.Embedding)
		}
	}
	return vectors, nil
}

// deriveTags is a fallback tag facet when no DB source is available (manifest mode).
func deriveTags(items []Item, limit int) []queries.TagCount {
	counts := map[string]int{}
	for i := range items {
		for _, tag := range items[i].Tags {
			counts[tag]++
		}
	}
	out := make([]queries.TagCount, 0, len(counts))
	for tag, n := range counts {
		out = append(out, queries.TagCount{Tag: tag, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Tag < out[j].Tag
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
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

func resolveChunkURL(manifestURL, publicBase string, chunk Chunk) (string, error) {
	if chunk.URL != "" {
		return chunk.URL, nil
	}
	if chunk.Path == "" {
		return "", fmt.Errorf("manifest chunk missing path")
	}
	if strings.HasPrefix(chunk.Path, "http://") || strings.HasPrefix(chunk.Path, "https://") {
		return chunk.Path, nil
	}
	if publicBase != "" {
		base := strings.TrimRight(publicBase, "/")
		return base + "/" + strings.TrimLeft(chunk.Path, "/"), nil
	}
	u, err := url.Parse(manifestURL)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(path.Dir(u.Path), chunk.Path)
	return u.String(), nil
}
