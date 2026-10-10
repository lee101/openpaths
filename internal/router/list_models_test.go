package router

import (
	"fmt"
	"sort"
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func listTestModels(n int) []model.ModelConfig {
	models := make([]model.ModelConfig, n)
	for i := range models {
		models[i] = model.ModelConfig{ID: fmt.Sprintf("m-%04d", n-i), Provider: "p", ProviderModelID: "x", InputPricePer1M: 1, OutputPricePer1M: 2}
	}
	return models
}

func TestListModelsSortedAndMatchesGetModelInfo(t *testing.T) {
	r := newTestRouter(listTestModels(20), "p")
	got := r.ListModels()
	if len(got) != 20 {
		t.Fatalf("len = %d", len(got))
	}
	if !sort.SliceIsSorted(got, func(i, j int) bool { return got[i].ID < got[j].ID }) {
		t.Fatal("ListModels not sorted by ID")
	}
	one, ok := r.GetModelInfo(got[3].ID)
	if !ok || one.ID != got[3].ID || one.Pricing.InputPer1M != got[3].Pricing.InputPer1M {
		t.Fatalf("GetModelInfo mismatch: %+v", one)
	}
}

func BenchmarkListModels(b *testing.B) {
	r := newTestRouter(listTestModels(500), "p")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		r.ListModels()
	}
}
