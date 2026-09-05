package meta

import (
	"strings"

	"github.com/openpaths/openpaths/internal/model"
	oai "github.com/openpaths/openpaths/internal/provider/openai"
)

const defaultBaseURL = "https://api.meta.ai/v1"

// Provider exposes Meta Model API through OpenPaths' chat, image, and
// transcription provider interfaces. Meta's documented base URL includes
// /v1, while the shared OpenAI-compatible client appends /v1 itself, so New
// keeps both a root URL and an API URL to avoid producing /v1/v1 paths.
type Provider struct {
	*oai.OpenAIProvider
	apiKey string
	apiURL string
}

func New(apiKey, baseURL string) *Provider {
	rootURL, apiURL := normalizeBaseURLs(baseURL)
	return &Provider{
		OpenAIProvider: oai.NewCompatible("meta", apiKey, rootURL, sanitizeChat),
		apiKey:         apiKey,
		apiURL:         apiURL,
	}
}

func normalizeBaseURLs(baseURL string) (rootURL, apiURL string) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if strings.HasSuffix(baseURL, "/v1") {
		return strings.TrimSuffix(baseURL, "/v1"), baseURL
	}
	return baseURL, baseURL + "/v1"
}

func sanitizeChat(req *model.ChatCompletionRequest) {
	req.Prefill = ""
	req.TaskTier = ""
	req.RoutingStrategy = ""
	req.Thinking = nil
	req.ChatTemplateKwargs = nil
}
