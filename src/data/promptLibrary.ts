// Types and a small bundled fallback for the OpenPaths prompt library (/prompts).
// The canonical data + semantic (gobed) search live in the Go backend
// (internal/promptlib); these types mirror its JSON, and the fallback lets the
// pages render before/without the API.

export interface PromptCategory {
  slug: string;
  name: string;
  shortName: string;
  description: string;
  icon: string;
}

export interface PromptModel {
  slug: string; // real OpenPaths model id
  name: string;
  description: string;
  modality: string;
  icon: string;
}

export interface PromptModality {
  slug: string;
  name: string;
  description: string;
  icon: string;
}

export interface LibraryPrompt {
  slug: string;
  title: string;
  summary: string;
  prompt: string;
  modelSlug: string;
  modelName: string;
  modelIcon?: string;
  categorySlug: string;
  categoryName: string;
  categoryIcon?: string;
  modality: string;
  modalityName: string;
  tags: string[];
  isFree: boolean;
  featured: boolean;
  popularity: number;
  url: string;
  modelUrl: string;
  categoryUrl: string;
  modalityUrl: string;
  score?: number;
}

export interface PromptMeta {
  total: number;
  categories: PromptCategory[];
  models: PromptModel[];
  types: PromptModality[];
  counts: {
    byCategory: Record<string, number>;
    byModel: Record<string, number>;
    byType: Record<string, number>;
    free: number;
  };
}

// A minimal bundled seed so /prompts is never empty even if the API is down.
export const fallbackPrompts: LibraryPrompt[] = [
  {
    slug: 'autocomplete-pytorch-coding',
    title: 'Autocomplete PyTorch Coding',
    summary: 'Turn a sketch of a PyTorch module into complete, idiomatic, runnable code.',
    prompt:
      'You are an expert PyTorch engineer. Complete the following code.\n\nRules:\n- Finish partially-written functions/classes; do not rewrite working parts.\n- Use idiomatic, modern PyTorch (nn.Module, typed tensors, device-agnostic).\n- Include shape annotations for every tensor op.\n\nCode so far:\n```python\nimport torch\nimport torch.nn as nn\n\nclass TinyResNetBlock(nn.Module):\n    def __init__(self, channels: int):\n        super().__init__()\n        # TODO: two 3x3 convs with BN + ReLU and a residual connection\n```\n\nReturn only the completed code block.',
    modelSlug: 'openpaths/auto-code',
    modelName: 'OpenPaths Auto (Code)',
    categorySlug: 'coding-dev',
    categoryName: 'Coding & Dev prompts',
    modality: 'text',
    modalityName: 'Text prompts',
    tags: ['pytorch', 'autocomplete', 'deep learning', 'python', 'code generation'],
    isFree: true,
    featured: true,
    popularity: 99,
    url: '/prompts/autocomplete-pytorch-coding',
    modelUrl: '/prompts/model/openpaths/auto-code',
    categoryUrl: '/prompts/category/coding-dev',
    modalityUrl: '/prompts/type/text',
  },
  {
    slug: 'summarize-long-doc',
    title: 'Summarize a Long Document',
    summary: 'Compress a long document into a structured, skimmable brief.',
    prompt:
      'Summarize the document below.\n\nOutput:\n- TL;DR (2-3 sentences).\n- Key points (bulleted, grouped by theme).\n- Decisions / action items with owners if present.\n- Open questions or risks.\n\nKeep it faithful; do not invent details.\n\nDocument:\n```\n[paste text]\n```',
    modelSlug: 'gemini-2.5-pro',
    modelName: 'Gemini 2.5 Pro',
    categorySlug: 'productivity-writing',
    categoryName: 'Productivity & Writing prompts',
    modality: 'text',
    modalityName: 'Text prompts',
    tags: ['summary', 'tldr', 'notes', 'productivity'],
    isFree: true,
    featured: true,
    popularity: 85,
    url: '/prompts/summarize-long-doc',
    modelUrl: '/prompts/model/gemini-2.5-pro',
    categoryUrl: '/prompts/category/productivity-writing',
    modalityUrl: '/prompts/type/text',
  },
  {
    slug: 'cinematic-key-art',
    title: 'Cinematic Key Art',
    summary: 'Dramatic poster-style key art for a story, game, or campaign.',
    prompt:
      'Create cinematic key art for [title/theme].\n\nDirection:\n- Strong focal subject, dramatic lighting, depth via atmosphere.\n- Color grade: [warm/cool/teal-orange]; mood: [epic/tense/hopeful].\n- Negative space at top for a title treatment.\n- Aspect 2:3 poster framing, photoreal-illustrative hybrid.',
    modelSlug: 'flux-pro',
    modelName: 'FLUX Pro',
    categorySlug: 'art-illustration',
    categoryName: 'Art & Illustration prompts',
    modality: 'image',
    modalityName: 'Image prompts',
    tags: ['key art', 'cinematic', 'poster', 'concept art'],
    isFree: true,
    featured: true,
    popularity: 89,
    url: '/prompts/cinematic-key-art',
    modelUrl: '/prompts/model/flux-pro',
    categoryUrl: '/prompts/category/art-illustration',
    modalityUrl: '/prompts/type/image',
  },
  {
    slug: 'lofi-study-beat',
    title: 'Lo-fi Study Beat',
    summary: 'A mellow lo-fi instrumental for focus and study sessions.',
    prompt:
      'A mellow lo-fi hip-hop instrumental for studying.\n\n- ~80 BPM, warm Rhodes chords, soft vinyl crackle, gentle swing drums.\n- Relaxed but motivating, no vocals.\n- Smooth, loopable structure, light tape saturation.',
    modelSlug: 'lyria-3-pro-preview',
    modelName: 'Lyria 3 Pro',
    categorySlug: 'music-audio',
    categoryName: 'Music & Audio prompts',
    modality: 'music',
    modalityName: 'Music prompts',
    tags: ['lofi', 'instrumental', 'study', 'chill'],
    isFree: true,
    featured: true,
    popularity: 81,
    url: '/prompts/lofi-study-beat',
    modelUrl: '/prompts/model/lyria-3-pro-preview',
    categoryUrl: '/prompts/category/music-audio',
    modalityUrl: '/prompts/type/music',
  },
];
