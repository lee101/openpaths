package promptlib

import (
	"context"
	"hash/fnv"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestCatalogBuilds(t *testing.T) {
	if Count() == 0 {
		t.Fatal("catalog is empty")
	}
	// Slugs must be unique.
	seen := map[string]bool{}
	for _, p := range All() {
		if seen[p.Slug] {
			t.Fatalf("duplicate slug %q", p.Slug)
		}
		seen[p.Slug] = true
		if p.Title == "" || p.Prompt == "" {
			t.Fatalf("prompt %q missing title/prompt", p.Slug)
		}
		if _, ok := categoryBySlug[p.CategorySlug]; !ok {
			t.Fatalf("prompt %q has unknown category %q", p.Slug, p.CategorySlug)
		}
		if _, ok := modelBySlug[p.ModelSlug]; !ok {
			t.Fatalf("prompt %q has unknown model %q", p.Slug, p.ModelSlug)
		}
	}
}

func TestFeaturedSortedFirst(t *testing.T) {
	prompts := All()
	seenNonFeatured := false
	for _, p := range prompts {
		if !p.Featured {
			seenNonFeatured = true
		} else if seenNonFeatured {
			t.Fatalf("featured prompt %q appears after a non-featured one", p.Slug)
		}
	}
}

func TestKeyPromptExists(t *testing.T) {
	p, ok := BySlug("autocomplete-pytorch-coding")
	if !ok {
		t.Fatal("expected autocomplete-pytorch-coding prompt to exist")
	}
	if p.CategorySlug != "coding-dev" {
		t.Fatalf("unexpected category %q", p.CategorySlug)
	}
}

func TestFilterLexical(t *testing.T) {
	got := Filter(Filters{Query: "pytorch"})
	if len(got) == 0 {
		t.Fatal("expected lexical results for 'pytorch'")
	}
	if !strings.Contains(strings.ToLower(got[0].SearchText), "pytorch") {
		t.Fatalf("top result %q does not mention pytorch", got[0].Slug)
	}

	coding := Filter(Filters{Category: "coding-dev"})
	for _, p := range coding {
		if p.CategorySlug != "coding-dev" {
			t.Fatalf("category filter leaked %q", p.Slug)
		}
	}
}

func TestSitemapPaths(t *testing.T) {
	paths := SitemapPaths()
	if len(paths) < Count() {
		t.Fatalf("sitemap has %d paths, want >= %d", len(paths), Count())
	}
	if paths[0] != "/prompts" {
		t.Fatalf("first sitemap path = %q, want /prompts", paths[0])
	}
}

// stubEmbedder produces a deterministic pseudo-embedding from token hashes so
// similar text yields similar vectors — enough to exercise the index.
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

func TestSemanticIndex(t *testing.T) {
	ix := New(stubEmbedder{})
	ix.Rebuild(context.Background())
	if !ix.Ready() {
		t.Fatalf("index not ready: %+v", ix.Status())
	}

	results, err := ix.Search(context.Background(), "pytorch deep learning code", Filters{}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected semantic results")
	}

	// Related must return without the seed itself and respect the limit.
	related := ix.Related("autocomplete-pytorch-coding", 3)
	if len(related) == 0 || len(related) > 3 {
		t.Fatalf("unexpected related count %d", len(related))
	}
	for _, p := range related {
		if p.Slug == "autocomplete-pytorch-coding" {
			t.Fatal("related results include the seed prompt")
		}
	}

	// Filters apply within semantic search.
	codingOnly, err := ix.Search(context.Background(), "tests", Filters{Category: "coding-dev"}, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range codingOnly {
		if r.CategorySlug != "coding-dev" {
			t.Fatalf("semantic filter leaked %q", r.Slug)
		}
	}
}

func TestRelatedFallbackWhenNotReady(t *testing.T) {
	ix := New(nil) // no embedder -> not ready -> metadata fallback
	related := ix.Related("autocomplete-pytorch-coding", 4)
	if len(related) == 0 {
		t.Fatal("expected metadata-based related fallback")
	}
}
