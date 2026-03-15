package fireworks

import (
	"strings"

	"github.com/openpaths/openpaths/internal/provider/openai"
)

type FireworksProvider struct {
	*openai.OpenAIProvider
}

func New(apiKey, baseURL string) *FireworksProvider {
	if baseURL == "" {
		baseURL = "https://api.fireworks.ai/inference"
	}
	return &FireworksProvider{
		OpenAIProvider: openai.New(apiKey, strings.TrimRight(baseURL, "/")),
	}
}

func (p *FireworksProvider) Name() string { return "fireworks" }
