package model

import "strings"

var reasoningEffortOrder = []string{"none", "minimal", "low", "medium", "high", "xhigh", "max"}

// reasoningVocabulary records the reasoning_effort values one model family's
// upstream API actually accepts. Anything outside the set is remapped to the
// nearest supported tier instead of being forwarded, which would 400. A
// vocabulary with strip set drops the parameter entirely, for models whose
// upstream rejects reasoning_effort at any value.
type reasoningVocabulary struct {
	matches   func(modelID string) bool
	supported map[string]bool
	strip     bool
}

var reasoningVocabularies = []reasoningVocabulary{
	// GPT-5.6 has neither a "minimal" nor a "max" tier.
	{matches: family("gpt-5.6"), supported: effortSet("none", "low", "medium", "high", "xhigh")},
	// Grok through OpenRouter makes reasoning mandatory, unlike xAI direct.
	{matches: family("x-ai/grok"), supported: effortSet("minimal", "low", "medium", "high", "xhigh", "max")},
	// GLM-5.3 always thinks: it rejects "none" and "minimal" outright and only
	// accepts three tiers. Matched on the bare family name so the Z.AI,
	// OpenRouter and Fireworks routes to the same model normalize alike.
	{matches: bareFamily("glm-5.3"), supported: effortSet("low", "high", "max")},
	// Gemini 3.7 through OpenRouter makes reasoning mandatory ("Reasoning is
	// mandatory for this endpoint and cannot be disabled"), unlike the same
	// model served directly. Keep this entry ahead of the direct one; the
	// namespaced id never matches the bare family anyway.
	{matches: family("google/gemini-3.7"), supported: effortSet("low", "medium", "high")},
	// Gemini 3.7 direct accepts none/low/medium/high; MINIMAL is rejected for
	// this model and xhigh/max are not valid values at all.
	{matches: family("gemini-3.7"), supported: effortSet("none", "low", "medium", "high")},
	// Qwen 3.8 2.4T reports "Supported types are xhigh (default), medium, and
	// low"; "none" is additionally accepted to turn thinking off. Listed before
	// the qwen3.8-max entry so the more specific family wins.
	{matches: anyFamily(bareFamily("qwen3.8-2.4t-a95b"), bareFamily("qwen3p8-2p4t-a95b")),
		supported: effortSet("none", "low", "medium", "xhigh")},
	// Qwen 3.8 Max through OpenRouter makes reasoning mandatory.
	{matches: family("qwen/qwen3.8-max"), supported: effortSet("minimal", "low", "medium", "high", "xhigh", "max")},
	// Magistral is Mistral's only reasoning family, and it exposes just two
	// tiers: "supported values: [high, none]".
	{matches: family("magistral"), supported: effortSet("none", "high")},
	// Every other Mistral model answers "reasoning_effort is not enabled for
	// this model" for any value, so the parameter has to come off entirely.
	{matches: mistralNonReasoning, strip: true},
}

// mistralNonReasoning matches the Mistral-served families that reject
// reasoning_effort outright. Magistral is deliberately absent: it is matched by
// the entry above, which wins by being earlier in the table.
func mistralNonReasoning(modelID string) bool {
	for _, name := range []string{
		"mistral-large", "mistral-medium", "mistral-small", "mistral-saba",
		"codestral", "pixtral", "ministral", "devstral",
		"open-mistral", "open-mixtral",
	} {
		if family(name)(modelID) {
			return true
		}
	}
	return false
}

// family matches a model ID that is exactly name or a name-suffixed variant.
func family(name string) func(string) bool {
	return func(modelID string) bool {
		return modelID == name || strings.HasPrefix(modelID, name+"-")
	}
}

// bareFamily matches like family but first strips any provider namespace, so
// "z-ai/glm-5.3" and "accounts/fireworks/models/glm-5.3" match "glm-5.3".
func bareFamily(name string) func(string) bool {
	match := family(name)
	return func(modelID string) bool {
		if idx := strings.LastIndex(modelID, "/"); idx >= 0 {
			modelID = modelID[idx+1:]
		}
		return match(modelID)
	}
}

// anyFamily matches when any of the given matchers does, for a model the
// providers spell differently (Together's "qwen3.8-2.4t-a95b" vs Fireworks'
// "qwen3p8-2p4t-a95b").
func anyFamily(matchers ...func(string) bool) func(string) bool {
	return func(modelID string) bool {
		for _, match := range matchers {
			if match(modelID) {
				return true
			}
		}
		return false
	}
}

func effortSet(efforts ...string) map[string]bool {
	set := make(map[string]bool, len(efforts))
	for _, effort := range efforts {
		set[effort] = true
	}
	return set
}

// CompatibleReasoningEffort preserves requested intent while smoothing over
// model-specific API vocabularies. Equal-distance choices prefer more thought.
func CompatibleReasoningEffort(modelID, requested string) string {
	requested = strings.ToLower(strings.TrimSpace(requested))
	modelID = strings.ToLower(strings.TrimSpace(modelID))
	if requested == "" {
		return requested
	}
	var supported map[string]bool
	for _, vocab := range reasoningVocabularies {
		if vocab.matches(modelID) {
			if vocab.strip {
				return ""
			}
			supported = vocab.supported
			break
		}
	}
	if supported == nil || supported[requested] {
		return requested
	}
	target := effortIndex(requested)
	if target < 0 {
		return requested
	}
	best, distance := requested, len(reasoningEffortOrder)+1
	for i, effort := range reasoningEffortOrder {
		if !supported[effort] {
			continue
		}
		d := i - target
		if d < 0 {
			d = -d
		}
		if d < distance || (d == distance && i > effortIndex(best)) {
			best, distance = effort, d
		}
	}
	return best
}

func effortIndex(effort string) int {
	for i, candidate := range reasoningEffortOrder {
		if candidate == effort {
			return i
		}
	}
	return -1
}
