package skillindex

import (
	"context"
	"hash/fnv"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

// stubEmbedder produces a deterministic pseudo-embedding from token hashes so
// similar text yields similar vectors — enough to exercise ranking.
type stubEmbedder struct{}

func (stubEmbedder) Name() string { return "stub" }

func (stubEmbedder) Embed(_ context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	var inputs []string
	switch v := req.Input.(type) {
	case string:
		inputs = []string{v}
	case []string:
		inputs = v
	}
	const dim = 64
	data := make([]model.EmbeddingData, len(inputs))
	for i, text := range inputs {
		vec := make([]float64, dim)
		for _, tok := range strings.Fields(strings.ToLower(text)) {
			h := fnv.New32a()
			_, _ = h.Write([]byte(tok))
			vec[h.Sum32()%dim] += 1
		}
		data[i] = model.EmbeddingData{Object: "embedding", Embedding: vec, Index: i}
	}
	return &model.EmbeddingResponse{Object: "list", Data: data, Model: req.Model}, nil
}

type stubSource struct{ skills []model.Skill }

func (s stubSource) IterForIndex(_ context.Context, _ int, fn func(model.Skill) error) (int, error) {
	for _, sk := range s.skills {
		if err := fn(sk); err != nil {
			return 0, err
		}
	}
	return len(s.skills), nil
}

func TestSkillIndexSearch(t *testing.T) {
	src := stubSource{skills: []model.Skill{
		{Slug: "codex-infinity/code-review", Name: "code review", Description: "review a pull request", Source: "codex-infinity"},
		{Slug: "hermes/docker", Name: "docker management", Description: "manage docker containers", Source: "hermes"},
		{Slug: "hermes/research", Name: "deep research", Description: "multi source research report", Source: "hermes"},
	}}
	ix := New(stubEmbedder{})
	ix.SetSource(src)
	ix.Rebuild(context.Background())
	if !ix.Ready() {
		t.Fatalf("index not ready: %+v", ix.Status())
	}

	results, err := ix.Search(context.Background(), "review a pull request", model.SkillFilters{}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 || results[0].Slug != "codex-infinity/code-review" {
		t.Fatalf("expected code-review first, got %+v", results)
	}

	// Source filter restricts the result set.
	filtered, err := ix.Search(context.Background(), "docker containers", model.SkillFilters{Source: "hermes"}, 5)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range filtered {
		if r.Source != "hermes" {
			t.Fatalf("filter leaked non-hermes: %s", r.Slug)
		}
	}
}
