package deepseek

import (
	"strings"

	"github.com/openpath/openpath/internal/provider/openai"
)

type DeepSeekProvider struct {
	*openai.OpenAIProvider
}

func New(apiKey, baseURL string) *DeepSeekProvider {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	return &DeepSeekProvider{
		OpenAIProvider: openai.New(apiKey, strings.TrimRight(baseURL, "/")),
	}
}

func (p *DeepSeekProvider) Name() string { return "deepseek" }
