package config

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestFreeModelsNeverFallBackToPaid(t *testing.T) {
	data, err := os.ReadFile("../../config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var raw Config
	if err := yaml.Unmarshal([]byte(os.ExpandEnv(string(data))), &raw); err != nil {
		t.Fatal(err)
	}
	free := map[string]bool{}
	for i := range raw.Models {
		free[raw.Models[i].ID] = modelIsFree(&raw.Models[i])
	}
	for i := range raw.Models {
		m := &raw.Models[i]
		if !modelIsFree(m) {
			continue
		}
		for _, f := range m.FallbackModels {
			if !free[f] {
				t.Errorf("%s -> %s", m.ID, f)
			}
		}
	}
}

func TestDropPaidFallbacks(t *testing.T) {
	c := &Config{}
	c.Models = append(c.Models, mk("a", 0, "b", "c", "gone"), mk("b", 0), mk("c", 1))
	c.dropPaidFallbacksFromFree()
	if got := c.Models[0].FallbackModels; len(got) != 1 || got[0] != "b" {
		t.Fatalf("got %v", got)
	}
}
