package config

import (
	"path/filepath"
	"testing"
)

func TestGeminiFlashRoutesUse38Except25(t *testing.T) {
	cfg, err := Load(filepath.Join("..", "..", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"gemini-3.8-flash": "gemini-3.8-flash",
		"gemini-3.7-flash": "gemini-3.8-flash",
		"gemini-3.6-flash": "gemini-3.8-flash",
		"gemini-3.5-flash": "gemini-3.8-flash",
		"gemini-latest":    "gemini-3.8-flash",
		"gemini-2.5-flash": "gemini-2.5-flash",
	}
	for _, m := range cfg.Models {
		if upstream, ok := want[m.ID]; ok {
			if m.ProviderModelID != upstream {
				t.Errorf("%s provider_model_id = %q, want %q", m.ID, m.ProviderModelID, upstream)
			}
			delete(want, m.ID)
		}
	}
	for id := range want {
		t.Errorf("model %s missing from config", id)
	}
}
