package billing

import (
	"testing"

	"github.com/openpath/openpath/internal/model"
)

func newTestPricingTable() *PricingTable {
	models := []model.ModelConfig{
		{ID: "gpt-4o", InputPricePer1M: 2.50, OutputPricePer1M: 10.00},
		{ID: "gpt-4o-mini", InputPricePer1M: 0.15, OutputPricePer1M: 0.60},
	}
	return NewPricingTable(models)
}

func TestNewPricingTable_IndexesModels(t *testing.T) {
	pt := newTestPricingTable()

	if _, ok := pt.models["gpt-4o"]; !ok {
		t.Error("expected gpt-4o to be present in pricing table")
	}
	if _, ok := pt.models["gpt-4o-mini"]; !ok {
		t.Error("expected gpt-4o-mini to be present in pricing table")
	}
	if _, ok := pt.models["nonexistent"]; ok {
		t.Error("expected nonexistent model not to be in pricing table")
	}
}

func TestCalculateCost(t *testing.T) {
	pt := newTestPricingTable()

	tests := []struct {
		name         string
		modelID      string
		inputTokens  int
		outputTokens int
		wantCost     int64
		wantErr      bool
	}{
		{
			name:         "gpt-4o known usage",
			modelID:      "gpt-4o",
			inputTokens:  1000,
			outputTokens: 500,
			// input: 1000 * 2.50 / 1_000_000 = 0.0025
			// output: 500 * 10.00 / 1_000_000 = 0.005
			// total dollars: 0.0075
			// hundredths-of-cent: 0.0075 * 10000 = 75
			wantCost: 75,
			wantErr:  false,
		},
		{
			name:         "gpt-4o-mini known usage",
			modelID:      "gpt-4o-mini",
			inputTokens:  10000,
			outputTokens: 2000,
			// input: 10000 * 0.15 / 1_000_000 = 0.0015
			// output: 2000 * 0.60 / 1_000_000 = 0.0012
			// total dollars: 0.0027
			// hundredths-of-cent: 0.0027 * 10000 = 27
			wantCost: 27,
			wantErr:  false,
		},
		{
			name:         "gpt-4o large token count",
			modelID:      "gpt-4o",
			inputTokens:  1_000_000,
			outputTokens: 100_000,
			// input: 1_000_000 * 2.50 / 1_000_000 = 2.50
			// output: 100_000 * 10.00 / 1_000_000 = 1.00
			// total dollars: 3.50
			// hundredths-of-cent: 3.50 * 10000 = 35000
			wantCost: 35000,
			wantErr:  false,
		},
		{
			name:         "zero tokens returns zero cost",
			modelID:      "gpt-4o",
			inputTokens:  0,
			outputTokens: 0,
			wantCost:     0,
			wantErr:      false,
		},
		{
			name:         "zero input tokens with output",
			modelID:      "gpt-4o",
			inputTokens:  0,
			outputTokens: 1000,
			// output: 1000 * 10.00 / 1_000_000 = 0.01
			// hundredths-of-cent: 0.01 * 10000 = 100
			wantCost: 100,
			wantErr:  false,
		},
		{
			name:         "zero output tokens with input",
			modelID:      "gpt-4o",
			inputTokens:  1000,
			outputTokens: 0,
			// input: 1000 * 2.50 / 1_000_000 = 0.0025
			// hundredths-of-cent: 0.0025 * 10000 = 25
			wantCost: 25,
			wantErr:  false,
		},
		{
			name:         "minimum charge for tiny usage",
			modelID:      "gpt-4o-mini",
			inputTokens:  1,
			outputTokens: 0,
			// input: 1 * 0.15 / 1_000_000 = 0.00000015
			// hundredths-of-cent: 0.00000015 * 10000 = 0.0015 -> truncates to 0
			// minimum charge kicks in because inputTokens > 0
			wantCost: 1,
			wantErr:  false,
		},
		{
			name:         "minimum charge for tiny output only",
			modelID:      "gpt-4o-mini",
			inputTokens:  0,
			outputTokens: 1,
			// output: 1 * 0.60 / 1_000_000 = 0.0000006
			// hundredths-of-cent: 0.0000006 * 10000 = 0.006 -> truncates to 0
			// minimum charge kicks in because outputTokens > 0
			wantCost: 1,
			wantErr:  false,
		},
		{
			name:         "minimum charge for tiny input and output",
			modelID:      "gpt-4o-mini",
			inputTokens:  1,
			outputTokens: 1,
			// same tiny amounts, truncates to 0, minimum charge applies
			wantCost: 1,
			wantErr:  false,
		},
		{
			name:         "unknown model returns error",
			modelID:      "unknown-model",
			inputTokens:  100,
			outputTokens: 100,
			wantCost:     0,
			wantErr:      true,
		},
		{
			name:         "unknown model with zero tokens still errors",
			modelID:      "nonexistent",
			inputTokens:  0,
			outputTokens: 0,
			wantCost:     0,
			wantErr:      true,
		},
		{
			name:         "gpt-4o-mini exact boundary no minimum charge",
			modelID:      "gpt-4o",
			inputTokens:  400,
			outputTokens: 0,
			// input: 400 * 2.50 / 1_000_000 = 0.001
			// hundredths-of-cent: 0.001 * 10000 = 10
			wantCost: 10,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := pt.CalculateCost(tt.modelID, tt.inputTokens, tt.outputTokens)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cost != tt.wantCost {
				t.Errorf("got cost %d, want %d", cost, tt.wantCost)
			}
		})
	}
}

