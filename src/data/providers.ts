export type ProviderKind = 'model' | 'search';

export interface Provider {
  slug: string;
  name: string;
  url: string;
  description: string;
  featured: boolean;
  logo?: string;
  logoSmall?: string;
  logoSrcSet?: string;
  // 'search' providers expose search/tool APIs rather than LLM models —
  // they have no entries in models.ts and must not render a model count.
  kind?: ProviderKind;
}

export const FALLBACK_LOGO = '/logos/openpaths.svg';

export const providers: Provider[] = [
  {
    slug: 'netwrck',
    name: 'Netwrck',
    url: 'https://netwrck.com',
    description: 'Creative media platform with RA1 image generation, ZImage anime art, RA2V/LTX/Wan video, and adjacent image editing tools.',
    featured: true,
    logo: '/logos/netwrck.webp'
  },
  {
    slug: 'cutedsl',
    name: 'CuteDSL',
    url: 'https://cutedsl.cc',
    description: 'Triton-accelerated model API: Z-Image Turbo image generation and chronos2 time-series forecasting (27x faster via custom kernels), plus Kokoro TTS, speech-to-text, and chat.',
    featured: true,
    logo: 'https://appstatic.app.nz/cutedsl/static/images/logo.webp'
  },
  {
    slug: 'text-generator',
    name: 'Text-Generator.io',
    url: 'https://text-generator.io',
    description: 'Text, vision, and speech API with privacy-first workflows; OpenPaths currently exposes its ModernBERT embedding lane for RAG and semantic search.',
    featured: true,
    logo: '/logos/textgenerator-brain.webp'
  },
  {
    slug: 'openpaths',
    name: 'OpenPaths',
    url: '/',
    description: 'Auto-routing tiers that intelligently select the best model for your task. From cheap flash models to frontier reasoning.',
    featured: true,
    logo: '/logos/openpaths.svg'
  },
  {
    slug: 'anthropic',
    name: 'Anthropic',
    url: 'https://anthropic.com',
    description: 'Makers of Claude. Opus 4.6, Sonnet 4.6, and Haiku 4.5 for coding, reasoning, and multimodal tasks.',
    featured: false,
    logo: '/logos/anthropic.svg'
  },
  {
    slug: 'openai',
    name: 'OpenAI',
    url: 'https://openai.com',
    description: 'GPT-5, GPT Realtime live voice, GPT Image, Sora, transcription, and o3/o4 reasoning models.',
    featured: false,
    logo: '/logos/openai.svg'
  },
  {
    slug: 'cursor',
    name: 'Cursor',
    url: 'https://cursor.com',
    description: 'Composer 2.5 (standard and fast tiers) via the Cursor Cloud Agents API. Agentic coding with tool use, routed through OpenPaths chat completions.',
    featured: false,
    logo: '/logos/cursor.svg'
  },
  {
    slug: 'google',
    name: 'Google',
    url: 'https://deepmind.google',
    description: 'Gemini 3.5 Flash, 2.5 Pro/Flash with up to 2M context windows and multimodal capabilities.',
    featured: false,
    logo: '/logos/google.svg'
  },
  {
    slug: 'xai',
    name: 'xAI',
    url: 'https://x.ai',
    description: 'Grok text models plus Voice Agent, Text to Speech, and Speech to Text APIs for realtime and batch audio.',
    featured: false,
    logo: '/logos/xai.svg'
  },
  {
    slug: 'deepseek',
    name: 'DeepSeek',
    url: 'https://deepseek.com',
    description: 'Open-source V3.2 and Reasoner models delivering frontier performance at extremely low cost.',
    featured: false,
    logo: '/logos/deepseek.svg'
  },
  {
    slug: 'moonshot',
    name: 'Moonshot AI',
    url: 'https://platform.moonshot.ai',
    description: 'Kimi K3 flagship (2.8T params, 1M context) plus long-context coding models served directly through Moonshot’s OpenAI-compatible API.',
    featured: false,
    logo: 'https://icons.duckduckgo.com/ip3/moonshot.ai.ico'
  },
  {
    slug: 'thinking-machines',
    name: 'Thinking Machines',
    url: 'https://thinkingmachines.ai',
    description: 'Inkling and Inkling-Small, open-weights multimodal mixture-of-experts models for coding, reasoning, tool use, and customizable thinking effort, with native text, image, and audio input.',
    featured: false,
    logo: 'https://icons.duckduckgo.com/ip3/thinkingmachines.ai.ico'
  },
  {
    slug: 'qwen',
    name: 'Qwen',
    url: 'https://qwen.ai',
    description: 'Alibaba Qwen multimodal and agent models served through DashScope OpenAI-compatible mode.',
    featured: false,
    logo: 'https://icons.duckduckgo.com/ip3/qwen.ai.ico'
  },
  {
    slug: 'mistral',
    name: 'Mistral',
    url: 'https://mistral.ai',
    description: 'Mistral Large 3, Codestral, Pixtral, Magistral, Devstral, and Ministral models. Strong European AI.',
    featured: false,
    logo: '/logos/mistral.svg'
  },
  {
    slug: 'together',
    name: 'Together AI',
    url: 'https://together.ai',
    description: 'Inference platform hosting Qwen, Kimi, GLM, MiniMax, DeepSeek, and FLUX image models.',
    featured: false,
    logo: '/logos/together.svg'
  },
  {
    slug: 'groq',
    name: 'Groq',
    url: 'https://groq.com',
    description: 'Ultra-fast LPU inference for Llama 3.3, Llama 3.1, Mixtral, and Whisper speech-to-text at 228x real-time.',
    featured: false,
    logo: '/logos/groq.svg'
  },
  {
    slug: 'minimax',
    name: 'MiniMax',
    url: 'https://minimax.io',
    description: 'MiniMax M-series long-context chat plus Hailuo video generation, including multimodal M3 routes and image-to-video workflows.',
    featured: false,
    logo: '/logos/minimax.svg'
  },
  {
    slug: 'zai',
    name: 'Z.AI',
    url: 'https://z.ai',
    description: 'GLM-5, GLM-4.7, GLM-4.6v vision, and GLM Image generation models.',
    featured: false,
    logo: '/logos/zai.svg'
  },
  {
    slug: 'sakana',
    name: 'Sakana AI',
    url: 'https://sakana.ai',
    description: 'Fugu and Fugu Ultra: orchestration models that route a single request across a pool of providers, billed on real per-request token usage.',
    featured: false,
    logo: '/logos/sakana.png'
  },
  {
    slug: 'nous',
    name: 'Nous Research',
    url: 'https://nousresearch.com',
    description: 'Open-source AI research lab. Hermes 4 70B and 405B models with deep thinking and tool use at ultra-low cost.',
    featured: false,
    logo: '/logos/nous.webp'
  },
  {
    slug: 'openrouter',
    name: 'OpenRouter',
    url: 'https://openrouter.ai',
    description: 'Gateway to 600+ models, including Kimi K2.7 Code, Qwen 3.7 Plus, Nemotron 3 Ultra, MiniMax M3, StepFun multimodal routes, and free tiers.',
    featured: false,
    logo: '/logos/openrouter.svg'
  },
  {
    slug: 'stepfun',
    name: 'StepFun',
    url: 'https://platform.stepfun.ai',
    description: 'StepFun multimodal chat models, including Step 3.7 Flash for text, image, video, and tool workflows.',
    featured: false,
    logo: 'https://icons.duckduckgo.com/ip3/stepfun.ai.ico'
  },
  {
    slug: 'inference_net',
    name: 'Inference.net',
    url: 'https://inference.net',
    description: 'OpenAI-compatible endpoint serving Nemotron 3 Super, Schematron, ClipTagger, GPT-OSS, Llama, DeepSeek, Qwen, Gemma, and Mistral routes.',
    featured: false,
    logo: '/logos/inference-net.webp'
  },
  {
    slug: 'fireworks',
    name: 'Fireworks AI',
    url: 'https://fireworks.ai',
    description: 'Fast inference platform hosting GPT-OSS 120B, GLM-5, and Whisper speech-to-text models.',
    featured: false,
    logo: '/logos/fireworks.svg'
  },
  {
    slug: 'nvidia',
    name: 'NVIDIA',
    url: 'https://build.nvidia.com',
    description: 'NVIDIA NIM inference hosting free DeepSeek V4 Pro, MiniMax M2.7, and other frontier open models.',
    featured: false,
    logo: '/logos/nvidia.svg'
  },
  {
    slug: 'fal',
    name: 'Fal',
    url: 'https://fal.ai',
    description: 'Fast serverless inference. FLUX image generation, Smart Resize image recomposition, and Whisper speech-to-text with chunk timestamps.',
    featured: false,
    logo: '/logos/fal.svg'
  },
  {
    slug: 'black-forest-labs',
    name: 'Black Forest Labs',
    url: 'https://bfl.ai',
    description: 'The frontier visual intelligence lab behind FLUX 3 Video, FLUX.2 image generation and editing, specialized FLUX Tools, and the open-weight FLUX.1 family.',
    featured: false,
    logo: '/logos/bfl-256.webp',
    logoSmall: '/logos/bfl-64.webp',
    logoSrcSet: '/logos/bfl-32.webp 32w, /logos/bfl-64.webp 64w, /logos/bfl-128.webp 128w, /logos/bfl-256.webp 256w, /logos/bfl-512.webp 512w'
  },
  {
    slug: 'alibaba',
    name: 'Alibaba',
    url: 'https://www.alibabacloud.com',
    description: 'Alibaba video generation models exposed through OpenPaths, including Happy Horse image-to-video on Fal infrastructure.',
    featured: false,
    logo: '/logos/alibaba.webp'
  },
  {
    slug: 'exa',
    name: 'Exa',
    url: 'https://exa.ai',
    description: 'Search API for AI applications with fast web search, highlights, full-page text, structured outputs, livecrawl freshness controls, domain filters, and date filtering.',
    featured: false,
    logo: '/logos/exa.svg',
    kind: 'search'
  },
  {
    slug: 'papers',
    name: 'Papers',
    url: 'https://papers.app.nz',
    description: 'Applied AI NZ research search for agents. Search papers, methods, datasets, and GitHub code with markdown output and app.nz API key billing.',
    featured: false,
    logo: '/logos/papers.webp',
    kind: 'search'
  },
];

export const providersByName: Record<string, Provider> = Object.fromEntries(
  providers.map(p => [p.name, p])
);

export function getProviderLogo(providerName: string): string {
  return providersByName[providerName]?.logoSmall || providersByName[providerName]?.logo || FALLBACK_LOGO;
}
