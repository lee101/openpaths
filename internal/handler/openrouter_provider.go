package handler

import (
	"fmt"
	"strings"

	"github.com/valyala/fasthttp"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/router"
)

// OpenRouterProviderHandler serves the OpenRouter-compatible models list
// so that OpenPaths can be registered as a provider on OpenRouter.
type OpenRouterProviderHandler struct {
	router *router.Router
}

func NewOpenRouterProviderHandler(r *router.Router) *OpenRouterProviderHandler {
	return &OpenRouterProviderHandler{router: r}
}

// ORModel is the OpenRouter model format.
type ORModel struct {
	ID                          string         `json:"id"`
	HuggingFaceID               string         `json:"hugging_face_id,omitempty"`
	Name                        string         `json:"name"`
	Created                     int64          `json:"created"`
	InputModalities             []string       `json:"input_modalities"`
	OutputModalities            []string       `json:"output_modalities"`
	ContextLength               int            `json:"context_length"`
	MaxOutputLength             int            `json:"max_output_length"`
	Pricing                     ORPricing      `json:"pricing"`
	SupportedSamplingParameters []string       `json:"supported_sampling_parameters"`
	SupportedFeatures           []string       `json:"supported_features"`
	Description                 string         `json:"description,omitempty"`
	Datacenters                 []ORDatacenter `json:"datacenters,omitempty"`
	OpenRouter                  *ORSlug        `json:"openrouter,omitempty"`
}

type ORPricing struct {
	Prompt         string `json:"prompt"`
	Completion     string `json:"completion"`
	Image          string `json:"image"`
	Request        string `json:"request"`
	InputCacheRead string `json:"input_cache_read"`
}

type ORDatacenter struct {
	CountryCode string `json:"country_code"`
}

type ORSlug struct {
	Slug string `json:"slug"`
}

func (h *OpenRouterProviderHandler) HandleListModels(ctx *fasthttp.RequestCtx) {
	models := h.router.ListModelConfigs()

	var orModels []ORModel
	for _, cfg := range models {
		orModel := configToORModel(cfg)
		if orModel != nil {
			orModels = append(orModels, *orModel)
		}
	}

	writeJSON(ctx, 200, map[string]any{
		"data": orModels,
	})
}

func configToORModel(cfg *model.ModelConfig) *ORModel {
	// Skip auto-routing models and internal aliases
	if strings.HasPrefix(cfg.ID, "auto") {
		return nil
	}
	// Skip OpenRouter-proxied models (we don't re-expose those)
	if strings.HasPrefix(cfg.ID, "or/") || strings.EqualFold(cfg.Provider, "openrouter") {
		return nil
	}

	inputMod := []string{"text"}
	outputMod := []string{"text"}

	// Determine modalities
	if cfg.SupportsVision {
		inputMod = append(inputMod, "image")
	}
	if cfg.PricePerImage > 0 {
		outputMod = []string{"image"}
		inputMod = []string{"text"}
	}
	if cfg.PricePerVideo > 0 || cfg.PricePerSecond > 0 {
		outputMod = []string{"video"}
		inputMod = []string{"text"}
	}

	// Convert pricing from per-1M-tokens to per-token
	promptPrice := cfg.InputPricePer1M / 1_000_000
	inputCacheReadPrice := cfg.InputCacheHitPricePer1M / 1_000_000
	completionPrice := cfg.OutputPricePer1M / 1_000_000
	imagePrice := cfg.PricePerImage

	pricing := ORPricing{
		Prompt:         formatPrice(promptPrice),
		Completion:     formatPrice(completionPrice),
		Image:          formatPrice(imagePrice),
		Request:        "0",
		InputCacheRead: formatPrice(inputCacheReadPrice),
	}

	// Sampling parameters
	samplingParams := []string{"temperature", "top_p", "stop", "max_tokens", "frequency_penalty", "presence_penalty", "seed"}

	// Features
	var features []string
	if cfg.SupportsTools {
		features = append(features, "tools")
	}
	if cfg.SupportsStreaming {
		features = append(features, "streaming")
	}

	// Build display name from ID
	name := buildDisplayName(cfg.ID)

	return &ORModel{
		ID:                          cfg.ID,
		Name:                        name,
		Created:                     1700000000, // static epoch
		InputModalities:             inputMod,
		OutputModalities:            outputMod,
		ContextLength:               cfg.ContextWindow,
		MaxOutputLength:             cfg.MaxOutputTokens,
		Pricing:                     pricing,
		SupportedSamplingParameters: samplingParams,
		SupportedFeatures:           features,
		Datacenters:                 []ORDatacenter{{CountryCode: "US"}},
		OpenRouter:                  &ORSlug{Slug: "openpaths/" + cfg.ID},
	}
}

