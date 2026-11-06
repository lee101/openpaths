package config

import (
	"log"

	"github.com/openpaths/openpaths/internal/model"
)

func modelIsFree(m *model.ModelConfig) bool {
	return m.InputPricePer1M == 0 && m.OutputPricePer1M == 0 &&
		m.PricePerRequest == 0 && m.PricePerImage == 0 && m.PricePerVideo == 0 && m.PricePerSecond == 0 &&
		len(m.PricePerSecondByResolution) == 0 && len(m.PricePerSecondWithVideoInputByResolution) == 0
}

func (c *Config) dropPaidFallbacksFromFree() {
	byID := make(map[string]*model.ModelConfig, len(c.Models))
	for i := range c.Models {
		byID[c.Models[i].ID] = &c.Models[i]
	}
	for i := range c.Models {
		m := &c.Models[i]
		if len(m.FallbackModels) == 0 || !modelIsFree(m) {
			continue
		}
		kept := m.FallbackModels[:0:0]
		for _, f := range m.FallbackModels {
			t, ok := byID[f]
			if !ok || !modelIsFree(t) {
				log.Printf("config: free model %s drops non-free fallback %s", m.ID, f)
				continue
			}
			kept = append(kept, f)
		}
		m.FallbackModels = kept
	}
}
