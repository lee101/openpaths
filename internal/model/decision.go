package model

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// DecisionRequest is the TypeSafe /v1/systemone schema (Jev, Alibaba decision-model).
type DecisionRequest struct {
	Model     string                     `json:"model"`
	State     json.RawMessage            `json:"state"`
	Questions map[string]json.RawMessage `json:"questions"`
}

type DecisionResponse struct {
	Model   string          `json:"model"`
	Answers json.RawMessage `json:"answers"`
	Usage   DecisionUsage   `json:"usage"`
}

type DecisionUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type decisionQuestion struct {
	Type         string          `json:"type"`
	Instructions json.RawMessage `json:"instructions"`
	Criteria     json.RawMessage `json:"criteria"`
}

// Validate mirrors TypeSafe's 422 rules and names the offending field.
func (r DecisionRequest) Validate() error {
	if strings.TrimSpace(r.Model) == "" {
		return fmt.Errorf("model is required")
	}
	if isEmptyJSON(r.State) {
		return fmt.Errorf("state is required")
	}
	if len(r.Questions) == 0 {
		return fmt.Errorf("questions must contain at least one question")
	}
	keys := make([]string, 0, len(r.Questions))
	for key := range r.Questions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		var q decisionQuestion
		if err := json.Unmarshal(r.Questions[key], &q); err != nil {
			return fmt.Errorf("questions.%s must be an object", key)
		}
		if isEmptyJSON(q.Instructions) {
			return fmt.Errorf("questions.%s.instructions is required", key)
		}
		switch q.Type {
		case "noul":
			if !isEmptyJSON(q.Criteria) {
				var criteria map[string]json.RawMessage
				if json.Unmarshal(q.Criteria, &criteria) != nil {
					return fmt.Errorf("questions.%s.criteria must be an object with true/false", key)
				}
				for k := range criteria {
					if k != "true" && k != "false" {
						return fmt.Errorf("questions.%s.criteria only accepts true and false", key)
					}
				}
			}
		case "choice":
			var criteria map[string]json.RawMessage
			if json.Unmarshal(q.Criteria, &criteria) != nil || len(criteria) < 2 {
				return fmt.Errorf("questions.%s.criteria must map at least two options to descriptions", key)
			}
			if len(criteria) > 255 {
				return fmt.Errorf("questions.%s.criteria allows at most 255 options", key)
			}
		case "score":
			var criteria []json.RawMessage
			if json.Unmarshal(q.Criteria, &criteria) != nil || len(criteria) < 2 || len(criteria) > 10 {
				return fmt.Errorf("questions.%s.criteria must be an array of 2 to 10 levels", key)
			}
		default:
			return fmt.Errorf("questions.%s.type must be noul, choice or score", key)
		}
	}
	return nil
}

func isEmptyJSON(raw json.RawMessage) bool {
	s := strings.TrimSpace(string(raw))
	return s == "" || s == "null" || s == `""` || s == "{}" || s == "[]"
}
