package mistral

import (
	"strings"

	"github.com/openpaths/openpaths/internal/provider/openai"
)

// Mistral uses an OpenAI-compatible API, so we wrap the OpenAI adapter.
type MistralProvider struct {
	*openai.OpenAIProvider
}

func New(apiKey, baseURL string) *MistralProvider {
	if baseURL == "" {
		baseURL = "https://api.mistral.ai"
	}
	return &MistralProvider{
		OpenAIProvider: openai.New(apiKey, strings.TrimRight(baseURL, "/")),
	}
}

func (p *MistralProvider) Name() string { return "mistral" }
