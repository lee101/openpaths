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
	// Liveness only: the shortest prompt that still forces a real generation,
	// with a hard cap and stop sequences so a chatty model cannot run away. A
	// reasoning model may spend the whole budget on hidden tokens and return
	// empty content -- that still proves the upstream is alive, so it counts as
	// OK (see probeSucceeded) rather than needing a large budget.
	probePrompt    = "say hi, nothing else"
	probeReferer   = "https://openpaths.io/stats"
	probeTitle     = "OpenPaths Model Probe"
	probeMaxTokens = 64
	// probeInterval is how often the loop wakes; probeDueInterval /
	// probeStableDueInterval decide which models are actually due on that tick.
	// A model that failed its last probe is retried daily so upstream breakage
	// is caught fast; one that is healthy is re-checked far less often, since
	// every probe is a billed generation against the live provider key.
	probeInterval          = 24 * time.Hour
	probeDueInterval       = 24 * time.Hour
	probeStableDueInterval = 7 * 24 * time.Hour
	probeConcurrency       = 2
	probeSlowTimeout       = 12 * time.Minute
	probeFastTimeout       = 3 * time.Minute

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
// supplied via env, it provisions a dedicated service user via EnsureServiceKey.
func (p *ModelProber) ensureProbeKey(ctx context.Context) error {
	if p.apiKey != "" {
		return nil
	}
	raw, _, err := EnsureServiceKey(ctx, ServiceKeyDeps{APIKeyQ: p.apiKeyQ, UserQ: p.userQ, CreditQ: p.creditQ},
		probeUserEmail, probeUserName, probeKeyName, probeRateLimitRPM, probeMinBalanceCents, probeTopupCents)
	if err != nil {
		return err
	}
	p.apiKey = raw
	return nil
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
	go p.loop()
	log.Printf("model-probe: started (%d chat models, base=%s)", len(targets), p.baseURL)
}

func (p *ModelProber) Stop() {
	select {
	case <-p.stop:
	default:
		close(p.stop)
	}
}

// RunOnce probes every model regardless of when it was last probed. This is the
// manual/CLI path (cmd/probe-models), where the caller explicitly wants a full
// sweep; the background loop uses the due-check in run instead.
func (p *ModelProber) RunOnce() {
	keyCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err := p.ensureProbeKey(keyCtx)
	cancel()
	if err != nil {
		log.Printf("model-probe: cannot run (%v)", err)
		return
	}
	p.run(true)
}

func (p *ModelProber) loop() {
	time.Sleep(30 * time.Second)
	p.run(false)
	ticker := time.NewTicker(probeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.run(false)
		case <-p.stop:
			return
		}
	}
}

// dueInterval reports how stale a model's last probe may be before it is
// re-probed. Failing models are retried daily, healthy ones weekly.
func dueInterval(prev model.ModelProbeResult) time.Duration {
	if prev.OK {
		return probeStableDueInterval
	}
	return probeDueInterval
}

// dueTargets filters targets down to the models whose last stored probe is
// older than their cadence. Models never probed before are always due. This is
// what makes a process restart cheap: without it, every boot re-probed the full
// catalogue, which is a billed generation per model.
func (p *ModelProber) dueTargets(ctx context.Context, targets []model.ModelConfig) []model.ModelConfig {
	prior, err := p.probeQ.List(ctx)
	if err != nil {
		log.Printf("model-probe: cannot read prior results, skipping this cycle: %v", err)
		return nil
	}
	last := make(map[string]model.ModelProbeResult, len(prior))
	for _, r := range prior {
		last[r.Model] = r
	}
	return filterDue(targets, last, time.Now())
}

func filterDue(targets []model.ModelConfig, last map[string]model.ModelProbeResult, now time.Time) []model.ModelConfig {
	due := make([]model.ModelConfig, 0, len(targets))
	for _, cfg := range targets {
		prev, seen := last[cfg.ID]
		if !seen || now.Sub(prev.ProbedAt) >= dueInterval(prev) {
			due = append(due, cfg)
		}
	}
	return due
}

func (p *ModelProber) run(force bool) {
	targets := ChatProbeModels(p.models)
	if len(targets) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
	defer cancel()

	total := len(targets)
	if !force {
		targets = p.dueTargets(ctx, targets)
		if len(targets) == 0 {
			log.Printf("model-probe: nothing due (%d models up to date)", total)
			return
		}
	}

	log.Printf("model-probe: probing %d of %d models...", len(targets), total)
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
		"stop":       []string{"\n\n"},
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
		Usage struct {
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		msg := "invalid JSON response"
		result.Error = &msg
		return result
	}
	content := ""
	if len(parsed.Choices) > 0 {
		content = strings.TrimSpace(parsed.Choices[0].Message.Content)
	}
	// A reasoning model can spend the whole (small) budget on hidden tokens and
	// come back with empty content. It answered, so it is alive -- flagging it
	// as failed just hides the providers that are genuinely down.
	if !probeSucceeded(len(parsed.Choices), content, parsed.Usage.CompletionTokens) {
		msg := "empty assistant content"
		result.Error = &msg
		return result
	}

	result.OK = true
	if content == "" {
		result.ResponsePreview = "(reasoning only, no visible content)"
	} else {
		result.ResponsePreview = truncatePreview(content, 120)
	}
	return result
}

func probeSucceeded(choiceCount int, content string, completionTokens int) bool {
	return choiceCount > 0 && (strings.TrimSpace(content) != "" || completionTokens > 0)
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
	if cfg.ID == "" || cfg.Deprecated {
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
