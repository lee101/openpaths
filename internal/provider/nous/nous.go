package nous

import (
	"strings"

	"github.com/openpaths/openpaths/internal/provider/openai"
)

type NousProvider struct {
	*openai.OpenAIProvider
}

func New(apiKey, baseURL string) *NousProvider {
	if baseURL == "" {
		baseURL = "https://inference-api.nousresearch.com"
	}
	return &NousProvider{
		OpenAIProvider: openai.New(apiKey, strings.TrimRight(baseURL, "/")),
	}
}

func (p *NousProvider) Name() string { return "nous" }
