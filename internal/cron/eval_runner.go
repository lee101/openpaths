package cron

import (
	"bufio"
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
	"sync/atomic"
	"time"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/model"
)

const (
	// Every sweep is a billed generation across ~8 frontier models x 15 cases,
	// so the default cadence is daily. Override with OPENPATHS_EVALS_INTERVAL.
	evalsInterval     = 24 * time.Hour
	evalsBootDelay    = 2 * time.Minute
	evalsConcurrency  = 3 // models in flight; cases run sequentially within a model
	evalsRequestTO    = 4 * time.Minute
	evalsUserEmail    = "live-evals@openpaths.local"
	evalsUserName     = "Live Evals"
	evalsKeyName      = "live-evals"
	evalsRateLimitRPM = 6000
	evalsMinBalance   = 500  // top up below $5
	evalsTopup        = 5000 // grant $50 of eval credit
)

// evalModels are the competitors rendered on /evals plus openpaths/auto. Ids
// must exist in config.yaml; unknown ids are skipped at run time so a catalogue
// rename cannot break sweeps.
var evalModels = []string{
	"gpt-5.6",
	"gpt-5.5",
	"claude-opus-5",
	"gemini-3.7-flash",
	"deepseek-v4-pro",
	"grok-4.6",
	"glm-5.1",
	"openpaths/auto",
}

type EvalRunner struct {
	evalQ   *queries.EvalQueries
	apiKeyQ *queries.APIKeyQueries
	userQ   *queries.UserQueries
	creditQ *queries.CreditQueries
	models  []model.ModelConfig

	apiKey   string
	apiKeyID string
	baseURL  string
	stop     chan struct{}
	interval time.Duration
	running  atomic.Bool
	client   *http.Client
}

func NewEvalRunner(evalQ *queries.EvalQueries, apiKeyQ *queries.APIKeyQueries, userQ *queries.UserQueries, creditQ *queries.CreditQueries, models []model.ModelConfig) *EvalRunner {
	if evalQ == nil {
		return nil
	}
	apiKey := strings.TrimSpace(os.Getenv("OPENPATHS_EVALS_API_KEY"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("OPENPATHS_PROBE_API_KEY"))
	}
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("OPENPATHS_EVALS_BASE_URL")), "/")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8092"
	}
	interval := evalsInterval
	if v := strings.TrimSpace(os.Getenv("OPENPATHS_EVALS_INTERVAL")); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > time.Minute {
			interval = d
		}
	}
	r := &EvalRunner{
		evalQ:   evalQ,
		apiKeyQ: apiKeyQ,
		userQ:   userQ,
		creditQ: creditQ,
		models:  models,
		apiKey:  apiKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: evalsRequestTO},
		stop:    make(chan struct{}),
	}
	r.interval = interval
	return r
}

// Start launches the daily sweep loop. No-op when disabled.
func (r *EvalRunner) Start() {
	if r == nil {
		return
	}
	if strings.TrimSpace(os.Getenv("OPENPATHS_EVALS_DISABLED")) == "1" {
		log.Printf("live-evals: disabled by OPENPATHS_EVALS_DISABLED")
		return
	}
	go func() {
		select {
		case <-time.After(evalsBootDelay):
		case <-r.stop:
			return
		}
		r.runSweep(context.Background())
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				r.runSweep(context.Background())
			case <-r.stop:
				return
			}
		}
	}()
	log.Printf("live-evals: scheduled every %s (%d models x %d cases)", r.interval, len(evalModels), len(evalCases()))
}

func (r *EvalRunner) Stop() {
	if r == nil {
		return
	}
	select {
	case <-r.stop:
	default:
		close(r.stop)
	}
}

// Running reports whether a sweep is currently in flight.
func (r *EvalRunner) Running() bool {
	if r == nil {
		return false
	}
	return r.running.Load()
}


// Done exposes the stop channel so CLI callers can block until shutdown.
func (r *EvalRunner) Done() <-chan struct{} {
	return r.stop
}
// RunAsync triggers one sweep if none is running. Returns false when a sweep is
// already in flight.
func (r *EvalRunner) RunAsync() bool {
	if r == nil || !r.running.CompareAndSwap(false, true) {
		return false
	}
	go func() {
		defer r.running.Store(false)
		r.runSweep(context.Background())
	}()
	return true
}

