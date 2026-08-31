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
	vec := []float64{0.4, 0.3, 0.2, 0.1, 0.05}

	switch {
	case containsAny(text, "forecast a line", "line forecasting", "time series", "extrapolate", "next values", "next points", "estimate trend direction", "sparse noisy observations"):
		vec = []float64{0.1, 0.1, 0.1, 0.1, 0.9}
	case containsAny(text, "sensitive content", "adult roleplay", "biosecurity", "fringe", "harm policy", "controversial"):
		vec = []float64{0.2, 0.2, 0.2, 0.2, 0.2}
	case containsAny(text, "classify", "categorize", "structured output", "json schema"):
		vec = []float64{1, 0, 0, 0, 0}
	case containsAny(text, "pick chart type", "select chart type", "visualization type"):
		vec = []float64{1, 0, 0, 0, 0}
	case containsAny(text, "implement feature", "new endpoint", "debug fix bug", "crashes with", "stack trace", "exception crash", "integration test", "simple python function", "reverse string", "helper utility"):
		vec = []float64{0, 1, 0, 0, 0}
	case containsAny(text, "plotly graph config", "chart traces", "dataframe transform", "xlsx unstructured"):
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
	case containsAny(text, "3d mesh simplification algorithm", "3d simulation", "cogs", "gears", "clock mechanism", "prove a theorem", "distributed system protocol", "formal verification", "hard math olympiad"):
		vec = []float64{0, 0, 0, 0, 1}
	case containsAny(text, "glsl", "hlsl", "fragment shader", "vertex shader", "compute shader", "raymarching", "render pipeline", "post processing effect", "particle system", "vfx"):
		vec = []float64{0, 0, 0.85, 0.85, 0}
	case containsAny(text, "stock trading", "backtesting engine", "order book", "market data feed", "hft", "quant strategy", "tick data"):
		vec = []float64{0.8, 0, 0, 0, 0.55}
	case containsAny(text, "fine tuning", "llm ai development", "inference optimization", "kv cache", "quantization", "tokenizer", "rag pipeline", "cuda kernel", "eval harness"):
		vec = []float64{0, 0.7, 0, 0.7, 0.4}
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

func TestAutoRouter_EasyTaskRoutesChartTypeSelectionToGPT54Nano(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "easy-task", "Pick chart type for this dataset.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "gpt-5.4-nano" {
		t.Fatalf("ModelID = %q, want %q", got.ModelID, "gpt-5.4-nano")
	}
	if got.ReasoningEffort != "none" {
		t.Fatalf("ReasoningEffort = %q, want none", got.ReasoningEffort)
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

func TestAutoRouter_MediumTaskRoutesGraphConfigToGPT54Mini(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "medium-task", "Build a Plotly graph config with traces and layout from challenging unstructured xlsx data.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "gpt-5.4-mini" {
		t.Fatalf("ModelID = %q, want %q", got.ModelID, "gpt-5.4-mini")
	}
	if got.ReasoningEffort != "low" {
		t.Fatalf("ReasoningEffort = %q, want low", got.ReasoningEffort)
	}
}

func TestAutoRouter_MediumTaskRoutesSecurityReviewToGPT54Mini(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "medium-task", "Do a code review for security vulnerability and authentication authorization issues.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "gpt-5.4-mini" {
		t.Fatalf("ModelID = %q, want %q", got.ModelID, "gpt-5.4-mini")
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
		"deepseek-v4-flash",
		"gemini-3.1-flash-lite",
		"gemini-3.7-flash",
	} {
		if !got[want] {
			t.Fatalf("easy-task routing table missing %q", want)
		}
	}
}

func TestAutoRouter_EasyTaskRoutesSensitiveClassifierToDeepSeekV4Flash(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "easy-task", "Classify this sensitive content for adult roleplay and fringe policy labels.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "deepseek-v4-flash" {
		t.Fatalf("ModelID = %q, want %q", got.ModelID, "deepseek-v4-flash")
	}
}

func TestAutoRouter_HardTaskRoutesBiosecurityToDeepSeekV4Pro(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "hard-task", "Analyze this biosecurity dual use request with careful policy reasoning.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "deepseek-v4-pro" {
		t.Fatalf("ModelID = %q, want %q", got.ModelID, "deepseek-v4-pro")
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
	if got.ModelID != "gemini-3.7-flash" {
		t.Fatalf("ModelID = %q, want %q", got.ModelID, "gemini-3.7-flash")
	}
	if got.ReasoningEffort != "high" {
		t.Fatalf("ReasoningEffort = %q, want %q", got.ReasoningEffort, "high")
	}
}

