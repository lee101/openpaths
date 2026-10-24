package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

// DecisionProvider answers TypeSafe-style typed questions (noul/choice/score).
type DecisionProvider interface {
	Name() string
	Decide(ctx context.Context, req *model.DecisionRequest) (*model.DecisionResponse, error)
}

// StructuredProvider runs typed-output models (field extraction, decision
// questions) that take a JSON body and return a JSON result, billed on the
// input tokens the upstream reports.
type StructuredProvider interface {
	Name() string
	RunStructured(ctx context.Context, providerModelID string, body map[string]any) (result json.RawMessage, inputTokens int, err error)
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

// TuneDefaultTransport raises the idle-connection limits of http.DefaultTransport.
// Every provider and search client shares it, and the stock MaxIdleConnsPerHost
// of 2 closes all but two keep-alive connections to a busy upstream after each
// burst, forcing a fresh TCP+TLS handshake on the next one.
func TuneDefaultTransport() {
	t, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return
	}
	t.MaxIdleConns = 1024
	t.MaxIdleConnsPerHost = 256
	t.IdleConnTimeout = 90 * time.Second
}

// defaultUserAgent identifies OpenPaths to upstream APIs. Go's default
// "Go-http-client/1.1" is rejected with HTTP 403 by some upstream CDNs
// (Cloudflare in front of the Nous Research inference API), which broke the
// `nous` provider's chat route and model discovery.
const defaultUserAgent = "openpaths/1.0 (+https://openpaths.io)"

// UserAgentTransport wraps base (defaulting to http.DefaultTransport) so every
// request carries a real User-Agent unless the caller set one explicitly.
// http.DefaultTransport itself is never mutated: safefetch.NewClient asserts it
// is a *http.Transport.
func UserAgentTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return userAgentTransport{base: base, ua: defaultUserAgent}
}

type userAgentTransport struct {
	base http.RoundTripper
	ua   string
}

func (t userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", t.ua)
	}
	return t.base.RoundTrip(req)
}
