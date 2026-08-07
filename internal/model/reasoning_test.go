package model

import "testing"

func TestCompatibleReasoningEffort(t *testing.T) {
	tests := []struct{ model, requested, want string }{
		{"gpt-5.6-sol", "minimal", "low"},
		{"gpt-5.6-terra", "high", "high"},
		{"gpt-5.6-luna", " minimal ", "low"},
		{"other-model", "minimal", "minimal"},
		{"gpt-5.6-sol", "future", "future"},
	}
	for _, tt := range tests {
		if got := CompatibleReasoningEffort(tt.model, tt.requested); got != tt.want {
			t.Errorf("CompatibleReasoningEffort(%q, %q) = %q, want %q", tt.model, tt.requested, got, tt.want)
		}
	}
}
