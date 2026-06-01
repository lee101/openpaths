package provider

import (
	"context"
	"fmt"

	"github.com/openpaths/openpaths/internal/model"
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

// TranscriptionProvider defines the interface for audio transcription.
type TranscriptionProvider interface {
	Name() string
	Transcribe(ctx context.Context, req *model.TranscriptionRequest) (*model.TranscriptionResponse, error)
}

// ImageProvider defines the interface for image generation.
type ImageProvider interface {
	Name() string
	GenerateImage(ctx context.Context, req *model.ImageGenerationRequest) (*model.ImageGenerationResponse, error)
}

// Model3DProvider defines the interface for image-to-3D generation.
type Model3DProvider interface {
	Name() string
	Generate3D(ctx context.Context, req *model.Model3DGenerationRequest) (*model.Model3DGenerationResponse, error)
}

// MeshRiggingProvider defines the interface for 3D mesh auto-rigging.
type MeshRiggingProvider interface {
	Name() string
	RigMesh(ctx context.Context, req *model.MeshRiggingRequest) (*model.MeshRiggingResponse, error)
}

// VideoProvider defines the interface for video generation.
type VideoProvider interface {
	Name() string
	GenerateVideo(ctx context.Context, req *model.VideoGenerationRequest) (*model.VideoGenerationResponse, error)
}

// ForecastingProvider defines the interface for time-series forecasting
// (e.g. CuteDSL chronos2). A forecaster takes a historical series and returns
// a point forecast plus optional quantiles.
type ForecastingProvider interface {
	Name() string
	GenerateForecast(ctx context.Context, req *model.ForecastingRequest) (*model.ForecastingResponse, error)
}

// MusicProvider defines the interface for music generation.
type MusicProvider interface {
	Name() string
	GenerateMusic(ctx context.Context, req *model.MusicGenerationRequest) (*model.MusicGenerationResponse, error)
}

// SpeechProvider defines the interface for text-to-speech.
type SpeechProvider interface {
	Name() string
	GenerateSpeech(ctx context.Context, req *model.SpeechRequest) (*model.SpeechResponse, error)
}

// EmbeddingProvider defines the interface for text embeddings.
type EmbeddingProvider interface {
	Name() string
	Embed(ctx context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error)
}

// FineTuneProvider defines the interface for fine-tuning.
type FineTuneProvider interface {
	Name() string
	UploadFineTuneFile(ctx context.Context, filename string, data []byte) (providerFileID string, err error)
	CreateFineTuneJob(ctx context.Context, req *model.FineTuneJobRequest, providerTrainingFileID string, providerValidationFileID string) (*model.ProviderFineTuneJob, error)
	GetFineTuneJob(ctx context.Context, providerJobID string) (*model.ProviderFineTuneJob, error)
	CancelFineTuneJob(ctx context.Context, providerJobID string) error
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
