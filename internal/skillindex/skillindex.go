// Package skillindex is the in-process gobed semantic index over the skills table.
// It mirrors internal/artindex but is DB-only and small (a few hundred skills), so
// it embeds every skill at startup without a disk vector cache. The handler falls
// back to SQL ILIKE (queries.SkillQueries.SearchILIKE) whenever the index is not
// ready or the embedder is unavailable.
package skillindex

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

// Source supplies skills for the index (the skills DB table).
type Source interface {
	IterForIndex(ctx context.Context, cap int, fn func(model.Skill) error) (int, error)
}

type Result struct {
	model.Skill
	Score float64 `json:"score"`
}

type Status struct {
	Enabled   bool   `json:"enabled"`
	Ready     bool   `json:"ready"`
	Indexing  bool   `json:"indexing"`
	Error     string `json:"error,omitempty"`
	Items     int    `json:"items"`
	IndexedAt string `json:"indexedAt,omitempty"`
}

type Service struct {
	embedder provider.EmbeddingProvider
	db       Source

	mu        sync.RWMutex
	ready     bool
	indexing  bool
	lastErr   string
	indexedAt time.Time
	items     []model.Skill
	vectors   [][]float32
}

func New(embedder provider.EmbeddingProvider) *Service {
	return &Service{embedder: embedder}
}

func (s *Service) SetSource(db Source) {
	if s != nil {
		s.db = db
	}
}

func (s *Service) Start(ctx context.Context) {
	if s == nil || s.embedder == nil {
		return
	}
	go s.Rebuild(ctx)
}

func (s *Service) Ready() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ready
}

func (s *Service) Status() Status {
	if s == nil {
		return Status{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	st := Status{Enabled: s.embedder != nil, Ready: s.ready, Indexing: s.indexing, Error: s.lastErr, Items: len(s.items)}
	if !s.indexedAt.IsZero() {
		st.IndexedAt = s.indexedAt.UTC().Format(time.RFC3339)
	}
	return st
}

func (s *Service) Rebuild(ctx context.Context) {
	if s == nil {
		return
	}
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

	if s.embedder == nil || s.db == nil {
		s.setError("no embedder or source configured")
		return
	}
	start := time.Now()
	items := make([]model.Skill, 0, 256)
	if _, err := s.db.IterForIndex(ctx, 0, func(sk model.Skill) error {
		items = append(items, sk)
		return nil
	}); err != nil {
		s.setError(err.Error())
		return
	}
	if len(items) == 0 {
		s.setError("skills source contained no skills")
		return
	}

	vectors, err := s.embedAll(ctx, items)
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
	log.Printf("skill index: embedded %d skills in %s", len(items), time.Since(start).Round(time.Millisecond))
}

func (s *Service) Search(ctx context.Context, query string, f model.SkillFilters, limit int) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query required")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	s.mu.RLock()
	ready := s.ready
	items := s.items
	vectors := s.vectors
	s.mu.RUnlock()
	if !ready {
		return nil, fmt.Errorf("skill index not ready")
	}
	resp, err := s.embedder.Embed(ctx, &model.EmbeddingRequest{Model: "gobed", Input: query, LongTextMode: "truncate"})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("embed query: no embedding returned")
	}
	qv := normalize(resp.Data[0].Embedding)
	out := make([]Result, 0, len(vectors))
	for i, v := range vectors {
		if i >= len(items) {
			break
		}
		if f.Source != "" && items[i].Source != f.Source {
			continue
		}
		if f.Category != "" && items[i].Category != f.Category {
			continue
		}
		out = append(out, Result{Skill: items[i], Score: float64(dot(qv, v))})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Service) embedAll(ctx context.Context, items []model.Skill) ([][]float32, error) {
	const batch = 128
	vectors := make([][]float32, len(items))
	for start := 0; start < len(items); start += batch {
		end := start + batch
		if end > len(items) {
			end = len(items)
		}
		inputs := make([]string, end-start)
		for i := start; i < end; i++ {
			inputs[i-start] = items[i].SearchText()
		}
		resp, err := s.embedder.Embed(ctx, &model.EmbeddingRequest{Model: "gobed", Input: inputs, LongTextMode: "truncate"})
		if err != nil {
			return nil, fmt.Errorf("embed skills %d-%d: %w", start, end, err)
		}
		if len(resp.Data) != len(inputs) {
			return nil, fmt.Errorf("embed skills %d-%d: got %d for %d inputs", start, end, len(resp.Data), len(inputs))
		}
		for _, d := range resp.Data {
			if d.Index < 0 || d.Index >= len(inputs) {
				return nil, fmt.Errorf("embed skills: invalid index %d", d.Index)
			}
			vectors[start+d.Index] = normalize(d.Embedding)
		}
	}
	return vectors, nil
}

func (s *Service) setError(message string) {
	log.Printf("skill index: %s", message)
	s.mu.Lock()
	s.lastErr = message
	s.ready = false
	s.mu.Unlock()
}

func normalize(values []float64) []float32 {
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
