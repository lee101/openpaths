package gobed

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	gobedlib "github.com/lee101/gobed"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

type GobedProvider struct {
	model *gobedlib.SimpleInt8Model512
	mu    sync.RWMutex
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
		emb, err := p.model.Embed(text)
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
		totalTokens += len(text) / 4
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
