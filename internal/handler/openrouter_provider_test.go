package handler

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestConfigToORModelSkipsOpenRouterProxiedModels(t *testing.T) {
	for _, cfg := range []*model.ModelConfig{
		{ID: "or/stepfun-flash", Provider: "openrouter"},
		{ID: "qwen/qwen3.7-plus", Provider: "openrouter"},
	} {
		if got := configToORModel(cfg); got != nil {
			t.Fatalf("configToORModel(%q, %q) = %#v, want nil", cfg.ID, cfg.Provider, got)
		}
	}
}
