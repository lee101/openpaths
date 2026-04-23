package nvidia

import (
	"strings"

	"github.com/openpaths/openpaths/internal/provider/openai"
)

type NvidiaProvider struct {
	*openai.OpenAIProvider
}

func New(apiKey, baseURL string) *NvidiaProvider {
	if baseURL == "" {
		baseURL = "https://integrate.api.nvidia.com"
	}
	return &NvidiaProvider{
		OpenAIProvider: openai.New(apiKey, strings.TrimRight(baseURL, "/")),
	}
}

func (p *NvidiaProvider) Name() string { return "nvidia" }
