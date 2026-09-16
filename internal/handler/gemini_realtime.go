package handler

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/openpaths/openpaths/internal/billing"
)

// geminiRealtimePath is the native Gemini Live API BidiGenerateContent
// websocket endpoint (see https://ai.google.dev/api/live).
const geminiRealtimePath = "/ws/google.ai.generativelanguage.v1beta.GenerativeService.BidiGenerateContent"

type geminiSetupEnvelope struct {
	Setup *struct {
		Model string `json:"model"`
	} `json:"setup"`
}

type geminiUsageEnvelope struct {
	UsageMetadata *struct {
		PromptTokenCount     int                        `json:"promptTokenCount"`
		CachedContentTokenCt int                        `json:"cachedContentTokenCount"`
		ResponseTokenCount   int                        `json:"responseTokenCount"`
		ThoughtsTokenCount   int                        `json:"thoughtsTokenCount"`
		PromptTokensDetails  []geminiModalityTokenCount `json:"promptTokensDetails"`
		CacheTokensDetails   []geminiModalityTokenCount `json:"cacheTokensDetails"`
		ResponseTokensDets   []geminiModalityTokenCount `json:"responseTokensDetails"`
	} `json:"usageMetadata"`
}

type geminiModalityTokenCount struct {
	Modality   string `json:"modality"`
	TokenCount int    `json:"tokenCount"`
}

// makeGeminiRealtimeURL builds the BidiGenerateContent websocket URL from the
// provider base URL (https://generativelanguage.googleapis.com).
func makeGeminiRealtimeURL(baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Host == "" {
		return "", errors.New("invalid provider URL")
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	case "wss", "ws":
	default:
		return "", errors.New("unsupported provider URL scheme")
	}
	u.Path = strings.TrimRight(u.Path, "/") + geminiRealtimePath
	u.RawQuery = ""
	return u.String(), nil
}

// parseGeminiUsage extracts billing.RealtimeUsage from a Live API server
// message carrying usageMetadata. Documented semantics
// (https://ai.google.dev/api/live#UsageMetadata): usageMetadata on a
// BidiGenerateContentServerMessage is per-turn, not cumulative for the
// session — promptTokenCount covers the effective prompt of that turn
// (cached content included), responseTokenCount/thoughtsTokenCount cover that
// turn's generated candidates and thoughts. The relay therefore deducts every
// message carrying usageMetadata and sums the per-turn values into the
// session total. Modality breakdown comes from
// promptTokensDetails/cacheTokensDetails/responseTokensDetails
// (TEXT, AUDIO, IMAGE, VIDEO, DOCUMENT).
func parseGeminiUsage(payload []byte) (billing.RealtimeUsage, bool) {
	var envelope geminiUsageEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil || envelope.UsageMetadata == nil {
		return billing.RealtimeUsage{}, false
	}
	u := envelope.UsageMetadata
	usage := billing.RealtimeUsage{}
	for _, d := range u.PromptTokensDetails {
		switch d.Modality {
		case "TEXT":
			usage.TextInputTokens += d.TokenCount
		case "AUDIO":
			usage.AudioInputTokens += d.TokenCount
		case "IMAGE", "VIDEO", "DOCUMENT":
			usage.ImageInputTokens += d.TokenCount
		}
	}
	for _, d := range u.CacheTokensDetails {
		switch d.Modality {
		case "TEXT":
			usage.CachedTextInputTokens += d.TokenCount
		case "AUDIO":
			usage.CachedAudioInputTokens += d.TokenCount
		case "IMAGE", "VIDEO", "DOCUMENT":
			usage.CachedImageInputTokens += d.TokenCount
		}
	}
	for _, d := range u.ResponseTokensDets {
		switch d.Modality {
		case "TEXT":
			usage.TextOutputTokens += d.TokenCount
		case "AUDIO":
			usage.AudioOutputTokens += d.TokenCount
		}
	}
	// Fallback when the modality breakdown is absent: attribute scalar totals.
	// Prompt tokens that are not audio/image are text; thoughts are billed at
	// the text output rate.
	usage.TextInputTokens = max(usage.TextInputTokens, u.PromptTokenCount-usage.AudioInputTokens-usage.ImageInputTokens)
	usage.TextOutputTokens = max(usage.TextOutputTokens, u.ResponseTokenCount-usage.AudioOutputTokens)
	usage.TextOutputTokens += u.ThoughtsTokenCount
	if len(u.CacheTokensDetails) == 0 && u.CachedContentTokenCt > 0 {
		usage.CachedTextInputTokens = u.CachedContentTokenCt
	}
	return usage, true
}