func formatPrice(price float64) string {
	if price == 0 {
		return "0"
	}
	return fmt.Sprintf("%.10f", price)
}

var orModelDisplayNames = map[string]string{
	"gpt-5-chat-latest":            "OpenAI: GPT-5 Chat",
	"gpt-5.5":                      "OpenAI: GPT-5.5",
	"gpt-5.4":                      "OpenAI: GPT-5.4",
	"gpt-5-codex":                  "OpenAI: GPT-5 Codex",
	"gpt-5-mini":                   "OpenAI: GPT-5 Mini",
	"codex-mini-latest":            "OpenAI: Codex Mini",
	"o3":                           "OpenAI: o3",
	"o4-mini":                      "OpenAI: o4-mini",
	"gpt-4o":                       "OpenAI: GPT-4o",
	"gpt-4o-mini":                  "OpenAI: GPT-4o Mini",
	"claude-sonnet-latest":         "Anthropic: Claude Sonnet 5 (Latest)",
	"claude-sonnet-5":              "Anthropic: Claude Sonnet 5",
	"claude-sonnet-4-6":            "Anthropic: Claude Sonnet 4.6",
	"claude-opus-latest":           "Anthropic: Claude Opus 5 (Latest)",
	"claude-opus-5":                "Anthropic: Claude Opus 5",
	"claude-opus-4-8":              "Anthropic: Claude Opus 4.8",
	"claude-opus-4-7":              "Anthropic: Claude Opus 4.7",
	"claude-opus-4-6":              "Anthropic: Claude Opus 4.6",
	"claude-haiku-4-5-20251001":    "Anthropic: Claude Haiku 4.5",
	"gemini-3.5-flash":             "Google: Gemini 3.5 Flash",
	"gemini-2.5-pro":               "Google: Gemini 2.5 Pro",
	"gemini-2.5-flash":             "Google: Gemini 2.5 Flash",
	"gemini-3.1-flash-lite":        "Google: Gemini 3.1 Flash Lite",
	"grok-4.6":                     "xAI: Grok 4.6",
	"grok-4.5":                     "xAI: Grok 4.5",
	"grok-4.3":                     "xAI: Grok 4.3",
	"grok-4.20-0309-reasoning":     "xAI: Grok 4.20 Reasoning",
	"grok-4.20-0309-non-reasoning": "xAI: Grok 4.20 Non-Reasoning",
	"grok-4.20-multi-agent-0309":   "xAI: Grok 4.20 Multi-Agent",
	"grok-build-0.1":               "xAI: Grok Build 0.1",
	"grok-3-mini":                  "xAI: Grok 3 Mini",
	"deepseek-chat":                "DeepSeek: Chat",
	"deepseek-reasoner":            "DeepSeek: Reasoner",
	"mistral-large-latest":         "Mistral: Large",
	"mistral-medium-latest":        "Mistral: Medium",
	"mistral-small-latest":         "Mistral: Small",
	"codestral-latest":             "Mistral: Codestral",
	"minimax-m3":                   "MiniMax: M3",
	"minimax-m2.7":                 "MiniMax: M2.7",
	"minimax-m2.7-highspeed":       "MiniMax: M2.7 Highspeed",
	"minimax-m2.5":                 "MiniMax: M2.5",
	"minimax-m2.5-direct":          "MiniMax: M2.5 Direct",
	"minimax-m2.5-highspeed":       "MiniMax: M2.5 Highspeed",
	"hermes-4-70b":                 "Nous: Hermes 4 70B",
	"hermes-4-405b":                "Nous: Hermes 4 405B",
	"llama-3.3-70b-versatile":      "Meta: Llama 3.3 70B",
	"llama-3.1-8b-instant":         "Meta: Llama 3.1 8B",
	"ra1":                          "Netwrck: RA1",
	"flux-schnell":                 "Black Forest Labs: FLUX.1 Schnell",
	"flux-dev":                     "Black Forest Labs: FLUX.2 Dev",
	"flux-pro":                     "Black Forest Labs: FLUX 1.1 Pro",
	"hailuo-2.3":                   "MiniMax: Hailuo 2.3 Video",
	"glm-5":                        "ZAI: GLM-5",
	"qwen3.8-2.4t-a95b":            "Alibaba: Qwen 3.8 2.4T A95B",
	"qwen3.5-397b":                 "Alibaba: Qwen 3.5 397B",
	"kimi-k2.5":                    "Moonshot: Kimi K2.5",
	"kimi-k3":                      "Moonshot: Kimi K3",
}

func buildDisplayName(id string) string {
	if name, ok := orModelDisplayNames[id]; ok {
		return name
	}

	// Fallback: capitalize and clean up the ID
	name := strings.ReplaceAll(id, "-", " ")
	name = strings.ReplaceAll(name, "/", ": ")
	return "OpenPaths: " + name
}
