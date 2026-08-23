package model

import "testing"

func TestPromoteSystemToUser(t *testing.T) {
	roles := func(msgs []ChatMessage) string {
		out := ""
		for i, m := range msgs {
			if i > 0 {
				out += ","
			}
			out += m.Role
		}
		return out
	}
	tests := []struct {
		name string
		in   []string
		want string
	}{
		// GLM rejects a system-only list, so the instruction becomes the user turn.
		{"system only", []string{"system"}, "user"},
		{"two systems", []string{"system", "system"}, "system,user"},
		{"developer only", []string{"developer"}, "user"},
		{"system then assistant", []string{"system", "assistant"}, "user,assistant"},
		// Anything already carrying a user turn is forwarded untouched.
		{"system then user", []string{"system", "user"}, "system,user"},
		{"consecutive users", []string{"user", "user"}, "user,user"},
		{"assistant first", []string{"assistant", "user"}, "assistant,user"},
		{"system mid", []string{"user", "system", "user"}, "user,system,user"},
		// No system message to promote, and nothing to map for an empty list.
		{"assistant only", []string{"assistant"}, "assistant"},
		{"empty", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ChatCompletionRequest{}
			for _, role := range tt.in {
				req.Messages = append(req.Messages, ChatMessage{Role: role, Content: "x"})
			}
			PromoteSystemToUser(req)
			if got := roles(req.Messages); got != tt.want {
				t.Errorf("roles = %q, want %q", got, tt.want)
			}
		})
	}
}

// A fallback candidate must not inherit a reshape done for an earlier provider.
func TestPromoteSystemToUserDoesNotMutateCallerSlice(t *testing.T) {
	original := []ChatMessage{{Role: "system", Content: "say only hi"}}
	req := &ChatCompletionRequest{Messages: original}
	PromoteSystemToUser(req)
	if original[0].Role != "system" {
		t.Errorf("caller slice mutated: %q", original[0].Role)
	}
	if req.Messages[0].Role != "user" {
		t.Errorf("request role = %q, want user", req.Messages[0].Role)
	}
	if req.Messages[0].Content != "say only hi" {
		t.Errorf("content mutated: %v", req.Messages[0].Content)
	}
}

func TestNeedsUserTurn(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"glm-5.3", true},
		{"glm-5.2", true},
		{"z-ai/glm-5.3", true},
		{"accounts/fireworks/models/glm-5.2", true},
		{"zai-org/GLM-5", true},
		{"openai/gpt-5.6-sol", false},
		{"claude-opus-5", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := NeedsUserTurn(tt.model); got != tt.want {
			t.Errorf("NeedsUserTurn(%q) = %v, want %v", tt.model, got, tt.want)
		}
	}
}
