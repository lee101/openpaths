package appnz

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

type route int

const (
	routeLocal route = iota
	routeRunpod
)

type localHealth struct {
	reachable  bool
	busy       bool
	queueDepth int
}

type limiter struct {
	mu     sync.Mutex
	tokens float64
	last   time.Time
	rpm    float64
}

func newLimiter(rpm float64) *limiter {
	return &limiter{rpm: rpm, tokens: rpm, last: time.Now()}
}

func (l *limiter) allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	l.tokens += now.Sub(l.last).Minutes() * l.rpm
	if l.tokens > l.rpm {
		l.tokens = l.rpm
	}
	l.last = now
	if l.tokens < 1 {
		return false
	}
	l.tokens--
	return true
}

var runpodLimiter = newLimiter(envFloat("APPNZ_RUNPOD_MAX_RPM", 30))

type AppNZProvider struct {
	apiKey         string
	baseURL        string
	runpodBaseURL  string
	runpodEndpoint string
	runpodAPIKey   string
	maxLocalQueue  int
	minRunpodChars int
	pollInterval   time.Duration
	limiter        *limiter
	client         *http.Client
	healthClient   *http.Client
}

type Option func(*AppNZProvider)

func WithRunpod(baseURL, endpointID, apiKey string) Option {
	return func(p *AppNZProvider) {
		p.runpodBaseURL = strings.TrimRight(baseURL, "/")
		p.runpodEndpoint = endpointID
		p.runpodAPIKey = apiKey
	}
}

