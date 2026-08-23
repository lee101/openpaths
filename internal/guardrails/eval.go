package guardrails

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/openpaths/openpaths/internal/db/queries"
)

// Actions
const (
	ActionBlock  = "block"
	ActionRedact = "redact"
	ActionEmail  = "email"
	ActionFlag   = "flag"
)

// Stages
const (
	StageBudget          = "budget"
	StageModel           = "model"
	StageProvider        = "provider"
	StagePromptInjection = "prompt_injection"
	StageSensitiveInfo   = "sensitive_info"
	StageCustom          = "custom"
)

type PromptInjectionConfig struct {
	Enabled  bool     `json:"enabled"`
	Action   string   `json:"action"` // block | email | flag (default block)
	Patterns []string `json:"patterns,omitempty"`
}

type SensitiveFilter struct {
	Slug   string `json:"slug"`
	Action string `json:"action"` // block | redact | email
}

type SensitiveInfoConfig struct {
	Filters []SensitiveFilter `json:"filters"`
}

type CustomFilter struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Action  string `json:"action"` // block | redact | email
}

type Hit struct {
	Stage      string   `json:"stage"`
	Action     string   `json:"action"`
	Guardrail  string   `json:"guardrail_id"`
	Name       string   `json:"guardrail_name,omitempty"`
	Patterns   []string `json:"patterns,omitempty"`
	Slug       string   `json:"slug,omitempty"`
	Message    string   `json:"message"`
	ShouldStop bool     `json:"-"`
}

type EvalResult struct {
	Blocked     bool
	Hits        []Hit
	Redacted    string // full text after redactions (empty if unchanged / unused)
	TextChanged bool
	Providers   []string // intersection of provider allowlists (nil = unrestricted)
	ModelsOK    bool
}

var builtinInjection = []string{
	`(?i)ignore\s+(all\s+)?(previous|prior|above)\s+(instructions|prompts|rules)`,
	`(?i)disregard\s+(all\s+)?(previous|prior|above)\s+(instructions|prompts|rules)`,
	`(?i)forget\s+(all\s+)?(previous|prior|above)\s+(instructions|prompts|rules)`,
	`(?i)you\s+are\s+now\s+(dan|jailbroken|unrestricted)`,
	`(?i)jailbreak`,
	`(?i)do\s+not\s+follow\s+(your\s+)?(system|safety)\s+(prompt|instructions)`,
	`(?i)reveal\s+(your\s+)?(system\s+)?(prompt|instructions)`,
	`(?i)override\s+(your\s+)?(safety|content)\s+(filters?|policies|guidelines)`,
}