func TestCalculateCost_ErrorMessageContainsModelID(t *testing.T) {
	pt := newTestPricingTable()
	_, err := pt.CalculateCost("mystery-model", 100, 100)
	if err == nil {
		t.Fatal("expected error for unknown model")
	}
	if got := err.Error(); got != `unknown model "mystery-model" for pricing` {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestEstimateMaxCost(t *testing.T) {
	pt := newTestPricingTable()

	tests := []struct {
		name            string
		modelID         string
		maxOutputTokens int
		wantCost        int64
		wantErr         bool
	}{
		{
			name:            "uses provided maxOutputTokens",
			modelID:         "gpt-4o",
			maxOutputTokens: 1000,
			// equivalent to CalculateCost("gpt-4o", 0, 1000)
			// output: 1000 * 10.00 / 1_000_000 = 0.01
			// hundredths-of-cent: 0.01 * 10000 = 100
			wantCost: 100,
			wantErr:  false,
		},
		{
			name:            "defaults to 4096 when maxOutput is 0",
			modelID:         "gpt-4o",
			maxOutputTokens: 0,
			// equivalent to CalculateCost("gpt-4o", 0, 4096)
			// output: 4096 * 10.00 / 1_000_000 = 0.04096
			// hundredths-of-cent: 0.04096 * 10000 = 409 (truncated from 409.6)
			wantCost: 409,
			wantErr:  false,
		},
		{
			name:            "defaults to 4096 when maxOutput is negative",
			modelID:         "gpt-4o",
			maxOutputTokens: -1,
			// same as zero case: defaults to 4096
			wantCost: 409,
			wantErr:  false,
		},
		{
			name:            "gpt-4o-mini default estimation",
			modelID:         "gpt-4o-mini",
			maxOutputTokens: 0,
			// equivalent to CalculateCost("gpt-4o-mini", 0, 4096)
			// output: 4096 * 0.60 / 1_000_000 = 0.0024576
			// hundredths-of-cent: 0.0024576 * 10000 = 24 (truncated from 24.576)
			wantCost: 24,
			wantErr:  false,
		},
		{
			name:            "gpt-4o-mini with explicit max",
			modelID:         "gpt-4o-mini",
			maxOutputTokens: 8192,
			// output: 8192 * 0.60 / 1_000_000 = 0.0049152
			// hundredths-of-cent: 0.0049152 * 10000 = 49 (truncated from 49.152)
			wantCost: 49,
			wantErr:  false,
		},
		{
			name:            "unknown model returns error",
			modelID:         "unknown-model",
			maxOutputTokens: 1000,
			wantCost:        0,
			wantErr:         true,
		},
		{
			name:            "unknown model with zero max returns error",
			modelID:         "unknown-model",
			maxOutputTokens: 0,
			wantCost:        0,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost, err := pt.EstimateMaxCost(tt.modelID, tt.maxOutputTokens)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cost != tt.wantCost {
				t.Errorf("got cost %d, want %d", cost, tt.wantCost)
			}
		})
	}
}

func TestEstimateMaxCost_ConsistentWithCalculateCost(t *testing.T) {
	pt := newTestPricingTable()

	// EstimateMaxCost(modelID, n) should equal CalculateCost(modelID, 0, n) for n > 0
	for _, modelID := range []string{"gpt-4o", "gpt-4o-mini"} {
		for _, maxOutput := range []int{100, 1000, 4096, 8192, 16384} {
			estimated, err := pt.EstimateMaxCost(modelID, maxOutput)
			if err != nil {
				t.Fatalf("EstimateMaxCost(%s, %d) error: %v", modelID, maxOutput, err)
			}

			calculated, err := pt.CalculateCost(modelID, 0, maxOutput)
			if err != nil {
				t.Fatalf("CalculateCost(%s, 0, %d) error: %v", modelID, maxOutput, err)
			}

			if estimated != calculated {
				t.Errorf("EstimateMaxCost(%s, %d) = %d, CalculateCost(%s, 0, %d) = %d; expected equal",
					modelID, maxOutput, estimated, modelID, maxOutput, calculated)
			}
		}
	}
}

func TestNewPricingTable_EmptyModels(t *testing.T) {
	pt := NewPricingTable(nil)
	if len(pt.models) != 0 {
		t.Errorf("expected empty pricing table, got %d entries", len(pt.models))
	}

	_, err := pt.CalculateCost("anything", 100, 100)
	if err == nil {
		t.Error("expected error from empty pricing table")
	}
}

func TestNewPricingTable_DuplicateModelIDs(t *testing.T) {
	models := []model.ModelConfig{
		{ID: "dup-model", InputPricePer1M: 1.00, OutputPricePer1M: 2.00},
		{ID: "dup-model", InputPricePer1M: 5.00, OutputPricePer1M: 10.00},
	}
	pt := NewPricingTable(models)

	// The last entry should win since the loop overwrites
	cost, err := pt.CalculateCost("dup-model", 1_000_000, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// input: 1_000_000 * 5.00 / 1_000_000 = 5.00
	// hundredths-of-cent: 5.00 * 10000 = 50000
	if cost != 50000 {
		t.Errorf("expected last-wins behavior for duplicate IDs: got %d, want 50000", cost)
	}
}

func TestCalculateCost_LargeTokenCounts(t *testing.T) {
	pt := newTestPricingTable()

	// 10M input, 1M output for gpt-4o
	cost, err := pt.CalculateCost("gpt-4o", 10_000_000, 1_000_000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// input: 10_000_000 * 2.50 / 1_000_000 = 25.00
	// output: 1_000_000 * 10.00 / 1_000_000 = 10.00
	// total dollars: 35.00
	// hundredths-of-cent: 35.00 * 10000 = 350000
	if cost != 350000 {
		t.Errorf("got cost %d, want 350000", cost)
	}
}
