export const GPT_IMAGE_BASE = '/static/blog/gpt-image-2-6';

export interface GptImageModel { id: string; name: string; version: string; tier: 'sunburst' | 'flare'; blurb: string }
export interface GptImagePrompt { slug: string; title: string; prompt: string; size: string; edit: boolean; inputUrl?: string }

export const GPT_IMAGE_MODELS: GptImageModel[] = [
  { id: 'gpt-image-2.6-sunburst', name: 'GPT Image 2.6 Sunburst', version: '2.6', tier: 'sunburst', blurb: "Slower tier: the detail-first option for posters and hero shots." },
  { id: 'gpt-image-2.6-flare', name: 'GPT Image 2.6 Flare', version: '2.6', tier: 'flare', blurb: "Fast tier: best for iteration and bulk generation." },
  { id: 'gpt-image-2.5-sunburst', name: 'GPT Image 2.5 Sunburst', version: '2.5', tier: 'sunburst', blurb: "Slower tier: the detail-first option for posters and hero shots." },
  { id: 'gpt-image-2.5-flare', name: 'GPT Image 2.5 Flare', version: '2.5', tier: 'flare', blurb: "Fast tier: best for iteration and bulk generation." },
];

export const GPT_IMAGE_PROMPTS: GptImagePrompt[] = [
  { slug: 'mug', title: "Product photo with chalkboard text", prompt: "A beige ceramic coffee mug on a wooden table, soft natural window light, editorial product photo, with the handwritten text 'OPEN PATHS' on a chalkboard behind it", size: '1024x1024', edit: false },
  { slug: 'poster', title: "Typographic poster", prompt: "Minimalist poster for a jazz night titled MIDNIGHT ROUTES, bold typography, deep blue and gold, saxophone silhouette, small text at the bottom: Fri 9pm, Blue Door Club", size: '1024x1536', edit: false },
  { slug: 'infographic', title: "Labelled infographic", prompt: "Clean infographic titled \"How an API gateway routes a request\" with four numbered steps in a left-to-right flow: 1 Client request, 2 Auth and rate limit, 3 Pick best model, 4 Stream response. Flat vector style, white background, teal and orange accents, crisp legible labels.", size: '1536x1024', edit: false },
  { slug: 'perfume-edit', title: "Reference edit: label and colour", prompt: "Change the label text to say OPENPATHS and make the liquid emerald green", size: '1024x1024', edit: true, inputUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/hidream-edit/perfume.jpg' },
];

/** Measured wall-clock seconds per generation through the OpenAI images API (quality high). */
export const GPT_IMAGE_SECONDS: Record<string, number> = {
  'gpt-image-2.6-sunburst/mug': 44,
  'gpt-image-2.6-sunburst/poster': 32,
  'gpt-image-2.6-sunburst/infographic': 37,
  'gpt-image-2.6-sunburst/perfume-edit': 45,
  'gpt-image-2.6-flare/mug': 10,
  'gpt-image-2.6-flare/poster': 9,
  'gpt-image-2.6-flare/infographic': 13,
  'gpt-image-2.6-flare/perfume-edit': 15,
  'gpt-image-2.5-sunburst/mug': 39,
  'gpt-image-2.5-sunburst/poster': 25,
  'gpt-image-2.5-sunburst/infographic': 27,
  'gpt-image-2.5-sunburst/perfume-edit': 39,
  'gpt-image-2.5-flare/mug': 20,
  'gpt-image-2.5-flare/poster': 16,
  'gpt-image-2.5-flare/infographic': 21,
  'gpt-image-2.5-flare/perfume-edit': 24,
};

export const gptImageOutput = (modelId: string, slug: string) => `${GPT_IMAGE_BASE}/${modelId}-${slug}.webp`;
export const gptImageThumb = (modelId: string, slug: string) => `${GPT_IMAGE_BASE}/${modelId}-${slug}-email.jpg`;

export const GPT_IMAGE_SIZES: Array<[string, string]> = [
  ['1024x1024', '1024x1024 · $0.06'],
  ['1536x1024', '1536x1024 · $0.06'],
  ['1024x1536', '1024x1536 · $0.06'],
  ['2048x2048', '2048x2048 · $0.11'],
  ['3840x2160', '3840x2160 · $0.11'],
];

export const GPT_IMAGE_QUALITIES = ['high', 'medium', 'low', 'auto'];
