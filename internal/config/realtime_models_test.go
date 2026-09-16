package config

import "testing"

func TestRealtimeVoiceModels(t *testing.T) {
	byName := loadAuditConfig(t)
	cases := []struct {
		id                                                           string
		input                                                        float64
		output                                                       float64
		context                                                      int
		maxOutput                                                    int
		cached, audioIn, cachedAudio, audioOut, imageIn, cachedImage float64
	}{
		{id: "gpt-realtime-2.1", input: 4.2, output: 25.2, context: 128000, maxOutput: 32000, cached: .42, audioIn: 33.6, cachedAudio: .42, audioOut: 67.2, imageIn: 5.25, cachedImage: .525},
		{id: "gpt-realtime-2.1-mini", input: .63, output: 2.52, context: 128000, maxOutput: 32000, cached: .063, audioIn: 10.5, cachedAudio: .315, audioOut: 21, imageIn: .84, cachedImage: .084},
		{id: "gemini-3.8-live-extended-thinking", input: .7875, output: 4.725, context: 131072, maxOutput: 65536, audioIn: 3.15, audioOut: 12.6, imageIn: 1.05},
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
		if model.InputCacheHitPricePer1M != tc.cached || model.AudioInputPricePer1M != tc.audioIn ||
			model.AudioInputCacheHitPricePer1M != tc.cachedAudio || model.AudioOutputPricePer1M != tc.audioOut ||
			model.ImageInputPricePer1M != tc.imageIn || model.ImageInputCacheHitPricePer1M != tc.cachedImage {
			t.Errorf("%s realtime pricing was not loaded completely", tc.id)
		}
		if model.ContextWindow != tc.context || model.MaxOutputTokens != tc.maxOutput {
			t.Errorf("%s limits = %d/%d, want %d/%d", tc.id, model.ContextWindow, model.MaxOutputTokens, tc.context, tc.maxOutput)
		}
		if !model.SupportsStreaming || !model.SupportsTools || !model.SupportsVision {
			t.Errorf("%s must advertise streaming, tools, and image input", tc.id)
		}
	}
}
