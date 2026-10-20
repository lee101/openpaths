package nous

import (
	"strings"

	"github.com/openpaths/openpaths/internal/model"
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
		OpenAIProvider: openai.NewCompatible("nous", apiKey, strings.TrimSuffix(strings.TrimRight(baseURL, "/"), "/v1"), func(req *model.ChatCompletionRequest) {
			req.Prefill = ""
			req.TaskTier = ""
			req.RoutingStrategy = ""
			req.Thinking = nil
			req.ChatTemplateKwargs = nil
		}),
	}
}

func (p *NousProvider) Name() string { return "nous" }
