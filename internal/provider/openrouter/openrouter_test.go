package openrouter

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

func TestSanitizeSamplingParametersByUpstreamModel(t *testing.T) {
	tests := []struct {
		name            string
		modelID         string
		wantTemperature bool
		wantTopP        bool
		wantPenalties   bool
		wantStop        bool
		wantTempValue   float64
	}{
		{name: "OpenAI reasoning", modelID: "openai/gpt-5.6-sol"},
		// Same upstream rejection profile as the 5.6 tiers: non-default
		// temperature is only accepted with reasoning disabled, so the gateway
		// drops it rather than risk an upstream 400.
		{name: "OpenAI GPT-6 reasoning", modelID: "openai/gpt-6-luna"},
		{name: "xAI reasoning", modelID: "x-ai/grok-4.6", wantTemperature: true, wantTopP: true, wantTempValue: 2},
		{name: "xAI non reasoning", modelID: "x-ai/grok-4.20-non-reasoning", wantTemperature: true, wantTopP: true, wantTempValue: 2},
		{name: "ZAI GLM", modelID: "z-ai/glm-5.3", wantTemperature: true, wantTopP: true, wantStop: true, wantTempValue: 1},
		{name: "ordinary model", modelID: "meta-llama/llama-3.3-70b", wantTemperature: true, wantTopP: true, wantPenalties: true, wantStop: true, wantTempValue: 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			temperature, topP, penalty := 2.0, 0.9, 0.4
			req := &model.ChatCompletionRequest{
				Model: tc.modelID, Temperature: &temperature, TopP: &topP,
				PresencePenalty: &penalty, FrequencyPenalty: &penalty, Stop: []string{"done"},
			}
			sanitizeForOpenRouter(req)
			if (req.Temperature != nil) != tc.wantTemperature || (req.TopP != nil) != tc.wantTopP ||
				(req.PresencePenalty != nil && req.FrequencyPenalty != nil) != tc.wantPenalties ||
				(len(req.Stop) > 0) != tc.wantStop {
				t.Fatalf("sanitized request = %#v", req)
			}
			if tc.wantTemperature && *req.Temperature != tc.wantTempValue {
				t.Fatalf("temperature = %v, want %v", *req.Temperature, tc.wantTempValue)
			}
		})
	}
}

func TestSanitizeFableNoneReasoningToLow(t *testing.T) {
	req := &model.ChatCompletionRequest{
		Model: "anthropic/claude-fable-5.1", ReasoningEffort: "none",
	}
	sanitizeForOpenRouter(req)
	if req.ReasoningEffort != "low" {
		t.Fatalf("reasoning_effort = %q, want low", req.ReasoningEffort)
	}
}