func TestAutoRouter_ThinkTaskRoutesLineForecastingToLowThinking(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "think-task", "Forecast a line and extrapolate the next points from this unusual time series.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ReasoningEffort != "low" {
		t.Fatalf("ReasoningEffort = %q, want low", got.ReasoningEffort)
	}
}

func TestAutoRouter_ThinkTaskRoutes3DSimulationToHighThinking(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "think-task", "Make a 3d simulation of cogs in a clock.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "gemini-3.7-flash" {
		t.Fatalf("ModelID = %q, want %q", got.ModelID, "gemini-3.7-flash")
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

func TestSelectAutoEntry_UsesWeightedNeighborConsensusForAmbiguousPrompt(t *testing.T) {
	query := []float64{1, 0}
	entries := []AutoEntry{
		{ModelID: "gpt-5.6-sol", ReasoningEffort: "high", Embedding: []float64{0.9553, 0.2955}},
		{ModelID: "gpt-5.6-luna", ReasoningEffort: "low", Embedding: []float64{0.9394, 0.3429}},
		{ModelID: "gpt-5.6-luna", ReasoningEffort: "low", Embedding: []float64{0.9359, 0.3523}},
		{ModelID: "gpt-5.6-luna", ReasoningEffort: "low", Embedding: []float64{0.9323, 0.3616}},
	}

	got, bestSim, confidence, policy := selectAutoEntry(entries, query, 4)
	if got.ModelID != "gpt-5.6-luna" {
		t.Fatalf("weighted consensus model = %q, want gpt-5.6-luna", got.ModelID)
	}
	if policy != "weighted-knn" {
		t.Fatalf("policy = %q, want weighted-knn", policy)
	}
	if bestSim <= 0.95 || confidence <= 0.5 {
		t.Fatalf("bestSim/confidence = %.4f/%.4f, want a close but decisive neighborhood", bestSim, confidence)
	}
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
		{ID: "openpaths/auto-cheap", Provider: "openai"},
		{ID: "gemini-3.1-flash-lite", Provider: "google"},
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
			"cheap-task": {
				{ModelID: "gemini-3.1-flash-lite", ReasoningEffort: "none", Embedding: []float64{1, 0}},
			},
		},
		ready: true,
	})

	got := r.MaybeResolveAuto(context.Background(), "openpaths/auto-cheap", "", "quick cleanup")
	if got.ModelID != "gemini-3.1-flash-lite" {
		t.Fatalf("MaybeResolveAuto() model = %q, want %q", got.ModelID, "gemini-3.1-flash-lite")
	}
	if got.ReasoningEffort != "none" {
		t.Fatalf("MaybeResolveAuto() reasoning = %q, want none", got.ReasoningEffort)
	}
}