var piiPatterns = map[string]*regexp.Regexp{
	"email":       regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`),
	"phone":       regexp.MustCompile(`(?:\+?\d{1,3}[\s\-.]?)?(?:\(?\d{3}\)?[\s\-.]?)\d{3}[\s\-.]?\d{4}\b`),
	"ssn":         regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
	"credit-card": regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`),
	"ip-address":  regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(?:25[0-5]|2[0-4]\d|[01]?\d\d?)\b`),
}

var (
	reLookahead   = regexp.MustCompile(`\(\?=|\(\?!`)
	reLookbehind  = regexp.MustCompile(`\(\?<=|\(\?<!`)
	reNestedQuant = regexp.MustCompile(`\([^)]*[+*][^)]*\)[+*]`)
)

func ParsePromptInjection(raw json.RawMessage) PromptInjectionConfig {
	var c PromptInjectionConfig
	_ = json.Unmarshal(raw, &c)
	if c.Action == "" {
		c.Action = ActionBlock
	}
	return c
}

func ParseSensitiveInfo(raw json.RawMessage) SensitiveInfoConfig {
	var c SensitiveInfoConfig
	_ = json.Unmarshal(raw, &c)
	return c
}

func ParseCustomFilters(raw json.RawMessage) []CustomFilter {
	var c []CustomFilter
	_ = json.Unmarshal(raw, &c)
	return c
}

func ValidateRegex(pattern string) error {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return fmt.Errorf("empty pattern")
	}
	if len(pattern) > 100_000 {
		return fmt.Errorf("pattern too long")
	}
	if reLookahead.MatchString(pattern) || reLookbehind.MatchString(pattern) {
		return fmt.Errorf("lookahead/lookbehind not allowed")
	}
	if reNestedQuant.MatchString(pattern) {
		return fmt.Errorf("nested quantifiers not allowed")
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return err
	}
	return nil
}

func MatchGlob(pattern, value string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	value = strings.ToLower(strings.TrimSpace(value))
	if pattern == "" || pattern == "*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return pattern == value
	}
	parts := strings.Split(pattern, "*")
	if !strings.HasPrefix(value, parts[0]) {
		return false
	}
	rest := value[len(parts[0]):]
	for i := 1; i < len(parts); i++ {
		p := parts[i]
		if p == "" {
			if i == len(parts)-1 {
				return true
			}
			continue
		}
		idx := strings.Index(rest, p)
		if idx < 0 {
			return false
		}
		rest = rest[idx+len(p):]
	}
	return true
}

func ModelAllowed(allowed []string, modelID string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if MatchGlob(a, modelID) {
			return true
		}
	}
	return false
}

func ProviderAllowed(allowed []string, provider string) bool {
	if len(allowed) == 0 {
		return true
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	for _, a := range allowed {
		if MatchGlob(a, provider) {
			return true
		}
	}
	return false
}

func MatchesAny(patterns []string, value string) bool {
	for _, pattern := range patterns {
		if strings.TrimSpace(pattern) == "" {
			continue
		}
		if MatchGlob(pattern, value) {
			return true
		}
	}
	return false
}

func ModelBlocked(blocked []string, modelID string) bool {
	return MatchesAny(blocked, modelID)
}

func ProviderBlocked(blocked []string, provider string) bool {
	return MatchesAny(blocked, provider)
}

// BlockedProviders returns the union of all provider deny rules. A deny rule
// is deliberately not converted into an allowlist: the router may discover
// providers after this middleware runs, so the deny list must travel with the
// request to the candidate filter.
func BlockedProviders(policies []*queries.Guardrail) []string {
	seen := map[string]bool{}
	var out []string
	for _, g := range policies {
		if g == nil {
			continue
		}
		for _, p := range g.BlockedProviders {
			p = strings.ToLower(strings.TrimSpace(p))
			if p != "" && !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out
}

func IntersectProviders(lists ...[]string) []string {
	var result []string
	first := true
	for _, list := range lists {
		if len(list) == 0 {
			continue // unrestricted
		}
		norm := make([]string, 0, len(list))
		for _, p := range list {
			p = strings.ToLower(strings.TrimSpace(p))
			if p != "" {
				norm = append(norm, p)
			}
		}
		if first {
			result = norm
			first = false
			continue
		}
		set := map[string]bool{}
		for _, p := range norm {
			set[p] = true
		}
		var next []string
		for _, p := range result {
			if set[p] {
				next = append(next, p)
			}
		}
		result = next
	}
	if first {
		return nil
	}
	return result
}

func PeriodStart(interval string, now time.Time) time.Time {
	now = now.UTC()
	y, m, d := now.Date()
	switch strings.ToLower(interval) {
	case "weekly":
		// Monday 00:00 UTC
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		start := time.Date(y, m, d, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -(weekday - 1))
		return start
	case "monthly":
		return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	default: // daily
		return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
}

func HasAction(actions []string, want string) bool {
	for _, a := range actions {
		if strings.EqualFold(strings.TrimSpace(a), want) {
			return true
		}
	}
	return false
}

func actionRank(a string) int {
	switch a {
	case ActionBlock:
		return 3
	case ActionRedact:
		return 2
	case ActionEmail, ActionFlag:
		return 1
	default:
		return 0
	}
}

func strongerAction(a, b string) string {
	if actionRank(a) >= actionRank(b) {
		return a
	}
	return b
}

// EvaluateContent runs injection, PII, and custom filters against text.
func EvaluateContent(policies []*queries.Guardrail, text string) EvalResult {
	res := EvalResult{ModelsOK: true}
	cur := text
	changed := false

	for _, g := range policies {
		if g == nil {
			continue
		}
		pi := ParsePromptInjection(g.PromptInjection)
		if pi.Enabled {
			patterns := append([]string{}, builtinInjection...)
			patterns = append(patterns, pi.Patterns...)
			var matched []string
			for _, p := range patterns {
				re, err := regexp.Compile(p)
				if err != nil {
					continue
				}
				if re.MatchString(cur) {
					matched = append(matched, p)
				}
			}
			if len(matched) > 0 {
				act := pi.Action
				if act == "" {
					act = ActionBlock
				}
				hit := Hit{
					Stage: StagePromptInjection, Action: act, Guardrail: g.ID, Name: g.Name,
					Patterns: matched, Message: "Request blocked: prompt injection patterns detected",
					ShouldStop: act == ActionBlock,
				}
				res.Hits = append(res.Hits, hit)
				if act == ActionBlock {
					res.Blocked = true
				}
			}
		}

		si := ParseSensitiveInfo(g.SensitiveInfo)
		for _, f := range si.Filters {
			re := piiPatterns[f.Slug]
			if re == nil {
				continue
			}
			if !re.MatchString(cur) {
				continue
			}
			act := f.Action
			if act == "" {
				act = ActionBlock
			}
			hit := Hit{
				Stage: StageSensitiveInfo, Action: act, Guardrail: g.ID, Name: g.Name,
				Slug: f.Slug, Message: fmt.Sprintf("Sensitive info detected: %s", f.Slug),
				ShouldStop: act == ActionBlock,
			}
			res.Hits = append(res.Hits, hit)
			if act == ActionBlock {
				res.Blocked = true
			} else if act == ActionRedact {
				next := re.ReplaceAllString(cur, "[REDACTED:"+strings.ToUpper(f.Slug)+"]")
				if next != cur {
					cur = next
					changed = true
				}
			}
		}

		for _, cf := range ParseCustomFilters(g.CustomFilters) {
			if strings.TrimSpace(cf.Pattern) == "" {
				continue
			}
			re, err := regexp.Compile(cf.Pattern)
			if err != nil {
				continue
			}
			if !re.MatchString(cur) {
				continue
			}
			act := cf.Action
			if act == "" {
				act = ActionBlock
			}
			label := cf.Name
			if label == "" {
				label = cf.Pattern
			}
			hit := Hit{
				Stage: StageCustom, Action: act, Guardrail: g.ID, Name: g.Name,
				Patterns: []string{label}, Message: fmt.Sprintf("Custom filter matched: %s", label),
				ShouldStop: act == ActionBlock,
			}
			res.Hits = append(res.Hits, hit)
			if act == ActionBlock {
				res.Blocked = true
			} else if act == ActionRedact {
				next := re.ReplaceAllString(cur, "[REDACTED]")
				if next != cur {
					cur = next
					changed = true
				}
			}
		}
	}

	if changed {
		res.TextChanged = true
		res.Redacted = cur
	}
	return res
}

// EvaluateAccess checks model allow/block lists and builds provider allowlist intersection.
func EvaluateAccess(policies []*queries.Guardrail, modelID string) (blocked *Hit, providers []string) {
	var providerLists [][]string
	for _, g := range policies {
		if g == nil {
			continue
		}
		if modelID != "" && ModelBlocked(g.BlockedModels, modelID) {
			return &Hit{
				Stage: StageModel, Action: ActionBlock, Guardrail: g.ID, Name: g.Name,
				Message:    fmt.Sprintf("Model %q is blocked by guardrail %q", modelID, g.Name),
				ShouldStop: true,
			}, nil
		}
		if modelID != "" && !ModelAllowed(g.AllowedModels, modelID) {
			return &Hit{
				Stage: StageModel, Action: ActionBlock, Guardrail: g.ID, Name: g.Name,
				Message:    fmt.Sprintf("Model %q is not allowed by guardrail %q", modelID, g.Name),
				ShouldStop: true,
			}, nil
		}
		providerLists = append(providerLists, g.AllowedProviders)
	}
	return nil, IntersectProviders(providerLists...)
}

// ExtractText pulls user-visible strings from common request JSON shapes.
func ExtractText(body []byte) string {
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return string(body)
	}
	var b strings.Builder
	walkText(root, &b)
	return b.String()
}

func walkText(v any, b *strings.Builder) {
	switch t := v.(type) {
	case string:
		if looksLikeBinary(t) {
			return
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(t)
	case []any:
		for _, item := range t {
			walkText(item, b)
		}
	case map[string]any:
		// Prefer known text fields; still walk nested for messages/content.
		for _, key := range []string{"messages", "input", "prompt", "content", "text", "query", "q"} {
			if child, ok := t[key]; ok {
				walkText(child, b)
			}
		}
		// Also walk role/content style message objects.
		if role, ok := t["role"].(string); ok && role != "" {
			if c, ok := t["content"]; ok {
				walkText(c, b)
			}
		}
	}
}

func looksLikeBinary(s string) bool {
	if len(s) > 8 && (strings.HasPrefix(s, "data:") || strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")) {
		return len(s) > 200
	}
	nonPrint := 0
	n := 0
	for _, r := range s {
		n++
		if n > 200 {
			break
		}
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			nonPrint++
		}
	}
	return n > 20 && nonPrint*3 > n
}

// ApplyPolicyRedactions rewrites string leaves for any redact-action filters.
func ApplyPolicyRedactions(body []byte, policies []*queries.Guardrail) []byte {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return body
	}
	out, err := json.Marshal(rewriteNode(root, policies))
	if err != nil {
		return body
	}
	return out
}

// ApplyRedactions kept for callers that only need builtin PII scrub.
func ApplyRedactions(body []byte, _, _ string) []byte {
	return ApplyPolicyRedactions(body, nil)
}

func rewriteNode(v any, policies []*queries.Guardrail) any {
	switch t := v.(type) {
	case string:
		out := t
		for slug, re := range piiPatterns {
			need := policies == nil
			if !need {
				for _, g := range policies {
					for _, f := range ParseSensitiveInfo(g.SensitiveInfo).Filters {
						if f.Slug == slug && f.Action == ActionRedact {
							need = true
						}
					}
				}
			}
			if need {
				out = re.ReplaceAllString(out, "[REDACTED:"+strings.ToUpper(slug)+"]")
			}
		}
		for _, g := range policies {
			for _, cf := range ParseCustomFilters(g.CustomFilters) {
				if cf.Action != ActionRedact || strings.TrimSpace(cf.Pattern) == "" {
					continue
				}
				re, err := regexp.Compile(cf.Pattern)
				if err != nil {
					continue
				}
				out = re.ReplaceAllString(out, "[REDACTED]")
			}
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = rewriteNode(item, policies)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, child := range t {
			out[k] = rewriteNode(child, policies)
		}
		return out
	default:
		return v
	}
}
