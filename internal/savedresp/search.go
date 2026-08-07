package savedresp

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"sort"
	"strings"

	"github.com/openpaths/openpaths/internal/model"
)

var errNoEmbedding = errors.New("no embedding returned")

// SearchResult is a saved response plus its search relevance and the mode used.
type SearchResult struct {
	Results []model.SavedResponse `json:"results"`
	Count   int                   `json:"count"`
	Total   int                   `json:"total"`
	Mode    string                `json:"mode"` // "semantic" | "trigram" | "browse"
}

// Search returns a user's saved responses for a kind. An empty query lists newest
// first (mode=browse). A non-empty query runs semantic search (when an embedder is
// available) blended with a pg_trgm pass for exact/substring matches.
func (s *Saver) Search(ctx context.Context, userID, kind, query string, limit, offset int) (*SearchResult, error) {
	if limit <= 0 || limit > 200 {
		limit = 48
	}
	if offset < 0 {
		offset = 0
	}
	query = strings.TrimSpace(query)
	total, _ := s.q.CountByKind(ctx, userID, kind)

	if query == "" {
		rows, err := s.q.List(ctx, userID, kind, limit, offset)
		if err != nil {
			return nil, err
		}
		return &SearchResult{Results: rows, Count: len(rows), Total: total, Mode: "browse"}, nil
	}

	mode := "trigram"
	var results []model.SavedResponse
	seen := map[string]bool{}

	if s.embedder != nil {
		if vec, err := s.embed(ctx, query); err == nil {
			q := normalize(vec)
			rows, err := s.q.ListVectors(ctx, userID, kind, 0)
			if err == nil && len(rows) > 0 {
				scored := make([]model.SavedResponse, 0, len(rows))
				for i := range rows {
					rv := decodeVec(rows[i].Embedding)
					if len(rv) == 0 {
						continue
					}
					rows[i].Score = dot(q, rv)
					rows[i].Embedding = nil
					scored = append(scored, rows[i])
				}
				sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
				if len(scored) > limit {
					scored = scored[:limit]
				}
				for i := range scored {
					seen[scored[i].ID] = true
				}
				results = scored
				mode = "semantic"
			}
		}
	}

	// Trigram pass: either the sole strategy, or a supplement for exact matches the
	// semantic ranker may have missed.
	tri, err := s.q.SearchTrigram(ctx, userID, kind, query, limit, 0)
	if err != nil && len(results) == 0 {
		return nil, err
	}
	for i := range tri {
		if seen[tri[i].ID] {
			continue
		}
		tri[i].Embedding = nil
		results = append(results, tri[i])
		seen[tri[i].ID] = true
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return &SearchResult{Results: results, Count: len(results), Total: total, Mode: mode}, nil
}

// Similar returns the saved responses most semantically similar to the given item.
func (s *Saver) Similar(ctx context.Context, userID, id string, limit int) ([]model.SavedResponse, error) {
	if limit <= 0 || limit > 50 {
		limit = 12
	}
	item, err := s.q.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if s.embedder == nil || item.Prompt == "" {
		return nil, nil
	}
	vec, err := s.embed(ctx, item.Prompt)
	if err != nil {
		return nil, err
	}
	q := normalize(vec)
	rows, err := s.q.ListVectors(ctx, userID, item.Kind, 0)
	if err != nil {
		return nil, err
	}
	out := make([]model.SavedResponse, 0, len(rows))
	for i := range rows {
		if rows[i].ID == id {
			continue
		}
		rv := decodeVec(rows[i].Embedding)
		if len(rv) == 0 {
			continue
		}
		rows[i].Score = dot(q, rv)
		rows[i].Embedding = nil
		out = append(out, rows[i])
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// Get returns a single saved response owned by the user (embedding stripped).
func (s *Saver) Get(ctx context.Context, userID, id string) (*model.SavedResponse, error) {
	r, err := s.q.Get(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	r.Embedding = nil
	return r, nil
}

// ---- vector helpers (512×float32 little-endian, L2-normalized) ----

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

func encodeVec(values []float64) []byte {
	norm := normalize(values)
	buf := make([]byte, len(norm)*4)
	for i, v := range norm {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(v))
	}
	return buf
}

func decodeVec(buf []byte) []float32 {
	if len(buf) < 4 || len(buf)%4 != 0 {
		return nil
	}
	out := make([]float32, len(buf)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(buf[i*4:]))
	}
	return out
}

func dot(a, b []float32) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var out float64
	for i := 0; i < n; i++ {
		out += float64(a[i]) * float64(b[i])
	}
	return out
}
