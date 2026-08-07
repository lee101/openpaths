package handler

import (
	"testing"

	"github.com/openpaths/openpaths/internal/config"
	"github.com/openpaths/openpaths/internal/model"
)

// applyDefaultEffort mirrors the per-candidate resolution in HandleChatCompletions
// and runFusionCompletion: the caller always wins, the model default fills a blank,
// and one candidate's default must never leak into the next.
func applyDefaultEffort(callerEffort string, cfg *model.ModelConfig) string {
	effort := callerEffort
	if effort == "" {
		effort = cfg.DefaultReasoningEffort
	}
	return effort
}

func TestDefaultReasoningEffort(t *testing.T) {
	grok := &model.ModelConfig{ID: "grok-4.5", DefaultReasoningEffort: "low"}
	plain := &model.ModelConfig{ID: "some-model"}

	if got := applyDefaultEffort("", grok); got != "low" {
		t.Fatalf("blank caller effort should take the model default, got %q", got)
	}
	if got := applyDefaultEffort("high", grok); got != "high" {
		t.Fatalf("caller effort must win, got %q", got)
	}
	if got := applyDefaultEffort("", plain); got != "" {
		t.Fatalf("model without a default should stay blank, got %q", got)
	}

	// Fallback ordering: grok's default must not carry into a model that has none.
	callerEffort := ""
	first := applyDefaultEffort(callerEffort, grok)
	second := applyDefaultEffort(callerEffort, plain)
	if first != "low" || second != "" {
		t.Fatalf("default leaked across candidates: %q then %q", first, second)
	}
}

func TestGrokConfiguredLowEffort(t *testing.T) {
	cfg, err := config.Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	for _, m := range cfg.Models {
		if m.ID == "grok-4.5" {
			if m.DefaultReasoningEffort != "low" {
				t.Fatalf("grok-4.5 default_reasoning_effort = %q, want low", m.DefaultReasoningEffort)
			}
			return
		}
	}
	t.Fatal("grok-4.5 not found in config.yaml")
}
