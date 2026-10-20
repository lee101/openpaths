package router

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

// freeTierLadder is the upstream order openpaths/free must degrade in: the
// configured primary, then each fallback hop as scripts/sync-openrouter-models.py
// ranked them, with OpenRouter's own free router as the catch-all last hop.
func freeTierLadder(t *testing.T) []string {
	t.Helper()
	byID := map[string]string{}
	var free *model.ModelConfig
	models := loadRepoConfig(t).Models
	for i := range models {
		byID[models[i].ID] = models[i].ProviderModelID
		if models[i].ID == "openpaths/free" {
			free = &models[i]
		}
	}
	if free == nil {
		t.Fatal("openpaths/free missing from config.yaml")
	}
	want := []string{free.ProviderModelID}
	for _, hop := range free.FallbackModels {
		want = append(want, byID[hop])
	}
	if want[len(want)-1] != "openrouter/free" {
		t.Fatalf("free ladder %v must end on openrouter/free", want)
	}
	return want
}

func freeTierRouter(t *testing.T) *Router {
	t.Helper()
	// Every free hop is an OpenRouter lane; other providers are deliberately
	// absent so a chain that drifted onto a direct provider fails here.
	return newTestRouter(loadRepoConfig(t).Models, "openrouter")
}

func freeTierUpstreams(t *testing.T, r *Router, modelName string) []string {
	t.Helper()
	candidates, err := r.ResolveWithRetries(modelName)
	if err != nil {
		t.Fatalf("ResolveWithRetries(%q) error = %v", modelName, err)
	}
	got := make([]string, 0, len(candidates))
	for _, cand := range candidates {
		if cand.Provider.Name() != "openrouter" {
			t.Fatalf("%s resolved onto provider %q, want the OpenRouter free lane", modelName, cand.Provider.Name())
		}
		got = append(got, cand.ModelCfg.ProviderModelID)
	}
	return got
}

func assertFreeTierLadder(t *testing.T, modelName string, want, got []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s ladder = %v, want %v", modelName, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s ladder = %v, want %v", modelName, got, want)
		}
	}
}

// The advertised entry point is the alias `openpaths-free`, and the first hop
// is the free preview lane — not one of the cheaper small lanes below it.
func TestFreeTierResolvesInLadderOrder(t *testing.T) {
	r := freeTierRouter(t)
	assertFreeTierLadder(t, "openpaths-free", freeTierLadder(t), freeTierUpstreams(t, r, "openpaths-free"))
}

// The chat handler orders candidates by price before trying them. Every free
// hop is zero-priced, so that ordering must leave the capability ladder intact
// rather than reshuffling it onto an arbitrary free lane.
func TestFreeTierLadderSurvivesPriceOrdering(t *testing.T) {
	r := freeTierRouter(t)
	candidates, err := r.ResolveWithRetries("openpaths/free")
	if err != nil {
		t.Fatalf("ResolveWithRetries() error = %v", err)
	}
	ordered := OrderCandidates(candidates, "price")
	got := make([]string, 0, len(ordered))
	for _, cand := range ordered {
		got = append(got, cand.ModelCfg.ProviderModelID)
	}
	assertFreeTierLadder(t, "price-ordered", freeTierLadder(t), got)
}

// A free lane that fails trips the health circuit, and the next resolve must
// open on the following lane instead of retrying the dead one — the step-down
// the free tier exists for. Clearing the circuit puts the lane back in place.
func TestFreeTierCircuitSkipsUnhealthyLane(t *testing.T) {
	r := freeTierRouter(t)
	ladder := freeTierLadder(t)
	second := loadRepoConfigModel(t, "openpaths/free").FallbackModels[0]

	r.MarkModelUnhealthy("openrouter", "openpaths/free")
	r.MarkModelUnhealthy("openrouter", second)
	// The primary hop and the second lane are out; resolution must open on the
	// third lane and continue down the rest of the ladder.
	assertFreeTierLadder(t, "after two failures", ladder[2:], freeTierUpstreams(t, r, "openpaths/free"))

	r.MarkModelHealthy("openrouter", second)
	got := freeTierUpstreams(t, r, "openpaths/free")
	if len(got) == 0 || got[0] != ladder[1] {
		t.Fatalf("ladder after recovery = %v, want the recovered lane %s back in place", got, ladder[1])
	}
}

func loadRepoConfigModel(t *testing.T, id string) *model.ModelConfig {
	t.Helper()
	models := loadRepoConfig(t).Models
	for i := range models {
		if models[i].ID == id {
			return &models[i]
		}
	}
	t.Fatalf("%s missing from config.yaml", id)
	return nil
}
