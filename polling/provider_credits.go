// Package polling contains low-cost operational checks that run independently
// of customer traffic and the gateway fallback router.
package polling

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

const (
	defaultAlertEmail = "leepenkman@gmail.com"
	defaultTimezone   = "Pacific/Auckland"
	defaultRunHour    = 8
	defaultRunMinute  = 0
	defaultTimeout    = 90 * time.Second
	probePrompt       = "say hi nothing else"
	probeMaxTokens    = 16
)

// targetSpecs are intentionally curated rather than inferred solely from
// catalogue prices. A free model may keep working after a provider's paid
// balance is exhausted, while an auto/fallback model can hide a failed primary
// provider. Each target below is a cheap paid chat route called directly.
var targetSpecs = []struct {
	Provider string
	ModelID  string
}{
	{Provider: "xai", ModelID: "grok-4.6"},
	{Provider: "openai", ModelID: "gpt-5.4-nano"},
	{Provider: "google", ModelID: "gemini-3.1-flash-lite"},
	{Provider: "anthropic", ModelID: "claude-haiku-4-5-20251001"},
	{Provider: "deepseek", ModelID: "deepseek-chat"},
	{Provider: "mistral", ModelID: "open-mistral-nemo"},
	{Provider: "groq", ModelID: "llama-3.1-8b-instant"},
	{Provider: "openrouter", ModelID: "openpaths/chat-latest"},
	{Provider: "together", ModelID: "together/deepseek-v3.1"},
	{Provider: "minimax", ModelID: "minimax-m2"},
	{Provider: "zai", ModelID: "glm-4.6v-flashx"},
	{Provider: "nous", ModelID: "hermes-4-70b"},
	{Provider: "fireworks", ModelID: "fireworks/gpt-oss-120b"},
	{Provider: "nvidia", ModelID: "nvidia/deepseek-v3.2"},
	{Provider: "cursor", ModelID: "composer-2.5"},
}

type providerRegistry interface {
	Get(name string) (provider.Provider, error)
}

type emailSender func(toEmail, subject, htmlBody string) error

type Target struct {
	Provider        string
	ModelID         string
	ProviderModelID string
}

type Result struct {
	Target     Target
	OK         bool
	CreditFail bool
	StatusCode int
	Error      string
	Latency    time.Duration
}

type Summary struct {
	StartedAt time.Time
	Results   []Result
}

type Monitor struct {
	registry   providerRegistry
	targets    []Target
	sendEmail  emailSender
	alertEmail string
	location   *time.Location
	runHour    int
	runMinute  int
	timeout    time.Duration
	now        func() time.Time
	stop       chan struct{}
	stopOnce   sync.Once
}

// NewProviderCreditMonitor creates a direct-provider balance monitor. Missing
// providers (disabled or lacking an API key) are skipped rather than alerted;
// key/configuration failures remain visible in logs but do not masquerade as a
// credit exhaustion alert.
func NewProviderCreditMonitor(registry providerRegistry, models []model.ModelConfig, send emailSender) *Monitor {
	location, err := time.LoadLocation(envOr("OPENPATHS_CREDIT_POLL_TIMEZONE", defaultTimezone))
	if err != nil {
		log.Printf("provider-credit-poll: invalid timezone, using %s: %v", defaultTimezone, err)
		location, _ = time.LoadLocation(defaultTimezone)
	}
	hour, minute := parseClock(envOr("OPENPATHS_CREDIT_POLL_TIME", "08:00"))
	timeout := defaultTimeout
	if raw := strings.TrimSpace(os.Getenv("OPENPATHS_CREDIT_POLL_TIMEOUT")); raw != "" {
		if parsed, parseErr := time.ParseDuration(raw); parseErr == nil && parsed > 0 {
			timeout = parsed
		} else {
			log.Printf("provider-credit-poll: invalid timeout %q, using %s", raw, defaultTimeout)
		}
	}

	return &Monitor{
		registry:   registry,
		targets:    configuredTargets(registry, models),
		sendEmail:  send,
		alertEmail: envOr("OPENPATHS_CREDIT_POLL_EMAIL", defaultAlertEmail),
		location:   location,
		runHour:    hour,
		runMinute:  minute,
		timeout:    timeout,
		now:        time.Now,
		stop:       make(chan struct{}),
	}
}

