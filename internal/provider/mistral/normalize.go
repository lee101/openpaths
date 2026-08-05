package mistral

import (
	"context"
	"strings"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

// Mistral's reasoning models (magistral-*) answer with a structured content
// array instead of the string every OpenAI-compatible client expects:
//
//	"content": [
//	  {"type": "thinking", "thinking": [{"type": "text", "text": "..."}]},
//	  {"type": "text", "text": "Hi there!"}
//	]
//
// Passed through verbatim this breaks any caller that unmarshals content into a
// string. Flatten the visible text into content and move the hidden chain of
// thought to reasoning_content, where the rest of the gateway already puts it.
func normalizeMessage(msg *model.ChatMessage) {
	if msg == nil {
		return
	}
	parts, ok := msg.Content.([]any)
	if !ok {
		return
	}
	var text, thinking strings.Builder
	for _, part := range parts {
		obj, ok := part.(map[string]any)
		if !ok {
			// A bare string element is still visible output.
			if s, ok := part.(string); ok {
				text.WriteString(s)
			}
			continue
		}
		switch obj["type"] {
		case "text":
			if s, ok := obj["text"].(string); ok {
				text.WriteString(s)
			}
		case "thinking":
			thinking.WriteString(flattenText(obj["thinking"]))
		}
	}
	msg.Content = text.String()
	if msg.ReasoningContent == "" {
		msg.ReasoningContent = thinking.String()
	}
}

// flattenText joins the text of a nested part list (Mistral nests the chain of
// thought one level deeper than the top-level content array).
func flattenText(v any) string {
	parts, ok := v.([]any)
	if !ok {
		if s, ok := v.(string); ok {
			return s
		}
		return ""
	}
	var b strings.Builder
	for _, part := range parts {
		switch p := part.(type) {
		case string:
			b.WriteString(p)
		case map[string]any:
			if s, ok := p["text"].(string); ok {
				b.WriteString(s)
			}
		}
	}
	return b.String()
}

func (p *MistralProvider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	resp, err := p.OpenAIProvider.ChatCompletion(ctx, req)
	if err != nil || resp == nil {
		return resp, err
	}
	for i := range resp.Choices {
		normalizeMessage(resp.Choices[i].Message)
	}
	return resp, nil
}

func (p *MistralProvider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (<-chan provider.StreamEvent, error) {
	upstream, err := p.OpenAIProvider.ChatCompletionStream(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make(chan provider.StreamEvent)
	go func() {
		defer close(out)
		for ev := range upstream {
			if ev.Chunk != nil {
				for i := range ev.Chunk.Choices {
					normalizeMessage(ev.Chunk.Choices[i].Delta)
				}
			}
			select {
			case out <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}
