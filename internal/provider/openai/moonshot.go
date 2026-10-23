package openai

import (
	"encoding/json"
	"strings"

	"github.com/openpaths/openpaths/internal/model"
)

const (
	MoonshotProvider = "moonshot"

	moonshotBigPrefixTokens = 16000
)

// SanitizeMoonshot adapts a request to the Kimi K3 family: sampling params are
// fixed server-side, reasoning_effort only takes low/high/max, and the cache
// write TTL is chosen per request.
func SanitizeMoonshot(req *model.ChatCompletionRequest) {
	req.Prefill = ""
	req.TaskTier = ""
	req.RoutingStrategy = ""
	req.Thinking = nil
	req.ChatTemplateKwargs = nil
	req.Temperature = nil
	req.TopP = nil
	req.PresencePenalty = nil
	req.FrequencyPenalty = nil
	req.ReasoningEffort = moonshotEffort(req.ReasoningEffort)
	req.PromptCacheOptions = moonshotCacheOptions(req)
}

func moonshotEffort(e string) string {
	switch strings.ToLower(strings.TrimSpace(e)) {
	case "", "auto":
		return ""
	case "none", "minimal", "low":
		return "low"
	case "medium", "high":
		return "high"
	default:
		return "max"
	}
}

// moonshotCacheOptions picks the cache write TTL. Writes are billed once and
// 1h costs 2x a 5m write, so 1h only pays off when a large stable prefix is
// reused across gaps longer than 5m (human-paced chat). Agent/tool loops reuse
// the prefix within seconds and each hit renews a 5m entry for free.
func moonshotCacheOptions(req *model.ChatCompletionRequest) *model.PromptCacheOptions {
	if o := req.PromptCacheOptions; o != nil && (o.TTL == "5m" || o.TTL == "1h") {
		return &model.PromptCacheOptions{Mode: "implicit", TTL: o.TTL}
	}
	if !humanPaced(req) || estimatePromptTokens(req) < moonshotBigPrefixTokens {
		return nil
	}
	return &model.PromptCacheOptions{Mode: "implicit", TTL: "1h"}
}

// humanPaced is true for a multi-turn chat whose last message is a user turn
// with no tool traffic anywhere in the history.
func humanPaced(req *model.ChatCompletionRequest) bool {
	n := len(req.Messages)
	if n < 3 || req.Messages[n-1].Role != "user" {
		return false
	}
	assistant := false
	for _, m := range req.Messages {
		if m.Role == "tool" || len(m.ToolCalls) > 0 {
			return false
		}
		if m.Role == "assistant" {
			assistant = true
		}
	}
	return assistant
}

func estimatePromptTokens(req *model.ChatCompletionRequest) int {
	chars := 0
	for _, m := range req.Messages {
		switch c := m.Content.(type) {
		case string:
			chars += len(c)
		case nil:
		default:
			if b, err := json.Marshal(c); err == nil {
				chars += len(b)
			}
		}
	}
	if len(req.Tools) > 0 {
		if b, err := json.Marshal(req.Tools); err == nil {
			chars += len(b)
		}
	}
	return chars / 4
}