func (m *Monitor) Start() {
	if m == nil || os.Getenv("OPENPATHS_CREDIT_POLL_DISABLED") == "1" {
		log.Printf("provider-credit-poll: disabled")
		return
	}
	if m.registry == nil || m.sendEmail == nil || len(m.targets) == 0 {
		log.Printf("provider-credit-poll: disabled (registry, email sender, or targets unavailable)")
		return
	}
	next := nextDailyRun(m.now(), m.location, m.runHour, m.runMinute)
	log.Printf("provider-credit-poll: %d providers scheduled daily at %02d:%02d %s (next %s)",
		len(m.targets), m.runHour, m.runMinute, m.location, next.Format(time.RFC3339))
	go m.loop()
}

func (m *Monitor) Stop() {
	if m == nil {
		return
	}
	m.stopOnce.Do(func() { close(m.stop) })
}

func (m *Monitor) loop() {
	for {
		next := nextDailyRun(m.now(), m.location, m.runHour, m.runMinute)
		delay := time.Until(next)
		if delay < 0 {
			delay = 0
		}
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
			m.RunOnce(context.Background())
		case <-m.stop:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		}
	}
}

// RunOnce performs the direct upstream checks and sends one combined email if
// any provider explicitly reports exhausted credit, balance, billing, spend,
// or account quota. Other failures are logged but do not page the recipient.
func (m *Monitor) RunOnce(ctx context.Context) Summary {
	summary := Summary{StartedAt: m.now()}
	for _, target := range m.targets {
		result := m.probe(ctx, target)
		summary.Results = append(summary.Results, result)
		if result.OK {
			log.Printf("provider-credit-poll: OK %s/%s (%s)", target.Provider, target.ModelID, result.Latency.Round(time.Millisecond))
		} else if result.CreditFail {
			log.Printf("provider-credit-poll: CREDIT %s/%s: %s", target.Provider, target.ModelID, result.Error)
		} else {
			log.Printf("provider-credit-poll: non-credit failure %s/%s: %s", target.Provider, target.ModelID, result.Error)
		}
	}

	failures := creditFailures(summary.Results)
	if len(failures) == 0 {
		return summary
	}
	subject, body := alertEmail(summary.StartedAt.In(m.location), failures)
	if err := m.sendEmail(m.alertEmail, subject, body); err != nil {
		log.Printf("provider-credit-poll: alert email to %s failed: %v", m.alertEmail, err)
	}
	return summary
}

func (m *Monitor) probe(parent context.Context, target Target) Result {
	started := m.now()
	result := Result{Target: target}
	p, err := m.registry.Get(target.Provider)
	if err != nil {
		result.Error = err.Error()
		result.Latency = m.now().Sub(started)
		return result
	}

	maxTokens := probeMaxTokens
	ctx, cancel := context.WithTimeout(parent, m.timeout)
	defer cancel()
	_, err = p.ChatCompletion(ctx, &model.ChatCompletionRequest{
		Model: target.ProviderModelID,
		Messages: []model.ChatMessage{
			{Role: "user", Content: probePrompt},
		},
		MaxTokens: &maxTokens,
	})
	result.Latency = m.now().Sub(started)
	if err == nil {
		result.OK = true
		return result
	}

	result.Error = truncate(strings.TrimSpace(err.Error()), 1200)
	var providerErr *provider.ProviderError
	if errors.As(err, &providerErr) {
		result.StatusCode = providerErr.StatusCode
		result.CreditFail = isCreditFailure(providerErr.StatusCode, providerErr.Message)
	} else {
		result.CreditFail = isCreditFailure(0, result.Error)
	}
	return result
}

