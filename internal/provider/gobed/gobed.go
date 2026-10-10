package gobed

import (
	"context"
	"fmt"
	"log"
	"math"
	"runtime"
	"sync"
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
)

type GobedProvider struct {
	model   *gobedlib.SimpleInt8Model512
	workers int
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
	p := &GobedProvider{model: m, workers: runtime.GOMAXPROCS(0)}
	log.Printf("gobed: model loaded in %dms", time.Since(start).Milliseconds())
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

// embedInto embeds text into out and returns the token count billed.
func (p *GobedProvider) embedInto(sc *scratch, mode, text string, dims int, b64 bool, out *model.EmbeddingData) int {
	limit := 0
	if mode == "truncate" {
		limit = gobedChunkSize
	}
	toks := p.model.TokenizeInto(text, sc.tokens[:0], limit)
	if cap(toks) > cap(sc.tokens) && cap(toks) <= 1<<14 {
		sc.tokens = toks[:0]
	}
	acc := sc.acc[:]
	p.model.EmbedTokensIntoSIMD(toks, acc)
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
	if b64 {
		out.Base64 = encodeF32(v)
	} else {
		out.Embedding = make([]float64, dims)
		for i, x := range v {
			out.Embedding[i] = float64(x)
		}
		out.F32 = true
	}
	return len(toks)
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
