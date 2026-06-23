package handler

import (
	"strings"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/savedresp"
)

// saveTextGeneration enqueues a chat/messages generation for the user if they have
// opted into text response saving. Safe to call with a nil saver.
func saveTextGeneration(saver *savedresp.Saver, userID, apiKeyID, modelName, provider string,
	messages []model.ChatMessage, output string, tokensIn, tokensOut int, cost int64) {
	if saver == nil || userID == "" || !saver.WantText(userID) {
		return
	}
	prompt := lastUserText(messages)
	if prompt == "" && len(messages) > 0 {
		prompt = messageContentText(messages[len(messages)-1].Content)
	}
	saver.Save(&model.SavedResponse{
		UserID:    userID,
		APIKeyID:  apiKeyID,
		Kind:      savedresp.KindText,
		Model:     modelName,
		Provider:  provider,
		Prompt:    prompt,
		Input:     messagesTranscript(messages),
		Output:    output,
		TokensIn:  tokensIn,
		TokensOut: tokensOut,
		CostCents: cost,
	})
}

// saveImageGeneration enqueues each generated image for the user if they have opted
// into image response saving.
func saveImageGeneration(saver *savedresp.Saver, userID, apiKeyID, modelName, provider, prompt string,
	resp *model.ImageGenerationResponse, cost int64) {
	if saver == nil || userID == "" || resp == nil || !saver.WantImage(userID) {
		return
	}
	for i := range resp.Data {
		url := resp.Data[i].URL
		if url == "" {
			// b64-only results aren't persistently hosted; skip rather than store megabytes.
			continue
		}
		revised := resp.Data[i].RevisedPrompt
		saver.Save(&model.SavedResponse{
			UserID:    userID,
			APIKeyID:  apiKeyID,
			Kind:      savedresp.KindImage,
			Model:     modelName,
			Provider:  provider,
			Prompt:    prompt,
			Output:    revised,
			ImageURL:  url,
			ThumbURL:  url,
			Width:     resp.Data[i].Width,
			Height:    resp.Data[i].Height,
			CostCents: cost,
		})
	}
}

func lastUserText(messages []model.ChatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			if t := messageContentText(messages[i].Content); t != "" {
				return t
			}
		}
	}
	return ""
}

// messagesTranscript renders the conversation as a plain-text transcript.
func messagesTranscript(messages []model.ChatMessage) string {
	var b strings.Builder
	for _, m := range messages {
		text := messageContentText(m.Content)
		if text == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(m.Role)
		b.WriteString(": ")
		b.WriteString(text)
	}
	return b.String()
}

// messageContentText flattens a chat message content (string or multipart array)
// into its text. Non-text parts (images) are ignored.
func messageContentText(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var parts []string
		for _, p := range v {
			m, ok := p.(map[string]any)
			if !ok {
				continue
			}
			if t, ok := m["text"].(string); ok && t != "" {
				parts = append(parts, t)
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

// chatResponseText extracts the assistant output text from a completion response.
func chatResponseText(resp *model.ChatCompletionResponse) string {
	if resp == nil || len(resp.Choices) == 0 || resp.Choices[0].Message == nil {
		return ""
	}
	return messageContentText(resp.Choices[0].Message.Content)
}
