package textgenerator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

type TextGeneratorProvider struct {
	apiKey string
	client *http.Client
}

func New(apiKey string) *TextGeneratorProvider {
	return &TextGeneratorProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *TextGeneratorProvider) Name() string { return "textgenerator" }

type featureRequest struct {
	Text        string `json:"text"`
	NumFeatures int    `json:"num_features"`
}

func (p *TextGeneratorProvider) Embed(ctx context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	inputs, err := normalizeInput(req.Input)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "textgenerator", StatusCode: 400, Message: err.Error(),
		}
	}

	dims := req.Dimensions
	if dims <= 0 {
		dims = 768
	}

	data := make([]model.EmbeddingData, 0, len(inputs))
	totalTokens := 0

	for i, text := range inputs {
		embedding, err := p.fetchEmbedding(ctx, text, dims)
		if err != nil {
			return nil, err
		}
		data = append(data, model.EmbeddingData{
			Object:    "embedding",
			Embedding: embedding,
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

func (p *TextGeneratorProvider) fetchEmbedding(ctx context.Context, text string, dims int) ([]float64, error) {
	body, _ := json.Marshal(featureRequest{Text: text, NumFeatures: dims})

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.text-generator.io/api/v1/feature-extraction", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("secret", p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "textgenerator", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, &provider.ProviderError{
			Provider:   "textgenerator",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}

	var embedding []float64
	if err := json.Unmarshal(respBody, &embedding); err != nil {
		return nil, fmt.Errorf("unmarshal embedding: %w", err)
	}
	return embedding, nil
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
