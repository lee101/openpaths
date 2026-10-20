package gobed

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"

	gobedlib "github.com/lee101/gobed"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

const gobedChunkSize = 256

type GobedProvider struct {
	model    *gobedlib.SimpleInt8Model512
	pool     *batcher
	overflow provider.EmbeddingProvider
}

func New() (*GobedProvider, error) {
	start := time.Now()
	gobedlib.SetSimpleInt8Verbose(false)
	m, err := gobedlib.LoadSimpleInt8Model512()
	if err != nil {
		return nil, fmt.Errorf("gobed load: %w", err)
	}
	log.Printf("gobed: model loaded in %dms", time.Since(start).Milliseconds())
	return &GobedProvider{model: m, pool: newBatcher(0, 0, 16, 0)}, nil
}

func (p *GobedProvider) Name() string { return "gobed" }

const (
	// MaxInputsPerRequest bounds a single /v1/embeddings call.
	MaxInputsPerRequest = 4096
	embedChunk          = 32
	embedDim            = gobedlib.Int8EmbeddingDim
)

var vecPool sync.Pool

func getVecs(n int) ([]float32, func()) {
	if v, ok := vecPool.Get().(*[]float32); ok && cap(*v) >= n {
		s := (*v)[:n]
		return s, func() { vecPool.Put(v) }
	}
	s := make([]float32, n)
	return s, func() { vecPool.Put(&s) }
}

// SetOverflow installs a backend used when the local queue is saturated.
func (p *GobedProvider) SetOverflow(o provider.EmbeddingProvider) { p.overflow = o }

func (p *GobedProvider) Embed(ctx context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	raw, err := p.EmbedRaw(ctx, req)
	if err != nil {
		return nil, err
	}
	defer raw.Release()
	data := make([]model.EmbeddingData, raw.N)
	for i := range data {
		v := raw.Vec(i)
		emb64 := make([]float64, len(v))
		for j, x := range v {
			emb64[j] = float64(x)
		}
		data[i] = model.EmbeddingData{Object: "embedding", Embedding: emb64, Index: i}
	}
	return &model.EmbeddingResponse{
		Object: "list",
		Data:   data,
		Model:  req.Model,
		Usage:  model.EmbeddingUsage{PromptTokens: raw.Tokens, TotalTokens: raw.Tokens},
	}, nil
}

// EmbedRaw embeds all inputs in parallel on the shared bounded pool and
// returns one contiguous float32 buffer. Dimensions truncates (MRL) and
// renormalizes; otherwise vectors are unchanged from the model output.
func (p *GobedProvider) EmbedRaw(ctx context.Context, req *model.EmbeddingRequest) (*model.RawEmbeddings, error) {
	inputs, err := normalizeInput(req.Input)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "gobed", StatusCode: 400, Message: err.Error()}
	}
	if len(inputs) > MaxInputsPerRequest {
		return nil, &provider.ProviderError{Provider: "gobed", StatusCode: 400,
			Message: fmt.Sprintf("too many inputs: %d (max %d)", len(inputs), MaxInputsPerRequest)}
	}
	dim := embedDim
	if req.Dimensions != 0 {
		if req.Dimensions < 1 || req.Dimensions > embedDim {
			return nil, &provider.ProviderError{Provider: "gobed", StatusCode: 400,
				Message: fmt.Sprintf("dimensions must be between 1 and %d", embedDim)}
		}
		dim = req.Dimensions
	}
	mode := normalizeLongTextMode(req.LongTextMode)
	if mode != "truncate" && mode != "average_chunks" {
		return nil, &provider.ProviderError{Provider: "gobed", StatusCode: 400,
			Message: fmt.Sprintf("unsupported long_text_mode %q", req.LongTextMode)}
	}

	n := len(inputs)
	full, release := getVecs(n * embedDim)
	var tokens atomic.Int64
	var failed atomic.Value
	jobs := make([]func(), 0, (n+embedChunk-1)/embedChunk)
	for start := 0; start < n; start += embedChunk {
		start, end := start, min(start+embedChunk, n)
		jobs = append(jobs, func() {
			t := 0
			for i := start; i < end; i++ {
				used, err := p.embedInto(mode, inputs[i], full[i*embedDim:(i+1)*embedDim])
				if err != nil {
					failed.Store(err)
					return
				}
				t += used
			}
			tokens.Add(int64(t))
		})
	}
	if err := p.pool.runAll(ctx, jobs); err != nil {
		release()
		if errors.Is(err, ErrSaturated) {
			if p.overflow != nil {
				return p.overflowRaw(ctx, req)
			}
			return nil, &provider.ProviderError{Provider: "gobed", StatusCode: 429, Message: "embedding queue saturated, retry shortly"}
		}
		return nil, err
	}
	if e, ok := failed.Load().(error); ok && e != nil {
		release()
		return nil, &provider.ProviderError{Provider: "gobed", StatusCode: 500, Message: e.Error(), Retryable: true, Err: e}
	}

	out := model.NewRawEmbeddings(n, embedDim, full, release)
	out.Tokens = int(tokens.Load())
	if dim != embedDim {
		trunc := make([]float32, n*dim)
		for i := 0; i < n; i++ {
			v := trunc[i*dim : (i+1)*dim]
			copy(v, full[i*embedDim:i*embedDim+dim])
			var ss float32
			for _, x := range v {
				ss += x * x
			}
			if ss > 0 {
				inv := float32(1 / math.Sqrt(float64(ss)))
				for k := range v {
					v[k] *= inv
				}
			}
		}
		release()
		out = model.NewRawEmbeddings(n, dim, trunc, nil)
		out.Tokens = int(tokens.Load())
	}
	return out, nil
}

