package router

import "testing"

// freeTierLadder is the upstream order openpaths/free must degrade in: the
// free preview frontier lane first, then the zero-priced OpenRouter lanes by
// capability, with OpenRouter's own free router as the catch-all last hop.
var freeTierLadder = []string{
	"stealth/space-bunny-alpha",
	"nvidia/nemotron-3-ultra-550b-a55b:free",
	"nex-agi/nex-n2.5-pro:free",
	"nvidia/nemotron-3-super-120b-a12b:free",
	"qwen/qwen3.8-27b:free",
	"cohere/north-mini-code:free",
	"nvidia/nemotron-3.5-lightning:free",
	"google/gemma-4-31b-it:free",
	"nvidia/nemotron-3-nano-30b-a3b:free",
	"openrouter/free",
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
	t.Skip("pending config: openpaths/free ladder; prod WIP test ahead of prod config")
	r := freeTierRouter(t)
	assertFreeTierLadder(t, "openpaths-free", freeTierLadder, freeTierUpstreams(t, r, "openpaths-free"))
}

// The chat handler orders candidates by price before trying them. Every free
// hop is zero-priced, so that ordering must leave the capability ladder intact
// rather than reshuffling it onto an arbitrary free lane.
func TestFreeTierLadderSurvivesPriceOrdering(t *testing.T) {
	t.Skip("pending config: openpaths/free ladder; prod WIP test ahead of prod config")
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
	assertFreeTierLadder(t, "price-ordered", freeTierLadder, got)
}

// A free lane that fails trips the health circuit, and the next resolve must
// open on the following lane instead of retrying the dead one — the step-down
// the free tier exists for. Clearing the circuit puts the lane back in place.
func TestFreeTierCircuitSkipsUnhealthyLane(t *testing.T) {
	t.Skip("pending config: openpaths/free ladder; prod WIP test ahead of prod config")
	r := freeTierRouter(t)

	r.MarkModelUnhealthy("openrouter", "openpaths/free")
	r.MarkModelUnhealthy("openrouter", "or/nemotron-ultra-free")
	// The primary hop and the second lane are out; resolution must open on the
	// third lane and continue down the rest of the ladder.
	assertFreeTierLadder(t, "after two failures", freeTierLadder[2:], freeTierUpstreams(t, r, "openpaths/free"))

	r.MarkModelHealthy("openrouter", "or/nemotron-ultra-free")
	got := freeTierUpstreams(t, r, "openpaths/free")
	if len(got) == 0 || got[0] != "nvidia/nemotron-3-ultra-550b-a55b:free" {
		t.Fatalf("ladder after recovery = %v, want the recovered lane back in place", got)
	}
}
