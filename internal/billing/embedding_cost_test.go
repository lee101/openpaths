package billing

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestEmbeddingBillionBillsOneDollar(t *testing.T) {
	pt := NewPricingTable([]model.ModelConfig{{ID: "openpaths-embed", PricePer1MEmbeddings: 0.001}})
	var carry, total, n int64
	sizes := []int{1, 7, 100, 2048}
	for i := 0; n < 1_000_000_000; i++ {
		c := sizes[i%len(sizes)]
		if rem := 1_000_000_000 - n; int64(c) > rem {
			c = int(rem)
		}
		sub, ok := pt.EmbeddingSubUnits("openpaths-embed", c)
		if !ok {
			t.Fatal("not per-embedding priced")
		}
		carry += sub
		total += carry / SubUnitsPerUnit
		carry %= SubUnitsPerUnit
		n += int64(c)
	}
	if d := total - 10000; d < -1 || d > 1 {
		t.Fatalf("1B embeddings billed %d units, want 10000", total)
	}
}

func TestEmbeddingEstimateAndOtherModels(t *testing.T) {
	pt := NewPricingTable([]model.ModelConfig{
		{ID: "e", PricePer1MEmbeddings: 0.001},
		{ID: "r", PricePerRequest: 0.001},
	})
	if c, _ := pt.EstimateMaxCost("e", 0); c != 1 {
		t.Fatalf("estimate = %d", c)
	}
	if _, ok := pt.EmbeddingSubUnits("r", 5); ok {
		t.Fatal("per-request model must not use embedding path")
	}
	e := &Engine{pricing: pt}
	if _, handled, _ := e.DeductEmbeddings(nil, "u", "r", 5); handled {
		t.Fatal("handled non-embedding model")
	}
	cost, handled, err := e.DeductEmbeddings(nil, "u", "e", 100)
	if !handled || cost != 0 || err != nil {
		t.Fatalf("small batch = %d %v %v", cost, handled, err)
	}
}
