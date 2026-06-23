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

	"github.com/openpaths/openpaths/internal/auth"
	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

const (
	probePrompt       = "say hi"
	probeReferer      = "https://openpaths.io/stats"
	probeTitle        = "OpenPaths Model Probe"
	probeMaxTokens    = 512 // generous so reasoning models emit content instead of spending the whole budget on hidden reasoning tokens
	probeConcurrency  = 2
	probeSlowTimeout  = 12 * time.Minute
	probeFastTimeout  = 3 * time.Minute

	// Auto-provisioned probe identity. The prober calls the gateway through the
	// normal /v1/chat/completions path, so it needs a valid op- API key and a
	// credit balance (BalanceCheck blocks zero-balance users). When no key is
	// supplied via env, we mint one for this dedicated service user.
	probeUserEmail       = "model-probe@openpaths.local"
	probeUserName        = "Model Probe"
	probeKeyName         = "model-probe"
	probeMinBalanceCents = 500  // top up when balance drops below $5
	probeTopupCents      = 2000 // grant $20 of probe credit
	probeRateLimitRPM    = 6000 // probes fan out across all models; avoid 429 throttling
)

type ModelProber struct {
	probeQ  *queries.ModelProbeQueries
	apiKeyQ *queries.APIKeyQueries
	userQ   *queries.UserQueries
	creditQ *queries.CreditQueries
	models  []model.ModelConfig
	apiKey  string
	baseURL string
	client  *http.Client
	stop    chan struct{}
}

func NewModelProber(probeQ *queries.ModelProbeQueries, apiKeyQ *queries.APIKeyQueries, userQ *queries.UserQueries, creditQ *queries.CreditQueries, models []model.ModelConfig) *ModelProber {
	// Explicit override only. NOTE: we intentionally do NOT fall back to
	// APP_API_KEY here -- that env var holds an upstream provider key
	// (e.g. the papers key), not a valid op- gateway key, and using it makes
	// every probe fail with 401 invalid_api_key.
	apiKey := strings.TrimSpace(os.Getenv("OPENPATHS_PROBE_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("OPENPATHS_API_KEY"))
	}
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("OPENPATHS_PROBE_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8092"
	}
	return &ModelProber{
		probeQ:  probeQ,
		apiKeyQ: apiKeyQ,
		userQ:   userQ,
		creditQ: creditQ,
		models:  models,
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: probeSlowTimeout},
		stop:    make(chan struct{}),
	}
}

// ensureProbeKey guarantees the prober has a valid op- API key. When none is
// supplied via env, it gets-or-creates a dedicated service user, ensures that
// user holds enough credit to pass BalanceCheck, and mints a fresh key.
func (p *ModelProber) ensureProbeKey(ctx context.Context) error {
	if p.apiKey != "" {
		return nil
	}
	if p.apiKeyQ == nil || p.userQ == nil {
		return fmt.Errorf("no probe API key configured and key/user queries unavailable")
	}

	user, err := p.userQ.GetByEmail(ctx, probeUserEmail)
	if err != nil {
		pwHash, herr := auth.HashPassword(randomProbeSecret())
		if herr != nil {
			return fmt.Errorf("hash probe password: %w", herr)
		}
		user, err = p.userQ.Create(ctx, probeUserEmail, pwHash, probeUserName)
		if err != nil {
			return fmt.Errorf("create probe user: %w", err)
		}
	}

	// Ensure the probe user has credit so requests reach the server's
	// configured provider keys instead of being rejected by BalanceCheck.
	if p.creditQ != nil {
		if ierr := p.creditQ.InitBalance(ctx, user.ID); ierr != nil {
			log.Printf("model-probe: init balance: %v", ierr)
		}
		if bal, berr := p.creditQ.GetBalance(ctx, user.ID); berr == nil && bal < probeMinBalanceCents {
			if derr := p.creditQ.Deposit(ctx, user.ID, probeTopupCents, "model probe credit grant"); derr != nil {
				log.Printf("model-probe: credit grant: %v", derr)
			}
		}
	}

	// Revoke any stale probe keys from previous boots so live keys don't pile up.
	if existing, lerr := p.apiKeyQ.ListByUser(ctx, user.ID); lerr == nil {
		for _, k := range existing {
			if k.Name == probeKeyName && !k.Revoked {
				if rerr := p.apiKeyQ.Revoke(ctx, k.ID, user.ID); rerr != nil {
					log.Printf("model-probe: revoke stale key: %v", rerr)
				}
			}
		}
	}

	raw, hash, prefix, err := auth.GenerateAPIKey()
	if err != nil {
		return fmt.Errorf("generate probe key: %w", err)
	}
	key, err := p.apiKeyQ.Create(ctx, user.ID, hash, prefix, probeKeyName)
	if err != nil {
		return fmt.Errorf("store probe key: %w", err)
	}
	// Probes fan out across every chat model; lift the default 60 rpm cap so the
	// run isn't throttled into 429s.
	if rerr := p.apiKeyQ.SetRateLimit(ctx, key.ID, probeRateLimitRPM); rerr != nil {
		log.Printf("model-probe: set rate limit: %v", rerr)
	}
	p.apiKey = raw
	return nil
}

func randomProbeSecret() string {
	// Reuse the API-key generator purely as a source of crypto-random bytes for
	// the service-account password (it can never be used to log in interactively).
	raw, _, _, err := auth.GenerateAPIKey()
	if err != nil {
		return "model-probe-no-login"
	}
	return raw
}

func (p *ModelProber) Start() {
	if p.probeQ == nil {
		log.Printf("model-probe: disabled (missing probe queries)")
		return
	}
	if os.Getenv("OPENPATHS_MODEL_PROBE_DISABLED") == "1" {
		log.Printf("model-probe: disabled via OPENPATHS_MODEL_PROBE_DISABLED=1")
		return
	}
	keyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err := p.ensureProbeKey(keyCtx)
	cancel()
	if err != nil {
		log.Printf("model-probe: disabled (%v)", err)
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
	keyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err := p.ensureProbeKey(keyCtx)
	cancel()
	if err != nil {
		log.Printf("model-probe: cannot run (%v)", err)
		return
	}
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
	// Embedding models have input pricing and a context window but produce no
	// generated output (output price 0, max_output_tokens 0). Probing them on
	// /v1/chat/completions just yields 400/404, so exclude them.
	if cfg.OutputPricePer1M <= 0 || cfg.MaxOutputTokens <= 0 {
		return false
	}
	// Codex models are only served on OpenAI's /v1/responses endpoint, not
	// /v1/chat/completions -- a chat probe always 404s.
	if strings.Contains(cfg.ID, "codex") {
		return false
	}
	if cfg.PricePerImage > 0 || cfg.PricePerVideo > 0 || cfg.PricePerSecond > 0 || cfg.PricePerMinute > 0 {
		return false
	}
	return true
}

func truncatePreview(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
