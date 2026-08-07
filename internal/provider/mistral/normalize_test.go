package mistral

import (
	"testing"

	"github.com/openpaths/openpaths/internal/model"
)

// magistral-medium-latest answers with a structured content array; clients that
// decode content into a string used to fail on it with "invalid JSON response".
func TestNormalizeMessageFlattensMagistralContent(t *testing.T) {
	msg := &model.ChatMessage{
		Role: "assistant",
		Content: []any{
			map[string]any{
				"type":     "thinking",
				"closed":   true,
				"thinking": []any{map[string]any{"type": "text", "text": "The user wants a greeting."}},
			},
			map[string]any{"type": "text", "text": "Hi there!"},
		},
	}

	normalizeMessage(msg)

	if got, want := msg.Content, "Hi there!"; got != want {
		t.Errorf("Content = %#v, want %q", got, want)
	}
	if got, want := msg.ReasoningContent, "The user wants a greeting."; got != want {
		t.Errorf("ReasoningContent = %q, want %q", got, want)
	}
}

func TestNormalizeMessageLeavesStringContentAlone(t *testing.T) {
	msg := &model.ChatMessage{Role: "assistant", Content: "already a string"}
	normalizeMessage(msg)
	if got, want := msg.Content, "already a string"; got != want {
		t.Errorf("Content = %#v, want %q", got, want)
	}
}

func TestNormalizeMessageKeepsExistingReasoning(t *testing.T) {
	msg := &model.ChatMessage{
		Role:             "assistant",
		ReasoningContent: "from upstream",
		Content: []any{
			map[string]any{"type": "thinking", "thinking": []any{map[string]any{"type": "text", "text": "ignored"}}},
			map[string]any{"type": "text", "text": "answer"},
		},
	}
	normalizeMessage(msg)
	if got, want := msg.ReasoningContent, "from upstream"; got != want {
		t.Errorf("ReasoningContent = %q, want %q", got, want)
	}
}

func TestNormalizeMessageNilSafe(t *testing.T) {
	normalizeMessage(nil) // must not panic on a chunk with no delta
}
