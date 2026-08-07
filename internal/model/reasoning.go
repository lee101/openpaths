package model

import "strings"

var reasoningEffortOrder = []string{"none", "minimal", "low", "medium", "high", "xhigh", "max"}

// CompatibleReasoningEffort preserves requested intent while smoothing over
// model-specific API vocabularies. Equal-distance choices prefer more thought.
func CompatibleReasoningEffort(modelID, requested string) string {
	requested = strings.ToLower(strings.TrimSpace(requested))
	modelID = strings.ToLower(strings.TrimSpace(modelID))
	if requested == "" || !(modelID == "gpt-5.6" || strings.HasPrefix(modelID, "gpt-5.6-")) {
		return requested
	}
	supported := map[string]bool{"none": true, "low": true, "medium": true, "high": true, "xhigh": true, "max": true}
	if supported[requested] {
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
