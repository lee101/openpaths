package router

import (
	"context"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

type fakeEmbedder struct{}

func (f *fakeEmbedder) Name() string { return "fake-embedder" }

func (f *fakeEmbedder) Embed(_ context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	text, _ := req.Input.(string)
	vec := []float64{0, 0, 0, 0, 1}

	switch {
	case containsAny(text, "classify", "categorize", "structured output", "json schema"):
		vec = []float64{1, 0, 0, 0, 0}
	case containsAny(text, "implement feature", "new endpoint", "debug fix bug", "integration test", "simple python function", "reverse string", "helper utility"):
		vec = []float64{0, 1, 0, 0, 0}
	case containsAny(text, "code review", "security vulnerability", "authentication authorization", "oauth jwt"):
		vec = []float64{0, 0, 1, 0, 0}
	case containsAny(text, "rewrite polish", "customer support reply", "improve tone concise"):
		vec = []float64{0, 0, 0, 1, 0}
	case containsAny(text, "say yes", "one number", "quick parse", "tiny edit", "trivial acknowledgement", "classify sentiment"):
		vec = []float64{1, 0, 0, 0, 0}
	case containsAny(text, "small helper function", "simple script", "short email response", "small bug", "explain a concept simply"):
		vec = []float64{0, 1, 0, 0, 0}
	case containsAny(text, "plan a refactor", "test strategy", "review code for bugs", "design an api endpoint", "tradeoffs architecture"):
		vec = []float64{0, 0, 1, 0, 0}
	case containsAny(text, "3d mesh simplification algorithm", "prove a theorem", "distributed system protocol", "formal verification", "hard math olympiad"):
		vec = []float64{0, 0, 0, 0, 1}
	}

	return &model.EmbeddingResponse{
		Data: []model.EmbeddingData{{Embedding: vec}},
	}, nil
}

func containsAny(s string, needles ...string) bool {
	for _, needle := range needles {
		if len(needle) > 0 && containsFold(s, needle) {
			return true
		}
	}
	return false
}

func containsFold(s, needle string) bool {
	if len(needle) == 0 {
		return true
	}
outer:
	for i := 0; i+len(needle) <= len(s); i++ {
		for j := 0; j < len(needle); j++ {
			a := s[i+j]
			b := needle[j]
			if 'A' <= a && a <= 'Z' {
				a += 'a' - 'A'
			}
			if 'A' <= b && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				continue outer
			}
		}
		return true
	}
	return false
}

func TestAutoRouter_EasyTaskRoutesClassificationToGPT54Nano(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "easy-task", "Classify this ticket by priority and return JSON schema output.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "gpt-5.4-nano" {
		t.Fatalf("ModelID = %q, want %q", got.ModelID, "gpt-5.4-nano")
	}
}

func TestAutoRouter_MediumTaskRoutesImplementationToGPT54Mini(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "medium-task", "Write a simple Python function to reverse a string.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "gpt-5.4-mini" {
		t.Fatalf("ModelID = %q, want %q", got.ModelID, "gpt-5.4-mini")
	}
}

func TestAutoRouter_MediumTaskRoutesSecurityReviewToClaudeSonnet(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "medium-task", "Do a code review for security vulnerability and authentication authorization issues.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "claude-sonnet-4-6" {
		t.Fatalf("ModelID = %q, want %q", got.ModelID, "claude-sonnet-4-6")
	}
}

func TestDefaultRoutingTables_EasyTaskIncludesRequestedProviders(t *testing.T) {
	entries := defaultRoutingTables()["easy-task"]
	got := map[string]bool{}
	for _, entry := range entries {
		got[entry.ModelID] = true
	}

	for _, want := range []string{
		"gpt-5.4-nano",
		"gemini-flash-lite",
		"gemini-2.5-flash",
		"claude-haiku-4-5-20251001",
	} {
		if !got[want] {
			t.Fatalf("easy-task routing table missing %q", want)
		}
	}
}

func TestAutoRouter_ThinkTaskRoutesTrivialPromptToNoThinking(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "think-task", "Say yes.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "gpt-5.4-nano" {
		t.Fatalf("ModelID = %q, want %q", got.ModelID, "gpt-5.4-nano")
	}
	if got.ReasoningEffort != "none" {
		t.Fatalf("ReasoningEffort = %q, want %q", got.ReasoningEffort, "none")
	}
}

func TestAutoRouter_ThinkTaskRoutesHardPromptToHighThinking(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "think-task", "Create a 3d mesh simplification algorithm that preserves topology and minimizes geometric error.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "gpt-5.4" {
		t.Fatalf("ModelID = %q, want %q", got.ModelID, "gpt-5.4")
	}
	if got.ReasoningEffort != "high" {
		t.Fatalf("ReasoningEffort = %q, want %q", got.ReasoningEffort, "high")
	}
}

func TestDefaultRoutingTables_ThinkTaskIncludesAllReasoningLevels(t *testing.T) {
	entries := defaultRoutingTables()["think-task"]
	got := map[string]bool{}
	for _, entry := range entries {
		got[entry.ReasoningEffort] = true
	}

	for _, want := range []string{"none", "low", "medium", "high"} {
		if !got[want] {
			t.Fatalf("think-task routing table missing reasoning level %q", want)
		}
	}
}
