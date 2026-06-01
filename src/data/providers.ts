export interface Provider {
  slug: string;
  name: string;
  url: string;
  description: string;
  featured: boolean;
  logo?: string;
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
    description: 'GPT-5.5, GPT-5 Chat Latest, GPT-5 Codex, o3/o4-mini reasoning models, and Codex Mini for code generation.',
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
    description: 'M2.5 chat models with 1M context and Hailuo video generation.',
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
    description: 'Gateway to 600+ models including free tiers from StepFun, Upstage, NVIDIA, Arcee, and Liquid.',
    featured: false,
    logo: '/logos/openrouter.svg'
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
    logo: '/logos/exa.svg'
  },
  {
    slug: 'papers',
    name: 'Papers',
    url: 'https://papers.app.nz',
    description: 'Applied AI NZ research search for agents. Search papers, methods, datasets, and GitHub code with markdown output and app.nz API key billing.',
    featured: false,
    logo: '/logos/papers.webp'
  },
];

export const providersByName: Record<string, Provider> = Object.fromEntries(
  providers.map(p => [p.name, p])
);

export function getProviderLogo(providerName: string): string {
  return providersByName[providerName]?.logo || FALLBACK_LOGO;
}
