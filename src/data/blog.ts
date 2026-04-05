export interface BlogPost {
  slug: string;
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

- \`auto-easy-task\` -- Routes to cheapest models (Gemini Flash Lite, MiniMax M2.5 Highspeed, GPT-4o Mini). For simple lookups, formatting, summarization. Starting at $0.02/1M input tokens.
- \`auto-medium-task\` -- Routes to mid-tier models (Claude Sonnet, Gemini Flash, DeepSeek, MiniMax M2.5). For coding, analysis, moderate complexity.
- \`auto-think\` -- Routes by reasoning depth and assigns \`none\`, \`low\`, \`medium\`, or \`high\` thinking automatically.
- \`auto\` -- Full intelligent routing across all tiers based on task complexity.

**Automatic fallbacks.** If Claude is down, your request falls through to GPT-5.2 or Gemini. No code changes needed.

**Unified billing.** One balance, one dashboard. Pay with Stripe or crypto (SOL/USDC).

## New: Auto Task Tiers

We added two new auto-routing models designed for cost optimization:

### auto-easy-task

For simple tasks that don't need a $15/1M-token model. The router picks from:
- **Gemini Flash Lite** -- $0.02 input, 1M context, vision
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

**Free:** 29 models (8%) -- Completely free to use. Qwen leads with 6 free models, Google has 5. These aren't toy models either -- Gemini Flash Lite gives you a 1M context window at zero cost.

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

The 1M+ club includes nearly every Gemini model, several Grok variants, and a few Qwen models. Google is clearly betting that massive context is a competitive advantage -- and at $0.10/1M tokens for Gemini Flash Lite, they might be right.

## The Best Value in AI

We calculated a "value score" -- context window size divided by price per million tokens. The winners might surprise you:

| Model | Context | Input $/1M | Value Score |
|-------|---------|-----------|-------------|
| Gemini 2.0 Flash Lite | 1M | $0.07 | 13.9M |
| Gemini 2.5 Flash Lite | 1M | $0.10 | 10.4M |
| GPT-4.1 Nano | 1M | $0.10 | 10.4M |
| Grok 4.1 Fast | 2M | $0.20 | 10.0M |
| GPT-5 Nano | 400K | $0.05 | 8.0M |

Google dominates value. Their Flash Lite models give you a million tokens of context for seven cents. For comparison, o1-pro gives you 200K context for $150 -- a value score of just 1,333. That's a 10,000x difference in tokens-per-dollar.

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

**The Big 5** (OpenAI, Anthropic, Google, Meta, Mistral) still dominate quality benchmarks. Gemini 3.1 Pro leads OpenRouter's intelligence rankings at 57.2, followed by GPT-5.3 Codex (54.0) and Claude Opus 4.6 (53.0).

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

**3. Pick the winner** -- The highest-scoring model gets the request. If it fails or is unhealthy, the router falls back through the chain: for \`auto\` chat, that's Gemini 3.1 Pro -> GPT-5.2 -> DeepSeek -> Claude Sonnet -> Grok -> OpenRouter fallbacks.

## Auto Image

\`auto-image\` works the same way but routes between image generators. The default chain is ra1 -> FLUX Pro -> zimage -> GLM Image. A prompt like "photorealistic portrait of a woman" routes differently than "anime character with sword" -- the former goes to ra1 or FLUX, the latter to zimage.

## Auto Video

\`auto-video\` routes between Hailuo 2.3 -> Wan -> LTX Video -> ra2v. Short creative clips go to Hailuo (fast, cheap). Longer or higher-quality requests route to Wan or ra2v.

## Using It

\`\`\`python
client = openai.OpenAI(
    base_url="https://api.openpaths.io/v1",
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

**Gemini 3.1 Pro** ($2.00/$12.00 per 1M) -- Our default auto model for good reason. 1M context window, strong reasoning, great at code. Google's latest and it shows. Best overall value at the frontier level.

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

**Gemini Flash Lite** ($0.00/$0.00) -- Free vision model from Google. 1M context. Great for prototyping.

**GLM 4.6v Flash** ($0.00/$0.00) -- Free vision model from Z.AI. Solid for basic visual tasks.

**Step Flash, Solar Pro 3, Nemotron Nano** -- Various free models good for testing and low-stakes tasks.

## The Code Specialists

**Qwen3 Coder** ($0.50/$1.20) -- Purpose-built for code. Strong at completion, refactoring, and generation across many languages.

**Codestral** ($0.30/$0.90) -- Mistral's code model. 256K context window is great for large codebases.

**Codex Mini** ($1.50/$6.00) -- OpenAI's latest code-focused model.

## Decision Framework

Ask yourself these questions:

1. **Does quality matter most?** -> Gemini 3.1 Pro or Claude Opus 4.6
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
    "https://api.openpaths.io/v1/video/generations",
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
    "https://api.openpaths.io/v1/video/generations",
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
    slug: 'openpath-vs-openrouter',
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
    "https://api.openpaths.io/v1/music/generations",
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
    "https://api.openpaths.io/v1/audio/speech",
    headers={"Authorization": "Bearer op_..."},
    json={
        "model": "speech-2.8-hd",
        "input": "Welcome to OpenPaths. The open source model router.",
        "voice": "alloy"
    }
)
# response contains audio data
\`\`\`

## Transcription

For the reverse direction -- audio to text -- we support Whisper through both Groq (fast) and Fal (accurate):

\`\`\`python
with open("recording.mp3", "rb") as f:
    response = requests.post(
        "https://api.openpaths.io/v1/audio/transcriptions",
        headers={"Authorization": "Bearer op_..."},
        files={"file": f},
        data={"model": "whisper-large-v3-turbo"}
    )
print(response.json()["text"])
\`\`\`

## Pricing Summary

| Model | Price | Use Case |
|-------|-------|----------|
| Music 2.5 | $0.01/gen | Music generation |
| Music 2.0 | $0.008/gen | Budget music gen |
| Speech 2.8 HD | $100/1M chars | Premium TTS |
| Speech 2.8 Turbo | $60/1M chars | Fast TTS |
| Whisper (Groq) | ~$0.01/min | Fast transcription |
| Whisper (Fal) | ~$0.01/min | Accurate transcription |`
  },
  {
    slug: 'free-ai-models',
    title: 'Free AI Models You Can Use Right Now on OpenPaths',
    excerpt: 'Several high-quality models are available at zero cost through OpenPaths. Here is what they are and what they can do.',
    date: '2026-02-05',
    author: 'OpenPaths Team',
    readTime: '3 min',
    tags: ['models', 'free', 'guide'],
    content: `Not everything costs money. OpenPaths routes to several free models through OpenRouter's free tier and Z.AI's free offerings. Here's what's available at $0.00.

## Free Chat Models

**GLM 4.6v Flash** (Z.AI) -- A free vision model that accepts both text and images. 128K context. Good for basic visual analysis, OCR, and image description tasks. Surprisingly capable for free.

**Gemini Flash Lite** (Google) -- 1M context window at zero cost. Fast, supports vision. The best free model for processing large documents.

**Step Flash** (StepFun) -- 256K context, tool support. A solid general-purpose free model.

**Solar Pro 3** (Upstage) -- 128K context with tool support. Good for Korean and English tasks.

**Nemotron Nano** (NVIDIA) -- 256K context. NVIDIA's lightweight free model with tool support.

**Arcee Trinity / Trinity Mini** (Arcee AI) -- 131K context. Two tiers of the same model family, both free with tool support.

**LFM Thinking / LFM Instruct** (Liquid) -- 32K context. Smaller but fast. The "thinking" variant shows its reasoning steps.

## Free Embeddings

**Nemotron Embed** (NVIDIA) -- Free vision-capable embedding model through OpenRouter. 8K context.

**OpenPaths Embed** (gobed) -- Our first-party embedding model. Runs locally in-process. $0.002/1M tokens -- practically free.

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
  }
];
