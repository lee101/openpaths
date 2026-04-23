package gobed

import (
	"context"
	"fmt"
	"log"
	"time"

	gobedlib "github.com/lee101/gobed"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

const gobedChunkSize = 256

type GobedProvider struct {
	model *gobedlib.SimpleInt8Model512
}

func New() (*GobedProvider, error) {
	start := time.Now()
	gobedlib.SetSimpleInt8Verbose(false)
	m, err := gobedlib.LoadSimpleInt8Model512()
	if err != nil {
		return nil, fmt.Errorf("gobed load: %w", err)
	}
	log.Printf("gobed: model loaded in %dms", time.Since(start).Milliseconds())
	return &GobedProvider{model: m}, nil
}

func (p *GobedProvider) Name() string { return "gobed" }

func (p *GobedProvider) Embed(ctx context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	inputs, err := normalizeInput(req.Input)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "gobed", StatusCode: 400, Message: err.Error(),
		}
	}

	data := make([]model.EmbeddingData, 0, len(inputs))
	totalTokens := 0

	for i, text := range inputs {
		emb, tokensUsed, err := p.embedText(req.LongTextMode, text)
		if err != nil {
			return nil, &provider.ProviderError{
				Provider: "gobed", StatusCode: 500, Message: err.Error(), Retryable: true, Err: err,
			}
		}

		emb64 := make([]float64, len(emb))
		for j, v := range emb {
			emb64[j] = float64(v)
		}

		data = append(data, model.EmbeddingData{
			Object:    "embedding",
			Embedding: emb64,
			Index:     i,
		})
		totalTokens += tokensUsed
	}

	return &model.EmbeddingResponse{
		Object: "list",
		Data:   data,
		Model:  req.Model,
		Usage: model.EmbeddingUsage{
			PromptTokens: totalTokens,
			TotalTokens:  totalTokens,
		},
	}, nil
}

func (p *GobedProvider) embedText(mode, text string) ([]float32, int, error) {
	tokens := p.model.SimpleTokenize(text)
	if len(tokens) == 0 {
		return make([]float32, gobedlib.Int8EmbeddingDim), 0, nil
	}

	switch normalizeLongTextMode(mode) {
	case "truncate":
		if len(tokens) > gobedChunkSize {
			tokens = tokens[:gobedChunkSize]
		}
		emb, err := p.model.EmbedTokens(tokens)
		return emb, len(tokens), err
	case "average_chunks":
		return p.embedAverageChunks(tokens)
	default:
		return nil, 0, fmt.Errorf("unsupported long_text_mode %q", mode)
	}
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
