package model

import "strings"

// PromoteSystemToUser promotes the last system message to a user message when a
// conversation has no user turn. OpenAI serves a system-only message list, but
// GLM models reject it with "The messages parameter is illegal" (code 1214), so
// without this a request that is valid on the surface we present would fail.
//
// GLM tolerates every other ordering we forward -- consecutive same-role turns,
// an assistant-first list, and system messages in the middle all answer
// normally -- so promoting the missing user turn is the only reshaping needed.
// An empty list is left alone: there is no intent to map, and the upstream
// "Input cannot be empty" error is the correct answer.
func PromoteSystemToUser(req *ChatCompletionRequest) {
	if len(req.Messages) == 0 {
		return
	}
	lastSystem := -1
	for i, msg := range req.Messages {
		switch msg.Role {
		case "user":
			return
		case "system", "developer":
			lastSystem = i
		}
	}
	if lastSystem < 0 {
		return
	}
	// Copy on write so a fallback candidate sharing this backing array is not
	// mutated by a provider that never got to send the request.
	promoted := make([]ChatMessage, len(req.Messages))
	copy(promoted, req.Messages)
	promoted[lastSystem].Role = "user"
	req.Messages = promoted
}

// NeedsUserTurn reports whether a model's upstream rejects a message list that
// has no user turn, so a passthrough provider knows to reshape it first.
func NeedsUserTurn(modelID string) bool {
	modelID = strings.ToLower(strings.TrimSpace(modelID))
	if idx := strings.LastIndex(modelID, "/"); idx >= 0 {
		modelID = modelID[idx+1:]
	}
	return strings.HasPrefix(modelID, "glm-") || modelID == "glm"
}
