package promptlib

import (
	"fmt"
	"strings"
)

// seedDefinitions is the curated, hand-authored set of prompts. Featured and
// high-popularity entries surface first. generateAcrossModels() expands the
// catalog by fanning blueprints across models.
var seedDefinitions = []definition{
	// ---------------------- Coding & Dev ----------------------
	{
		Slug:      "autocomplete-pytorch-coding",
		Title:     "Autocomplete PyTorch Coding",
		Summary:   "Turn a sketch of a PyTorch module into complete, idiomatic, runnable code.",
		ModelSlug: "openpaths/auto-code", CategSlug: "coding-dev", Modality: "text",
		Tags:   []string{"pytorch", "autocomplete", "deep learning", "python", "code generation"},
		IsFree: true, Featured: true, Popularity: 99,
		Prompt: `You are an expert PyTorch engineer. Complete the following code.

Rules:
- Finish partially-written functions/classes; do not rewrite working parts.
- Use idiomatic, modern PyTorch (nn.Module, typed tensors, device-agnostic).
- Add concise inline comments only where intent is non-obvious.
- Include shape annotations for every tensor op.
- If a training loop is implied, add optimizer, loss, and a step that runs.

Code so far:
` + "```python\n" + `import torch
import torch.nn as nn

class TinyResNetBlock(nn.Module):
    def __init__(self, channels: int):
        super().__init__()
        # TODO: two 3x3 convs with BN + ReLU and a residual connection
` + "```\n\nReturn only the completed code block.",
	},
	{
		Slug:      "refactor-function-clean",
		Title:     "Refactor This Function",
		Summary:   "Refactor a messy function for readability and testability without changing behavior.",
		ModelSlug: "gpt-5-codex", CategSlug: "coding-dev", Modality: "text",
		Tags:   []string{"refactor", "clean code", "review", "maintainability"},
		IsFree: true, Featured: true, Popularity: 96,
		Prompt: `Refactor the function below.

Constraints:
- Preserve exact behavior and public signature.
- Improve naming, reduce nesting, and extract small helpers where it clarifies intent.
- Note any latent bugs you spot in a short list AFTER the code.
- Keep the same language and style as the surrounding codebase.

Paste your function between the fences:
` + "```\n[your function]\n```",
	},
	{
		Slug:      "write-unit-tests",
		Title:     "Write Unit Tests",
		Summary:   "Generate a thorough table-driven test suite for a function, including edge cases.",
		ModelSlug: "openpaths/auto-code", CategSlug: "coding-dev", Modality: "text",
		Tags:   []string{"testing", "unit tests", "edge cases", "tdd"},
		IsFree: true, Featured: true, Popularity: 94,
		Prompt: `Write a comprehensive test suite for the code below.

Include:
- Happy-path cases and at least 5 edge cases (empty, nil/None, boundary, overflow, malformed).
- Table-driven structure where the language supports it.
- Clear test names that describe the scenario.
- No network or filesystem unless the function requires it (mock instead).

Target framework: match the project's existing test style.

Code:
` + "```\n[paste code]\n```",
	},
	{
		Slug:      "explain-this-code",
		Title:     "Explain This Code",
		Summary:   "Get a layered explanation of unfamiliar code, from one-liner to line-by-line.",
		ModelSlug: "claude-opus-4-7", CategSlug: "coding-dev", Modality: "text",
		Tags:   []string{"explain", "onboarding", "documentation", "review"},
		IsFree: true, Featured: true, Popularity: 92,
		Prompt: `Explain the following code in three layers:

1. One sentence: what it does and why it exists.
2. A short walkthrough of the control flow and key data structures.
3. Line-by-line notes for any non-obvious or clever parts.

Flag anything that looks like a bug, race, or performance trap.

Code:
` + "```\n[paste code]\n```",
	},
	{
		Slug:      "debug-stack-trace",
		Title:     "Debug This Stack Trace",
		Summary:   "Diagnose an error from a stack trace and propose the minimal fix.",
		ModelSlug: "gpt-5-codex", CategSlug: "coding-dev", Modality: "text",
		Tags:   []string{"debugging", "stack trace", "error", "fix"},
		IsFree: true, Featured: true, Popularity: 91,
		Prompt: `Diagnose this error.

Given the stack trace and the relevant code, tell me:
- The most likely root cause (and 1-2 alternates).
- The minimal change that fixes it.
- How to verify the fix and prevent regressions.

Stack trace:
` + "```\n[paste trace]\n```\n\nRelevant code:\n```\n[paste code]\n```",
	},
	{
		Slug:      "sql-query-builder",
		Title:     "Build a SQL Query",
		Summary:   "Translate a plain-English reporting question into a correct, efficient SQL query.",
		ModelSlug: "openpaths/auto-code", CategSlug: "coding-dev", Modality: "text",
		Tags:     []string{"sql", "database", "analytics", "query"},
		Featured: true, Popularity: 88,
		Prompt: `Write a SQL query for this request.

Request: [describe what you want, e.g. "top 10 customers by revenue last quarter, excluding refunds"]

Schema:
` + "```\n[paste table definitions]\n```\n\n" + `Requirements:
- Standard SQL unless I name a dialect.
- Prefer CTEs over deep subqueries.
- Add a one-line comment explaining any window function or join choice.
- Note assumptions about the data.`,
	},
	{
		Slug:      "react-component-from-spec",
		Title:     "React Component From a Spec",
		Summary:   "Generate a typed, accessible React component from a short description.",
		ModelSlug: "composer-2.5", CategSlug: "coding-dev", Modality: "text",
		Tags:     []string{"react", "typescript", "frontend", "component", "accessibility"},
		Featured: true, Popularity: 87,
		Prompt: `Build a React component.

Spec: [describe the component, props, and states]

Requirements:
- TypeScript with explicit prop types.
- Accessible (labels, roles, keyboard nav).
- Tailwind for styling unless told otherwise.
- No external deps unless essential; if used, list them.
- Include a small usage example.`,
	},
	{
		Slug:      "regex-from-description",
		Title:     "Regex From a Description",
		Summary:   "Get a tested regex plus an explanation and sample matches.",
		ModelSlug: "openpaths/auto-code", CategSlug: "coding-dev", Modality: "text",
		Tags:   []string{"regex", "parsing", "validation"},
		IsFree: true, Popularity: 80,
		Prompt: `Write a regular expression that: [describe what to match].

Provide:
- The regex (PCRE-style unless I name a flavor).
- A plain-English breakdown of each group.
- 3 strings that match and 3 that should NOT match.`,
	},
	{
		Slug:      "git-commit-message",
		Title:     "Write a Commit Message",
		Summary:   "Turn a diff into a clean, conventional commit message.",
		ModelSlug: "openpaths/auto-code", CategSlug: "coding-dev", Modality: "text",
		Tags:   []string{"git", "commit", "conventional commits", "workflow"},
		IsFree: true, Popularity: 78,
		Prompt: `Write a commit message for this diff.

Format:
- Conventional Commits header (type(scope): summary), <=72 chars.
- A short body explaining the why, not the what.
- A footer for breaking changes or issue refs if relevant.

Diff:
` + "```\n[paste git diff]\n```",
	},
	{
		Slug:      "api-docstring-generator",
		Title:     "Generate Docstrings",
		Summary:   "Add complete, accurate docstrings to a module without changing logic.",
		ModelSlug: "gpt-5-codex", CategSlug: "coding-dev", Modality: "text",
		Tags:       []string{"documentation", "docstrings", "api", "python"},
		Popularity: 74,
		Prompt: `Add docstrings to every public function and class below.

Rules:
- Match the project's docstring style (Google/NumPy/JSDoc — infer from context).
- Document params, returns, raises, and side effects.
- Do not change any code.

Code:
` + "```\n[paste code]\n```",
	},

	// ---------------------- Logo & Icon ----------------------
	{
		Slug:      "instagram-logo-prompt",
		Title:     "Instagram Logo Prompt",
		Summary:   "Build a modern social-first logo direction for an Instagram-native brand.",
		ModelSlug: "flux-pro", CategSlug: "logo-icon", Modality: "image",
		Tags:   []string{"instagram", "logo", "social media", "brand identity", "avatar"},
		IsFree: true, Featured: true, Popularity: 90,
		Prompt: `Design a distinctive Instagram-ready logo for [brand].

Requirements:
- Feel premium, modern, and memorable without copying Instagram's existing mark.
- Work as a square avatar, app icon, and profile highlight cover.
- Use a bold gradient-led palette with one dark neutral anchor.
- Show 3 variations: icon-only, wordmark, and stacked lockup.
- Keep linework simple enough to stay legible at 48px.`,
	},
	{
		Slug:      "minimal-app-icon",
		Title:     "Minimal App Icon",
		Summary:   "A clean, scalable app icon that reads at any size.",
		ModelSlug: "zimage", CategSlug: "logo-icon", Modality: "image",
		Tags:     []string{"app icon", "minimal", "logo", "ios", "android"},
		Featured: true, Popularity: 84,
		Prompt: `Create a minimal app icon for [app name], a [one-line description].

Requirements:
- Single strong concept, legible from 16px to 1024px.
- Flat or subtle-gradient style, rounded-square safe area.
- Two color directions: vivid and monochrome.
- No text inside the icon.`,
	},
	{
		Slug:      "mascot-logo",
		Title:     "Mascot Logo",
		Summary:   "A friendly mascot mark with strong community energy.",
		ModelSlug: "flux-dev", CategSlug: "logo-icon", Modality: "image",
		Tags:       []string{"mascot", "logo", "community", "character"},
		Popularity: 70,
		Prompt: `Design a mascot logo for [brand], a platform about [topic].

Requirements:
- Friendly, modern mascot with clean geometry.
- Works tiny as an avatar and large on merch.
- Provide a mascot-led version and a simplified symbol-only version.
- Suggest a 3-color palette.`,
	},

	// ---------------------- Art & Illustration ----------------------
	{
		Slug:      "cinematic-key-art",
		Title:     "Cinematic Key Art",
		Summary:   "Dramatic poster-style key art for a story, game, or campaign.",
		ModelSlug: "flux-pro", CategSlug: "art-illustration", Modality: "image",
		Tags:   []string{"key art", "cinematic", "poster", "concept art"},
		IsFree: true, Featured: true, Popularity: 89,
		Prompt: `Create cinematic key art for [title/theme].

Direction:
- Strong focal subject, dramatic lighting, depth via atmosphere.
- Color grade: [warm/cool/teal-orange]; mood: [epic/tense/hopeful].
- Negative space at top for a title treatment.
- Aspect 2:3 poster framing, photoreal-illustrative hybrid.`,
	},
	{
		Slug:      "isometric-illustration",
		Title:     "Isometric Illustration",
		Summary:   "A crisp isometric scene for a landing page or explainer.",
		ModelSlug: "zimage", CategSlug: "art-illustration", Modality: "image",
		Tags:     []string{"isometric", "illustration", "landing page", "vector"},
		Featured: true, Popularity: 81,
		Prompt: `Create an isometric illustration of [scene, e.g. a small developer workspace].

Style:
- Clean isometric perspective, soft shadows, rounded forms.
- Limited 4-5 color palette with one accent.
- Flat-vector feel suitable for a SaaS landing page.
- Plenty of breathing room around the subject.`,
	},
	{
		Slug:      "anime-character-concept",
		Title:     "Anime Character Concept",
		Summary:   "A full character concept sheet in a modern anime style.",
		ModelSlug: "flux-dev", CategSlug: "art-illustration", Modality: "image",
		Tags:       []string{"anime", "character design", "concept art", "turnaround"},
		Popularity: 76,
		Prompt: `Design an anime character: [name, role, vibe].

Deliver in one image:
- Full-body front pose, clean line art, cel shading.
- Expressive face, distinctive silhouette and color palette.
- A small prop or detail that hints at their story.
- Soft studio background so the character pops.`,
	},

	// ---------------------- Photography ----------------------
	{
		Slug:      "product-photography-hero",
		Title:     "Product Hero Shot",
		Summary:   "A premium studio product shot ready for an ecommerce hero.",
		ModelSlug: "hidream-o1-image-dev", CategSlug: "photography", Modality: "image",
		Tags:   []string{"product photography", "studio", "ecommerce", "hero"},
		IsFree: true, Featured: true, Popularity: 86,
		Prompt: `Studio product photo of [product].

Setup:
- Seamless [color] backdrop, soft key light + subtle rim light.
- Slight reflection on a matte surface, shallow depth of field.
- Tack-sharp on the product, premium commercial look.
- 4:5 framing with space for copy on the left.`,
	},
	{
		Slug:      "editorial-portrait",
		Title:     "Editorial Portrait",
		Summary:   "A magazine-style portrait with intentional lighting and mood.",
		ModelSlug: "hidream-o1-image-dev", CategSlug: "photography", Modality: "image",
		Tags:     []string{"portrait", "editorial", "fashion", "lighting"},
		Featured: true, Popularity: 79,
		Prompt: `Editorial portrait of [subject].

- 85mm look, shallow depth of field, catchlights in the eyes.
- Lighting: [Rembrandt/soft window/dramatic single source].
- Wardrobe and palette: [describe]; mood: [confident/pensive].
- Clean background, magazine-cover quality.`,
	},

	// ---------------------- Graphic & Design ----------------------
	{
		Slug:      "packaging-design",
		Title:     "Product Packaging Design",
		Summary:   "A retail-ready packaging concept with shelf presence.",
		ModelSlug: "flux-pro", CategSlug: "graphic-design", Modality: "image",
		Tags:     []string{"packaging", "branding", "retail", "design"},
		Featured: true, Popularity: 77,
		Prompt: `Design packaging for [product] aimed at [audience].

- Strong shelf presence, clear hierarchy, legible at a glance.
- Define palette, type pairing, and one signature graphic device.
- Show front-of-pack mock; note material/finish (matte, foil, kraft).
- Keep it production-friendly (limited spot colors).`,
	},
	{
		Slug:      "social-media-template",
		Title:     "Social Post Template",
		Summary:   "A reusable, on-brand social post layout system.",
		ModelSlug: "zimage", CategSlug: "graphic-design", Modality: "image",
		Tags:   []string{"social media", "template", "layout", "branding"},
		IsFree: true, Popularity: 72,
		Prompt: `Create a social post template for [brand].

- 1:1 and 4:5 versions, consistent grid and type scale.
- Clear slots for headline, supporting line, and logo.
- On-brand palette with one accent for CTAs.
- Modern, uncluttered, thumb-stopping.`,
	},

	// ---------------------- Productivity & Writing ----------------------
	{
		Slug:      "summarize-long-doc",
		Title:     "Summarize a Long Document",
		Summary:   "Compress a long document into a structured, skimmable brief.",
		ModelSlug: "gemini-2.5-pro", CategSlug: "productivity-writing", Modality: "text",
		Tags:   []string{"summary", "tldr", "notes", "productivity"},
		IsFree: true, Featured: true, Popularity: 85,
		Prompt: `Summarize the document below.

Output:
- TL;DR (2-3 sentences).
- Key points (bulleted, grouped by theme).
- Decisions / action items with owners if present.
- Open questions or risks.

Keep it faithful; do not invent details.

Document:
` + "```\n[paste text]\n```",
	},
	{
		Slug:      "rewrite-clearer",
		Title:     "Rewrite This More Clearly",
		Summary:   "Tighten and clarify a passage while keeping the meaning and voice.",
		ModelSlug: "claude-opus-4-7", CategSlug: "productivity-writing", Modality: "text",
		Tags:   []string{"editing", "clarity", "writing", "rewrite"},
		IsFree: true, Featured: true, Popularity: 83,
		Prompt: `Rewrite the passage below to be clearer and tighter.

- Keep the original meaning and the author's voice.
- Cut filler, fix awkward phrasing, prefer plain words.
- Match this register: [formal/neutral/casual].
- Return the rewrite, then a 1-line note on what you changed.

Passage:
` + "```\n[paste text]\n```",
	},
	{
		Slug:      "meeting-notes-to-actions",
		Title:     "Notes to Action Items",
		Summary:   "Convert raw meeting notes into clear owners, actions, and deadlines.",
		ModelSlug: "openpaths/auto", CategSlug: "productivity-writing", Modality: "text",
		Tags:       []string{"meetings", "action items", "productivity", "notes"},
		Popularity: 71,
		Prompt: `Turn these raw meeting notes into a clean summary.

Output:
- Decisions made.
- Action items as: owner — action — due date (infer if implied, mark [TBD] otherwise).
- Follow-ups and parking-lot items.

Notes:
` + "```\n[paste notes]\n```",
	},

	// ---------------------- Marketing & Business ----------------------
	{
		Slug:      "cold-email-sequence",
		Title:     "Cold Email Sequence",
		Summary:   "A short, human outbound sequence that earns replies.",
		ModelSlug: "gpt-5.5", CategSlug: "marketing-business", Modality: "text",
		Tags:   []string{"email", "outbound", "sales", "copywriting"},
		IsFree: true, Featured: true, Popularity: 82,
		Prompt: `Write a 3-email cold outbound sequence.

Context: selling [product] to [persona] who cares about [pain].

Rules:
- Email 1: one specific, relevant hook + a soft ask. <90 words.
- Email 2: a proof point or angle change. <70 words.
- Email 3: a short, graceful break-up. <50 words.
- No jargon, no "just checking in", clear single CTA each.`,
	},
	{
		Slug:      "landing-page-copy",
		Title:     "Landing Page Copy",
		Summary:   "Conversion-focused hero, benefits, and CTA copy.",
		ModelSlug: "claude-opus-4-7", CategSlug: "marketing-business", Modality: "text",
		Tags:     []string{"landing page", "copywriting", "conversion", "saas"},
		Featured: true, Popularity: 80,
		Prompt: `Write landing page copy for [product].

Audience: [persona]. Core value: [outcome].

Deliver:
- 3 hero headline options + subheads.
- 3 benefit blocks (benefit-led, not feature-led).
- Social-proof line and a primary CTA + microcopy.
- Tone: confident, concrete, no hype.`,
	},
	{
		Slug:      "positioning-statement",
		Title:     "Positioning Statement",
		Summary:   "A crisp positioning statement and messaging pillars.",
		ModelSlug: "gpt-5.5", CategSlug: "marketing-business", Modality: "text",
		Tags:       []string{"positioning", "strategy", "messaging", "brand"},
		Popularity: 69,
		Prompt: `Draft positioning for [product].

Provide:
- A classic positioning statement (For [who], who [need], [product] is a [category] that [benefit], unlike [alt], because [reason]).
- 3 messaging pillars with one proof point each.
- The single sentence you'd lead with on the homepage.`,
	},

	// ---------------------- Video & Motion ----------------------
	{
		Slug:      "product-demo-video",
		Title:     "Product Demo Video Prompt",
		Summary:   "A short, punchy product demo clip with clear camera direction.",
		ModelSlug: "wan", CategSlug: "video-motion", Modality: "video",
		Tags:   []string{"product video", "demo", "motion", "camera"},
		IsFree: true, Featured: true, Popularity: 84,
		Prompt: `A clean product demo clip of [product].

- Open on a slow push-in on the product on a [surface].
- Soft studio lighting, shallow depth of field, premium feel.
- One smooth camera move (dolly or orbit), no fast cuts.
- 5 seconds, 16:9, subtle ambient audio.`,
	},
	{
		Slug:      "cinematic-broll",
		Title:     "Cinematic B-Roll",
		Summary:   "Atmospheric establishing b-roll for an intro or background.",
		ModelSlug: "wan", CategSlug: "video-motion", Modality: "video",
		Tags:     []string{"b-roll", "cinematic", "establishing", "atmosphere"},
		Featured: true, Popularity: 73,
		Prompt: `Cinematic b-roll: [scene, e.g. rain on a neon city street at night].

- Slow, steady camera move; volumetric light; shallow focus.
- Color grade: teal-and-orange, filmic contrast.
- No people unless specified; ambient soundscape.
- 16:9, smooth and loopable.`,
	},

	// ---------------------- Music & Audio ----------------------
	{
		Slug:      "lofi-study-beat",
		Title:     "Lo-fi Study Beat",
		Summary:   "A mellow lo-fi instrumental for focus and study sessions.",
		ModelSlug: "lyria-3-pro-preview", CategSlug: "music-audio", Modality: "music",
		Tags:   []string{"lofi", "instrumental", "study", "chill"},
		IsFree: true, Featured: true, Popularity: 81,
		Prompt: `A mellow lo-fi hip-hop instrumental for studying.

- ~80 BPM, warm Rhodes chords, soft vinyl crackle, gentle swing drums.
- Relaxed but motivating, no vocals.
- Smooth, loopable structure, light tape saturation.`,
	},
	{
		Slug:      "brand-jingle",
		Title:     "Brand Jingle",
		Summary:   "A short, memorable sonic logo for a brand.",
		ModelSlug: "lyria-3-pro-preview", CategSlug: "music-audio", Modality: "music",
		Tags:       []string{"jingle", "sonic logo", "branding", "audio"},
		Popularity: 68,
		Prompt: `A short brand jingle / sonic logo for [brand], which feels [adjective].

- 3-5 seconds, instantly memorable, hummable motif.
- Bright, modern instrumentation; resolves on an uplifting note.
- Clean ending suitable as an app/startup sound.`,
	},
}

