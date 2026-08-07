package handler

import "testing"

func TestMakeUserProviderThinkingMachines(t *testing.T) {
	p := makeUserProvider("thinkingmachines", "tinker-test-key")
	if p == nil {
		t.Fatal("thinkingmachines BYOK provider is nil")
	}
	if got := p.Name(); got != "thinkingmachines" {
		t.Fatalf("provider name = %q, want thinkingmachines", got)
	}
}