func New(apiKey, baseURL string, opts ...Option) *AppNZProvider {
	if baseURL == "" {
		// The omniserve-native front door. Nothing has listened on 9080 since
		// local inference moved behind it, so that default made every appnz
		// call fail on connect and fall through to RunPod.
		baseURL = "http://127.0.0.1:8791"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	p := &AppNZProvider{
		apiKey:         apiKey,
		baseURL:        baseURL,
		runpodBaseURL:  "https://api.runpod.ai",
		runpodEndpoint: os.Getenv("APPNZ_RUNPOD_ENDPOINT_ID"),
		runpodAPIKey:   os.Getenv("APPNZ_RUNPOD_API_KEY"),
		maxLocalQueue:  envInt("APPNZ_MAX_LOCAL_QUEUE", 4),
		minRunpodChars: envInt("APPNZ_RUNPOD_MIN_CHARS", 40),
		pollInterval:   2 * time.Second,
		limiter:        runpodLimiter,
		client:         &http.Client{Timeout: 2 * time.Minute},
		healthClient:   &http.Client{Timeout: 300 * time.Millisecond},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *AppNZProvider) Name() string { return "appnz" }

func (p *AppNZProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	return nil, &provider.ProviderError{Provider: "appnz", StatusCode: 400, Message: "appnz does not support chat", Retryable: false}
}

func (p *AppNZProvider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	return nil, &provider.ProviderError{Provider: "appnz", StatusCode: 400, Message: "appnz does not support chat", Retryable: false}
}

func (p *AppNZProvider) HealthCheck(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("appnz health: status %d", resp.StatusCode)
	}
	return nil
}

func (p *AppNZProvider) GenerateSpeech(ctx context.Context, req *model.SpeechRequest) (*model.SpeechResponse, error) {
	text := req.Input
	if text == "" {
		text = req.Text
	}
	voice := req.Voice
	if voice == "" {
		voice = req.VoiceID
	}

	payload := map[string]any{
		"model":           req.Model,
		"input":           text,
		"response_format": "mp3",
	}
	if voice != "" {
		payload["voice"] = voice
	}
	if req.Speed > 0 {
		payload["speed"] = req.Speed
	}
	if req.Language != "" {
		payload["language"] = req.Language
	}

	h := p.checkHealth(ctx)
	if p.decideRoute(h, len(text)) == routeRunpod {
		return p.runpodSpeech(ctx, payload, len(text))
	}
	return p.localSpeech(ctx, payload, len(text))
}

func (p *AppNZProvider) decideRoute(h localHealth, inputLen int) route {
	if h.reachable && !h.busy && h.queueDepth <= p.maxLocalQueue {
		return routeLocal
	}
	if p.runpodEndpoint == "" || p.runpodAPIKey == "" {
		return routeLocal
	}
	if inputLen < p.minRunpodChars {
		return routeLocal
	}
	if !p.limiter.allow() {
		return routeLocal
	}
	return routeRunpod
}

func (p *AppNZProvider) checkHealth(ctx context.Context) localHealth {
	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/health", nil)
	if err != nil {
		return localHealth{}
	}
	resp, err := p.healthClient.Do(req)
	if err != nil {
		return localHealth{}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return localHealth{}
	}
	var body struct {
		Busy       bool `json:"busy"`
		QueueDepth int  `json:"queue_depth"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return localHealth{reachable: true}
	}
	return localHealth{reachable: true, busy: body.Busy, queueDepth: body.QueueDepth}
}

func (p *AppNZProvider) localSpeech(ctx context.Context, payload map[string]any, chars int) (*model.SpeechResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	respBody, err := p.do(ctx, p.client, "POST", p.baseURL+"/v1/audio/speech", p.apiKey, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return &model.SpeechResponse{
		Audio:      base64.StdEncoding.EncodeToString(respBody),
		Format:     "mp3",
		Characters: chars,
	}, nil
}

type runpodJob struct {
	ID     string        `json:"id"`
	Status string        `json:"status"`
	Output *runpodOutput `json:"output"`
}

type runpodOutput struct {
	AudioB64 string `json:"audio_b64"`
	Format   string `json:"format"`
}

func (p *AppNZProvider) runpodSpeech(ctx context.Context, payload map[string]any, chars int) (*model.SpeechResponse, error) {
	body, err := json.Marshal(map[string]any{"input": payload})
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	respBody, err := p.do(ctx, p.client, "POST", p.runpodBaseURL+"/v2/"+p.runpodEndpoint+"/runsync", p.runpodAPIKey, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var job runpodJob
	if err := json.Unmarshal(respBody, &job); err != nil {
		return nil, &provider.ProviderError{Provider: "appnz", StatusCode: 502, Message: "runpod unmarshal: " + err.Error(), Retryable: true}
	}

	deadline := time.Now().Add(120 * time.Second)
	for job.Status == "IN_QUEUE" || job.Status == "IN_PROGRESS" {
		if time.Now().After(deadline) {
			return nil, &provider.ProviderError{Provider: "appnz", StatusCode: 504, Message: "runpod job " + job.ID + " timed out", Retryable: true}
		}
		select {
		case <-ctx.Done():
			return nil, &provider.ProviderError{Provider: "appnz", StatusCode: 499, Message: ctx.Err().Error(), Retryable: false, Err: ctx.Err()}
		case <-time.After(p.pollInterval):
		}
		respBody, err = p.do(ctx, p.client, "GET", p.runpodBaseURL+"/v2/"+p.runpodEndpoint+"/status/"+job.ID, p.runpodAPIKey, nil)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(respBody, &job); err != nil {
			return nil, &provider.ProviderError{Provider: "appnz", StatusCode: 502, Message: "runpod unmarshal: " + err.Error(), Retryable: true}
		}
	}

	if job.Status != "COMPLETED" || job.Output == nil {
		return nil, &provider.ProviderError{Provider: "appnz", StatusCode: 502, Message: "runpod job status " + job.Status, Retryable: true}
	}
	format := job.Output.Format
	if format == "" {
		format = "mp3"
	}
	return &model.SpeechResponse{
		Audio:      job.Output.AudioB64,
		Format:     format,
		Characters: chars,
	}, nil
}

func (p *AppNZProvider) do(ctx context.Context, client *http.Client, method, url, key string, body io.Reader) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "appnz", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	httpReq.Header.Set("Authorization", "Bearer "+key)
	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "appnz", StatusCode: 502, Message: err.Error(), Retryable: true, Err: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &provider.ProviderError{Provider: "appnz", StatusCode: 502, Message: "read response: " + err.Error(), Retryable: true, Err: err}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &provider.ProviderError{
			Provider:   "appnz",
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
			Retryable:  resp.StatusCode >= 500 || resp.StatusCode == 429,
		}
	}
	return respBody, nil
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return def
}