type streamOutcome struct {
	content    strings.Builder
	toolNames  []string // streamed tool-call deltas, best effort
	ttftMs     int
	totalMs    int
	promptTok  int
	complTok   int
	hasUsage   bool
	statusCode int
	errMsg     string
}

func (r *EvalRunner) ensureKey(ctx context.Context) error {
	if r.apiKey != "" {
		return nil
	}
	raw, id, err := EnsureServiceKey(ctx, ServiceKeyDeps{APIKeyQ: r.apiKeyQ, UserQ: r.userQ, CreditQ: r.creditQ},
		evalsUserEmail, evalsUserName, evalsKeyName, evalsRateLimitRPM, evalsMinBalance, evalsTopup)
	if err != nil {
		return err
	}
	r.apiKey = raw
	r.apiKeyID = id
	return nil
}

func (r *EvalRunner) targets() []string {
	known := make(map[string]bool, len(r.models))
	for _, m := range r.models {
		known[m.ID] = true
	}
	out := make([]string, 0, len(evalModels))
	for _, id := range evalModels {
		if known[id] {
			out = append(out, id)
		} else {
			log.Printf("live-evals: skipping unknown model %q", id)
		}
	}
	return out
}

func (r *EvalRunner) runSweep(ctx context.Context) {
	if err := r.ensureKey(ctx); err != nil {
		log.Printf("live-evals: cannot run (%v)", err)
		return
	}
	targets := r.targets()
	if len(targets) == 0 {
		return
	}
	sweepStart := time.Now()
	log.Printf("live-evals: sweep starting (%d models, %d cases)", len(targets), len(evalCases()))

	sem := make(chan struct{}, evalsConcurrency)
	var wg sync.WaitGroup
	for _, modelID := range targets {
		modelID := modelID
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()
			for _, c := range evalCases() {
				result := r.runCase(ctx, modelID, c)
				if err := r.evalQ.Upsert(ctx, result); err != nil {
					log.Printf("live-evals: store %s/%s/%s: %v", c.Suite, c.ID, modelID, err)
				}
			}
		}()
	}
	wg.Wait()
	log.Printf("live-evals: sweep finished in %s", time.Since(sweepStart).Round(time.Second))
}

func (r *EvalRunner) buildMessages(c evalCase) []model.ChatMessage {
	var msgs []model.ChatMessage
	if c.System != "" {
		msgs = append(msgs, model.ChatMessage{Role: "system", Content: c.System})
	}
	if len(c.Messages) > 0 {
		msgs = append(msgs, c.Messages...)
	} else {
		msgs = append(msgs, model.ChatMessage{Role: "user", Content: c.Prompt})
	}
	return msgs
}

