// Package promptlib is the source of truth for the OpenPaths prompt library
// (the /prompts directory). It defines prompt categories, the OpenPaths models
// each prompt targets, and a curated + generated set of prompt definitions.
//
// The data here is intentionally self-contained (no DB) so it can be embedded,
// embedded-for-search via gobed (see index.go), and served read-only over HTTP.
// It mirrors the structure of text-generator.io's prompt fixtures but maps every
// prompt to a real OpenPaths model id so "Open in Playground" deep-links work.
package promptlib

import (
	"sort"
	"strings"
)

// Category is a business/use-case grouping for prompts.
type Category struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	ShortName   string `json:"shortName"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// Modality is the prompt output type (image/text/video/music) plus the synthetic
// "free" type for free-to-try starters.
type Modality struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// Model is an OpenPaths model a prompt is tuned for. Slug is the real model id
// used by the playground/API so deep-links resolve.
type Model struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Modality    string `json:"modality"`
	Icon        string `json:"icon"`
}

// Prompt is a fully-built, render-ready prompt entry.
type Prompt struct {
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	Prompt       string   `json:"prompt"`
	ModelSlug    string   `json:"modelSlug"`
	ModelName    string   `json:"modelName"`
	ModelIcon    string   `json:"modelIcon"`
	CategorySlug string   `json:"categorySlug"`
	CategoryName string   `json:"categoryName"`
	CategoryIcon string   `json:"categoryIcon"`
	Modality     string   `json:"modality"`
	ModalityName string   `json:"modalityName"`
	Tags         []string `json:"tags"`
	IsFree       bool     `json:"isFree"`
	Featured     bool     `json:"featured"`
	Popularity   int      `json:"popularity"`
	URL          string   `json:"url"`
	ModelURL     string   `json:"modelUrl"`
	CategoryURL  string   `json:"categoryUrl"`
	ModalityURL  string   `json:"modalityUrl"`
	SearchText   string   `json:"-"`
}

// definition is the hand-authored / generated input shape, built into a Prompt.
type definition struct {
	Slug       string
	Title      string
	Summary    string
	Prompt     string
	ModelSlug  string
	CategSlug  string
	Modality   string
	Tags       []string
	IsFree     bool
	Featured   bool
	Popularity int
}

// ---------------------------------------------------------------------------
// Taxonomy
// ---------------------------------------------------------------------------

var categories = []Category{
	{"coding-dev", "Coding & Dev prompts", "Coding & Dev", "Autocomplete, refactors, tests, debugging, and code generation prompts for real engineering work.", "code"},
	{"art-illustration", "Art & Illustration prompts", "Art & Illustration", "Concept art, key art, editorial scenes, and visual storytelling prompts.", "palette"},
	{"logo-icon", "Logo & Icon prompts", "Logo & Icon", "Brand marks, icon systems, social avatars, and recognizable symbol work.", "gesture"},
	{"graphic-design", "Graphic & Design prompts", "Graphic & Design", "Layouts, packaging, design systems, and production-ready creative directions.", "grid"},
	{"productivity-writing", "Productivity & Writing prompts", "Productivity & Writing", "Operational writing, summaries, memos, planning docs, and knowledge work prompts.", "edit"},
	{"marketing-business", "Marketing & Business prompts", "Marketing & Business", "Positioning, campaigns, ads, email, brand strategy, and growth prompts.", "campaign"},
	{"photography", "Photography prompts", "Photography", "Portrait, product, editorial, cinematic, and brand photography prompts.", "camera"},
	{"video-motion", "Video & Motion prompts", "Video & Motion", "Storyboards, camera moves, b-roll, and cinematic motion prompts.", "movie"},
	{"music-audio", "Music & Audio prompts", "Music & Audio", "Song concepts, scores, jingles, and sound design prompts.", "music"},
}

var modalities = []Modality{
	{"text", "Text prompts", "Prompts for code, copywriting, analysis, planning, summaries, and documents.", "description"},
	{"image", "Image prompts", "Prompts for still image generation, concept art, logos, and design work.", "image"},
	{"video", "Video prompts", "Prompts for motion, storyboards, camera moves, and cinematic outputs.", "movie"},
	{"music", "Music prompts", "Prompts for songs, scores, jingles, and sound design.", "music"},
	{"free", "Free prompts", "Prompts marked as free-to-try starters for rapid experimentation.", "bolt"},
}

// Models map to real OpenPaths model ids.
var models = []Model{
	// text / code
	{"openpaths/auto", "OpenPaths Auto", "Automatically routes to the best chat model for the task.", "text", "auto_awesome"},
	{"openpaths/auto-code", "OpenPaths Auto (Code)", "Auto-routes to the strongest coding model available.", "text", "code"},
	{"gpt-5-codex", "GPT-5 Codex", "OpenAI coding model for autocomplete, refactors, and generation.", "text", "terminal"},
	{"composer-2.5", "Composer 2.5", "Cursor's fast agentic coding model.", "text", "bolt"},
	{"cursor-grok-4.6", "Cursor Grok 4.6", "Cursor-hosted Grok model for long-running agents and knowledge work.", "text", "smart_toy"},
	{"cursor-grok-4.5", "Cursor Grok 4.5", "Cursor-hosted Grok model for coding and agentic knowledge work.", "text", "smart_toy"},
	{"gpt-5.5", "GPT-5.5", "OpenAI flagship for reasoning, writing, and analysis.", "text", "smart_toy"},
	{"claude-opus-5", "Claude Opus 5", "Anthropic's most capable model for writing and reasoning.", "text", "psychology"},
	{"claude-opus-4-8", "Claude Opus 4.8", "Previous-gen Anthropic flagship for writing and reasoning.", "text", "psychology"},
	{"claude-opus-4-7", "Claude Opus 4.7", "Older Anthropic flagship for writing and reasoning.", "text", "psychology"},
	{"gemini-2.5-pro", "Gemini 2.5 Pro", "Google's long-context reasoning and writing model.", "text", "stars"},
	// image
	{"zimage", "ZImage", "OpenPaths image model for fast, high-quality generation.", "image", "image"},
	{"flux-pro", "FLUX Pro", "Crisp, modern image generation with strong prompt adherence.", "image", "auto_awesome"},
	{"flux-dev", "FLUX Dev", "Flexible open image model for iteration and style work.", "image", "brush"},
	{"hidream-o1-image-dev", "HiDream o1", "Detailed photoreal and stylized image generation.", "image", "photo_filter"},
	{"grok-imagine-image", "Grok Imagine", "xAI image generation for concepts and visuals.", "image", "image_search"},
	// video
	{"wan", "Wan", "Text-to-video generation for motion and cinematic clips.", "video", "movie_creation"},
	// music
	{"lyria-3-pro-preview", "Lyria 3 Pro", "Google music generation for songs, scores, and clips.", "music", "music_note"},
}

var (
	categoryBySlug = indexCategories()
	modalityBySlug = indexModalities()
	modelBySlug    = indexModels()
)

func indexCategories() map[string]Category {
	m := make(map[string]Category, len(categories))
	for _, c := range categories {
		m[c.Slug] = c
	}
	return m
}
func indexModalities() map[string]Modality {
	m := make(map[string]Modality, len(modalities))
	for _, x := range modalities {
		m[x.Slug] = x
	}
	return m
}
func indexModels() map[string]Model {
	m := make(map[string]Model, len(models))
	for _, x := range models {
		m[x.Slug] = x
	}
	return m
}

// ---------------------------------------------------------------------------
// Build
// ---------------------------------------------------------------------------

func buildPrompt(d definition) Prompt {
	model := modelBySlug[d.ModelSlug]
	cat := categoryBySlug[d.CategSlug]
	mod := modalityBySlug[d.Modality]

	tags := dedupeSorted(d.Tags)
	parts := []string{d.Title, d.Summary, d.Prompt, model.Name, cat.Name, mod.Name, strings.Join(tags, " ")}
	if d.IsFree {
		parts = append(parts, "free prompts")
	}

	return Prompt{
		Slug:         d.Slug,
		Title:        d.Title,
		Summary:      d.Summary,
		Prompt:       d.Prompt,
		ModelSlug:    model.Slug,
		ModelName:    model.Name,
		ModelIcon:    model.Icon,
		CategorySlug: cat.Slug,
		CategoryName: cat.Name,
		CategoryIcon: cat.Icon,
		Modality:     d.Modality,
		ModalityName: mod.Name,
		Tags:         tags,
		IsFree:       d.IsFree,
		Featured:     d.Featured,
		Popularity:   d.Popularity,
		URL:          "/prompts/" + d.Slug,
		ModelURL:     "/prompts/model/" + model.Slug,
		CategoryURL:  "/prompts/category/" + cat.Slug,
		ModalityURL:  "/prompts/type/" + d.Modality,
		SearchText:   strings.ToLower(strings.Join(parts, " ")),
	}
}

func dedupeSorted(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// allPrompts is the materialized, sorted catalog (curated seed + generated variants).
var allPrompts = buildAll()

func buildAll() []Prompt {
	defs := append([]definition{}, seedDefinitions...)
	defs = append(defs, generateAcrossModels(existingSlugs(defs))...)

	prompts := make([]Prompt, 0, len(defs))
	for _, d := range defs {
		if _, ok := categoryBySlug[d.CategSlug]; !ok {
			continue
		}
		if _, ok := modelBySlug[d.ModelSlug]; !ok {
			continue
		}
		prompts = append(prompts, buildPrompt(d))
	}
	sort.SliceStable(prompts, func(i, j int) bool {
		if prompts[i].Featured != prompts[j].Featured {
			return prompts[i].Featured
		}
		if prompts[i].Popularity != prompts[j].Popularity {
			return prompts[i].Popularity > prompts[j].Popularity
		}
		return prompts[i].Title < prompts[j].Title
	})
	return prompts
}

func existingSlugs(defs []definition) map[string]struct{} {
	m := make(map[string]struct{}, len(defs))
	for _, d := range defs {
		m[d.Slug] = struct{}{}
	}
	return m
}

// ---------------------------------------------------------------------------
// Public getters
// ---------------------------------------------------------------------------

// All returns the full prompt catalog (already sorted).
func All() []Prompt { return allPrompts }

// Count returns the total number of prompts.
func Count() int { return len(allPrompts) }

// BySlug returns a prompt and whether it was found.
func BySlug(slug string) (Prompt, bool) {
	for _, p := range allPrompts {
		if p.Slug == slug {
			return p, true
		}
	}
	return Prompt{}, false
}

// Filters narrows the catalog.
type Filters struct {
	Category string
	Model    string
	Modality string // image|text|video|music; "free" is handled via Free
	Free     bool
	Query    string // lexical fallback search
}

func (f Filters) matches(p Prompt) bool {
	if f.Category != "" && p.CategorySlug != f.Category {
		return false
	}
	if f.Model != "" && p.ModelSlug != f.Model {
		return false
	}
	if f.Free && !p.IsFree {
		return false
	}
	if f.Modality != "" {
		if f.Modality == "free" {
			if !p.IsFree {
				return false
			}
		} else if p.Modality != f.Modality {
			return false
		}
	}
	return true
}

// Filter applies metadata filters and an optional lexical query (no embeddings).
func Filter(f Filters) []Prompt {
	terms := tokenize(f.Query)
	out := make([]Prompt, 0, len(allPrompts))
	type scored struct {
		p     Prompt
		score int
	}
	var ranked []scored
	for _, p := range allPrompts {
		if !f.matches(p) {
			continue
		}
		if len(terms) == 0 {
			out = append(out, p)
			continue
		}
		s := lexicalScore(p, terms, strings.ToLower(strings.TrimSpace(f.Query)))
		if s > 0 {
			ranked = append(ranked, scored{p, s})
		}
	}
	if len(terms) == 0 {
		return out
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].p.Popularity > ranked[j].p.Popularity
	})
	res := make([]Prompt, len(ranked))
	for i, r := range ranked {
		res[i] = r.p
	}
	return res
}

func lexicalScore(p Prompt, terms []string, raw string) int {
	score := 0
	for _, t := range terms {
		if strings.Contains(p.SearchText, t) {
			if len(t) > 4 {
				score += 3
			} else {
				score++
			}
		}
		for _, tag := range p.Tags {
			if strings.EqualFold(tag, t) {
				score += 4
			}
		}
	}
	if raw != "" {
		if strings.Contains(strings.ToLower(p.Title), raw) {
			score += 8
		} else if strings.Contains(strings.ToLower(p.Summary), raw) {
			score += 4
		}
	}
	if p.Featured {
		score++
	}
	return score
}

func tokenize(s string) []string {
	var out []string
	var b strings.Builder
	flush := func() {
		if b.Len() > 1 {
			out = append(out, b.String())
		}
		b.Reset()
	}
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return out
}

// RelatedMeta returns related prompts by category/model overlap (used when the
// semantic index is unavailable).
func RelatedMeta(slug string, limit int) []Prompt {
	target, ok := BySlug(slug)
	if !ok {
		return nil
	}
	if limit <= 0 {
		limit = 4
	}
	type scored struct {
		p     Prompt
		score int
	}
	var ranked []scored
	for _, p := range allPrompts {
		if p.Slug == slug {
			continue
		}
		score := 0
		if p.CategorySlug == target.CategorySlug {
			score += 3
		}
		if p.ModelSlug == target.ModelSlug {
			score += 2
		}
		if p.Modality == target.Modality {
			score++
		}
		for _, t := range target.Tags {
			for _, t2 := range p.Tags {
				if t == t2 {
					score++
				}
			}
		}
		if score == 0 {
			continue
		}
		if p.Featured {
			score++
		}
		ranked = append(ranked, scored{p, score})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].p.Popularity > ranked[j].p.Popularity
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]Prompt, len(ranked))
	for i, r := range ranked {
		out[i] = r.p
	}
	return out
}

// Categories returns the taxonomy with per-category prompt counts.
func Categories() []Category { return append([]Category(nil), categories...) }

// Modalities returns the modality/type taxonomy.
func Modalities() []Modality { return append([]Modality(nil), modalities...) }

// Models returns the model taxonomy.
func Models() []Model { return append([]Model(nil), models...) }

// Counts returns counts keyed by category, model, and modality slug.
func Counts() (byCategory, byModel, byModality map[string]int, free int) {
	byCategory = map[string]int{}
	byModel = map[string]int{}
	byModality = map[string]int{}
	for _, p := range allPrompts {
		byCategory[p.CategorySlug]++
		byModel[p.ModelSlug]++
		byModality[p.Modality]++
		if p.IsFree {
			free++
		}
	}
	return
}

// SitemapPaths returns every prompt-library URL path for the sitemap.
func SitemapPaths() []string {
	paths := []string{"/prompts"}
	for _, c := range categories {
		paths = append(paths, "/prompts/category/"+c.Slug)
	}
	for _, m := range models {
		paths = append(paths, "/prompts/model/"+m.Slug)
	}
	for _, m := range modalities {
		paths = append(paths, "/prompts/type/"+m.Slug)
	}
	for _, p := range allPrompts {
		paths = append(paths, p.URL)
	}
	return paths
}