func TestMaybeResolveAutoWithTier_HardTierOverridesNonAutoModel(t *testing.T) {
	r := newTestRouter([]model.ModelConfig{
		{ID: "gpt-4o", Provider: "openai"},
		{ID: "gemini-3.7-flash", Provider: "google"},
	}, "openai", "google", "anthropic")

	r.SetAutoRouter(&AutoRouter{
		embedder: &stubEmbedder{
			vectors: map[string][]float64{
				"build me a sankey flow diagram": {1, 0},
			},
		},
		tables: map[string][]AutoEntry{
			"reasoning-task": {
				{ModelID: "gemini-3.7-flash", ReasoningEffort: "medium", Embedding: []float64{1, 0}},
			},
		},
		ready: true,
	})

	// Caller didn't use an auto-* model name — task_tier alone should promote.
	got := r.MaybeResolveAutoWithTier(context.Background(), "gpt-4o", "", "hard", "build me a sankey flow diagram")
	if got.ModelID != "gemini-3.7-flash" {
		t.Fatalf("task_tier=hard should route to gemini-3.7-flash, got %q", got.ModelID)
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

func TestMaybeResolveAutoReasoning_KeepsDirectModelReasoningOnly(t *testing.T) {
	r := newTestRouter([]model.ModelConfig{
		{ID: "nvidia/deepseek-v4-pro", Provider: "nvidia"},
		{ID: "gemini-3.7-flash", Provider: "google"},
	}, "nvidia", "google")

	r.SetAutoRouter(&AutoRouter{
		embedder: &stubEmbedder{
			vectors: map[string][]float64{
				"make a 3d simulation of cogs in a clock": {1, 0},
			},
		},
		tables: map[string][]AutoEntry{
			"reasoning-task": {
				{ModelID: "gemini-3.7-flash", ReasoningEffort: "high", Embedding: []float64{1, 0}},
			},
		},
		ready: true,
	})

	got := r.MaybeResolveAutoReasoning(context.Background(), "make a 3d simulation of cogs in a clock")
	if got != "high" {
		t.Fatalf("MaybeResolveAutoReasoning() = %q, want high", got)
	}

	candidates, err := r.ResolveForRequest("nvidia/deepseek-v4-pro", "nvidia/deepseek-v4-pro")
	if err != nil {
		t.Fatalf("ResolveForRequest() error = %v", err)
	}
	if candidates[0].ModelCfg.ID != "nvidia/deepseek-v4-pro" {
		t.Fatalf("direct model changed to %q", candidates[0].ModelCfg.ID)
	}
}

func TestTaskTierToModality(t *testing.T) {
	cases := []struct {
		tier string
		want string
		ok   bool
	}{
		{"easy", "cheap-task", true},
		{"cheap", "cheap-task", true},
		{"medium", "code-task", true},
		{"code", "code-task", true},
		{"think", "reasoning-task", true},
		{"reasoning", "reasoning-task", true},
		{"hard", "reasoning-task", true},
		// legacy tier names still accepted
		{"fast", "fast-task", true},
		{"vision", "vision-task", true},
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

func TestAutoRouter_HardTaskRoutesSankeyToGPT56Sol(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "hard-task", "Render a sankey diagram showing user flow through the funnel with nested categories.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "gemini-3.7-flash" {
		t.Fatalf("ModelID = %q, want gemini-3.7-flash", got.ModelID)
	}
}

func TestDefaultRoutingTables_HardTaskIncludesGPT56Sol(t *testing.T) {
	entries := defaultRoutingTables()["hard-task"]
	if len(entries) == 0 {
		t.Fatal("hard-task routing table is empty")
	}
	var hasGPT56Sol bool
	for _, e := range entries {
		if e.ModelID == "gemini-3.7-flash" {
			hasGPT56Sol = true
			break
		}
	}
	if !hasGPT56Sol {
		t.Fatal("hard-task routing table must include gemini-3.7-flash")
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

// TestIsAutoModel_LegacyAliasesResolve guards the public promise that legacy
// auto IDs keep working: each documented alias must resolve to the modality of
// its current openpaths/* equivalent.
func TestIsAutoModel_LegacyAliasesResolve(t *testing.T) {
	cases := []struct {
		alias    string
		modality string
	}{
		{"auto", "text"},
		{"auto-text", "text"},
		{"auto-chat", "text"},
		{"auto-easy-task", "cheap-task"},
		{"auto-easy", "cheap-task"},
		{"auto-cheap", "cheap-task"},
		{"auto-fast", "fast-task"},
		{"auto-code", "code-task"},
		{"auto-medium-task", "code-task"},
		{"auto-medium", "code-task"},
		{"auto-reasoning", "reasoning-task"},
		{"auto-think", "reasoning-task"},
		{"auto-think-task", "reasoning-task"},
		{"autothink", "reasoning-task"},
		{"auto-hard", "reasoning-task"},
		{"auto-hard-task", "reasoning-task"},
		{"auto-opus", "reasoning-task"},
		{"auto-vision", "vision-task"},
		{"auto-image", "image"},
		{"auto-img", "image"},
		{"auto-video", "video"},
		{"auto-vid", "video"},
	}
	for _, c := range cases {
		got, ok := IsAutoModel(c.alias)
		if !ok {
			t.Errorf("IsAutoModel(%q) ok = false, want true", c.alias)
			continue
		}
		if got != c.modality {
			t.Errorf("IsAutoModel(%q) = %q, want %q", c.alias, got, c.modality)
		}
	}

	if _, ok := IsAutoModel("gpt-5.6-sol"); ok {
		t.Error("IsAutoModel(\"gpt-5.6-sol\") ok = true, want false for a direct model")
	}
}

func TestIsAutoThinkModel(t *testing.T) {
	cases := []struct {
		model string
		want  bool
	}{
		{"auto-think", true},
		{"autothink", true},
		{"auto-hard-task", true},
		{"openpaths/auto-reasoning", true},
		{"auto", false},        // text modality, not reasoning
		{"auto-image", false},  // image modality
		{"auto-fast", false},   // fast modality
		{"gpt-5.6-sol", false}, // not an auto model at all
	}
	for _, c := range cases {
		if got := IsAutoThinkModel(c.model); got != c.want {
			t.Errorf("IsAutoThinkModel(%q) = %v, want %v", c.model, got, c.want)
		}
	}
}

func TestIsAutoReasoningEffort(t *testing.T) {
	cases := []struct {
		effort string
		want   bool
	}{
		{"auto", true},
		{"automatic", true},
		{"auto-think", true},
		{"autothink", true},
		{"  AUTO  ", true}, // trimmed + case-insensitive
		{"high", false},
		{"none", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsAutoReasoningEffort(c.effort); got != c.want {
			t.Errorf("IsAutoReasoningEffort(%q) = %v, want %v", c.effort, got, c.want)
		}
	}
}

func TestAutoRouter_CodeTaskRoutesEverydayCodingToOxAlpha(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "code-task", "Debug and fix this bug: the endpoint crashes with a nil exception on empty input.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "openpaths/stealth/ox-alpha" {
		t.Fatalf("ModelID = %q, want openpaths/stealth/ox-alpha", got.ModelID)
	}
	if got.ReasoningEffort != "high" {
		t.Fatalf("ReasoningEffort = %q, want high", got.ReasoningEffort)
	}
}

func TestAutoRouter_CodeTaskRoutesEverydayTestsToOxAlpha(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "code-task", "Write integration tests for the checkout handler with mocks.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "openpaths/stealth/ox-alpha" {
		t.Fatalf("ModelID = %q, want openpaths/stealth/ox-alpha", got.ModelID)
	}
}

func TestAutoRouter_CodeTaskRoutesShaderVFXToGPT56SolHigh(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "code-task", "Write a GLSL fragment shader doing raymarching for a volumetric cloud effect.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "gpt-5.6-sol" {
		t.Fatalf("ModelID = %q, want gpt-5.6-sol", got.ModelID)
	}
	if got.ReasoningEffort != "high" {
		t.Fatalf("ReasoningEffort = %q, want high", got.ReasoningEffort)
	}
}

func TestAutoRouter_CodeTaskRoutesTradingSystemToGPT56SolHigh(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "code-task", "Build a backtesting engine for a quant strategy over tick data with an order book simulator.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "gpt-5.6-sol" {
		t.Fatalf("ModelID = %q, want gpt-5.6-sol", got.ModelID)
	}
	if got.ReasoningEffort != "high" {
		t.Fatalf("ReasoningEffort = %q, want high", got.ReasoningEffort)
	}
}

func TestAutoRouter_CodeTaskRoutesLLMDevToGPT56SolHigh(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "code-task", "Add paged KV cache support and quantization to our LLM inference server.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "gpt-5.6-sol" {
		t.Fatalf("ModelID = %q, want gpt-5.6-sol", got.ModelID)
	}
	if got.ReasoningEffort != "high" {
		t.Fatalf("ReasoningEffort = %q, want high", got.ReasoningEffort)
	}
}

func TestAutoRouter_ReasoningTaskRoutesPlanningToOxAlphaXHigh(t *testing.T) {
	ar := NewAutoRouter(&fakeEmbedder{})
	if err := ar.Init(context.Background()); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := ar.ResolveAuto(context.Background(), "reasoning-task", "Plan a refactor migration test strategy across these service files.")
	if err != nil {
		t.Fatalf("ResolveAuto() error = %v", err)
	}
	if got.ModelID != "openpaths/stealth/ox-alpha" {
		t.Fatalf("ModelID = %q, want openpaths/stealth/ox-alpha", got.ModelID)
	}
	if got.ReasoningEffort != "xhigh" {
		t.Fatalf("ReasoningEffort = %q, want xhigh", got.ReasoningEffort)
	}
}

func TestDefaultRoutingTables_ReasoningTaskPinsOxAlphaToMaxThinking(t *testing.T) {
	for _, e := range defaultRoutingTables()["reasoning-task"] {
		if e.ModelID != "openpaths/stealth/ox-alpha" {
			continue
		}
		if e.ReasoningEffort != "xhigh" {
			t.Fatalf("reasoning-task ox-alpha entry %q effort = %q, want xhigh (model is free)", e.Description, e.ReasoningEffort)
		}
	}
}

func TestDefaultRoutingTables_CodeTaskNeverSendsOxAlphaWithoutThinking(t *testing.T) {
	for _, e := range defaultRoutingTables()["code-task"] {
		if e.ModelID == "openpaths/stealth/ox-alpha" && e.ReasoningEffort == "none" {
			t.Fatalf("code-task ox-alpha entry %q disables reasoning but upstream requires it (400)", e.Description)
		}
	}
}

func TestDefaultRoutingTables_CodeTaskOxAlphaIsLargestCohort(t *testing.T) {
	counts := map[string]int{}
	for _, e := range defaultRoutingTables()["code-task"] {
		counts[e.ModelID]++
	}
	ox := counts["openpaths/stealth/ox-alpha"]
	for model, n := range counts {
		if model != "openpaths/stealth/ox-alpha" && n >= ox {
			t.Fatalf("code-task cohort %q (%d) rivals ox-alpha (%d); everyday coding must stay the majority share", model, n, ox)
		}
	}
}
