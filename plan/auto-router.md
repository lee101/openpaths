# Auto-Router: Embedding-Based Model Selection

## Overview
Replace static fallback chains for "auto" models with intelligent routing via cosine similarity on pre-computed embeddings. Returns both model ID and reasoning effort level.

## Architecture
- At startup: embed all descriptor strings using embedding provider, cache in memory
- At request time: embed the incoming prompt, find nearest-neighbor among descriptors for that modality
- Return matched model ID + reasoning effort, then normal router handles provider resolution

## Files Changed
- `internal/router/autorouter.go` - Core engine with routing tables
- `internal/router/router.go` - MaybeResolveAuto returns AutoRouteResult{ModelID, ReasoningEffort}
- `internal/model/chat.go` - Added ReasoningEffort field to ChatCompletionRequest
- `internal/provider/google/google.go` - ThinkingConfig with budget mapped from ReasoningEffort
- `internal/handler/chat.go` - Applies auto result (model + reasoning effort)
- `internal/handler/image.go` - Applies auto result (model)
- `internal/handler/video.go` - Applies auto result (model)
- `config.yaml` - Added gpt-5.3, gemini-flash-lite, auto-image, auto-video
- `cmd/openpaths/main.go` - Wires AutoRouter with first available embedder

## Text Routing Table (modality: "text")

### Super Easy -> gemini-flash-lite (reasoning: none)
- Info lookup, definitions, simple questions
- Summarize, TLDR, overviews
- List, enumerate, format, convert
- Spell check, grammar, extract data
- Translate short phrases

### Easy -> glm-5 (reasoning: none/low)
- Git operations (commit, push, merge, branch)
- Rename, simple refactor, lint fixes
- Comments, docstrings, readme
- Config changes, env vars, flags
- Shell scripts, Docker, CI/CD
- Install packages, boilerplate, regex

### Moderate Coding -> claude-sonnet-4-6 (reasoning: low/medium)
- Implement features, endpoints, handlers
- Debug, fix bugs, investigate issues
- Code review, security checks
- Database, SQL, migrations
- API integration, auth, backend
- Test suites, error handling

### Design / Frontend -> gemini-3.1-pro-preview (reasoning: low/medium)
- Website/app design, UI/UX, React/Vue
- Game design, game mechanics
- 3D object design, modeling
- CSS, animations, theming
- Mobile app design, wireframes
- Design systems, data viz, SVG

### Advanced Coding -> gpt-5.3 (reasoning: medium)
- System architecture, distributed systems
- Performance optimization, profiling
- Large refactors, migrations
- Security audits, complex algorithms
- Concurrency, networking, full stack

### Very Hard / Research -> gpt-5.3 (reasoning: high)
- Mathematics, proofs, theorems
- Academic research, ML/AI
- Complex debugging (race conditions, memory leaks)
- Compiler/language design, cryptography
- OS/kernel, quantum computing

### Creative -> gpt-5-chat-latest (reasoning: low)
- Creative writing, stories, poetry
- Translation, copywriting, blog posts

### General -> gemini-3.1-pro-preview (reasoning: none/low)
- Conversation, explanations, comparisons, brainstorming

### Deep Reasoning -> o3 (reasoning: high)
- Logic puzzles, competitive programming

## ReasoningEffort -> Provider Mapping
- OpenAI: passes through as `reasoning_effort` JSON field
- Google: maps to ThinkingConfig.ThinkingBudget (none=0, low=1024, medium=8192, high=32768)
