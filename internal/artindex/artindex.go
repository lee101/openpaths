package artindex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

const DefaultManifestURL = "https://openpathsstatic.openpaths.io/static/data/zimage-art/manifest.json"

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

type Item struct {
	ID        string   `json:"id"`
	Slug      string   `json:"slug"`
	Title     string   `json:"title,omitempty"`
	Prompt    string   `json:"prompt"`
	ImageURL  string   `json:"imageUrl"`
	ThumbURL  string   `json:"thumbUrl,omitempty"`
	Width     int      `json:"width,omitempty"`
	Height    int      `json:"height,omitempty"`
	Model     string   `json:"model"`
	Seed      int64    `json:"seed,omitempty"`
	Steps     int      `json:"steps,omitempty"`
	Source    string   `json:"source,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	CreatedAt string   `json:"createdAt,omitempty"`
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
	IndexedAt string `json:"indexedAt,omitempty"`
	SourceURL string `json:"sourceUrl,omitempty"`
}

type Service struct {
	manifestURL string
	embedder    provider.EmbeddingProvider
	client      *http.Client

	mu        sync.RWMutex
	ready     bool
	indexing  bool
	lastErr   string
	indexedAt time.Time
	items     []Item
	vectors   [][]float32
}

func New(manifestURL string, embedder provider.EmbeddingProvider) *Service {
	if strings.TrimSpace(manifestURL) == "" {
		manifestURL = DefaultManifestURL
	}
	return &Service{
		manifestURL: manifestURL,
		embedder:    embedder,
		client:      &http.Client{Timeout: 45 * time.Second},
	}
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
		SourceURL: s.manifestURL,
	}
	if !s.indexedAt.IsZero() {
		status.IndexedAt = s.indexedAt.UTC().Format(time.RFC3339)
	}
	return status
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
	items, err := s.fetchItems(ctx)
	if err != nil {
		s.setError(err.Error())
		return
	}
	if len(items) == 0 {
		s.setError("art manifest contained no items")
		return
	}

	vectors, err := embedPrompts(ctx, s.embedder, items, 128)
	if err != nil {
		s.setError(err.Error())
		return
	}

	s.mu.Lock()
	s.items = items
	s.vectors = vectors
	s.ready = true
	s.indexedAt = time.Now()
	s.lastErr = ""
	s.mu.Unlock()

	log.Printf("zimage art index: indexed %d prompts from %s in %s", len(items), s.manifestURL, time.Since(start).Round(time.Millisecond))
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
	items := append([]Item(nil), s.items...)
	vectors := append([][]float32(nil), s.vectors...)
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