// ---------------------------------------------------------------------------
// Generator: fan blueprints across models to expand the catalog.
// ---------------------------------------------------------------------------

type blueprint struct {
	title    string
	summary  string
	prompt   string
	category string
	modality string
	tags     []string
}

var imageBlueprints = []blueprint{
	{"Watercolor Portrait", "A soft, painterly watercolor portrait with delicate washes.", "A watercolor portrait of [subject], loose brushwork, soft bleeds, warm paper texture, gentle color palette.", "art-illustration", "image", []string{"watercolor", "portrait", "painterly"}},
	{"Flat Vector Mascot", "A friendly flat-vector mascot ready for branding.", "A flat-vector mascot of [character], bold shapes, limited palette, clean outlines, transparent-friendly background.", "logo-icon", "image", []string{"vector", "mascot", "flat"}},
	{"Neon Cyberpunk Scene", "A moody cyberpunk street drenched in neon.", "A cyberpunk alley at night, neon signage, rain reflections, volumetric haze, cinematic wide shot.", "art-illustration", "image", []string{"cyberpunk", "neon", "cinematic"}},
	{"Minimal Poster", "A bold, minimal typographic poster.", "A minimalist poster for [topic], strong grid, one accent color, generous negative space, Swiss style.", "graphic-design", "image", []string{"poster", "minimal", "typography"}},
	{"Food Photography", "An appetizing, well-lit food shot.", "Overhead food photo of [dish], natural light, fresh garnish, shallow depth of field, rustic surface.", "photography", "image", []string{"food", "photography", "overhead"}},
}

