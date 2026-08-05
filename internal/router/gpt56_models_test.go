package router

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestGPT56AliasesResolveToCanonicalProviderModels(t *testing.T) {
	models := []model.ModelConfig{
		{ID: "gpt-5.6-sol", Provider: "openai", ProviderModelID: "gpt-5.6-sol", Aliases: []string{"gpt5.6-sol"}},
		{ID: "gpt-5.6-terra", Provider: "openai", ProviderModelID: "gpt-5.6-terra", Aliases: []string{"gpt5.6-terra"}},
		{ID: "gpt-5.6-luna", Provider: "openai", ProviderModelID: "gpt-5.6-luna", Aliases: []string{"gpt5.6-luna"}},
	}
	r := newTestRouter(models, "openai")

	for _, tier := range []string{"sol", "terra", "luna"} {
		requested := "gpt5.6-" + tier
		want := "gpt-5.6-" + tier
		candidates, err := r.ResolveWithRetries(requested)
		if err != nil {
			t.Fatalf("resolve %q: %v", requested, err)
		}
		if len(candidates) != 1 || candidates[0].ModelCfg.ProviderModelID != want {
			t.Fatalf("resolve %q = %#v, want provider model %q", requested, candidates, want)
		}
	}
}
