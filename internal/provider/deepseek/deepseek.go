package deepseek

import (
	"strings"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider/openai"
)

type DeepSeekProvider struct {
	*openai.OpenAIProvider
}

func New(apiKey, baseURL string) *DeepSeekProvider {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	return &DeepSeekProvider{
		OpenAIProvider: openai.NewCompatible("deepseek", apiKey, strings.TrimRight(baseURL, "/"), sanitizeForDeepSeek),
	}
}

func (p *DeepSeekProvider) Name() string { return "deepseek" }

func sanitizeForDeepSeek(req *model.ChatCompletionRequest) {
	req.Prefill = ""
	req.TaskTier = ""

	// DeepSeek controls reasoning through the `thinking` field, not the
	// OpenAI-style `reasoning_effort` parameter. Its API rejects unknown
	// reasoning_effort values (notably "none", which the autorouter sets for
	// fast/cheap tiers) with a deserialization error, so translate the effort
	// hint into `thinking` for v4 models and always drop reasoning_effort
	// before the request reaches the upstream.
	if strings.HasPrefix(req.Model, "deepseek-v4-") && req.Thinking == nil {
		switch req.ReasoningEffort {
		case "none":
			req.Thinking = &model.ThinkingConfig{Type: "disabled"}
		case "low", "medium", "high":
			req.Thinking = &model.ThinkingConfig{Type: "enabled"}
		}
	}

	req.ReasoningEffort = ""
}
