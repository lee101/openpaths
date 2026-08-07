package agent

import (
	"context"
	"encoding/binary"
	"math"
	"sort"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

// Ingest converts is already done by caller; this chunks+embeds+stores markdown
// for one document under a data source. Returns the number of chunks written.
func (e *Engine) Ingest(ctx context.Context, userID, agentID, sourceID, title, markdown string) (int, error) {
	chunks := Chunk(markdown, 1400, 160)
	for i, c := range chunks {
		doc := model.AgentDocument{
			DataSourceID: sourceID,
			AgentID:      agentID,
			UserID:       userID,
			Title:        title,
			ChunkIndex:   i,
			Content:      c,
		}
		if e.embedder != nil {
			if vec, err := embedOne(ctx, e.embedder, c); err == nil {
				doc.Embedding = encodeVec(vec)
			}
		}
		if err := e.q.InsertDocument(ctx, doc); err != nil {
			return i, err
		}
	}
	return len(chunks), nil
}

// SearchDocs runs semantic (gobed) search blended with a pg_trgm fallback over an
// agent's connected document chunks.
func (e *Engine) SearchDocs(ctx context.Context, agentID, query string, limit int) ([]model.AgentDocument, error) {
	if limit <= 0 || limit > 20 {
		limit = 6
	}
	seen := map[string]bool{}
	var results []model.AgentDocument

	if e.embedder != nil {
		if vec, err := embedOne(ctx, e.embedder, query); err == nil {
			q := normalize(vec)
			rows, err := e.q.ListVectors(ctx, agentID)
			if err == nil && len(rows) > 0 {
				for i := range rows {
					rv := decodeVec(rows[i].Embedding)
					if len(rv) == 0 {
						continue
					}
					rows[i].Score = dot(q, rv)
					rows[i].Embedding = nil
					results = append(results, rows[i])
				}
				sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
				if len(results) > limit {
					results = results[:limit]
				}
				for i := range results {
					seen[results[i].ID] = true
				}
			}
		}
	}

	tri, _ := e.q.SearchTrigram(ctx, agentID, query, limit)
	for i := range tri {
		if seen[tri[i].ID] {
			continue
		}
		results = append(results, tri[i])
	}
	if len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func embedOne(ctx context.Context, emb provider.EmbeddingProvider, text string) ([]float64, error) {
	resp, err := emb.Embed(ctx, &model.EmbeddingRequest{Model: "gobed", Input: text})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 || len(resp.Data[0].Embedding) == 0 {
		return nil, errNoEmbedding
	}
	return resp.Data[0].Embedding, nil
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
