package config

import "testing"

func TestAlibabaDirectModels(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]int{}
	for i, m := range cfg.Models {
		byID[m.ID] = i
	}
	for _, want := range []struct {
		id, provider, upstream string
		in, out                float64
		fallback               string
	}{
		{"qwen/qwen3.8-max", "qwen", "qwen3.8-max", 2, 6, "or/qwen3.8-max"},
		{"qwen/qwen3.8-max-0902", "qwen", "qwen3.8-max-0902", 2, 6, "qwen/qwen3.8-max"},
		{"qwen/qwen3.8-omni-flash", "qwen", "qwen3.8-omni-flash", .15, .47, "qwen/qwen3.8-flash"},
		{"qwen/qwen3.8-27b", "qwen", "qwen3.8-27b", .5, 3, "runanywhere/qwen3.8-27b"},
		{"kimi-k3", "qwen", "kimi-k3", 3, 15, "or/kimi-k3"},
		{"decision-model-preview", "alibaba", "decision-model-preview", .042, 0, ""},
		{"openpaths-decision", "gobed", "openpaths-decision", .002, 0, ""},
	} {
		i, ok := byID[want.id]
		if !ok {
			t.Errorf("%s missing", want.id)
			continue
		}
		m := cfg.Models[i]
		if m.Provider != want.provider || m.ProviderModelID != want.upstream || m.InputPricePer1M != want.in || m.OutputPricePer1M != want.out {
			t.Errorf("%s = %s/%s %v/%v", want.id, m.Provider, m.ProviderModelID, m.InputPricePer1M, m.OutputPricePer1M)
		}
		if want.fallback != "" {
			if len(m.FallbackModels) == 0 || m.FallbackModels[0] != want.fallback {
				t.Errorf("%s fallbacks = %v, want %s first", want.id, m.FallbackModels, want.fallback)
			}
			if _, ok := byID[want.fallback]; !ok {
				t.Errorf("%s fallback %s is not a configured model", want.id, want.fallback)
			}
		}
	}
}
