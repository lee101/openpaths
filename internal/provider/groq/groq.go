package groq

import (
	"strings"

	"github.com/openpath/openpath/internal/provider/openai"
)

// Groq uses an OpenAI-compatible API, so we wrap the OpenAI adapter.
type GroqProvider struct {
	*openai.OpenAIProvider
}

func New(apiKey, baseURL string) *GroqProvider {
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai"
	}
	return &GroqProvider{
		OpenAIProvider: openai.New(apiKey, strings.TrimRight(baseURL, "/")),
	}
}

func (p *GroqProvider) Name() string { return "groq" }
