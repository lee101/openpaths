package polling

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/config"
	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

type fakeProvider struct {
	name    string
	err     error
	lastReq *model.ChatCompletionRequest
}

func (p *fakeProvider) Name() string { return p.name }
func (p *fakeProvider) ChatCompletion(_ context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	p.lastReq = req
	if p.err != nil {
		return nil, p.err
	}
	return &model.ChatCompletionResponse{}, nil
}
func (p *fakeProvider) ChatCompletionStream(context.Context, *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	return nil, errors.New("unused")
}
func (p *fakeProvider) HealthCheck(context.Context) error { return nil }

type fakeRegistry map[string]provider.Provider

func (r fakeRegistry) Get(name string) (provider.Provider, error) {
	p, ok := r[name]
	if !ok {
		return nil, errors.New("not registered")
	}
	return p, nil
}

func TestDefaultTargetsExistInCatalogue(t *testing.T) {
	cfg, err := config.Load("../config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	byID := make(map[string]model.ModelConfig, len(cfg.Models))
	for _, item := range cfg.Models {
		byID[item.ID] = item
	}
	for _, spec := range targetSpecs {
		item, ok := byID[spec.ModelID]
		if !ok {
			t.Errorf("poll target %s/%s is absent from config", spec.Provider, spec.ModelID)
			continue
		}
		if item.Provider != spec.Provider || item.ProviderModelID == "" {
			t.Errorf("poll target %s/%s resolves to %s/%s", spec.Provider, spec.ModelID, item.Provider, item.ProviderModelID)
		}
	}
}

func TestCreditFailureClassification(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		message string
		want    bool
	}{
		{"xai credits", 403, "This team has used all credits or reached its monthly spending limit", true},
		{"openai quota", 429, `{"code":"insufficient_quota"}`, true},
		{"google quota", 429, "RESOURCE_EXHAUSTED: quota exceeded", true},
		{"payment required", 402, "", true},
		{"ordinary rate limit", 429, "requests per minute rate limit reached", false},
		{"invalid key", 403, "invalid api key", false},
		{"upstream outage", 503, "service unavailable", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isCreditFailure(tc.status, tc.message); got != tc.want {
				t.Fatalf("isCreditFailure(%d, %q) = %v, want %v", tc.status, tc.message, got, tc.want)
			}
		})
	}
}

func TestRunOnceEmailsOnlyCreditFailures(t *testing.T) {
	t.Setenv("OPENPATHS_CREDIT_POLL_EMAIL", "")
	xaiProvider := &fakeProvider{name: "xai", err: &provider.ProviderError{
		Provider: "xai", StatusCode: 403, Message: "team has used all credits or reached monthly spending limit",
	}}
	registry := fakeRegistry{
		"xai": xaiProvider,
		"openai": &fakeProvider{name: "openai", err: &provider.ProviderError{
			Provider: "openai", StatusCode: 503, Message: "service unavailable",
		}},
		"google": &fakeProvider{name: "google"},
	}
	models := []model.ModelConfig{
		{ID: "grok-4.6", Provider: "xai", ProviderModelID: "grok-4.6"},
		{ID: "gpt-5.4-nano", Provider: "openai", ProviderModelID: "gpt-5.4-nano"},
		{ID: "gemini-3.1-flash-lite", Provider: "google", ProviderModelID: "gemini-3.1-flash-lite"},
	}
	var to, subject, body string
	monitor := NewProviderCreditMonitor(registry, models, func(gotTo, gotSubject, gotBody string) error {
		to, subject, body = gotTo, gotSubject, gotBody
		return nil
	})
	monitor.now = func() time.Time { return time.Date(2026, 9, 1, 8, 0, 0, 0, time.FixedZone("NZST", 12*60*60)) }

	summary := monitor.RunOnce(context.Background())
	if len(summary.Results) != 3 {
		t.Fatalf("results = %d, want 3", len(summary.Results))
	}
	if to != defaultAlertEmail {
		t.Fatalf("to = %q, want %q", to, defaultAlertEmail)
	}
	if !strings.Contains(subject, "xai") || strings.Contains(subject, "openai") {
		t.Fatalf("unexpected subject %q", subject)
	}
	if !strings.Contains(body, "grok-4.6") || strings.Contains(body, "service unavailable") {
		t.Fatalf("unexpected body %q", body)
	}
	if xaiProvider.lastReq == nil || xaiProvider.lastReq.Model != "grok-4.6" {
		t.Fatalf("direct xAI model request = %#v", xaiProvider.lastReq)
	}
	if len(xaiProvider.lastReq.Messages) != 1 || xaiProvider.lastReq.Messages[0].Content != probePrompt {
		t.Fatalf("probe messages = %#v", xaiProvider.lastReq.Messages)
	}
}

func TestRunOnceDoesNotEmailForNonCreditFailure(t *testing.T) {
	registry := fakeRegistry{"openai": &fakeProvider{name: "openai", err: &provider.ProviderError{
		Provider: "openai", StatusCode: 429, Message: "requests per minute rate limit reached",
	}}}
	models := []model.ModelConfig{{ID: "gpt-5.4-nano", Provider: "openai", ProviderModelID: "gpt-5.4-nano"}}
	called := false
	monitor := NewProviderCreditMonitor(registry, models, func(string, string, string) error {
		called = true
		return nil
	})
	monitor.RunOnce(context.Background())
	if called {
		t.Fatal("non-credit failures must not send an alert email")
	}
}

func TestNextDailyRunUsesAucklandDST(t *testing.T) {
	location, err := time.LoadLocation("Pacific/Auckland")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		now  time.Time
		want time.Time
	}{
		{
			name: "summer NZDT",
			now:  time.Date(2026, 1, 10, 18, 30, 0, 0, time.UTC),
			want: time.Date(2026, 1, 11, 8, 0, 0, 0, location),
		},
		{
			name: "winter NZST after today's run",
			now:  time.Date(2026, 7, 10, 1, 0, 0, 0, time.UTC),
			want: time.Date(2026, 7, 11, 8, 0, 0, 0, location),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := nextDailyRun(tc.now, location, 8, 0); !got.Equal(tc.want) {
				t.Fatalf("nextDailyRun() = %s, want %s", got, tc.want)
			}
		})
	}
}
