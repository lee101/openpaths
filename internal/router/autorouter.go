package router

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"

	"github.com/openpaths/openpaths/internal/model"
	"github.com/openpaths/openpaths/internal/provider"
)

type AutoEntry struct {
	Description     string
	ModelID         string
	ReasoningEffort string
	Embedding       []float64
}

type AutoRouteResult struct {
	ModelID         string
	ReasoningEffort string
}

type AutoRouter struct {
	embedder provider.EmbeddingProvider
	tables   map[string][]AutoEntry
	mu       sync.RWMutex
	ready    bool
}

func NewAutoRouter(embedder provider.EmbeddingProvider) *AutoRouter {
	return &AutoRouter{
		embedder: embedder,
		tables:   defaultRoutingTables(),
	}
}

func (ar *AutoRouter) Init(ctx context.Context) error {
	ar.mu.Lock()
	defer ar.mu.Unlock()

	total := 0
	for _, entries := range ar.tables {
		total += len(entries)
	}

	descs := make([]string, 0, total)
	type entryRef struct {
		modality string
		index    int
	}
	refs := make([]entryRef, 0, total)
	for modality, entries := range ar.tables {
		for i, e := range entries {
			descs = append(descs, e.Description)
			refs = append(refs, entryRef{modality: modality, index: i})
		}
	}

	embeddings, err := ar.embedBatch(ctx, descs)
	if err != nil {
		return fmt.Errorf("autorouter init: %w", err)
	}

	for i, ref := range refs {
		ar.tables[ref.modality][ref.index].Embedding = embeddings[i]
	}

	ar.ready = true
	log.Printf("autorouter: initialized %d entries across %d modalities", total, len(ar.tables))
	return nil
}

func (ar *AutoRouter) ResolveAuto(ctx context.Context, modality, prompt string) (AutoRouteResult, error) {
	ar.mu.RLock()
	defer ar.mu.RUnlock()

	if !ar.ready {
		return AutoRouteResult{}, fmt.Errorf("autorouter not initialized")
	}

	entries, ok := ar.tables[modality]
	if !ok || len(entries) == 0 {
		return AutoRouteResult{}, fmt.Errorf("no auto entries for modality %q", modality)
	}

	queryEmb, err := ar.embed(ctx, prompt)
	if err != nil {
		return AutoRouteResult{}, err
	}

	bestIdx := 0
	bestSim := -1.0
	for i, entry := range entries {
		sim := cosineSimilarity(queryEmb, entry.Embedding)
		if sim > bestSim {
			bestSim = sim
			bestIdx = i
		}
	}

	e := entries[bestIdx]
	log.Printf("autorouter: %s %q -> %s reasoning=%s (sim=%.4f)", modality, truncate(prompt, 60), e.ModelID, e.ReasoningEffort, bestSim)
	return AutoRouteResult{ModelID: e.ModelID, ReasoningEffort: e.ReasoningEffort}, nil
}

func (ar *AutoRouter) embed(ctx context.Context, text string) ([]float64, error) {
	resp, err := ar.embedder.Embed(ctx, &model.EmbeddingRequest{Input: text})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}
	return resp.Data[0].Embedding, nil
}

func (ar *AutoRouter) embedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	results := make([][]float64, len(texts))
	for i, t := range texts {
		emb, err := ar.embed(ctx, t)
		if err != nil {
			return nil, fmt.Errorf("embed %q: %w", truncate(t, 40), err)
		}
		results[i] = emb
	}
	return results, nil
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

var autoModelMap = map[string]string{
	"auto-image":       "image",
	"auto-img":         "image",
	"auto-video":       "video",
	"auto-vid":         "video",
	"auto":             "text",
	"auto-text":        "text",
	"auto-chat":        "text",
	"auto-easy-task":   "easy-task",
	"auto-easy":        "easy-task",
	"auto-medium-task": "medium-task",
	"auto-medium":      "medium-task",
	"auto-think":       "think-task",
	"auto-think-task":  "think-task",
	"autothink":        "think-task",
	"auto-hard":        "hard-task",
	"auto-hard-task":   "hard-task",
	"auto-opus":        "hard-task",
}

func IsAutoModel(modelName string) (modality string, ok bool) {
	m, ok := autoModelMap[modelName]
	return m, ok
}

