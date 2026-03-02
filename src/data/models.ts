export type Tag = 'programming' | 'roleplay' | 'art generation' | 'general' | 'vision' | 'fast' | 'reasoning' | 'open-source';

export interface Model {
  id: string;
  name: string;
  provider: string;
  description: string;
  contextLength: string;
  priceInput: number; // per 1M tokens
  priceOutput: number; // per 1M tokens
  tags: Tag[];
}

export const models: Model[] = [
  {
    id: 'anthropic/claude-3.5-sonnet',
    name: 'Claude 3.5 Sonnet',
    provider: 'Anthropic',
    description: 'The most intelligent model for complex tasks, coding, and reasoning.',
    contextLength: '200K',
    priceInput: 3.00,
    priceOutput: 15.00,
    tags: ['programming', 'reasoning', 'general', 'vision']
  },
  {
    id: 'anthropic/claude-3-opus',
    name: 'Claude 3 Opus',
    provider: 'Anthropic',
    description: 'Powerful model for highly complex tasks.',
    contextLength: '200K',
    priceInput: 15.00,
    priceOutput: 75.00,
    tags: ['reasoning', 'general']
  },
  {
    id: 'openai/gpt-4o',
    name: 'GPT-4o',
    provider: 'OpenAI',
    description: 'Fast, versatile flagship model with vision capabilities.',
    contextLength: '128K',
    priceInput: 5.00,
    priceOutput: 15.00,
    tags: ['general', 'vision', 'programming']
  },
  {
    id: 'openai/o1-preview',
    name: 'o1-preview',
    provider: 'OpenAI',
    description: 'Advanced reasoning model for complex problem solving.',
    contextLength: '128K',
    priceInput: 15.00,
    priceOutput: 60.00,
    tags: ['reasoning', 'programming']
  },
  {
    id: 'meta-llama/llama-3.1-405b-instruct',
    name: 'Llama 3.1 405B',
    provider: 'Meta',
    description: 'State-of-the-art open-source model with massive parameter count.',
    contextLength: '128K',
    priceInput: 2.70,
    priceOutput: 2.70,
    tags: ['open-source', 'general', 'reasoning']
  },
  {
    id: 'meta-llama/llama-3.1-70b-instruct',
    name: 'Llama 3.1 70B',
    provider: 'Meta',
    description: 'Highly capable and efficient open-source model.',
    contextLength: '128K',
    priceInput: 0.52,
    priceOutput: 0.52,
    tags: ['open-source', 'general', 'fast']
  },
  {
    id: 'mistral/mistral-large-2407',
    name: 'Mistral Large 2',
    provider: 'Mistral',
    description: 'Top-tier reasoning model from Mistral AI.',
    contextLength: '128K',
    priceInput: 2.00,
    priceOutput: 6.00,
    tags: ['reasoning', 'general', 'programming']
  },
  {
    id: 'google/gemini-1.5-pro',
    name: 'Gemini 1.5 Pro',
    provider: 'Google',
    description: 'Massive context window model for analyzing large documents and codebases.',
    contextLength: '2M',
    priceInput: 3.50,
    priceOutput: 10.50,
    tags: ['general', 'vision', 'programming']
  },
  {
    id: 'google/gemini-1.5-flash',
    name: 'Gemini 1.5 Flash',
    provider: 'Google',
    description: 'Fast and cost-effective model for high-volume tasks.',
    contextLength: '1M',
    priceInput: 0.075,
    priceOutput: 0.30,
    tags: ['fast', 'general', 'vision']
  },
  {
    id: 'deepseek/deepseek-coder-v2',
    name: 'DeepSeek Coder V2',
    provider: 'DeepSeek',
    description: 'Open-source model specialized in code generation and understanding.',
    contextLength: '128K',
    priceInput: 0.14,
    priceOutput: 0.28,
    tags: ['programming', 'open-source']
  },
  {
    id: 'nousresearch/hermes-3-llama-3.1-405b',
    name: 'Hermes 3 (405B)',
    provider: 'NousResearch',
    description: 'Uncensored, highly steerable model excellent for roleplay and creative writing.',
    contextLength: '128K',
    priceInput: 3.00,
    priceOutput: 3.00,
    tags: ['roleplay', 'open-source']
  },
  {
    id: 'gryphe/mythomax-l2-13b',
    name: 'MythoMax L2 13B',
    provider: 'Gryphe',
    description: 'A highly popular model specifically tuned for roleplay and storytelling.',
    contextLength: '4K',
    priceInput: 0.10,
    priceOutput: 0.10,
    tags: ['roleplay', 'open-source', 'fast']
  },
  {
    id: 'ra1/art-generator',
    name: 'RA1 Art Generator',
    provider: 'RA1',
    description: 'State-of-the-art image generation model with incredible photorealism.',
    contextLength: 'N/A',
    priceInput: 40.00, // per 1000 images
    priceOutput: 0,
    tags: ['art generation']
  },
  {
    id: 'stabilityai/stable-diffusion-3',
    name: 'Stable Diffusion 3',
    provider: 'Stability AI',
    description: 'Open-weights image generation model with excellent prompt adherence.',
    contextLength: 'N/A',
    priceInput: 15.00, // per 1000 images
    priceOutput: 0,
    tags: ['art generation', 'open-source']
  },
  {
    id: 'black-forest-labs/flux-1-pro',
    name: 'FLUX.1 Pro',
    provider: 'Black Forest Labs',
    description: 'Cutting-edge image generation model with exceptional typography and detail.',
    contextLength: 'N/A',
    priceInput: 30.00, // per 1000 images
    priceOutput: 0,
    tags: ['art generation']
  }
];
