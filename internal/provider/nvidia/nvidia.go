package nvidia

import (
	"strings"

	"github.com/openpaths/openpaths/internal/model"
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
		OpenAIProvider: openai.NewCompatible("nvidia", apiKey, strings.TrimRight(baseURL, "/"), sanitizeForNvidia),
	}
}

func (p *NvidiaProvider) Name() string { return "nvidia" }

func sanitizeForNvidia(req *model.ChatCompletionRequest) {
	req.Prefill = ""
	req.TaskTier = ""
	req.RoutingStrategy = ""

	if !strings.HasPrefix(req.Model, "deepseek-ai/deepseek-") {
		req.Thinking = nil
		req.ChatTemplateKwargs = nil
		return
	}

	if req.ChatTemplateKwargs == nil {
		req.ChatTemplateKwargs = make(map[string]any, 2)
	}

	if _, ok := req.ChatTemplateKwargs["thinking"]; !ok {
		switch {
		case req.Thinking != nil && req.Thinking.Type == "disabled":
			req.ChatTemplateKwargs["thinking"] = false
		case req.Thinking != nil && req.Thinking.Type == "enabled":
			req.ChatTemplateKwargs["thinking"] = true
		case req.ReasoningEffort == "none":
			req.ChatTemplateKwargs["thinking"] = false
		case req.ReasoningEffort == "minimal" || req.ReasoningEffort == "low" || req.ReasoningEffort == "medium" || req.ReasoningEffort == "high" || req.ReasoningEffort == "xhigh" || req.ReasoningEffort == "max":
			req.ChatTemplateKwargs["thinking"] = true
		case strings.Contains(req.Model, "deepseek-v4-pro"):
			req.ChatTemplateKwargs["thinking"] = true
		}
	}

	req.Thinking = nil
	req.ReasoningEffort = ""
	if len(req.ChatTemplateKwargs) == 0 {
		req.ChatTemplateKwargs = nil
	}
}
