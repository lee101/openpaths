package xai

import (
	"strings"

	"github.com/openpath/openpath/internal/provider/openai"
)

type XAIProvider struct {
	*openai.OpenAIProvider
}

func New(apiKey, baseURL string) *XAIProvider {
	if baseURL == "" {
		baseURL = "https://api.x.ai"
	}
	return &XAIProvider{
		OpenAIProvider: openai.New(apiKey, strings.TrimRight(baseURL, "/")),
	}
}

func (p *XAIProvider) Name() string { return "xai" }
