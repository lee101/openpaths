package queries

import "testing"

func TestGlobMatch(t *testing.T) {
	cases := []struct {
		glob, model string
		want        bool
	}{
		{"*", "anything", true},
		{"anthropic/*", "anthropic/claude-opus-4-8", true},
		{"anthropic/*", "openai/gpt-5", false},
		{"gpt-5-codex", "gpt-5-codex", true},
		{"gpt-5-codex", "gpt-5", false},
		{"ANTHROPIC/*", "anthropic/claude", true}, // case-insensitive
	}
	for _, c := range cases {
		if got := globMatch(c.glob, c.model); got != c.want {
			t.Errorf("globMatch(%q,%q)=%v want %v", c.glob, c.model, got, c.want)
		}
	}
}

func TestDecideModel(t *testing.T) {
	// Default-open: no rules => allowed.
	if ok, _ := DecideModel(nil, "anthropic/claude-opus-4-8"); !ok {
		t.Fatal("no rules should allow")
	}

	denyFamily := []RuleLite{{Glob: "anthropic/*", Effect: "deny"}}
	if ok, _ := DecideModel(denyFamily, "anthropic/claude-opus-4-8"); ok {
		t.Fatal("anthropic/* deny should block")
	}
	if ok, _ := DecideModel(denyFamily, "openai/gpt-5"); !ok {
		t.Fatal("unrelated model should stay allowed")
	}

	// Most-specific-wins: narrow allow beats broad deny.
	mixed := []RuleLite{
		{Glob: "anthropic/*", Effect: "deny"},
		{Glob: "anthropic/claude-haiku-4-5", Effect: "allow"},
	}
	if ok, _ := DecideModel(mixed, "anthropic/claude-haiku-4-5"); !ok {
		t.Fatal("specific allow should override broad deny")
	}
	if ok, _ := DecideModel(mixed, "anthropic/claude-opus-4-8"); ok {
		t.Fatal("opus should remain denied")
	}

	// Deny overrides allow at equal specificity.
	tie := []RuleLite{
		{Glob: "gpt-5", Effect: "allow"},
		{Glob: "gpt-5", Effect: "deny"},
	}
	if ok, _ := DecideModel(tie, "gpt-5"); ok {
		t.Fatal("deny should win at equal specificity")
	}
}
