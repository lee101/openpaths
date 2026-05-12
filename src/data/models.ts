export type Tag = 'programming' | 'roleplay' | 'art generation' | 'video generation' | 'audio' | 'embedding' | 'general' | 'vision' | 'fast' | 'reasoning' | 'open-source' | 'free' | 'agentic';

export type SortOption = 'popular' | 'newest' | 'price-low' | 'price-high' | 'context-high';

export interface Model {
  id: string;
  name: string;
  provider: string;
  description: string;
  contextLength: string;
  priceInput: number;
  priceOutput: number;
  tags: Tag[];
  released: string;
  popularity: number;
  pricingType?: 'token' | 'request' | 'chars' | 'hour' | 'second' | 'megapixel';
}

export function parseContextLength(ctx: string): number {
  if (ctx === 'N/A') return 0;
  const match = ctx.match(/([\d.]+)([KMB]?)/);
  if (!match) return 0;
  const num = parseFloat(match[1]);
  const unit = match[2];
  if (unit === 'M') return num * 1_000_000;
  if (unit === 'K') return num * 1_000;
  return num;
}

export const models: Model[] = [
  // --- First-Party Partners (pinned) ---
  {
    id: 'ra1',
    name: 'RA1 Art Generator',
    provider: 'Netwrck',
    description: 'State-of-the-art image generation with incredible photorealism. First-party from netwrck.com.',
    contextLength: 'N/A',
    priceInput: 0.04,
    priceOutput: 0,
    tags: ['art generation'],
    released: '2025-06-01',
    popularity: -2,
    pricingType: 'request'
  },
  {
    id: 'openpaths-embed',
    name: 'OpenPaths Embed',
    provider: 'OpenPaths',
    description: 'An extremely efficient local embedding model for the lowest-cost text embeddings and similarity checks. Open source at lee101/gobed, so it is portable across environments.',
    contextLength: '256',
    priceInput: 0.001,
    priceOutput: 0,
    tags: ['embedding'],
    released: '2026-04-23',
    popularity: -2,
    pricingType: 'request'
  },
  {
    id: 'text-embedding',
    name: 'ModernBERT Embedding',
    provider: 'Text-Generator.io',
    description: 'High-quality text embeddings powered by ModernBERT. First-party from text-generator.io.',
    contextLength: '8K',
    priceInput: 0.10,
    priceOutput: 0,
    tags: ['embedding'],
    released: '2025-06-01',
    popularity: -1
  },
  {
    id: 'gemini-embedding-001',
    name: 'Gemini Embedding 001',
    provider: 'Google',
    description: 'Google text embedding model for search, retrieval, clustering, and RAG. OpenPaths exposes it through the standard embeddings endpoint.',
    contextLength: '2K',
    priceInput: 0.15,
    priceOutput: 0,
    tags: ['embedding'],
    released: '2025-06-01',
    popularity: 8
  },
  {
    id: 'gemini-embedding-2-preview',
    name: 'Gemini Embedding 2 Preview',
    provider: 'Google',
    description: 'Google’s newer embedding model with 8K text context and multimodal pricing on the upstream API. OpenPaths exposes the text embedding path today.',
    contextLength: '8K',
    priceInput: 0.20,
    priceOutput: 0,
    tags: ['embedding'],
    released: '2026-04-17',
    popularity: 7
  },

  // --- OpenPaths Auto Tiers ---
  {
    id: 'auto-easy-task',
    name: 'Auto Easy Task',
    provider: 'OpenPaths',
    description: 'Low-cost routing across GPT-5.4 Nano, Gemini Flash tiers, Claude Haiku, and DeepSeek V4 Flash for simple or sensitive classifiers.',
    contextLength: '1M',
    priceInput: 0.02,
    priceOutput: 0.10,
    tags: ['fast', 'general'],
    released: '2026-03-03',
    popularity: 0
  },
  {
    id: 'auto-medium-task',
    name: 'Auto Medium Task',
    provider: 'OpenPaths',
    description: 'Balanced routing led by GPT-5.4 Mini with Claude Sonnet, Gemini 2.5 Flash, and DeepSeek V4 Flash for sensitive roleplay, fringe, and policy-heavy prompts.',
    contextLength: '400K',
    priceInput: 0.75,
    priceOutput: 4.50,
    tags: ['programming', 'general', 'reasoning'],
    released: '2026-03-03',
    popularity: 0
  },
  {
    id: 'auto-think',
    name: 'Auto Think',
    provider: 'OpenPaths',
    description: 'Embedding-routed reasoning tier that picks none, low, medium, or high thinking; direct thinking models can use the same classifier with reasoning_effort="auto".',
    contextLength: '400K',
    priceInput: 0.75,
    priceOutput: 4.50,
    tags: ['programming', 'general', 'reasoning'],
    released: '2026-03-17',
    popularity: 0
  },

  // --- Anthropic ---
  {
    id: 'claude-opus-4-7',
    name: 'Claude Opus 4.7',
    provider: 'Anthropic',
    description: 'Latest flagship Claude model. State-of-the-art reasoning, coding, and long-horizon agentic tasks.',
    contextLength: '200K',
    priceInput: 5.00,
    priceOutput: 25.00,
    tags: ['programming', 'reasoning', 'general', 'vision'],
    released: '2026-04-15',
    popularity: 1
  },
  {
    id: 'claude-opus-4-6',
    name: 'Claude Opus 4.6',
    provider: 'Anthropic',
    description: 'Previous-gen Opus. Strong reasoning, coding, and agentic tasks.',
    contextLength: '200K',
    priceInput: 5.00,
    priceOutput: 25.00,
    tags: ['programming', 'reasoning', 'general', 'vision'],
    released: '2026-02-15',
    popularity: 3
  },
  {
    id: 'claude-sonnet-4-6',
    name: 'Claude Sonnet 4.6',
    provider: 'Anthropic',
    description: 'Best balance of speed and intelligence. Ideal for coding, analysis, and daily tasks.',
    contextLength: '200K',
    priceInput: 3.00,
    priceOutput: 15.00,
    tags: ['programming', 'reasoning', 'general', 'vision'],
    released: '2026-02-15',
    popularity: 1
  },
  {
    id: 'claude-haiku-4-5-20251001',
    name: 'Claude Haiku 4.5',
    provider: 'Anthropic',
    description: 'Fast, affordable model with strong capabilities for high-volume tasks.',
    contextLength: '200K',
    priceInput: 1.00,
    priceOutput: 5.00,
    tags: ['fast', 'general', 'vision', 'programming'],
    released: '2025-10-01',
    popularity: 10
  },
  {
    id: 'claude-opus-4-5-20251101',
    name: 'Claude Opus 4.5',
    provider: 'Anthropic',
    description: 'Previous-gen Opus with excellent reasoning and creative writing.',
    contextLength: '200K',
    priceInput: 5.00,
    priceOutput: 25.00,
    tags: ['reasoning', 'general', 'vision'],
    released: '2025-11-01',
    popularity: 30
  },
  {
    id: 'claude-sonnet-4-5-20250929',
    name: 'Claude Sonnet 4.5',
    provider: 'Anthropic',
    description: 'Previous-gen Sonnet with strong all-around performance.',
    contextLength: '200K',
    priceInput: 3.00,
    priceOutput: 15.00,
    tags: ['programming', 'general', 'vision'],
    released: '2025-09-29',
    popularity: 31
  },

  // --- OpenAI ---
  {
    id: 'gpt-5-chat-latest',
    name: 'GPT-5 Chat Latest',
    provider: 'OpenAI',
    description: 'ChatGPT-style GPT-5 alias for conversational and creative workloads.',
    contextLength: '400K',
    priceInput: 1.25,
    priceOutput: 10.00,
    tags: ['general', 'vision', 'programming'],
    released: '2026-02-22',
    popularity: 3
  },
  {
    id: 'gpt-5.5',
    name: 'GPT-5.5',
    provider: 'OpenAI',
    description: 'Latest OpenAI flagship model with long context, strong reasoning, and multimodal support.',
    contextLength: '1.05M',
    priceInput: 5.00,
    priceOutput: 30.00,
    tags: ['general', 'vision', 'programming', 'reasoning'],
    released: '2026-04-23',
    popularity: 4
  },
  {
    id: 'gpt-5.4',
    name: 'GPT-5.4',
    provider: 'OpenAI',
    description: 'OpenAI flagship model with long context, strong reasoning, and multimodal support.',
    contextLength: '1.05M',
    priceInput: 2.50,
    priceOutput: 15.00,
    tags: ['general', 'vision', 'programming', 'reasoning'],
    released: '2026-03-05',
    popularity: 4
  },
  {
    id: 'gpt-5-codex',
    name: 'GPT-5 Codex',
    provider: 'OpenAI',
    description: 'Latest OpenAI coding model for software engineering and agentic coding tasks.',
    contextLength: '400K',
    priceInput: 1.25,
    priceOutput: 10.00,
    tags: ['programming', 'reasoning', 'vision'],
    released: '2026-02-07',
    popularity: 15
  },
  {
    id: 'gpt5.3-codex',
    name: 'GPT-5.3 Codex',
    provider: 'OpenAI',
    description: 'Advanced coding model. Free with your own OpenAI API key or Codex Max plan.',
    contextLength: '400K',
    priceInput: 1.25,
    priceOutput: 10.00,
    tags: ['programming', 'reasoning', 'vision'],
    released: '2026-03-10',
    popularity: 14
  },
  {
    id: 'gpt5.3-codex-spark',
    name: 'GPT-5.3 Codex Spark',
    provider: 'OpenAI',
    description: 'Fast, lightweight coding model with low reasoning mode. Free with your own OpenAI key or Max plan.',
    contextLength: '400K',
    priceInput: 0.50,
    priceOutput: 2.00,
    tags: ['programming', 'fast'],
    released: '2026-03-10',
    popularity: 13
  },
  {
    id: 'o3',
    name: 'o3',
    provider: 'OpenAI',
    description: 'Advanced reasoning model for math, science, and complex problem solving.',
    contextLength: '200K',
    priceInput: 2.00,
    priceOutput: 8.00,
    tags: ['reasoning', 'programming'],
    released: '2025-06-01',
    popularity: 8
  },
  {
    id: 'o4-mini',
    name: 'o4-mini',
    provider: 'OpenAI',
    description: 'Cost-effective reasoning model with strong performance.',
    contextLength: '200K',
    priceInput: 1.10,
    priceOutput: 4.40,
    tags: ['reasoning', 'programming', 'fast'],
    released: '2025-08-01',
    popularity: 11
  },
  {
    id: 'gpt-5-mini',
    name: 'GPT-5 Mini',
    provider: 'OpenAI',
    description: 'Affordable GPT-5 class model for high-volume applications.',
    contextLength: '400K',
    priceInput: 0.25,
    priceOutput: 2.00,
    tags: ['general', 'vision', 'fast'],
    released: '2025-12-15',
    popularity: 12
  },
  {
    id: 'codex-mini-latest',
    name: 'Codex Mini',
    provider: 'OpenAI',
    description: 'Compact coding model optimized for fast code generation.',
    contextLength: '400K',
    priceInput: 1.50,
    priceOutput: 6.00,
    tags: ['programming', 'fast'],
    released: '2026-01-15',
    popularity: 20
  },
  {
    id: 'gpt-5.4-mini',
    name: 'GPT-5.4 Mini',
    provider: 'OpenAI',
    description: 'Compact GPT-5.4 with strong coding, reasoning, and tool use at low cost.',
    contextLength: '400K',
    priceInput: 0.75,
    priceOutput: 4.50,
    tags: ['programming', 'reasoning', 'general', 'vision', 'fast'],
    released: '2026-03-17',
    popularity: 5
  },
  {
    id: 'gpt-5.4-nano',
    name: 'GPT-5.4 Nano',
    provider: 'OpenAI',
    description: 'Fastest, cheapest GPT-5.4 variant. Ideal for classification, extraction, and simple coding tasks.',
    contextLength: '400K',
    priceInput: 0.20,
    priceOutput: 1.25,
    tags: ['general', 'fast', 'vision'],
    released: '2026-03-17',
    popularity: 6
  },
  {
    id: 'gpt-4o',
    name: 'GPT-4o',
    provider: 'OpenAI',
    description: 'Previous-gen multimodal model, still widely used.',
    contextLength: '128K',
    priceInput: 2.50,
    priceOutput: 10.00,
    tags: ['general', 'vision', 'programming'],
    released: '2024-05-13',
    popularity: 17
  },
  {
    id: 'gpt-4o-mini',
    name: 'GPT-4o Mini',
    provider: 'OpenAI',
    description: 'Ultra-cheap previous-gen model for simple tasks.',
    contextLength: '128K',
    priceInput: 0.15,
    priceOutput: 0.60,
    tags: ['general', 'fast', 'vision'],
    released: '2024-07-18',
    popularity: 24
  },

  // --- Google Gemini ---
  {
    id: 'gemini-3.1-pro-preview',
    name: 'Gemini 3.1 Pro',
    provider: 'Google',
    description: 'Latest Google flagship with 1M context window and multimodal capabilities.',
    contextLength: '1M',
    priceInput: 2.00,
    priceOutput: 12.00,
    tags: ['general', 'vision', 'programming', 'reasoning'],
    released: '2026-02-20',
    popularity: 5
  },
  {
    id: 'gemini-2.5-pro',
    name: 'Gemini 2.5 Pro',
    provider: 'Google',
    description: 'Massive 2M context with strong reasoning and code abilities.',
    contextLength: '2M',
    priceInput: 1.25,
    priceOutput: 10.00,
    tags: ['general', 'vision', 'programming', 'reasoning'],
    released: '2025-03-25',
    popularity: 13
  },
  {
    id: 'gemini-2.5-flash',
    name: 'Gemini 2.5 Flash',
    provider: 'Google',
    description: 'Fast and affordable with 1M context. Great for high-throughput workloads.',
    contextLength: '1M',
    priceInput: 0.30,
    priceOutput: 2.50,
    tags: ['fast', 'general', 'vision'],
    released: '2025-05-20',
    popularity: 9
  },
  {
    id: 'gemini-flash-lite',
    name: 'Gemini Flash Lite',
    provider: 'Google',
    description: 'Cheapest Google model. Ideal for classification, routing, and simple tasks.',
    contextLength: '1M',
    priceInput: 0.02,
    priceOutput: 0.10,
    tags: ['fast', 'general', 'vision'],
    released: '2025-09-01',
    popularity: 26
  },
  {
    id: 'gemini-3.1-flash-tts-preview',
    name: 'Gemini 3.1 Flash TTS Preview',
    provider: 'Google',
    description: 'Low-latency Gemini speech generation with natural outputs, steerable prompts, expressive audio tags, and multi-speaker voice control.',
    contextLength: 'Jan 2025 cutoff',
    priceInput: 1.00,
    priceOutput: 20.00,
    tags: ['audio', 'fast'],
    released: '2026-04-16',
    popularity: 7
  },

  // --- xAI Grok ---
  {
    id: 'grok-4.3',
    name: 'Grok 4.3',
    provider: 'xAI',
    description: 'xAI recommended reasoning model with 1M context and low-cost frontier generation.',
    contextLength: '1M',
    priceInput: 1.25,
    priceOutput: 2.50,
    tags: ['reasoning', 'programming', 'general', 'vision'],
    released: '2026-05-01',
    popularity: 6
  },
  {
    id: 'grok-4.20-non-reasoning',
    name: 'Grok 4.20 Non-Reasoning',
    provider: 'xAI',
    description: 'xAI recommended non-reasoning model for fast direct responses and tool-calling workloads.',
    contextLength: '2M',
    priceInput: 1.25,
    priceOutput: 2.50,
    tags: ['fast', 'programming', 'general'],
    released: '2026-03-09',
    popularity: 18
  },
  {
    id: 'grok-3-mini',
    name: 'Grok 3 Mini',
    provider: 'xAI',
    description: 'Compact, affordable Grok for lightweight reasoning tasks.',
    contextLength: '131K',
    priceInput: 0.30,
    priceOutput: 0.50,
    tags: ['reasoning', 'fast'],
    released: '2025-03-15',
    popularity: 27
  },
  {
    id: 'grok-voice-think-fast-1.0',
    name: 'Grok Voice Think Fast',
    provider: 'xAI',
    description: 'Realtime speech-to-speech voice agent API with sub-second latency, tool use, and five built-in voices.',
    contextLength: 'N/A',
    priceInput: 3.00,
    priceOutput: 0,
    tags: ['audio', 'fast', 'agentic'],
    released: '2026-05-01',
    popularity: 6,
    pricingType: 'hour'
  },
  {
    id: 'xai-tts',
    name: 'xAI Text to Speech',
    provider: 'xAI',
    description: 'Generate expressive speech with Eve, Ara, Rex, Sal, and Leo voices plus telephony-friendly codecs.',
    contextLength: '15K chars',
    priceInput: 15.00,
    priceOutput: 0,
    tags: ['audio'],
    released: '2026-05-01',
    popularity: 6,
    pricingType: 'chars'
  },
  {
    id: 'xai-stt',
    name: 'xAI Speech to Text',
    provider: 'xAI',
    description: 'Transcribe audio in 25 languages with batch uploads, streaming mode, word timestamps, and diarization options.',
    contextLength: '500 MB',
    priceInput: 0.20,
    priceOutput: 0,
    tags: ['audio'],
    released: '2026-05-01',
    popularity: 6,
    pricingType: 'hour'
  },
  {
    id: 'lyria-3-pro-preview',
    name: 'Lyria 3 Pro Preview',
    provider: 'Google',
    description: 'Full-song music generation with deep compositional understanding, structural control, and complex transitions across musical styles.',
    contextLength: 'Jan 2025 cutoff',
    priceInput: 0.08,
    priceOutput: 0,
    tags: ['audio'],
    released: '2026-03-25',
    popularity: 6,
    pricingType: 'request'
  },
  {
    id: 'lyria-3-clip-preview',
    name: 'Lyria 3 Clip Preview',
    provider: 'Google',
    description: 'Low-latency music generation optimized for high-fidelity audio clips and precise rhythmic control.',
    contextLength: 'Jan 2025 cutoff',
    priceInput: 0.04,
    priceOutput: 0,
    tags: ['audio', 'fast'],
    released: '2026-03-25',
    popularity: 6,
    pricingType: 'request'
  },
  {
    id: 'grok-imagine-image',
    name: 'Grok Imagine Image',
    provider: 'xAI',
    description: 'xAI image generation and image-to-image editing model with aspect ratio control and multi-image merge support.',
    contextLength: 'N/A',
    priceInput: 0.02,
    priceOutput: 0,
    tags: ['art generation', 'vision'],
    released: '2026-03-23',
    popularity: 7,
    pricingType: 'request'
  },

  // --- DeepSeek ---
  {
    id: 'deepseek-v4-flash',
    name: 'DeepSeek V4 Flash',
    provider: 'DeepSeek',
    description: 'Direct DeepSeek V4 Flash with 1M context, thinking/non-thinking modes, JSON output, and tool calls at very low cost.',
    contextLength: '1M',
    priceInput: 0.14,
    priceOutput: 0.28,
    tags: ['general', 'programming', 'reasoning', 'fast', 'roleplay', 'open-source'],
    released: '2026-04-24',
    popularity: 3
  },
  {
    id: 'deepseek-v4-pro',
    name: 'DeepSeek V4 Pro',
    provider: 'DeepSeek',
    description: 'Direct DeepSeek V4 Pro with 1M context and 384K max output for difficult reasoning, sensitive policy handling, roleplay, and bio/fringe-style requests.',
    contextLength: '1M',
    priceInput: 1.74,
    priceOutput: 3.48,
    tags: ['general', 'programming', 'reasoning', 'roleplay', 'open-source'],
    released: '2026-04-24',
    popularity: 4
  },
  {
    id: 'deepseek-chat',
    name: 'DeepSeek V3.2',
    provider: 'DeepSeek',
    description: 'Top open-source model. GPT-5 class reasoning at a fraction of the cost.',
    contextLength: '128K',
    priceInput: 0.28,
    priceOutput: 0.42,
    tags: ['general', 'programming', 'reasoning', 'open-source'],
    released: '2025-12-02',
    popularity: 6
  },
  {
    id: 'deepseek-reasoner',
    name: 'DeepSeek Reasoner',
    provider: 'DeepSeek',
    description: 'Dedicated reasoning model with extended thinking capabilities.',
    contextLength: '128K',
    priceInput: 0.28,
    priceOutput: 0.42,
    tags: ['reasoning', 'programming', 'open-source'],
    released: '2025-01-20',
    popularity: 14
  },
  {
    id: 'together/deepseek-r1',
    name: 'DeepSeek R1',
    provider: 'Together AI',
    description: 'Open-source reasoning model hosted on Together AI.',
    contextLength: '128K',
    priceInput: 3.00,
    priceOutput: 7.00,
    tags: ['reasoning', 'open-source'],
    released: '2025-01-20',
    popularity: 33
  },
  {
    id: 'together/deepseek-v3.1',
    name: 'DeepSeek V3.1',
    provider: 'Together AI',
    description: 'Previous-gen DeepSeek model on Together AI infrastructure.',
    contextLength: '128K',
    priceInput: 0.60,
    priceOutput: 1.70,
    tags: ['general', 'programming', 'open-source'],
    released: '2025-08-01',
    popularity: 38
  },

  // --- Mistral ---
  {
    id: 'mistral-large-latest',
    name: 'Mistral Large 3',
    provider: 'Mistral',
    description: 'Mistral flagship with 262K context, vision, and function calling.',
    contextLength: '262K',
    priceInput: 0.50,
    priceOutput: 1.50,
    tags: ['general', 'programming', 'vision'],
    released: '2025-12-02',
    popularity: 16
  },
  {
    id: 'mistral-medium-latest',
    name: 'Mistral Medium 3',
    provider: 'Mistral',
    description: 'Balanced Mistral model for general-purpose tasks.',
    contextLength: '131K',
    priceInput: 0.40,
    priceOutput: 2.00,
    tags: ['general', 'vision'],
    released: '2025-10-01',
    popularity: 35
  },
  {
    id: 'mistral-small-latest',
    name: 'Mistral Small 3',
    provider: 'Mistral',
    description: 'Efficient small model with vision support.',
    contextLength: '131K',
    priceInput: 0.35,
    priceOutput: 0.56,
    tags: ['general', 'fast', 'vision'],
    released: '2025-09-01',
    popularity: 36
  },
  {
    id: 'codestral-latest',
    name: 'Codestral',
    provider: 'Mistral',
    description: 'Mistral code-specialized model with 256K context.',
    contextLength: '256K',
    priceInput: 0.30,
    priceOutput: 0.90,
    tags: ['programming'],
    released: '2025-07-01',
    popularity: 21
  },
  {
    id: 'pixtral-large-latest',
    name: 'Pixtral Large',
    provider: 'Mistral',
    description: 'Vision-focused model for image understanding and analysis.',
    contextLength: '131K',
    priceInput: 2.00,
    priceOutput: 6.00,
    tags: ['vision', 'general'],
    released: '2025-06-01',
    popularity: 39
  },
  {
    id: 'magistral-medium-latest',
    name: 'Magistral Medium',
    provider: 'Mistral',
    description: 'Reasoning-oriented model from Mistral.',
    contextLength: '131K',
    priceInput: 0.40,
    priceOutput: 2.00,
    tags: ['reasoning', 'vision'],
    released: '2025-11-01',
    popularity: 37
  },
  {
    id: 'devstral-medium-latest',
    name: 'Devstral Medium',
    provider: 'Mistral',
    description: 'Developer-focused model for coding and agentic workflows.',
    contextLength: '262K',
    priceInput: 0.40,
    priceOutput: 2.00,
    tags: ['programming'],
    released: '2025-11-01',
    popularity: 40
  },
  {
    id: 'open-mistral-nemo',
    name: 'Mistral Nemo',
    provider: 'Mistral',
    description: 'Ultra-cheap open-source model for basic tasks.',
    contextLength: '131K',
    priceInput: 0.02,
    priceOutput: 0.04,
    tags: ['general', 'fast', 'open-source'],
    released: '2024-07-18',
    popularity: 42
  },
  {
    id: 'ministral-8b-latest',
    name: 'Ministral 8B',
    provider: 'Mistral',
    description: 'Tiny efficient model with vision, 262K context.',
    contextLength: '262K',
    priceInput: 0.15,
    priceOutput: 0.15,
    tags: ['fast', 'vision', 'open-source'],
    released: '2025-12-03',
    popularity: 43
  },
  {
    id: 'ministral-14b-latest',
    name: 'Ministral 14B',
    provider: 'Mistral',
    description: 'Mid-size Ministral with frontier capabilities and vision.',
    contextLength: '262K',
    priceInput: 0.20,
    priceOutput: 0.20,
    tags: ['fast', 'vision', 'open-source'],
    released: '2025-12-03',
    popularity: 44
  },

  // --- Nous Research ---
  {
    id: 'hermes-4-70b',
    name: 'Hermes 4 70B',
    provider: 'Nous Research',
    description: 'Open-source deep thinking model with extended chain-of-thought reasoning at ultra-low cost.',
    contextLength: '131K',
    priceInput: 0.05,
    priceOutput: 0.20,
    tags: ['reasoning', 'general', 'open-source'],
    released: '2026-02-01',
    popularity: 34
  },
  {
    id: 'hermes-4-405b',
    name: 'Hermes 4 405B',
    provider: 'Nous Research',
    description: 'Massive 405B parameter model with deep reasoning, tool use, and system prompt adherence.',
    contextLength: '131K',
    priceInput: 0.09,
    priceOutput: 0.37,
    tags: ['reasoning', 'general', 'open-source'],
    released: '2026-02-01',
    popularity: 34
  },

  // --- Qwen / Moonshot / Together ---
  {
    id: 'qwen3.5-397b',
    name: 'Qwen 3.5 397B',
    provider: 'Together AI',
    description: 'Massive sparse MoE model from Alibaba with strong reasoning.',
    contextLength: '131K',
    priceInput: 0.60,
    priceOutput: 3.60,
    tags: ['reasoning', 'general', 'open-source'],
    released: '2026-02-01',
    popularity: 19
  },
  {
    id: 'qwen3-coder',
    name: 'Qwen 3 Coder',
    provider: 'Together AI',
    description: 'Code-specialized Qwen model optimized for software tasks.',
    contextLength: '131K',
    priceInput: 0.50,
    priceOutput: 1.20,
    tags: ['programming', 'open-source'],
    released: '2025-11-15',
    popularity: 23
  },
  {
    id: 'kimi-k2.5',
    name: 'Kimi K2.5',
    provider: 'Together AI',
    description: 'Moonshot AI model with strong coding and reasoning.',
    contextLength: '131K',
    priceInput: 0.50,
    priceOutput: 2.80,
    tags: ['programming', 'reasoning', 'open-source'],
    released: '2026-01-15',
    popularity: 22
  },

  // --- GLM / Z.AI ---
  {
    id: 'glm-5.1',
    name: 'GLM-5.1',
    provider: 'Z.AI',
    description: 'Z.AI flagship agentic model with 200K context, tool use, and long-horizon task execution.',
    contextLength: '200K',
    priceInput: 1.00,
    priceOutput: 3.20,
    tags: ['general', 'reasoning', 'agentic', 'open-source'],
    released: '2026-04-07',
    popularity: 15
  },
  {
    id: 'glm-5',
    name: 'GLM-5',
    provider: 'Together AI',
    description: 'Z.AI flagship model with 200K+ context and tool use.',
    contextLength: '202K',
    priceInput: 1.00,
    priceOutput: 3.20,
    tags: ['general', 'reasoning', 'open-source'],
    released: '2026-01-10',
    popularity: 25
  },
  {
    id: 'glm-4.7',
    name: 'GLM-4.7',
    provider: 'Together AI',
    description: 'Efficient Z.AI model with strong general performance.',
    contextLength: '202K',
    priceInput: 0.45,
    priceOutput: 2.00,
    tags: ['general', 'open-source'],
    released: '2025-10-01',
    popularity: 41
  },
  {
    id: 'glm-4.6v',
    name: 'GLM-4.6v',
    provider: 'Z.AI',
    description: 'Vision-capable GLM model for multimodal tasks.',
    contextLength: '128K',
    priceInput: 1.50,
    priceOutput: 5.00,
    tags: ['vision', 'general'],
    released: '2025-08-01',
    popularity: 45
  },

  // --- MiniMax ---
  {
    id: 'minimax-m2.7',
    name: 'MiniMax M2.7',
    provider: 'MiniMax',
    description: 'Latest MiniMax flagship model with 1M context and tool use.',
    contextLength: '1M',
    priceInput: 0.30,
    priceOutput: 1.10,
    tags: ['general', 'fast'],
    released: '2026-04-01',
    popularity: 50
  },
  {
    id: 'minimax-m2.5',
    name: 'MiniMax M2.5',
    provider: 'Together AI',
    description: '1M context model with excellent long-document performance.',
    contextLength: '1M',
    priceInput: 0.30,
    priceOutput: 1.20,
    tags: ['general', 'fast'],
    released: '2025-12-01',
    popularity: 28
  },
  {
    id: 'minimax-m2',
    name: 'MiniMax M2',
    provider: 'MiniMax',
    description: 'High-output model with 128K output tokens and tool use.',
    contextLength: '200K',
    priceInput: 0.26,
    priceOutput: 1.00,
    tags: ['general'],
    released: '2025-08-01',
    popularity: 46
  },
  {
    id: 'nvidia/minimax-m2.7',
    name: 'MiniMax M2.7 (NVIDIA)',
    provider: 'NVIDIA',
    description: 'Latest MiniMax M2.7 hosted on NVIDIA NIM. Auto-routed as a fallback for MiniMax traffic.',
    contextLength: '1M',
    priceInput: 0.30,
    priceOutput: 1.10,
    tags: ['general', 'fast'],
    released: '2026-04-01',
    popularity: 27
  },
  {
    id: 'nvidia/deepseek-v3.2',
    name: 'DeepSeek V3.2 (NVIDIA)',
    provider: 'NVIDIA',
    description: 'DeepSeek V3.2 on NVIDIA NIM. Supports reasoning_content streaming with chat_template_kwargs.thinking=true. Balanced with deepseek.com via circuit breakers.',
    contextLength: '128K',
    priceInput: 0.28,
    priceOutput: 0.42,
    tags: ['general', 'reasoning', 'open-source'],
    released: '2026-04-01',
    popularity: 26
  },
  {
    id: 'nvidia/deepseek-v4-pro',
    name: 'DeepSeek V4 Pro (NVIDIA)',
    provider: 'NVIDIA',
    description: 'Free NVIDIA-hosted DeepSeek V4 Pro with high reasoning enabled through chat_template_kwargs.',
    contextLength: '1M',
    priceInput: 0,
    priceOutput: 0,
    tags: ['general', 'programming', 'reasoning', 'open-source', 'free'],
    released: '2026-04-24',
    popularity: 24
  },
  {
    id: 'nvidia/devstral-2-123b',
    name: 'Devstral 2 123B (NVIDIA)',
    provider: 'NVIDIA',
    description: 'Mistral Devstral 2 123B instruct on NVIDIA NIM. Coding-focused; balanced with Mistral devstral-medium via circuit breakers.',
    contextLength: '256K',
    priceInput: 0.40,
    priceOutput: 2.00,
    tags: ['programming', 'open-source'],
    released: '2026-04-01',
    popularity: 25
  },

  // --- Groq (Fast Inference) ---
  {
    id: 'llama-3.3-70b-versatile',
    name: 'Llama 3.3 70B',
    provider: 'Groq',
    description: 'Meta open-source model on Groq for ultra-fast inference.',
    contextLength: '128K',
    priceInput: 0.59,
    priceOutput: 0.79,
    tags: ['general', 'fast', 'open-source'],
    released: '2024-12-06',
    popularity: 29
  },
  {
    id: 'llama-3.1-8b-instant',
    name: 'Llama 3.1 8B',
    provider: 'Groq',
    description: 'Ultra-fast small model for instant responses.',
    contextLength: '128K',
    priceInput: 0.05,
    priceOutput: 0.08,
    tags: ['fast', 'open-source'],
    released: '2024-07-23',
    popularity: 32
  },
  {
    id: 'mixtral-8x7b-32768',
    name: 'Mixtral 8x7B',
    provider: 'Groq',
    description: 'Classic MoE model on Groq for fast, cheap inference.',
    contextLength: '32K',
    priceInput: 0.24,
    priceOutput: 0.24,
    tags: ['fast', 'open-source'],
    released: '2024-01-08',
    popularity: 47
  },

  // --- Fireworks AI ---
  {
    id: 'fireworks/gpt-oss-120b',
    name: 'GPT-OSS 120B',
    provider: 'Fireworks AI',
    description: 'Open-source 120B parameter model on Fireworks with fast inference and 128K context.',
    contextLength: '128K',
    priceInput: 0.90,
    priceOutput: 0.90,
    tags: ['general', 'programming', 'open-source'],
    released: '2026-03-01',
    popularity: 34
  },
  {
    id: 'fireworks/glm-5',
    name: 'GLM-5 (Fireworks)',
    provider: 'Fireworks AI',
    description: 'Z.AI GLM-5 hosted on Fireworks for fast, reliable inference.',
    contextLength: '128K',
    priceInput: 0.90,
    priceOutput: 0.90,
    tags: ['general', 'reasoning', 'open-source'],
    released: '2026-03-01',
    popularity: 35
  },

  // --- Free Models ---
  {
    id: 'or/stepfun-flash',
    name: 'StepFun Flash',
    provider: 'OpenRouter',
    description: 'Free 256K context model from StepFun with tool use.',
    contextLength: '256K',
    priceInput: 0,
    priceOutput: 0,
    tags: ['free', 'general'],
    released: '2025-10-01',
    popularity: 48
  },
  {
    id: 'or/solar-pro-3',
    name: 'Solar Pro 3',
    provider: 'OpenRouter',
    description: 'Free model from Upstage with tool use support.',
    contextLength: '128K',
    priceInput: 0,
    priceOutput: 0,
    tags: ['free', 'general'],
    released: '2025-09-01',
    popularity: 49
  },
  {
    id: 'or/nemotron-nano',
    name: 'Nemotron Nano 30B',
    provider: 'OpenRouter',
    description: 'Free NVIDIA reasoning model with 256K context.',
    contextLength: '256K',
    priceInput: 0,
    priceOutput: 0,
    tags: ['free', 'reasoning'],
    released: '2025-11-01',
    popularity: 50
  },
  {
    id: 'or/arcee-trinity',
    name: 'Arcee Trinity',
    provider: 'OpenRouter',
    description: 'Free large preview model with tool use.',
    contextLength: '131K',
    priceInput: 0,
    priceOutput: 0,
    tags: ['free', 'general'],
    released: '2025-12-01',
    popularity: 51
  },
  {
    id: 'glm-4.6v-flash',
    name: 'GLM-4.6v Flash',
    provider: 'Z.AI',
    description: 'Free vision model from Z.AI with tool use.',
    contextLength: '128K',
    priceInput: 0,
    priceOutput: 0,
    tags: ['free', 'vision'],
    released: '2025-08-01',
    popularity: 52
  },
  {
    id: 'or/hunter-alpha',
    name: 'Hunter Alpha',
    provider: 'OpenRouter',
    description: 'Free 1T parameter frontier model built for agentic use, long-horizon planning, and complex reasoning.',
    contextLength: '1M',
    priceInput: 0,
    priceOutput: 0,
    tags: ['free', 'reasoning', 'general'],
    released: '2026-03-11',
    popularity: 46
  },

  // --- Image Generation ---
  {
    id: 'flux-pro',
    name: 'FLUX.1 Pro',
    provider: 'Together AI',
    description: 'Top-tier image generation with exceptional typography and detail.',
    contextLength: 'N/A',
    priceInput: 0.04,
    priceOutput: 0,
    tags: ['art generation'],
    released: '2024-08-01',
    popularity: 53,
    pricingType: 'request'
  },
  {
    id: 'flux-dev',
    name: 'FLUX.2 Dev',
    provider: 'Together AI',
    description: 'Development-tier FLUX model for fast image iteration.',
    contextLength: 'N/A',
    priceInput: 0.015,
    priceOutput: 0,
    tags: ['art generation', 'open-source'],
    released: '2025-11-25',
    popularity: 54,
    pricingType: 'request'
  },
  {
    id: 'flux-schnell',
    name: 'FLUX.1 Schnell',
    provider: 'Together AI',
    description: 'Ultra-fast, ultra-cheap image generation.',
    contextLength: 'N/A',
    priceInput: 0.003,
    priceOutput: 0,
    tags: ['art generation', 'open-source', 'fast'],
    released: '2024-08-01',
    popularity: 55,
    pricingType: 'request'
  },
  {
    id: 'klein',
    name: 'FLUX Klein 4B',
    provider: 'Fal',
    description: 'Compact FLUX model for fast, affordable image generation.',
    contextLength: 'N/A',
    priceInput: 0.02,
    priceOutput: 0,
    tags: ['art generation', 'fast'],
    released: '2025-10-01',
    popularity: 56,
    pricingType: 'request'
  },
  {
    id: 'stable-diffusion-3',
    name: 'Stable Diffusion 3',
    provider: 'Together AI',
    description: 'Open-weights image generation with excellent prompt adherence.',
    contextLength: 'N/A',
    priceInput: 0.002,
    priceOutput: 0,
    tags: ['art generation', 'open-source'],
    released: '2024-06-12',
    popularity: 57,
    pricingType: 'request'
  },
  {
    id: 'glm-image',
    name: 'GLM Image',
    provider: 'Z.AI',
    description: 'Z.AI image generation model with multiple aspect ratios.',
    contextLength: 'N/A',
    priceInput: 0.015,
    priceOutput: 0,
    tags: ['art generation'],
    released: '2025-05-01',
    popularity: 58,
    pricingType: 'request'
  },
  {
    id: 'zimage',
    name: 'ZImage',
    provider: 'Netwrck',
    description: 'Anime-style image generation at very low cost.',
    contextLength: 'N/A',
    priceInput: 0.007,
    priceOutput: 0,
    tags: ['art generation', 'fast'],
    released: '2025-03-01',
    popularity: 59,
    pricingType: 'request'
  },
  {
    id: 'hidream-o1-image-dev',
    name: 'HiDream O1 Image Dev',
    provider: 'Fal',
    description: 'Unified image generation, editing, and subject personalization up to about 2K output.',
    contextLength: 'N/A',
    priceInput: 0.006,
    priceOutput: 0,
    tags: ['art generation', 'vision'],
    released: '2026-05-01',
    popularity: 60,
    pricingType: 'megapixel'
  },

  // --- Video Generation ---
  {
    id: 'hailuo-2.3',
    name: 'Hailuo 2.3',
    provider: 'MiniMax',
    description: 'Leading video generation model with natural motion.',
    contextLength: 'N/A',
    priceInput: 0,
    priceOutput: 0,
    tags: ['video generation'],
    released: '2025-12-01',
    popularity: 60
  },
  {
    id: 'wan',
    name: 'Wan Video',
    provider: 'Netwrck',
    description: 'High-quality video generation from text prompts.',
    contextLength: 'N/A',
    priceInput: 0,
    priceOutput: 0,
    tags: ['video generation'],
    released: '2025-09-01',
    popularity: 61
  },
  {
    id: 'ltx-video',
    name: 'LTX Video',
    provider: 'Netwrck',
    description: 'Fast, affordable video generation.',
    contextLength: 'N/A',
    priceInput: 0,
    priceOutput: 0,
    tags: ['video generation', 'fast'],
    released: '2025-06-01',
    popularity: 62
  },
  {
    id: 'ra2v',
    name: 'RA2V',
    provider: 'Netwrck',
    description: 'Smart video generation with scene understanding.',
    contextLength: 'N/A',
    priceInput: 0,
    priceOutput: 0,
    tags: ['video generation'],
    released: '2025-10-01',
    popularity: 63
  },
  {
    id: 'seedance-2.0-fast-text-to-video',
    name: 'Seedance 2.0 Fast Text to Video',
    provider: 'Fal',
    description: 'Fast ByteDance Seedance text-to-video with 720p cinematic output and optional synchronized audio.',
    contextLength: 'N/A',
    priceInput: 0.26609,
    priceOutput: 0,
    tags: ['video generation', 'fast'],
    released: '2026-04-09',
    popularity: 64,
    pricingType: 'second'
  },
  {
    id: 'seedance-2.0-text-to-video',
    name: 'Seedance 2.0 Text to Video',
    provider: 'Fal',
    description: 'Standard Seedance 2.0 text-to-video endpoint for cinematic prompt-only video generation.',
    contextLength: 'N/A',
    priceInput: 0.33374,
    priceOutput: 0,
    tags: ['video generation'],
    released: '2026-04-09',
    popularity: 65,
    pricingType: 'second'
  },
  {
    id: 'seedance-2.0-image-to-video',
    name: 'Seedance 2.0 Image to Video',
    provider: 'Fal',
    description: 'Animate a still image into cinematic video with motion prompts, optional end frame control, and synchronized audio.',
    contextLength: 'N/A',
    priceInput: 0.33264,
    priceOutput: 0,
    tags: ['video generation', 'vision'],
    released: '2026-04-09',
    popularity: 66,
    pricingType: 'second'
  },
  {
    id: 'seedance-2.0-fast-reference-to-video',
    name: 'Seedance 2.0 Fast Reference to Video',
    provider: 'Fal',
    description: 'Fast Seedance reference-to-video from prompt plus image, video, and audio URLs.',
    contextLength: 'N/A',
    priceInput: 0.26609,
    priceOutput: 0,
    tags: ['video generation', 'vision', 'fast'],
    released: '2026-04-09',
    popularity: 67,
    pricingType: 'second'
  },
  {
    id: 'seedance-2.0-reference-to-video',
    name: 'Seedance 2.0 Reference to Video',
    provider: 'Fal',
    description: 'Seedance reference-to-video with image, video, and audio references and 1080p support.',
    contextLength: 'N/A',
    priceInput: 0.33264,
    priceOutput: 0,
    tags: ['video generation', 'vision'],
    released: '2026-04-09',
    popularity: 68,
    pricingType: 'second'
  },
];