func (p *GobedProvider) overflowRaw(ctx context.Context, req *model.EmbeddingRequest) (*model.RawEmbeddings, error) {
	resp, err := p.overflow.Embed(ctx, req)
	if err != nil {
		return nil, err
	}
	dim := 0
	if len(resp.Data) > 0 {
		dim = len(resp.Data[0].Embedding)
	}
	vecs := make([]float32, 0, len(resp.Data)*dim)
	for _, d := range resp.Data {
		for _, x := range d.Embedding {
			vecs = append(vecs, float32(x))
		}
	}
	out := model.NewRawEmbeddings(len(resp.Data), dim, vecs, nil)
	out.Tokens = resp.Usage.TotalTokens
	return out, nil
}

// embedInto writes one embedding into dst (len embedDim) and returns tokens used.
func (p *GobedProvider) embedInto(mode, text string, dst []float32) (int, error) {
	tokens := p.model.SimpleTokenize(text)
	if len(tokens) == 0 {
		clear(dst)
		return 0, nil
	}
	if mode == "average_chunks" {
		emb, used, err := p.embedAverageChunks(tokens)
		if err != nil {
			return 0, err
		}
		copy(dst, emb)
		return used, nil
	}
	if len(tokens) > gobedChunkSize {
		tokens = tokens[:gobedChunkSize]
	}
	p.model.EmbedTokensIntoSIMD(tokens, dst)
	return len(tokens), nil
}

func (p *GobedProvider) embedAverageChunks(tokens []int16) ([]float32, int, error) {
	accum := make([]float32, gobedlib.Int8EmbeddingDim)
	totalWeight := 0

	for start := 0; start < len(tokens); start += gobedChunkSize {
		end := start + gobedChunkSize
		if end > len(tokens) {
			end = len(tokens)
		}
		chunk := tokens[start:end]
		emb, err := p.model.EmbedTokens(chunk)
		if err != nil {
			return nil, 0, err
		}
		weight := len(chunk)
		totalWeight += weight
		for i, v := range emb {
			accum[i] += v * float32(weight)
		}
	}

	if totalWeight > 0 {
		inv := 1 / float32(totalWeight)
		for i := range accum {
			accum[i] *= inv
		}
	}

	return accum, len(tokens), nil
}

func normalizeLongTextMode(mode string) string {
	switch mode {
	case "", "truncate":
		return "truncate"
	case "average_chunks", "avg_chunks", "chunk_mean":
		return "average_chunks"
	default:
		return mode
	}
}

func normalizeInput(input any) ([]string, error) {
	switch v := input.(type) {
	case string:
		return []string{v}, nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("input array must contain strings")
			}
			out = append(out, s)
		}
		return out, nil
	case []string:
		return v, nil
	default:
		return nil, fmt.Errorf("input must be a string or array of strings")
	}
}
