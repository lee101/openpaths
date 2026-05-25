package artindex

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

type fakeEmbedder struct{}

func (fakeEmbedder) Name() string { return "gobed" }

func (fakeEmbedder) Embed(_ context.Context, req *model.EmbeddingRequest) (*model.EmbeddingResponse, error) {
	var inputs []string
	switch v := req.Input.(type) {
	case string:
		inputs = []string{v}
	case []string:
		inputs = v
	default:
		inputs = []string{}
	}
	data := make([]model.EmbeddingData, 0, len(inputs))
	for i, input := range inputs {
		data = append(data, model.EmbeddingData{
			Object:    "embedding",
			Index:     i,
			Embedding: testEmbedding(input),
		})
	}
	return &model.EmbeddingResponse{Object: "list", Data: data, Model: req.Model}, nil
}

func testEmbedding(input string) []float64 {
	var z, city, nature float64
	for _, word := range []string{"anime", "manga", "character", "koi", "lantern"} {
		if contains(input, word) {
			z += 1
		}
	}
	for _, word := range []string{"city", "neon", "street"} {
		if contains(input, word) {
			city += 1
		}
	}
	for _, word := range []string{"forest", "moss", "river"} {
		if contains(input, word) {
			nature += 1
		}
	}
	return []float64{z, city, nature}
}

func contains(s, needle string) bool {
	return strings.Contains(strings.ToLower(s), needle)
}

func TestServiceBuildsAndSearchesManifestChunks(t *testing.T) {
	items := []Item{
		{ID: "1", Slug: "koi", Prompt: "anime character waiting by lantern koi water", ImageURL: "https://cdn.test/1.webp", Model: "zimage"},
		{ID: "2", Slug: "city", Prompt: "neon city street photography", ImageURL: "https://cdn.test/2.webp", Model: "zimage"},
		{ID: "3", Slug: "forest", Prompt: "moss forest river concept art", ImageURL: "https://cdn.test/3.webp", Model: "zimage"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/manifest.json":
			_ = json.NewEncoder(w).Encode(Manifest{
				Version: 1,
				Kind:    "zimage-art-index",
				Model:   "zimage",
				Count:   len(items),
				Chunks:  []Chunk{{Path: "chunk-000.json", Count: len(items)}},
			})
		case "/chunk-000.json":
			_ = json.NewEncoder(w).Encode(items)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	service := New(srv.URL+"/manifest.json", fakeEmbedder{})
	service.Rebuild(context.Background())

	status := service.Status()
	if !status.Ready || status.Items != 3 {
		t.Fatalf("status = %+v, want ready with 3 items", status)
	}

	results, err := service.Search(context.Background(), "anime manga lantern", 2)
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].ID != "1" {
		t.Fatalf("top result ID = %q, want 1", results[0].ID)
	}
}

func TestResolveChunkURL(t *testing.T) {
	got, err := resolveChunkURL("https://cdn.test/static/data/zimage-art/manifest.json", "", Chunk{Path: "chunks/0.json"})
	if err != nil {
		t.Fatal(err)
	}
	want := "https://cdn.test/static/data/zimage-art/chunks/0.json"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	got, err = resolveChunkURL("https://ignored.test/manifest.json", "https://cdn.test/base", Chunk{Path: "chunks/0.json"})
	if err != nil {
		t.Fatal(err)
	}
	want = "https://cdn.test/base/chunks/0.json"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
