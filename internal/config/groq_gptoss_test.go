package config

import "testing"

func TestGroqGptOssModels(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	byID := map[string]string{}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		byID[m.ID] = m.Provider + "/" + m.ProviderModelID
	}

	for id, route := range map[string]string{
		"groq/gpt-oss-120b": "groq/openai/gpt-oss-120b",
		"groq/gpt-oss-20b":  "groq/openai/gpt-oss-20b",
	} {
		if got := byID[id]; got != route {
			t.Errorf("%s routes %q, want %q", id, got, route)
		}
	}

	for _, id := range []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768"} {
		found := false
		for i := range cfg.Models {
			m := &cfg.Models[i]
			if m.ID != id {
				continue
			}
			found = true
			if !m.Deprecated {
				t.Errorf("%s should be deprecated (Groq decommissioned it 2026-08-16)", id)
			}
		}
		if !found {
			t.Errorf("compatibility id %s missing from config", id)
		}
	}
}
