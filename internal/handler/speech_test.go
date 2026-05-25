package handler

import "testing"

func TestEstimateSpeechInputTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{name: "empty", input: "  ", want: 0},
		{name: "short", input: "hey", want: 1},
		{name: "rounds up", input: "hello", want: 2},
		{name: "unicode", input: "héllo", want: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := estimateSpeechInputTokens(tt.input); got != tt.want {
				t.Fatalf("estimateSpeechInputTokens(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
