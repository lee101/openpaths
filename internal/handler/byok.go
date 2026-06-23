package handler

import (
	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/db/queries"
	"github.com/openpaths/openpaths/internal/middleware"
	"github.com/openpaths/openpaths/internal/provider"
	"github.com/openpaths/openpaths/internal/provider/anthropic"
	"github.com/openpaths/openpaths/internal/provider/deepseek"
	"github.com/openpaths/openpaths/internal/provider/fal"
	"github.com/openpaths/openpaths/internal/provider/google"
	"github.com/openpaths/openpaths/internal/provider/groq"
	"github.com/openpaths/openpaths/internal/provider/minimax"
	"github.com/openpaths/openpaths/internal/provider/mistral"
	"github.com/openpaths/openpaths/internal/provider/netwrck"
	"github.com/openpaths/openpaths/internal/provider/openai"
	"github.com/openpaths/openpaths/internal/provider/openrouter"
	"github.com/openpaths/openpaths/internal/provider/sakana"
	"github.com/openpaths/openpaths/internal/provider/together"
	"github.com/openpaths/openpaths/internal/provider/xai"
	"github.com/openpaths/openpaths/internal/provider/zai"
)

var providerBaseURLs = map[string]string{
	"openai":     "https://api.openai.com",
	"anthropic":  "https://api.anthropic.com",
	"google":     "https://generativelanguage.googleapis.com",
	"mistral":    "https://api.mistral.ai",
	"groq":       "https://api.groq.com/openai",
	"xai":        "https://api.x.ai",
	"deepseek":   "https://api.deepseek.com",
	"openrouter": "https://openrouter.ai/api",
	"together":   "https://api.together.xyz",
	"minimax":    "https://api.minimax.io",
	"netwrck":    "https://netwrck.com",
	"zai":        "https://api.z.ai",
	"sakana":     "https://api.sakana.ai",
	"fal":        "https://fal.run",
}

func getUserProviderKeys(ctx *fasthttp.RequestCtx) map[string]*queries.UserProviderKey {
	v, _ := ctx.UserValue(middleware.CtxKeyUserProviderKeys).(map[string]*queries.UserProviderKey)
	return v
}

func makeUserProvider(providerName, apiKey string) provider.Provider {
	baseURL := providerBaseURLs[providerName]
	switch providerName {
	case "openai":
		return openai.New(apiKey, baseURL)
	case "anthropic":
		return anthropic.New(apiKey, baseURL)
	case "google":
		return google.New(apiKey, baseURL)
	case "mistral":
		return mistral.New(apiKey, baseURL)
	case "groq":
		return groq.New(apiKey, baseURL)
	case "xai":
		return xai.New(apiKey, baseURL)
	case "deepseek":
		return deepseek.New(apiKey, baseURL)
	case "openrouter":
		return openrouter.New(apiKey, baseURL)
	case "together":
		return together.New(apiKey, baseURL)
	case "minimax":
		return minimax.New(apiKey)
	case "netwrck":
		return netwrck.New(apiKey, baseURL)
	case "zai":
		// BYOK GLM keys are GLM Coding Plan keys, which live on the coding
		// endpoint (/api/coding/paas/v4). NewCoding tries that first and falls
		// back to the standard endpoint for plain pay-as-you-go API keys.
		return zai.NewCoding(apiKey, baseURL)
	case "sakana":
		return sakana.New(apiKey, baseURL)
	case "fal":
		return fal.New(apiKey)
	}
	return nil
}

func getBYOKProvider(ctx *fasthttp.RequestCtx, providerName string) (provider.Provider, bool) {
	keys := getUserProviderKeys(ctx)
	if keys == nil {
		return nil, false
	}
	uk, ok := keys[providerName]
	if !ok || uk.APIKey == "" {
		return nil, false
	}
	p := makeUserProvider(providerName, uk.APIKey)
	if p == nil {
		return nil, false
	}
	return p, true
}