var textBlueprints = []blueprint{
	{"Code Review Checklist", "Run a structured review over a diff.", "Review this diff for correctness, security, and style. List issues by severity with file:line and a suggested fix.\n\n```\n[paste diff]\n```", "coding-dev", "text", []string{"code review", "diff", "quality"}},
	{"Explain It Simply", "Explain a hard concept in plain language.", "Explain [concept] simply, with one everyday analogy and one concrete example. Avoid jargon.", "productivity-writing", "text", []string{"explain", "learning", "plain language"}},
	{"Blog Post Outline", "Outline a focused, skimmable blog post.", "Outline a blog post titled [title] for [audience]: a hook, 4-6 H2 sections with one-line intents, and a CTA.", "marketing-business", "text", []string{"blog", "outline", "content"}},
	{"Optimize This Function", "Make a function faster without breaking it.", "Optimize the function below for time/space. Keep behavior identical, explain the tradeoff, and note the new complexity.\n\n```\n[paste code]\n```", "coding-dev", "text", []string{"performance", "optimization", "algorithms"}},
	{"Product Update Email", "Announce a release in a friendly, concise email.", "Write a product update email announcing [feature]. Lead with the benefit, keep it under 150 words, one clear CTA.", "marketing-business", "text", []string{"email", "release", "announcement"}},
}

