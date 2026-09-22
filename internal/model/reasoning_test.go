package model

import "testing"

func TestCompatibleReasoningEffort(t *testing.T) {
	tests := []struct{ model, requested, want string }{
		{"gpt-5.6-sol", "minimal", "low"},
		{"gpt-5.6-terra", "high", "high"},
		{"gpt-5.6-luna", " minimal ", "low"},
		{"other-model", "minimal", "minimal"},
		{"gpt-5.6-sol", "future", "future"},
		{"gpt-5.6-sol", "max", "xhigh"},
		{"gpt-5.6-terra", "xhigh", "xhigh"},
		// GPT-6 carries the GPT-5.6 effort vocabulary on chat completions:
		// "max" is a Responses-API-only value, and "minimal" is rejected too.
		{"gpt-6-sol", "max", "xhigh"},
		{"gpt-6-sol", "minimal", "low"},
		{"gpt-6-sol", "xhigh", "xhigh"},
		{"gpt-6-luna", "none", "none"},
		{"gpt-6-luna", " future ", "future"},
		// Grok via OpenRouter cannot disable reasoning; xAI direct can.
		{"x-ai/grok-4.6", "none", "minimal"},
		{"x-ai/grok-4.6", "max", "max"},
		{"grok-4.6", "none", "none"},
		// GLM-5.3 cannot disable thinking and has no medium/xhigh tier.
		{"glm-5.3", "none", "low"},
		{"glm-5.3", "minimal", "low"},
		{"glm-5.3", "low", "low"},
		{"glm-5.3", "medium", "high"},
		{"glm-5.3", "high", "high"},
		{"glm-5.3", "xhigh", "max"},
		{"glm-5.3", "max", "max"},
		{"glm-5.3", "", ""},
		{"glm-5.3-flash", "none", "low"},
		{"z-ai/glm-5.3-flash", "xhigh", "max"},
		// The same model reached through a namespaced provider route.
		{"z-ai/glm-5.3", "none", "low"},
		{"accounts/fireworks/models/glm-5.3", "medium", "high"},
		// Neighbouring GLM versions keep the full vocabulary.
		{"glm-5.2", "none", "none"},
		{"z-ai/glm-5.1", "medium", "medium"},
		// Gemini 3.7 via OpenRouter cannot disable reasoning; direct can.
		{"google/gemini-3.7-flash", "none", "low"},
		{"google/gemini-3.7-flash", "minimal", "low"},
		{"google/gemini-3.7-flash", "max", "high"},
		{"google/gemini-3.7-flash", "medium", "medium"},
		{"gemini-3.7-flash", "none", "none"},
		{"gemini-3.7-flash", "minimal", "low"},
		{"gemini-3.7-flash", "max", "high"},
		{"gemini-3.5-flash", "minimal", "minimal"},
		// Magistral exposes only none and high. "low" is equidistant between
		// them, and the tie-break prefers more thought over silently disabling
		// reasoning on a reasoning model.
		{"magistral-medium-latest", "low", "high"},
		{"magistral-medium-latest", "medium", "high"},
		{"magistral-small-latest", "max", "high"},
		{"magistral-medium-latest", "none", "none"},
		// Every other Mistral family rejects the parameter outright.
		{"mistral-large-latest", "high", ""},
		{"mistral-small-latest", "none", ""},
		{"codestral-latest", "medium", ""},
		{"pixtral-large-latest", "low", ""},
		{"ministral-8b-latest", "max", ""},
		{"devstral-medium-latest", "high", ""},
		{"open-mistral-nemo", "low", ""},
		// An empty request stays empty rather than becoming a stripped value.
		{"mistral-large-latest", "", ""},
		// Qwen 3.8 2.4T has no minimal/high/max, under either provider spelling.
		{"Qwen/Qwen3.8-2.4T-A95B", "minimal", "low"},
		{"qwen/qwen3.8-2.4t-a95b", "high", "xhigh"},
		{"qwen/qwen3.8-2.4t-a95b", "max", "xhigh"},
		{"accounts/fireworks/models/qwen3p8-2p4t-a95b", "high", "xhigh"},
		{"qwen/qwen3.8-2.4t-a95b", "none", "none"},
		// Qwen 3.8 Max via OpenRouter cannot disable reasoning.
		{"qwen/qwen3.8-max", "none", "minimal"},
		{"qwen/qwen3.8-max", "max", "max"},
		// Qwen 3.8 27B on Cerebras defaults high; none disables it.
		{"qwen-3.8-27b", "none", "none"},
		{"qwen-3.8-27b", "max", "high"},
		{"qwen-3.8-27b", "xhigh", "high"},
		// RunAnywhere serves the same 27B from a vLLM stack that 400s on max
		// and minimal, so those remap; none/low/medium/xhigh pass through.
		{"qwen3.8-27b", "max", "xhigh"},
		{"qwen3.8-27b", "minimal", "low"},
		{"qwen3.8-27b", "none", "none"},
		{"qwen3.8-27b", "medium", "medium"},
		// The app.nz spelling of the family keeps the untouched pass-through.
		{"qwen3.8-27b-uncensored", "max", "max"},
		// GPT-OSS rejects none/minimal/max; nearest supported tier wins.
		{"openai/gpt-oss-20b", "none", "low"},
		{"openai/gpt-oss-120b", "max", "high"},
		{"accounts/fireworks/models/gpt-oss-120b", "minimal", "low"},
	}
	for _, tt := range tests {
		if got := CompatibleReasoningEffort(tt.model, tt.requested); got != tt.want {
			t.Errorf("CompatibleReasoningEffort(%q, %q) = %q, want %q", tt.model, tt.requested, got, tt.want)
		}
	}
}
