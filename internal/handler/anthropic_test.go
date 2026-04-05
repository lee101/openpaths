package handler

import "testing"

func TestParseAnthropicThinking(t *testing.T) {
	tests := []struct {
		name     string
		thinking any
		want     string
	}{
		{
			name:     "disabled maps to none",
			thinking: map[string]any{"type": "disabled"},
			want:     "none",
		},
		{
			name:     "low budget",
			thinking: map[string]any{"type": "enabled", "budget_tokens": 1024.0},
			want:     "low",
		},
		{
			name:     "medium budget",
			thinking: map[string]any{"type": "enabled", "budget_tokens": 8192.0},
			want:     "medium",
		},
		{
			name:     "high budget",
			thinking: map[string]any{"type": "enabled", "budget_tokens": 20000.0},
			want:     "high",
		},
		{
			name:     "missing budget ignored",
			thinking: map[string]any{"type": "enabled"},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseAnthropicThinking(tt.thinking); got != tt.want {
				t.Fatalf("parseAnthropicThinking() = %q, want %q", got, tt.want)
			}
		})
	}
}
