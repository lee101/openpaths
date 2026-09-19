package model

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Skill is a single record in the searchable agent-skill library (skills table).
// JSON tags match app-site's /api/skills shape and the corpus skills_seed.json so
// the same data flows between both systems unchanged.
type Skill struct {
	ID             string      `json:"id"`
	Slug           string      `json:"slug"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	Body           string      `json:"body,omitempty"`
	Source         string      `json:"source"`
	SourceRepo     string      `json:"sourceRepo,omitempty"`
	Category       string      `json:"category,omitempty"`
	Tags           []string    `json:"tags"`
	SetupPreamble  string      `json:"setupPreamble,omitempty"`
	OwnerID        string      `json:"ownerId,omitempty"`
	CurrentVersion string      `json:"currentVersion,omitempty"`
	VersionCount   int         `json:"versionCount,omitempty"`
	SetupScript    string      `json:"setupScript,omitempty"`
	SetupPrompt    string      `json:"setupPrompt,omitempty"`
	SkillPrompt    string      `json:"skillPrompt,omitempty"`
	Versions       []string    `json:"versions,omitempty"`
	Files          []SkillFile `json:"files,omitempty"`
	CreatedAt      string      `json:"createdAt,omitempty"`
	UpdatedAt      string      `json:"updatedAt,omitempty"`
}

// SkillFile is a file-attachment manifest entry (no bytes in JSON; fetch raw
// bytes via GET /v1/skills/{slug}/files/{path}).
type SkillFile struct {
	Path string `json:"path"`
	Mime string `json:"mime,omitempty"`
	Size int    `json:"size,omitempty"`
}

// SkillVersion is one immutable snapshot of a skill's content.
type SkillVersion struct {
	SkillID     string `json:"skillId"`
	Version     string `json:"version"`
	Body        string `json:"body,omitempty"`
	SetupScript string `json:"setupScript,omitempty"`
	SetupPrompt string `json:"setupPrompt,omitempty"`
	SkillPrompt string `json:"skillPrompt,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
}

var skillVersionRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)

// IsValidVersion reports whether v is semver-ish '1.0.0'.
func IsValidVersion(v string) bool {
	return skillVersionRe.MatchString(strings.TrimSpace(v))
}

// NormalizeVersion trims whitespace; empty stays empty so callers can
// distinguish "not supplied" (auto bump) from an explicit pin.
func NormalizeVersion(v string) string {
	return strings.TrimSpace(v)
}

// BumpPatchVersion increments the patch component ('1.2.3' -> '1.2.4').
// Malformed or empty input yields '1.0.0'.
func BumpPatchVersion(v string) string {
	m := skillVersionRe.FindStringSubmatch(strings.TrimSpace(v))
	if m == nil {
		return "1.0.0"
	}
	patch, _ := strconv.Atoi(m[3])
	return fmt.Sprintf("%s.%s.%d", m[1], m[2], patch+1)
}

// SkillFilters narrows browse/search queries.
type SkillFilters struct {
	Source   string
	Category string
}

// SearchText is the text embedded for semantic search: name, description, tags,
// and a leading excerpt of the body.
func (s Skill) SearchText() string {
	body := s.Body
	if r := []rune(body); len(r) > 800 {
		body = string(r[:800])
	}
	return s.Name + "\n" + s.Description + "\n" + strings.Join(s.Tags, " ") + "\n" + body
}

// Markdown is the copy-paste-ready payload: the Setup clone preamble prepended to
// the skill body.
func (s Skill) Markdown() string {
	if strings.TrimSpace(s.SetupPreamble) == "" {
		return s.Body
	}
	return s.SetupPreamble + "\n\n" + s.Body
}
