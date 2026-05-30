package cron

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

const (
	probePrompt       = "say hi"
	probeReferer      = "https://openpaths.io/stats"
	probeTitle        = "OpenPaths Model Probe"
	probeMaxTokens    = 24
	probeConcurrency  = 2
	probeSlowTimeout  = 12 * time.Minute
	probeFastTimeout  = 3 * time.Minute
)

type ModelProber struct {
	probeQ  *queries.ModelProbeQueries
	models  []model.ModelConfig
	apiKey  string
	baseURL string
	client  *http.Client
	stop    chan struct{}
}

func NewModelProber(probeQ *queries.ModelProbeQueries, models []model.ModelConfig) *ModelProber {
	apiKey := strings.TrimSpace(os.Getenv("OPENPATHS_PROBE_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("OPENPATHS_API_KEY"))
	}
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("APP_API_KEY"))
	}
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("OPENPATHS_PROBE_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8092"
	}
	return &ModelProber{
		probeQ:  probeQ,
		models:  models,
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: probeSlowTimeout},
		stop:    make(chan struct{}),
	}
}

func (p *ModelProber) Start() {
	if p.probeQ == nil || p.apiKey == "" {
		log.Printf("model-probe: disabled (missing probe queries or probe API key)")
		return
	}
	if os.Getenv("OPENPATHS_MODEL_PROBE_DISABLED") == "1" {
		log.Printf("model-probe: disabled via OPENPATHS_MODEL_PROBE_DISABLED=1")
		return
	}
	targets := ChatProbeModels(p.models)
	if len(targets) == 0 {
		log.Printf("model-probe: no chat models configured")
		return
	}
	go p.loop(len(targets))
	log.Printf("model-probe: started (%d chat models, base=%s)", len(targets), p.baseURL)
}

func (p *ModelProber) Stop() {
	select {
	case <-p.stop:
	default:
		close(p.stop)
	}
}

func (p *ModelProber) RunOnce() {
	p.run()
}

func (p *ModelProber) loop(modelCount int) {
	time.Sleep(30 * time.Second)
	p.run()
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.run()
		case <-p.stop:
			return
		}
	}
}

func (p *ModelProber) run() {
	targets := ChatProbeModels(p.models)
	if len(targets) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
	defer cancel()

	log.Printf("model-probe: probing %d models...", len(targets))
	started := time.Now()

	sem := make(chan struct{}, probeConcurrency)
	var wg sync.WaitGroup
	var okCount, failCount int
	var mu sync.Mutex

	for _, cfg := range targets {
		cfg := cfg
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			result := p.probeOne(ctx, cfg)
			if err := p.probeQ.Upsert(ctx, result); err != nil {
				log.Printf("model-probe: store %s: %v", cfg.ID, err)
			}
			mu.Lock()
			if result.OK {
				okCount++
			} else {
				failCount++
				if result.Error != nil {
					log.Printf("model-probe: FAIL %s (%dms): %s", cfg.ID, result.LatencyMs, *result.Error)
				}
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
	log.Printf("model-probe: finished in %s (%d ok, %d failed)", time.Since(started).Round(time.Second), okCount, failCount)
}

func (p *ModelProber) probeOne(ctx context.Context, cfg model.ModelConfig) model.ModelProbeResult {
	now := time.Now()
	result := model.ModelProbeResult{
		Model:    cfg.ID,
		Provider: cfg.Provider,
		ProbedAt: now,
	}

	body, _ := json.Marshal(map[string]any{
		"model": cfg.ID,
		"messages": []map[string]string{
			{"role": "user", "content": probePrompt},
		},
		"max_tokens": probeMaxTokens,
	})

	timeout := probeFastTimeout
	if strings.HasPrefix(cfg.Provider, "cursor") || strings.Contains(cfg.ID, "composer") {
		timeout = probeSlowTimeout
	}

	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		msg := err.Error()
		result.Error = &msg
		result.LatencyMs = int(time.Since(now).Milliseconds())
		return result
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("HTTP-Referer", probeReferer)
	req.Header.Set("X-Title", probeTitle)

	resp, err := p.client.Do(req)
	result.LatencyMs = int(time.Since(now).Milliseconds())
	if err != nil {
		msg := err.Error()
		result.Error = &msg
		return result
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	result.StatusCode = resp.StatusCode

	if resp.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(string(respBody))
		if len(msg) > 240 {
			msg = msg[:240] + "..."
		}
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		result.Error = &msg
		return result
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		msg := "invalid JSON response"
		result.Error = &msg
		return result
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		msg := "empty assistant content"
		result.Error = &msg
		return result
	}

	result.OK = true
	result.ResponsePreview = truncatePreview(parsed.Choices[0].Message.Content, 120)
	return result
}

func ChatProbeModels(models []model.ModelConfig) []model.ModelConfig {
	out := make([]model.ModelConfig, 0, len(models))
	for _, cfg := range models {
		if IsChatProbeModel(cfg) {
			out = append(out, cfg)
		}
	}
	return out
}

func IsChatProbeModel(cfg model.ModelConfig) bool {
	if cfg.ID == "" {
		return false
	}
	hasTokenPricing := cfg.InputPricePer1M > 0 || cfg.OutputPricePer1M > 0
	if !hasTokenPricing {
		return false
	}
	if cfg.PricePerImage > 0 || cfg.PricePerVideo > 0 || cfg.PricePerSecond > 0 || cfg.PricePerMinute > 0 {
		return false
	}
	return cfg.ContextWindow > 0 || cfg.MaxOutputTokens > 0
}

func truncatePreview(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
