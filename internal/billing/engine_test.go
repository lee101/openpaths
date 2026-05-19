package billing

import "testing"

func TestFormatUsageDescription(t *testing.T) {
	got := formatUsageDescription("gpt-5.4-mini", 120, 340, "medium")
	want := "Model: gpt-5.4-mini, reasoning: medium, in: 120, out: 340"
	if got != want {
		t.Fatalf("formatUsageDescription() = %q, want %q", got, want)
	}
}

func TestFormatUsageDescriptionWithoutReasoning(t *testing.T) {
	got := formatUsageDescription("gemini-3.1-flash-lite", 12, 3, "")
	want := "Model: gemini-3.1-flash-lite, in: 12, out: 3"
	if got != want {
		t.Fatalf("formatUsageDescription() = %q, want %q", got, want)
	}
}
