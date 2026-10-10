package gobed

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	gobedlib "github.com/lee101/gobed"
	"github.com/lee101/gobed/pkg/ann/simd"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

const (
	gobedChunkSize = 256
	dim            = gobedlib.Int8EmbeddingDim
	// inputs per worker before a batch is split across goroutines
	parallelGrain = 8
	maxInputBytes = 4 << 20
	// MaxInputsPerRequest bounds a single /v1/embeddings call.
	MaxInputsPerRequest = 4096
	// embedChunk is the number of inputs per pooled job in EmbedRaw.
	embedChunk = 32
)

type GobedProvider struct {
	model   *gobedlib.SimpleInt8Model512
	tok     *fastTokenizer
	table   [][]int8
	scales  []float32
	workers int
	// pool is the bounded worker queue used by EmbedRaw; when saturated the
	// request diverts to overflow (if set) or is shed with a 429.
	pool     *batcher
	overflow provider.EmbeddingProvider
}

type scratch struct {
	tokens []int16
	acc    [dim]float32
}

var scratchPool = sync.Pool{New: func() any {
	return &scratch{tokens: make([]int16, 0, gobedChunkSize)}
}}

func New() (*GobedProvider, error) {
	start := time.Now()
	gobedlib.SetSimpleInt8Verbose(false)
	m, err := gobedlib.LoadSimpleInt8Model512()
	if err != nil {
		return nil, fmt.Errorf("gobed load: %w", err)
	}
	p := &GobedProvider{model: m, table: m.EmbeddingTable(), scales: m.ScaleTable(), workers: runtime.GOMAXPROCS(0), pool: newBatcher(0, 0, 16, 0)}
	if tok, err := loadFastTokenizer("model/tokenizer.json"); err != nil {
		log.Printf("gobed: fast tokenizer disabled (%v)", err)
	} else {
		p.tok = tok
	}
	log.Printf("gobed: model loaded in %dms (fast tokenizer=%t)", time.Since(start).Milliseconds(), p.tok != nil)
	return p, nil
}

func (p *GobedProvider) Name() string { return "gobed" }

func (p *GobedProvider) Embed(ctx context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	inputs, err := normalizeInput(req.Input)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "gobed", StatusCode: 400, Message: err.Error()}
	}
	mode := normalizeLongTextMode(req.LongTextMode)
	if mode != "truncate" && mode != "average_chunks" {
		return nil, &provider.ProviderError{Provider: "gobed", StatusCode: 400, Message: fmt.Sprintf("unsupported long_text_mode %q", req.LongTextMode)}
	}
	dims := req.Dimensions
	if dims < 0 || dims > dim {
		return nil, &provider.ProviderError{Provider: "gobed", StatusCode: 400, Message: fmt.Sprintf("dimensions must be between 1 and %d", dim)}
	}
	if dims == 0 {
		dims = dim
	}
	b64 := req.EncodingFormat == "base64"
	for _, s := range inputs {
		if len(s) > maxInputBytes {
			return nil, &provider.ProviderError{Provider: "gobed", StatusCode: 400, Message: fmt.Sprintf("input exceeds %d bytes", maxInputBytes)}
		}
	}

	data := make([]model.EmbeddingData, len(inputs))
	tokens := make([]int, len(inputs))
	run := func(lo, hi int) {
		sc := scratchPool.Get().(*scratch)
		for i := lo; i < hi; i++ {
			tokens[i] = p.embedInto(sc, mode, inputs[i], dims, b64, &data[i])
			data[i].Object = "embedding"
			data[i].Index = i
		}
		scratchPool.Put(sc)
	}

	w := len(inputs) / parallelGrain
	if w > p.workers {
		w = p.workers
	}
	if w <= 1 {
		run(0, len(inputs))
	} else {
		var wg sync.WaitGroup
		step := (len(inputs) + w - 1) / w
		for lo := 0; lo < len(inputs); lo += step {
			hi := lo + step
			if hi > len(inputs) {
				hi = len(inputs)
			}
			wg.Add(1)
			go func(lo, hi int) {
				defer wg.Done()
				run(lo, hi)
			}(lo, hi)
		}
		wg.Wait()
	}

	total := 0
	for _, n := range tokens {
		total += n
	}
	return &model.EmbeddingResponse{
		Object: "list",
		Data:   data,
		Model:  req.Model,
		Usage:  model.EmbeddingUsage{PromptTokens: total, TotalTokens: total},
	}, nil
}

func (p *GobedProvider) tokenize(sc *scratch, mode, text string) []int16 {
	limit := math.MaxInt32
	if mode == "truncate" {
		limit = gobedChunkSize
	}
	if p.tok != nil && isASCII(text) {
		return p.tok.tokenize(text, sc.tokens[:0], limit)
	}
	t := p.model.SimpleTokenize(text)
	if len(t) > limit {
		t = t[:limit]
	}
	return t
}

