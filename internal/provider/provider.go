package provider

import (
	"context"
	"fmt"

	"github.com/openpath/openpath/internal/model"
)

// Provider defines the interface every LLM provider must implement.
type Provider interface {
	Name() string
	ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error)
	ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan StreamEvent, error)
	HealthCheck(ctx context.Context) error
}

// StreamEvent represents one event in an SSE stream.
type StreamEvent struct {
	Chunk *model.ChatCompletionChunk
	Usage *model.UsageInfo
	Done  bool
	Err   error
}

// ProviderError wraps errors with provider-specific context.
type ProviderError struct {
	Provider   string
	StatusCode int
	Message    string
	Retryable  bool
	Err        error
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider %s: %d %s", e.Provider, e.StatusCode, e.Message)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}
