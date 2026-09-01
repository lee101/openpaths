package cron

import (
	"reflect"
	"testing"
	"time"

	"github.com/openpaths/openpaths/internal/model"
)

func TestProbePayloadExercisesPlaygroundCompatibilityControls(t *testing.T) {
	payload := probePayload("grok-4.6")
	want := map[string]any{
		"temperature": 0.7, "top_p": 1.0, "presence_penalty": 0.0,
		"frequency_penalty": 0.0, "max_tokens": probeMaxTokens,
		"stop": []string{"\n\n"},
	}
	if payload["model"] != "grok-4.6" {
		t.Fatalf("model = %v", payload["model"])
	}
	for key, value := range want {
		if !reflect.DeepEqual(payload[key], value) {
			t.Errorf("%s = %#v, want %#v", key, payload[key], value)
		}
	}
}

func TestIsChatProbeModel(t *testing.T) {
	if !IsChatProbeModel(model.ModelConfig{
		ID: "composer-2.5", Provider: "cursor",
		InputPricePer1M: 0.5, OutputPricePer1M: 2.5, ContextWindow: 256000, MaxOutputTokens: 128000,
	}) {
		t.Fatal("expected composer-2.5 to be probeable")
	}
	if IsChatProbeModel(model.ModelConfig{
		ID: "zimage", Provider: "netwrck", PricePerImage: 0.007,
	}) {
		t.Fatal("image model should not be probeable")
	}
	if IsChatProbeModel(model.ModelConfig{
		ID: "auto-video", Provider: "fal", PricePerVideo: 0.1, InputPricePer1M: 1,
	}) {
		t.Fatal("video model should not be probeable")
	}
	// Embedding models produce no generated output (output price 0, max output 0).
	if IsChatProbeModel(model.ModelConfig{
		ID: "text-embedding", Provider: "textgenerator",
		InputPricePer1M: 0.1, OutputPricePer1M: 0, ContextWindow: 8192, MaxOutputTokens: 0,
	}) {
		t.Fatal("embedding model should not be probeable")
	}
	// Codex models are only served on /v1/responses, not /v1/chat/completions.
	if IsChatProbeModel(model.ModelConfig{
		ID: "gpt-5-codex", Provider: "openai",
		InputPricePer1M: 1.25, OutputPricePer1M: 10, ContextWindow: 400000, MaxOutputTokens: 128000,
	}) {
		t.Fatal("codex model should not be probeable")
	}
	if IsChatProbeModel(model.ModelConfig{
		ID: "retired-chat", Provider: "openrouter", Deprecated: true,
		InputPricePer1M: 1, OutputPricePer1M: 2, ContextWindow: 128000, MaxOutputTokens: 8192,
	}) {
		t.Fatal("deprecated model should not be probeable")
	}
}

func TestFilterDue(t *testing.T) {
	now := time.Now()
	targets := []model.ModelConfig{
		{ID: "never-probed"},
		{ID: "healthy-fresh"},
		{ID: "healthy-stale"},
		{ID: "failing-fresh"},
		{ID: "failing-stale"},
	}
	last := map[string]model.ModelProbeResult{
		// Healthy models ride the weekly cadence.
		"healthy-fresh": {Model: "healthy-fresh", OK: true, ProbedAt: now.Add(-2 * 24 * time.Hour)},
		"healthy-stale": {Model: "healthy-stale", OK: true, ProbedAt: now.Add(-8 * 24 * time.Hour)},
		// Failing models are retried daily.
		"failing-fresh": {Model: "failing-fresh", OK: false, ProbedAt: now.Add(-2 * time.Hour)},
		"failing-stale": {Model: "failing-stale", OK: false, ProbedAt: now.Add(-25 * time.Hour)},
	}

	got := map[string]bool{}
	for _, cfg := range filterDue(targets, last, now) {
		got[cfg.ID] = true
	}

	for _, id := range []string{"never-probed", "healthy-stale", "failing-stale"} {
		if !got[id] {
			t.Errorf("%s: expected due", id)
		}
	}
	for _, id := range []string{"healthy-fresh", "failing-fresh"} {
		if got[id] {
			t.Errorf("%s: expected to be skipped", id)
		}
	}
}

// A restart must not re-probe a catalogue that was probed minutes ago -- that
// was the bug that turned every deploy into a full billed sweep.
func TestFilterDueSkipsEverythingRightAfterASweep(t *testing.T) {
	now := time.Now()
	targets := []model.ModelConfig{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	last := map[string]model.ModelProbeResult{
		"a": {Model: "a", OK: true, ProbedAt: now.Add(-5 * time.Minute)},
		"b": {Model: "b", OK: true, ProbedAt: now.Add(-5 * time.Minute)},
		"c": {Model: "c", OK: false, ProbedAt: now.Add(-5 * time.Minute)},
	}
	if due := filterDue(targets, last, now); len(due) != 0 {
		t.Fatalf("expected no models due right after a sweep, got %d", len(due))
	}
}

func TestProbeSucceeded(t *testing.T) {
	tests := []struct {
		name             string
		choiceCount      int
		content          string
		completionTokens int
		want             bool
	}{
		{name: "visible output", choiceCount: 1, content: "hi", want: true},
		{name: "reasoning only", choiceCount: 1, completionTokens: 12, want: true},
		{name: "empty choice", choiceCount: 1, want: false},
		{name: "usage without choice", completionTokens: 12, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := probeSucceeded(tt.choiceCount, tt.content, tt.completionTokens); got != tt.want {
				t.Fatalf("probeSucceeded() = %v, want %v", got, tt.want)
			}
		})
	}
}