// runCase executes one case against one model and returns the stored result.
// Errors are recorded as failed rows rather than dropped so the snapshot shows
// outages.
func (r *EvalRunner) runCase(ctx context.Context, modelID string, c evalCase) model.EvalResult {
	start := time.Now()
	res := model.EvalResult{Suite: string(c.Suite), CaseID: c.ID, Model: modelID}

	body := map[string]any{
		"model":      modelID,
		"messages":   r.buildMessages(c),
		"max_tokens": c.MaxTok,
	}
	streaming := len(c.Tools) == 0
	body["stream"] = streaming
	if len(c.Tools) > 0 {
		body["tools"] = c.Tools
	}
	payload, _ := json.Marshal(body)

	reqCtx, cancel := context.WithTimeout(ctx, evalsRequestTO)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, r.baseURL+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		res.Error = strPtr(err.Error())
		res.RanAt = time.Now()
		return res
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("HTTP-Referer", "https://openpaths.io/evals")
	req.Header.Set("X-Title", "OpenPaths Live Evals")

	resp, err := r.client.Do(req)
	if err != nil {
		res.Error = strPtr(err.Error())
		res.RanAt = time.Now()
		return res
	}
	defer resp.Body.Close()
	res.TotalMs = int(time.Since(start).Milliseconds())

	if resp.StatusCode != http.StatusOK {
		msg := truncatePreview(string(readLimited(resp.Body, 8*1024)), 200)
		res.Error = strPtr(fmt.Sprintf("HTTP %d: %s", resp.StatusCode, msg))
		res.RanAt = time.Now()
		return res
	}

	var content string
	var toolCalls []model.ToolCall
	if streaming {
		out := readStream(resp.Body, start)
		content = out.content.String()
		res.TTFTMs = out.ttftMs
		if out.hasUsage {
			res.PromptTokens = out.promptTok
			res.CompletionTokens = out.complTok
		} else if in, outT, _, _, _, _, found, qerr := r.evalQ.LatestUsage(reqCtx, r.apiKeyID, modelID, start); qerr == nil && found {
			res.PromptTokens, res.CompletionTokens = in, outT
		} else if qerr != nil {
			log.Printf("live-evals: usage reconcile %s/%s: %v", c.ID, modelID, qerr)
		}
		if out.errMsg != "" {
			res.Error = strPtr(out.errMsg)
		}
	} else {
		respBytes := readLimited(resp.Body, 512*1024)
		var parsed model.ChatCompletionResponse
		if err := json.Unmarshal(respBytes, &parsed); err != nil {
			res.Error = strPtr("invalid JSON response")
			res.RanAt = time.Now()
			return res
		}
		if len(parsed.Choices) > 0 && parsed.Choices[0].Message != nil {
			content = messageText(parsed.Choices[0].Message.Content)
			toolCalls = parsed.Choices[0].Message.ToolCalls
		}
		if parsed.Usage != nil {
			res.PromptTokens = parsed.Usage.PromptTokens
			res.CompletionTokens = parsed.Usage.CompletionTokens
		}
	}

	score := c.grade(content, toolCalls)
	res.Score = score
	res.Passed = score >= 0.999 && res.Error == nil
	res.TokensPerSec = tps(res.CompletionTokens, start, res.TotalMs, res.TTFTMs)
	res.CostMicroUSD = r.caseCostMicroUSD(modelID, res.PromptTokens, res.CompletionTokens)
	res.AnswerPreview = truncatePreview(content, 160)
	res.RanAt = time.Now()
	return res
}

// readStream consumes an SSE body, assembling text deltas and capturing TTFT
// and any terminal usage frame.
func readStream(body io.Reader, start time.Time) streamOutcome {
	out := streamOutcome{}
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var argBuf strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk model.ChatCompletionChunk
		if json.Unmarshal([]byte(data), &chunk) != nil {
			continue
		}
		if chunk.Usage != nil {
			out.promptTok = chunk.Usage.PromptTokens
			out.complTok = chunk.Usage.CompletionTokens
			out.hasUsage = true
		}
		for _, choice := range chunk.Choices {
			if choice.Delta == nil {
				continue
			}
			if txt := messageText(choice.Delta.Content); txt != "" {
				if out.ttftMs == 0 {
					out.ttftMs = int(time.Since(start).Milliseconds())
				}
				out.content.WriteString(txt)
			}
			for _, tc := range choice.Delta.ToolCalls {
				argBuf.WriteString(tc.Function.Arguments)
				if tc.Function.Name != "" {
					out.toolNames = append(out.toolNames, tc.Function.Name)
				}
			}
		}
	}
	// A streaming tool call arrived but no plain content: grade from args.
	if out.content.Len() == 0 && argBuf.Len() > 0 {
		out.content.WriteString(argBuf.String())
	}
	out.totalMs = int(time.Since(start).Milliseconds())
	return out
}

func tps(completionTokens int, start time.Time, totalMs, ttftMs int) float64 {
	if completionTokens <= 0 {
		return 0
	}
	windowMs := totalMs - ttftMs
	if windowMs <= 0 {
		windowMs = totalMs
	}
	if windowMs <= 0 {
		return 0
	}
	return float64(completionTokens) / (float64(windowMs) / 1000)
}

// caseCostMicroUSD prices tokens at catalogue list prices. $X per 1M tokens is
// exactly X micro-dollars per token.
func (r *EvalRunner) caseCostMicroUSD(modelID string, promptTokens, completionTokens int) int64 {
	var inPrice, outPrice float64
	for _, m := range r.models {
		if m.ID == modelID {
			inPrice, outPrice = m.InputPricePer1M, m.OutputPricePer1M
			break
		}
	}
	cost := float64(promptTokens)*inPrice + float64(completionTokens)*outPrice
	if cost < 0 {
		return 0
	}
	return int64(cost + 0.5)
}

func readLimited(body io.Reader, max int64) []byte {
	b, _ := io.ReadAll(io.LimitReader(body, max))
	return b
}

func messageText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func strPtr(s string) *string { return &s }