var videoBlueprints = []blueprint{
	{"Logo Reveal", "A clean animated logo reveal sting.", "A short logo reveal for [brand]: elements assemble with smooth easing, soft light sweep, settle on the final mark. 4s, 16:9.", "video-motion", "video", []string{"logo reveal", "motion", "branding"}},
	{"Nature Timelapse", "A serene natural timelapse.", "A timelapse of [scene, e.g. clouds over mountains], smooth motion, golden-hour light, calm and cinematic. 16:9.", "video-motion", "video", []string{"timelapse", "nature", "cinematic"}},
}

func modelsForModality(modality string) []Model {
	var out []Model
	for _, m := range models {
		if m.Modality == modality {
			out = append(out, m)
		}
	}
	return out
}

// generateAcrossModels fans each blueprint across the models that support its
// modality, producing one prompt per (blueprint, model) pair with a stable slug.
func generateAcrossModels(existing map[string]struct{}) []definition {
	var out []definition
	groups := [][]blueprint{imageBlueprints, textBlueprints, videoBlueprints}
	for _, group := range groups {
		for _, bp := range group {
			mdls := modelsForModality(bp.modality)
			for i, m := range mdls {
				slug := slugify(bp.title) + "-" + m.Slug
				if _, ok := existing[slug]; ok {
					continue
				}
				existing[slug] = struct{}{}
				out = append(out, definition{
					Slug:       slug,
					Title:      fmt.Sprintf("%s (%s)", bp.title, m.Name),
					Summary:    bp.summary,
					Prompt:     bp.prompt,
					ModelSlug:  m.Slug,
					CategSlug:  bp.category,
					Modality:   bp.modality,
					Tags:       append(append([]string{}, bp.tags...), strings.ToLower(m.Name)),
					IsFree:     i == 0,
					Featured:   false,
					Popularity: 60 - i*3,
				})
			}
		}
	}
	return out
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
