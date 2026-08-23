package guardrails

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/router"
)

func TestFilterRouteCandidates_BlockedProviderWins(t *testing.T) {
	candidates := []router.RouteCandidate{
		{ModelCfg: &model.ModelConfig{Provider: "openai"}},
		{ModelCfg: &model.ModelConfig{Provider: "anthropic"}},
	}

	got, err := FilterRouteCandidates(candidates, []string{"openai", "anthropic"}, []string{"openai"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ModelCfg.Provider != "anthropic" {
		t.Fatalf("got candidates %#v", got)
	}
}

func TestFilterRouteCandidates_AllBlocked(t *testing.T) {
	candidates := []router.RouteCandidate{{ModelCfg: &model.ModelConfig{Provider: "openai"}}}
	if _, err := FilterRouteCandidates(candidates, nil, []string{"openai"}); err == nil {
		t.Fatal("expected all-blocked error")
	}
}
