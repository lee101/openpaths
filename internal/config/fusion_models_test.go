package config

import "testing"

// fusionPanelModelIDs mirrors the openpathsId values shipped by the Fusion space
// (src/pages/Fusion.tsx MODEL_OPTIONS). The /fusion page sends these as native
// OpenPaths model ids to /v1/chat/completions, so every one must resolve to a
// real model id or alias. Keep this list in sync when editing MODEL_OPTIONS.
var fusionPanelModelIDs = []string{
	"claude-opus-latest",
	"gpt-5.5",
	"gemini-latest",
	"gemini-3.5-flash",
	"deepseek-v4-flash",
	"kimi-k2.5",
	"grok-4.3",
	"qwen3.5-397b",
	"glm-5.2",
}

func TestFusionPanelModelIDsResolve(t *testing.T) {
	cfg, err := Load("../../config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	known := map[string]bool{}
	for i := range cfg.Models {
		m := &cfg.Models[i]
		known[m.ID] = true
		for _, a := range m.Aliases {
			known[a] = true
		}
	}
	for _, id := range fusionPanelModelIDs {
		if !known[id] {
			t.Errorf("fusion panel model id %q does not resolve to any config model id or alias", id)
		}
	}
}
