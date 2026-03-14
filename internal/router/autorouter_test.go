package router

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/openpaths/openpaths/internal/config"
	"github.com/openpaths/openpaths/internal/model"
)

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
