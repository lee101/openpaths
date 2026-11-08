package config

import "github.com/openpaths/openpaths/internal/model"

func mk(id string, in float64, fb ...string) model.ModelConfig {
	return model.ModelConfig{ID: id, InputPricePer1M: in, FallbackModels: fb}
}
