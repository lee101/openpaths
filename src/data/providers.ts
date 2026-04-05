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
    description: 'First-party image and video generation. Home of RA1 art generator, ZImage anime art, and Wan/LTX/RA2V video models.',
    featured: true,
    logo: '/logos/netwrck.svg'
  },
  {
    slug: 'text-generator',
    name: 'Text-Generator.io',
    url: 'https://text-generator.io',
    description: 'First-party embedding provider. ModernBERT-powered text embeddings for search, RAG, and semantic similarity.',
    featured: true,
    logo: '/logos/textgenerator.svg'
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
    description: 'GPT-5.4, GPT-5 Chat Latest, GPT-5 Codex, o3/o4-mini reasoning models, and Codex Mini for code generation.',
    featured: false,
    logo: '/logos/openai.svg'
  },
  {
    slug: 'google',
    name: 'Google',
    url: 'https://deepmind.google',
    description: 'Gemini 3.1 Pro, 2.5 Pro/Flash with up to 2M context windows and multimodal capabilities.',
    featured: false,
    logo: '/logos/google.svg'
  },
  {
    slug: 'xai',
    name: 'xAI',
    url: 'https://x.ai',
    description: 'Grok 4 flagship, Grok 4.1 Fast with 2M context, and Grok 3 Mini for affordable reasoning.',
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
    description: 'Ultra-fast LPU inference for Llama 3.3, Llama 3.1, and Mixtral models.',
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
    logo: '/logos/nous.svg'
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
    description: 'Fast inference platform hosting GPT-OSS 120B, GLM-5, and other open-source models.',
    featured: false,
    logo: '/logos/fireworks.svg'
  },
  {
    slug: 'fal',
    name: 'Fal',
    url: 'https://fal.ai',
    description: 'Fast serverless inference. Home of FLUX Klein 4B compact image generation.',
    featured: false,
    logo: '/logos/fal.svg'
  },
];

export const providersByName: Record<string, Provider> = Object.fromEntries(
  providers.map(p => [p.name, p])
);

export function getProviderLogo(providerName: string): string {
  return providersByName[providerName]?.logo || FALLBACK_LOGO;
}