func configuredTargets(registry providerRegistry, models []model.ModelConfig) []Target {
	byID := make(map[string]model.ModelConfig, len(models))
	for _, cfg := range models {
		byID[cfg.ID] = cfg
	}
	targets := make([]Target, 0, len(targetSpecs))
	for _, spec := range targetSpecs {
		if registry == nil {
			break
		}
		if _, err := registry.Get(spec.Provider); err != nil {
			continue
		}
		cfg, ok := byID[spec.ModelID]
		if !ok || cfg.Deprecated || cfg.Provider != spec.Provider || cfg.ProviderModelID == "" {
			log.Printf("provider-credit-poll: skipping invalid target %s/%s", spec.Provider, spec.ModelID)
			continue
		}
		targets = append(targets, Target{Provider: spec.Provider, ModelID: spec.ModelID, ProviderModelID: cfg.ProviderModelID})
	}
	return targets
}

func isCreditFailure(status int, message string) bool {
	text := strings.ToLower(message)
	markers := []string{
		"all credits", "credit balance", "credits exhausted", "credit exhausted",
		"insufficient credit", "not enough credit", "no credits", "out of credits",
		"insufficient balance", "prepaid balance", "insufficient_quota", "insufficient quota",
		"billing quota", "billing limit", "billing issue", "billing hard limit",
		"monthly spending limit", "spending limit", "spend limit",
		"payment required", "add funds", "top up", "top-up",
		"account balance", "balance is too low", "quota exceeded", "resource_exhausted",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	// HTTP 402 is unambiguously payment-related even when a provider returns a
	// terse or empty response body. A plain 403/429 is deliberately not enough:
	// those also represent auth, safety, RPM, and transient rate limits.
	return status == 402
}

func creditFailures(results []Result) []Result {
	failures := make([]Result, 0)
	for _, result := range results {
		if result.CreditFail {
			failures = append(failures, result)
		}
	}
	sort.Slice(failures, func(i, j int) bool { return failures[i].Target.Provider < failures[j].Target.Provider })
	return failures
}

func alertEmail(at time.Time, failures []Result) (string, string) {
	providers := make([]string, 0, len(failures))
	var rows strings.Builder
	for _, failure := range failures {
		providers = append(providers, failure.Target.Provider)
		fmt.Fprintf(&rows, "<tr><td>%s</td><td><code>%s</code></td><td>%d</td><td><pre style=\"white-space:pre-wrap\">%s</pre></td></tr>",
			html.EscapeString(failure.Target.Provider), html.EscapeString(failure.Target.ModelID), failure.StatusCode, html.EscapeString(failure.Error))
	}
	subject := fmt.Sprintf("[OpenPaths] Provider credits need attention: %s", strings.Join(providers, ", "))
	body := fmt.Sprintf(`<h2>OpenPaths provider credit alert</h2>
<p>The daily direct-provider probe at <strong>%s</strong> received a credit, balance, billing, spend-limit, or quota-exhaustion response.</p>
<p>Prompt: <code>%s</code>. These calls bypass OpenPaths fallback routing.</p>
<table border="1" cellpadding="8" cellspacing="0"><thead><tr><th>Provider</th><th>Model</th><th>HTTP</th><th>Response</th></tr></thead><tbody>%s</tbody></table>`,
		html.EscapeString(at.Format("2006-01-02 15:04 MST")), probePrompt, rows.String())
	return subject, body
}

func nextDailyRun(now time.Time, location *time.Location, hour, minute int) time.Time {
	localNow := now.In(location)
	next := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), hour, minute, 0, 0, location)
	if !next.After(localNow) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func parseClock(raw string) (int, int) {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	if len(parts) == 2 {
		hour, hourErr := strconv.Atoi(parts[0])
		minute, minuteErr := strconv.Atoi(parts[1])
		if hourErr == nil && minuteErr == nil && hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59 {
			return hour, minute
		}
	}
	log.Printf("provider-credit-poll: invalid run time %q, using %02d:%02d", raw, defaultRunHour, defaultRunMinute)
	return defaultRunHour, defaultRunMinute
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max-3] + "..."
}