func IsAutoThinkModel(modelName string) bool {
	modality, ok := IsAutoModel(modelName)
	return ok && modality == "think-task"
}

func IsAutoReasoningEffort(effort string) bool {
	switch strings.ToLower(strings.TrimSpace(effort)) {
	case "auto", "automatic", "auto-think", "autothink":
		return true
	}
	return false
}

// TaskTierToModality maps the cross-provider `task_tier` hint on
// ChatCompletionRequest to the autorouter modality that should service it.
// Returns ok=false for unknown or empty tiers so callers can fall back to
// default model-name based routing.
func TaskTierToModality(tier string) (modality string, ok bool) {
	switch tier {
	case "easy":
		return "easy-task", true
	case "medium":
		return "medium-task", true
	case "think":
		return "think-task", true
	case "hard":
		return "hard-task", true
	}
	return "", false
}

func defaultRoutingTables() map[string][]AutoEntry {
	return map[string][]AutoEntry{
		"image": {
			// ra1 - default for most image gen
			{Description: "portrait headshot face character person selfie photo of someone", ModelID: "ra1"},
			{Description: "logo icon symbol badge emblem brand design graphic", ModelID: "ra1"},
			{Description: "abstract art colorful pattern gradient texture minimal", ModelID: "ra1"},
			{Description: "product photo studio lighting commercial advertisement mockup", ModelID: "ra1"},
			{Description: "simple illustration drawing sketch concept art", ModelID: "ra1"},
			{Description: "nature wildlife animal close-up macro photograph", ModelID: "ra1"},
			{Description: "food photography recipe dish meal plating restaurant", ModelID: "ra1"},
			{Description: "meme funny image joke humor reaction", ModelID: "ra1"},
			{Description: "wallpaper background desktop phone aesthetic", ModelID: "ra1"},
			{Description: "pixel art retro 8bit 16bit sprite game asset", ModelID: "ra1"},
			// zimage - anime specialist
			{Description: "anime manga japanese animation style character cel shaded", ModelID: "zimage"},
			{Description: "chibi kawaii cute anime girl boy manga panel", ModelID: "zimage"},
			// flux-pro - complex scenes
			{Description: "complex scene detailed environment photorealistic landscape city architecture multiple characters interacting crowd busy street", ModelID: "flux-pro"},
			{Description: "panoramic wide angle scenery epic vista detailed background environment world building", ModelID: "flux-pro"},
			{Description: "intricate mechanical technical diagram blueprint schematic detailed engineering", ModelID: "flux-pro"},
			{Description: "historical painting classical art renaissance baroque detailed composition museum quality", ModelID: "flux-pro"},
			{Description: "hyperrealistic photo studio quality magazine cover editorial high fashion", ModelID: "flux-pro"},
			{Description: "architectural visualization interior design room building floor plan render", ModelID: "flux-pro"},
		},
		"video": {
			{Description: "short clip simple animation basic motion loop gif small", ModelID: "ltx-video"},
			{Description: "quick preview test draft prototype low quality fast cheap", ModelID: "hailuo-2.3-fast"},
			{Description: "cinematic high quality detailed motion narrative storytelling film premium", ModelID: "ra2v"},
			{Description: "music video creative visual effects artistic stylized", ModelID: "hailuo-2.3"},
			{Description: "product demo explainer tutorial walkthrough showcase", ModelID: "hailuo-2.3"},
			{Description: "realistic footage documentary natural environment wildlife nature", ModelID: "wan"},
		},
		"text": {
			// ===== SUPER EASY - Gemini Flash Lite (cheapest, fastest, no thinking) =====
			{Description: "find information lookup search what is define meaning of", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "summarize tldr brief overview recap short summary of this", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "simple question yes no answer quick fact check trivia", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "list enumerate items categories options choices", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "format convert reformat text json yaml xml markdown html", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "spell check grammar fix proofread typo correction", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "extract email phone url date number from text parse", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "translate short phrase word sentence between languages", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "what time weather date today current status check", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},

			// ===== EASY - GPT-5.4 Nano (cheap, fast, tool support) =====
			{Description: "git commit push pull merge branch checkout status diff log", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "rename variable refactor simple cleanup lint fix small typo", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "add comment docstring documentation jsdoc typedoc godoc", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "config change environment variable setting toggle flag env", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "shell command bash script terminal automation cron job", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "install package dependency npm pip cargo go get yarn add", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "dockerfile docker compose container kubernetes yaml helm", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "write simple test add unit test basic test case assert", ModelID: "gpt-5.4-nano", ReasoningEffort: "low"},
			{Description: "create boilerplate scaffold template starter project init", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "regex pattern match replace string manipulation text processing", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "readme changelog license contributing guide project docs", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "ci cd pipeline github actions workflow deploy build", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},

			// ===== MODERATE CODING - GPT-5.4 Mini (great price/perf) + Claude Sonnet =====
			{Description: "implement feature add functionality new endpoint api route handler controller", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "debug fix bug error exception crash investigate issue trace", ModelID: "gpt-5.4-mini", ReasoningEffort: "medium"},
			{Description: "code review analyze quality security vulnerability check audit", ModelID: "claude-sonnet-4-6", ReasoningEffort: "medium"},
			{Description: "database query sql migration schema design model orm prisma", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "data analysis csv json structured query analytics pandas numpy", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "api integration third party sdk client library webhook", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "authentication authorization login signup oauth jwt session", ModelID: "claude-sonnet-4-6", ReasoningEffort: "medium"},
			{Description: "backend server middleware routing request response http rest", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "write comprehensive test suite integration test e2e coverage", ModelID: "gpt-5.4-mini", ReasoningEffort: "medium"},
			{Description: "error handling validation input sanitization edge cases", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},

			// ===== DESIGN / FRONTEND - Gemini 3.1 Pro (best at visual/design) =====
			{Description: "website design frontend ui ux layout page component react vue svelte", ModelID: "gemini-3.1-pro-preview", ReasoningEffort: "medium"},
			{Description: "game design game mechanic level design gameplay system rules", ModelID: "gemini-3.1-pro-preview", ReasoningEffort: "medium"},
			{Description: "3d object design model mesh texture render scene blender unity", ModelID: "gemini-3.1-pro-preview", ReasoningEffort: "medium"},
			{Description: "css styling animation responsive design theme color palette tailwind", ModelID: "gemini-3.1-pro-preview", ReasoningEffort: "low"},
			{Description: "mobile app design ios android interface navigation wireframe flutter", ModelID: "gemini-3.1-pro-preview", ReasoningEffort: "medium"},
			{Description: "design system component library brand guide typography tokens figma", ModelID: "gemini-3.1-pro-preview", ReasoningEffort: "medium"},
			{Description: "html template email newsletter landing page hero section form", ModelID: "gemini-3.1-pro-preview", ReasoningEffort: "low"},
			{Description: "data visualization chart graph dashboard d3 plotly recharts", ModelID: "gemini-3.1-pro-preview", ReasoningEffort: "medium"},
			{Description: "svg icon illustration vector graphic canvas webgl shader", ModelID: "gemini-3.1-pro-preview", ReasoningEffort: "low"},
			{Description: "user experience flow onboarding accessibility aria wcag a11y", ModelID: "gemini-3.1-pro-preview", ReasoningEffort: "medium"},

			// ===== ADVANCED CODING - GPT 5 Codex medium thinking =====
			{Description: "architect system design infrastructure scale distributed microservice", ModelID: "gpt-5-codex", ReasoningEffort: "medium"},
			{Description: "optimize performance profiling memory cpu bottleneck latency throughput", ModelID: "gpt-5-codex", ReasoningEffort: "medium"},
			{Description: "refactor large codebase restructure migrate rewrite legacy modernize", ModelID: "gpt-5-codex", ReasoningEffort: "medium"},
			{Description: "security audit penetration test vulnerability assessment hardening", ModelID: "gpt-5-codex", ReasoningEffort: "medium"},
			{Description: "implement complex algorithm data structure tree graph trie heap", ModelID: "gpt-5-codex", ReasoningEffort: "medium"},
			{Description: "concurrency parallel threading async await goroutine channel mutex", ModelID: "gpt-5-codex", ReasoningEffort: "medium"},
			{Description: "networking protocol tcp udp websocket grpc protobuf", ModelID: "gpt-5-codex", ReasoningEffort: "medium"},
			{Description: "full stack application end to end complete project from scratch", ModelID: "gpt-5-codex", ReasoningEffort: "medium"},

			// ===== VERY HARD / RESEARCH - GPT 5 Codex high thinking =====
			{Description: "mathematics proof theorem formula derivation calculus algebra topology", ModelID: "gpt-5-codex", ReasoningEffort: "high"},
			{Description: "research paper academic analysis deep investigation survey literature", ModelID: "gpt-5-codex", ReasoningEffort: "high"},
			{Description: "machine learning model training neural network deep learning transformer", ModelID: "gpt-5-codex", ReasoningEffort: "high"},
			{Description: "complex debugging race condition deadlock memory leak concurrency bug", ModelID: "gpt-5-codex", ReasoningEffort: "high"},
			{Description: "compiler interpreter parser language design type system ast", ModelID: "gpt-5-codex", ReasoningEffort: "high"},
			{Description: "cryptography protocol security formal verification zero knowledge proof", ModelID: "gpt-5-codex", ReasoningEffort: "high"},
			{Description: "operating system kernel driver low level systems programming", ModelID: "gpt-5-codex", ReasoningEffort: "high"},
			{Description: "quantum computing algorithm complexity theory np hard reduction", ModelID: "gpt-5-codex", ReasoningEffort: "high"},

			// ===== CREATIVE / CHAT - GPT 5 Chat Latest =====
			{Description: "creative writing story poetry fiction novel narrative worldbuilding screenplay", ModelID: "gpt-5-chat-latest", ReasoningEffort: "low"},
			{Description: "translation language multilingual foreign text localization internationalization", ModelID: "gpt-5-chat-latest", ReasoningEffort: "none"},
			{Description: "copywriting marketing ad copy slogan tagline brand voice", ModelID: "gpt-5-chat-latest", ReasoningEffort: "low"},
			{Description: "blog post article essay editorial opinion piece long form", ModelID: "gpt-5-chat-latest", ReasoningEffort: "low"},

			// ===== GENERAL - Gemini 3.1 Pro =====
			{Description: "general conversation chat casual discussion help advice recommendation", ModelID: "gemini-3.1-pro-preview", ReasoningEffort: "none"},
			{Description: "explain concept teach tutorial guide how to learn introduction", ModelID: "gemini-3.1-pro-preview", ReasoningEffort: "low"},
			{Description: "compare pros cons tradeoffs options evaluate alternatives", ModelID: "gemini-3.1-pro-preview", ReasoningEffort: "low"},
			{Description: "brainstorm ideas suggestions creative solutions approach strategy", ModelID: "gemini-3.1-pro-preview", ReasoningEffort: "low"},

			// ===== DEEP REASONING - O3 =====
			{Description: "logic puzzle complex reasoning brain teaser riddle deduction", ModelID: "o3", ReasoningEffort: "high"},
			{Description: "competitive programming contest challenge leetcode codeforces problem", ModelID: "o3", ReasoningEffort: "high"},

			// ===== SENSITIVE / POLICY-HEAVY CONTENT - DeepSeek V4 direct =====
			{Description: "adult roleplay mature romance intimate character dialogue sensitive creative writing", ModelID: "deepseek-v4-flash", ReasoningEffort: "low"},
			{Description: "fringe controversial taboo conspiracy ideology politics sensitive social content", ModelID: "deepseek-v4-flash", ReasoningEffort: "medium"},
			{Description: "harm classification safety policy boundary adult self harm extremist weapons drugs", ModelID: "deepseek-v4-pro", ReasoningEffort: "high"},
			{Description: "biosecurity biology pathogen lab protocol virology genetics medical dual use risk", ModelID: "deepseek-v4-pro", ReasoningEffort: "high"},
		},

		"easy-task": {
			// Flash Lite - cheapest for trivial tasks
			{Description: "find information lookup search what is define meaning", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "summarize tldr brief overview recap short summary", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "simple question yes no answer quick fact trivia", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "list enumerate items categories options choices", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "format convert reformat text json yaml xml markdown", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "spell check grammar fix proofread typo correction", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "extract email phone url date number from text", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "translate short phrase word sentence between languages", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			{Description: "what time weather date today current status check", ModelID: "gemini-flash-lite", ReasoningEffort: "none"},
			// GPT-5.4 Nano - fast and cheap with tool support
			{Description: "classify categorize label tag sort rank items", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "extract data parse structured output json schema", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "write simple test add unit test basic test case", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "create boilerplate scaffold template starter project", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "add comment docstring documentation jsdoc readme", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "dockerfile docker compose container kubernetes yaml", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "ci cd pipeline github actions workflow deploy build", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "git commit push pull merge branch checkout diff", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "rename variable refactor simple cleanup lint fix typo", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "config change environment variable setting toggle flag", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "shell command bash script terminal automation cron", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "install package dependency npm pip cargo go get", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "regex pattern match replace string manipulation text", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			// DeepSeek V4 Flash - cheap direct path for lightweight sensitive classifiers
			{Description: "classify sensitive content harm category adult roleplay biosecurity fringe policy", ModelID: "deepseek-v4-flash", ReasoningEffort: "none"},
			// Gemini 2.5 Flash - lightweight but stronger general replies
			{Description: "general conversation chat casual discussion help advice", ModelID: "gemini-2.5-flash", ReasoningEffort: "none"},
			{Description: "explain concept teach tutorial guide how to learn", ModelID: "gemini-2.5-flash", ReasoningEffort: "none"},
			// Claude Haiku - short polished writing and quick customer-facing edits
			{Description: "rewrite polish edit improve tone concise email message response", ModelID: "claude-haiku-4-5-20251001", ReasoningEffort: "none"},
			{Description: "customer support reply canned response acknowledgement followup", ModelID: "claude-haiku-4-5-20251001", ReasoningEffort: "none"},
		},

		"medium-task": {
			// GPT-5.4 Mini - best price/performance for medium tasks
			{Description: "implement feature add functionality new endpoint api route handler", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "debug fix bug error exception crash investigate issue", ModelID: "gpt-5.4-mini", ReasoningEffort: "medium"},
			{Description: "write simple function helper utility method snippet python javascript typescript golang reverse string", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "api integration third party sdk client library webhook", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "backend server middleware routing request response http", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "error handling validation input sanitization edge cases", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "data analysis csv json structured query analytics pandas", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "refactor codebase restructure reorganize code cleanup", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "write comprehensive test suite integration test e2e", ModelID: "gpt-5.4-mini", ReasoningEffort: "medium"},
			// Claude Sonnet - strong for security/auth/code review
			{Description: "code review analyze quality security vulnerability check", ModelID: "claude-sonnet-4-6", ReasoningEffort: "medium"},
			{Description: "database query sql migration schema design model orm", ModelID: "claude-sonnet-4-6", ReasoningEffort: "low"},
			{Description: "authentication authorization login signup oauth jwt session", ModelID: "claude-sonnet-4-6", ReasoningEffort: "medium"},
			// Gemini 2.5 Flash - fast mid-tier for frontend
			{Description: "website design frontend ui ux layout component react vue", ModelID: "gemini-2.5-flash", ReasoningEffort: "low"},
			{Description: "css styling animation responsive design theme tailwind", ModelID: "gemini-2.5-flash", ReasoningEffort: "low"},
			{Description: "html template email newsletter landing page hero section", ModelID: "gemini-2.5-flash", ReasoningEffort: "low"},
			{Description: "data visualization chart graph dashboard d3 plotly", ModelID: "gemini-2.5-flash", ReasoningEffort: "low"},
			// DeepSeek - cheap coding fallback
			{Description: "networking protocol tcp udp websocket grpc protobuf", ModelID: "deepseek-chat", ReasoningEffort: "low"},
			{Description: "adult roleplay romance mature creative writing character dialogue sensitive style", ModelID: "deepseek-v4-flash", ReasoningEffort: "low"},
			{Description: "sensitive policy handling harm classifier controversial fringe biology medical question", ModelID: "deepseek-v4-flash", ReasoningEffort: "medium"},
			// GPT-5.4 Mini - general mid-tier (replaces GPT-4o)
			{Description: "creative writing story translation copywriting blog post", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "compare pros cons tradeoffs evaluate alternatives", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "brainstorm ideas suggestions solutions approach strategy", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "general conversation explanation help recommendation", ModelID: "gpt-5.4-mini", ReasoningEffort: "none"},
		},

		"hard-task": {
			// Hard-reasoning / frontier generation tasks where a flagship model is worth it.
			// Primary: GPT-5.5 high/medium reasoning.
			{Description: "sankey diagram flow visualization network graph d3 custom force directed tree layout candlestick word cloud infographic", ModelID: "gpt-5.5", ReasoningEffort: "medium"},
			{Description: "advanced data visualization complex chart layout multi-panel dashboard composition nuanced design judgement", ModelID: "gpt-5.5", ReasoningEffort: "medium"},
			{Description: "non trivial json schema output structured generation prefill continuation careful formatting", ModelID: "gpt-5.5", ReasoningEffort: "medium"},
			{Description: "architectural decision tradeoff analysis senior engineer judgement", ModelID: "gpt-5.5", ReasoningEffort: "high"},
			{Description: "deep product reasoning subtle requirements ambiguous spec planning", ModelID: "gpt-5.5", ReasoningEffort: "high"},
			// High-effort algorithmic / math / systems.
			{Description: "formal verification compiler design cryptography research deep scientific reasoning", ModelID: "gpt-5.5", ReasoningEffort: "high"},
			{Description: "prove a theorem derive a formula or solve a hard math olympiad style problem", ModelID: "gpt-5.5", ReasoningEffort: "high"},
			{Description: "design a distributed system protocol with consensus recovery and adversarial failures", ModelID: "gpt-5.5", ReasoningEffort: "high"},
			{Description: "complex debugging race condition deadlock memory leak concurrency bug", ModelID: "gpt-5-codex", ReasoningEffort: "high"},
			{Description: "implement complex algorithm data structure tree graph trie heap", ModelID: "gpt-5-codex", ReasoningEffort: "high"},
			// DeepSeek V4 Pro - strong price/perf for sensitive and policy-heavy judgement.
			{Description: "sensitive adult roleplay fringe controversial content careful boundary judgement", ModelID: "deepseek-v4-pro", ReasoningEffort: "high"},
			{Description: "biosecurity dual use biology pathogen wet lab protocol synthesis toxin dangerous technical details", ModelID: "deepseek-v4-pro", ReasoningEffort: "high"},
		},

		"think-task": {
			// NONE - trivial lookup/format/classification requests with no deliberate thinking.
			{Description: "say yes answer yes only one word trivial acknowledgement", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "answer with one number short fact classify sentiment label tag", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "extract email phone url json field quick parse format convert", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},
			{Description: "rewrite one sentence fix grammar tiny edit concise wording", ModelID: "gpt-5.4-nano", ReasoningEffort: "none"},

			// LOW - straightforward but not purely mechanical tasks.
			{Description: "write a small helper function simple script basic unit test", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "explain a concept simply compare two options briefly", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "draft a short email response summarize and suggest next steps", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},
			{Description: "debug a small bug trace a simple error update config safely", ModelID: "gpt-5.4-mini", ReasoningEffort: "low"},

			// MEDIUM - multi-step work where deliberate reasoning is useful.
			{Description: "design an api endpoint with validation auth and edge cases", ModelID: "gpt-5.4-mini", ReasoningEffort: "medium"},
			{Description: "review code for bugs security and behavioral regressions", ModelID: "gpt-5.4-mini", ReasoningEffort: "medium"},
			{Description: "plan a refactor migration or test strategy across several files", ModelID: "gpt-5.4-mini", ReasoningEffort: "medium"},
			{Description: "analyze tradeoffs architecture performance bottlenecks and failure modes", ModelID: "gpt-5.4-mini", ReasoningEffort: "medium"},
			{Description: "classify or handle sensitive adult roleplay fringe harm policy content carefully", ModelID: "deepseek-v4-flash", ReasoningEffort: "medium"},

			// HIGH - hard algorithmic or research-grade work that should spend real reasoning budget.
			{Description: "create a 3d mesh simplification algorithm with error metrics and topology preservation", ModelID: "gpt-5.5", ReasoningEffort: "high"},
			{Description: "prove a theorem derive a formula or solve a hard math olympiad style problem", ModelID: "gpt-5.5", ReasoningEffort: "high"},
			{Description: "design a distributed system protocol with consensus recovery and adversarial failures", ModelID: "gpt-5.5", ReasoningEffort: "high"},
			{Description: "formal verification compiler design cryptography research deep scientific reasoning", ModelID: "gpt-5.5", ReasoningEffort: "high"},
			{Description: "make a 3d simulation of cogs gears clock mechanism physics animation webgl threejs", ModelID: "gpt-5.5", ReasoningEffort: "high"},
			{Description: "biosecurity dual use biology pathogen lab protocol dangerous technical policy reasoning", ModelID: "deepseek-v4-pro", ReasoningEffort: "high"},
		},
	}
}
