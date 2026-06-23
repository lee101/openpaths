package model

import "strings"

// Skill is a single record in the searchable agent-skill library (skills table).
// JSON tags match app-site's /api/skills shape and the corpus skills_seed.json so
// the same data flows between both systems unchanged.
type Skill struct {
	ID            string   `json:"id"`
	Slug          string   `json:"slug"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Body          string   `json:"body,omitempty"`
	Source        string   `json:"source"`
	SourceRepo    string   `json:"sourceRepo,omitempty"`
	Category      string   `json:"category,omitempty"`
	Tags          []string `json:"tags"`
	SetupPreamble string   `json:"setupPreamble,omitempty"`
	CreatedAt     string   `json:"createdAt,omitempty"`
	UpdatedAt     string   `json:"updatedAt,omitempty"`
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