// embedVec embeds text into sc.acc and returns the (MRL-truncated and
// renormalized when dims < dim) vector plus the token count billed. The slice
// aliases sc and is only valid until the scratch is reused.
func (p *GobedProvider) embedVec(sc *scratch, mode, text string, dims int) ([]float32, int) {
	toks := p.tokenize(sc, mode, text)
	if cap(toks) > cap(sc.tokens) && cap(toks) <= 1<<14 {
		sc.tokens = toks[:0]
	}
	acc := sc.acc[:]
	for i := range acc {
		acc[i] = 0
	}
	valid := 0
	for _, t := range toks {
		if t < 0 || int(t) >= len(p.table) {
			continue
		}
		simd.EmbedAccum(p.table[t], p.scales[t], acc)
		valid++
	}
	if valid > 1 {
		simd.ScaleF32(acc, 1/float32(valid))
	}
	v := acc[:dims]
	if dims < dim {
		var n float64
		for _, x := range v {
			n += float64(x) * float64(x)
		}
		if n > 0 {
			simd.ScaleF32(v, float32(1/math.Sqrt(n)))
		}
	}
	return v, len(toks)
}

// embedInto embeds text into out and returns the token count billed.
func (p *GobedProvider) embedInto(sc *scratch, mode, text string, dims int, b64 bool, out *model.EmbeddingData) int {
	v, n := p.embedVec(sc, mode, text, dims)
	if b64 {
		out.Base64 = encodeF32(v)
	} else {
		out.Embedding = make([]float64, dims)
		for i, x := range v {
			out.Embedding[i] = float64(x)
		}
		out.F32 = true
	}
	return n
}

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

// EmbedRaw embeds all inputs on the shared bounded pool and returns one
// contiguous float32 buffer (N*Dim), skipping per-vector allocation and float64
// conversion. Dimensions truncates (MRL) and renormalizes. Call Release on the
// result once it has been written.
func (p *GobedProvider) EmbedRaw(ctx context.Context, req *model.EmbeddingRequest) (*model.RawEmbeddings, error) {
	inputs, err := normalizeInput(req.Input)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "gobed", StatusCode: 400, Message: err.Error()}
	}
	if len(inputs) > MaxInputsPerRequest {
		return nil, &provider.ProviderError{Provider: "gobed", StatusCode: 400,
			Message: fmt.Sprintf("too many inputs: %d (max %d)", len(inputs), MaxInputsPerRequest)}
	}
	dims := dim
	if req.Dimensions != 0 {
		if req.Dimensions < 1 || req.Dimensions > dim {
			return nil, &provider.ProviderError{Provider: "gobed", StatusCode: 400,
				Message: fmt.Sprintf("dimensions must be between 1 and %d", dim)}
		}
		dims = req.Dimensions
	}
	mode := normalizeLongTextMode(req.LongTextMode)
	if mode != "truncate" && mode != "average_chunks" {
		return nil, &provider.ProviderError{Provider: "gobed", StatusCode: 400,
			Message: fmt.Sprintf("unsupported long_text_mode %q", req.LongTextMode)}
	}
	for _, s := range inputs {
		if len(s) > maxInputBytes {
			return nil, &provider.ProviderError{Provider: "gobed", StatusCode: 400, Message: fmt.Sprintf("input exceeds %d bytes", maxInputBytes)}
		}
	}

	n := len(inputs)
	buf, release := getVecs(n * dims)
	var tokens atomic.Int64
	jobs := make([]func(), 0, (n+embedChunk-1)/embedChunk)
	for start := 0; start < n; start += embedChunk {
		start, end := start, min(start+embedChunk, n)
		jobs = append(jobs, func() {
			sc := scratchPool.Get().(*scratch)
			t := 0
			for i := start; i < end; i++ {
				v, used := p.embedVec(sc, mode, inputs[i], dims)
				copy(buf[i*dims:(i+1)*dims], v)
				t += used
			}
			scratchPool.Put(sc)
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
	out := model.NewRawEmbeddings(n, dims, buf, release)
	out.Tokens = int(tokens.Load())
	return out, nil
}

func (p *GobedProvider) overflowRaw(ctx context.Context, req *model.EmbeddingRequest) (*model.RawEmbeddings, error) {
	resp, err := p.overflow.Embed(ctx, req)
	if err != nil {
		return nil, err
	}
	d := 0
	if len(resp.Data) > 0 {
		d = len(resp.Data[0].Embedding)
	}
	vecs := make([]float32, 0, len(resp.Data)*d)
	for _, e := range resp.Data {
		for _, x := range e.Embedding {
			vecs = append(vecs, float32(x))
		}
	}
	out := model.NewRawEmbeddings(len(resp.Data), d, vecs, nil)
	out.Tokens = resp.Usage.TotalTokens
	return out, nil
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
