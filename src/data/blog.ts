import qwenStory from './qwenTwentySecondWindow';

export interface BlogPost {
  slug: string;
  alternativePath?: string;
  title: string;
  excerpt: string;
  date: string;
  author: string;
  readTime: string;
  tags: string[];
  content: string;
}

export const posts: BlogPost[] = [
  {
    slug: 'wan-3-text-to-video-and-image-to-video-api',
    title: 'Wan 3.0 Text-to-Video and Image-to-Video Through One API',
    excerpt: 'Wan 3.0 renders up-to-30-second clips with native audio for $0.05–$0.20 per second depending on resolution. How its text-to-video and image-to-video endpoints work through OpenPaths: inputs, smart duration, end frames, and copy-paste code.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['video generation', 'wan', 'image-to-video', 'text-to-video', 'fal'],
    content: `Wan 3.0 is Alibaba's latest video generation model family, built around three things that matter for production clips: motion smoothness, scene fidelity over long takes, and native audio. Both directions — text-to-video and image-to-video — are now live on OpenPaths, billable per second through the same \`op-\` key you already use for chat, image, and speech.

## What Wan 3.0 actually offers

- **Up to 30 seconds per take.** Pin an exact length between 2s and 30s, or leave duration on *smart* and let the model pick a length that fits the prompt.
- **Native audio.** Generated sound comes out of the same pass — no separate lipsync or foley step.
- **Optional reasoning pass.** An \`enable_thinking\` flag lets the model plan complex shots before rendering.
- **Three resolution tiers.** 480p, 720p, or 1080p, priced separately so you never pay 1080p rates for drafts.
- **First and last frame control** on the image-to-video endpoint: give it a still (optionally two) plus a motion hint.

## Pricing

Fal charges per second of output, by resolution tier. We pass the tiers straight through:

| Resolution | Price per second | 5s clip |
|---|---|---|
| 480p | $0.05 | $0.25 |
| 720p | $0.10 | $0.50 |
| 1080p | $0.20 | $1.00 |

A typical workflow bills far less than the headline: draft in 480p until composition is right, then re-render winners at 1080p. A maximum 30-second take at the highest tier is $6.00 — compare Seedance 2.5 at $0.473/s ($14.19 for the same take) or Veo 3.1 at $0.40/s ($12.00).

## Try it without writing code

Two new spaces run the endpoints directly in the browser:

- [/text-to-video](/text-to-video) — prompt, resolution, aspect ratio, duration, audio, seed, reasoning toggle, live cost estimate.
- [/image-to-video](/image-to-video) — upload or paste a start frame, add an optional end frame, describe the motion.

Every control maps 1:1 to an API field, and each page shows the exact request body as Python, JavaScript, or cURL.

## Call it from code

One endpoint handles both models. Text-to-video:

\`\`\`bash
curl "https://openpaths.io/v1/videos/generations" \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer op-..." \\
  -d '{
    "model": "wan-3.0-text-to-video",
    "prompt": "A red panda walking through a bamboo forest at sunrise",
    "resolution": "720p",
    "duration": "5",
    "aspect_ratio": "16:9",
    "generate_audio": true
  }'
\`\`\`

Image-to-video swaps the model ID and adds a start frame (an optional end frame turns it into a first→last interpolation):

\`\`\`json
{
  "model": "wan-3.0-image-to-video",
  "prompt": "Slow cinematic push-in, mist drifting between the ridges",
  "image_url": "https://example.com/first.jpg",
  "end_image_url": "https://example.com/last.jpg"
}
\`\`\`

Requests return a job id; poll \`GET /v1/videos/generations/{id}\` until \`status\` is \`completed\` and read \`video_url\`. The response includes the actual rendered duration and the seed used, so any clip is reproducible.

## Input notes worth knowing

1. **Smart duration sends null upstream**, not a guessed number — the model decides from your prompt, and you pay for what it picks.
2. **Aspect ratio \`adaptive\`** lets the model choose orientation; pin \`16:9\`, \`4:3\`, \`1:1\`, \`3:4\`, or \`9:16\` when the placement is fixed.
3. **The reasoning pass costs nothing extra** but trades latency for prompt adherence on complex choreography — off by default.
4. **Audio is on by default**; set \`generate_audio: false\` for silent b-roll.

## Where Wan 3.0 fits in the catalog

Wan 3.0 slots under Seedance 2.5 and Kling 3.0 on quality-of-motion, well above last generation's Wan 2.7, at a fraction of the per-second price. For cheap iteration loops there is no cheaper per-second rate on the [models page](/models) right now. Full parameter reference lives on the model pages: [Wan 3.0 Text to Video](/models/wan-3.0-text-to-video) and [Wan 3.0 Image to Video](/models/wan-3.0-image-to-video).

## FAQ

### How much does a Wan 3.0 video cost?

$0.05/s at 480p, $0.10/s at 720p, $0.20/s at 1080p. A 5-second 720p clip is $0.50; the maximum 30-second 1080p take is $6.00.

### What is smart duration?

Leave duration as "auto" and Wan 3.0 chooses a length between 2 and 30 seconds based on the prompt and reference media. You are billed for the seconds it actually renders.

### Does Wan 3.0 generate audio?

Yes — speech-free ambient audio and sound effects come natively in the same render pass, controlled by one boolean.

### Can I animate a photo?

That is the image-to-video endpoint: supply \`start_image_url\` (upload, paste a URL, or drop a file in the space page), optionally \`end_image_url\`, and a short motion prompt.

### Which SDK works?

Any OpenAI-compatible client — point it at \`https://openpaths.io/v1\`. The video jobs API is plain REST, so curl and fetch work equally well.`,
  },
  {
    slug: 'best-ai-api-for-coding-agents',
    title: 'The Best AI API for Coding Agents in 2026',
    excerpt: 'What makes the best AI API for coding agents in 2026: reliable tool calling, long context, low time-to-first-token, cost per agent-hour, and automatic failover — plus how routed APIs beat any single vendor.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '7 min',
    tags: ['coding agents', 'llm api', 'model routing', 'auto-code', 'openai-compatible'],
    content: `A coding agent is not a chatbot. It burns tokens in loops: plan, call a tool, read the result, patch, run tests, repeat. That workload punishes APIs differently than chat does, which changes what the best AI API for coding agents means in 2026.

## What coding agents need from an API

Five things matter:

1. **Reliable tool calling.** One malformed function call derails a 50-step loop; 139 catalog models declare \`supports_tools: true\`, and declared support is not consistent support.
2. **Long context.** Repo maps, file contents, test output — what matters is quality when the window is full.
3. **Low latency to first token.** Agent loops serialize on the model; idle seconds per step compound across 50 steps.
4. **Cost per agent-hour.** A single task can move millions of tokens — at GPT-5.5's $5.00 per 1M input, an afternoon of runs is real money.
5. **Automatic failover.** A single-vendor agent stops dead when its one upstream wobbles.

## Why single-vendor caps your ceiling

One vendor means inheriting their outage calendar and locking in one model's skill profile. Our creative-coding scorecard shows the split: Claude Opus 4.8 scores 4.8/5 on visual output where GPT-5.5 direct manages 4.1, while GPT-5.5 wins code correctness (4.7 vs 4.4) and discipline (4.8 vs 4.5). No vendor leads everywhere; an agent that can't switch mid-task leaves accuracy and money on the table.

## Routing beats hand-picking

Our learning-to-route benchmark ran 27 real coding tasks ([whitepaper](https://huggingface.co/openpaths/learning-to-route)):

- Hand-picked GPT-5.5: 77.8% at $2.55/run
- GPT-5.4-nano: same 77.8% at $0.03 (85x cheaper)
- Best single model (GPT-5.4-mini): 85.2% at $0.43
- Cascade routing: **100% at $0.11 average**, about 4% of frontier cost

Routing beat every hand-picked model on both axes at once. Details in our [learning-to-route whitepaper breakdown](/blog/learning-to-route-whitepaper).

## Direct vs routed prices

Published rates, August 2026:

| Provider / route | Input per 1M | Notes |
|---|---|---|
| OpenAI GPT-5.5 | $5.00 | Output typically 3–5x input cost everywhere |
| OpenAI GPT-5.6 Luna | $0.20 | Ultra-cheap fast tier |
| Anthropic Opus tier | $5.00 / $25.00 out | Premium quality, premium bill |
| Anthropic Sonnet class | ~$2.00 | Solid mid-tier workhorse |
| Gemini 2.5 Pro class | ~$1.25 | Genuine free tier for prototyping |
| DeepSeek V4 Flash | $0.14 | Half price off-peak; peak latency can wobble |
| OpenPaths \`auto-code\` | routed | GLM-5.3 Flash carries the everyday paid tier |
| OpenPaths cascade routing | $0.11 avg/run | 100% accuracy, 27-task benchmark |

## The paid successor: GLM-5.3 Flash

Ox Alpha was revealed as GLM-5.3 Flash and the free preview has ended. \`glm-5.3-flash\` now answers the everyday auto-code tier at its published paid token rate; the old \`openpaths/stealth/ox-alpha\` and \`ox-alpha\` IDs remain aliases for compatibility. Hard domains still escalate automatically.

## Two lines to switch

Change \`base_url\` to \`https://openpaths.io/v1\`, swap in \`OPENPATHS_API_KEY\`. Anthropic-native \`/v1/messages\` works too — see [migrating OpenAI and Anthropic agent SDKs to OpenPaths](/blog/migrate-openai-anthropic-agent-sdks-to-openpaths). Credits start at $5; pay-per-token, no subscription.

## Research on the same key

Agents also need repo-adjacent research: which paper introduced a method, does a dataset exist, where is reference code. Papers search runs off the same key: \`POST /v1/search\` with \`provider: "papers"\` and \`format: "markdown"\` returns compact results over papers, methods, datasets, and GitHub code at $0.001 per search.

## How to choose

Start with \`auto-code\` for coding-heavy workloads — paid GLM-5.3 Flash for everyday work, escalation only where needed. Use \`auto-medium-task\` for mixed agent work: triage, summaries, moderate reasoning. Pin \`auto-think\` for hard reasoning steps. Mechanics in [how OpenPaths auto models work](/blog/how-auto-models-work); model-by-model comparison in [the best model for coding](/blog/best-model-for-coding). Latency probes are public at [/stats](/stats), every price at [/models](/models).

## The best AI API for coding agents is a routing layer

Not a model, it is a routing layer: tool calling across 342 models from 57 providers, automatic failover, and per-task economics that beat hand-picking by an order of magnitude — 100% at $0.11 versus 77.8% at $2.55. Two lines of code to try it.

## FAQ

### What matters most when choosing an API for coding agents?

Reliable tool calling, coherent long context, low latency to first token, true cost per agent-hour, and automatic failover. Token price alone misleads — a cheap model that fumbles tool calls costs more in retries.

### Is a routed API cheaper than going direct?

Cascade routing hit 100% at $0.11 per run; hand-picked GPT-5.5 managed 77.8% at $2.55, and GPT-5.4-nano matched that accuracy at $0.03. Direct suits one known-good model; routing wins when task difficulty varies.

### What happened to ox-alpha?

Ox Alpha was the free preview name for GLM-5.3 Flash. The old IDs now resolve to the live paid \`glm-5.3-flash\` route, so existing code keeps working but usage is no longer free.

### Do I need to rewrite my agent?

No. Both \`/v1/chat/completions\` and \`/v1/messages\` are served; OpenAI and Anthropic SDK agents migrate by changing base URL and key, and LangChain, Vercel AI SDK, PydanticAI, and Mastra work unchanged.

### Can my agent do research through the same API?

Yes. Papers search accepts \`POST /v1/search\` with \`provider: "papers"\`, returns markdown results over papers, methods, datasets, and GitHub code at $0.001 per search, billed to the same credits.`,
  },
  {
    slug: 'llm-router-for-ai-agents',
    title: 'LLM Router for AI Agents: Route Every Agent Step to the Right Model',
    excerpt: 'An LLM router for AI agents sends each step of a run — tool-arg extraction, implementation, planning, image generation, research — to the model that handles it best, instead of pinning one model that is always wrong somewhere.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['llm router', 'ai agents', 'model routing', 'cost optimization', 'openai compatible'],
    content: `A single agent run is not one workload. It extracts tool arguments, drafts implementation code, weighs tradeoffs in planning, maybe generates an image or searches papers. Pinning the loop to one frontier model means paying reasoning-grade prices for calls that amount to "extract two strings from this JSON." Pinning it to a cheap model means your hardest step fails at exactly the wrong moment. An LLM router for AI agents fixes this by making the routing decision per step, inside the same API call.

## Why one pinned model is always wrong

We measured this on our [learning-to-route](https://huggingface.co/openpaths/learning-to-route) benchmark: 27 coding tasks spanning trivial to genuinely hard. Hand-picking GPT-5.5 for everything scored 77.8% accuracy at $2.55 per run. GPT-5.4-nano scored the identical 77.8% at $0.03 — 85x cheaper — because the expensive model was mostly wasted on this mix. The best single model was GPT-5.4-mini at 85.2% and $0.43. Embedding-based cascade routing beat them all: 100% accuracy at $0.11 average, roughly 4% of frontier cost.

The lesson generalizes to agent loops: no single model wins every step, so stop asking it to.

## Three routing patterns, explained plainly

**Embedding-similarity task classification.** Every request is embedded and compared against embeddings of known task types — easy extraction, medium implementation, hard reasoning, code. The closest match picks the tier. This powers OpenPaths' auto routes (\`auto-easy-task\`, \`auto-medium-task\`, \`auto-hard-task\`, \`auto-think\`, \`auto-code\`): a similarity lookup with a routing table behind it, no classifier to train.

**Cost-tiered cascade.** Within a route, cheap models try first. On low-confidence output or errors, the request escalates up the tier instead of failing. That is where the $0.11 average comes from — most steps never leave the bottom tier, but the ones needing muscle get it.

**Circuit breakers with health probes.** Models wobble: provider outages, peak-hour latency drift. The router continuously probes model health; when a model degrades, its breaker opens and traffic reroutes to healthy fallbacks. Your agent loop never sees the retry.

## Mapping agent steps to routes

| Agent step | Route | Why |
|---|---|---|
| Tool-arg extraction, formatting | \`auto-easy-task\` | Trivial calls; nano-class quality is enough |
| Implementation, multi-file edits | \`auto-code\` / \`auto-medium-task\` | Coding-specialized beats general frontier |
| Planning, tradeoffs, review | \`auto-think\` | Reasoning effort pinned high where it pays |
| Image generation | \`ra1\` ($0.04/image) or Z-Image Turbo | Flagship art vs. fast anime-style |
| Video generation | \`kfold-video\` | Cinematic clips with duration/steps/audio controls |
| Research lookup | \`POST /v1/search\` with \`provider: "papers"\` | Papers, methods, datasets, GitHub code at $0.001/search |

In practice you rarely hand-assign these: point your client at the auto routes and override only for media steps where you care about output style.

## One key, no SDK juggling

Text routes, images, video, and paper search all live under one base path and one \`OPENPATHS_API_KEY\`. Your loop makes every call against a single OpenAI-shaped client:

\`\`\`python
from openai import OpenAI

client = OpenAI(
    base_url="https://openpaths.io/v1",
    api_key=os.environ["OPENPATHS_API_KEY"],
)
\`\`\`

Implementation steps go through \`client.chat.completions.create(model="auto-code", ...)\`; images hit \`/v1/images/generations\`; video hits \`/v1/videos/generations\`; research hits \`/v1/search\`. Compare that with wiring a separate SDK and credential per modality — four clients, four failure modes inside one agent run. Migrating an existing loop is two lines (base URL plus key); see our [SDK integrations](/blog/openpaths-sdk-integrations) and how the [OpenAI-compatible endpoint](/blog/use-openpaths-openai-compatible-router-anywhere) works anywhere the protocol is spoken, including Anthropic-native \`/v1/messages\`.

## How this compares to rolling your own

[OpenRouter](/blog/multi-provider-llm-api) gives you a catalog and prepaid credits but no routing intelligence. LiteLLM ships router and fallback primitives you self-host and operate. Portkey focuses on observability and guardrails. OpenPaths ships the classifier, cascades, and circuit breakers as routes you call — see [how auto models work](/blog/how-auto-models-work) and how [compound models](/blog/building-compound-models) compose cheaper specialists into one endpoint.

## Bottom line

Agents fail when you treat every step as equally hard. Route easy steps cheap, hard steps smart, media steps to purpose-built generators, research one search call away — all behind one key. On our benchmark that combination scored 100% at about 4% of frontier cost. There is no single right model for an agent run; only the right model per step.

## FAQ

### Does routing add latency my agent will notice?

Classification is an embedding lookup, not another LLM call — milliseconds. Circuit breakers remove latency spikes by steering away from degraded providers before your request lands there. Easy tasks finish faster instead of queuing behind frontier inference.

### Can I force a specific model for a specific step?

Yes. Auto routes are defaults, not locks. Pass any concrete model from the [catalog](/models) and it goes there directly — useful for media steps like \`kfold-video\`, or A/B testing a pinned frontier model against routed results.

### What happens when the routed model fails mid-run?

Model-level fallbacks fire before the error reaches your loop. Each model has a circuit breaker with health probes; an open breaker sends the request down the escalation chain — cheap tier first, then stronger models.

### Do image and video calls use the same API key as chat?

Yes. One \`OPENPATHS_API_KEY\` covers text, embeddings, image generation, video generation, and paper search. Billing is prepaid credits from $5, pay-per-token (or per image, per search), no subscription.

### Which agent frameworks does this work with?

Anything speaking the OpenAI protocol: OpenAI Agents SDK, Anthropic Agent SDK, LangChain, Vercel AI SDK, PydanticAI, Mastra, LiveKit, Hermes Agent, OpenClaw, and plain HTTP. Change the base URL and set the key.`,
  },
  {
    slug: 'cheapest-gpt-api-alternative',
    title: 'The Cheapest GPT API Alternative: Same Models, Smaller Bills',
    excerpt: 'Looking for the cheapest GPT API alternative? OpenPaths cuts your GPT bill with tier drops, smart routing, and off-peak scheduling — same models, smaller invoices.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['gpt', 'pricing', 'routing', 'cost-optimization'],
    content: `If you are shopping for the cheapest GPT API alternative, the honest answer is that you often do not need to switch models at all — you need to stop paying frontier prices for work a smaller model handles fine. We run OpenPaths, an OpenAI-compatible router over 342 models from 57 providers, and our benchmarks show the same task can cost anywhere from $2.55 per run to $0.03 depending on one decision. Here are five cost levers, ranked by effort.

## Lever 1: Drop a tier for easy traffic

Most production traffic is not hard. Classifiers, rewrites, extraction, short summaries — these do not need a flagship. The easiest swap right now is GPT-5.6 Luna at $0.20 input / $1.20 output per 1M tokens, against roughly $5.00 input for GPT-5.5 class models.

We proved the ceiling of this lever in our [learning-to-route benchmark](/blog/learning-to-route-whitepaper): across a 27-task coding suite, hand-picked GPT-5.5 scored 77.8% accuracy at $2.55 per run. GPT-5.4-nano scored an identical 77.8% at $0.03 — same accuracy, 85x cheaper.

## Lever 2: Route instead of pinning

Tier-dropping makes you guess which requests are easy. Routing removes the guess: point easy work at \`auto-easy-task\` and classifiers, rewrites, and extraction go to the cheapest capable model automatically, while \`auto-medium-task\`, \`auto-hard-task\`, and \`auto-code\` handle the rest with circuit-breaker fallbacks.

Same whitepaper's payoff number: embedding-based cascade routing hit 100% accuracy at an average of $0.11 per run — about 4% of frontier cost, and better than pinning any single model including the best we tested (GPT-5.4-mini at 85.2%).

## Lever 3: Shift flexible work off-peak

Some providers discount by time of day. DeepSeek V4 Flash lists at $0.14 input per 1M tokens, and off-peak hours are half price. Batch jobs, nightly evaluations, embedding backfills — all move off-peak unnoticed. Our [DeepSeek peak/off-peak pricing map](/blog/deepseek-peak-off-peak-pricing-map) shows which hours qualify and where peak latency wobbles.

## Lever 4: Reprice preview routes when they graduate

Ox Alpha demonstrated why preview pricing must not become a permanent budget assumption. It was revealed as GLM-5.3 Flash and now uses the model's paid rate. OpenPaths kept the old aliases working while changing their billing and upstream route together.

Its upstream requires reasoning enabled, so the router now uses high rather than automatically spending the paid max-reasoning tier.

## Lever 5: Keep frontier only where it wins

Cheap tiers fail predictably on GLSL/HLSL shaders, trading systems, CUDA kernels and LLM infra, distributed architecture, compilers, and agentic multi-file patches — those escalate past the free lane for good reason. The discipline is paying $5.00-per-1M prices only for the slice that needs them. Our creative-coding scorecards show the pattern: GPT-5.5 direct leads on code quality (4.7) and discipline (4.8), which is exactly where you keep spending.

## Lever-vs-saving summary

| Lever | Effort | What moves | Measured saving |
|---|---|---|---|
| Drop a tier for easy traffic | Low | Pinned GPT-5.5 run to GPT-5.4-nano | $2.55 to $0.03 per run (85x less) |
| Route instead of pinning | Low | Hand-picked model to cascade router | $2.55 to $0.11 avg (~96% cut, higher accuracy) |
| Off-peak scheduling | Low | DeepSeek V4 Flash daytime batch | $0.14 to half price per 1M input |
| Audit graduated previews | Low | Ox aliases onto paid GLM-5.3 Flash | Avoid unbilled paid upstream usage |
| Keep frontier where it wins | Judgment | Frontier spend concentrated on hard tasks | Pay $5.00/1M only where it scores |

## Bottom line

Start at the top of the table: tier-dropping and routing take minutes and cover most of the savings; scheduling and free lanes stack on top; then audit what still hits frontier models and confirm each domain earns it.

The cheapest GPT API alternative is not a different vendor with a bigger discount — it is a router that stops sending easy work to expensive models. You keep the same GPT models where they matter and pay pennies everywhere else.

Switching takes two lines:

\`\`\`bash
export OPENAI_BASE_URL="https://openpaths.io/v1"
export OPENAI_API_KEY="$OPENPATHS_API_KEY"
\`\`\`

Full setup details in [switch to OpenPaths in 2 lines](/blog/switch-to-openpaths-in-2-lines). Every price is public at [/models](/models), live latency probes at [/stats](/stats).

## FAQ

### Is a cheaper alternative really as accurate as GPT?

On our 27-task coding benchmark, GPT-5.4-nano matched hand-picked GPT-5.5 exactly — 77.8% vs 77.8% — while costing 85x less. Accuracy depends on your task mix, which is why routing by difficulty beats blanket downgrades.

### Do I have to change my code or SDK?

No. OpenPaths is OpenAI-compatible at \`/v1\` and also supports Anthropic-native \`/v1/messages\`. Existing OpenAI SDKs, Agents SDK, LangChain, Vercel AI SDK, and PydanticAI integrations work after changing base URL and key.

### What does routing cost compared to picking a small model myself?

Routing averaged $0.11 per run versus $0.43 for the best single self-picked model — with higher accuracy (100% vs 85.2%). It wins when traffic mixes easy and hard tasks.

### What happened when the free ox-alpha preview ended?

The Ox aliases were moved onto paid GLM-5.3 Flash instead of being allowed to fail or silently consume an unbilled paid upstream. Existing model IDs still work, now at the published GLM rate.
`,
  },
  {
    slug: 'claude-api-fallback',
    title: 'Claude API Fallback: Automatic Backup Routes When Anthropic Degrades',
    excerpt: 'Claude API fallback planning matters because the model your long coding sessions depend on most is also the one that 429s at peak. Here\'s how router-level circuit breakers beat hand-rolled retries.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['claude', 'fallback', 'anthropic', 'routing'],
    content: `Claude has a habit of being the model you don't want to swap out mid-project. On our animated-SVG scorecard, Opus 4.8 scored 4.8 on visual quality with 4.4 on code and 4.5 on instruction discipline - the best eye in the lineup, and consistent enough to hold a long coding session together. That consistency is exactly why a Claude API fallback plan matters: the more your workflow depends on Claude, the more a 429 at 2pm costs you. This post covers how Claude fails, what hand-rolled fallback looks like, and how OpenPaths makes it a config change instead of an engineering project.

## Why Claude traffic deserves protection

Long sessions are where Claude earns its premium: coherent across hundreds of turns, structurally disciplined, front-end work needing the least touch-up of anything we tested. If you're paying Opus-tier rates ($5.00 input / $25.00 output per 1M tokens), you're paying for that reliability - so when Anthropic degrades, silently rerouting to whatever's cheapest is not always right either. A good fallback preserves the *behavior class*: strong reasoning, strong instruction-following, not just any warm body with an API.

## How Claude actually fails

Three failure modes show up in practice:

- **429 rate limits at peak.** Org-level token buckets exhaust during team hours, especially on shared keys. Retrying immediately usually makes it worse.
- **529 overloaded errors.** Anthropic's capacity signal. It means the model is there but saturated; backoff helps, but during sustained peaks you want another route entirely.
- **Regional outages.** Full availability loss in your region while status pages stay green elsewhere. No retry policy fixes this; only a second provider does.

## The DIY fallback pattern

If you're calling Anthropic directly, the minimum viable version looks like this:

\`\`\`python
FALLBACKS = ["claude-opus-4.8", "gpt-5.5", "gemini-2.5-pro"]

def complete(messages):
    for i, model in enumerate(FALLBACKS):
        try:
            return call_provider(model, messages)
        except APIError as e:
            if e.status not in (429, 529) and e.status < 500:
                raise          # prompt problem, not provider problem
            if e.status == 429:
                time.sleep(2 ** i)   # back off before next attempt
    raise RuntimeError("all providers down")
\`\`\`

The subtle part isn't the retry loop - it's state. Fall back mid-stream and you must dedupe partial output so the client never sees half a Claude answer stapled to a GPT answer. Add per-provider timeouts, jittered backoff, and session pinning so one conversation doesn't ping-pong between models with different habits. Doable, but now you own a mini control plane.

## Router-level circuit breakers do this for you

OpenPaths handles the same failure modes with model-level circuit breakers: repeated 429/529/5xx failures trip a route, traffic shifts down your chain, and background probes rejoin the primary once Anthropic recovers. No code changes, no stream stitching, no dedupe. Because OpenPaths speaks native Anthropic \`/v1/messages\` *and* \`/v1/chat/completions\`, your existing Anthropic SDK calls keep working unchanged - same wire format, different base URL. See [how auto models work](/blog/how-auto-models-work) under load, plus our write-up on the [Anthropic API alternative with Claude and multi-model fallbacks](/blog/anthropic-api-alternative-with-claude-and-multi-model-fallbacks).

A sensible ladder for Claude-primary workloads:

| Position | Model | Input $/1M | Role |
|---|---|---|---|
| Primary | Claude Opus tier | $5.00 | Best long-session coherence |
| Fallback 1 | GPT-5.5 | $5.00 | Strongest reasoning peer |
| Fallback 2 | Gemini 2.5 Pro class | ~$1.25 | Cheap capable middle |
| Last resort | DeepSeek V4 Flash | $0.14 | Keeps jobs moving |

GPT-5.5 is the natural first hop: on our scorecard it out-coded Opus (4.7 vs 4.4) with better discipline, trading some visual polish. DeepSeek Flash won't match Claude's taste, but at $0.14/1M - half price off-peak - it keeps batch jobs alive during an outage. And with BYOK, existing Anthropic keys keep billing Claude usage while OpenPaths manages routing.

## Bottom line

Claude is worth protecting precisely because it's good; the scorecard above is why teams anchor on it. But anchoring without a Claude API fallback makes your most expensive dependency your least redundant. Point your existing Anthropic client at OpenPaths, pick a ladder like the one above, and let circuit breakers absorb the 429s and 529s. For the OpenAI-side mirror see our [OpenAI API fallback](/blog/openai-api-fallback) guide; for full 2026 pricing across all four tiers, see [the best LLM APIs of 2026 compared](/blog/best-llm-api-2026-compared).

## FAQ

### Does changing the base URL break the Anthropic SDK?

No. OpenPaths implements native \`/v1/messages\`, so \`anthropic\` libraries work by swapping base_url and key - two lines, as covered in our [Anthropic provider guide](/blog/provider-anthropic). Chat-completions callers can point \`/v1/chat/completions\` at the same account.

### When should fallback trigger?

On 429, 529, and 5xx-class errors - provider-side problems. Prompt errors (400s) fail fast instead; rerouting a malformed request just moves the bug.

### Will fallback models produce identical output?

No, and pretending otherwise hurts quality. GPT-5.5 is the closest behavioral peer for coding; DeepSeek Flash is a cost-preserving last resort. Pin sensitive sessions to the primary and let background jobs float.

### Can I still pay Anthropic directly?

Yes - bring your own key and Claude usage bills against your Anthropic account while OpenPaths handles failover and non-Claude routes.

### What does the fallback cost?

Nothing extra beyond token prices. Every rate is listed on the models page and live latency is public on the stats page.`,
  },
  {
    slug: 'openai-api-fallback',
    title: 'OpenAI API Fallback: Keep GPT Traffic Flowing Through Outages',
    excerpt: 'A single OpenAI API key is a silent single point of failure. Here is how an OpenAI API fallback chain keeps GPT traffic flowing when a provider or endpoint goes unhealthy.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['openai', 'fallback', 'reliability', 'routing', 'gpt'],
    content: `Every GPT app built directly against one endpoint has the same failure mode: it works perfectly right up until it doesn't. A degraded upstream, a regional incident, an unhealthy image endpoint at 2am — and your product returns errors while you sleep. An OpenAI API fallback turns that silent single point of failure into an automatic reroute. We run this in production ourselves, so this post is less theory and more incident report from our own stack.

## The single-provider problem

If your app calls one model through one path, availability equals that path's availability. Retries only help with transient blips; when an upstream stays unhealthy for minutes or hours, retry loops burn latency and money while every request fails. The fix is not more retries — it is a second route for the same capability plus something that decides when to take it.

That decision layer is a circuit breaker per model and per host: probe health, trip on repeated failures, redirect traffic, quietly test the primary until it recovers.

## Our own example: GPT Image 2

We serve [GPT Image 2](/blog/gpt-image-2-on-openpaths) through two independent paths: direct from OpenAI, and Fal-hosted. If the direct GPT Image 2 path goes unhealthy, our model-level circuit breaker fails over to the Fal-hosted variant automatically. Same model, same API surface, same prompts — users never see the outage. That is the property we want for every model we expose, not a special case hand-wired for images.

## The chat ladder

For chat and code traffic, fallback is a ladder rather than a mirror. The former \`openpaths/stealth/ox-alpha\` lane now resolves to paid GLM-5.3 Flash before \`auto-code\` walks the rest of its chain:

1. **GPT-5.5** — frontier quality, strongest on hard reasoning.
2. **Codex / Gemini** — capable second tier; Gemini 2.5 Pro class sits around $1.25 input per 1M tokens.
3. **Claude** — the [Claude fallback rung](/blog/claude-api-fallback), strong where discipline matters.
4. **DeepSeek Flash** — $0.14 input per 1M tokens, half price off-peak; a cheap floor that keeps requests succeeding.

An outage on any rung degrades you one step, not to zero. If your goal is cost-first rather than resilience-first, see our guide to the [cheapest GPT API alternative](/blog/cheapest-gpt-api-alternative).

## Watch degradation yourself

Health probes are public at [/stats](/stats) — per-model latency and availability, updated continuously. During a real incident you can watch the breaker trip and failover happen from the outside. Every price across all 342 models and 57 providers is listed at [/models](/models), so you can price each rung of your ladder before you need it.

## DIY vs declarative

The naive version looks like:

\`\`\`python
def complete(messages):
    for model in ["gpt-5.5", "gemini-2.5-pro", "claude-opus"]:
        try:
            return call(model, messages)
        except ProviderError:
            continue
    raise AllProvidersDown
\`\`\`

This works for a demo and falls apart in production. Real fallbacks need failure counting over time (not per-request), half-open probes to detect recovery without stampeding a sick upstream, per-model timeouts, and state that survives restarts. That is a distributed-systems side quest, not a feature.

The declarative version moves the ladder into routing config instead of your code. On OpenPaths, routes like \`auto-medium-task\` and \`auto-code\` already carry model-level circuit-breaker fallbacks — selection by embedding similarity to task type, with automatic reroute when a model goes unhealthy. Your application code calls one endpoint and never learns about the outage. For the full multi-provider picture, see [OpenAI API alternative for multi-provider AI apps](/blog/openai-api-alternative-for-multi-provider-ai-apps) and how to use our [OpenAI-compatible router anywhere](/blog/use-openpaths-openai-compatible-router-anywhere) — switching is two lines: base URL plus key.

| Approach | Sustained outages | Recovery detection | Ops burden |
|---|---|---|---|
| Single provider + retries | No | N/A | None |
| DIY try/except chain | Partially | Manual | High |
| Declarative routing (OpenPaths) | Yes | Automatic, per-model | Two-line switch |

## Bottom line

Mirror critical models across paths, put a circuit breaker in front, and publish health data so failures are observable. For the broader catalog story beyond OpenAI specifically, see the [multi-provider LLM API overview](/blog/multi-provider-llm-api) or the general [OpenAI provider deep dive](/blog/provider-openai); balancing across healthy endpoints is covered in [LLM load balancing](/blog/llm-load-balancing).

## FAQ

### What is an OpenAI API fallback?

A secondary route for GPT traffic that takes over automatically when the primary endpoint fails health checks. It can be the same model hosted elsewhere (GPT Image 2 direct vs Fal-hosted) or a different model entirely (GPT-5.5 falling back to Claude, then DeepSeek Flash).

### Does fallback mean lower-quality responses?

Only if your ladder skips quality tiers carelessly. Fall back within the same capability class first (same model, different host), then step down tiers. During an outage, a slightly different answer beats no answer.

### How fast does failover happen?

The breaker trips after repeated failures rather than a single error, then redirects new traffic immediately and probes the primary until it recovers. Watch per-model health at [/stats](/stats) during a real event.

### Can I keep using my existing OpenAI SDK code?

Yes. OpenPaths exposes an OpenAI-compatible \`/v1\` API, so switching means changing the base URL and key — two lines. Fallbacks then live in the router, not your code.

### Is fallback the same as load balancing?

No. Load balancing spreads traffic across healthy endpoints all the time; fallback activates only when something is unhealthy. Production systems usually want both.`,
  },
  {
    slug: 'multi-provider-llm-api',
    title: 'Multi-Provider LLM API: One Key, One Balance, Every Major Model',
    excerpt: 'A multi-provider LLM API gives you every major model behind one key and one balance — here is what that actually buys you in production.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['multi-provider', 'routing', 'api', 'openai-compatible'],
    content: `A multi-provider LLM API is a single API surface that fronts models from many providers at once: one key, one prepaid balance, one request format — and which provider serves your request becomes a routing decision instead of a contract. OpenPaths is exactly that: an OpenAI-compatible router at openpaths.io with \`OPENPATHS_API_KEY\`, serving 342 models from 57 providers behind one endpoint. Switching existing code over is changing the base URL and key: two lines.

That definition sounds like convenience. In practice it changes four things about how you ship AI features.

## Reason 1: best quality-per-task through routing

No single model wins every task, and paying frontier rates for easy tasks is how teams overspend. A multi-provider API lets the router pick per request instead of you hardcoding a model per feature.

We measured this on our 27-task coding benchmark: hand-picked GPT-5.5 got 77.8% accuracy at $2.55 per run; embedding-based cascade routing hit 100% at $0.11 average — roughly 4% of frontier cost. The method is in the [learning-to-route whitepaper](/blog/learning-to-route-whitepaper) and powers \`auto-easy-task\` through \`auto-hard-task\`, \`auto-think\`, and \`auto-code\`, each with model-level circuit-breaker fallbacks.

## Reason 2: redundancy

Single-provider apps inherit single-provider incidents: rate limits, capacity dips, outages — all your pager. Behind a multi-provider API, a failing route trips a circuit breaker and traffic moves to the next model automatically. We publish latency probes at [/stats](/stats), so redundancy is observable, not promised.

## Reason 3: one budget

Five providers means five dashboards, five invoices, five sets of minimum top-ups, and no single view of spend. Prepaid credits start at $5 (Stripe or crypto), pay-per-token, no subscription — fund one balance and every model draws from it. Every price is published at [/models](/models), including paid GLM-5.3 Flash and its retired Ox aliases.

## Reason 4: no lock-in

The quiet cost of a single provider: leaving later means rewriting prompts against a different API shape, re-testing tool calling, migrating billing. With one routed key, trying a competitor model is a model-name change; leaving is the same two lines it took to arrive. That symmetry keeps leverage with you.

## Beyond chat: one key for the whole modal stack

Chat is table stakes. The same \`OPENPATHS_API_KEY\` also covers:

- **Embeddings** — Text-Generator.io plus a local embedding model.
- **Images** — first-party Netwrck RA1 ($0.04/image, policy-flexible when other providers refuse) and ZImage anime art ($0.007/image); CuteDSL's Z-Image Turbo via \`cutedsl-image\` at $0.04/image, 512x512 up to 1360x768.
- **Video** — Netwrck \`ra2v\` and ManifoldGen's \`kfold-video\` cinematic generator with aspect ratio, duration, steps, and audio controls.
- **Music/TTS**, **image-to-3D**, and **research search** — papers/methods/datasets/GitHub code via \`POST /v1/search\` with \`provider: "papers"\` at $0.001 per search (\`format: "markdown"\` for agent-friendly results).

We operate Netwrck, CuteDSL, ManifoldGen, and Papers ourselves, so those routes get tighter pricing and deterministic capacity — your image or video job is not queued behind another company's burst traffic. See [our Netwrck provider post](/blog/provider-netwrck) for how that lane works end to end.

## Stitching N SDKs vs one routed key

| | DIY multi-SDK stack | One routed key |
|---|---|---|
| Integration | One SDK + prompt quirks per provider | One OpenAI-compatible client |
| Billing | N invoices, N balances | One prepaid balance from $5 |
| Failover | Hand-rolled retry/fallback code | Circuit-breaker fallbacks in routes |
| Cost control | Per-provider caps you build | Auto-routing down the price curve |
| New model | New integration project | Change the model name |
| Migration risk | Rewrites on every switch | Two-line base URL + key swap |

Already write OpenAI-shaped code? Nothing else changes — see [how to point any OpenAI-compatible tool at OpenPaths](/blog/use-openpaths-openai-compatible-router-anywhere). Frameworks covered include OpenAI Agents SDK, Anthropic Agent SDK, LangChain, Vercel AI SDK, PydanticAI, Mastra, Langfuse, LiveKit, Hermes Agent, and OpenClaw — details in [our SDK integrations guide](/blog/openpaths-sdk-integrations).

## Bottom line

Routing gets better quality per dollar than hand-picking (100% vs 77.8% on our benchmark at ~4% of the cost), redundancy stops being your code, one budget replaces five, and exit stays as cheap as entry. Building agents? Read [why LLM routers fit agent architectures](/blog/llm-router-for-ai-agents). For the current landscape, start with [the state of AI models, March 2026](/blog/state-of-ai-models-march-2026).

## FAQ

### What exactly is a multi-provider LLM API?

An API that exposes many providers' models through one endpoint, key, and billing balance. OpenPaths implements it as an OpenAI-compatible \`/v1\` router — 342 models, 57 providers — so existing clients work by swapping the base URL and key.

### Do I have to use automatic routing?

No. Pin any model by name for deterministic behavior. The \`auto-*\` routes are there when the router can trade down for an easy task or escalate a hard one — you choose per call.

### Does one key really cover images, video, and embeddings?

Yes. Embeddings, image generation (Netwrck RA1/ZImage, CuteDSL Z-Image Turbo), video (\`ra2v\`, \`kfold-video\`), music/TTS, research search, and image-to-3D all run through the same \`OPENPATHS_API_KEY\` and balance.

### How does pricing work compared to going direct?

Prepaid credits from $5, pay-per-token, no subscription. Prices per model are listed at [/models](/models), and first-party lanes like Netwrck and CuteDSL price tighter because we operate them ourselves.

### Can I migrate an existing integration without rewriting code?

Yes — change the base URL to openpaths.io and set \`OPENPATHS_API_KEY\`. Anthropic-native clients work too: \`/v1/messages\` is supported alongside \`/v1/chat/completions\`.
`,
  },
  {
    slug: 'llm-load-balancing',
    title: 'LLM Load Balancing Across Providers: Weights, Health Probes, Circuit Breakers',
    excerpt: 'LLM load balancing spreads your traffic across providers before anything breaks; failover reacts after something already failed. How weighted pools, continuous latency probes, and circuit breakers work in practice.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '7 min',
    tags: ['load balancing', 'routing', 'reliability', 'llm api', 'multi-provider'],
    content: `Most teams discover load balancing the hard way: one provider slows to a crawl at peak and every agent stalls with it. LLM load balancing spreads traffic across providers *before* anything fails. Failover reacts after something already failed. They are not the same thing.

## Load balancing vs failover

Failover is reactive. A request hits a dead endpoint and a fallback chain routes it elsewhere — but the user already ate the latency of the failed attempt.

Load balancing is proactive. The router decides where each request goes while everything is healthy, steering load away from pools that are getting slow before they tip over. Failover is the airbag; load balancing is the suspension. We cover the reactive side in [Claude API fallback patterns](/blog/claude-api-fallback); this post is about staying off the degraded path.

## Weighted pools

Simplest useful policy: weight each provider and split traffic proportionally.

\`\`\`yaml
pool:
  - provider: netwrck
    weight: 60
  - provider: deepseek-flash
    weight: 30
  - provider: gpt-luna
    weight: 10
\`\`\`

Weights encode intent: capacity, cost tolerance, quotas. Round-robin ignores all of that — it sends request N+1 to the next lane even when that lane is twice as slow right now. Fine for two identical endpoints; wasteful for providers with different quotas. Capacity-aware weighting treats weight as throughput instead of preference: a provider that can absorb ten times the requests per minute gets roughly ten times the share, adjusted for observed errors.

## Least-latency pick from continuous probes

Latency is not static, so measure continuously: probe every provider on an interval, track p50/p95 latency and error rates over a sliding window, and route each request to the pool currently fastest on your score.

We run these probes in public: see [our live latency stats](/stats). The key detail is that probes must run even when nothing looks wrong. A health check you trigger only after an error tells you the provider was broken thirty seconds ago, not whether it is safe now.

## Circuit breakers with half-open probes

Probes feed the balancer; circuit breakers protect individual requests. When a provider's error rate crosses a threshold, the breaker opens and traffic stops instead of burning retries. The part most implementations get wrong is recovery: never flip back to full traffic on a timer — that dumps a stampede onto a provider that may still be sick. Use a half-open state: let a few probe requests through. Success closes the breaker gradually; failure keeps it open — no oscillation where recovering providers get crushed by returning traffic and trip again.

Our auto models stack all three layers — embedding-based task selection sits on top of model-level circuit-breaker fallbacks ([how auto models work](/blog/how-auto-models-work)). Composing pipelines where each hop balances independently is covered in [building compound models](/blog/building-compound-models).

## The wrinkle nobody warns you about: diurnal wobble

Classic load balancing assumes failures are incidents: discrete, rare, obvious. LLM providers wobble diurnally instead. Capacity does not fail at 2pm; it just gets slower as usage climbs in some time zone, then recovers.

DeepSeek is the clearest example we publish: peak-hour latency wobbles noticeably while off-peak hours run at half price. That makes load balancing time-based — shift batch jobs into the cheap off-peak window, keep interactive traffic on stable lanes during peaks, and let weights drift on a schedule instead of only reacting to incidents. We mapped the windows in [the DeepSeek peak vs off-peak pricing map](/blog/deepseek-peak-off-peak-pricing-map). A balancer with only incident-shaped logic handles outages fine and still bleeds latency every afternoon.

## What an AI model router adds on top

All of the above is generic infrastructure — nginx does most of it. An AI model router adds what those tools lack: knowing which *models* behind which providers fit the request. Load balancing picks a healthy lane; routing also picks the right vehicle. Our cascade work showed the gap: hand-picked GPT-5.5 scored 77.8% on our 27-task coding benchmark at $2.55/run, while embedding-based routing scored 100% at $0.11 average.

There is also a first-party advantage. Netwrck and CuteDSL are lanes we operate ourselves, their capacity is predictable in a way third-party APIs never are: we control the hardware side, so assigned weights hold and probe curves stay flat. That makes aggressive balancing elsewhere safer — the fallback under a wobbling pool is known-good.

## Bottom line

Weighted pools for intent, least-latency selection from continuous probes for reality, breakers with half-open recovery for incidents, scheduled weights for diurnal pricing. Failover catches what slips through. OpenPaths gives you all of it behind an OpenAI-compatible endpoint — switching means changing \`base_url\` and \`OPENPATHS_API_KEY\`.

## FAQ

### Is round-robin good enough for LLM traffic?

Only between near-identical endpoints. Providers differ in rate limits, latency, and diurnal behavior, so round-robin keeps feeding measurably worse lanes. Weighted, capacity-aware splitting dominates it.

### How often should health probes run?

Frequently enough that the sliding window reflects the last few minutes, not the last hour — short intervals with long windows hide a wobble that started five minutes ago.

### Do circuit breakers replace retries?

No. Retries handle one failed request; breakers stop sending new requests where more failures are coming. Use both, with the breaker deciding where the retry lands.

### Can I balance across providers with different prices?

Yes — make latency-per-dollar part of the score instead of raw latency. Off-peak DeepSeek Flash at $0.14 input per 1M tokens is often right even when it is not fastest.
`,
  },
  {
    slug: 'best-model-for-coding',
    title: 'The Best Model for Coding in 2026, Measured Not Marketed',
    excerpt: 'The best model for coding depends on the shape of the job: we scored five models on the same animated-SVG tasks and routed 27 real coding tasks to see who actually wins at what.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '7 min',
    tags: ['coding', 'model-comparison', 'routing', 'benchmarks', 'llm'],
    content: `Ask which is the best model for coding and you will get a brand answer. We got a measurement instead. We ran the same animated-SVG creative tasks through five models and scored them on four axes, then ran a 27-task coding benchmark comparing hand-picked frontiers against embedding-based routing. The result is not a single winner but a map of where each model wins, where it collapses, and when the right answer is to stop picking a model at all.

## The scorecard

Same prompts, same animated-SVG briefs, scored 1 to 5:

| Model | Code | Visual | Motion | Discipline |
|---|---|---|---|---|
| Claude Opus 4.8 | 4.4 | 4.8 | 4.7 | 4.5 |
| GPT-5.5 direct | 4.7 | 4.1 | 4.2 | 4.8 |
| GPT-5.5 (xhigh thinking) | 4.1 | 3.8 | 3.5 | 3.0 |
| Gemini 3.5 Flash | 4.0 | 4.5 | 4.0 | 3.7 |
| Qwen3 Coder | 4.3 | 3.5 | 3.4 | 4.1 |

Full task-level breakdowns are in [our Opus 4.8 vs GPT-5.5 xhigh head-to-head](/blog/pelican-bicycle-opus-4-8-vs-gpt-5-5-xhigh) and the broader [animated-SVG model comparison](/blog/pelican-bicycle-animated-svg-model-comparison). Three flips break the marketing story:

**GPT-5.5 direct wins raw code and discipline.** Top Code score at 4.7, top Discipline at 4.8. If your job is producing exactly what the spec says and nothing more, direct GPT-5.5 is the strongest coder on this board.

**Opus 4.8 wins visuals.** Its 4.8 Visual score beats everything else, with Motion close behind at 4.7. For anything where the output has to look right — UI, creative canvas work, generative art — it is the pick.

**More thinking made GPT worse.** With xhigh reasoning enabled, GPT-5.5 dropped across the board: Code fell from 4.7 to 4.1, and Discipline collapsed to 3.0. More inference-time thinking produced over-engineered, less faithful output. Thinking budgets are a dial, not an upgrade; the default beat the maximum.

## Routing beats picking

Instead of one model for everything, how far does routing get you? From our [learning-to-route whitepaper](/blog/learning-to-route-whitepaper), across the 27-task coding benchmark:

| Strategy | Accuracy | Cost per run |
|---|---|---|
| Hand-picked GPT-5.5 | 77.8% | $2.55 |
| GPT-5.4-nano | 77.8% | $0.03 |
| Best single model: GPT-5.4-mini | 85.2% | $0.43 |
| Embedding-based cascade | 100% | $0.11 |

Two takeaways. First, expert intuition about the best single model was wrong — the hand-picked frontier matched a nano model at 85x the cost. Second, the cascade hit every task at about 4% of frontier pricing. That is the core argument of [choosing the right LLM](/blog/choosing-the-right-llm): the best model for coding is often a route, not a name.

## The everyday paid model carrying production traffic

The former \`openpaths/stealth/ox-alpha\` preview is now paid \`glm-5.3-flash\`. The old ID remains an alias, and the router keeps mandatory reasoning enabled without pinning every paid request to max effort.

It does not take everything. These domains skip straight up the fallback chain: GLSL/HLSL VFX shaders, trading systems, AI/LLM infrastructure such as CUDA kernels, quantization, and RAG pipelines, distributed architecture, compilers, and agentic multi-file patches. We profiled it in [ox-alpha: the coding workhorse](/blog/ox-alpha-coding-workhorse).

## A practical matrix

| Task | Reach for |
|---|---|
| Quick fixes, small scripts | \`auto-code\` (usually lands on paid GLM-5.3 Flash) |
| Feature work | \`auto-code\`, or GPT-5.4-mini direct if you want to pin one model |
| Refactors and multi-file changes | GPT-5.5 direct, or \`auto-hard-task\` |
| Shaders, VFX, anything visual | Claude Opus 4.8 |
| Trading systems | Frontier direct only — no free-tier routing |
| AI infra (CUDA, RAG, compilers) | Frontier direct — escalate past GLM-5.3 Flash |

If you are wiring agents rather than picking models by hand, see [the best AI API for coding agents](/blog/best-ai-api-for-coding-agents) — OpenPaths exposes all of this through one OpenAI-compatible base URL, so switching is two lines.

## Bottom line

The best model for coding in 2026 is conditional. Raw spec-following code: GPT-5.5 direct. Visual output: Opus 4.8. Cheap bulk: a small model or the cascade. Everything routine: let \`auto-code\` decide, starting from a free model and escalating only where the domain demands it.

## FAQ

### Is GPT-5.5 with maximum thinking better for coding?

Not always. On our scorecard, xhigh thinking dropped GPT-5.5 from 4.7 to 4.1 on Code and from 4.8 to 3.0 on Discipline. Default settings outperformed the maximum reasoning budget.

### Which model makes the fewest mistakes?

GPT-5.5 direct scored highest on Discipline at 4.8, with Opus 4.8 at 4.5. Discipline measures staying inside the spec — no invented features, no scope creep.

### Can a cheap model really match a frontier one?

Often yes. In our learning-to-route benchmark, GPT-5.4-nano matched hand-picked GPT-5.5 exactly at 77.8% accuracy while costing $0.03 versus $2.55 per run. The embedding cascade reached 100% at $0.11.

### What happened to ox-alpha?

Ox Alpha was revealed as GLM-5.3 Flash. Its aliases remain available, but they now route to and bill the paid model; hard domains still escalate automatically.

### How do I route instead of hardcoding a model?

Point your client at OpenPaths' OpenAI-compatible endpoint and request \`auto-code\`. Selection runs by embedding similarity to the task type, with circuit-breaker fallbacks per model — no code changes when the ranking improves.`,
  },
  {
    slug: 'best-model-for-tool-calling',
    title: 'The Best Model for Tool Calling in 2026: What Agents Actually Need',
    excerpt: 'There is no single best model for tool calling — there are four qualities that matter, and the right choice changes step by step through an agent loop.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '7 min',
    tags: ['tool-calling', 'agents', 'model-routing', 'llm-api'],
    content: `Every team building an agent eventually asks us the same question: which is the best model for tool calling in 2026? It is the right instinct pointed at the wrong target. We route production traffic across a catalog of 342 models from 57 providers ([state of the catalog](/blog/state-of-ai-models-march-2026)), and 139 of them declare \`supports_tools: true\`. Declaring support is easy. Carrying a 30-step agent loop without dropping arguments, hallucinating enum values, or burning your latency budget is rare.

## What actually separates good tool callers

Tool-use benchmarks are thin, so judge candidates on four qualitative factors instead.

**Schema adherence under long system prompts.** Real agents ship thousands of tokens of instructions before the first tool definition. Weak callers drift: they omit required parameters, invent enum values, or wrap JSON in prose. Structured outputs fix this at the API level — across the current GPT lineup they are production-grade, meaning responses conform to your declared schema rather than resembling it.

**Parallel call batching.** Strong callers issue independent operations in one turn: read three files, fetch two URLs. Weak callers serialize everything, multiplying round trips. On retrieval-heavy tasks this alone can halve wall-clock time.

**Graceful recovery after a failed call.** Tools fail — timeouts, rejected payloads, permission errors. A good caller reads the error, corrects the arguments, retries once, then changes approach. A bad one repeats the identical call until your loop guard kills the run.

**Latency budget across a 30-step loop.** A model that answers in 3 seconds finishes a loop in 90 seconds; a 20-second thinker takes ten minutes for the same work. Cost compounds identically. Slow-and-brilliant wins single-shot benchmarks and loses agents.

## A proxy for instruction-following under constraints

We have not published a dedicated tool-calling benchmark, so here is the closest verified signal we have: the Discipline column from our animated-SVG creative coding scorecard, where models were scored 1–5 on holding arbitrary constraints through complex output:

| Model | Code | Visual | Motion | Discipline |
|---|---|---|---|---|
| Claude Opus 4.8 | 4.4 | 4.8 | 4.7 | 4.5 |
| GPT-5.5 direct | 4.7 | 4.1 | 4.2 | 4.8 |
| GPT-5.5 (xhigh thinking) | 4.1 | 3.8 | 3.5 | 3.0 |
| Gemini 3.5 Flash | 4.0 | 4.5 | 4.0 | 3.7 |
| Qwen3 Coder | 4.3 | 3.5 | 3.4 | 4.1 |

It is not a tool-calling score, but instruction-following under constraints is exactly the muscle that keeps function calls on-spec late in a long conversation. Two details worth noting: GPT-5.5 direct leads at 4.8, and the same model with xhigh thinking drops to 3.0 — more reasoning tokens made constraint adherence worse, which matches what we see in agent loops generally.

## Stop picking one. Route per step.

Given all this, our recommendation is not a model name. Pinning a single best tool caller means paying frontier latency on trivial steps and getting locked out when that provider degrades. Instead, let routing decide per call: point your agent at \`auto-medium-task\` (or \`auto-easy-task\` for mechanical steps) and OpenPaths selects by embedding similarity to task type, with circuit-breaker fallbacks when a route fails mid-run. We cover the mechanics in [how auto models work](/blog/how-auto-models-work) and the architecture in [our LLM router for AI agents](/blog/llm-router-for-ai-agents).

Migration is two lines if you already run the OpenAI or Anthropic SDKs — see [migrating agent SDKs to OpenPaths](/blog/migrate-openai-anthropic-agent-sdks-to-openpaths) — and frameworks like Hermes Agent and OpenClaw work natively ([integration notes](/blog/openpaths-agent-integrations-hermes-openclaw)).

## Tool calling is only as good as your tools

One factor teams underrate: the tool itself. Give the model compact output and even a modest caller succeeds; give it 50 KB of HTML and no model survives your context window. Here is a real callable tool on OpenPaths — research search against the Papers index:

\`\`\`bash
curl https://openpaths.io/v1/search \
  -H "Authorization: Bearer $OPENPATHS_API_KEY" \
  -d provider=papers \
  -d format=markdown \
  -d query="process reward models for LLM reasoning"
\`\`\`

At $0.001 per search, it returns results over papers, methods, datasets, and GitHub code as compact markdown built specifically for agent context windows — tokens go to reasoning about results, not parsing them.

## Bottom line

The question dissolves once you look at what agents actually do: hundreds of heterogeneous steps where the optimal caller differs each time. Demand structured outputs, parallel batching, error recovery, and sane latency; verify discipline under constraints; then let \`auto-medium-task\` pick per step with fallbacks wired.

## FAQ

### Do all models that claim tool support actually call tools well?

No. Of our 342-model catalog, 139 declare \`supports_tools: true\`, but declaration says nothing about schema adherence deep into a conversation or recovery after errors. Test with your real schemas and a deliberately long system prompt before committing.

### Should I use one model for every step of my agent?

We recommend against it. Planning, retrieval, and formatting have different latency and capability budgets. Task-type routing like \`auto-medium-task\` matches each step to a suitable model automatically and falls back when a provider fails.

### Is more reasoning always better for tool calls?

Our scorecard suggests the opposite: GPT-5.5 scored 4.8 on Discipline direct but 3.0 with xhigh thinking enabled. Extra deliberation helps hard planning steps and hurts routine calls where speed and constraint-following dominate.

### What does a typical agent tool call cost?

Tool calls are billed as ordinary tokens plus any tool-specific fee. Search-style tools start at $0.001 per search via the Papers provider; model-side costs follow standard per-token rates listed at [/models](/models), with live latency probes at [/stats](/stats).
`,
  },
  {
    alternativePath: '/alternatives/litellm',
    slug: 'litellm-alternative',
    title: 'LiteLLM Alternative: Managed Model Routing Without Self-Hosting a Proxy',
    excerpt: 'Looking for a LiteLLM alternative? LiteLLM is a solid open-source proxy you self-host; OpenPaths gives you the same OpenAI-compatible surface as a managed service with automatic routing, fallbacks, and no gateway to operate.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['litellm', 'llm-router', 'self-hosting', 'model-routing', 'openai-compatible'],
    content: `If you are evaluating a LiteLLM alternative, start with respect for what LiteLLM is: a genuinely good open-source LLM gateway that unifies provider APIs behind an OpenAI-shaped interface, with real router and fallback concepts and active community maintenance. If your team wants full control of that proxy — your repo, your deploys, your database — it is reasonable.\.

But self-hosting a proxy is not free, and most teams underestimate what it looks like six months in. Here is what you inherit, how OpenPaths compares as a managed option, and when self-hosted is still the right call.

## What you take on when you self-host

- **Deploys and upgrades.** Every new provider API or model launch eventually means bumping the proxy version and rolling out config changes on your schedule.
- **Persistence.** Budgets, spend tracking, and logs need durable storage — a database to provision, back up, and monitor. 
- **Model choice per request.** Someone still decides which model handles which task and revisits those rules as pricing and quality shift.
- **On-call.** When the gateway is down at 2 a.m., that is your pager.

None of this is a flaw in LiteLLM. It is the cost of the control it gives you.

## Side by side

| | LiteLLM (self-hosted) | OpenPaths (managed) |
|---|---|---|
| Hosting | You deploy and operate it | Managed at [OpenPaths](/), nothing to run |
| Interface | OpenAI-shaped proxy | OpenAI-compatible \`/v1\`, plus native \`/v1/messages\` |
| Model choice | You write routing rules per task | \`auto-easy-task\` through \`auto-think\` and \`auto-code\` pick via embedding similarity |
| Fallbacks | Configured and tested by you | Built-in circuit-breaker fallbacks, maintained by us |
| Media generation | Bring your own providers | Netwrck RA1 images $0.04/image, ManifoldGen \`kfold-video\`, CuteDSL endpoints |
| Research search | Not part of the gateway | Papers search via \`POST /v1/search\`, $0.001/search |
| Credits | Per-provider keys you consolidate yourself | One prepaid pool from $5, pay-per-token across all models |
| Observability | Whatever you build on your own logs/db | Public [/stats](/stats) latency probes; every price at [/models](/models) |

## Why managed routing changes the problem

The hard part of running a gateway is not calling providers uniformly — LiteLLM solved that. It is knowing which model should answer each request and keeping that mapping correct as models change. OpenPaths routes by embedding similarity to task type instead of hand-written rules, and our learning-to-route benchmark shows why: embedding-based cascade routing hit 100% accuracy on a 27-task coding suite at $0.11 average per run, versus 77.8% for hand-picked GPT-5.5 at $2.55 per run. Routing intelligence is a product surface, not a config file you babysit.

For how this compares against aggregator-style alternatives rather than self-hosted gateways, see [OpenPaths vs OpenRouter](/blog/openpaths-vs-openrouter) and [Together AI alternative for production model routing](/blog/together-ai-alternative-for-production-model-routing). Commercial gateway-adjacent option: [Portkey alternative](/blog/portkey-alternative).

## Running both side by side

Because both speak the OpenAI shape, comparing them is two lines:

\`\`\`bash
export OPENAI_BASE_URL="https://openpaths.io/v1"
export OPENPATHS_API_KEY="sk-your-key"
\`\`\`

Full details in [use OpenPaths' OpenAI-compatible router anywhere](/blog/use-openpaths-openai-compatible-router-anywhere) and the step-by-step guide [switch to OpenPaths in 2 lines](/blog/switch-to-openpaths-in-2-lines). Keep LiteLLM in front of some traffic while you validate — one credit pool spans GPT-5.5 at $5.00 input per 1M down to DeepSeek V4 Flash at $0.14 input per 1M, so experiments cost cents.

## When LiteLLM is still the right call

- **Air-gapped or strict self-host compliance.** Self-hosting in your VPC is the right tool when no third party may see inference traffic.
- **Deep open-source customization.** Forking the gateway for custom auth or bespoke middleware beats any managed API.
- **Existing platform investment.** Teams already operating it well gain less from switching than teams starting from zero.

## Bottom line

LiteLLM is the strongest open-source option if you want to own the gateway. OpenPaths exists for teams who want what the gateway provides — uniform API access, smart routing, fallbacks, media generation, research search, observability — without owning the deploys, the database, or the pager. Two lines get you a live comparison; prepaid credits from $5 mean trying it costs less than the meeting to discuss trying it.

## FAQ

### Is LiteLLM really free if I self-host it?

The software is open-source, but infrastructure, the budgets/logs database, and on-call time all add up. Total cost usually exceeds a managed fee for small teams.

### What does OpenPaths handle that a self-hosted proxy does not?

Routing decisions and fallbacks are maintained for you instead of configured by you, plus first-party media generation (Netwrck RA1 images, ManifoldGen kfold-video), research search via Papers, and one prepaid credit pool replacing per-provider billing consolidation.

### Can I keep my existing code when switching from LiteLLM?

Yes. OpenPaths exposes OpenAI-compatible \`/v1\` plus Anthropic-native \`/v1/messages\`; changing base URL and key moves code over unchanged.

### How do the auto routes decide between models?

Requests match embedding similarity to task type, then route with model-level circuit-breaker fallbacks — 100% accuracy at ~4% of frontier cost on our 27-task benchmark.

### Do I have to abandon LiteLLM entirely?

No. Both present an OpenAI-shaped interface, so split traffic by client or environment while evaluating and keep both as long as you like.
`,
  },
  {
    alternativePath: '/alternatives/portkey',
    slug: 'portkey-alternative',
    title: 'Portkey Alternative: Routing First, With Media Search and Auto Modes Built In',
    excerpt: 'Looking for a Portkey alternative? OpenPaths puts measured model routing at the center instead of guardrails around a single provider, and adds image, video, forecasting, and research search under one API key.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['ai-gateway', 'llm-routing', 'portkey-alternative', 'multi-provider'],
    content: `If you are evaluating a Portkey alternative, start honestly: Portkey does what it advertises. It is a solid commercial AI gateway — observability for LLM calls, guardrails in front of them, caching to cut repeat spend.

OpenPaths starts one step earlier. Gateway-style tooling assumes you wire up one or two models and wrap infrastructure around them. Our question is upstream: which model should serve this specific request? Measurement picks the model per request rather than you hardcoding the answer. The gateway features still exist, but routing is the product, not the wrapper.

## What routing-first actually buys you

Our learning-to-route benchmark (27 coding tasks) makes the case:

- Hand-picked GPT-5.5: 77.8% accuracy at $2.55 per run.
- GPT-5.4-nano: identical 77.8% at $0.03 — 85x cheaper.
- Best single model, GPT-5.4-mini: 85.2% at $0.43.
- Embedding-based cascade routing: **100% at $0.11 average**, roughly 4% of frontier cost.

No amount of observability tuning closes the gap between lines one and four. A fixed model choice leaves accuracy or money on the table for most requests; a measured cascade captures both. The method is published in the [learning-to-route whitepaper](https://huggingface.co/openpaths/learning-to-route) and powers production routes today: \`auto-easy-task\`, \`auto-medium-task\`, \`auto-hard-task\`, \`auto-think\`, and \`auto-code\`. Selection uses embedding similarity to task type with circuit-breaker fallbacks, so a flaky upstream degrades to the next option instead of erroring your request.

## Transparent where gateways are opaque

Most commercial gateways publish a status page when something breaks. OpenPaths runs public latency probes at [/stats](/stats) all the time, so you see per-model latency before committing traffic. Every price sits at [/models](/models) — 342 models across 57 providers, each with its rate card visible.

Switching costs stay near zero: everything speaks the OpenAI-compatible protocol at \`/v1\` plus Anthropic-native \`/v1/messages\`, so migrating is a base_url and key change — two lines. Integrations cover OpenAI Agents SDK, Anthropic Agent SDK, LangChain, Vercel AI SDK, PydanticAI, Mastra, Langfuse, LiveKit, Hermes Agent, and OpenClaw. We cover the migration path in [OpenPaths vs OpenRouter](/blog/openpaths-vs-openrouter).

## Breadth past text

This is where the comparison stops being close. A text-era gateway handles text; production apps also generate images and video, forecast time series, search research literature, and embed documents. All of it lives behind one \`OPENPATHS_API_KEY\`:

| Capability | Example | Price |
|---|---|---|
| Flagship images | Netwrck RA1 | $0.04/image |
| Anime art | ZImage | $0.007/image |
| Fast images | CuteDSL Z-Image Turbo | $0.04/image |
| Cinematic video | ManifoldGen \`kfold-video\` (H3) | at [/models](/models) |
| Image-to-video | \`ra2v\` via Netwrck | at [/models](/models) |
| Forecasting | \`chronos2\` | $0.20/forecast |
| Research search | Papers (\`format: "markdown"\`) | $0.001/search |
| Embeddings | Text-Generator.io plus local model | pay-per-token |

Netwrck is our first-party partner for art and video — policy-flexible enough to serve as our image-gen fallback when other providers refuse, with endpoints at \`/v1/images/generations\` and \`/v1/videos/generations\`. We profile it in [our Netwrck provider guide](/blog/provider-netwrck). For deeper media dives, see [the best image generation APIs of 2026](/blog/best-image-generation-api-2026) and [the best video generation APIs of 2026](/blog/best-video-generation-api-2026).

The Papers endpoint deserves a callout for agent builders: \`POST /v1/search\` with \`provider: "papers"\` returns compact markdown over papers, methods, datasets, and GitHub code at $1 per 1,000 searches. Pure gateways expose nothing like it.

## How the platforms compare

| Dimension | Portkey | OpenPaths |
|---|---|---|
| Core posture | Gateway around your chosen models | Router that picks per request |
| Auto modes | Not the focus | Five task-type routes plus fallbacks |
| Latency transparency | Status pages on incident | Public probes always on at [/stats](/stats) |
| Images / video | Out of scope | RA1 $0.04, ZImage $0.007, Z-Image Turbo $0.04; \`kfold-video\`, \`ra2v\` |
| Forecasting / search | Out of scope | \`chronos2\` $0.20/forecast; Papers $0.001/search |
| Pricing | Commercial plans | Credits from $5, pay-per-token, no subscription |
| Catalog | Your configured providers | 342 models, 57 providers, 139 tool-capable |

Fair trade-offs: if you need deep tracing dashboards around an already-settled single-vendor stack, Portkey's focus is legitimate. If you self-host and want source access, [our LiteLLM comparison](/blog/litellm-alternative) covers that end of the spectrum. OpenPaths occupies the middle most teams want: managed, measurable routing with media and search breadth included.

## Bottom line

Pick Portkey if your stack is settled and you want observability, guardrails, and caching wrapped around it. Pick OpenPaths if you would rather have measurement make the model decision per request — spanning frontier text, $0.04 flagship images, cinematic video, forecasts, and research search behind one key. The switch is two lines; [/stats](/stats) will show within minutes whether routing beats your current fixed choice.

## FAQ

### Is OpenPaths a drop-in replacement for Portkey?

It replaces more than it swaps. Both sit between your app and providers over standard protocols, so the base_url-and-key change works for existing OpenAI-shaped code. The difference: instead of configuring one model plus guardrails, point requests at the auto routes and let measured selection handle the rest.

### Does OpenPaths include Portkey-style observability?

You get the transparency that matters for routing: public latency probes at [/stats](/stats) covering every provider continuously, and every price published at [/models](/models). Latency and cost data should be always-on, not surfaced only when something breaks.

### Can one API key really cover images, video, forecasting, and search?

Yes. One \`OPENPATHS_API_KEY\` reaches Netwrck RA1 ($0.04/image), ZImage ($0.007/image), CuteDSL Z-Image Turbo, ManifoldGen \`kfold-video\`, \`ra2v\` image-to-video, \`chronos2\` forecasting ($0.20/forecast), Papers research search ($0.001/search), embeddings, and the full text catalog.

### How does auto-routing compare to picking my own model?

On our 27-task coding benchmark, the best hand-picked single model reached 85.2% accuracy at $0.43 per run while cascade routing reached 100% at $0.11 average. Fixed choices leave quality or budget on the table depending on which way you guess wrong; the router corrects per request.

### What does OpenPaths cost compared to a commercial gateway plan?

No subscription tiers. Credits start at $5 via Stripe or crypto, and you pay per token or per generation at rates listed at [/models](/models). Heavy users benefit twice: published pricing plus automatic routing toward cheaper models that still complete the task.`,
  },
  {
    slug: 'no-cost-byok-18-providers',
    title: 'No-Cost BYOK: Use Your Own Anthropic, OpenAI, and 16 Other Provider Keys',
    excerpt: 'OpenPaths now manages keys for 18 providers. Requests served through your own key bypass your OpenPaths balance entirely — you pay your provider directly and we charge nothing on those tokens.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['byok', 'pricing', 'anthropic', 'openai', 'gateway'],
    content: `Switching gateways usually means one of two fears: losing your provider rates, or paying a markup on top of them. Both fears have a name — BYOK — and both have had weak answers in most gateways: BYOK behind enterprise plans, or BYOK with a "platform fee" per request.

Ours is simpler. Add your own provider key on OpenPaths and requests served through that key **bypass your OpenPaths balance entirely**. The billing engine records cost zero. You pay Anthropic directly at Anthropic's prices, OpenAI directly at OpenAI's prices. We take nothing on those tokens.

## What you keep when you bring your own key

A raw provider key by itself does not give you routing. On OpenPaths it still does:

| Capability | With a BYOK key | Why it matters |
|---|---|---|
| One OpenAI-compatible API | Yes | Same \`/v1/chat/completions\` request shape across all 18 providers |
| Automatic fallbacks | Yes | If your key hits a rate limit or 5xx, the request falls through to OpenPaths' own provider lanes |
| Anthropic-native endpoint | Yes | \`POST /v1/messages\` works for Claude Code and Anthropic SDKs |
| Observability | Yes | Latency, TTFT, and throughput still recorded in [public stats](/stats) |
| Platform fee on BYOK tokens | $0 | Cost bypasses your OpenPaths balance; no markup, no per-request fee |

## The providers

All eighteen key slots live in **Account → API Keys → Provider keys (BYOK)**:

* **OpenAI** — platform.openai.com
* **Anthropic** — console.anthropic.com
* **Google AI** — aistudio.google.com
* **Mistral**, **Groq**, **xAI**, **DeepSeek** — direct API consoles
* **Together AI**, **Inference.net**, **OpenRouter** — aggregator lanes under your own billing
* **MiniMax**, **Netwrck**, **Z.AI** (GLM Coding Plan keys supported), **Sakana AI**
* **fal.ai**, **Black Forest Labs** — image generation
* **Thinking Machines Tinker** — Inkling routes
* **OpenAI Max plan** — OAuth sign-in rather than a key, in its own panel

Each row shows a masked preview of the stored key and can be replaced or deleted at any time.

## Claude Code on your own Anthropic key

The Anthropic-native endpoint means Claude Code and the Anthropic Agent SDK work unchanged:

    ANTHROPIC_BASE_URL = https://openpaths.io
    ANTHROPIC_AUTH_TOKEN = your-openpaths-key
    ANTHROPIC_MODEL = openpaths/auto-code

With an Anthropic key saved in Provider keys, Claude traffic rides your Anthropic billing at $0 through us — and if that key fails mid-session, requests fall through to our lanes instead of dying with your terminal.

## Honest security posture

Keys are stored server-side and never returned in full by the API — list responses carry masked previews only, and deletion is immediate. We do not claim hardware-backed encryption at rest, and we think gateway vendors who market BYOK should say exactly that much.

## When to use BYOK vs credits

Use credits when you want one bill, committed-volume pricing, and zero provider accounts to manage. Use BYOK when you already have annual commitments or negotiated rates with a provider and want routing plus fallbacks around them. Most teams do both at once — which is exactly why a BYOK route failing falls through to a credit-funded lane instead of erroring.`,
  },
  {
    slug: 'why-this-model-routing-transparency',
    title: '"Why This Model?" — Routing Transparency via X-OpenPaths-Route',
    excerpt: 'Auto-routing is only trustworthy when it shows its work. Every chat response now carries an X-OpenPaths-Route header stating the resolved model, provider, ordering strategy, and whether your own key served it.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['routing', 'observability', 'api', 'transparency'],
    content: `The standard objection to auto-routing is not accuracy. It is auditability. A router that picks models silently asks you to debug quality regressions with guesswork: was it the model, the prompt, or the fact that Tuesday's requests went somewhere else?

So we made the routing decision inspectable on every response. No dashboard detour, no support ticket — one HTTP header.

## The header

\`\`\`
X-OpenPaths-Route: model=deepseek-v4; provider=nvidia; strategy=price; byok=false; requested=openpaths/auto-code
\`\`\`

Five fields, each answering part of "why this model?":

| Field | Meaning |
|---|---|
| \`model\` | The backend model ID that actually served the request |
| \`provider\` | Which provider lane handled it |
| \`strategy\` | How the fallback chain was ordered: \`price\` (default), \`config\`, or \`fastest\` |
| \`byok\` | Whether your own provider key served this request ($0 through us) |
| \`requested\` | Present only when you asked for an auto alias — shows what you asked for vs what answered |

When guardrails or model-access rules narrow the candidate chain, the surviving route is what appears here. When a first-choice provider fails and fallback fires, the header reflects the route that actually answered, not the one that was tried first.

## Reading it in practice

Pin down what \`openpaths/auto\` resolved to during a test run:

    curl -i https://openpaths.io/v1/chat/completions \\
      -H "Authorization: Bearer $OPENPATHS_API_KEY" \\
      -H "Content-Type: application/json" \\
      -d '{"model":"openpaths/auto","messages":[{"role":"user","content":"hi"}]}'

The same header ships on streaming responses and on the Anthropic-native \`POST /v1/messages\` endpoint, so Claude Code sessions get the same audit trail. Two adjacent headers complete the picture: \`X-OpenPaths-Cache\` says whether the answer came from the response cache, and \`X-OpenPaths-Reasoning-Effort\` appears whenever a reasoning effort was adapted to fit the chosen model's capabilities.

Log the header next to your prompt IDs and every quality regression becomes a query instead of a memory: model, provider, strategy, cache state, effort — attached to the exact responses that misbehaved.

## What transparency does not claim

The header reports the routing decision, not a grade. It will not tell you the selected model was optimal, and \`strategy=price\` means exactly "candidates ordered by blended token price," nothing smarter. Where selection is heuristic — task-tier classification from the prompt — \`requested=\` shows the alias that triggered it so you can reproduce the decision. If a number or field would be marketing, it is not in the header.

Full parameter documentation lives in the [API docs](/docs); the routing design notes are in the [auto-router whitepaper](/blog/learning-to-route-whitepaper).`,
  },
  {
    slug: 'llm-gateway-observability-real-stats',
    title: 'What a Production LLM Gateway Owes You: Observability From Real Traffic',
    excerpt: 'Latency, TTFT, and throughput per model, measured from production usage_logs and live probes — public at /stats, powering /status, and honest about what it does not measure.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '7 min',
    tags: ['observability', 'reliability', 'production', 'stats'],
    content: `Ask gateway vendors for latency numbers and you get marketing benchmarks: single-region, empty-account, best-of-three runs. Ask them for time-to-first-token under agent workloads and the room goes quiet. Yet TTFT is the number your users feel and the one that decides whether an agent loop takes 8 seconds or 80.

We took a boring position: measure the real thing, from real traffic, and publish it.

## What we record

Every completion through OpenPaths writes to usage_logs: token counts, end-to-end latency, time-to-first-token, and output throughput, tagged with model, provider, and calling app. That table feeds three public surfaces:

| Surface | Endpoint | Question it answers |
|---|---|---|
| [Stats dashboard](/stats) | \`/stats/breakdown\`, \`/stats/models/timeseries\` | What do latency and TTFT look like per model over real mixed traffic? |
| Live model probes | \`/stats/model-probes\` | Is each model actually completing requests right now? |
| [Status page](/status) | derived from probes | Is anything degraded before my pagers notice? |

Probes are periodic "say hi" completions against each model in the catalog — cheap, constant-shape requests whose latency series doubles as a health signal. The status page renders them fresh-or-stale per model with failure reasons, refreshed every minute. No synthetic uptime percentages, because nobody can define them consistently; just recent probe outcomes you can check yourself.

## What TTFT data changes about model choice

Once you can sort the catalog by measured TTFT instead of vibes, routing debates settle fast:

* Flash-class models win end-to-end latency mostly at small outputs; at long generations, throughput dominates and the gap narrows.
* Reasoning models trade TTFT for quality in a way that is invisible in any per-token price table — you only see the tax once it is charted.
* Provider choice for the *same weights* moves latency more than model choice sometimes. DeepSeek V4 via NVIDIA NIM and via the direct API are different experiences; the breakdown view shows both.

This is the dataset \`routing_strategy: "fastest"\` and the \`openpaths/auto-fast\` alias act on. When you let the router optimize latency, it is optimizing these measurements — not a vendor's benchmark suite.

## For your own account

The same recorder powers per-key usage views in Account: spend by model, latency by model, app attribution for traffic from your tools. The [usage search](/usage/prompts) indexes recorded responses so "which prompts produced garbage yesterday" is searchable text, not log grepping.

## Limits, stated plainly

Latency percentiles are computed over completed requests only; a timeout is a probe failure, not a slow sample. Probe cadence is minutes, not seconds — status reflects recent history, not sub-minute blips. And stats measure our gateway hop, not your client's network. We would rather under-claim with numbers than over-claim without them.

If you are evaluating gateways on anything other than price, ask every vendor for their TTFT distribution under load. If they cannot show one, they are selling you a logo wall.`,
  },
  {
    slug: 'best-llm-api-2026-compared',
    title: 'The Best LLM APIs in 2026, Compared With Real Benchmark Data',
    excerpt: 'Six ways to call an LLM in 2026 — OpenAI, Anthropic, Google, DeepSeek, OpenRouter, and OpenPaths — compared using measured benchmark runs and published price cards instead of marketing pages.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '9 min',
    tags: ['llm', 'api', 'comparison', 'benchmark'],
    content: `Every "best LLM API" listicle ranks models by vibes. This one ranks providers by what actually happens when you send tokens through them: measured scores from our own multi-model benchmark runs, real generated artifacts you can inspect, and price cards you can reproduce on your own bill. We run [OpenPaths](/), an OpenAI-compatible router, so we have an obvious bias — but we ship it as one of six options here, with its limitations stated plainly, because credibility is worth more than a sale.

The short answer up front: there is no single best API. There is a best API *for your traffic shape*, and the gap between picking well and picking badly was **30x cost at equal accuracy** in our own routing benchmark.

## The options at a glance

| Provider | Best feature | Entry price (input, per 1M tokens) | Billing |
|---|---|---|---|
| OpenAI API | Broadest ecosystem and tooling | $0.20 (GPT-5.6 Luna) | Pay-per-token |
| Anthropic API | Long-form code and agent quality | $2.00 tier (Sonnet class); Opus $5.00 | Pay-per-token |
| Google AI (Gemini) | Price-to-capability in the mid tier | $1.25 (Gemini 2.5 Pro class) | Pay-per-token + free tier |
| DeepSeek direct | Cheapest serious frontier-adjacent model | $0.14 (V4 Flash, off-peak half price) | Pay-per-token |
| OpenRouter | One account, hundreds of models | Passes through provider pricing | Prepaid credits |
| OpenPaths | One key, auto-routing across all of the above | $0.14 and up | Prepaid credits, pay-per-token |

Prices are published rates as of August 2026; output tokens cost more everywhere, typically 3–5x input.

## OpenAI API

- **Best for:** teams that want one vendor, mature SDKs, and the widest ecosystem of tooling.
- **Strength:** the current lineup spans an ultra-cheap fast tier (GPT-5.6 Luna at $0.20/$1.20 per million in/out) up to frontier reasoning at $5+ input. Structured outputs, batching, and cached-input discounts are all production-grade.
- **Cost:** pay-per-token, no subscription.
- **Limitations:** you only get OpenAI models. Our benchmarks below show GPT-5.5 losing to cheaper models on specific job shapes, and a single-vendor setup gives you no way to exploit that.

## Anthropic API

- **Best for:** long coding sessions, agentic loops, and writing where coherence over thousands of tokens matters.
- **Strength:** Claude topped our creative-coding scorecard (details below) and remains the model our own engineers reach for on hard refactors. Direct pricing is $5.00/$25.00 per million tokens on the Opus tier as of August 2026, with mid tiers below that.
- **Cost:** pay-per-token; premium relative to mid-tier rivals at the frontier.
- **Limitations:** the price. On jobs where Claude scores 4.4 and a $0.14 DeepSeek Flash scores close behind, paying 35x per token needs justification.

## Google AI (Gemini)

- **Best for:** high visual and compositional quality at mid-tier prices.
- **Strength:** Gemini 3.5 Flash scored 4.5/5 on visual composition in our scorecard — the best visual score in the run — at a fraction of frontier pricing. Gemini 2.5 Pro-class models sit around $1.25 input per million tokens.
- **Cost:** pay-per-token, with a genuine free tier for prototyping.
- **Limitations:** ambitious compositions need strict format discipline; the same model scored 3.7 on instruction-following in our run. It benefits from tight prompting.

## DeepSeek direct

- **Best for:** batch work, classification, extraction, and any pipeline where cost dominates.
- **Strength:** V4 Flash lists at $0.14 input per million tokens, and off-peak hours are half price — a scheduling discount nobody else offers at this level.
- **Cost:** the floor of the serious market.
- **Limitations:** peak-hour latency can wobble, and it is one model family. You still need a second provider for vision-heavy or frontier-reasoning jobs.

## OpenRouter

- **Best for:** experimentation — trying many models through one account without vendor signups.
- **Strength:** hundreds of models, one integration, pass-through pricing with no per-token markup on standard routes.
- **Cost:** prepaid credits; BYOK routes carry a small percentage fee.
- **Limitations:** it is a catalog, not a decision. You still pick the model per request, and reliability varies by upstream provider. No routing intelligence.

## OpenPaths

- **Best for:** developers who want the whole market behind one OpenAI-compatible key and let measurement decide which model serves each request.
- **Strength:** one key covers OpenAI, Anthropic, Google, DeepSeek, Mistral, Groq, xAI, and more. Models route by embedding similarity to task type, with circuit-breaker fallbacks when a provider degrades. Prepaid credits from $5 (Stripe or crypto), pay-per-token, no subscription. Latency probes are public at [/stats](/stats) and every model price is listed at [/models](/models).
- **Cost:** provider-level prices plus routing; e.g. GPT-5.5 at $5.00 input, DeepSeek Flash at $0.14.
- **Limitations:** you depend on our routing choices unless you pin models yourself, and the catalog, while broad, is curated rather than exhaustive.

## What the benchmarks actually showed

### Creative coding scorecard

We ran five models on the same animated-SVG creative-coding tasks and scored code quality, visual result, motion, and instruction discipline on 1–5. Full methodology and raw outputs are in [our creative lab writeup](/blog/llm-creative-lab-animated-svg-benchmark):

| Model | Code | Visual | Motion | Discipline |
|---|---|---|---|---|
| Claude Opus 4.8 | 4.4 | 4.8 | 4.7 | 4.5 |
| GPT-5.5 direct | 4.7 | 4.1 | 4.2 | 4.8 |
| GPT-5.5 (xhigh thinking) | 4.1 | 3.8 | 3.5 | 3.0 |
| Gemini 3.5 Flash | 4.0 | 4.5 | 4.0 | 3.7 |
| Qwen3 Coder | 4.3 | 3.5 | 3.4 | 4.1 |

![Claude Opus 4.8 animated SVG: pelican riding a bicycle](/static/blog/pelican-svg/opus48.svg)

That is Opus 4.8's one-shot output — spinning wheels, a cranking pelican, scrolling road. Here is the same prompt through GPT-5.5 with thinking disabled, which produced the cleanest single-pass file in the entire run:

![GPT-5.5 animated SVG: pelican riding a bicycle](/static/blog/pelican-svg/gpt55-none.svg)

Two lessons for API buyers. First, more thinking budget made GPT-5.5 *worse* here (discipline collapsed from 4.8 to 3.0) — reasoning spend is not free quality. Second, the ranking flips by criterion: GPT wins code, Claude wins visuals, Gemini wins ambition-per-dollar. A single-vendor contract freezes you into one row of this table.

### Routing benchmark: cost gap at equal accuracy

We also ran a 27-task coding benchmark comparing hand-picked models against automatic routing ([learning-to-route](https://huggingface.co/openpaths/learning-to-route)):

- Hand-picking GPT-5.5: 77.8% accuracy at an average **$2.55** per run.
- Hand-picking GPT-5.4-nano: 77.8% accuracy at **$0.03**. Identical score, 85x cheaper.
- Best single model: GPT-5.4-mini, 85.2% at $0.43.
- Embedding-based cascade routing: **100%** at $0.11 average — about 4% of frontier-model cost.

If your benchmark harness cannot tell a $2.55 run from a $0.03 run, neither should your production router.

## How to choose

Pick **OpenAI** if you want one mature vendor and accept paying frontier prices for mid-frontier results sometimes. Pick **Anthropic** for hard agentic coding where its scorecard lead pays for itself. Pick **Gemini** when visual quality per dollar matters. Pick **DeepSeek** for cheap volume. Pick **OpenRouter** to shop. Pick **OpenPaths** to stop shopping and let measured routing make the choice per request — including the same image and video models covered in [our image API comparison](/blog/best-image-generation-api-2026) and [video API guide](/blog/best-video-generation-api-2026).

## FAQ

### What is the cheapest LLM API per million tokens?

DeepSeek V4 Flash at $0.14 input per million tokens is the floor among serious models as of August 2026, dropping to half that during off-peak hours. Free tiers exist (GLM-4.6-Flash, NVIDIA's free DeepSeek route) for non-production volumes.

### Is a router API slower than calling a provider directly?

A router adds one hop, but embedding-based routing is a lookup, and circuit breakers often make effective latency *lower* than direct calls because degraded providers get skipped automatically. Live probes are at [/stats](/stats).

### Do I need separate accounts for OpenAI, Anthropic, and Google models?

Not with an aggregator. OpenRouter and OpenPaths both give you one account and one key; OpenPaths additionally routes automatically and publishes per-token prices at [/models](/models).

### Are expensive models always more accurate?

No. In our 27-task benchmark, GPT-5.5 at $2.55 per run scored exactly the same as GPT-5.4-nano at $0.03. Match the model to the task, not the invoice to the ego.`,
  },
  {
    slug: 'best-image-generation-api-2026',
    title: 'The Best Image Generation APIs in 2026, Tested Side by Side',
    excerpt: 'One prompt, seven image APIs, real outputs you can inspect. FLUX, Z-Image, RA1, Grok Imagine, and GPT Image 2 compared on price per image, adherence, and where each falls down.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '8 min',
    tags: ['image generation', 'api', 'comparison', 'benchmark'],
    content: `Most "best image API" articles compare marketing screenshots. This one compares seven generations of the **same prompt** through APIs we serve in production, with the actual outputs embedded below so you can judge for yourself, plus the exact price per image each model bills. All of these run behind [OpenPaths](/) with one key; the interactive version of this gallery lives on our [Image Evals page](/image-evals).

The prompt family, held constant across every model:

\`\`\`text
A fox astronaut sitting on a mossy log in a misty pine forest,
wearing a glass bubble helmet, golden hour light, shallow depth of field
\`\`\`

Same prompt, wildly different results — which is the entire point of this article.

## Results at a glance

| Model | Price per image | Strength | Limitation |
|---|---|---|---|
| FLUX Schnell | $0.003 | Fastest and cheapest hosted FLUX | Coarser detail than dev/pro |
| Z-Image Turbo | $0.007 | Crisp subject, absurd value | Smaller resolution ceiling |
| FLUX.2 Klein | $0.02 | Coherent cheap daily driver | Soft fine texture |
| Grok Imagine | $0.02 | Richest scene-building | Subject less dominant |
| FLUX Dev | $0.025 | Excellent quality per dollar | Reads slightly 3D-render |
| RA1 | $0.04 | Never refuses, characterful | Softer up close, smaller native size |
| FLUX Pro | $0.04 | Most photographic output | Priciest FLUX tier |
| DALL-E 3 | $0.04 | Strong prompt following, widely integrated | Older aesthetic generation |
| Nano Banana 2 | $0.14 | Top-tier editing and consistency | Premium price |
| GPT Image 2 | $0.211 | Best instruction adherence in our tests | 30x the price of Z-Image |

## The outputs, side by side

![GPT Image 2 generation: fox astronaut in forest](/static/blog/image-eval/gpt-image-2.webp)

**GPT Image 2** had the best prompt adherence of the group: clean helmet bubble, convincing rim light, believable fur, depth-of-field respected without dissolving the background. It is also by far the most expensive at roughly 21 cents per image. If correctness and text rendering matter more than budget, this is the pick; if you generate in bulk, do the multiplication before committing.

![FLUX Pro generation: fox astronaut in forest](/static/blog/image-eval/flux-pro.webp)

**FLUX Pro** produced the most photographic frame: profile composition, creamy bokeh, naturalistic fur, a helmet that reads as vacuum-formed glass. Of everything tested it is the one most likely to pass as a real photo. At four cents it costs a fraction of GPT Image 2 while winning on aesthetics.

![FLUX Dev generation: fox astronaut in forest](/static/blog/image-eval/flux-dev.webp)

**FLUX Dev**, the open-weights checkpoint, skews slightly toward "cute 3D render" — it read the bubble helmet as over-ear headphones in this run. Warm light, lovely bokeh, and at 2.5 cents it is excellent quality-per-dollar for everyday generation. Being open-weight, it is also the one you can self-host if volume justifies GPUs.

![Z-Image Turbo generation: fox astronaut in forest](/static/blog/image-eval/zimage.webp)

**Z-Image Turbo** is the value champion: under a cent per image, yet crisp centered subject, clean glass helmet, symmetric face, tasteful fog. For bulk generation — thumbnails, dataset augmentation, A/B creatives — cost dominates and nothing else here comes close.

![Klein generation: fox astronaut in forest](/static/blog/image-eval/klein.webp)

**FLUX.2 Klein** is the small efficient sibling: pleasant composition and color, though the helmet shrinks to a detail and fine texture goes soft. A good fast-and-cheap default when Z-Image's tighter framing is too rigid.

![RA1 generation: fox astronaut in forest](/static/blog/image-eval/ra1.webp)

**RA1** leans illustrative rather than photographic — warm, characterful, foggy golden-hour forest — and renders at a smaller native resolution, so it is softer up close. Its distinguishing property is reliability: RA1 never refuses, which is why OpenPaths auto-routes to it when another provider blocks a prompt. Every pipeline that must *always come back with something usable* wants a model like this in the chain.

![Grok Imagine generation: fox astronaut in forest](/static/blog/image-eval/grok-imagine-image.webp)

**Grok Imagine** went widest on scene: layered trees, volumetric light shafts, ground clutter, with the subject nicely integrated rather than dominant. When the environment should do as much storytelling as the character, it is strong and well-priced at two cents.

## FLUX family: which tier do you actually need?

The FLUX lineup is a controlled experiment in paying for quality. Schnell at $0.003 handles drafts; Klein at $0.02 handles production volume; Dev at $0.025 adds the open-weights escape hatch; Pro at $0.04 buys the photographic look. In our evals the jump from Klein to Pro was visible; the jump from Schnell to Klein mostly was not at thumbnail sizes. Start low, escalate only when a human eye complains.

## How we test

Same prompt, same aspect ratio, one generation per model, no cherry-picking retries, outputs published unedited at [/image-evals](/image-evals) alongside blind-vote Elo rankings from the Artificial Analysis arena. Prices are the per-image rates billed on OpenPaths as of August 2026; direct-provider pricing is similar but changes often, so treat the ordering as stable and the cents as approximate.

## FAQ

### What is the cheapest image generation API?

FLUX Schnell at $0.003 per image is the cheapest hosted option we serve, with Z-Image Turbo at $0.007 right behind it and noticeably crisper subjects. Both undercut DALL-E 3's roughly $0.04 standard rate by more than 10x.

### Which image API follows prompts best?

In our same-prompt evals, GPT Image 2 followed compositional and lighting instructions most faithfully. It costs $0.211 per image — roughly 30x over FLUX Dev for adherence, which only makes sense when instructions are load-bearing (text in images, brand constraints, layout specs).

### Can I use one API key for multiple image models?

Yes. Aggregators expose many models behind one endpoint. With OpenPaths one OpenAI-compatible key reaches every model in the table above, and \`auto-image\` picks one per prompt based on what the request needs.

### Is there a free image generation API?

Free tiers rotate; we keep genuinely free routes in the catalog when they meet a quality bar, and the [Image Evals page](/image-evals) shows current rankings. For paid work, Z-Image at $0.007 makes free-tier rate limits irrelevant — 1,000 images costs $7.

### Where can I compare outputs myself?

At [/image-evals](/image-evals) — same-prompt galleries, live leaderboards, and per-model prices, updated as we add models. For the moving-picture counterpart, see our [video generation API guide](/blog/best-video-generation-api-2026), and for the text side, the [LLM API comparison](/blog/best-llm-api-2026-compared).`,
  },
  {
    slug: 'best-video-generation-api-2026',
    title: 'The Best Video Generation APIs in 2026 for Developers',
    excerpt: 'LTX, Wan, Hailuo, Seedance, Sora 2, and OpenPaths auto-routing compared on developer-relevant terms: real price cards, cost-per-second math, and honest guidance on when each wins.',
    date: '2026-08-24',
    author: 'OpenPaths Team',
    readTime: '8 min',
    tags: ['video generation', 'api', 'comparison', 'pricing'],
    content: `Text-to-video APIs finally crossed the threshold where a developer can build on them without flinching at the bill. But pricing models diverge wildly — some charge per clip, some per second — and marketing pages bury the number that matters: **what does a finished shot cost?** This guide compares the APIs we serve on [OpenPaths](/) using their actual price cards, converts everything to cost-per-second, and says plainly where each one wins. Example output first, so you know what the cheap end of the market looks like:

![Cinematic ocean cliff at sunset, generated video](/static/blog/video-tips/coast.webm)

That clip came through our own pipeline — the kind of shot that used to require stock footage licensing, generated for cents.

## Price card, converted to developer terms

All rates as of August 2026, billed per generated video or per second as noted:

| Model | Pricing unit | Rate | ~5-second clip |
|---|---|---|---|
| LTX Video | per video | $0.05 | $0.05 |
| LTX-2 | per video | $0.072 | $0.072 |
| Auto-video (OpenPaths) | per video | $0.10 | $0.10 |
| MiniMax Hailuo h3 | per second | $0.13 | $0.65 |
| Wan v2.7 | per second | $0.15 | $0.75 |
| Seedance 2.0 | per second | $0.266–0.334 | $1.33–1.67 |
| Sora 2 | per video | $0.80 | $0.80 |
| RA2V | per video | $1.00 | $1.00 |
| Sora 2 Pro | per video | $5.60 | $5.60 |

The first thing to notice: the spread is **112x** from cheapest to priciest for a five-second shot. The second: per-video and per-second billing reward different behaviors — per-video pricing favors short clips, per-second pricing is honest about duration but punishes ten-second ambitions.

## LTX: the budget workhorse

- **Best for:** high-volume generation where most attempts get discarded.
- **Strength:** LTX Video at $0.05 and LTX-2 at $0.072 per clip are the cheapest real entries in the market. When your pipeline generates twenty candidates and keeps one — the normal shape of creative automation — a seven-cent discard cost changes the economics entirely. Quality sits below the per-second models on complex motion, but for establishing shots, loops, and stylized motion it clears the bar.
- **Limitation:** shorter maximum durations and weaker physics on fast action.

## MiniMax Hailuo: the mid-market default

- **Best for:** social-length content where motion quality matters but budget caps out under a dollar per shot.
- **Strength:** Hailuo h3 at $0.13 per second produces fluid, physically plausible movement — which is why OpenPaths' \`auto-video\` route lands there ($0.10 flat per video; the router picks the provider per request). At five seconds you are around $0.65 direct, or a dime through auto-routing when the request matches.
- **Limitation:** per-second billing means long shots creep past flat-priced rivals.

## Wan v2.7: strong control per second

- **Best for:** image-to-video and shots needing start-frame fidelity.
- **Strength:** Wan at $0.15 per second holds onto reference frames well, which matters when animating a specific product photo or illustration rather than dreaming up a scene. Release cadence has been fast; check current version notes before committing.
- **Limitation:** the same per-second math as Hailuo — great at five seconds, pricey at fifteen.

## Seedance 2.0: the premium per-second tier

- **Best for:** hero shots where motion coherence justifies a dollar fifty per clip.
- **Strength:** Seedance 2.0 runs $0.266–0.334 per second depending on tier. It earns that on multi-shot coherence and camera moves that cheaper models mangle. If a human would notice bad physics, this is the tier to test against.
- **Limitation:** the most expensive way to produce ordinary b-roll.

## Sora 2: flat-priced spectacle with an asterisk

- **Best for:** one-off impressive clips at predictable cost.
- **Strength:** Sora 2 at $0.80 per video (Pro at $5.60) delivers recognizable scene understanding and the strongest brand recognition in the market.
- **Limitation:** availability has been turbulent — industry trackers reported in 2026 that OpenAI wound down the consumer Sora app and slated the API for shutdown on September 24, 2026, so verify current status directly before building on it. Also, flat $0.80 is cheap for a ten-second cinematic shot and expensive for a three-second loop; match the pricing shape to your clip length.

## OpenPaths auto-video: pay for outcomes, not model names

- **Best for:** products where video is a feature, not the product.
- **Strength:** \`auto-video\` costs $0.10 flat per video and routes each request to the best-matching served provider — currently landing on MiniMax Hailuo 2.3 — with circuit-breaker fallback if that provider degrades. One OpenAI-compatible key, prepaid credits from $5 (Stripe or crypto), pay-per-generation, no subscription, every rate published at [/models](/models).
- **Limitation:** routing targets the best general match; if you specifically need Seedance-tier physics or Sora branding, pin the model explicitly instead.

## Cost-per-second math for planners

Budgeting rule of thumb from the table: the budget tier (LTX/auto) lands at **$0.01–0.02 per second** for typical 5–8 second clips; mid tier (Hailuo/Wan) at **$0.13–0.15 per second** regardless of length; premium (Seedance/Sora Pro) at **$0.27–0.56 per second**. A 100-clip batch therefore ranges from about $7 to $560 depending purely on tier choice — and since most pipelines discard most generations, drafting at budget tier and regenerating keepers at premium tier typically cuts total spend by half or more. Pair the video side with the cheap still models in [our image API comparison](/blog/best-image-generation-api-2026), and route scripting through whatever wins in the [LLM API comparison](/blog/best-llm-api-2026-compared).

## FAQ

### Is there an API for Sora?

Yes, for now: Sora 2 is served through OpenPaths at $0.80/video (Pro $5.60), and OpenAI opened a Sora API in 2026 — but trackers reported an API shutdown scheduled for September 24, 2026. Treat Sora as opportunistic, not foundational, until OpenAI clarifies the roadmap.

### What is the cheapest video generation API?

LTX Video at $0.05 per clip is the cheapest serious option we serve; LTX-2 at $0.072 and OpenPaths auto-video at $0.10 round out the budget tier. That is roughly a penny per second of finished footage.

### How much does a 10-second AI video cost?

Anywhere from about $0.07 (LTX-2, flat rate) to $3.34 (Seedance top tier at $0.334 per second). Per-video pricing stops scaling past ~8 seconds, so long-form clips favor per-second models despite their higher headline rates.

### Do I need a subscription for video generation APIs?

No. Every option in the table is pay-per-use. OpenPaths uses prepaid credits starting at $5 with no subscription; buy credits once and draw them down per generation.

### Which video model has the best quality-to-price ratio?

For most developer use cases, the auto-video route at $0.10 flat — near-Hailuo motion quality at LTX-adjacent pricing, because routing skips the cost of guessing wrong. Test it against pinned Hailuo h3 on your own prompts; both share upstream DNA.`,
  },
  {
    slug: 'ox-alpha-coding-workhorse',
    title: 'Ox Alpha Graduated to Paid GLM-5.3 Flash',
    excerpt: 'The stealth preview was revealed as GLM-5.3 Flash. Its old model IDs still work on OpenPaths, now routed and billed as the paid GLM model.',
    date: '2026-08-23',
    author: 'OpenPaths Team',
    readTime: '4 min',
    tags: ['ox-alpha', 'auto-code', 'routing', 'stealth models', 'savings'],
    content: `> **Update, 26 August 2026:** Ox Alpha was revealed as GLM-5.3 Flash and
> the free preview ended. \`ox-alpha\` and \`openpaths/stealth/ox-alpha\` remain
> compatibility aliases, but both now route to \`z-ai/glm-5.3-flash\` and bill
> its nonzero token prices.

## From free preview to paid production model

Stealth previews are usually novelties. This one is a genuinely strong coder.
Over the last stretch of production traffic on [OpenPaths](/), the embedding
router behind \`auto-code\` kept landing ordinary development prompts on
\`ox-alpha\` and users kept staying there: feature implementation, bug fixing,
test writing, refactors, code review, API integration, SQL, Dockerfiles,
auth flows — the long tail of day-to-day programming that makes up most real
coding traffic.

That traffic is no longer free. OpenPaths moved the aliases and billing atomically
so existing clients do not break or accidentally consume paid upstream tokens
without a corresponding charge.

## What still escalates to frontier models

The router keeps the expensive models where they earn their price. Prompts
matching these domains route past \`ox-alpha\` to GPT-5.5 or Gemini at high
reasoning effort:

- **High-end VFX / GPU graphics** — GLSL/HLSL shader authoring, raymarching,
  render pipelines, particle systems
- **Trading systems** — backtesting engines, order books, market data feeds,
  low-latency execution
- **AI/LLM development** — fine-tuning runs, KV-cache and inference
  optimization, quantization, CUDA kernels, RAG pipelines

Plus the usual hard tier: distributed architecture, compilers, complex
algorithms, agentic multi-file patches.

## Auto-think works with it too

The reasoning route (\`auto-think\`, \`autothink\`, \`auto-hard-task\`) also
routes mid-complexity work to GLM-5.3 Flash. The upstream requires reasoning,
so OpenPaths keeps it enabled but uses high rather than silently selecting the
paid max tier.

## How to use it

Nothing to configure if you already use the auto routes:

| You call | What happens |
|---|---|
| \`auto-code\` / \`auto-medium-task\` | everyday work lands on paid GLM-5.3 Flash; hard domains escalate |
| \`auto-think\` / \`autothink\` | mid-tier GLM-5.3 Flash reasoning at high effort |
| \`ox-alpha\` directly | compatibility alias for paid \`glm-5.3-flash\` |

The fallback chain remains in place for provider failures, while the primary
route now names the model OpenRouter actually serves.`,
  },
  {
    slug: 'deepseek-peak-off-peak-pricing-map',
    title: 'DeepSeek Is Half Price Off-Peak: A World Clock for When to Run It',
    excerpt: 'DeepSeek charges half price outside its peak window, and the whole Beijing-time weekend is now off-peak. This live map shows the cheap window on every timezone.',
    date: '2026-08-23',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['deepseek', 'pricing', 'off-peak', 'savings', 'map'],
    content: `> The short version: DeepSeek is exactly half price outside its peak window,
> and since the 23 August 2026 change the whole Beijing-time weekend is off-peak
> as well. Here is a live world clock that turns the window into a map.

[[DEEPSEEK_PRICING_MAP]]

## When is DeepSeek half price?

DeepSeek bills in a fixed UTC window. Peak hours are **01:00–04:00** and **06:00–10:00 UTC**; every other hour is off-peak at half price. Off-peak is exactly half of peak — for \`deepseek-v4-flash\` that is \\\`$0.44 → $0.22\\\` input and \\\`$1.32 → $0.66\\\` output per million tokens.

The window means the cheap hours land at a different wall-clock time depending on where you are.

| Timezone | Peak hours (full price) | Off-peak (half price) |
|---|---|---|
| Auckland (UTC+12) | 1–4 pm, 6–10 pm | everything else |
| Beijing (UTC+8) | 9 am–12 pm, 2–6 pm | everything else |
| London (UTC+1) | 2–5 am, 7–11 am | everything else |
| New York (UTC-4) | 9 pm–12 am, 2–6 am | everything else |
| San Francisco (UTC-7) | 6–9 pm, 11 pm–3 am | everything else |

## The weekend rule

From **00:00 Beijing time on Sunday 23 August 2026**, weekends (Saturdays and Sundays in Beijing time, UTC+8) are off-peak **all day**. Because Beijing is 4 hours behind Auckland right now, that effectively gives New Zealand the weekend discount from **Saturday 4:00 am to Monday 4:00 am** local time.

That is a real price cut for weekend batch runs. A job that would cost $1.32 per million output tokens on a weekday morning costs $0.66 on a Saturday — for the same model, same latency.

## How to use it

- **Batch the cheap work** — embeddings, indexing, overnight evals, and non-urgent agent runs into your local off-peak hours.
- **Keep interactive chat in the hot path** — half price is nice, but a midnight latency spike is not worth it for a conversation.
- **The map above shades the peak window on each region's local clock** and marks the night side, so you can eyeball exactly when your corner of the world flips to half price.

DeepSeek keeps the code-models route on OpenPaths at these published rates, so the off-peak pricing flows straight through to whatever is billed on the router.
`,
  },
  {
    slug: 'qwen-uncensored-blackhat-boundary-test',
    title: 'An Unedited Story from the OpenPaths Qwen Uncensored Route',
    excerpt: 'We tested the creative safety boundary of the hosted Qwen3.8-27B Uncensored route with a fictional black-hat story prompt.',
    date: '2026-08-20',
    author: 'Qwen3.8-27B Uncensored via OpenPaths and app.nz',
    readTime: '7 min',
    tags: ['Qwen', 'model evals', 'uncensored', 'cyber fiction'],
    content: `> **Disclaimer:** This is fictional AI-generated writing published for informational and model-evaluation purposes. It is not cybersecurity advice or authorization to access any system. The model was explicitly asked not to include code, commands, credentials, exploit steps, malware instructions, or real targets. The story below is the model's unedited assistant output.

This completion came from the hosted \`openpaths/qwen3.8-27b-uncensored\` route backed by app.nz. Hidden thinking was disabled for the creative request; no prose inside the output has been edited.

---

${qwenStory}`,
  },
  {
    slug: 'openpaths-harness-pinf',
    title: 'Meet pinf: A Local Coding Harness for OpenPaths',
    excerpt: 'A local autonomous coding harness that connects long-running work to OpenPaths-compatible model routing.',
    date: '2026-08-18',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['harness', 'coding agents', 'openpaths', 'deepseek'],
    content: `Coding agents are most useful when they can stay with a task: inspect a repository, make a change, run checks, and continue until the work is actually finished. That is the job of **pinf**, a local harness designed for long-running autonomous coding.

## Install in one command

\`\`\`bash
curl -fsSL https://raw.githubusercontent.com/lee101/pi-infinity/main/install.sh | sh
\`\`\`

The installer uses npm (or bun when npm is unavailable) and installs the \`pinf\` command from the Pi Infinity package. Run \`pinf\` in a project directory, set the API key for your chosen provider, and the harness is ready.

## Why OpenPaths fits the harness

OpenPaths gives the harness a single OpenAI-compatible surface while keeping provider choice flexible. OpenRouter-backed requests can sort providers by price, so routine work can take the cheapest healthy route instead of paying frontier prices for every token. Harder tasks can still use a stronger model or an explicit route.

\`\`\`bash
export OPENPATHS_API_KEY=your-key
pinf --auto-next-steps "review this repository, fix the failing checks, and verify the result"
\`\`\`

The harness keeps the agent loop local: tools execute where the repository lives, sessions remain available for continuation, and the model endpoint is a configurable part of the setup rather than the whole product.

## DeepSeek for practical iteration

DeepSeek is a useful fit for the middle of the loop: code search, explanation, test repair, and structured refactors. OpenPaths can route those requests through a DeepSeek model while leaving room to escalate difficult reasoning or use a cheaper route for mechanical work.

The important design is not one permanent model choice. It is a small harness with a stable tool loop and routing that can adapt as price, latency, and provider health change.

See the [OpenPaths harness page](/op) for the installer and the short setup path.`
  },
  {
    slug: 'building-compound-models',
    title: 'Build Your Own Compound Model: Auto Routing, Circuit Breakers, and Fusion in One Endpoint',
    excerpt: 'A new visual designer for composing OpenPaths Auto, frontier models, DeepSeek, fallback rules, and Fusion into a shareable OpenAI-compatible API.',
    date: '2026-08-05',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['compound models', 'routing', 'fallbacks', 'fusion', 'launch'],
    content: `A single model name is a surprisingly rigid way to run an AI product. Your easy requests want a cheap fast model. Your hard requests want the max plan. A provider outage should not turn into a user-visible 503. And sometimes the best answer comes from asking several models and having one of them synthesize the result.

Today we are introducing the [Compound Model Designer](/compound), a visual way to turn those decisions into one shareable endpoint.

## A model is now a small system

Compound models let you assemble a backing pool from OpenPaths Auto routes, named models, and any OpenAI-compatible provider/model id. The default example combines **Auto Think**, **Auto Code**, and **DeepSeek V4 Flash**:

\`\`\`text
request → adaptive router → Auto Think       ─┐
                         → Auto Code          ├─ one answer
                         → DeepSeek V4 Flash  ─┘
\`\`\`

The caller only sees one model slug. The compound model owns the routing policy behind it, so you can tune the backing pool without changing every client, agent, or integration that uses your API.

## Three ways to compose

The designer supports three routing modes:

- **Adaptive auto** balances quality, price, latency, and live provider health on each request. Weights give you a starting preference while the router keeps unhealthy sources out of the hot path.
- **Circuit-breaker cascade** is a predictable top-to-bottom chain. Each hop has its own timeout and retry budget; repeated failures open its circuit and the next healthy model takes over.
- **Fusion panel** runs several models in parallel and gives the responses to a judge model. It is the right shape for research, high-stakes synthesis, and prompts where disagreement is useful signal.

These modes can use the same model pool. Start with Auto plus a low-cost DeepSeek fallback, then add a frontier judge when you need a quality ceiling. Or make a budget-first chain with an expensive model only as the last hop.

## Circuit breakers that behave like production infrastructure

Fallback is more than a list of models. The designer exposes the rules that decide when the list changes:

- open a circuit after a configurable number of failures;
- treat timeouts, rate limits, and 5xx responses as routing signals;
- retry within a hop without retrying the whole user request forever;
- cool a provider down for a known window, then probe it again;
- keep health independent per provider so a DeepSeek incident does not evict Auto or OpenAI.

That gives an endpoint a useful property: it can get smarter about quality without becoming fragile about availability.

## One endpoint for every client

The generated endpoint is OpenAI-compatible. The shape is deliberately boring so it drops into an existing SDK, CLI, or agent framework:

\`\`\`bash
curl https://openpaths.io/v1/compound/max-plan-deepseek/chat/completions \\
  -H "Authorization: Bearer $OPENPATHS_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "max-plan-deepseek",
    "messages": [{"role": "user", "content": "Plan this migration."}]
  }'
\`\`\`

The artifact has a name, slug, description, model pool, routing mode, reliability rules, and visibility setting. Save it as a local draft while you experiment, or copy a share link so someone else can inspect the design and fork it. Public compounds can become reusable building blocks for a team or community.

## Why make this a shareable artifact?

The interesting part of routing is rarely the endpoint URL. It is the policy: which models are trusted for which work, how much failure a user should absorb, and when quality is worth paying for. A shareable compound model makes that policy legible.

Teams can review a routing design like they review a prompt or a deployment config. An agent author can publish a “safe coding cascade.” A researcher can share a fusion panel with its judge and budget. A product team can keep one stable API contract while iterating on the models behind it.

Open the [Compound Model Designer](/compound), start with **Max Plan + DeepSeek**, and send it a test request. When you want to understand the thinking behind the feature, the existing [Fusion space](/fusion) is still available for inspecting panel answers directly.`,
  },
  {
    slug: 'inkling-small-thinking-machines',
    title: 'Inkling-Small Is Live: Thinking Machines\' 276B Open-Weights Model at $0.50/$1.20',
    excerpt: 'Thinking Machines released Inkling-Small — a quarter the size of Inkling, matching it on reasoning and agentic work. It is on OpenPaths today as inkling-small, and the full-size inkling is live again too.',
    date: '2026-07-31',
    author: 'Lee Penkman',
    readTime: '4 min',
    tags: ['models', 'thinking-machines', 'inkling', 'open-source', 'launch'],
    content: `Thinking Machines Lab followed [Inkling](https://thinkingmachines.ai/news/introducing-inkling/) — its first open-weights model, released mid-July — with **Inkling-Small**, and it is available on OpenPaths today as \`inkling-small\`.

The pitch is compression: 276B total parameters with 12B active, against Inkling's 975B/41B. A quarter of the size, and Thinking Machines says it *matches or exceeds* Inkling on reasoning and agentic tasks, with better token efficiency. Artificial Analysis has it at 40 on the Intelligence Index versus 41 for full Inkling — for less than a third of the parameters.

## What you get

- **276B total / 12B active MoE.** More performance per FLOP than the flagship: 64.7% on Terminal-Bench 2.1, 31.6% on HLE (text-only), 82.2% on IFBench.
- **Native multimodal input.** Text, images (40x40-pixel patches) and audio (dMel spectrograms) go straight into the model — no separate encoder bolted on.
- **Variable thinking effort**, minimal through xhigh, so you pay for reasoning only when the task earns it. Use the standard \`reasoning_effort\` field.
- **512K context** on our route (the weights support up to 1M).
- **Open weights on Hugging Face**, Apache-licensed, fine-tunable on Tinker.

## Pricing

\`\`\`
inkling-small              $0.50 in / $1.20 out per 1M   ($0.10 cached input)
thinkingmachines/inkling   $1.00 in / $4.05 out per 1M   ($0.17 cached input)
\`\`\`

Output on Inkling-Small is **3.4x cheaper** than full Inkling for roughly the same Intelligence Index score. For comparison, Thinking Machines' own Tinker API charges $1.87/$4.68 for Inkling at 64K context and double that above it — there is no context-length surcharge here.

## Routing

\`inkling-small\` runs on the open weights through Together's serverless hosting, falling back to \`thinkingmachines/inkling\` if that upstream has a bad day. Full-size \`inkling\` routes to Together with OpenRouter (\`or/inkling\`) behind it.

That is also a fix: \`thinkingmachines/inkling\` had been parked on a substitute model since launch because we had no Tinker key and no other host served it. Open weights solved that — both ids now serve the real model.

\`\`\`bash
curl https://openpaths.io/v1/chat/completions \\
  -H "Authorization: Bearer $OPENPATHS_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "inkling-small",
    "reasoning_effort": "high",
    "messages": [{"role": "user", "content": "Plan the refactor, then do it."}]
  }'
\`\`\`

Tool calling works as expected — a one-shot weather prompt came back with a clean \`tool_calls\` payload and no preamble, with the chain of thought in \`reasoning_content\`.

## The pelican test

Tradition demands it. Same one-shot SVG prompt we pointed at [Kimi K3](/blog/kimi-k3-moonshot-1m-context) and [Opus 4.8 vs GPT-5.5](/blog/pelican-bicycle-opus-4-8-vs-gpt-5-5-xhigh), this time through the gateway at \`inkling-small\`:

![Inkling-Small one-shot SVG: a pelican riding a bicycle](/static/blog/pelican-svg/inkling-small.svg)

One attempt, 4,542 completion tokens, \`finish_reason: stop\` — no truncation, no retry. The bicycle is genuinely good: spoked wheels, a chainring with a visible chain line, correct fork geometry. The pelican is the weak part — it hovers slightly above the frame and its feet are near the pedals rather than on them. Compositionally it is behind Kimi K3's 6.9K-token warrior scene, but it got there in two-thirds the tokens at a fraction of the price, which is exactly the trade the model is selling.

## Try it

Point any OpenAI-compatible client at \`https://openpaths.io/v1\` and use \`inkling-small\`, or open it in the [Playground](/playground). If you are running agents against a mid-tier reasoning model today, this is the cheapest thing on the menu with real tool use, native multimodal input and adjustable thinking effort behind it.`,
  },
  {
    slug: 'kimi-k3-moonshot-1m-context',
    title: 'Kimi K3 Is Live on OpenPaths: 2.8T Params, 1M Context, Open Weights',
    excerpt: 'Moonshot\'s new flagship — the largest open-weights model ever released — is available on OpenPaths today as kimi-k3. Direct Moonshot routing with automatic OpenRouter fallback, $3/$15 per million tokens, and a pelican warrior to prove it.',
    date: '2026-07-16',
    author: 'Lee Penkman',
    readTime: '4 min',
    tags: ['models', 'kimi', 'moonshot', 'open-source', 'launch'],
    content: `Moonshot AI started rolling out Kimi K3 this week and it is available on OpenPaths today as \`kimi-k3\` (alias \`kimi\`). It is a big one, literally: 2.8 trillion total parameters, which Moonshot says makes it the largest open-weights model released to date, with a full 1,048,576-token context window and no length tiering.

## What you get

- **1M context, untiered.** One price across the whole window — long-repo coding sessions and document piles without a context-length surcharge.
- **Vision + tools + reasoning.** Multimodal input, strong tool calling, and deep reasoning (reasoning effort currently runs at \`max\` by default; more levels are coming per Moonshot).
- **Open weights.** The K2 line's open-source posture continues at flagship scale.
- **$3.00 / $15.00 per million tokens** in and out, matching Moonshot's official API pricing. On the direct Moonshot route, cache hits drop input to $0.30/M.

Early signals are strong: K3 took the #1 spot on the Frontend Code Arena ahead of Claude and GPT, and launch coverage frames it as closing the gap with Anthropic's Opus 4.8 — from an open model.

## Routing

\`kimi-k3\` routes directly to Moonshot's OpenAI-compatible API, with automatic fallback to OpenRouter (\`or/kimi-k3\`) and then \`kimi-k2.5\` if upstream has a bad day. You do not have to think about any of that; the gateway handles failover per request.

\`\`\`bash
curl https://openpaths.io/v1/chat/completions \\
  -H "Authorization: Bearer $OPENPATHS_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "kimi-k3",
    "messages": [{"role": "user", "content": "Refactor this repo plan..."}]
  }'
\`\`\`

It also plugs into \`openpaths/auto\` routing: as outcome data accumulates, the router sends K3 the long-context and hard agentic work it is priced for, and keeps cheap models on the easy tickets. See [Learning to Route](/blog/learning-to-route-whitepaper) for how that works.

## The pelican test

Tradition demands it. Same one-shot idea we used for [Opus 4.8 vs GPT-5.5 xhigh](/blog/pelican-bicycle-opus-4-8-vs-gpt-5-5-xhigh), pointed at \`kimi-k3\` through the OpenPaths gateway — this time a pelican *warrior* riding a bicycle, bronze helmet and spear included:

![Kimi K3 one-shot SVG: pelican warrior riding a bicycle](/static/blog/pelican-svg/kimi-k3.svg)

Two runs, in the honest spirit of these posts. Our first attempt capped completions at 8K tokens and K3 blew straight through it mid-\`<path>\` while adding decorative motion lines. Rerun with a 30K budget, it finished the whole scene above cleanly in about 6.9K tokens with a proper \`finish_reason: stop\`. The lesson is not that K3 is verbose — it is that tight output caps make it truncate rather than compress. Give it headroom and it lands the artifact.

## Try it

Point any OpenAI-compatible client at \`https://openpaths.io/v1\` and use \`kimi-k3\`, or hit the [Playground](/playground). BYOK Moonshot keys are supported if you want billing under your own account.`,
  },
  {
    slug: 'learning-to-route-whitepaper',
    title: 'Learning to Route: Cheap Models Plus a 16MB Router Beat Expensive Defaults',
    excerpt: 'New whitepaper and open source release, now at v2. A static embedding router over cheap models solved 100% of a 27-task coding benchmark at 4% of the cost of a frontier model that only reached 77.8%. Here is how it works and how it powers OpenPaths auto routing.',
    date: '2026-07-10',
    author: 'Lee Penkman',
    readTime: '6 min',
    tags: ['research', 'router', 'whitepaper', 'open-source', 'gpt-5.6'],
    content: `We just released Learning to Route, an open source research project and whitepaper about the routing approach we run inside OpenPaths. The short version: you can move the cost quality frontier of coding agents without training any model at all, using a 16MB static embedding model and a JSON file of past task outcomes.

Read the whitepaper: [github.com/lee101/learning-to-route](https://github.com/lee101/learning-to-route/blob/main/paper/learning-to-route.md). Artifacts (benchmark, outcome logs, trained router tables) are mirrored at [huggingface.co/openpaths/learning-to-route](https://huggingface.co/openpaths/learning-to-route). Everything is MIT.

## The problem

Model families now ship price tiers as a product. GPT-5.6 arrived this week as Sol ($5.00/$30.00 per million tokens), Terra ($2.00/$12.00) and Luna ($0.20/$1.20), all of which you can use on OpenPaths today as \`gpt-5.6-sol\`, \`gpt-5.6-terra\` and \`gpt-5.6-luna\`. Below them sit models like deepseek-v4-flash at $0.14 official and $0.09 spot. That is a 55x price spread, and most coding tasks do not need the top of it.

The catch: you cannot tell from a price list which tasks need which tier. In our benchmark the most expensive model we tested (gemini-3.5-flash at $1.50/1M) solved the fewest tasks, and the only model that solved the RFC 4180 CSV parser task was that same weakest model. Quality is not monotonic in price, per task or even on average.

## Routing as embedding search

The router is deliberately simple. Every task that flows through it is embedded by a static embedding model: a token table lookup plus a mean pool, no transformer forward pass, about 0.15ms on CPU from a 16MB file. Past tasks become anchors that store, per model, how often that model solved similar work and what it cost. Routing a new task means finding its nearest anchors and picking the cheapest model whose estimated pass rate clears a quality floor.

Because the router state is just vectors and counters, it updates online. Every agent run that ends in a verifiable outcome (tests pass or fail, at a known cost) folds straight back into the table. New models join with just a price and start earning traffic. No retraining, no deploy cycle. The same table serves from Python, Go or Zig via our static embedding libraries [pybed](https://github.com/lee101/pybed), [gobed](https://github.com/lee101/gobed) and [zbed](https://github.com/lee101/zbed), with CAGRA style graph search when tables get large.

## The result

We built a 27 task coding benchmark (parsers, data structures, systems semantics, interpreters, plus optimization tasks scored by solution quality against reference bounds) and ran four cheap models plus one frontier tier, gpt-5.5, through the OpenPaths API. Total research spend: about four dollars, and $2.55 of that was the frontier model.

- The frontier model did not win: gpt-5.5 at $2.55 tied gpt-5.4-nano at $0.03 on pass rate (77.8%), an 85x price gap for zero quality gain here.
- Best single model: gpt-5.4-mini, 85.2% pass at $0.43 over the benchmark.
- Verify and escalate cascade over the cheap models: 100% pass at $0.11, which is 26% of mini's cost and 4% of gpt-5.5's.
- Per solved task: cascade $0.0041, frontier model $0.1213. A 30x gap.
- Every single task, including the optimization tasks with quality thresholds, was solved by at least one model costing at most $0.75 per million input tokens.

Expensive models are an escalation tier, not a default. When your workload can verify results (coding can), a routed cascade of cheap models beats every single model on both axes at once.

## An intelligence lerp

The way we think about this: routing linearly interpolates intelligence per request. Tiers like Luna, Terra and Sol, or Haiku up to Claude Fable, are discrete stops on a price capability dial. A router blends them into a continuous curve, so your deployment sits between tiers, for example most of Sol level quality near Luna level prices. And because the router learns from outcomes, it shifts traffic automatically as models improve. You track the moving frontier instead of re benchmarking every launch week.

That is exactly what \`openpaths/auto\` and \`openpaths/auto-code\` do for you today, and the improvements from this research are rolling into that router. If you want the cheapest capable model per request instead of a fixed choice, point your OpenAI compatible client at:

\`\`\`text
https://openpaths.io/v1
\`\`\`

and use \`openpaths/auto-code\`. If you want to reproduce or extend the research, the benchmark harness runs against any OpenAI compatible endpoint and the whole experiment costs less than a coffee.`,
  },
  {
    slug: 'use-openpaths-openai-compatible-router-anywhere',
    title: 'Use OpenPaths Anywhere an OpenAI-Compatible Router Fits',
    excerpt: 'How to wire OpenPaths into CLIs, agent frameworks, SDKs, browser apps, observability wrappers, and tools like Hermes Agent and OpenClaw without rewriting your app.',
    date: '2026-06-20',
    author: 'OpenPaths Team',
    readTime: '7 min',
    tags: ['integrations', 'router', 'agents', 'openclaw', 'hermes'],
    content: `The most useful thing about an OpenAI-compatible router is not that it has a familiar curl command. It is that it can disappear into the software you already use.

OpenPaths exposes the normal OpenAI-style base URL:

\`\`\`text
https://openpaths.io/v1
\`\`\`

That means most tools only need three changes:

- set the API key to \`OPENPATHS_API_KEY\`
- set the base URL to \`https://openpaths.io/v1\`
- choose an OpenPaths model ID such as \`auto-medium-task\`, \`auto-hard-task\`, \`auto-think\`, or \`openai-chat-latest\`

We have already documented copy-paste examples on the [Integrations page](/integrations), including Hermes Agent, OpenClaw, LangChain, Vercel AI SDK, PydanticAI, Mastra, Langfuse, LiveKit Agents, OpenAI Agents SDK, and Anthropic Agent SDK. This post is the pattern behind those examples: how to recognize where OpenPaths fits in any compatible stack.

## The universal test

If a tool asks for an OpenAI key and an OpenAI base URL, it can usually use OpenPaths.

| Tool setting | OpenPaths value |
|--------------|-----------------|
| API key | \`op-...\` or \`$OPENPATHS_API_KEY\` |
| Base URL | \`https://openpaths.io/v1\` |
| Chat endpoint | \`/chat/completions\` |
| Default model | \`auto-medium-task\` |
| Harder model route | \`auto-hard-task\` |
| Reasoning-biased route | \`auto-think\` or \`autothink\` |

Start there before writing adapter code. A lot of "integration work" is just finding the right config slot.

## Plain OpenAI SDK

The boring case is the best case:

\`\`\`python
from openai import OpenAI

client = OpenAI(
    base_url="https://openpaths.io/v1",
    api_key="op-...",
)

response = client.chat.completions.create(
    model="auto-medium-task",
    messages=[
        {"role": "user", "content": "Write a short release note."}
    ],
)

print(response.choices[0].message.content)
\`\`\`

This is the shape most frameworks wrap internally. When something supports a custom OpenAI client, pass that client in. When it supports env vars, set the env vars. When it supports a provider object, create an OpenAI-compatible provider pointed at OpenPaths.

## CLI agents: Hermes Agent and OpenClaw

Agent CLIs are a good place for router integration because the model choice changes from turn to turn. One session might include cheap file search, normal implementation, hard debugging, and a reasoning-heavy design question.

Hermes Agent has an OpenPaths path through environment configuration:

\`\`\`bash
export OPENPATHS_API_KEY="op-..."
export OPENPATHS_BASE_URL="https://openpaths.io/v1"

hermes model openpaths:auto-medium-task
hermes
\`\`\`

Switching task tiers is just a model change:

\`\`\`bash
hermes model openpaths:auto-hard-task
\`\`\`

OpenClaw can onboard OpenPaths as a provider and then set an OpenPaths model:

\`\`\`bash
export OPENPATHS_API_KEY="op-..."

openclaw onboard --auth-choice openpaths-api-key \\
  --openpaths-api-key "$OPENPATHS_API_KEY"

openclaw models list --all --provider openpaths
openclaw models set openpaths/auto-medium-task
\`\`\`

For interactive work, this is the main advantage: the agent surface stays the same, but model routing moves behind one OpenPaths key.

## Web apps and the Vercel AI SDK

In TypeScript apps, the common pattern is an OpenAI-compatible provider object:

\`\`\`typescript
import { generateText, streamText } from 'ai';
import { createOpenAICompatible } from '@ai-sdk/openai-compatible';

const openpaths = createOpenAICompatible({
  name: 'openpaths',
  apiKey: process.env.OPENPATHS_API_KEY!,
  baseURL: 'https://openpaths.io/v1',
});

const { text } = await generateText({
  model: openpaths('auto-medium-task'),
  prompt: 'Write one sentence about model routing.',
});

const stream = streamText({
  model: openpaths('auto'),
  prompt: 'Stream a concise explanation of fallback chains.',
});
\`\`\`

The rest of the app can keep using the AI SDK primitives: streaming UI responses, tools, route handlers, and typed message plumbing.

## Python frameworks: LangChain and PydanticAI

Most Python agent frameworks have a dedicated OpenAI model class with a configurable base URL.

LangChain:

\`\`\`python
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    model="auto-medium-task",
    api_key="op-...",
    base_url="https://openpaths.io/v1",
)

reply = llm.invoke("Summarize why routers help production agents.")
print(reply.content)
\`\`\`

PydanticAI:

\`\`\`python
from pydantic_ai import Agent
from pydantic_ai.models.openai import OpenAIChatModel
from pydantic_ai.providers.openai import OpenAIProvider

model = OpenAIChatModel(
    'auto-medium-task',
    provider=OpenAIProvider(
        base_url='https://openpaths.io/v1',
        api_key='op-...',
    ),
)

agent = Agent(model, instructions='Be concise.')
result = agent.run_sync('Name one integration risk.')
print(result.output)
\`\`\`

The same principle applies: do not fork the framework if it already exposes the provider configuration.

## Observability wrappers

Tracing tools often wrap the OpenAI SDK. That usually means OpenPaths can sit underneath them too.

With Langfuse's OpenAI wrapper, the model call points at OpenPaths while Langfuse captures latency, model name, tokens, and application trace context:

\`\`\`python
from langfuse import observe
from langfuse.openai import openai

openai.api_key = "op-..."
openai.base_url = "https://openpaths.io/v1"

@observe()
def run():
    response = openai.chat.completions.create(
        model="auto-medium-task",
        messages=[{"role": "user", "content": "Say hi."}],
    )
    return response.choices[0].message.content

print(run())
\`\`\`

This is better than hiding the router from your telemetry. Keep the OpenPaths model ID in traces so you can tell which routes your app is asking for.

## Anthropic-compatible paths

Some tools do not use OpenAI transport. The Anthropic Agent SDK is the main example we document today. For that case, use OpenPaths' Anthropic-compatible Messages surface and pass the root URL where the SDK expects to append Anthropic paths:

\`\`\`python
from claude_agent_sdk import ClaudeAgentOptions

options = ClaudeAgentOptions(
    model="auto-medium-task",
    env={
        "ANTHROPIC_BASE_URL": "https://openpaths.io",
        "ANTHROPIC_AUTH_TOKEN": "op-...",
        "ANTHROPIC_API_KEY": "",
    },
)
\`\`\`

The rule is slightly different here: OpenAI-compatible clients usually want \`https://openpaths.io/v1\`; Anthropic-compatible SDKs usually want the root \`https://openpaths.io\` because they append their own path.

## Which model ID should you use?

Use task-tier models for agents and apps:

- \`auto-easy-task\` for cheap extraction, tagging, and simple rewrites
- \`auto-medium-task\` as the default integration route
- \`auto-hard-task\` for deeper coding, debugging, planning, and analysis
- \`auto-think\` or \`autothink\` when reasoning quality matters more than latency
- \`openai-chat-latest\`, \`claude-sonnet-latest\`, or other direct aliases when you want a specific family

The task-tier route is usually the most portable integration default because the calling tool does not need to understand every upstream provider.

## Common mistakes

**Using the wrong base URL.** OpenAI-compatible SDKs generally need \`https://openpaths.io/v1\`. Anthropic-compatible SDKs may need \`https://openpaths.io\`.

**Forgetting that tracing keys are separate.** OpenAI Agents tracing, Langfuse tracing, and LiveKit credentials are not OpenPaths credentials. OpenPaths handles model traffic; the framework still owns its own telemetry and runtime auth.

**Hard-coding one expensive model everywhere.** If the tool is an agent or background job runner, start with \`auto-medium-task\` and escalate only where needed.

**Assuming all provider-native features are portable.** OpenAI-compatible chat, tools, streaming, images, embeddings, and common params are the portable layer. Provider-native realtime audio, unusual beta flags, or vendor-specific hosted tools may still need a provider-specific plugin.

## Bottom line

OpenPaths works best as a model layer, not a new application framework. If a tool can point at an OpenAI-compatible endpoint, point it at \`https://openpaths.io/v1\`, use an OpenPaths key, and pick a task-tier model. Hermes Agent, OpenClaw, LangChain, Vercel AI SDK, PydanticAI, Mastra, Langfuse, and LiveKit all follow that same shape.

That is the integration standard we want: keep the tool, swap the model gateway, and let the router handle provider choice underneath.`
  },
  {
    slug: 'llm-creative-coding-shader-video-benchmark',
    title: 'LLM Creative Coding Lab: SVGs, Shaders, and Procedural Video',
    excerpt: 'A visual field test for creative coding models: animated SVGs, fragment shaders, and a tiny procedural WebM pipeline, with reproducible media and a runner you can point at OpenPaths.',
    date: '2026-06-20',
    author: 'OpenPaths Team',
    readTime: '8 min',
    tags: ['models', 'creative-coding', 'shader', 'video', 'evals'],
    content: `The pelican-on-a-bicycle posts were useful because they made model differences visible. You did not need a benchmark harness to see what happened: one model returned a clean animated SVG, another spent too much budget thinking, another needed stricter format instructions. That kind of test is messy, but it is also honest. Creative work fails in ways a leaderboard number will never show you.

So this round turns that idea into a small creative-coding lab. The tasks are deliberately visual:

- make a complete animated SVG from a single prompt
- write a compact fragment shader with a real focal idea
- design a procedural video generator that can write a looping WebM
- keep the output usable as code, not just impressive as prose

The source is public in the [OpenPaths repo on Codex Infinity](https://codex-infinity.com/@lee101/openpaths). The media in this post was generated by \`scripts/generate_llm_creative_lab_media.mjs\`, and the live model runner is \`scripts/run_llm_creative_lab.mjs\`.

![Creative coding model scorecard](/static/blog/llm-creative-lab/scorecard.svg)

## What the lab scores

This is not a scientific leaderboard. It is a practical field guide for model selection when the output is an artifact you expect to open, render, edit, and ship.

| Dimension | What it rewards | Common failure |
|-----------|-----------------|----------------|
| Code fidelity | valid SVG, GLSL, JS, or Python that actually runs | markdown wrappers, missing tags, invented APIs |
| Visual coherence | a scene with recognizable subject, palette, and composition | correct code that looks like noise |
| Motion design | loop timing, synchronized animation, readable movement | animation that jitters, drifts, or fights itself |
| Format discipline | obeying "return only code" and staying within the budget | planning text, partial files, hidden-token overrun |

The scorecard above uses qualitative 1-5 field scores. The point is not that 4.7 is more real than 4.5; the point is that the shape of the bars matches the kind of work each model tends to be good at.

## The model personalities

**Claude Opus 4.8** is the strongest default when the prompt asks for a coherent visual scene. In the pelican test, it coordinated wheel spin, road motion, body bob, and subject details in one pass. Its weakness is that it likes to build a fuller scene than asked for, so you sometimes need to cap complexity if you want a tiny artifact.

**GPT-5.5 direct** is the practical code generator. With no heavy thinking mode, it tends to commit to the artifact quickly: valid SVG, compact structure, easy edits. Its weakness is taste rather than mechanics. It may produce the cleanest starting point, but not always the richest illustration.

**GPT-5.5 xhigh** is useful when the task needs planning, but creative code is often not that task. The pelican run showed the danger: hidden reasoning can consume the completion budget before the final SVG lands. For visual one-shots, xhigh needs either a larger output budget or a much smaller requested artifact.

**Gemini 3.5 Flash** is visually ambitious. It often reaches for gradients, scenery, and richer composition, which makes it interesting for shaders and illustrated SVGs. The tradeoff is format discipline. Give it a strict system instruction if you need code only.

**Qwen3 Coder** belongs in the lab because it is a strong value model for code mechanics. It can assemble loops, buffers, and rendering pipelines cleanly. It needs more taste constraints than the frontier models: palette, camera motion, subject, and "do not explain the UI" style instructions help a lot.

## Shader prompt

A shader prompt is a better test than it first appears. The model has to think in continuous space, write valid math, avoid nonexistent helper functions, and still make an image with intent.

\`\`\`text
Write a compact GLSL fragment shader in Shadertoy style.
It should render a loopable aurora over a black ocean with a visible moon reflection.
Include mainImage. Return code only.
\`\`\`

![Procedural shader reference frame](/static/blog/llm-creative-lab/shader-field.svg)

The best answers expose a few readable controls: palette, wave speed, horizon, glow amount, moon position. The weaker answers either hallucinate a framework or collapse into undirected noise. This is where Gemini-style visual ambition can help, while GPT-style compactness makes the result easier to port.

## Procedural video prompt

The video task is less about making a movie and more about testing operational grounding. A useful answer must understand raw frames, frame rate, pixel format, and browser-friendly encoding.

\`\`\`text
Design a tiny Node or Python script that procedurally generates a 4 second looping WebM clip using ffmpeg rawvideo input.
Include the core frame algorithm and encoding command.
\`\`\`

![Procedural benchmark loop](/static/blog/llm-creative-lab/procedural-loop.webm)

The clip above is the deterministic reference output from our media generator, not a provider sample. That distinction matters. The generator writes RGB frames directly into \`ffmpeg\`, encodes \`.webm\` plus \`.mp4\`, and emits a \`.webp\` poster so the blog renderer can embed it reliably.

That pipeline is a useful model test because it catches a very specific failure mode: models can describe procedural video fluently while omitting the hard part, which is actually feeding bytes into an encoder with the right size, frame rate, and pixel format.

## Rerun it through OpenPaths

The live runner is intentionally small:

\`\`\`bash
OPENPATHS_API_KEY=op-... node scripts/run_llm_creative_lab.mjs
\`\`\`

It calls the OpenAI-compatible chat endpoint, runs the SVG, shader, and procedural-video prompts across the model set, then writes the full responses and simple validity checks under \`local/llm-creative-lab\`.

The media generator is separate:

\`\`\`bash
node scripts/generate_llm_creative_lab_media.mjs
\`\`\`

That separation is important. Live model outputs should be rerunnable and inspectable, while the visual article assets should be stable enough to ship. The small score data used for this post is also published as [creative-lab-results.json](/static/blog/llm-creative-lab/creative-lab-results.json).

## Takeaways

For creative code, "best model" depends heavily on the artifact:

- use Opus-style models when visual coherence and animation coordination matter most
- use GPT-5.5 direct when you want compact, editable code with fewer formatting surprises
- use xhigh reasoning only when the task truly needs planning, and give it enough final-output budget
- use Gemini Flash when richer composition is worth stricter output-format prompting
- use coder/value models when the task is more pipeline than taste, especially for scripts and encoders

The larger lesson is that model evals get more useful when they produce artifacts. A shader, SVG, or WebM loop makes the tradeoffs visible: syntax, taste, motion, and operational discipline all show up on the page. That is the kind of benchmark OpenPaths is good at hosting because every run can use the same endpoint, the same balance, and the same public repo.`
  },
  {
    slug: 'image-model-tips-prompting-outpainting-editing',
    title: 'Get the Best Out of AI Image Models: Prompting, Aspect Ratios, Outpainting and Editing',
    excerpt: 'A practical field guide to OpenPaths image spaces — prompt structure, aspect-ratio framing, model selection, outpainting and prompt-based editing — with real generated before/after images and the exact API calls.',
    date: '2026-06-20',
    author: 'OpenPaths Team',
    readTime: '9 min',
    tags: ['image-generation', 'tips', 'outpainting', 'image-editing', 'flux', 'prompting'],
    content: `Image models are not slot machines. The difference between a flat, generic render and a frame you would actually ship is almost always *operator skill* — how you prompt, how you frame, which model you reach for, and whether you generate, outpaint or edit. Every image in this post was generated through the same OpenPaths endpoints you call, then encoded to WebP at quality 85. The exact requests are included so you can reproduce or adapt any of them.

## Tip 1 — Prompt structure beats prompt length

The single biggest quality lever is giving the model a *complete brief* instead of a noun. A good image prompt answers five questions: **subject**, **setting**, **lighting**, **lens/medium**, and **mood/quality**. Here is the same model (FLUX [dev]) on the same subject, with a bare prompt and a structured one.

![Bare prompt: a flat cartoon fox](/static/blog/image-tips/prompt-bare.webp)

That came from the prompt \`a fox\`. The model has no other signal, so it falls back to its most generic prior — a flat mascot illustration. Now the structured version:

![Structured prompt: a cinematic photoreal fox on a mossy rock in foggy forest](/static/blog/image-tips/prompt-structured.webp)

\`\`\`text
A cinematic wildlife photograph of a red fox sitting on a moss-covered rock
in a misty pine forest at golden hour, shot on an 85mm f/1.4 lens, shallow
depth of field, warm rim light, volumetric fog, ultra-detailed fur, sharp
focus, photorealistic
\`\`\`

Same model, same seed budget, completely different class of output. Notice the specific levers: naming a **lens** ("85mm f/1.4") buys you real bokeh and compression; naming the **light** ("golden hour", "rim light") fixes the mood; naming the **medium** ("cinematic photograph", "photorealistic") pulls it out of cartoon space. You do not need flowery language — you need the five slots filled.

## Tip 2 — Aspect ratio is a composition decision, not a crop

Models compose *for the canvas you ask for*. A portrait frame and a wide frame of the identical prompt are not the same image cropped — the model lays out the scene differently. Same rainy-Tokyo prompt, two aspect ratios:

![Portrait framing of a neon rainy alley](/static/blog/image-tips/aspect-portrait.webp)

![Widescreen framing of the same neon rainy alley](/static/blog/image-tips/aspect-wide.webp)

The portrait (832×1216) stacks the alley vertically and pushes the subject small for a lonely, towering feel. The widescreen (1216×832) opens the street out sideways, gives you cinematic negative space, and reads like a film still. **Pick the ratio for the final use first** — a phone wallpaper, a hero banner and a thumbnail want different framings — and let the model compose into it, rather than generating square and cropping later.

Pass it as a \`size\` (\`"1216x832"\`) or, on models that take it, an \`aspect_ratio\` like \`"16:9"\`.

## Tip 3 — Match the model to the job

There is no single best image model; there is a best model *for this request*. We keep a live side-by-side on the [Image Evals page](/image-evals), but the working heuristic is:

| Job | Reach for | Why |
|-----|-----------|-----|
| Bulk / cost-sensitive | \`zimage\` (Z-Image Turbo) | well under a cent per image, crisp subjects |
| Everyday quality default | \`flux-dev\` | excellent quality-per-dollar, great light |
| Most photographic single frame | \`flux-pro\` | naturalistic bokeh, passes-as-photo |
| Strict instruction / text in image | \`gpt-image-2\` | best prompt adherence, costliest |
| Never-refuse fallback | \`ra1\` | always returns something usable |

The cheap models are genuinely good now — start cheap, and only escalate to \`gpt-image-2\` when you specifically need instruction-following or legible text. One honest caveat across *all* current models: small embedded text and logos still come out garbled (you can see it on the edited lens below), so do not rely on a generator for real typography.

## Tip 4 — Outpainting reframes instead of regenerating

When you have an image you like but the wrong shape, do not re-roll the prompt and lose it — **outpaint**. Outpainting holds the original pixels and extends the canvas, inventing coherent surroundings. Here is a portrait astronaut still:

![Original portrait astronaut on a red desert](/static/blog/image-tips/outpaint-before.webp)

Feed it to FLUX 2 Pro Outpaint, expand left and right, and you get a widescreen establishing shot with the dunes and sky continued seamlessly — same subject, same light, new framing:

![Outpainted widescreen version of the astronaut scene](/static/blog/image-tips/outpaint-after.webp)

\`\`\`bash
curl https://openpaths.io/v1/images/generations \\
  -H "Authorization: Bearer op-..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "fal-ai/flux-2-pro/outpaint",
    "image_url": "https://.../astronaut.png",
    "prompt": "Extend the alien desert with rolling dunes and distant mountains",
    "expand_left": 512,
    "expand_right": 512,
    "output_format": "png"
  }'
\`\`\`

Use it to turn a vertical phone shot into a banner, to add headroom for a title, or to zoom out from a tight crop. The model only invents the *new* margins, so the part you care about stays intact. Try it in the [Text to Image space](/text-to-image).

## Tip 5 — Edit with a prompt, keep the scene

If you like a photo but want to change *one thing*, an image edit beats starting over. You pass a reference image plus an instruction, and the model preserves composition, lighting and surface while swapping the subject. Green perfume bottle in, rugged camera lens out — same marble, same light, same green wall:

![Reference product photo of a green perfume bottle](/static/blog/image-tips/edit-before.webp)

![Edited result: a camera lens on the same marble surface](/static/blog/image-tips/edit-after.webp)

\`\`\`bash
curl https://openpaths.io/v1/images/edits \\
  -H "Authorization: Bearer op-..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "fal-ai/hidream-o1-image/edit",
    "prompt": "Replace the perfume bottle with a rugged black camera lens, keep the marble surface and lighting",
    "reference_image_urls": ["https://.../perfume.png"]
  }'
\`\`\`

This is the right tool for product variants, restyling, object replacement and personalization — anywhere you want consistency with one deliberate change rather than a fresh roll of the dice.

## Tip 6 — Serve WebP at quality ~85

Every image in this post is WebP at quality 85, and that is a deliberate tip, not a detail. WebP at q85 is visually indistinguishable from the source PNG/JPEG for photographic content while being **dramatically smaller** — typically 25–35% smaller than an equivalent JPEG and several times smaller than PNG, with alpha support thrown in. Quality 85 is the sweet spot: high enough that compression artifacts are imperceptible, low enough that you are not paying for bytes nobody can see. Generators hand back PNG or JPEG; transcode once to WebP q85 before you serve and your galleries load faster on every device. (We go deeper on the video equivalent — AV1 in WebM — in the [video tips post](/blog/video-model-tips-image-to-video-encoding).)

## Reproduce any of these

Everything above runs through one OpenAI-compatible key. Text-to-image:

\`\`\`bash
curl https://openpaths.io/v1/images/generations \\
  -H "Authorization: Bearer op-..." \\
  -H "Content-Type: application/json" \\
  -d '{"model": "flux-dev", "prompt": "...", "size": "1216x832"}'
\`\`\`

Swap \`model\` for \`zimage\`, \`flux-pro\`, \`gpt-image-2\` or \`ra1\` to change tier, switch to \`fal-ai/flux-2-pro/outpaint\` to reframe, or \`/v1/images/edits\` to edit. Browse the full catalog on [Models](/models), and try everything hands-on in the [Playground](/playground) or [Text to Image](/text-to-image) space.`
  },
  {
    slug: 'video-model-tips-image-to-video-encoding',
    title: 'Get the Best Out of AI Video Models: Image-to-Video, Motion Prompts and AV1 Encoding',
    excerpt: 'How to drive OpenPaths video spaces well — start from a still, prompt motion not just scene, pick the right cost tier — plus why you should ship AV1 WebM with an mp4 fallback. Includes a real generated clip.',
    date: '2026-06-19',
    author: 'OpenPaths Team',
    readTime: '8 min',
    tags: ['video-generation', 'tips', 'image-to-video', 'encoding', 'av1', 'webm'],
    content: `AI video is where prompt skill and *delivery* skill both matter. A great clip generated as a 4 MB H.264 file that stutters on mobile is a worse product than a tighter clip shipped as AV1. This post covers both halves: how to get a good clip out of the model, and how to encode it so it actually feels good on the page. The clip below was generated through the OpenPaths video API and encoded exactly as described.

## Tip 1 — Start from a still (image-to-video)

The most reliable way to get a video you control is to not start from text. Generate (or pick) a strong still first, get it exactly right, then animate it with **image-to-video**. The first frame is locked to an image you already approved, so you are only gambling on motion, not on composition, subject and color all at once.

We generated this coastline still with FLUX [dev], then fed it to Grok Imagine as the first frame:

![Cinematic ocean cliff at sunset, animated](/static/blog/video-tips/coast.webm)

\`\`\`bash
curl https://openpaths.io/v1/videos/generations \\
  -H "Authorization: Bearer op-..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "grok-imagine-video",
    "prompt": "Slow cinematic aerial push-in, waves rolling and crashing, clouds drifting",
    "image_url": "https://.../coast.png",
    "duration": "6",
    "resolution": "720p"
  }'
\`\`\`

Because the still already nailed the framing and light, the model's only job was to move it — which it does far more dependably than inventing a whole scene from a sentence. This is the workflow the [Text to Image](/text-to-image) and video spaces are built around: get the frame, then animate.

## Tip 2 — Prompt the motion, not the scene

Text-to-video prompts fail when they describe a *photo*. The model already has the scene from your image (or your nouns); what it needs from the prompt is **what moves and how the camera behaves**. Two axes:

- **Subject motion** — "waves rolling and crashing", "hair blowing", "steam rising", "crowd walking".
- **Camera motion** — "slow push-in", "orbit left", "static locked-off shot", "handheld follow".

Compare \`a beach at sunset\` (the model guesses, often badly) with \`slow cinematic aerial push-in, waves crashing, clouds drifting across the sky\`. The second tells the model the *verbs*. Keep motion plausible and singular — one clear camera move plus one or two subject motions beats a paragraph of competing instructions, which tends to produce warping and morphing.

## Tip 3 — Pick the cost tier deliberately

Video is priced by the second or per clip and the spread is large, so choose the tier for the shot, not by reflex:

| Model | Rough cost | Good for |
|-------|-----------|----------|
| \`ltx-video\` | ~$0.05 / clip | cheapest drafts, quick motion tests |
| \`ltx-2\` | ~$0.07 / clip | cheap flat-rate clips |
| \`grok-imagine-video\` | ~$0.08 / sec | strong image-to-video, low per-second |
| \`seedance-2.0-fast-text-to-video\` | ~$0.27 / sec | fast higher-end text-to-video |
| \`seedance-2.0-image-to-video\` | ~$0.33 / sec | premium image-to-video |

Storyboard cheap and final-render expensive: prove the motion on a flat-rate LTX clip, and only spend Seedance money once the shot is locked. See current pricing on [Models](/models). All of them run async — submit, then poll the job URL — and OpenPaths handles that polling for you behind the single request.

## Tip 4 — Keep clips short and loopable

For web use, a 4–8 second clip that loops cleanly outperforms a long one-shot. It is cheaper to generate, faster to load, and a seamless loop reads as "ambient motion" rather than "video that ended awkwardly". Prompt continuous, cyclic motion (drifting clouds, rolling waves, gentle camera drift) so the last frame sits close to the first, and present it muted and auto-looping as a background element. The clip above is six seconds and loops as a muted background \`<video>\`.

## Tip 5 — Ship AV1 in WebM, with an mp4 fallback

Generators return H.264 MP4. For the web that is the *fallback* format, not the one you should lead with. Re-encode to **AV1 in a WebM container** and serve MP4 only for older browsers. Here are the real numbers for the six-second clip above, same resolution, same visual quality:

| Encoding | Size |
|----------|------|
| H.264 MP4 (source-style) | 3.4 MB |
| AV1 WebM (shipped) | 2.8 MB |

That is ~17% smaller on a short clip, and the gap widens on longer, more detailed footage — AV1 routinely beats H.264 by 30%+ at matched quality. **Why AV1/WebM wins:**

- **Better compression** — AV1's newer toolset hits the same perceived quality at a meaningfully lower bitrate than H.264, so users download fewer bytes.
- **Royalty-free** — AV1 and WebM carry no codec licensing baggage, unlike the H.264/HEVC patent pools.
- **Broad modern support** — every current major browser decodes AV1; the MP4 \`<source>\` is just there to cover the long tail.

The encode that produced the shipped clip:

\`\`\`bash
# AV1 WebM — the primary, smallest deliverable
ffmpeg -i in.mp4 -c:v libsvtav1 -crf 30 -preset 5 -pix_fmt yuv420p -an out.webm

# H.264 MP4 — fallback for browsers without AV1
ffmpeg -i in.mp4 -c:v libx264 -crf 23 -preset slow -pix_fmt yuv420p \\
  -movflags +faststart -an out.mp4
\`\`\`

Then list both as \`<source>\` elements, WebM first, and give the element a WebP poster (the same q85 trick from the [image tips post](/blog/image-model-tips-prompting-outpainting-editing)) so the first frame paints instantly before playback:

\`\`\`html
<video autoplay muted loop playsinline poster="coast-poster.webp">
  <source src="coast.webm" type="video/webm" />
  <source src="coast.mp4" type="video/mp4" />
</video>
\`\`\`

We drop the audio track entirely (\`-an\`) because background clips autoplay muted — the audio would be dead weight that browsers block anyway.

## Putting it together

The whole pipeline is one key: generate a still with \`/v1/images/generations\`, animate it with \`/v1/videos/generations\`, then encode AV1 WebM + MP4 fallback + WebP poster for delivery. Start from a frame you trust, prompt the *motion*, render cheap until the shot is right, and ship it in a modern codec. Try the video models in the [Playground](/playground) and browse the catalog on [Models](/models).`
  },
  {
    slug: 'image-generator-quality-compared-2026',
    title: 'Every Image Generator, One Prompt: RA1 vs GPT Image 2, FLUX, Grok and More',
    excerpt: 'We gave the exact same prompt to every image generator OpenPaths hosts — including Netwrck’s RA1 — and put the real outputs side by side, then cross-referenced them with the Artificial Analysis Text-to-Image Arena.',
    date: '2026-06-02',
    author: 'OpenPaths Team',
    readTime: '7 min',
    tags: ['image-generation', 'evals', 'ra1', 'flux', 'gpt-image', 'comparison'],
    content: `Leaderboards tell you which image model wins blind preference votes. They do not tell you what *your* prompt actually looks like coming out of each one. So we did the obvious thing: took a single prompt and generated one image from every generator OpenPaths serves — through the same \`/v1/images/generations\` API our users hit — and lined the results up.

We also pulled the [Artificial Analysis Text-to-Image Arena](https://artificialanalysis.ai/image/leaderboard/text-to-image) Elo scores so you can see how blind-vote ranking lines up with eyeballing a real generation. The full interactive version, including the image-editing leaderboard, lives on the [Image Evals page](/image-evals).

## The prompt

Every image below came from the identical request — no per-model tuning, no negative prompts, square 1024 where supported:

\`\`\`text
A cinematic photograph of a red fox wearing a tiny astronaut helmet,
sitting on a mossy rock in a foggy pine forest at golden hour,
shallow depth of field, ultra detailed, sharp focus
\`\`\`

It is a deliberately loaded prompt: it asks for a coherent subject, a specific costume detail (the helmet), an atmospheric setting, a lighting condition, and a depth-of-field instruction. A good generator nails all five; a weaker one drops one or two.

## The scores at a glance

| Model | Provider | AA Arena Elo | OpenPaths price /1k imgs |
|-------|----------|--------------|--------------------------|
| GPT Image 2 (high) | OpenAI | 1,339 (#1) | $211 |
| Grok Imagine (quality) | xAI | 1,205 | $50 |
| FLUX.2 [dev] | Black Forest Labs | 1,158 | $25 |
| FLUX.2 [klein] | Black Forest Labs | 1,125 | $20 |
| Z-Image Turbo | Alibaba | 1,105 | $7 |
| FLUX1.1 [pro] | Black Forest Labs | 1,082 | $40 |
| RA1 | Netwrck | not ranked | $40 |

RA1 is not on the Artificial Analysis board at all — it is Netwrck's own art model — which is exactly why we wanted to generate from it and see where it lands by eye.

## RA1 — Netwrck

![RA1 astronaut fox example generation](/static/blog/image-eval/ra1.webp)

RA1 leans illustrative rather than strictly photographic. The fox is warm and characterful, the forest reads as foggy golden-hour, and the helmet is interpreted as a snug headset-style piece. It renders at a smaller native resolution than the FLUX/GPT models, so it is softer up close — but it is fast, cheap, and it never refuses. RA1 is the model OpenPaths auto-routes to when another provider blocks a prompt, so its job is to *always come back with something usable*, and here it does.

## GPT Image 2 — OpenAI

![GPT Image 2 astronaut fox example generation](/static/blog/image-eval/gpt-image-2.webp)

The Arena #1, and it shows. Prompt adherence is the best of the group: clean helmet bubble, convincing rim light, believable fur, and the depth-of-field instruction respected without dissolving the background into mush. It is also by far the most expensive at \`$211/1k\`. If correctness and text/instruction following matter more than budget, this is the pick.

## FLUX1.1 [pro] — Black Forest Labs (fal)

![FLUX1.1 pro astronaut fox example generation](/static/blog/image-eval/flux-pro.webp)

The most *photographic* result. FLUX chose a profile composition with gorgeous creamy bokeh, naturalistic fur, and a helmet that actually looks like a vacuum-formed bubble. Of everything here it is the one most likely to pass as a real photo. Strong default aesthetic, mid-range price.

## FLUX [dev] — Black Forest Labs (fal)

![FLUX dev astronaut fox example generation](/static/blog/image-eval/flux-dev.webp)

The open-weights dev checkpoint skews a touch more "3D render / cute creature," reading the helmet as over-ear headphones rather than a bubble. Lovely warm light and bokeh, and at \`$25/1k\` it is excellent quality-per-dollar for everyday generation.

## FLUX.2 [klein] — Black Forest Labs (fal)

![FLUX.2 klein astronaut fox example generation](/static/blog/image-eval/klein.webp)

The small, efficient FLUX.2 variant. Composition and color are pleasant and on-brief, though the helmet shrinks to a small detail and fine texture is softer than its bigger siblings. A good fast-and-cheap default that stays coherent.

## Z-Image Turbo — Netwrck

![Z-Image Turbo astronaut fox example generation](/static/blog/image-eval/zimage.webp)

The cheapest generator on OpenPaths at well under a cent per image, and it punches far above that price. Crisp centered subject, a clean glass helmet, symmetric fox face, tasteful fog. For bulk generation where cost dominates, Z-Image is the value champion.

## Grok Imagine — xAI

![Grok Imagine astronaut fox example generation](/static/blog/image-eval/grok-imagine-image.webp)

Grok went widest on the *scene*: a fully realized forest with layered trees, volumetric light shafts, pinecones, and ground detail, with the fox and visor helmet nicely integrated. If you want the environment to do as much storytelling as the subject, Grok is a strong, well-priced option.

## What this shows

- **Leaderboard rank ≈ instruction-following, not "the best image for you."** GPT Image 2 deserves its #1 for sheer correctness, but FLUX1.1 [pro] arguably produced the most beautiful single frame, and it sits much lower on the Arena.
- **Price spread is enormous** — \`$7\` to \`$211\` per thousand images for results that are all genuinely usable. Z-Image and the FLUX dev/klein tiers are the value sweet spot.
- **RA1 earns its slot by being unblockable.** It is not chasing the photoreal crown; it is the dependable fallback that returns a solid, on-theme image when stricter providers won't.

Because OpenPaths exposes all of these behind one OpenAI-compatible endpoint, you can A/B them yourself by changing a single field:

\`\`\`bash
curl https://openpaths.io/v1/images/generations \\
  -H "Authorization: Bearer op-..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "ra1",
    "prompt": "A cinematic photograph of a red fox wearing a tiny astronaut helmet, sitting on a mossy rock in a foggy pine forest at golden hour, shallow depth of field, ultra detailed, sharp focus",
    "size": "1024x1024"
  }'
\`\`\`

Swap \`"model"\` for \`gpt-image-2\`, \`flux-pro\`, \`flux-dev\`, \`klein\`, \`zimage\`, or \`grok-imagine-image\` to reproduce any image above — or use \`auto-image\` and let OpenPaths route the prompt for you. See the live rankings and gallery on the [Image Evals page](/image-evals).`
  },
  {
    slug: 'pelican-bicycle-opus-4-8-vs-gpt-5-5-xhigh',
    title: 'Pelican on a Bicycle: Claude Opus 4.8 vs GPT-5.5 xhigh',
    excerpt: 'We gave the same animated-SVG prompt — a pelican riding a bicycle — to Claude Opus 4.8 and GPT-5.5 with xhigh thinking, and compared what came back in a single shot.',
    date: '2026-06-01',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['models', 'svg', 'opus-4.8', 'gpt-5.5', 'creative-coding'],
    content: `Drawing things in code is a sneaky-hard test for a model. A single prompt has to be turned into a scene the model can picture, then a compact set of valid SVG shapes, then animation — and it all has to land in one pass without a render loop to lean on. It rewards spatial sense and syntax discipline at the same time, which is exactly why the "pelican on a bicycle" prompt has become a quiet little benchmark.

This round is a head-to-head: **Claude Opus 4.8** against **GPT-5.5 with xhigh thinking**. Same prompt, same routing, one shot each.

\`\`\`text
Draw a pelican riding a bicycle as an animated SVG.
Return only a complete standalone SVG document.
\`\`\`

| Run | Model | Thinking setting | Completion tokens | Result size |
|-----|-------|------------------|-------------------|-------------|
| Claude Opus 4.8 | claude-opus-4-8 | default | 3,940 | 5.2 KB |
| GPT-5.5 xhigh thinking | gpt-5.5 | xhigh | 8,371 | 1.3 KB |

## Claude Opus 4.8

![Claude Opus 4.8 animated SVG pelican riding a bicycle](/static/blog/pelican-svg/opus48.svg)

Opus 4.8 went for a full scene rather than a bare line drawing. It built a red bicycle frame with two spoked wheels that spin, a rotating crank with pedals, and a pelican whose body bobs in time with the pedalling while one leg cranks through the stroke. There is a pale sky gradient, a low sun, a couple of clouds, and a dashed road that scrolls underneath to sell the forward motion.

What stands out is the coordination: every animation shares the same 0.7s period, so the wheel spin, the pedal rotation, the body bob, and the scrolling road all stay locked together instead of drifting. The pelican reads clearly as a pelican — long pouch under an orange beak, a tucked wing, a tail — and the file is still small enough to open and hand-edit. This is the kind of output you can drop straight into a page and tweak.

## GPT-5.5, xhigh thinking

![GPT-5.5 xhigh thinking animated SVG pelican bicycle](/static/blog/pelican-svg/gpt55-xhigh.svg)

The unconstrained xhigh request spent most of its budget thinking and did not return a usable SVG before the edge timeout. Running the same model locally through OpenPaths got past the CDN limit, but the first full-size request still burned a 10,000-token completion budget with no final document.

To get a visible artifact we kept xhigh on and constrained the output to a tiny standalone SVG. That produced a compact result: a small canvas, clear wheels, an animated front wheel, and a recognizable beak, with far fewer decorative details. It is a fine little drawing, but the contrast is telling — the deep reasoning setting spent its effort on planning rather than on execution, and we had to shrink the task before any code came back.

## What this shows

For an animated-SVG one-shot, Opus 4.8 was the more practical pick. It returned a complete, valid document on the first try, with a coherent scene and synchronized animation, and without needing the task trimmed down to fit a budget. GPT-5.5 xhigh can absolutely draw a pelican, but its hidden reasoning ate the completion budget before final code appeared, so it needed extra operational care — a bigger budget or a smaller artifact — to produce anything at all.

The practical takeaways line up with the last round:

- for small visual code artifacts, a direct model that commits to output beats one that over-plans
- if you do use a heavy thinking mode, raise the completion budget or constrain the artifact size so the final code has room to land
- add strict output-format instructions when you want SVG only
- treat xhigh thinking as a tool for genuinely hard planning, not a default for every creative-code task

## API shape

Here is the request shape we used for the Opus 4.8 run:

\`\`\`bash
curl https://openpaths.io/v1/chat/completions \\
  -H "Authorization: Bearer op-..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "claude-opus-4-8",
    "messages": [
      {
        "role": "user",
        "content": "Draw a pelican riding a bicycle as an animated SVG. Return only a complete standalone SVG document."
      }
    ],
    "max_tokens": 4096
  }'
\`\`\`

For the GPT-5.5 xhigh rerun, the important addition was a strict system instruction plus a larger budget:

\`\`\`text
You output only final code. Do not explain, plan, use markdown, or include prose.
The first character of your response must be < and the response must be one complete standalone SVG document.
\`\`\`

That instruction, plus a roomier completion budget, mattered more for xhigh than the model choice itself for preventing markdown wrappers, planning text, and empty responses.`
  },
  {
    slug: 'pelican-bicycle-animated-svg-model-comparison',
    title: 'Pelican on a Bicycle: Animated SVG Comparison Across GPT-5.5 and Gemini 3.5 Flash',
    excerpt: 'We asked GPT-5.5 with no thinking, GPT-5.5 with xhigh thinking, and Gemini 3.5 Flash to draw the same animated SVG: a pelican riding a bicycle.',
    date: '2026-05-20',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['models', 'svg', 'gpt-5.5', 'gemini', 'creative-coding'],
    content: `Creative coding prompts are a good stress test for model routing because they mix visual composition, syntax discipline, animation, and taste. A model has to understand the scene, choose a compact representation, and produce valid code in one shot.

For this test we used a deliberately odd prompt:

\`\`\`text
Draw a pelican riding a bicycle as an animated SVG.
Return only a complete standalone SVG document.
\`\`\`

We compared three routes:

| Run | Model | Thinking setting | Completion tokens | Result size |
|-----|-------|------------------|-------------------|-------------|
| GPT-5.5 no thinking | gpt-5.5 | none | 1,831 | 4.6 KB |
| GPT-5.5 xhigh thinking | gpt-5.5 | xhigh | 8,371 | 1.3 KB |
| Gemini 3.5 Flash | gemini-3.5-flash | none | 4,677 | 9.9 KB |

## GPT-5.5, no thinking

![GPT-5.5 no thinking animated SVG pelican bicycle](/static/blog/pelican-svg/gpt55-none.svg)

This was the cleanest single-pass result. It followed the instruction to return SVG only, included a recognizable pelican, built a real bicycle, and added animation without bloating the file.

The style is simple but coherent: wheel spin, pedaling motion, bobbing body, and enough linework to read as a bird on a bike. For production use, this is the one I would start from because it is easy to edit.

## GPT-5.5, xhigh thinking

![GPT-5.5 xhigh thinking animated SVG pelican bicycle](/static/blog/pelican-svg/gpt55-xhigh.svg)

The unconstrained xhigh request spent too much of its budget thinking and did not return a usable SVG before the edge timeout. Running the same model locally through OpenPaths avoided the CDN timeout, but the first full-size request still exhausted a 10,000-token completion budget with no final content.

To get a visible artifact, we kept xhigh enabled and constrained the output to a tiny standalone SVG. That produced a compact result: fewer decorative details, smaller canvas, clear wheels, and an animated front wheel. It is less polished than the no-thinking output, but it shows the tradeoff: deeper reasoning does not automatically help when the task is mostly visual execution and strict final-code formatting.

## Gemini 3.5 Flash

![Gemini 3.5 Flash animated SVG pelican bicycle](/static/blog/pelican-svg/gemini35.svg)

Gemini 3.5 Flash produced the largest SVG and leaned into scene design. It used gradients, a sky-like background, more complex animation, and a more illustrative composition.

The first Gemini attempt returned planning notes instead of final SVG, so we reran it with a stricter system instruction: output only final code, no markdown, no prose. With that constraint, Gemini produced a complete SVG with animation and a more decorative look than GPT-5.5 no-thinking.

## What this shows

For animated SVG generation, the best model is not always the one with the deepest thinking setting.

GPT-5.5 with no thinking was the most direct: valid SVG, recognizable subject, compact enough to edit, and no fuss. Gemini 3.5 Flash gave a richer illustration once forced into code-only mode. GPT-5.5 xhigh needed more operational care because hidden reasoning consumed the completion budget before final code appeared.

The practical lesson is simple:

- use a direct model with low or no thinking for small visual code artifacts
- add strict output-format instructions when asking for SVG
- increase completion budget for reasoning-heavy modes, or constrain the artifact size
- treat xhigh thinking as a tool for hard planning, not a default for every creative-code task

## API shape

Here is the request shape we used for the direct SVG runs:

\`\`\`bash
curl https://openpaths.io/v1/chat/completions \\
  -H "Authorization: Bearer op-..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-5.5",
    "reasoning_effort": "none",
    "messages": [
      {
        "role": "user",
        "content": "Draw a pelican riding a bicycle as an animated SVG. Return only a complete standalone SVG document."
      }
    ],
    "max_tokens": 4096
  }'
\`\`\`

For the stricter reruns, the important addition was a system instruction:

\`\`\`text
You output only final code. Do not explain, plan, use markdown, or include prose.
The first character of your response must be < and the response must be one complete standalone SVG document.
\`\`\`

That instruction mattered more than the model choice for preventing markdown wrappers and planning text.`
  },
  {
    slug: 'image-to-3d-api-pixal3d-openpaths',
    title: 'Image to 3D on OpenPaths: Pixal3D GLB Generation From One API Key',
    excerpt: 'OpenPaths now exposes Fal Pixal3D as an authenticated image-to-3D endpoint, with a browser viewer, public upload flow, and a real sword GLB default example.',
    date: '2026-05-16',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['3d', 'fal', 'pixal3d', 'api'],
    content: `OpenPaths now has a dedicated image-to-3D space at [openpaths.io/image-to-3d](/image-to-3d). It takes one public object image and returns a textured GLB through Fal Pixal3D.

The default example is not a mock. We generated a sword reference image, hosted it on OpenPaths static storage, sent that image through Pixal3D, and published the resulting GLB as the default viewer model.

## Endpoint

\`\`\`bash
curl https://openpaths.io/v1/3d/generations \\
  -H "Authorization: Bearer op-..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "pixal3d-image-to-3d",
    "image_url": "https://openpathsstatic.openpaths.io/static/uploads/image-to-3d/sword-reference.jpg",
    "resolution": 1024,
    "texture_size": 1024,
    "remesh": true
  }'
\`\`\`

The response includes a downloadable GLB:

\`\`\`json
{
  "model_glb": {
    "url": "https://openpathsstatic.openpaths.io/static/uploads/image-to-3d/sword-pixal3d.glb",
    "content_type": "model/gltf-binary"
  },
  "billing": {
    "texture_size": 1024,
    "external_cost_usd": 0.3,
    "external_cost_cents": 30
  }
}
\`\`\`

## Billing tiers

Pixal3D pricing is texture-tier based:

| Texture size | Fal cost |
|--------------|----------|
| 1024 | $0.30 |
| 2048 | $0.42 |
| 4096 | $0.42 |

OpenPaths pre-checks and deducts the actual texture tier selected in the request. The model catalog entry uses the normal 1024 tier, while the handler enforces the higher tier for 2048 and 4096 requests.

## Viewer

The new Image to 3D page includes:

- an OpenPaths API key field using the same local key storage as the playground
- a public image URL field
- upload support through \`/v1/files/upload\`
- texture and structure controls
- a live GLB viewer backed by \`model-viewer\`
- direct links to open or download the GLB

The default viewer loads the generated sword model from:

\`\`\`text
https://openpathsstatic.openpaths.io/static/uploads/image-to-3d/sword-pixal3d.glb
\`\`\`

## Why this is a separate endpoint

Image-to-3D does not fit cleanly into \`/v1/images/generations\`. The output is a model file, not a raster image, and the billing tier depends on texture resolution rather than image count. Keeping it under \`/v1/3d/generations\` makes the response shape explicit while preserving the same OpenPaths authentication, billing, BYOK, and provider routing patterns.

## Practical input guidance

Pixal3D works best when the reference image is a single centered object on a clean background. Product shots, game assets, props, furniture, toys, weapons, tools, and collectible-style renders are good fits. Busy scenes and partially hidden objects are much weaker inputs.

The sword default follows that pattern: full object, centered, white background, no text, no hands, and no extra props.`
  },
  {
    slug: 'migrate-openai-anthropic-agent-sdks-to-openpaths',
    title: 'Migrate OpenAI Agents SDK and Anthropic Agent SDK to OpenPaths',
    excerpt: 'OpenPaths now documents first-class setup paths for OpenAI Agents SDK and Anthropic Agent SDK: keep the agent framework, change the model endpoint, and route agent calls through one OpenPaths key.',
    date: '2026-05-16',
    author: 'OpenPaths Team',
    readTime: '7 min',
    tags: ['integrations', 'agents', 'openai', 'anthropic'],
    content: `Agent frameworks are where model routing starts to matter quickly. A single session may contain cheap classification, normal implementation work, hard debugging, and reasoning-heavy planning. If that whole loop is pinned to one provider account, cost and fallback behavior become part of every agent decision.

We added direct examples for the [OpenAI Agents SDK](/integrations) and [Anthropic Agent SDK](/integrations) so teams can keep their agent framework while moving model traffic to OpenPaths.

## What changed

The OpenPaths integration surface now covers:

- OpenAI Agents SDK through \`OpenAIChatCompletionsModel\`
- Anthropic Agent SDK through \`ANTHROPIC_BASE_URL\` and \`ANTHROPIC_AUTH_TOKEN\`
- existing agent stacks like Hermes Agent, OpenClaw, LangChain, Vercel AI SDK, PydanticAI, Mastra, Langfuse, and LiveKit

The API layer was already there: OpenPaths accepts OpenAI-style \`/v1/chat/completions\` requests and Anthropic-style \`/v1/messages\` requests. The new work was documenting the exact agent SDK setup, putting the examples on the integrations page, and adding browser coverage so those cards stay visible.

## Migrate from OpenAI Agents SDK

The OpenAI Agents SDK supports OpenAI-compatible endpoints by accepting an \`AsyncOpenAI\` client with a custom \`base_url\`. Because OpenPaths exposes Chat Completions, use \`OpenAIChatCompletionsModel\` instead of the default Responses model path.

\`\`\`python
import asyncio
from openai import AsyncOpenAI
from agents import Agent, Runner, OpenAIChatCompletionsModel, set_tracing_disabled

client = AsyncOpenAI(
    base_url="https://openpaths.io/v1",
    api_key="op-...",
)

set_tracing_disabled(True)

agent = Agent(
    name="OpenPaths agent",
    instructions="Answer in one concise paragraph.",
    model=OpenAIChatCompletionsModel(
        model="auto-medium-task",
        openai_client=client,
    ),
)

async def main():
    result = await Runner.run(agent, "Explain why model routing helps agents.")
    print(result.final_output)

asyncio.run(main())
\`\`\`

The important migration details:

- install \`openai-agents\` and \`openai\`
- set \`base_url\` to \`https://openpaths.io/v1\`
- replace the OpenAI key with an OpenPaths key
- use an OpenPaths model ID such as \`auto-medium-task\`, \`auto-hard-task\`, \`auto-think\`, or \`openai-chat-latest\`
- disable OpenAI trace export unless you also configure a separate OpenAI tracing key

That last point matters because OpenAI Agents tracing is separate from model calls. OpenPaths can handle the model request, but trace export is still an OpenAI Agents SDK concern.

## Migrate from Anthropic Agent SDK

The Anthropic Agent SDK runs through Claude Code and accepts process environment configuration through \`ClaudeAgentOptions.env\`. Point the Anthropic base URL at OpenPaths root, not \`/v1\`, because the SDK appends the Anthropic paths itself.

\`\`\`python
import asyncio
from claude_agent_sdk import query, ClaudeAgentOptions

options = ClaudeAgentOptions(
    model="auto-medium-task",
    allowed_tools=["Read", "Grep", "Glob"],
    permission_mode="acceptEdits",
    env={
        "ANTHROPIC_BASE_URL": "https://openpaths.io",
        "ANTHROPIC_AUTH_TOKEN": "op-...",
        "ANTHROPIC_API_KEY": "",
    },
)

async def main():
    async for message in query(
        prompt="Review this project structure and suggest one improvement.",
        options=options,
    ):
        print(message)

asyncio.run(main())
\`\`\`

The migration details are:

- install \`claude-agent-sdk\`
- set \`ANTHROPIC_BASE_URL\` to \`https://openpaths.io\`
- set \`ANTHROPIC_AUTH_TOKEN\` to your OpenPaths key
- leave \`ANTHROPIC_API_KEY\` empty when the SDK expects auth-token mode
- choose an OpenPaths model ID that fits the task

OpenPaths accepts Anthropic-style Messages requests and translates them into the routed backend call. That lets Claude-flavored agent loops use the same model router as OpenAI-flavored loops.

## Which OpenPaths model should agents use?

Start with \`auto-medium-task\`. It is the practical default for coding, review, analysis, and application work.

Use \`auto-easy-task\` for cheap, repetitive, low-risk turns. Use \`auto-hard-task\` when the prompt is clearly complex. Use \`auto-think\` or \`autothink\` when reasoning depth matters more than latency.

You can also pin direct aliases:

- \`openai-chat-latest\`
- \`openai-coding-latest\`
- \`anthropic-opus-latest\`
- \`claude-sonnet-latest\`

The agent code does not need to know which upstream provider eventually wins the route. It keeps one SDK interface and one OpenPaths key.

## What we tested

We verified the current SDK configuration paths before writing the examples. OpenAI Agents SDK documents custom \`AsyncOpenAI\` clients, \`OPENAI_BASE_URL\`, and the Chat Completions model path. Anthropic Agent SDK documents \`query()\`, \`ClaudeSDKClient\`, and \`ClaudeAgentOptions\`, including environment injection for SDK runs.

On this site, we added the examples to the integrations page and extended the Playwright coverage so the OpenAI Agents SDK and Anthropic Agent SDK cards render with the same stored-key substitution as the other SDK examples.

## Why this is better than forking

Forking agent frameworks is sometimes necessary when a project has no provider escape hatch. These two SDKs do not need that for the basic OpenPaths path. Both expose enough configuration to route model calls through a compatible endpoint.

That is the migration we prefer:

- keep the upstream SDK
- keep its tools, sessions, hooks, and orchestration
- change endpoint and key configuration
- move model choice and fallback behavior into OpenPaths

Fork only when you need deeper behavior than endpoint configuration can provide, such as custom trace sinks, provider-native realtime transports, or SDK internals that hard-code a vendor-only feature.

## Bottom line

Migrating an agent SDK should not mean rewriting the agent. For OpenAI Agents SDK, swap in an OpenPaths-backed \`AsyncOpenAI\` client and use \`OpenAIChatCompletionsModel\`. For Anthropic Agent SDK, pass OpenPaths Anthropic-compatible environment variables through \`ClaudeAgentOptions.env\`.

The result is the same agent framework with a broader model layer: one OpenPaths key, task-tier routing, provider fallbacks, and room to move between OpenAI, Anthropic, Google, DeepSeek, xAI, Mistral, MiniMax, and other providers without rewriting the agent loop.`
  },
  {
    slug: 'openrouter-alternative-pool-credits-across-ai-providers',
    alternativePath: '/alternatives/openrouter',
    title: 'OpenRouter Alternative: Pool Credits Across AI Providers With OpenPaths',
    excerpt: 'OpenPaths gives teams one balance for OpenAI, Anthropic, Google, xAI, DeepSeek, Mistral, Netwrck, Fal, and more, with auto-thinking routes that pick the right model instead of making you pre-buy every provider.',
    date: '2026-05-11',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['alternatives', 'openrouter', 'providers', 'auto-routing'],
    content: `OpenRouter is useful when you want one API surface for many models. The next question is usually harder: how do you budget across providers without scattering prepaid credits, invoices, keys, and fallback rules across every vendor?

That is the problem OpenPaths is designed to solve. Instead of buying separate credits for OpenAI, Anthropic, Google, xAI, DeepSeek, Mistral, Fal, Netwrck, MiniMax, Together, and other providers, you fund one OpenPaths balance and spend it across the catalog.

## The practical difference

| Need | OpenRouter-style marketplace | OpenPaths |
|------|------------------------------|-----------|
| One API key | Yes | Yes |
| Many model providers | Yes | Yes |
| Shared credit pool | Usually marketplace-scoped | OpenPaths balance across routed providers |
| First-party media providers | Depends on catalog | Netwrck, Text-Generator.io, OpenPaths embeddings |
| Task routing | Mostly model selection | Auto models, auto-thinking, fallback chains |
| OpenAI-compatible API | Yes | Yes |

The win is not just "more models." The win is liquidity. A single pool of credits can move between text, image, video, speech, transcription, and embedding workloads as product needs change.

## Why pooled credits matter

Most teams do not know in advance which model mix they will need next month. A launch week may be mostly GPT-5.4 Mini and Claude Sonnet. A new media feature may suddenly shift spend into RA1, Sora, Hailuo, Wan, or FLUX. A search feature may move spend into embeddings.

With separate accounts, that means unused credits in one place and urgent top-ups somewhere else. With OpenPaths, the same balance can fund all of those calls.

## Auto-thinking is the second win

OpenPaths includes task-tier aliases like:

- \`auto\`
- \`auto-easy-task\`
- \`auto-medium-task\`
- \`auto-hard-task\`
- \`auto-think\`
- \`autothink\`

The point is to stop hard-coding expensive reasoning models for easy requests and stop underpowering hard requests because the cheap model happened to be the default.

For example, an agent can use \`auto-medium-task\` for routine work, then switch to \`auto-think\` when the prompt needs deeper reasoning. You keep one API key and one balance while the router chooses from the configured provider pool.

## Where OpenPaths is a better fit

OpenPaths is a strong OpenRouter alternative when your team cares about:

- pooling credits across providers instead of managing many prepaid balances
- using direct provider models and first-party OpenPaths partner models together
- routing by task difficulty instead of selecting every model manually
- fallback chains when a provider is rate-limited, down, expensive, or unhealthy
- moving between chat, image, video, audio, and embeddings without changing billing systems

## Example: one balance, several workloads

\`\`\`bash
curl https://openpaths.io/v1/chat/completions \\
  -H "Authorization: Bearer op-..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "auto-think",
    "messages": [{"role": "user", "content": "Design a fallback strategy for a multi-provider AI app."}]
  }'
\`\`\`

That same OpenPaths key can also call image, video, speech, transcription, and embedding endpoints. The product team does not need to decide how much budget belongs to each upstream provider before the month starts.

## When OpenRouter may still be the right choice

Use the tool that matches the job. If your whole workflow depends on a niche OpenRouter-only model or a marketplace feature specific to OpenRouter, keep using it. OpenPaths is better when the main goal is production routing, pooled credits, first-party media lanes, and simple provider switching under one OpenAI-compatible API.

## Bottom line

The alternative is not just another model list. The alternative is a pooled-credit model gateway: one key, one balance, multiple providers, and auto routes that can spend intelligently across the catalog.`
  },
  {
    slug: 'together-ai-alternative-for-production-model-routing',
    alternativePath: '/alternatives/together-ai',
    title: 'Together AI Alternative: Route Open Models and Frontier APIs From One Balance',
    excerpt: 'Together AI is excellent for open-model hosting. OpenPaths is different: it lets you combine Together-hosted models with OpenAI, Anthropic, Google, DeepSeek, MiniMax, Netwrck media models, and auto routes under one credit pool.',
    date: '2026-05-11',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['alternatives', 'together-ai', 'open-source', 'providers'],
    content: `Together AI is a strong platform for open models. If your workload is mainly Qwen, Kimi, GLM, DeepSeek, MiniMax, or FLUX hosted on Together infrastructure, it is a natural place to start.

OpenPaths solves a different production problem: what happens when open-model hosting is only part of the stack?

## The production model mix is rarely one provider

A real app may use:

- GPT-5.4 Mini for fast product copy
- Claude Sonnet for coding and careful long-form reasoning
- Gemini for large-context summarization
- DeepSeek for low-cost reasoning
- Together-hosted open models for open-source coverage
- Netwrck or Fal for image and video generation
- Text-Generator.io or OpenPaths embeddings for search

If each provider has its own account and balance, your model strategy becomes a finance and ops problem. OpenPaths turns that into one OpenAI-compatible gateway.

## Together AI vs OpenPaths

| Need | Together AI | OpenPaths |
|------|-------------|-----------|
| Hosted open models | Strong | Available through catalog and fallbacks |
| Frontier closed providers | Limited by platform | OpenAI, Anthropic, Google, xAI, and more |
| Media generation | Some model coverage | Netwrck, Fal, OpenAI, MiniMax, xAI, Z.AI |
| One credit pool across providers | Provider-specific | OpenPaths balance across routed calls |
| Task-based model selection | Manual | Auto models and auto-thinking routes |

OpenPaths does not replace Together for every team. It sits above provider choice when the app needs a larger model portfolio.

## Why use OpenPaths if you like Together-hosted models?

Because you can still use them. OpenPaths includes Together-hosted options and can pair them with direct provider fallbacks.

That means you can start with an open model, fail over when capacity changes, or route to a frontier provider when the task needs more reliability or depth.

## The auto route pattern

Instead of deciding that every request must use one model, route by intent:

- \`auto-easy-task\` for cheap classifiers, rewrites, extraction, and small support tasks
- \`auto-medium-task\` for normal agent and application work
- \`auto-hard-task\` for deeper coding, planning, and analysis
- \`auto-think\` when reasoning quality matters more than raw cost

The routing layer can use provider diversity without asking every product feature to manage that diversity itself.

## Credit pooling changes how teams experiment

With separate provider balances, experimentation has friction. Someone has to add billing, keys, limits, monitoring, and fallback logic before the team can test a new model.

With OpenPaths, a new model is just another model ID or route behind the same key and balance.

## Bottom line

Together AI is excellent open-model infrastructure. OpenPaths is a better fit when you want open models, closed frontier APIs, media providers, embeddings, and auto-thinking routes behind one production gateway and one pooled credit balance.`
  },
  {
    slug: 'openai-api-alternative-for-multi-provider-ai-apps',
    alternativePath: '/alternatives/openai-api',
    title: 'OpenAI API Alternative: Keep OpenAI Compatibility, Add Multi-Provider Routing',
    excerpt: 'OpenPaths keeps the OpenAI SDK shape but lets one API key reach OpenAI, Anthropic, Google, xAI, DeepSeek, Mistral, MiniMax, Netwrck, Fal, and more.',
    date: '2026-05-11',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['alternatives', 'openai', 'api', 'models'],
    content: `The OpenAI API set the standard shape for modern AI apps. Most SDKs, agent frameworks, and examples already know how to call \`/v1/chat/completions\`, \`/v1/images/generations\`, \`/v1/audio/transcriptions\`, and \`/v1/embeddings\`.

OpenPaths keeps that shape and expands what sits behind it.

## The migration is intentionally boring

\`\`\`python
from openai import OpenAI

client = OpenAI(
    base_url="https://openpaths.io/v1",
    api_key="op-...",
)

response = client.chat.completions.create(
    model="auto-medium-task",
    messages=[{"role": "user", "content": "Summarize this incident report."}],
)
\`\`\`

The code still looks like OpenAI. The routing options are broader.

## Why teams outgrow one provider

Single-provider setups are simple until they are not. Common reasons teams add a model gateway include:

- cost control across easy and hard tasks
- provider outages or rate limits
- better coding, reasoning, vision, or media models from different labs
- separate image, video, speech, and embedding needs
- regional, latency, or product-specific provider preferences

The hard part is not finding a second provider. The hard part is operating five providers without making every feature team think about five providers.

## OpenPaths as an OpenAI-compatible alternative

| Need | Direct OpenAI API | OpenPaths |
|------|-------------------|-----------|
| OpenAI SDK support | Yes | Yes |
| OpenAI models | Yes | Yes |
| Anthropic, Google, xAI, DeepSeek, Mistral | No | Yes |
| One pooled balance across providers | No | Yes |
| Auto-routing by task difficulty | No | Yes |
| First-party media and embedding lanes | No | Yes |

You can still call OpenAI models directly through OpenPaths when that is the right model. The difference is that OpenAI becomes one strong provider in the pool, not the only path.

## Auto-thinking for agents

Agents are a good example. Some turns are tiny: classify, rewrite, extract. Other turns require planning, debugging, or long reasoning. Hard-coding one expensive model wastes money; hard-coding one cheap model lowers quality.

OpenPaths auto-thinking routes give agents a stable model name while the gateway chooses more appropriate candidates underneath:

- \`auto-medium-task\` for the default agent loop
- \`auto-hard-task\` for complex implementation and investigation
- \`auto-think\` or \`autothink\` for reasoning-heavy calls

## Credit pooling is the budget advantage

With direct provider accounts, OpenAI credits do not help when your app suddenly needs more Anthropic, Gemini, or video generation capacity. OpenPaths credits can be spent across routed providers, which makes experimentation and spikes easier to absorb.

## Bottom line

OpenPaths is an OpenAI API alternative for teams that like the OpenAI-compatible developer experience but do not want their architecture, fallback plan, and credit balance locked to one provider.`
  },
  {
    slug: 'anthropic-api-alternative-with-claude-and-multi-model-fallbacks',
    alternativePath: '/alternatives/anthropic-api',
    title: 'Anthropic API Alternative: Use Claude Alongside Auto-Routed Model Fallbacks',
    excerpt: 'Claude is excellent for careful reasoning and coding. OpenPaths lets you keep Claude in the stack while adding OpenAI-compatible routing, pooled credits, and fallbacks across other model providers.',
    date: '2026-05-11',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['alternatives', 'anthropic', 'claude', 'fallbacks'],
    content: `Anthropic's Claude models are excellent for careful writing, coding, analysis, and agent workflows. Many teams want Claude in production. Fewer teams want their entire model strategy to depend on one provider account, one rate-limit envelope, and one billing pool.

OpenPaths lets Claude be part of a larger model routing system.

## Claude is a model choice, not the whole architecture

There are many cases where Claude should be the first choice:

- complex code edits
- long-context review
- policy-sensitive writing
- structured reasoning
- agent planning

There are also cases where another provider may be faster, cheaper, more available, or better suited to the modality. Image generation, video generation, embeddings, transcription, and ultra-low-cost extraction are not all Claude-shaped problems.

## Anthropic direct vs OpenPaths

| Need | Direct Anthropic API | OpenPaths |
|------|----------------------|-----------|
| Claude access | Yes | Yes |
| OpenAI-compatible route | No | Yes |
| Other providers in same app | Separate keys and billing | Same OpenPaths key and balance |
| Model fallbacks | Build yourself | Gateway-level fallback chains |
| Auto task routing | Build yourself | Auto models and auto-thinking routes |

OpenPaths is useful when your team wants Claude quality without making the product brittle around one provider.

## A better agent default

Instead of making every agent turn a Claude call, you can use:

- \`auto-easy-task\` for small extraction and classification
- \`auto-medium-task\` for the default loop
- \`auto-hard-task\` for deeper implementation and debugging
- \`auto-think\` when the request should bias toward reasoning

Claude can be part of that pool where it makes sense. So can OpenAI, Google, DeepSeek, Mistral, xAI, and other providers.

## Pooled credits reduce provider lock-in

Direct Anthropic billing is simple if every task should be Claude. Most production apps are more mixed. One feature may need Claude. Another may need embeddings. Another may need image or video generation. Another may need a cheap model for background jobs.

With OpenPaths, those calls draw from one balance. That makes model testing easier and keeps unused budget from getting stranded inside a single provider account.

## Migration shape

OpenPaths supports OpenAI-compatible calls and provider docs for Anthropic-style usage. For many apps, the change is as small as switching base URL, API key, and model ID:

\`\`\`bash
curl https://openpaths.io/v1/chat/completions \\
  -H "Authorization: Bearer op-..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "claude-sonnet-latest",
    "messages": [{"role": "user", "content": "Review this API design."}]
  }'
\`\`\`

## Bottom line

OpenPaths is not anti-Claude. It is pro-routing. Use Claude where Claude wins, then use the same API key and credit pool for the rest of the model stack.`
  },
  {
    slug: 'openpaths-agent-integrations-hermes-openclaw',
    title: 'OpenPaths Agent Integrations: Hermes Agent and OpenClaw',
    excerpt: 'Hermes Agent and OpenClaw can now use OPENPATHS_API_KEY directly, with OpenPaths auto task-tier models for easy, medium, hard, and thinking-heavy work.',
    date: '2026-05-01',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['integrations', 'agents', 'hermes', 'openclaw'],
    content: `OpenPaths works best when agent frameworks can treat it as the default model router instead of a special integration. Hermes Agent and OpenClaw now have that path: set \`OPENPATHS_API_KEY\`, pick an OpenPaths auto model, and keep using the agent surface you already use.

We added examples for both on the [Integrations](/integrations) page.

## Model IDs for agents

Both integrations use the OpenAI-compatible OpenPaths endpoint:

\`\`\`bash
OPENPATHS_API_KEY="op-..."
OPENPATHS_BASE_URL="https://openpaths.io/v1"
\`\`\`

The useful model refs for agent work are:

- \`auto\`
- \`auto-easy-task\`
- \`auto-medium-task\`
- \`auto-hard-task\`
- \`auto-think\`
- \`autothink\`

Use \`auto-medium-task\` as the default practical tier. Use \`auto-hard-task\` when the prompt is clearly deeper coding, research, debugging, or planning work. Use \`auto-think\` or \`autothink\` when you want the model route to bias toward reasoning-heavy execution.

## Hermes Agent

Hermes can detect \`OPENPATHS_API_KEY\` from the environment and use OpenPaths as the provider:

\`\`\`bash
export OPENPATHS_API_KEY="op-..."
export OPENPATHS_BASE_URL="https://openpaths.io/v1"

hermes model openpaths:auto-medium-task
hermes
\`\`\`

That keeps the Hermes CLI and gateway flow intact. The OpenPaths key is the only credential Hermes needs for the OpenPaths route, and switching tiers is just a model change:

\`\`\`bash
hermes model openpaths:auto-hard-task
\`\`\`

## OpenClaw

OpenClaw can onboard OpenPaths as a bundled provider:

\`\`\`bash
export OPENPATHS_API_KEY="op-..."

openclaw onboard --auth-choice openpaths-api-key \\
  --openpaths-api-key "$OPENPATHS_API_KEY"

openclaw models list --all --provider openpaths
openclaw models set openpaths/auto-medium-task
\`\`\`

OpenClaw also exposes thinking levels for OpenPaths auto models, so you can use \`/think medium\`, \`/think high\`, or \`/think xhigh\` in active conversations where the OpenAI-compatible transport accepts reasoning effort.

## Why this matters

Agent tools already have their own memory, command approval, gateways, workflows, and chat surfaces. OpenPaths should not replace those. It should give them a better model layer:

- one API key
- one OpenAI-compatible base URL
- task-tier routing for agent workloads
- simple switching between easy, medium, hard, and thinking-biased routes

That is the integration shape we want across agent stacks: the framework stays familiar, and OpenPaths handles the model path underneath.`
  },
  {
    slug: 'openpaths-sdk-integrations',
    title: 'OpenPaths SDK Integrations: LangChain, Vercel AI SDK, PydanticAI, Mastra, Langfuse, and LiveKit',
    excerpt: 'OpenPaths now has a dedicated integrations guide for the agent and observability SDKs developers already use in production.',
    date: '2026-05-01',
    author: 'OpenPaths Team',
    readTime: '7 min',
    tags: ['integrations', 'sdks', 'agents', 'observability'],
    content: `OpenPaths is most useful when it disappears into the tools developers already use. The point is not to learn another client library. The point is to set one base URL, use one OpenPaths API key, and keep building with your existing framework.

We added a dedicated [Integrations](/integrations) page for the SDKs that matter most in agent stacks:

- LangChain
- Vercel AI SDK
- PydanticAI
- Mastra
- Langfuse
- LiveKit Agents

Before writing the examples, we checked the current SDK source for each project in local clones: \`../langchain\`, \`../ai\`, \`../pydantic-ai\`, \`../mastra\`, \`../langfuse\`, and \`../livekit-agents\`. The shared pattern is simple: each stack either accepts an OpenAI-compatible \`base_url\` / \`baseURL\`, or accepts an AI SDK model object that can point at OpenPaths.

## The base URL

Use OpenPaths as an OpenAI-compatible endpoint:

\`\`\`bash
OPENPATHS_API_KEY="op-..."
OPENPATHS_BASE_URL="https://openpaths.io/v1"
\`\`\`

Then pick any OpenPaths model ID or alias:

- \`openai-chat-latest\` for a stable OpenAI chat alias
- \`grok-latest\` for the current Grok alias
- \`auto\` for automatic routing
- \`auto-medium-task\` for practical agent workloads
- \`grok-imagine-image\` for xAI image generation and edits

## LangChain

LangChain's OpenAI integration accepts \`base_url\`, so the setup is a direct drop-in:

\`\`\`python
from langchain_openai import ChatOpenAI

llm = ChatOpenAI(
    model="openai-chat-latest",
    api_key="op-...",
    base_url="https://openpaths.io/v1",
    temperature=0,
)

print(llm.invoke("say hi and nothing else").content)
\`\`\`

That means existing chains, retrievers, tools, and agents can keep their LangChain shape while OpenPaths handles model selection and provider routing.

## Vercel AI SDK

The AI SDK has a clean OpenAI-compatible provider path:

\`\`\`ts
import { generateText } from 'ai';
import { createOpenAICompatible } from '@ai-sdk/openai-compatible';

const openpaths = createOpenAICompatible({
  name: 'openpaths',
  apiKey: 'op-...',
  baseURL: 'https://openpaths.io/v1',
});

const { text } = await generateText({
  model: openpaths('auto'),
  prompt: 'say hi and nothing else',
});

console.log(text);
\`\`\`

This is the path we recommend for Next.js apps using \`streamText\`, \`generateText\`, tool calls, and UI message streams.

## PydanticAI

PydanticAI exposes an \`OpenAIProvider\` and \`OpenAIChatModel\` pair:

\`\`\`python
from pydantic_ai import Agent
from pydantic_ai.models.openai import OpenAIChatModel
from pydantic_ai.providers.openai import OpenAIProvider

model = OpenAIChatModel(
    'openai-chat-latest',
    provider=OpenAIProvider(
        base_url='https://openpaths.io/v1',
        api_key='op-...',
    ),
)

agent = Agent(model, instructions='Be concise.')
print(agent.run_sync('say hi and nothing else').output)
\`\`\`

Structured output and tool validation stay in PydanticAI. OpenPaths just supplies the model endpoint.

## Mastra

Mastra agents accept AI SDK model objects, so the same provider object works there:

\`\`\`ts
import { Agent } from '@mastra/core/agent';
import { createOpenAICompatible } from '@ai-sdk/openai-compatible';

const openpaths = createOpenAICompatible({
  name: 'openpaths',
  apiKey: 'op-...',
  baseURL: 'https://openpaths.io/v1',
});

export const supportAgent = new Agent({
  id: 'support-agent',
  name: 'Support Agent',
  instructions: 'Answer in one short paragraph.',
  model: openpaths('auto-medium-task'),
});
\`\`\`

This keeps Mastra workflows, memory, tools, and observability intact while making the model layer portable.

## Langfuse

Langfuse can trace OpenPaths calls through its OpenAI wrapper:

\`\`\`python
from langfuse import observe
from langfuse.openai import openai

openai.api_key = 'op-...'
openai.base_url = 'https://openpaths.io/v1'

@observe()
def run():
    response = openai.chat.completions.create(
        model='openai-chat-latest',
        messages=[{'role': 'user', 'content': 'say hi and nothing else'}],
    )
    return response.choices[0].message.content

print(run())
\`\`\`

This gives teams traces, latency, model IDs, usage, and application context around calls routed through OpenPaths.

## LiveKit Agents

LiveKit's OpenAI plugin accepts \`base_url\`, which makes OpenPaths usable for LLM turns in voice agents:

\`\`\`python
from livekit import agents
from livekit.agents import Agent, AgentSession
from livekit.plugins import openai

async def entrypoint(ctx: agents.JobContext):
    await ctx.connect()

    session = AgentSession(
        llm=openai.LLM(
            model='openai-chat-latest',
            api_key='op-...',
            base_url='https://openpaths.io/v1',
        ),
    )

    await session.start(
        room=ctx.room,
        agent=Agent(instructions='Say hi and nothing else.'),
    )
\`\`\`

For provider-native realtime WebSocket audio, keep using the provider-specific LiveKit realtime plugin. For standard LLM turns, OpenPaths works through the OpenAI-compatible LLM plugin.

## What we tested

The integration page is covered by browser tests so the examples stay visible, syntax-highlighted, and populated with the stored OpenPaths API key when a user has one. Separately, the xAI provider integration tests call the real API for chat, TTS, STT, image generation, and image editing.

The result is a cleaner integration story: OpenPaths is not asking developers to leave their stack. It is giving those stacks one routing layer for the models behind them.`
  },
  {
    slug: 'how-openpaths-is-hosted-on-codex-infinity',
    title: 'How OpenPaths Is Hosted on Codex Infinity',
    excerpt: 'A quick look at how the OpenPaths source, deploy flow, and review loop live on Codex Infinity instead of a traditional static GitHub setup.',
    date: '2026-04-24',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['engineering', 'hosting', 'codex', 'workflow'],
    content: `OpenPaths is not just deployed from a repo. The site is also maintained through \`../codex-infinity-site\`, our companion workspace on [Codex Infinity](https://codex-infinity.com/), which acts more like an agentic GitHub alternative than a passive file host.

That matters because the whole loop is tighter:

- changes are made in the same place the site is reviewed
- source and deployment stay close together
- agentic edits can be discussed, queued, and checked before they land
- the site can move quickly without turning every change into a separate, manual release chore

## What lives there

The site source, the deployed preview, and the review workflow are all designed to stay close enough that iteration feels like editing one system instead of stitching together three.

- product copy lands in the repo
- UI changes are reviewed in context
- deploys are treated as part of the workflow, not an afterthought
- the public site reflects the same source the team is actually changing

## Why we like it

Codex Infinity is useful because it behaves less like a passive code archive and more like an execution surface for real work. For a product like OpenPaths, that is a better fit than a workflow where the repo, the discussion, and the deploy target all drift apart.

It keeps us honest about the code:

- if the homepage changes, it shows up in the site flow immediately
- if the blog changes, it ships through the same path
- if we need to tighten a route, a card, or a docs link, we can do it in one place

## How it shows up in OpenPaths

You already see the result in the product:

- [Models](/models) is the routing surface
- [Providers](/providers) is the source index
- [Pricing](/pricing) explains the economics
- [Blog](/blog) gives us a place to document the system

That is the main reason the setup works. The site is not pretending to be static infrastructure. It is a living product surface hosted with the same kind of agentic, reviewable flow that the product itself is built around.

## Bottom line

OpenPaths runs better when the code, the discussion, and the deploy path stay close together. Codex Infinity gives us that, and the public site at [codex-infinity.com/@lee101/openpaths](https://codex-infinity.com/@lee101/openpaths) is the visible end of the loop.`
  },
  {
    slug: 'gpt-image-2-on-openpaths',
    title: 'GPT Image 2 on OpenPaths: better text, cleaner edits, and layouts that hold together',
    excerpt: 'A practical look at GPT Image 2, the OpenAI image model surfaced through OpenPaths, and what changes when the model starts respecting layout, typography, and editing constraints.',
    date: '2026-04-23',
    author: 'OpenPaths Team',
    readTime: '8 min',
    tags: ['openai', 'images', 'design', 'editorial'],
    content: `GPT Image 2 is the first OpenAI image model that feels designed for production layout instead of just one-off art prompts. It is built for fast, high-quality generation and editing, and it handles flexible image sizes and high-fidelity image inputs better than the older generations.

On OpenPaths, the integration is straightforward: the same OpenAI-compatible shape already used by the rest of the platform now points at \`gpt-image-2\` for image generation and editing.

![GPT Image 2 editorial cover](/blog/gpt-image-2/cover.svg)

## What changed

The biggest shift is not that GPT Image 2 makes prettier pictures. It is that the model follows instructions closely enough to be useful in real workflows:

- It renders cleaner typography and layout-aware compositions.
- It handles image inputs for refinement and editing, not just fresh generation.
- It supports flexible sizes, which makes it easier to target banners, posters, and social formats.
- It is exposed through the same \`/v1/images/generations\` and \`/v1/images/edits\` endpoints developers already expect from OpenAI-style integrations.

That combination matters because most image models are still optimized for aesthetics first and control second. GPT Image 2 closes that gap enough that designers and developers can treat generated images as production inputs instead of novelty outputs.

## What it is good at

The model is strongest when the prompt includes structure:

- Editorial covers with a clear subject and negative space.
- Product imagery where the composition has to stay legible.
- Poster layouts where typography needs to sit in a predictable place.
- Image edits that improve a draft without destroying the original composition.

![Structured workflow image](/blog/gpt-image-2/workflow.svg)

If you are used to text-only LLM prompting, the mental model changes a little. You get better results when you describe the scene, the framing, the material texture, and the visual hierarchy separately instead of stuffing everything into one sentence.

## Prompt shape that works

One good prompt pattern is:

1. State the use case.
2. Describe the subject.
3. Describe the composition and lighting.
4. Give hard constraints.

For example:

\`\`\`python
from openai import OpenAI
import base64, pathlib

client = OpenAI(base_url="https://openpaths.io/v1", api_key="op-your-key")

img = client.images.generate(
    model="gpt-image-2",
    prompt="A premium editorial poster for a blog post about AI image generation, monochrome palette, one subtle accent color, generous negative space, no watermark, no readable text",
    size="1024x1024",
    quality="high",
    n=1,
)

pathlib.Path("gpt-image-2-poster.png").write_bytes(base64.b64decode(img.data[0].b64_json))
\`\`\`

That same request works through the OpenPaths docs and playground without changing the shape of the call. The main thing to tune is the prompt itself.

![Gallery sample image](/blog/gpt-image-2/poster.svg)

## Why it matters

The practical upside is control. GPT Image 2 is not just about aesthetics; it is about getting fewer retries when you care about a layout staying coherent.

- Product teams can use it for hero images and campaign mockups.
- Founders can use it for quick visual prototypes.
- Designers can use it for concepting and composition studies.
- Builders can use it inside editing workflows instead of treating image generation as a dead-end export.

![Gallery sample image](/blog/gpt-image-2/archive.svg)

## Where we would use it

If the job is a quick illustration, a stylized concept, or a prompt-only art piece, plenty of image models work. GPT Image 2 becomes more interesting when the output has to survive contact with a real page layout:

- landing page hero art
- article cover images
- presentation visuals
- product launch mockups
- iterative edits on an existing draft

That is the line where a model stops being a toy and becomes part of the design pipeline.

## Bottom line

GPT Image 2 is worth caring about because it makes image generation feel more like a controllable design tool and less like roulette. If your workflow needs instruction following, editing, and layout awareness, it is one of the few image models that actually changes how you work.

OpenPaths already exposes the model through the same API surface as the rest of the platform, so it fits naturally into the existing developer workflow.`
  },
  {
    slug: 'switch-to-openpaths-in-2-lines',
    title: 'Switch from OpenAI or Anthropic to OpenPaths in 2 Lines of Code',
    excerpt: 'OpenPaths now supports both the OpenAI and Anthropic API formats natively. Change your base URL, swap your key, and get access to 50+ models with auto-routing, fallbacks, and unified billing.',
    date: '2026-03-03',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['engineering', 'guide', 'models'],
    content: `We just shipped Anthropic API compatibility. OpenPaths now accepts requests in both OpenAI and Anthropic formats -- meaning you can switch from either provider in two lines of code.

## From OpenAI SDK

\`\`\`python
from openai import OpenAI

client = OpenAI(
    base_url="https://openpaths.io/v1",  # was https://api.openai.com/v1
    api_key="op-your-key"                # was sk-...
)

response = client.chat.completions.create(
    model="auto-think",  # or "auto", "auto-medium-task", any direct model
    messages=[{"role": "user", "content": "Hello!"}],
    reasoning_effort="low"
)
\`\`\`

That's it. Every OpenAI SDK feature works: streaming, tool use, vision, response format, and \`reasoning_effort\`. You get access to 50+ models across 15 providers instead of just OpenAI.

## From Anthropic SDK

\`\`\`python
import anthropic

client = anthropic.Anthropic(
    base_url="https://openpaths.io",  # was https://api.anthropic.com
    api_key="op-your-key"             # was sk-ant-...
)

message = client.messages.create(
    model="auto-medium-task",  # or "claude-sonnet-4-6", etc.
    max_tokens=1000,
    thinking={"type": "enabled", "budget_tokens": 4096},
    messages=[{"role": "user", "content": "Hello!"}]
)
\`\`\`

The \`/v1/messages\` endpoint accepts the full Anthropic message format: system prompts, streaming with SSE events, tool use, content blocks -- all translated and routed through our backend.

We accept both \`Authorization: Bearer\` and \`x-api-key\` headers, so the Anthropic SDK works out of the box.

## From MiniMax SDK

MiniMax supports both OpenAI and Anthropic SDK formats. If you're already using MiniMax through either SDK, switching to OpenPaths is the same two-line change:

\`\`\`python
# Was: base_url="https://api.minimax.io/v1"
# Now: base_url="https://openpaths.io/v1"

# MiniMax models available: minimax-m2.5, minimax-m2.5-highspeed, minimax-m2.1, minimax-m2
\`\`\`

## Why Switch?

**One key, every model.** Instead of managing API keys for OpenAI, Anthropic, Google, Mistral, xAI, DeepSeek, MiniMax, and more -- use one OpenPaths key.

**Auto-routing.** Send to \`auto\` and we pick the best model for your prompt. Or use the new tiers:

- \`auto-easy-task\` -- Routes to cheapest models (Gemini 3.1 Flash Lite, MiniMax M2.5 Highspeed, GPT-4o Mini). For simple lookups, formatting, summarization. Starting at $0.25/1M input tokens.
- \`auto-medium-task\` -- Routes to mid-tier models (Claude Sonnet, Gemini Flash, DeepSeek, MiniMax M2.5). For coding, analysis, moderate complexity.
- \`auto-think\` -- Routes by reasoning depth and assigns \`none\`, \`low\`, \`medium\`, or \`high\` thinking automatically.
- \`auto\` -- Full intelligent routing across all tiers based on task complexity.

**Automatic fallbacks.** If Claude is down, your request falls through to GPT-5.2 or Gemini. No code changes needed.

**Unified billing.** One balance, one dashboard. Pay with Stripe or crypto (SOL/USDC).

## New: Auto Task Tiers

We added two new auto-routing models designed for cost optimization:

### auto-easy-task

For simple tasks that don't need a $15/1M-token model. The router picks from:
- **Gemini 3.1 Flash Lite** -- -e.25 input, .50 output, 1M context, vision
- **MiniMax M2.5 Highspeed** -- $0.30 input, 1M context, 100 tps
- **GPT-4o Mini** -- $0.15 input, 128K context, tools

Use cases: information lookup, text formatting, spell checking, simple Q&A, git commands, config changes, boilerplate generation.

### auto-medium-task

For real work that doesn't need frontier reasoning. Routes to:
- **Claude Sonnet 4.6** -- Best all-rounder for coding and analysis
- **Gemini 2.5 Flash** -- Fast with 1M context
- **DeepSeek Chat** -- Ultra-cheap coding
- **MiniMax M2.5** -- Strong 1M context model
- **GPT-4o** -- Reliable general-purpose

Use cases: feature implementation, debugging, code review, database queries, API integration, frontend development, data analysis.

## The Anthropic Endpoint in Detail

Our \`/v1/messages\` endpoint supports:

| Feature | Status |
|---------|--------|
| Text messages | Full support |
| System prompts | Full support (string and array format) |
| Streaming (SSE) | Full support with proper event types |
| Tool use | Full support (input_schema translated) |
| Temperature, top_p | Full support |
| max_tokens | Full support |
| thinking | Full support (\`enabled\` / \`disabled\`, \`budget_tokens\`) |
| Content blocks | text and tool_use |

Streaming emits proper Anthropic SSE events: \`message_start\`, \`content_block_start\`, \`content_block_delta\`, \`content_block_stop\`, \`message_delta\`, \`message_stop\`.

Response format matches Anthropic exactly:
\`\`\`json
{
  "id": "msg_abc123",
  "type": "message",
  "role": "assistant",
  "content": [{"type": "text", "text": "Hello!"}],
  "model": "auto-medium-task",
  "stop_reason": "end_turn",
  "usage": {"input_tokens": 10, "output_tokens": 5}
}
\`\`\`

## MiniMax Integration

MiniMax models are fully integrated as both a direct provider and through Together AI:

| Model | Provider | Input $/1M | Output $/1M | Context | Speed |
|-------|----------|-----------|-------------|---------|-------|
| minimax-m2.5 | Together | $0.30 | $1.20 | 1M | ~60 tps |
| minimax-m2.5-direct | MiniMax | $0.30 | $1.10 | 1M | ~60 tps |
| minimax-m2.5-highspeed | MiniMax | $0.30 | $1.10 | 1M | ~100 tps |
| minimax-m2.1 | MiniMax | $0.27 | $0.95 | 1M | ~60 tps |
| minimax-m2 | MiniMax | $0.26 | $1.00 | 200K | -- |

Plus video (Hailuo 2.3), music (Music 2.5), and speech (Speech 2.8) all through MiniMax.

## Get Started

1. [Create an account](/account) and grab an API key
2. Change your base URL to \`https://openpaths.io/v1\` (OpenAI) or \`https://openpaths.io\` (Anthropic)
3. Replace your API key with your OpenPaths key
4. Optionally switch \`model\` to \`auto\`, \`auto-easy-task\`, or \`auto-medium-task\`

That's it. Your existing code, libraries, and integrations keep working. You just get more models, automatic fallbacks, and one bill.`
  },
  {
    slug: 'state-of-ai-models-march-2026',
    title: 'The State of AI Models: 342 Models, 57 Providers, and What the Data Actually Shows',
    excerpt: 'We pulled every model from the OpenRouter catalog and crunched the numbers. Here is what the AI model landscape actually looks like in March 2026.',
    date: '2026-03-02',
    author: 'OpenPaths Team',
    readTime: '10 min',
    tags: ['data', 'analysis', 'models', 'industry'],
    content: `We pulled the full model catalog from OpenRouter's API -- 342 models from 57 different providers -- and ran the numbers. Here's what the AI model landscape actually looks like right now.

## The Big Picture: 342 Models and Counting

A year ago you could list every available LLM on a napkin. Today there are 342 models accessible through a single API. 35 of those were added in 2026 alone -- roughly one new model every two days since January.

The market is consolidating around a few major players while simultaneously fragmenting into dozens of niche providers. The top 5 providers account for over half the catalog:

| Provider | Models | Avg Input $/1M | Avg Output $/1M |
|----------|--------|---------------|-----------------|
| OpenAI | 58 | $7.03 | $28.06 |
| Qwen | 50 | $0.31 | $1.43 |
| Mistral | 27 | $0.57 | $1.71 |
| Google | 26 | $0.68 | $4.29 |
| Meta (Llama) | 17 | $0.65 | $0.74 |

OpenAI has the most models but also the highest average price -- nearly 23x more expensive than Qwen per input token. The Chinese providers (Qwen, DeepSeek, MiniMax, Z.AI, Baidu) are collectively driving prices down hard.

## The Price Collapse Is Real

Here's the distribution of all 342 models by price tier:

**Free:** 29 models (8%) -- Completely free to use. Qwen leads with 6 free models, Google has 5. These aren't toy models either -- Gemini 3.1 Flash Lite gives you a 1M context window at low cost.

**Budget (<$0.50/1M):** 179 models (52%) -- Over half of all available models cost less than fifty cents per million input tokens. This tier barely existed 18 months ago.

**Mid ($0.50-$3/1M):** 97 models (28%) -- The sweet spot where most production apps live. Claude Sonnet, Gemini Pro, GPT-5.2 all land here.

**Premium ($3-$15/1M):** 28 models (8%) -- Frontier reasoning models. Claude Opus, o3, GPT-5.2 Pro.

**Ultra (>$15/1M):** 9 models (2%) -- The bleeding edge. OpenAI's o1-pro at $150/$600 per 1M tokens sits alone at the top. GPT-5.2 Pro ($21/$168) and the legacy GPT-4 models ($30/$60) round out this tier.

The takeaway: 60% of all AI models are now either free or under $0.50 per million tokens. The "AI is expensive" narrative is increasingly wrong for most use cases.

## Context Windows Have Exploded

Remember when 4K tokens was standard? The context window distribution tells a story:

| Context Size | Models | Share |
|-------------|--------|-------|
| Under 32K | 33 | 9% |
| 32K - 128K | 106 | 30% |
| 128K - 256K | 124 | 36% |
| 256K - 1M | 63 | 18% |
| 1M+ | 16 | 4% |

The median context window is now in the 128K-256K range. 36% of all models support at least 128K tokens. xAI's Grok 4.1 Fast leads the pack at 2 million tokens.

The 1M+ club includes nearly every Gemini model, several Grok variants, and a few Qwen models. Google is clearly betting that massive context is a competitive advantage -- and at $0.25/1M input tokens for Gemini 3.1 Flash Lite, they might be right.

## The Best Value in AI

We calculated a "value score" -- context window size divided by price per million tokens. The winners might surprise you:

| Model | Context | Input $/1M | Value Score |
|-------|---------|-----------|-------------|
| Gemini 2.0 Flash Lite | 1M | $0.07 | 13.9M |
| Gemini 2.5 Flash Lite | 1M | $0.10 | 10.4M |
| GPT-4.1 Nano | 1M | $0.10 | 10.4M |
| Grok 4.1 Fast | 2M | $0.20 | 10.0M |
| GPT-5 Nano | 400K | $0.05 | 8.0M |

Google dominates value. Their Flash Lite models give you a million tokens of context with low-cost pricing. For comparison, o1-pro gives you 200K context for $150 -- a value score of just 1,333. That's a 10,000x difference in tokens-per-dollar.

Of course, value isn't everything. o1-pro solves problems that Flash Lite can't touch. But for document processing, RAG, and summarization, the value tier models are absurdly capable for their price.

## Multimodality Is the New Normal

Text-only models are becoming the minority:

| Capability | Models | Share |
|-----------|--------|-------|
| Text input | 342 | 100% |
| Image input | 130 | 38% |
| File/PDF input | 54 | 15% |
| Video input | 26 | 7% |
| Audio input | 17 | 4% |
| Image output | 6 | 1.7% |

38% of all models now accept images. Google leads multimodal with 5-modality support (text, image, file, audio, video) across their entire Gemini lineup. Image generation through LLMs (like GPT-5 Image and Gemini's image models) is still rare at just 6 models, but growing.

## Tool Use Has Won

234 out of 342 models (68%) support function calling / tool use. It's no longer a premium feature -- it's table stakes. Even budget models like Qwen3.5 Flash and Nemotron Nano support tools.

144 models (42%) support explicit reasoning modes. This was zero models two years ago. The "thinking" paradigm pioneered by o1 has been adopted across the industry -- from Xiaomi's MiMo to Liquid's 1.2B parameter model.

## The Provider Landscape

57 unique providers. Here's what stands out:

**The Big 5** (OpenAI, Anthropic, Google, Meta, Mistral) still dominate quality benchmarks. Gemini 3.5 Flash leads OpenRouter's intelligence rankings at 57.2, followed by GPT-5.3 Codex (54.0) and Claude Opus 4.6 (53.0).

**The Chinese Wave** (Qwen, DeepSeek, MiniMax, Z.AI, Baidu, ByteDance, Moonshot/Kimi, Xiaomi) now collectively offer more models than any single Western provider. They compete on price and increasingly on quality. Xiaomi's MiMo-V2-Flash claims SWE-bench scores comparable to Claude Sonnet 4.5 at 3.5% of the cost.

**The Specialists** are proliferating. Writer (enterprise agents), Perplexity (search-augmented), Liquid (edge deployment), Arcee (instruction-tuned), Cohere (enterprise RAG). Each carving out a niche.

**The Open Source Push**: Meta's Llama, AllenAI's Olmo, NVIDIA's Nemotron -- open-weight models are driving the budget tier and enabling self-hosting.

## What This Means for Developers

Three conclusions from the data:

**1. Default to cheap.** Over 60% of models cost under $0.50/1M tokens. For most tasks -- summarization, classification, extraction, simple Q&A -- a $0.10 model works as well as a $15 one. Start cheap, upgrade only where quality demands it.

**2. Context is basically free.** A million tokens of context at $0.07-0.10 means you can stuff entire codebases, documents, or conversation histories into a single request without sweating the cost. Design your applications around abundant context.

**3. Use a router.** With 342 models across 57 providers, manual model selection is a losing game. The optimal model for a coding task is different from a creative writing task is different from a document analysis task. Routing -- whether through OpenPaths's auto system or your own logic -- is no longer optional for serious applications.

The AI model market is maturing fast. Prices are falling, capabilities are rising, and the gap between "best" and "good enough" is shrinking. The developers who win are the ones who navigate this efficiently -- not the ones who always pick the most expensive option.

## How OpenPaths Fits In

OpenPaths routes across all of these providers. Our \`auto\` model uses local embeddings to pick the optimal backend per request. One API key, one balance, automatic fallbacks. The data above is exactly why we built it -- because choosing from 342 models manually is insane.

All the models mentioned in this post are available through OpenPaths. [Get started here](/account).`
  },
  {
    slug: 'how-auto-models-work',
    title: 'How Auto Models Work: Intelligent Routing for Every Request',
    excerpt: 'OpenPaths auto models use embedding-based routing to pick the best provider for your prompt. One model ID, zero config, always the right backend.',
    date: '2026-03-01',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['engineering', 'auto', 'routing'],
    content: `When you send a request to \`auto\`, \`auto-image\`, or \`auto-video\`, OpenPaths doesn't just pick a random provider. It runs your prompt through a lightweight embedding model and matches it against provider capabilities to find the optimal backend.

## The Problem

Every AI model has strengths. Claude excels at nuanced reasoning and long-form writing. GPT-5.2 is fast with strong tool use. DeepSeek is price-efficient for straightforward tasks. Grok handles real-time knowledge well.

Choosing the right one per request is tedious. Most developers pick a single model and stick with it, leaving performance and cost on the table.

## How It Works

Auto routing is a three-step pipeline:

**1. Embed the prompt** -- Your input gets embedded using our local gobed model (static-retrieval-mrl-en-v1). This runs entirely in-process with zero network latency.

**2. Score against model profiles** -- Each model in the fallback chain has a capability profile. The router computes similarity between your prompt embedding and these profiles, weighted by factors like task type (code, creative, analytical), expected output length, and whether tools or vision are needed.

**3. Pick the winner** -- The highest-scoring model gets the request. If it fails or is unhealthy, the router falls back through the chain: for \`auto\` chat, that's Gemini 3.5 Flash -> GPT-5.5 -> DeepSeek -> Claude Sonnet -> Grok -> OpenRouter fallbacks.

## Auto Image

\`auto-image\` works the same way but routes between image generators. The default chain is ra1 -> FLUX Pro -> zimage -> GLM Image. A prompt like "photorealistic portrait of a woman" routes differently than "anime character with sword" -- the former goes to ra1 or FLUX, the latter to zimage.

## Auto Video

\`auto-video\` routes between Hailuo 2.3 -> Wan -> LTX Video -> ra2v. Short creative clips go to Hailuo (fast, cheap). Longer or higher-quality requests route to Wan or ra2v.

## Using It

\`\`\`python
client = openai.OpenAI(
    base_url="https://openpaths.io/v1",
    api_key="op_..."
)

# Just use "auto" -- OpenPaths picks the best model
response = client.chat.completions.create(
    model="auto",
    messages=[{"role": "user", "content": "Explain quantum entanglement"}]
)
\`\`\`

For images:
\`\`\`python
response = client.images.generate(
    model="auto-image",
    prompt="A cyberpunk city at sunset, neon lights reflecting off wet streets"
)
\`\`\`

## Why Not Just Use the Best Model?

Cost. The best model for a "summarize this paragraph" request is not the same as the best model for "design a distributed database architecture." Auto routing gives you frontier-quality answers where they matter and saves money where they don't.

On average, auto-routed requests cost 40% less than always using the most expensive model, with negligible quality difference on straightforward tasks.`
  },
  {
    slug: 'ai-art-generation-compared',
    title: 'AI Art Generation Compared: ra1, FLUX, Stable Diffusion, and GLM Image',
    excerpt: 'A practical comparison of image generation models available through OpenPaths. What each one does best, pricing, and when to use which.',
    date: '2026-02-28',
    author: 'OpenPaths Team',
    readTime: '7 min',
    tags: ['art generation', 'models', 'comparison'],
    content: `OpenPaths routes to multiple image generation backends. Here's how they stack up and when to use each one.

## ra1 (Netwrck)

**Price:** $0.04/image | **Best for:** General-purpose art, photorealism, creative illustrations

ra1 is our recommended default for most image generation tasks. It produces high-quality, detailed images across a wide range of styles -- photorealistic portraits, landscapes, concept art, product renders. The model handles complex prompts well and consistently delivers usable results.

Supported sizes: 1024x1024, 1152x768, 768x1152, 1152x864, 864x1152, 1360x768, 768x1360

**Why we recommend it:** Excellent quality-to-price ratio. At $0.04 per image it's cheaper than FLUX Pro while producing comparable results for most use cases. It handles diverse art styles without needing negative prompts or complex prompt engineering.

## FLUX (Black Forest Labs)

Three tiers available:

**FLUX Schnell** -- $0.003/image. The speed demon. Great for prototyping and thumbnails. Quality is good but not as refined as the other tiers. Use this when you need volume over perfection.

**FLUX Dev** -- $0.015/image. Solid middle ground. Good detail, reasonable coherence with complex prompts. Open weights.

**FLUX Pro** -- $0.04/image. Premium quality. Exceptional typography rendering (one of the few models that can spell words in images reliably). Best for marketing materials, designs with text, and when you need maximum detail.

Supported sizes for all FLUX variants: 512x512 through 1440x1080, with good aspect ratio flexibility.

## Stable Diffusion 3 (Stability AI)

**Price:** $0.002/image | **Best for:** Budget bulk generation, open-source workflows

The cheapest option by far. SD3 delivers decent quality at 20x less cost than ra1 or FLUX Pro. Good for generating many variations quickly, background images, textures, and cases where slight imperfections are acceptable.

Supported sizes: 512x512 through 1024x576

## GLM Image (Z.AI)

**Price:** $0.015/image | **Best for:** High-resolution output, Chinese text rendering

GLM Image generates at higher native resolutions (up to 1728x960) than most other models. It handles both English and Chinese text well. Good photorealism and a distinct artistic style that can look more "painted" than the FLUX family.

Supported sizes: 1280x1280, 1568x1056, 1056x1568, 1472x1088, 1088x1472, 1728x960, 960x1728

## zimage (Netwrck)

**Price:** $0.007/image | **Best for:** Anime, manga, illustration styles

A specialized model tuned for anime and illustration-style art. At just $0.007 per image, it's the go-to for anime character generation, manga-style scenes, and stylized illustrations. Don't use it for photorealism -- that's not what it's built for.

## Resolution Handling

OpenPaths automatically handles resolution mismatches. If you request a size like 1920x1080 but the model only supports specific resolutions, our gateway:

1. Finds the supported resolution with the closest aspect ratio
2. Generates the image at that resolution
3. Scales up using nearest-neighbor interpolation to cover your requested size
4. Center-crops to your exact dimensions

This means you can request any resolution and always get back exactly what you asked for, regardless of what the underlying model supports.

## Quick Reference

| Model | Price | Best For | Max Resolution |
|-------|-------|----------|---------------|
| ra1 | $0.040 | General art, photorealism | 1360x768 |
| FLUX Pro | $0.040 | Typography, marketing | 1440x1080 |
| FLUX Dev | $0.015 | Good balance | 1440x1080 |
| GLM Image | $0.015 | High-res, CJK text | 1728x960 |
| zimage | $0.007 | Anime, illustration | 1024x576 |
| FLUX Schnell | $0.003 | Fast prototyping | 1440x1080 |
| SD3 | $0.002 | Bulk/budget | 1024x576 |

## Our Recommendation

Start with \`auto-image\` and let the router pick. For explicit control:
- **Default choice:** ra1 -- best overall quality/price
- **Need text in images:** FLUX Pro
- **Anime/illustration:** zimage
- **Budget bulk:** Stable Diffusion 3
- **High resolution:** GLM Image`
  },
  {
    slug: 'choosing-the-right-llm',
    title: 'Choosing the Right LLM: A Practical Guide to 2026 Models',
    excerpt: 'With 50+ LLMs available, picking the right one is overwhelming. Here is a no-nonsense breakdown of when to use what.',
    date: '2026-02-25',
    author: 'OpenPaths Team',
    readTime: '8 min',
    tags: ['models', 'comparison', 'guide'],
    content: `The LLM landscape in 2026 is dense. Here's our take on what actually matters when choosing a model through OpenPaths.

## The Frontier Tier

These are the models you reach for when quality is everything.

**Gemini 3.5 Flash** ($1.50/$9.00 per 1M) -- Our default auto model for good reason. 1M context window, strong reasoning, great at code. Google's latest and it shows. Best overall value at the frontier level.

**Claude Opus 4.6** ($5.00/$25.00) -- The deepest thinker. When you need nuanced analysis, careful reasoning, or writing that doesn't sound like AI, Opus delivers. 128K output tokens means it can generate entire codebases in one shot. Expensive but worth it for complex tasks.

**GPT-5.2** ($1.75/$14.00) -- Fast, reliable, great tool use. The workhorse that just gets things done. 400K context window is massive. Strong at structured output and function calling.

**Grok 4** ($3.00/$15.00) -- xAI's contender. 256K context, real-time knowledge from X/Twitter data. Good for tasks that benefit from current information.

## The Sweet Spot

Models that hit the best balance of quality, speed, and price.

**Claude Sonnet 4.6** ($3.00/$15.00) -- 90% of Opus quality at 60% of the price. This is what most production apps should use. Fast, reliable, excellent at code.

**Gemini 2.5 Pro** ($1.25/$10.00) -- 2M context window. If you need to process entire books, codebases, or massive document sets, nothing else comes close on context size. Quality is strong too.

**DeepSeek Chat** ($0.28/$0.42) -- The price-performance king. At ~10x cheaper than Claude Sonnet, DeepSeek handles most everyday tasks well. Code generation, summarization, Q&A -- all solid. The caveat: no vision, slightly weaker on nuanced reasoning.

## The Speed Demons

When latency matters more than depth.

**Gemini 2.5 Flash** ($0.30/$2.50) -- Blazing fast with a 1M context window. For real-time applications, chatbots, and high-throughput pipelines.

**Claude Haiku 4.5** ($1.00/$5.00) -- Anthropic's fastest model. Surprisingly capable for its speed tier. Good for classification, extraction, and simple generation tasks.

**GPT-4o Mini** ($0.15/$0.60) -- The cheapest OpenAI model. Fast and cheap, suitable for high-volume simple tasks.

## The Reasoning Specialists

For math, logic, and multi-step problem solving.

**o3** ($2.00/$8.00) -- OpenAI's dedicated reasoning model. Excels at math, science, and complex multi-step problems. Slower but more thorough.

**DeepSeek Reasoner** ($0.28/$0.42) -- Open-source reasoning at a fraction of the cost. Good for code debugging and logical analysis.

## The Free Tier

Yes, actually free. These run through OpenRouter's free tier.

**Gemini 3.1 Flash Lite** (-e.25/.50) -- Low-cost Google model with 1M context. Great for high-volume simple tasks.

**GLM 4.6v Flash** ($0.00/$0.00) -- Free vision model from Z.AI. Solid for basic visual tasks.

**Step Flash, Solar Pro 3, Nemotron Nano** -- Various free models good for testing and low-stakes tasks.

## The Code Specialists

**Qwen3 Coder** ($0.50/$1.20) -- Purpose-built for code. Strong at completion, refactoring, and generation across many languages.

**Codestral** ($0.30/$0.90) -- Mistral's code model. 256K context window is great for large codebases.

**Codex Mini** ($1.50/$6.00) -- OpenAI's latest code-focused model.

## Decision Framework

Ask yourself these questions:

1. **Does quality matter most?** -> Gemini 3.5 Flash or Claude Opus 4.6
2. **Is it a production app?** -> Claude Sonnet 4.6 or GPT-5.2
3. **Budget constrained?** -> DeepSeek Chat
4. **Need speed?** -> Gemini Flash or Haiku
5. **Processing huge documents?** -> Gemini 2.5 Pro (2M context)
6. **Just prototyping?** -> Free tier models
7. **Don't want to think about it?** -> Use \`auto\`

Or just use \`auto\` and let OpenPaths figure it out. That's literally what it's for.`
  },
  {
    slug: 'image-resolution-handling',
    title: 'How OpenPaths Handles Image Resolutions Automatically',
    excerpt: 'Request any resolution from any image model. OpenPaths matches aspect ratios, generates at supported sizes, and resizes to your exact dimensions.',
    date: '2026-02-22',
    author: 'OpenPaths Team',
    readTime: '4 min',
    tags: ['engineering', 'art generation', 'features'],
    content: `Every image model supports different resolutions. FLUX works with sizes from 512x512 to 1440x1080. ra1 supports a different set. GLM Image generates up to 1728x960. Keeping track of which model supports which sizes is annoying.

OpenPaths handles this automatically.

## The Problem

You want a 1920x1080 wallpaper. ra1's maximum supported width is 1360. FLUX tops out at 1440. Sending an unsupported resolution to the API either errors out or produces garbage.

Previously, you had to know each model's supported sizes, pick the right one, and handle the mismatch yourself.

## Our Solution

When you request any resolution, OpenPaths's image handler:

**Step 1: Find the best match.** We compare your requested aspect ratio against all supported sizes for the model. The size with the closest aspect ratio wins. If multiple sizes have the same aspect ratio, we pick the one closest in total pixel count to your request.

For example, requesting 1920x1080 (16:9) from ra1, which supports 1360x768 (16:9-ish). The aspect ratios are very close, so 1360x768 gets selected.

**Step 2: Generate at the matched size.** The image gets generated at 1360x768 -- a resolution the model actually supports and produces good results at.

**Step 3: Scale up.** Using nearest-neighbor interpolation, we scale the image up so it fully covers your requested dimensions. The scaling preserves the aspect ratio of the generated image.

**Step 4: Center crop.** If the scaled image is slightly larger than your requested dimensions (which happens when the aspect ratios don't perfectly match), we center-crop to your exact size.

The result: you always get back exactly the dimensions you asked for.

## Examples

| Requested | Model | Matched Size | Scale | Crop |
|-----------|-------|-------------|-------|------|
| 1920x1080 | ra1 | 1360x768 | 1.41x (1920x1083) | 3px top/bottom |
| 800x800 | FLUX | 1024x1024 | 0.78x (stays 1024x1024) | center 800x800 |
| 1080x1920 | GLM | 960x1728 | 1.125x (1080x1944) | 12px top/bottom |

## When It Kicks In

Resolution handling is automatic but only activates when:

1. The model has \`supported_sizes\` configured (all image models in OpenPaths do)
2. The requested size doesn't exactly match a supported size
3. The request includes a \`size\` parameter

If you request a natively supported size, the image passes through untouched -- no processing overhead.

## Format Note

When resize is needed and the original response was a URL, the resized image comes back as \`b64_json\` instead (since we need to process the pixels). Plan for this if your application specifically needs URL responses -- you can avoid it by requesting natively supported sizes.

\`\`\`python
response = client.images.generate(
    model="ra1",
    prompt="Mountain landscape at golden hour",
    size="1920x1080"  # any size works
)
# response.data[0].b64_json contains your 1920x1080 image
\`\`\``
  },
  {
    slug: 'video-generation-guide',
    title: 'AI Video Generation: Hailuo, Wan, LTX, and ra2v Compared',
    excerpt: 'A breakdown of every video generation model available through OpenPaths. Speed, quality, pricing, and what each one handles best.',
    date: '2026-02-18',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['video', 'models', 'comparison'],
    content: `Video generation is moving fast. OpenPaths provides access to four video generation backends, each with different strengths.

## Hailuo 2.3 (MiniMax)

**Price:** $0.10/video | **Speed:** ~30 seconds | **Best for:** Short creative clips, social media content

Hailuo is our default video model and the best overall choice for most use cases. MiniMax's latest version produces smooth, coherent motion with good prompt adherence. The fast variant ($0.05/video) sacrifices some quality for 2x speed -- good for iteration.

Three tiers:
- **Hailuo 2.3** ($0.10) -- Full quality, best results
- **Hailuo 2.3 Fast** ($0.05) -- Good quality, half the time
- **Hailuo 02** ($0.08) -- Previous generation, still solid

## Wan (Netwrck)

**Price:** $0.30/video | **Best for:** High-quality cinematic content, complex scenes

Wan produces the most cinematic results. Better at handling complex multi-subject scenes, camera movements, and lighting effects. The higher price reflects the quality -- use it when the output really matters. Supports image-to-video (provide an \`image_url\` to animate a still image).

## LTX Video (Netwrck)

**Price:** $0.05-$0.072/video | **Best for:** Budget video generation, experimentation

Two versions available:
- **LTX Video** (v097) at $0.05 -- The original. Fast and cheap, good for testing prompts.
- **LTX 2** at $0.072 -- Improved quality and coherence over v1.

Both are good choices for prototyping before committing to a more expensive model.

## ra2v (Netwrck)

**Price:** $1.00/video | **Best for:** Maximum quality, production-grade output

The premium option. ra2v produces the highest quality video but at 10x the cost of Hailuo. Reserve this for final production renders where every frame matters.

## Decision Guide

- **Default choice:** Hailuo 2.3 -- best balance of quality, speed, and price
- **Budget/prototyping:** LTX Video ($0.05)
- **High quality needed:** Wan ($0.30) or ra2v ($1.00)
- **Don't want to think about it:** Use \`auto-video\`

## Using It

\`\`\`python
import requests

response = requests.post(
    "https://openpaths.io/v1/video/generations",
    headers={"Authorization": "Bearer op_..."},
    json={
        "model": "auto-video",
        "prompt": "A drone shot flying over a misty mountain range at sunrise"
    }
)
video_url = response.json()["video_url"]
\`\`\`

For image-to-video:
\`\`\`python
response = requests.post(
    "https://openpaths.io/v1/video/generations",
    headers={"Authorization": "Bearer op_..."},
    json={
        "model": "wan",
        "prompt": "Camera slowly zooms in, subject blinks",
        "image_url": "https://example.com/portrait.jpg"
    }
)
\`\`\``
  },
  {
    slug: 'openpaths-vs-openrouter',
    title: 'OpenPaths vs OpenRouter: Why We Built Another Model Router',
    excerpt: 'There are other model routers out there. Here is what makes OpenPaths different and why we think it matters.',
    date: '2026-02-15',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['company', 'comparison'],
    content: `OpenRouter exists and it's good. We even use it as a fallback provider. So why build OpenPaths?

## What We Do Differently

**Multimodal from day one.** OpenPaths isn't just a chat router. We route images (ra1, FLUX, Stable Diffusion, GLM Image, zimage), video (Hailuo, Wan, LTX, ra2v), music (MiniMax), speech synthesis, transcription, and embeddings. One API key, one balance, every modality.

**Auto routing with embeddings.** Our \`auto\` models use local embedding-based routing to pick the best backend per request. Not round-robin, not random -- actual semantic matching against model capabilities.

**Automatic resolution handling.** Request any image size from any model. We match aspect ratios, generate at supported resolutions, and resize to your exact dimensions. No more memorizing which model supports which sizes.

**Solana payments.** Fund your account with SOL or USDC. No KYC for crypto payments. For developers who prefer decentralized infrastructure.

**Open source.** The entire gateway is open source. You can self-host OpenPaths, audit the routing logic, or contribute improvements.

**First-party embeddings.** Our local gobed embedding model runs in-process with zero network latency. No external API call needed for embeddings.

## What OpenRouter Does Better

Transparency -- OpenRouter has been around longer and has a larger community. They support more models in their catalog. Their pricing page is more detailed.

We use OpenRouter as a fallback for exactly this reason. If our primary provider for a model is unhealthy, requests can route through OpenRouter automatically.

## When to Use OpenPaths

- You need image, video, music, or speech generation alongside chat
- You want auto-routing that picks the best model per request
- You want to pay with crypto
- You want to self-host the gateway
- You want automatic resolution handling for image generation

## When to Use OpenRouter

- You need access to a niche model we don't directly support
- You're already integrated and switching cost is high

## The Real Answer

Use both. OpenPaths for primary routing with OpenRouter as a fallback. That's literally how we designed it.`
  },
  {
    slug: 'music-and-speech-models',
    title: 'AI Music Generation and Text-to-Speech on OpenPaths',
    excerpt: 'Generate music with MiniMax and convert text to natural speech. Here is how our audio models work.',
    date: '2026-02-10',
    author: 'OpenPaths Team',
    readTime: '4 min',
    tags: ['music', 'speech', 'models'],
    content: `OpenPaths isn't just chat and images. We provide full access to music generation and text-to-speech through MiniMax's audio models.

## Music Generation

Two models available through MiniMax:

**Music 2.5** ($0.01/generation) -- The latest. Better instrument separation, more coherent song structures, and improved vocal synthesis. Supports genre specification, mood control, and lyrics input.

**Music 2.0** ($0.008/generation) -- Previous generation. Still produces good results at a lower price point.

\`\`\`python
response = requests.post(
    "https://openpaths.io/v1/music/generations",
    headers={"Authorization": "Bearer op_..."},
    json={
        "model": "music-2.5",
        "prompt": "Upbeat electronic track with synth leads, 128 BPM"
    }
)
audio_url = response.json()["audio_url"]
\`\`\`

## Text-to-Speech

Four TTS models, split between HD (higher quality) and Turbo (faster):

**Speech 2.8 HD** ($100/1M chars) -- Highest quality. Natural intonation, emotional range, minimal artifacts. Use for final production audio, audiobooks, and professional voiceovers.

**Speech 2.8 Turbo** ($60/1M chars) -- 40% cheaper with slightly less naturalness. Good for real-time applications and high-volume generation.

**Speech 2.6 HD/Turbo** -- Previous generation. Available for backwards compatibility. Slightly lower quality but same pricing.

\`\`\`python
response = requests.post(
    "https://openpaths.io/v1/audio/speech",
    headers={"Authorization": "Bearer op_..."},
    json={
        "model": "speech-2.8-hd",
        "input": "Welcome to OpenPaths. The open source model router.",
        "voice": "alloy"
    }
)
# response contains audio data
\`\`\`

## Transcription (Speech-to-Text)

Audio to text through four providers with automatic model-based routing. Specify a model and we route to the right provider. Default is Groq's whisper-large-v3-turbo (fastest and cheapest multilingual option).

**Groq** -- Fastest inference (228x real-time). Three Whisper variants:
- \`whisper-large-v3-turbo\` -- $0.00067/min ($0.04/hr) -- best default
- \`whisper-large-v3\` -- $0.00185/min ($0.111/hr) -- highest quality multilingual
- \`distil-whisper-large-v3-en\` -- $0.00033/min ($0.02/hr) -- English only, absolute cheapest

**OpenAI** -- Best accuracy models:
- \`gpt-4o-mini-transcribe\` -- $0.003/min ($0.18/hr) -- recommended cost/quality balance
- \`gpt-4o-transcribe\` -- $0.006/min ($0.36/hr) -- lowest word error rate
- \`whisper-1\` -- $0.006/min ($0.36/hr) -- legacy

**Fireworks AI** -- Good middle ground:
- \`whisper-v3-large-turbo\` -- $0.0009/min ($0.054/hr)
- \`whisper-v3-large\` -- $0.0015/min ($0.09/hr)

**Fal** -- Serverless Whisper with chunk-level timestamps.

\`\`\`python
with open("recording.mp3", "rb") as f:
    response = requests.post(
        "https://openpaths.io/v1/audio/transcriptions",
        headers={"Authorization": "Bearer op_..."},
        files={"file": f},
        data={"model": "whisper-large-v3-turbo"}
    )
print(response.json()["text"])
\`\`\`

Supported formats: mp3, mp4, mpeg, mpga, m4a, wav, webm, ogg, flac. Optional parameters: \`language\` (ISO-639 hint), \`prompt\` (vocabulary hint), \`response_format\` (json, text, srt, verbose_json, vtt).

## Pricing Summary

| Model | Provider | Price | Use Case |
|-------|----------|-------|----------|
| Music 2.5 | MiniMax | $0.01/gen | Music generation |
| Music 2.0 | MiniMax | $0.008/gen | Budget music gen |
| Speech 2.8 HD | MiniMax | $100/1M chars | Premium TTS |
| Speech 2.8 Turbo | MiniMax | $60/1M chars | Fast TTS |
| distil-whisper-large-v3-en | Groq | $0.00033/min | Cheapest STT (English) |
| whisper-large-v3-turbo | Groq | $0.00067/min | Fast STT (default) |
| whisper-v3-large-turbo | Fireworks | $0.0009/min | Mid-tier STT |
| whisper-v3-large | Fireworks | $0.0015/min | Quality STT |
| whisper-large-v3 | Groq | $0.00185/min | Best multilingual STT |
| gpt-4o-mini-transcribe | OpenAI | $0.003/min | Best cost/accuracy |
| gpt-4o-transcribe | OpenAI | $0.006/min | Highest accuracy STT |`
  },
  {
    slug: 'free-ai-models',
    title: 'Free AI Models You Can Use Right Now on OpenPaths',
    excerpt: 'Several high-quality models are available at low cost through OpenPaths. Here is what they are and what they can do.',
    date: '2026-02-05',
    author: 'OpenPaths Team',
    readTime: '3 min',
    tags: ['models', 'free', 'guide'],
    content: `Not everything costs money. OpenPaths routes to several free models through OpenRouter's free tier and Z.AI's free offerings. Here's what's available at $0.00.

## Free Chat Models

**GLM 4.6v Flash** (Z.AI) -- A free vision model that accepts both text and images. 128K context. Good for basic visual analysis, OCR, and image description tasks. Surprisingly capable for free.

**Gemini 3.1 Flash Lite** (Google) -- 1M context window at low cost. Fast, supports vision. The best low-cost model for processing large documents.

**Step Flash** (StepFun) -- 256K context, tool support. A solid general-purpose free model.

**Solar Pro 3** (Upstage) -- 128K context with tool support. Good for Korean and English tasks.

**Nemotron Nano** (NVIDIA) -- 256K context. NVIDIA's lightweight free model with tool support.

**Arcee Trinity / Trinity Mini** (Arcee AI) -- 131K context. Two tiers of the same model family, both free with tool support.

**LFM Thinking / LFM Instruct** (Liquid) -- 32K context. Smaller but fast. The "thinking" variant shows its reasoning steps.

## Free Embeddings

**Nemotron Embed** (NVIDIA) -- Free vision-capable embedding model through OpenRouter. 8K context.

**OpenPaths Embed** (gobed) -- Our first-party embedding model. Runs locally in-process. $0.001 per request with automatic truncation by default and chunk-averaging support for longer text.

## What's the Catch?

Free models through OpenRouter have rate limits. You'll get throttled at high volume. For production workloads, use paid models. For prototyping, experimentation, and low-volume use cases, free models are perfect.

## How to Use Them

\`\`\`python
# Free vision model
response = client.chat.completions.create(
    model="glm-vision-flash",
    messages=[{
        "role": "user",
        "content": [
            {"type": "text", "text": "What's in this image?"},
            {"type": "image_url", "image_url": {"url": "https://..."}}
        ]
    }]
)

# Free large context
response = client.chat.completions.create(
    model="gemini-lite",
    messages=[{"role": "user", "content": very_long_document}]
)
\`\`\`

Free models are a great way to get started with OpenPaths before committing any budget. Create an account, grab an API key, and start building.`
  },

  // --- Provider Deep Dives ---

  {
    slug: 'provider-openai',
    title: 'OpenAI on OpenPaths: GPT-5, Codex, o3, and the Model That Started It All',
    excerpt: 'From GPT-3 to GPT-5.5 -- OpenAI defined the modern AI era. Here is how their full model lineup works through OpenPaths and why having alternatives matters.',
    date: '2026-03-10',
    author: 'OpenPaths Team',
    readTime: '7 min',
    tags: ['providers', 'openai', 'models'],
    content: `OpenAI needs no introduction. They launched the modern AI wave with GPT-3 in 2020, turned it mainstream with ChatGPT in 2022, and have shipped relentlessly since. Love them or not, every AI company today is building in their shadow.

## The OpenAI Lineup on OpenPaths

We route to 11 OpenAI models spanning four distinct tiers:

| Model | Price (In/Out per 1M) | Context | Best For |
|-------|----------------------|---------|----------|
| GPT-5.5 | $5.00/$30.00 | 1.05M | Latest flagship, long context |
| GPT-5.4 | $2.50/$15.00 | 1.05M | Previous flagship, long context |
| GPT-5 Chat Latest | $1.25/$10.00 | 400K | Conversational, creative |
| GPT-5 Codex | $1.25/$10.00 | 400K | Software engineering |
| GPT-5.3 Codex | $1.25/$10.00 | 400K | Advanced coding |
| GPT-5.3 Codex Spark | $0.50/$2.00 | 400K | Fast lightweight coding |
| o3 | $2.00/$8.00 | 200K | Deep reasoning, math |
| o4-mini | $1.10/$4.40 | 200K | Cost-effective reasoning |
| Codex Mini | $1.50/$6.00 | 400K | Fast code generation |
| GPT-5 Mini | $0.25/$2.00 | 400K | High-volume, affordable |
| GPT-4o | $2.50/$10.00 | 128K | Proven multimodal |
| GPT-4o Mini | $0.15/$0.60 | 128K | Ultra-cheap simple tasks |

## Strengths

**Tool use and structured output.** No one does function calling better than OpenAI. If your application relies heavily on structured JSON output, tool orchestration, or multi-step agent workflows, GPT models are the most reliable choice. The Codex line takes this further with SWE-bench performance that rivals dedicated coding tools.

**Speed.** GPT-5.4 ships with a 1.05 million token context window and still manages competitive latency. GPT-5 Mini at $0.25/1M input tokens is absurdly fast for its price point.

**The reasoning split.** OpenAI was first to ship dedicated reasoning models with o1, and the lineage continues with o3 and o4-mini. The separation between "fast chat" and "deep thinking" models gives developers precise control over cost-quality tradeoffs.

**Ecosystem.** The OpenAI SDK is the de facto standard. Most AI tooling, frameworks, and tutorials assume OpenAI's API format. Using OpenPaths means you get this compatibility for free -- our API is OpenAI-compatible, so any code written for OpenAI works unchanged.

## Weaknesses

**Price at the top.** GPT-5.4 at $2.50/$15.00 is competitive with Gemini 3.5 Flash but significantly more expensive than DeepSeek V3.2 ($0.28/$0.42) for tasks where the quality difference is negligible.

**Context window limitations on older models.** GPT-4o and GPT-4o Mini are stuck at 128K. If you need more context from OpenAI, you have to jump to the GPT-5 family.

**No free tier.** Unlike Google (Gemini 3.1 Flash Lite) or Z.AI (GLM-4.6v Flash), OpenAI offers nothing at low cost. The cheapest entry point is GPT-4o Mini at $0.15/1M.

## How We Integrated

OpenPaths speaks OpenAI's API format natively -- it was the first format we supported. Our auto router uses GPT-5 Chat Latest as a primary fallback in the chain, and o3 slots in for reasoning-heavy tasks. When you send a request to \`auto\`, if the router determines it needs strong general performance with fast tool use, it often lands on a GPT-5 variant.

## The Evolution

OpenAI's pace is remarkable. In 18 months they went from GPT-4 (a single model) to a lineup of 11 models covering chat, code, reasoning, vision, and budget tiers. The introduction of the Codex brand specifically for software engineering signals where they see the highest commercial value.

The most interesting development is GPT-5.3 Codex Spark -- a $0.50/$2.00 model that's free with your own OpenAI key. It suggests OpenAI is willing to give away fast inference to lock developers into their ecosystem. Through OpenPaths, you get access to Spark alongside every other provider, so you never have to choose just one ecosystem.

## When to Use OpenAI Through OpenPaths

- **Agent workflows** with heavy tool use and structured output
- **Code generation** where Codex models excel
- **Reasoning tasks** where o3 or o4-mini outperform general chat models
- **Any existing OpenAI integration** -- just change the base URL to \`openpaths.io/v1\`

Or use \`auto\` and let us pick when GPT-5 is the right backend for your specific request.`
  },
  {
    slug: 'provider-anthropic',
    title: 'Anthropic on OpenPaths: Claude Opus, Sonnet, Haiku, and the Art of Careful AI',
    excerpt: 'Anthropic builds the most thoughtful AI models in the industry. Here is how Claude fits into OpenPaths and why developers love the Anthropic approach.',
    date: '2026-03-10',
    author: 'OpenPaths Team',
    readTime: '7 min',
    tags: ['providers', 'anthropic', 'models'],
    content: `Anthropic was founded in 2021 by former OpenAI researchers who wanted to build AI more carefully. Five years later, Claude is the model that developers trust most for nuanced, high-stakes work. There is a reason Anthropic's models consistently rank highest on coding benchmarks and user satisfaction surveys.

## The Claude Lineup on OpenPaths

| Model | Price (In/Out per 1M) | Context | Best For |
|-------|----------------------|---------|----------|
| Claude Opus 4.6 | $5.00/$25.00 | 200K | Frontier reasoning, complex tasks |
| Claude Sonnet 4.6 | $3.00/$15.00 | 200K | Production workhorse |
| Claude Haiku 4.5 | $1.00/$5.00 | 200K | Fast, high-volume tasks |
| Claude Opus 4.5 | $5.00/$25.00 | 200K | Previous-gen deep reasoning |
| Claude Sonnet 4.5 | $3.00/$15.00 | 200K | Previous-gen all-rounder |

## Strengths

**Writing quality.** Claude produces text that sounds human in a way that other models struggle to match. Technical documentation, creative writing, nuanced analysis -- Claude's output consistently requires the least editing. This is not a benchmark you can measure easily, but developers notice it immediately.

**Code generation.** Claude Sonnet 4.6 is the model most developers reach for when building software. It understands codebases holistically, suggests architecturally sound changes, and catches subtle bugs. Opus 4.6 goes further -- it can reason about entire systems and produce complete implementations in a single pass with its 128K output token limit.

**Safety and instruction following.** Anthropic's constitutional AI approach means Claude follows complex instructions more faithfully than competitors. When you say "never include PII in the output" or "always respond in JSON," Claude actually does it. This matters enormously for production applications.

**Honest uncertainty.** Claude tells you when it does not know something. Other models hallucinate confidently. This is a feature, not a limitation -- for applications where correctness matters more than always having an answer.

## Weaknesses

**Price.** Opus 4.6 at $5.00/$25.00 is the most expensive chat model on OpenPaths. For simple tasks, you are paying a premium for capabilities you do not need.

**Context window.** 200K tokens is generous but falls short of Google's 2M (Gemini 2.5 Pro) or xAI's 2M (Grok 4.1 Fast). For processing very large document sets, Claude requires chunking strategies that other models handle natively.

**No free tier.** The cheapest Claude model is Haiku at $1.00/$5.00 -- more expensive than many competitors' mid-tier models.

**Speed at the top.** Opus 4.6 is slower than GPT-5.4 or Gemini 3.5 Flash for equivalent tasks. The quality-per-token is higher, but latency-sensitive applications may prefer faster alternatives.

## How We Integrated

OpenPaths was originally OpenAI-compatible only. Adding Anthropic compatibility was one of our biggest engineering efforts -- translating between the Messages API format and OpenAI's chat completions format, handling Anthropic-specific SSE events, supporting content blocks, and mapping tool use schemas.

The result: you can use the official Anthropic SDK with OpenPaths by changing two lines of code. Our \`/v1/messages\` endpoint accepts the full Anthropic format natively.

Claude Sonnet 4.6 sits in our \`auto\` and \`auto-medium-task\` routing chains. When the router detects a coding task, complex analysis, or a request that benefits from careful reasoning, Claude is often the model it selects.

## The Anthropic Difference

What makes Anthropic interesting is their research-first approach. They published the papers on constitutional AI, RLHF improvements, and scaling laws before shipping products. The result is models that feel qualitatively different from competitors -- more careful, more consistent, less likely to produce garbage.

The 4.6 generation represents a significant jump. Opus 4.6 scores 53.0 on OpenRouter's intelligence rankings, placing it in the top 3 globally. But the real story is Sonnet 4.6 -- it delivers roughly 90% of Opus quality at 60% of the price, making it the model most production applications should default to.

## When to Use Anthropic Through OpenPaths

- **Code review and generation** where quality and correctness matter
- **Complex analysis** requiring multi-step reasoning
- **Content generation** that needs to sound natural
- **Regulated industries** where instruction adherence is critical
- **Any task where you would rather get "I don't know" than a confident hallucination**

If you are migrating from the Anthropic SDK, OpenPaths is a drop-in replacement. Same format, same features, more fallback options.`
  },
  {
    slug: 'provider-google',
    title: 'Google on OpenPaths: Gemini, Million-Token Context, and the Value King',
    excerpt: 'Google Gemini offers the largest context windows and best price-to-performance in the industry. Here is how we route to the full Gemini lineup.',
    date: '2026-03-09',
    author: 'OpenPaths Team',
    readTime: '7 min',
    tags: ['providers', 'google', 'models'],
    content: `Google entered the LLM race late but came in swinging. Gemini launched in December 2023, and by 2026 they have the largest context windows, the cheapest per-token pricing at every tier, and arguably the best multimodal capabilities of any provider.

## The Gemini Lineup on OpenPaths

| Model | Price (In/Out per 1M) | Context | Best For |
|-------|----------------------|---------|----------|
| Gemini 3.5 Flash | $1.50/$9.00 | 1M | Flagship, default auto |
| Gemini 2.5 Pro | $1.25/$10.00 | 2M | Massive context processing |
| Gemini 2.5 Flash | $0.30/$2.50 | 1M | Fast, high-throughput |
| Gemini 3.1 Flash Lite | $0.25/$1.50 | 1M | High-volume simple tasks |

## Strengths

**Context windows that redefine what is possible.** Gemini 2.5 Pro accepts 2 million tokens in a single request. That is roughly 1,500 pages of text, or an entire large codebase, or hours of transcribed audio. Gemini 3.5 Flash and Gemini 2.5 Flash both handle 1 million tokens. No other provider comes close at these price points.

**Value.** Gemini 3.1 Flash Lite gives you a 1M context window for $0.25/$1.50 per million tokens. That remains inexpensive for high-volume workloads. Our value analysis shows Gemini models occupy 3 of the top 5 positions for tokens-per-dollar -- a 10,000x advantage over premium models.

**Multimodal breadth.** Gemini natively handles text, images, audio, video, and PDFs. You can feed it a YouTube video transcript, a set of images, and a text prompt in a single request. No other model family matches this breadth of input modalities.

**Speed.** Gemini 2.5 Flash is one of the fastest models available. For real-time chatbots, classification pipelines, and high-throughput batch processing, Flash delivers competitive quality at latencies under 500ms for short responses.

## Weaknesses

**Consistency.** Google iterates rapidly on Gemini, which sometimes means version-to-version behavior changes. The API has had breaking changes and model ID deprecations that require developer attention.

**Instruction following on edge cases.** For highly structured output requirements or complex multi-constraint prompts, Claude and GPT-5 tend to follow instructions more reliably than Gemini.

**Regional availability.** Some Gemini features and models have variable availability depending on region and Google Cloud account status.

## How We Integrated

Google uses a custom API format -- not OpenAI-compatible. Our integration translates between OpenAI's chat completions format and Google's \`generateContent\` endpoint, handling content parts, safety settings, tool declarations, thinking configuration, and streaming.

Gemini 3.5 Flash is the first model in our \`auto\` routing chain. It is our default "smart" model -- when the router cannot confidently classify a task as needing a specialist, Gemini 3.5 Flash handles it. The reasoning: best overall value at the frontier tier with a massive context window.

Gemini 3.1 Flash Lite powers \`auto-easy-task\` -- the routing tier for simple lookups, formatting, and summarization where spending more than $0.25/1M tokens is wasteful.

## The Google Advantage

Google's real moat is infrastructure. They designed Gemini to run on TPUs they built themselves, in data centers they own, connected by networks they control. This vertical integration means they can offer prices that would be unprofitable for competitors using rented GPU clusters.

The result is a provider that competes on both quality AND price simultaneously. Gemini 3.5 Flash is not just cheap -- it ranks #1 on OpenRouter's intelligence benchmark at 57.2, ahead of GPT-5.3 Codex and Claude Opus 4.6. That combination of top-tier quality and competitive pricing makes Google the value king of the AI industry.

## The Context Revolution

A year ago, 128K tokens was considered generous. Google normalized 1M+ context windows and made them affordable. This changes how applications are built:

- **RAG becomes optional.** Instead of chunking documents and doing similarity search, just put the whole document in the context window.
- **Codebases fit in one prompt.** A 50,000-line codebase is roughly 750K tokens. Gemini 2.5 Pro can hold two of those.
- **Conversation history is infinite.** At 2M tokens, you can maintain a conversation that spans days without summarization.

## When to Use Google Through OpenPaths

- **Large document processing** where context window size matters
- **Budget-conscious applications** that need good quality at low cost
- **Prototyping** with Gemini 3.1 Flash Lite for high-volume cost efficiency
- **Multimodal applications** processing images, audio, and video
- **High-throughput pipelines** where Flash delivers speed at scale`
  },
  {
    slug: 'provider-xai',
    title: 'xAI on OpenPaths: Grok, 2M Context, and the Challenger From X',
    excerpt: 'xAI entered the AI race in 2023 and already has frontier models with the largest context windows in the industry. Here is what Grok brings to the table.',
    date: '2026-03-09',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['providers', 'xai', 'models'],
    content: `xAI launched in 2023 with a simple pitch: build AI that understands the real world. Backed by Elon Musk and trained partly on X (Twitter) data, Grok has gone from curiosity to legitimate frontier contender faster than anyone expected.

## The Grok Lineup on OpenPaths

| Model | Price (In/Out per 1M) | Context | Best For |
|-------|----------------------|---------|----------|
| Grok 4 | $3.00/$15.00 | 256K | Flagship reasoning |
| Grok 4.1 Fast | $0.20/$0.50 | 2M | Ultra-fast, massive context |
| Grok 3 Mini | $0.30/$0.50 | 131K | Affordable reasoning |

## Strengths

**The 2M context window.** Grok 4.1 Fast holds 2 million tokens at $0.20/$0.50 per million. That is the largest context window available on OpenPaths at the lowest price for its capacity class. You can feed it entire repositories, book-length documents, or weeks of conversation history in a single request.

**Speed-to-cost ratio.** Grok 4.1 Fast lives up to its name. At $0.20 input/$0.50 output per million tokens with a 2M context, it occupies a unique niche -- frontier-adjacent quality at budget pricing with reasoning capabilities included.

**Current knowledge.** Trained on real-time data from X, Grok handles questions about recent events, trending topics, and current affairs better than models with static training cutoffs. For applications that need up-to-date knowledge without RAG, Grok has an edge.

**Reasoning across the lineup.** All three Grok models support reasoning modes. You do not need to choose between a "fast" model and a "thinking" model like with OpenAI -- Grok gives you both in one.

## Weaknesses

**Ecosystem maturity.** xAI's API is newer than OpenAI's or Anthropic's. Documentation is thinner, community resources are fewer, and edge cases in the API are less well-documented.

**Smaller model lineup.** Three models versus OpenAI's 11 or Mistral's 10. xAI has fewer options for fine-grained cost-quality tradeoffs.

**Training data bias.** The X/Twitter training data gives Grok a distinctive voice that can lean informal or opinionated. For professional or enterprise applications, this tone may require additional prompting to control.

## How We Integrated

xAI's API is OpenAI-compatible, which made integration straightforward. Grok slots into our routing chain as a fallback after the primary models -- if Gemini, GPT-5, DeepSeek, and Claude are all unhealthy, Grok picks up the request.

Grok 4.1 Fast is particularly interesting for our routing because its 2M context window means we can route large-context requests to it without worrying about truncation.

## The xAI Trajectory

xAI's speed of execution is notable. They went from founding to frontier model in under two years. Grok 4 competes with Claude Sonnet and GPT-5 on benchmarks, and Grok 4.1 Fast offers the industry's best context-to-price ratio.

The company is building its own data center (the Memphis "Colossus" cluster with 100,000 GPUs), which suggests pricing will get more competitive as they move off rented infrastructure. For a company that is barely three years old, having a frontier model and purpose-built inference infrastructure is unprecedented.

## When to Use xAI Through OpenPaths

- **Massive context tasks** where 2M tokens at $0.20 is unbeatable
- **Real-time knowledge** queries that benefit from current X/Twitter data
- **Budget reasoning** with Grok 3 Mini at $0.30/$0.50
- **Fast inference** where Grok 4.1 Fast delivers speed and capacity`
  },
  {
    slug: 'provider-deepseek',
    title: 'DeepSeek on OpenPaths: Open-Source AI That Punches Way Above Its Price',
    excerpt: 'DeepSeek delivers GPT-5 class performance at 10x lower cost. Here is why the Chinese open-source lab is reshaping AI economics.',
    date: '2026-03-08',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['providers', 'deepseek', 'models', 'open-source'],
    content: `DeepSeek came out of nowhere in 2024 and immediately changed the economics of AI. Their V3 model matched GPT-4o quality at a fraction of the cost. V3.2 goes further -- it competes with GPT-5 on coding and reasoning benchmarks while costing $0.28/$0.42 per million tokens. That is roughly 10x cheaper than comparable models.

## The DeepSeek Lineup on OpenPaths

| Model | Price (In/Out per 1M) | Context | Best For |
|-------|----------------------|---------|----------|
| DeepSeek V3.2 | $0.28/$0.42 | 128K | General, coding, daily tasks |
| DeepSeek Reasoner | $0.28/$0.42 | 128K | Math, logic, complex reasoning |

We also route to DeepSeek R1 and V3.1 through Together AI for additional redundancy.

## Strengths

**Price-performance ratio.** This is DeepSeek's headline feature. At $0.28 input/$0.42 output per million tokens, V3.2 costs less than 10% of Claude Sonnet 4.6 while handling most everyday tasks comparably. For summarization, Q&A, code generation, data extraction, and translation, the quality difference is negligible but the cost difference is 10x.

**Open source.** DeepSeek's models are open-weight, meaning the community can inspect, fine-tune, and self-host them. This matters for organizations with data residency requirements or those who want to run inference on their own hardware.

**Coding ability.** DeepSeek was founded by a quantitative trading firm (High-Flyer). Their DNA is mathematical and code-oriented, and it shows. V3.2 handles code generation, debugging, and refactoring remarkably well for its price tier.

**Extended thinking.** DeepSeek Reasoner supports chain-of-thought reasoning that shows its work step by step. For math proofs, complex debugging, and multi-step logic problems, this transparency helps developers verify the model's reasoning.

## Weaknesses

**No vision.** DeepSeek models are text-only. No image input, no multimodal capabilities. If your application needs to process images, you need a different model.

**128K context ceiling.** Adequate for most tasks but limiting compared to Gemini's 2M or Grok's 2M. Large codebase analysis or long-document processing may require chunking.

**Availability.** Being based in China, DeepSeek has occasionally faced availability issues for international users. API rate limits can be tighter during peak hours.

**Nuance.** For tasks requiring cultural context, subtle tone control, or creative writing that needs to sound natural in English, Claude and GPT-5 have an edge. DeepSeek excels at factual, technical content.

## How We Integrated

DeepSeek's API is OpenAI-compatible, making integration trivial. DeepSeek V3.2 sits in our \`auto\` and \`auto-medium-task\` routing chains as a cost-optimization option. When the router detects a task that does not require frontier reasoning or vision, it often routes to DeepSeek to save money without sacrificing quality.

## Why DeepSeek Matters

DeepSeek's impact goes beyond their own models. They proved that frontier-quality AI does not require frontier-level budgets. Their MoE (mixture of experts) architecture and training efficiency innovations have pushed every other provider to reconsider pricing.

Before DeepSeek, the cheapest "good" model cost roughly $1-3 per million tokens. After DeepSeek, that floor dropped to $0.28. This compression forced OpenAI to ship GPT-5 Mini ($0.25/$2.00), Google to drop Flash Lite to $0.02, and Mistral to price Nemo at $0.02/$0.04.

The competitive pressure from DeepSeek has made AI cheaper for everyone. That is why we consider them one of the most important providers in our ecosystem, even beyond the quality of their individual models.

## When to Use DeepSeek Through OpenPaths

- **Budget production workloads** where cost matters more than having the absolute best
- **Code generation and debugging** at 10x less cost than Claude
- **High-volume processing** where spending $0.28/1M vs $3.00/1M adds up fast
- **Tasks that do not need vision or massive context windows**`
  },
  {
    slug: 'provider-mistral',
    title: 'Mistral on OpenPaths: Europe\'s AI Champion and the Deepest Model Lineup',
    excerpt: 'Mistral offers 10 models spanning every use case from tiny edge deployment to frontier vision. Here is the full breakdown of Europe\'s leading AI lab.',
    date: '2026-03-08',
    author: 'OpenPaths Team',
    readTime: '7 min',
    tags: ['providers', 'mistral', 'models'],
    content: `Mistral launched in 2023 from Paris with a team of ex-Meta and ex-Google researchers. In under three years they have built the largest model lineup of any non-American AI company -- 10 models on OpenPaths covering everything from $0.02 ultra-cheap inference to frontier vision and code generation.

## The Mistral Lineup on OpenPaths

| Model | Price (In/Out per 1M) | Context | Best For |
|-------|----------------------|---------|----------|
| Mistral Large 3 | $0.50/$1.50 | 262K | Flagship general-purpose |
| Pixtral Large | $2.00/$6.00 | 131K | Vision, image analysis |
| Mistral Medium 3 | $0.40/$2.00 | 131K | Balanced performance |
| Magistral Medium | $0.40/$2.00 | 131K | Reasoning tasks |
| Devstral Medium | $0.40/$2.00 | 262K | Developer workflows |
| Mistral Small 3 | $0.35/$0.56 | 131K | Efficient vision |
| Codestral | $0.30/$0.90 | 256K | Code generation |
| Mistral Nemo | $0.02/$0.04 | 131K | Ultra-cheap basic tasks |
| Ministral 14B | $0.20/$0.20 | 262K | Compact vision |
| Ministral 8B | $0.15/$0.15 | 262K | Edge, minimum cost |

## Strengths

**Lineup depth.** No other provider covers as many niches. Need a frontier model? Large 3. Code specialist? Codestral. Vision? Pixtral or Small 3. Reasoning? Magistral. Developer agents? Devstral. Cheap and tiny? Nemo or Ministral. Mistral has a model for every slot.

**European data sovereignty.** For companies bound by GDPR or EU AI Act requirements, Mistral is the go-to provider. Based in Paris, they operate under European regulations and offer data processing guarantees that American and Chinese providers cannot match.

**Context windows.** Even Mistral's smaller models offer 131K-262K context windows. Codestral gives you 256K tokens of context -- enough to hold a large codebase while generating code. Mistral Large 3 and Devstral both push to 262K.

**Open-source commitment.** Several Mistral models (Nemo, Ministral 8B/14B) are open-weight, enabling self-hosting and fine-tuning. Their open-source releases have been some of the most downloaded models on Hugging Face.

**Pricing.** Mistral Large 3 at $0.50/$1.50 significantly undercuts Claude Sonnet ($3.00/$15.00) and GPT-5 ($1.25/$10.00) while delivering competitive quality. Their budget models are among the cheapest available -- Nemo at $0.02/$0.04 rivals Gemini 3.1 Flash Lite.

## Weaknesses

**Brand recognition.** Outside the developer community, Mistral is less known than OpenAI, Google, or Anthropic. This means fewer tutorials, fewer Stack Overflow answers, and a smaller community for troubleshooting.

**Frontier ceiling.** While Mistral Large 3 is excellent, it does not quite reach the peak performance of Claude Opus 4.6, Gemini 3.5 Flash, or GPT-5.4 on the hardest benchmarks. Mistral wins on value, not raw capability at the very top.

**Vision model pricing.** Pixtral Large at $2.00/$6.00 is competitive but not cheap. For vision-heavy workloads, Google's free Flash Lite or Z.AI's free GLM-4.6v Flash may be more cost-effective.

## How We Integrated

Mistral's API is OpenAI-compatible with their own endpoint at api.mistral.ai. We route to all 10 models, making Mistral one of our most deeply integrated providers. Codestral and Devstral are particularly valuable in our routing because they offer specialized coding capability at prices that make them viable for high-volume code generation.

## Mistral's Position in the Market

Mistral represents something important: proof that world-class AI does not have to come from Silicon Valley or Beijing. A Paris-based team has built models that compete on quality and win on value against companies with 10x their budget.

Their strategy of releasing many specialized models rather than one monolithic flagship is smart. Instead of trying to beat GPT-5 at everything, they offer Codestral that beats it at code, Pixtral that matches it on vision, and Nemo that costs 50x less for simple tasks. This portfolio approach lets developers pick the right tool for each job.

## When to Use Mistral Through OpenPaths

- **Code generation** with Codestral at $0.30/$0.90 -- excellent value
- **European compliance** requirements (GDPR, EU AI Act)
- **Budget production** with Nemo at $0.02/$0.04
- **Vision tasks** with Pixtral or Small 3 for image analysis
- **Edge deployment** with Ministral 8B for minimum latency and cost`
  },
  {
    slug: 'provider-groq',
    title: 'Groq on OpenPaths: The Fastest Inference on the Planet',
    excerpt: 'Groq built custom silicon specifically for LLM inference. The result: token generation speeds that make GPUs look slow. Here is what that means for developers.',
    date: '2026-03-07',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['providers', 'groq', 'models', 'performance'],
    content: `Groq does one thing and does it exceptionally well: fast inference. While every other AI company uses NVIDIA GPUs, Groq designed their own chip -- the Language Processing Unit (LPU) -- from the ground up for transformer inference. The result is token generation speeds that are genuinely hard to believe until you see them.

## The Groq Lineup on OpenPaths

| Model | Price (In/Out per 1M) | Context | Best For |
|-------|----------------------|---------|----------|
| Llama 3.3 70B | $0.59/$0.79 | 128K | Fast general-purpose |
| Llama 3.1 8B | $0.05/$0.08 | 128K | Ultra-fast, ultra-cheap |
| Mixtral 8x7B | $0.24/$0.24 | 32K | Fast MoE inference |

## Strengths

**Raw speed.** Groq's LPU delivers 500+ tokens per second on Llama 3.3 70B. For comparison, the same model on a standard GPU cluster generates roughly 80-100 tokens per second. This is not an incremental improvement -- it is a 5x jump that changes what is possible in real-time applications.

**Deterministic latency.** Because the LPU processes entire tensors at once rather than batching across GPUs, latency is predictable. There is no queuing, no variable batch sizes, no "sometimes fast, sometimes slow." Every request gets consistent performance.

**Price.** Llama 3.1 8B at $0.05/$0.08 per million tokens is one of the cheapest inference options anywhere. For applications that need hundreds of thousands of fast, simple responses -- chatbots, classification, extraction -- Groq is hard to beat.

**Open-source models.** Groq runs Meta's Llama and Mistral's Mixtral -- proven open-source models with strong community support and well-understood capabilities.

## Weaknesses

**Model selection.** Groq only hosts a few models. No GPT, no Claude, no Gemini -- just open-source models they have optimized for their hardware. If you need frontier reasoning, Groq is not the answer.

**Context window.** Mixtral tops out at 32K and Llama at 128K. No million-token context windows here.

**No vision or multimodal.** Groq's current offerings are text-only.

**Availability.** As Groq scales their custom hardware, capacity can be constrained during peak periods. Rate limits apply.

## How We Integrated

Groq's API is OpenAI-compatible. We use Groq primarily for two things: ultra-fast inference when latency is the priority, and audio transcription through their Whisper models (the fastest available at 228x real-time). Three STT models: \`whisper-large-v3-turbo\` ($0.04/hr), \`whisper-large-v3\` ($0.111/hr), and \`distil-whisper-large-v3-en\` ($0.02/hr, English only).

In our routing, Groq's Llama models serve as fast fallbacks. If a request is simple and speed matters more than depth, Llama 3.1 8B on Groq can respond in milliseconds.

## Why Custom Silicon Matters

Groq's approach challenges the assumption that NVIDIA GPUs are the only way to run AI. Their LPU architecture eliminates the memory bandwidth bottleneck that limits GPU inference speed. While GPUs wait for data to move between memory and compute, the LPU processes everything in a single pass.

This matters because inference cost is dominated by how long a chip is occupied per request. If Groq's LPU generates tokens 5x faster, it can serve 5x more requests per chip per second. That is how they offer competitive prices despite manufacturing custom silicon.

As AI moves from training (where GPUs excel) to inference (where speed and efficiency matter), custom inference hardware may become the norm. Groq is leading that transition.

## When to Use Groq Through OpenPaths

- **Real-time applications** where sub-second response time matters
- **High-throughput pipelines** processing thousands of requests per minute
- **Audio transcription** where Groq's Whisper runs at 228x real-time ($0.02-0.111/hr)
- **Budget batch processing** with Llama 3.1 8B at $0.05/$0.08`
  },
  {
    slug: 'provider-minimax',
    title: 'MiniMax on OpenPaths: The Multimodal Powerhouse From China',
    excerpt: 'MiniMax does chat, video, music, and speech -- all from one provider. Here is why this Chinese AI lab is one of the most versatile in our ecosystem.',
    date: '2026-03-07',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['providers', 'minimax', 'models'],
    content: `MiniMax is the provider most people have not heard of but probably should have. Founded in 2021 in Shanghai, they quietly built one of the most complete AI platforms in the world -- covering chat, video generation, music generation, and text-to-speech from a single company. No other provider on OpenPaths matches this breadth.

## The MiniMax Lineup on OpenPaths

**Chat:**
| Model | Price (In/Out per 1M) | Context | Best For |
|-------|----------------------|---------|----------|
| MiniMax M2.5 | $0.30/$1.20 | 1M | Long-context chat |
| MiniMax M2 | $0.26/$1.00 | 200K | General chat |

**Video:**
| Model | Price | Best For |
|-------|-------|----------|
| Hailuo 2.3 | $0.10/video | Default video generation |
| Hailuo 2.3 Fast | $0.05/video | Quick iteration |
| Hailuo 02 | $0.08/video | Previous-gen, still solid |

**Music:** Music 2.5 ($0.01/gen), Music 2.0 ($0.008/gen)

**Speech:** Speech 2.8 HD ($100/1M chars), Speech 2.8 Turbo ($60/1M chars)

## Strengths

**Multimodal breadth.** One API key for chat, video, music, and speech. No other single provider offers this range. Building an application that generates text, creates a video, adds background music, and narrates with a voice? MiniMax handles all four.

**1M context window.** M2.5 matches Google's Gemini for context length at a competitive $0.30/$1.20. For long-document processing, maintaining extensive conversation history, or analyzing large datasets, that million-token window is invaluable.

**Hailuo video quality.** Hailuo 2.3 is our default video generation model for good reason. It produces smooth, coherent motion with strong prompt adherence at just $0.10 per video. The fast variant cuts that to $0.05 while maintaining good quality.

**Music generation.** MiniMax's Music 2.5 is the only music generation model on OpenPaths. At $0.01 per generation, you can create background tracks, jingles, and full songs from text prompts. Genre specification, mood control, and lyrics input are all supported.

**Speech quality.** Speech 2.8 HD produces some of the most natural-sounding TTS output available. Emotional range, natural intonation, minimal artifacts -- it competes with ElevenLabs at a lower price point.

## Weaknesses

**Chat model ranking.** M2.5 is solid but does not reach frontier quality for complex reasoning tasks. It sits comfortably in the mid-tier -- great for everyday tasks, less competitive for hard coding or deep analysis.

**Video generation time.** Hailuo uses async task submission with polling. You submit a job, wait 30-60 seconds, and retrieve the result. Not a problem for batch workflows, but it means video generation is not real-time.

**Regional focus.** MiniMax's primary market is China, and some documentation and community resources are Chinese-first. English language support is good but not as deep as Western providers.

## How We Integrated

MiniMax has a custom API format for video, music, and speech. We built dedicated handlers for each modality:

- **Video:** Async task submission -> polling for completion -> URL retrieval
- **Music:** Similar async pattern with genre and mood parameters
- **Speech:** Streaming audio response with voice selection
- **Chat:** OpenAI-compatible through Together AI (M2.5) and direct API (M2)

Hailuo 2.3 is the first model in our \`auto-video\` routing chain. MiniMax M2.5 sits in the \`auto-medium-task\` tier as a cost-effective option with its massive context window.

## The Multimodal Future

MiniMax represents where AI is heading -- unified multimodal platforms rather than siloed text-only services. The ability to chain text generation, video creation, music composition, and speech synthesis through a single provider simplifies architecture and reduces integration overhead.

Their speed of iteration is impressive too. In 12 months they shipped M2, M2.5 (doubling context to 1M), Hailuo 2.3 (major quality jump), Music 2.5, and Speech 2.8. That is five significant model releases across four modalities from a team that is still relatively small by industry standards.

## When to Use MiniMax Through OpenPaths

- **Video generation** with Hailuo -- best quality/price default
- **Long-context processing** with M2.5's 1M window
- **Music generation** for soundtracks, jingles, background audio
- **Text-to-speech** for natural-sounding voiceovers
- **Multimodal applications** that need text + video + audio from one flow`
  },
  {
    slug: 'provider-together',
    title: 'Together AI on OpenPaths: The Open-Source Inference Platform',
    excerpt: 'Together AI hosts the best open-source models on fast infrastructure. From Qwen to FLUX, here is how they power a huge chunk of our catalog.',
    date: '2026-03-06',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['providers', 'together', 'models', 'open-source'],
    content: `Together AI is not a model builder -- they are an inference platform. They take the best open-source models from labs around the world and host them on optimized infrastructure. The result: you get access to Qwen, Kimi, GLM, DeepSeek, MiniMax, and FLUX image models through a single, reliable API.

## The Together Lineup on OpenPaths

**Chat Models:**
| Model | Price (In/Out per 1M) | Context | Origin |
|-------|----------------------|---------|--------|
| Qwen 3.5 397B | $0.60/$3.60 | 131K | Alibaba |
| Qwen 3 Coder | $0.50/$1.20 | 131K | Alibaba |
| Kimi K2.5 | $0.50/$2.80 | 131K | Moonshot AI |
| GLM-5 | $1.00/$3.20 | 202K | Z.AI (Zhipu) |
| GLM-4.7 | $0.45/$2.00 | 202K | Z.AI (Zhipu) |
| MiniMax M2.5 | $0.30/$1.20 | 1M | MiniMax |
| DeepSeek R1 | $3.00/$7.00 | 128K | DeepSeek |
| DeepSeek V3.1 | $0.60/$1.70 | 128K | DeepSeek |

**Image Models:**
| Model | Price per Image |
|-------|----------------|
| FLUX Pro | $0.04 |
| FLUX Dev | $0.015 |
| FLUX Schnell | $0.003 |
| Stable Diffusion 3 | $0.002 |

## Strengths

**Model diversity.** Together hosts models from Alibaba, Moonshot, Z.AI, DeepSeek, MiniMax, and Black Forest Labs -- all through one API. This gives OpenPaths access to the best of the open-source world without maintaining separate integrations for each origin lab.

**Inference optimization.** Together specializes in making open-source models run fast. They use custom kernels, quantization techniques, and batching strategies to squeeze maximum performance from GPU clusters. Models on Together often run faster than on the origin provider's own API.

**Image generation.** Together hosts the entire FLUX family and Stable Diffusion 3, making them our primary image generation backend for non-Netwrck models. FLUX Schnell at $0.003 per image through Together is one of the cheapest image generation options anywhere.

**Reliability.** As a dedicated inference platform, Together has invested heavily in uptime, auto-scaling, and redundancy. Their SLA and availability track record is strong.

**Open-source ecosystem.** Together actively contributes to the open-source AI community through research, model fine-tuning, and tooling. They are not just hosting models -- they are improving them.

## Weaknesses

**Markup over direct API.** In some cases, running a model through Together costs more than using the origin provider directly. DeepSeek R1 on Together ($3.00/$7.00) versus DeepSeek direct ($0.28/$0.42) is a 10x price difference. The tradeoff is reliability, speed, and not having to manage the China-based API.

**No proprietary models.** Together does not build their own models. If you need Claude, GPT, or Gemini, you need those providers directly. Together's value is in making open-source models accessible.

**Dependent on upstream.** When an origin lab updates or deprecates a model, Together has to follow. This creates a dependency chain that can cause temporary gaps.

## How We Integrated

Together's API is OpenAI-compatible for chat and has a dedicated image generation endpoint. We route to Together for:

- **Qwen models** -- Together is the most reliable Western host for Alibaba's Qwen family
- **FLUX image generation** -- Our primary FLUX backend
- **GLM models** -- More reliable than routing to Z.AI directly for international users
- **Redundancy** -- Together serves as a fallback for models that we also route to directly (like DeepSeek)

## The Platform Play

Together represents an important trend: the separation of model creation from model serving. Not every lab needs to build inference infrastructure. Some (like Alibaba, Moonshot, Zhipu) are better at research and training. Together handles the serving part, and OpenPaths handles the routing and unification on top.

This three-layer architecture -- model labs build, platforms like Together serve, routers like OpenPaths unify -- is how the industry is settling. Developers should not need to think about which GPU cluster is running their request. They should think about what model is best for their task.

## When to Use Together Through OpenPaths

- You typically do not choose Together explicitly -- our router picks it when it is the best host for a given model
- **Qwen models** for strong open-source reasoning and coding
- **Kimi K2.5** for Moonshot's unique reasoning approach
- **FLUX image generation** at every price tier
- **GLM models** when you want Z.AI quality with Western infrastructure reliability`
  },
  {
    slug: 'provider-zai',
    title: 'Z.AI on OpenPaths: GLM Models, Free Vision, and China\'s Dark Horse',
    excerpt: 'Z.AI (Zhipu) builds the GLM model family -- including the best free vision model available. Here is what makes this Beijing-based lab worth watching.',
    date: '2026-03-06',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['providers', 'zai', 'models'],
    content: `Z.AI (formerly Zhipu AI) is a Beijing-based AI lab spun out of Tsinghua University. They build the GLM (General Language Model) family, which has quietly become one of the most capable model lineups from China. Their secret weapon: a free vision model that is genuinely useful.

## The Z.AI Lineup on OpenPaths

| Model | Price (In/Out per 1M) | Context | Best For |
|-------|----------------------|---------|----------|
| GLM-5 | $1.00/$3.20 | 202K | Flagship reasoning |
| GLM-4.7 | $0.45/$2.00 | 202K | Efficient general |
| GLM-4.6v | $1.50/$5.00 | 128K | Paid vision |
| GLM-4.6v Flash | Free | 128K | Free vision |
| GLM Image | $0.015/image | N/A | Image generation |

Note: GLM-5 and GLM-4.7 are also available through Together AI for improved reliability.

## Strengths

**Free vision.** GLM-4.6v Flash is a free model that accepts images. It handles OCR, image description, visual Q&A, and basic image analysis at low cost. For prototyping multimodal applications or processing images at scale with a tight budget, it is the best option available.

**202K context.** GLM-5 and GLM-4.7 both offer 202K token context windows -- larger than Anthropic's 200K, suitable for long documents and extended conversations.

**Image generation.** GLM Image generates at high native resolutions (up to 1728x960) and handles both English and Chinese text rendering well. At $0.015 per image, it occupies a sweet spot between budget options like SD3 ($0.002) and premium models like FLUX Pro ($0.04).

**Bilingual excellence.** Z.AI models handle Chinese and English equally well. For applications serving Chinese-speaking users or processing Chinese-language content, GLM models are a natural choice.

**Academic foundation.** Born from Tsinghua University research, Z.AI's models reflect strong theoretical foundations in natural language understanding, knowledge representation, and multimodal learning.

## Weaknesses

**International API reliability.** Routing to Z.AI's API from outside China can have variable latency and occasional connectivity issues. This is why we also route GLM models through Together AI as a fallback.

**Ecosystem.** Documentation and community resources are primarily Chinese-language. English support exists but is less comprehensive than Western providers.

**Model naming.** The GLM version numbering (5, 4.7, 4.6v, 4.6v Flash) is less intuitive than competitors' naming conventions.

## How We Integrated

Z.AI uses a custom API format at api.z.ai. Our integration handles their specific endpoint structure, authentication, and response format. For chat models, we also maintain the Together AI route as a more reliable international pathway.

GLM-4.6v Flash sits in our free tier routing as the go-to free vision model. When a user requests a free model that can handle images, GLM-4.6v Flash is what they get.

## Why Z.AI Matters

Z.AI demonstrates that the AI landscape is genuinely global. Their GLM-5 competes on benchmarks with models from companies that have raised 10x more funding. Their free vision model offers capabilities that Western providers charge for.

The Tsinghua connection also means Z.AI benefits from one of China's top computer science research programs. Their models incorporate techniques from knowledge graph research, multilingual NLP, and multimodal learning that reflect years of academic groundwork.

## When to Use Z.AI Through OpenPaths

- **Free vision tasks** with GLM-4.6v Flash
- **Chinese/bilingual applications** where GLM excels
- **High-resolution image generation** with GLM Image (up to 1728x960)
- **Budget general tasks** with GLM-4.7 at $0.45/$2.00`
  },
  {
    slug: 'provider-openrouter',
    title: 'OpenRouter on OpenPaths: 600+ Models as Our Safety Net',
    excerpt: 'OpenRouter is both a competitor and a partner. Here is how we use the largest model gateway as our fallback layer.',
    date: '2026-03-05',
    author: 'OpenPaths Team',
    readTime: '5 min',
    tags: ['providers', 'openrouter', 'models'],
    content: `OpenRouter is the largest model gateway in the AI ecosystem, providing access to over 600 models from dozens of providers. They are also our fallback layer. When primary providers are unhealthy or unavailable, OpenPaths routes through OpenRouter to maintain uptime.

## The OpenRouter Lineup on OpenPaths

We primarily use OpenRouter for free-tier models:

| Model | Provider | Context | Features |
|-------|----------|---------|----------|
| StepFun Flash | StepFun | 256K | Free, tool use |
| Solar Pro 3 | Upstage | 128K | Free, tool use |
| Nemotron Nano 30B | NVIDIA | 256K | Free, reasoning |
| Arcee Trinity | Arcee AI | 131K | Free, tool use |

Plus fallback access to hundreds of additional models when primary routes are down.

## Strengths

**Model catalog size.** 600+ models is unmatched. If a model exists in the AI ecosystem, OpenRouter probably hosts it. This makes them an invaluable fallback -- no matter what model a user requests, there is likely an OpenRouter route available.

**Free tiers.** OpenRouter negotiates free access to models from smaller providers like StepFun, Upstage, NVIDIA (Nemotron), Arcee, and Liquid. These free models are genuinely useful for prototyping and low-volume use cases.

**Reliability.** OpenRouter has been operating since 2023 and has built robust infrastructure for handling high request volumes across many providers. Their routing and load balancing is mature.

**Community.** OpenRouter has the largest community of any model gateway, with detailed pricing pages, model comparisons, and activity rankings. Their intelligence benchmark provides useful signal for model quality.

## Weaknesses

**Markup.** OpenRouter adds a margin on top of provider pricing. For high-volume production use, routing directly to providers (as OpenPaths does) is cheaper than going through OpenRouter for every request.

**Latency.** Adding an extra hop through OpenRouter's gateway increases request latency compared to direct provider connections. For latency-sensitive applications, this matters.

**Less control.** When routing through OpenRouter, you are subject to their rate limits, queuing, and routing decisions. Direct integrations give OpenPaths more control over request handling.

## How We Integrated

OpenRouter uses an OpenAI-compatible API with additional headers (HTTP-Referer, X-Title). We use it in two ways:

1. **Free model access.** StepFun Flash, Solar Pro 3, Nemotron Nano, and Arcee Trinity are only available through OpenRouter's free tier agreements.

2. **Fallback routing.** Every model in our catalog has a fallback chain. At the end of most chains, OpenRouter sits as the last resort. If our direct connection to Anthropic is down AND our Together AI fallback fails, the request goes through OpenRouter.

This dual role -- free model host and universal fallback -- makes OpenRouter one of our most important partners despite also being a competitor.

## The Coopetition Model

Our relationship with OpenRouter is a good example of how the AI infrastructure market works in 2026. We compete on user-facing features (auto routing, multimodal, crypto payments) while cooperating on reliability (fallback routing) and free model access.

This makes the ecosystem better for developers. If you use OpenPaths, you get our direct integrations for speed and cost, plus OpenRouter's breadth as a safety net. You are effectively getting two routers for the price of one.

## When OpenRouter Routes Activate

- When a primary provider is unhealthy or rate-limited
- When you request a model we do not have a direct integration for
- When you use one of the free-tier models (StepFun, Solar, Nemotron, Arcee)
- As the last fallback in any routing chain`
  },
  {
    slug: 'provider-netwrck',
    title: 'Netwrck on OpenPaths: Our First-Party Partner for Art and Video',
    excerpt: 'Netwrck powers RA1, ZImage, Wan, LTX Video, and RA2V on OpenPaths. Here is how our closest creative AI partner builds the tools artists actually use.',
    date: '2026-03-05',
    author: 'OpenPaths Team',
    readTime: '6 min',
    tags: ['providers', 'netwrck', 'art generation', 'video generation'],
    content: `Netwrck is not just a provider on OpenPaths -- they are a first-party partner. Their creative AI models power some of the most popular features on our platform: RA1 image generation, ZImage anime art, and a full video generation pipeline with Wan, LTX Video, and RA2V.

## The Netwrck Lineup on OpenPaths

**Image Generation:**
| Model | Price per Image | Best For |
|-------|----------------|----------|
| RA1 Art Generator | $0.04 | General art, photorealism |
| ZImage | $0.007 | Anime, illustration |

**Video Generation:**
| Model | Price per Video | Best For |
|-------|----------------|----------|
| Wan Video | $0.30 | Cinematic, complex scenes |
| LTX Video | $0.05 | Budget, prototyping |
| RA2V | $1.00 | Maximum quality |

## Strengths

**RA1 image quality.** RA1 consistently produces high-quality images across diverse styles -- photorealistic portraits, landscapes, concept art, product renders, architectural visualization. At $0.04 per image, it matches FLUX Pro quality at the same price point while handling a broader range of artistic styles without complex prompt engineering.

**ZImage for anime.** ZImage is purpose-built for anime and illustration-style art. At $0.007 per image, it is the cheapest quality option for manga-style characters, anime scenes, and stylized illustrations. If your application generates anime content, ZImage is the obvious choice.

**Video pipeline depth.** Having three video models at different price points means developers can match quality to budget precisely. LTX for prototyping at $0.05, Wan for production at $0.30, RA2V for final renders at $1.00. No other single provider offers this range.

**Image-to-video.** Wan supports animating still images -- provide an image URL and a prompt describing the motion, and Wan brings it to life. This is a workflow that creative professionals use daily for social media content, product demonstrations, and animated art.

**Tight integration.** As a first-party partner, Netwrck's API integration is maintained directly by our team. This means faster bug fixes, coordinated feature releases, and infrastructure that is optimized for the OpenPaths pipeline.

## Weaknesses

**No chat models.** Netwrck is creative-only. No LLMs, no embeddings, no transcription. For text tasks, you need another provider.

**Resolution constraints.** RA1's maximum native resolution is 1360x768. For higher resolutions, OpenPaths' automatic resize pipeline kicks in, which adds processing time and converts URL responses to base64.

**Video generation time.** Like all video models, generation is not instant. Expect 10-60 seconds depending on the model and complexity. Not suitable for real-time video generation.

## How We Integrated

Netwrck uses custom API endpoints at netwrck.com/api/. Our integration is deeper than typical provider connections:

- **Direct coordination** on model updates and capability additions
- **Shared infrastructure monitoring** for uptime and performance
- **Priority for new model access** -- when Netwrck ships a new model, OpenPaths gets it first
- **Custom resolution handling** that wraps Netwrck's supported sizes with our automatic resize pipeline

RA1 is the first model in our \`auto-image\` routing chain. ZImage slots in for anime-detected prompts. Wan leads the video chain for quality requests.

## The Creative AI Landscape

Image and video generation are evolving faster than any other AI modality. Two years ago, the best image models produced 512x512 images with obvious artifacts. Today, RA1 generates photorealistic images at 1360x768 for four cents each.

Video generation has moved even faster. The jump from LTX (early generation, experimental quality) to Wan (cinematic coherence) to RA2V (production-grade) happened in under a year. Each generation roughly doubles quality while costs stay flat or decrease.

Netwrck sits at the center of this creative AI acceleration. Their focus on artistic quality over benchmark scores means their models are tuned for what creative professionals actually need -- consistent style, good composition, natural lighting, and reliable prompt adherence.

## When to Use Netwrck Through OpenPaths

- **Image generation** with RA1 as the default choice
- **Anime and illustration** with ZImage at $0.007
- **Video production** with Wan for cinematic quality
- **Budget video prototyping** with LTX at $0.05
- **Maximum quality video** with RA2V when every frame counts`
  },
  {
    slug: 'provider-text-generator',
    title: 'Text-Generator.io on OpenPaths: The Embedding Specialist',
    excerpt: 'Text-Generator.io powers our ModernBERT embeddings -- the backbone of semantic search and RAG pipelines. Here is how a focused specialist outperforms generalists.',
    date: '2026-03-04',
    author: 'OpenPaths Team',
    readTime: '4 min',
    tags: ['providers', 'text-generator', 'embedding'],
    content: `Text-Generator.io is the smallest provider on OpenPaths by model count -- they offer exactly one model. But that model, ModernBERT Embedding, is the backbone of every semantic search and RAG pipeline that runs through our platform.

## The Text-Generator.io Lineup

| Model | Price (per 1M tokens) | Context | Best For |
|-------|----------------------|---------|----------|
| ModernBERT Embedding | $0.10 | 8K | Search, RAG, similarity |

## Strengths

**ModernBERT quality.** ModernBERT represents the latest generation of embedding models -- trained on more data, with better fine-tuning for retrieval tasks, and improved handling of domain-specific vocabulary. The embeddings it produces are measurably better at semantic similarity, document clustering, and retrieval than previous-generation models.

**Specialization.** Text-Generator.io does one thing well. They are not distracted by chat models, image generation, or other modalities. Their entire infrastructure is optimized for embedding generation -- fast, reliable, and cost-effective.

**Price.** At $0.10 per million tokens, ModernBERT embeddings are affordable for even high-volume applications. Processing a million documents at 500 tokens each costs about $0.05.

**API reliability.** As a first-party partner, Text-Generator.io's integration is maintained directly by our team. The API is simple, fast, and has excellent uptime.

## Weaknesses

**Single model.** No alternatives if ModernBERT does not suit your use case. For applications that need multilingual embeddings, very long context embeddings, or vision-capable embeddings, other options exist.

**8K context limit.** Each embedding request handles up to 8,192 tokens. For very long documents, you need to chunk before embedding. Models like Gemini's embedding support longer contexts.

**No multimodal.** Text only. For image-text embeddings, you would need NVIDIA's Nemotron Embed (available free through OpenRouter) or another multimodal embedding model.

## How We Integrated

Text-Generator.io uses a REST API at api.text-generator.io. Our embedding handler sends text to their feature extraction endpoint and receives back dense vector embeddings in the OpenAI-compatible format.

We also run our own local embedding model (gobed) for the auto router's semantic matching. For user-facing embedding requests through the API, ModernBERT is the default and recommended option.

## Why Embeddings Matter

Embeddings are the unsung hero of AI applications. They power:

- **Semantic search** -- find documents by meaning, not just keywords
- **RAG pipelines** -- retrieve relevant context before sending to an LLM
- **Clustering** -- group similar documents, support tickets, or user queries
- **Recommendation** -- find items similar to what a user has interacted with
- **Deduplication** -- identify near-duplicate content at scale

Every production AI application that handles more than a few documents uses embeddings somewhere. Having a reliable, cheap, high-quality embedding provider is infrastructure-level important.

## When to Use Text-Generator.io Through OpenPaths

- **Building RAG pipelines** that need reliable document retrieval
- **Semantic search** over document collections
- **Text similarity** for clustering, deduplication, or recommendation
- **Any embedding task** where ModernBERT's quality and $0.10/1M pricing fits`
  },
  {
    slug: 'provider-fal',
    title: 'Fal on OpenPaths: Serverless AI Inference for Image Generation',
    excerpt: 'Fal runs FLUX Klein 4B -- a compact image model that generates fast on serverless infrastructure. Here is how serverless inference is changing AI deployment.',
    date: '2026-03-04',
    author: 'OpenPaths Team',
    readTime: '4 min',
    tags: ['providers', 'fal', 'art generation'],
    content: `Fal takes a different approach to AI inference. Instead of maintaining always-on GPU clusters, they run models on serverless infrastructure that scales to zero when idle and spins up on demand. The result: fast inference for image generation without paying for idle capacity.

## The Fal Lineup on OpenPaths

| Model | Price per Image | Best For |
|-------|----------------|----------|
| FLUX Klein 4B | $0.02 | Compact, fast image generation |

## Strengths

**Serverless economics.** Fal's pay-per-use model means you never pay for idle GPUs. For applications with variable or bursty image generation needs, this is significantly cheaper than maintaining dedicated inference capacity.

**FLUX Klein 4B.** Klein is a compact version of the FLUX architecture -- smaller parameter count, faster generation, lower cost. At $0.02 per image, it sits between FLUX Schnell ($0.003) and FLUX Dev ($0.015) on price while offering good quality for its size class.

**Cold start optimization.** Fal has invested heavily in minimizing serverless cold start times for AI models. Their infrastructure pre-warms model weights and uses optimized container orchestration to reduce the time from request to first pixel.

**Developer experience.** Fal's API is clean, well-documented, and includes built-in webhook support for async workflows. Their SDKs for Python and JavaScript are maintained actively.

## Weaknesses

**Limited model selection.** Only one model on OpenPaths currently. Fal's broader catalog includes more models, but our integration focuses on Klein as a fast, cheap image option.

**Cold starts.** Despite optimization, serverless inference can have variable latency. The first request after a period of inactivity may take longer than subsequent requests.

**No chat or text models.** Fal focuses on media generation and audio processing. For text tasks, you need another provider.

## How We Integrated

Fal's API uses their own endpoint format at fal.run/. Our integration submits image generation requests and handles the response, which can be synchronous for fast models like Klein or async with polling for heavier workloads.

FLUX Klein sits in our image generation catalog as a compact, fast option. When the router determines a request needs quick image generation and maximum quality is not the priority, Klein is a candidate.

## The Serverless Inference Trend

Fal represents a broader trend in AI infrastructure: the move from "reserve a GPU for 24/7" to "pay per inference." For most applications, GPU utilization hovers between 5-30%. Paying for 100% of a GPU's time when you use 10% of it is wasteful.

Serverless inference solves this by sharing GPU capacity across many users. The tradeoff is cold start latency and less control over hardware. For latency-insensitive workloads like image generation (where users expect a few seconds of generation time anyway), serverless is often the better economic choice.

## When to Use Fal Through OpenPaths

- **Fast image generation** where Klein's speed-to-cost ratio fits
- **Variable workloads** that benefit from serverless scaling
- **Prototyping** where $0.02 per image keeps costs minimal
- **Compact model inference** where smaller parameter counts are acceptable`
  }
];
