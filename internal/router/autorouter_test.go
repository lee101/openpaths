package router

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/openpaths/openpaths/internal/config"
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
type stubEmbedder struct {
	vectors map[string][]float64
}

func (s *stubEmbedder) Name() string { return "stub" }

func (s *stubEmbedder) Embed(_ context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	var key string
	switch v := req.Input.(type) {
	case string:
		key = v
	case []string:
		if len(v) > 0 {
			key = v[0]
		}
	}
	vec := s.vectors[key]
	return &model.EmbeddingResponse{
		Data: []model.EmbeddingData{{Embedding: vec}},
	}, nil
}

func TestMaybeResolveAuto_UsesNamedTierWhenModalityIsEmpty(t *testing.T) {
	r := newTestRouter([]model.ModelConfig{
		{ID: "auto-easy-task", Provider: "google"},
		{ID: "gemini-flash-lite", Provider: "google"},
		{ID: "gpt-4o", Provider: "openai"},
	}, "google", "openai")

	r.SetAutoRouter(&AutoRouter{
		embedder: &stubEmbedder{
			vectors: map[string][]float64{
				"quick cleanup": {1, 0},
			},
		},
		tables: map[string][]AutoEntry{
			"text": {
				{ModelID: "gpt-4o", ReasoningEffort: "low", Embedding: []float64{1, 0}},
			},
			"easy-task": {
				{ModelID: "gemini-flash-lite", ReasoningEffort: "none", Embedding: []float64{1, 0}},
			},
		},
		ready: true,
	})

	got := r.MaybeResolveAuto(context.Background(), "auto-easy-task", "", "quick cleanup")
	if got.ModelID != "gemini-flash-lite" {
		t.Fatalf("MaybeResolveAuto() model = %q, want %q", got.ModelID, "gemini-flash-lite")
	}
	if got.ReasoningEffort != "none" {
		t.Fatalf("MaybeResolveAuto() reasoning = %q, want none", got.ReasoningEffort)
	}
}

func TestMaybeResolveAutoWithTier_HardTierOverridesNonAutoModel(t *testing.T) {
	r := newTestRouter([]model.ModelConfig{
		{ID: "gpt-4o", Provider: "openai"},
		{ID: "claude-opus-latest", Provider: "anthropic"},
	}, "openai", "anthropic")

	r.SetAutoRouter(&AutoRouter{
		embedder: &stubEmbedder{
			vectors: map[string][]float64{
				"build me a sankey flow diagram": {1, 0},
			},
		},
		tables: map[string][]AutoEntry{
			"hard-task": {
				{ModelID: "claude-opus-latest", ReasoningEffort: "medium", Embedding: []float64{1, 0}},
			},
		},
		ready: true,
	})

	// Caller didn't use an auto-* model name — task_tier alone should promote.
	got := r.MaybeResolveAutoWithTier(context.Background(), "gpt-4o", "", "hard", "build me a sankey flow diagram")
	if got.ModelID != "claude-opus-latest" {
		t.Fatalf("task_tier=hard should route to claude-opus-latest, got %q", got.ModelID)
	}
	if got.ReasoningEffort != "medium" {
		t.Fatalf("reasoning effort = %q, want medium", got.ReasoningEffort)
	}
}

func TestMaybeResolveAutoWithTier_EmptyTierFallsBackToModelNameBehaviour(t *testing.T) {
	r := newTestRouter([]model.ModelConfig{
		{ID: "gpt-4o", Provider: "openai"},
	}, "openai")

	// No autorouter tables for the non-auto model → should pass through untouched.
	r.SetAutoRouter(&AutoRouter{
		embedder: &stubEmbedder{},
		tables:   map[string][]AutoEntry{},
		ready:    true,
	})

	got := r.MaybeResolveAutoWithTier(context.Background(), "gpt-4o", "", "", "hello")
	if got.ModelID != "gpt-4o" {
		t.Fatalf("non-auto model with empty tier should pass through, got %q", got.ModelID)
	}
}

func TestTaskTierToModality(t *testing.T) {
	cases := []struct {
		tier string
		want string
		ok   bool
	}{
		{"easy", "easy-task", true},
		{"medium", "medium-task", true},
		{"think", "think-task", true},
		{"hard", "hard-task", true},
		{"", "", false},
		{"bogus", "", false},
	}
	for _, c := range cases {
		got, ok := TaskTierToModality(c.tier)
		if got != c.want || ok != c.ok {
			t.Errorf("TaskTierToModality(%q) = (%q,%v), want (%q,%v)", c.tier, got, ok, c.want, c.ok)
		}
	}
}

func TestAutoRouter_HardTaskRoutesSankeyToClaudeOpus(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "hard-task", "Render a sankey diagram showing user flow through the funnel with nested categories.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "claude-opus-latest" {
		t.Fatalf("ModelID = %q, want claude-opus-latest", got.ModelID)
	}
}

func TestDefaultRoutingTables_HardTaskIncludesOpus(t *testing.T) {
	entries := defaultRoutingTables()["hard-task"]
	if len(entries) == 0 {
		t.Fatal("hard-task routing table is empty")
	}
	var hasOpus bool
	for _, e := range entries {
		if e.ModelID == "claude-opus-latest" {
			hasOpus = true
			break
		}
	}
	if !hasOpus {
		t.Fatal("hard-task routing table must include claude-opus-latest")
	}
}

func TestDefaultRoutingTablesTargetsExistInConfig(t *testing.T) {
	oldJWT := os.Getenv("JWT_SECRET")
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Setenv("JWT_SECRET", oldJWT)

	cfg, err := config.Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatalf("Load(config.yaml): %v", err)
	}

	modelIDs := make(map[string]struct{}, len(cfg.Models))
	modelNames := make(map[string]struct{}, len(cfg.Models))
	for _, m := range cfg.Models {
		modelIDs[m.ID] = struct{}{}
		modelNames[m.ID] = struct{}{}
		for _, alias := range m.Aliases {
			modelNames[alias] = struct{}{}
		}
	}

	for modality, entries := range defaultRoutingTables() {
		for _, entry := range entries {
			if _, ok := modelIDs[entry.ModelID]; !ok {
				t.Fatalf("routing table %q references unknown model %q", modality, entry.ModelID)
			}
		}
	}

	for autoModel := range autoModelMap {
		if _, ok := modelNames[autoModel]; !ok {
			t.Fatalf("auto model %q is missing from config.yaml ids and aliases", autoModel)
		}
	}
}
