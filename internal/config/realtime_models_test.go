package config

import "testing"

func TestRealtimeVoiceModels(t *testing.T) {
	byName := loadAuditConfig(t)
	cases := []struct {
		id        string
		input     float64
		output    float64
		context   int
		maxOutput int
	}{
		{id: "gpt-realtime-2.1", input: 4, output: 24, context: 128000, maxOutput: 32000},
		{id: "gpt-realtime-2.1-mini", input: 0.6, output: 2.4, context: 128000, maxOutput: 32000},
	}
	for _, tc := range cases {
		model := byName[tc.id]
		if model == nil {
			t.Errorf("%s missing from config.yaml", tc.id)
			continue
		}
		if model.InputPricePer1M != tc.input || model.OutputPricePer1M != tc.output {
			t.Errorf("%s text pricing = %v/%v, want %v/%v", tc.id, model.InputPricePer1M, model.OutputPricePer1M, tc.input, tc.output)
		}
		if model.ContextWindow != tc.context || model.MaxOutputTokens != tc.maxOutput {
			t.Errorf("%s limits = %d/%d, want %d/%d", tc.id, model.ContextWindow, model.MaxOutputTokens, tc.context, tc.maxOutput)
		}
		if !model.SupportsStreaming || !model.SupportsTools || !model.SupportsVision {
			t.Errorf("%s must advertise streaming, tools, and image input", tc.id)
		}
	}
}
