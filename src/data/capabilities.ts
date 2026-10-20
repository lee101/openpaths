import { models, type Model } from './models';
import { slugify } from '../lib/modelPricing';

export type TryKind = 'chat' | 'messages' | 'embeddings' | 'tts' | 'stt' | 'image' | 'studio';

export interface StaticRow {
  id: string;
  name: string;
  provider: string;
  price: string;
}

export interface Capability {
  slug: string;
  path: string;
  name: string;
  h1: string;
  tagline: string;
  seoTitle: string;
  seoDescription: string;
  intro: string;
  kind: TryKind;
  endpoint: string;
  defaultModel: string;
  /** Existing full studio for async/long-running media (video, music, 3D, search). */
  studioPath?: string;
  studioLabel?: string;
  select: (m: Model) => boolean;
  sort?: 'popularity' | 'price';
  staticRows?: StaticRow[];
  /** Sample JSON body for studio-kind snippets. */
  sampleBody?: Record<string, unknown>;
  faq: Array<{ q: string; a: string }>;
  related: string[];
  vsLimit?: number;
}

export const API_BASE = 'https://openpaths.io/v1';
export const NON_CHAT_TAGS = ['art generation', 'video generation', 'audio', 'embedding'];

export function isChatModel(m: Model) {
  return !m.tags.some(tag => NON_CHAT_TAGS.includes(tag)) && m.contextLength !== 'N/A' && !m.tags.includes('forecasting') && !m.id.startsWith('openpaths/auto');
}

/**
 * Chat models that bill $0 per token. Every entry is an external-provider route
 * (OpenRouter, Z.AI, NVIDIA): none touches app.nz / RunPod GPU capacity. The Go
 * guard test internal/config/free_models_guard_test.go enforces that against config.yaml.
 */
export const FREE_MODEL_IDS = ['or/nemotron-nano', 'glm-4.6v-flash', 'nvidia/deepseek-v4-flash', 'or/space-bunny-alpha'];

const FREE_SET = new Set(FREE_MODEL_IDS);
const isFreeChat = (m: Model) => FREE_SET.has(m.id);

const FAQ_KEY = {
  openai: { q: 'Is it compatible with the OpenAI SDK?', a: 'Yes. Set base_url to https://openpaths.io/v1 and use your OpenPaths API key; request and response shapes follow the OpenAI API.' },
  signup: { q: 'How do I get an API key?', a: 'Create an account and your first key is issued immediately. The same key works for every model and endpoint on OpenPaths.' },
  pricing: { q: 'How is it priced?', a: 'Prepaid, pay-as-you-go credits at close to provider list price with no subscription. The tables on this page are generated from the live OpenPaths model catalog.' },
};

export const CAPABILITIES: Capability[] = [
  {
    slug: 'chat-completions',
    path: '/api/chat-completions',
    name: 'Chat Completions',
    h1: 'OpenAI-Compatible Chat Completions API',
    tagline: 'One endpoint, every frontier and open model',
    seoTitle: 'OpenAI-Compatible Chat Completions API - 130+ Models | OpenPaths',
    seoDescription: 'Drop-in OpenAI-compatible /v1/chat/completions for GPT, Claude, Gemini, DeepSeek, Qwen and more. Live pricing table, try-it box, curl/Python/JS snippets.',
    intro: 'Point any OpenAI SDK at OpenPaths and switch between hundreds of chat models by changing one string. Streaming, tool calls, vision and JSON mode pass through; routing falls back automatically when a provider is down.',
    kind: 'chat',
    endpoint: '/chat/completions',
    defaultModel: 'or/nemotron-nano',
    select: isChatModel,
    sort: 'popularity',
    faq: [
      FAQ_KEY.openai,
      { q: 'Which models can I call?', a: 'Every chat model in the table below, plus auto-routing aliases such as openpaths/auto that pick a model per request.' },
      FAQ_KEY.signup,
      FAQ_KEY.pricing,
    ],
    related: ['free-llm', 'coding-llm', 'reasoning-llm', 'vision-llm', 'anthropic-messages'],
    vsLimit: 6,
  },
  {
    slug: 'free-llm',
    path: '/api/free-llm',
    name: 'Free LLM API',
    h1: 'Free LLM API (OpenAI-Compatible)',
    tagline: 'Models that cost $0 per token',
    seoTitle: 'Free LLM API - OpenAI-Compatible, No Per-Token Cost | OpenPaths',
    seoDescription: 'Call free hosted LLMs through an OpenAI-compatible API. Nemotron, GLM Flash and DeepSeek V4 Flash bill $0 per token; try them in the browser and copy the code.',
    intro: 'These models bill $0 per token on OpenPaths, so a new account with an empty balance can call them right away. Paid models on the same key unlock when you add credits. Free routes are best-effort and rate limited per key.',
    kind: 'chat',
    endpoint: '/chat/completions',
    defaultModel: 'or/nemotron-nano',
    select: isFreeChat,
    sort: 'popularity',
    faq: [
      { q: 'Are these models really free?', a: 'They bill $0 per input and output token. Free routes are served by upstream providers and may be rate limited or change; the list below reflects the current OpenPaths catalog.' },
      { q: 'Do I need a credit card?', a: 'No. Create an account, copy your key, and call a free model. A balance is only needed for paid models.' },
      FAQ_KEY.openai,
    ],
    related: ['chat-completions', 'coding-llm', 'embeddings'],
  },
  {
    slug: 'coding-llm',
    path: '/api/coding-llm',
    name: 'Coding LLM API',
    h1: 'Coding LLM API for Agents and IDEs',
    tagline: 'Programming models behind one OpenAI-compatible key',
    seoTitle: 'Coding LLM API - OpenAI-Compatible Code Models & Pricing | OpenPaths',
    seoDescription: 'Compare code-focused LLMs by price and context window and call them through one OpenAI-compatible endpoint. Works with Cursor, Cline, Aider and your own agents.',
    intro: 'Code-tuned and agentic models from every major lab, compared on price and context. Use them from Cursor, Cline, Aider, Claude Code-style agents or your own harness with a single base URL.',
    kind: 'chat',
    endpoint: '/chat/completions',
    defaultModel: 'or/nemotron-nano',
    select: m => isChatModel(m) && m.tags.includes('programming'),
    sort: 'popularity',
    faq: [FAQ_KEY.openai, { q: 'Does it support tool calls?', a: 'Yes, tool calling passes through for models that support it. Agentic models are tagged in the catalog.' }, FAQ_KEY.pricing],
    related: ['chat-completions', 'reasoning-llm', 'free-llm'],
    vsLimit: 5,
  },
  {
    slug: 'reasoning-llm',
    path: '/api/reasoning-llm',
    name: 'Reasoning LLM API',
    h1: 'Reasoning LLM API',
    tagline: 'Thinking models, compared on price',
    seoTitle: 'Reasoning LLM API - OpenAI-Compatible Thinking Models | OpenPaths',
    seoDescription: 'Call reasoning and thinking models (o-series, Claude, Gemini, DeepSeek, Qwen) through one OpenAI-compatible API with a live price comparison.',
    intro: 'Extended-thinking models for math, planning and hard coding problems, with reasoning effort passed through where the provider supports it.',
    kind: 'chat',
    endpoint: '/chat/completions',
    defaultModel: 'or/nemotron-nano',
    select: m => isChatModel(m) && m.tags.includes('reasoning'),
    sort: 'popularity',
    faq: [FAQ_KEY.openai, { q: 'How do I set reasoning effort?', a: 'Send reasoning_effort in the request body; it is forwarded to providers that support it.' }, FAQ_KEY.pricing],
    related: ['chat-completions', 'coding-llm', 'free-llm'],
    vsLimit: 5,
  },
  {
    slug: 'vision-llm',
    path: '/api/vision-llm',
    name: 'Vision LLM API',
    h1: 'Vision LLM API (Image Understanding)',
    tagline: 'Send images, get answers',
    seoTitle: 'Vision LLM API - OpenAI-Compatible Image Understanding | OpenPaths',
    seoDescription: 'Multimodal chat models that accept images, through an OpenAI-compatible API. Compare vision model prices and context windows and try one live.',
    intro: 'Pass image_url content parts to any vision model using the standard OpenAI message format: captioning, OCR, chart reading and UI analysis.',
    kind: 'chat',
    endpoint: '/chat/completions',
    defaultModel: 'glm-4.6v-flash',
    select: m => isChatModel(m) && m.tags.includes('vision'),
    sort: 'popularity',
    faq: [FAQ_KEY.openai, { q: 'How do I send an image?', a: 'Use a content array with {"type":"image_url","image_url":{"url":"https://..."}} alongside a text part, exactly as with OpenAI.' }, FAQ_KEY.pricing],
    related: ['chat-completions', 'free-llm', 'image-generation'],
    vsLimit: 5,
  },
  {
    slug: 'anthropic-messages',
    path: '/api/anthropic-messages',
    name: 'Anthropic Messages API',
    h1: 'Anthropic-Compatible Messages API',
    tagline: 'Claude SDKs and agent SDKs, any model',
    seoTitle: 'Anthropic-Compatible Messages API (/v1/messages) | OpenPaths',
    seoDescription: 'Use the Anthropic SDK or Claude-style agents against OpenPaths /v1/messages and route to Claude or any other model. Pricing table, live try-it and snippets.',
    intro: 'OpenPaths speaks the Anthropic Messages protocol, so the official Anthropic SDKs and agent frameworks work by changing the base URL. Claude models are billed at provider rates.',
    kind: 'messages',
    endpoint: '/messages',
    defaultModel: 'claude-haiku-4-5-20251001',
    select: m => isChatModel(m) && m.provider === 'Anthropic',
    sort: 'popularity',
    faq: [{ q: 'Which auth header do I use?', a: 'Either x-api-key (Anthropic style) or Authorization: Bearer; both accept your OpenPaths key.' }, FAQ_KEY.signup, FAQ_KEY.pricing],
    related: ['chat-completions', 'coding-llm', 'reasoning-llm'],
  },
  {
    slug: 'embeddings',
    path: '/api/embeddings',
    name: 'Embeddings',
    h1: 'OpenAI-Compatible Embeddings API',
    tagline: 'Vectors for search and RAG',
    seoTitle: 'OpenAI-Compatible Embeddings API - Cheap Text Embeddings | OpenPaths',
    seoDescription: 'Generate text embeddings through /v1/embeddings with the OpenAI SDK. Compare OpenPaths, OpenAI and Gemini embedding prices and try it live.',
    intro: 'A drop-in /v1/embeddings endpoint for semantic search, RAG and clustering, including a low-cost CPU-served OpenPaths embedding model.',
    kind: 'embeddings',
    endpoint: '/embeddings',
    defaultModel: 'openpaths-embed',
    select: m => m.tags.includes('embedding'),
    sort: 'price',
    faq: [FAQ_KEY.openai, { q: 'Can I batch inputs?', a: 'Yes. input accepts a string or an array of strings.' }, FAQ_KEY.pricing],
    related: ['chat-completions', 'web-search', 'free-llm'],
  },
  {
    slug: 'text-to-speech',
    path: '/api/text-to-speech',
    name: 'Text to Speech',
    h1: 'Text-to-Speech API',
    tagline: 'Natural voices over one endpoint',
    seoTitle: 'Text-to-Speech API - OpenAI-Compatible /v1/audio/speech | OpenPaths',
    seoDescription: 'Generate speech from text with Gemini TTS, xAI TTS and CosyVoice through /v1/audio/speech. Price comparison, in-browser player and code snippets.',
    intro: 'Send text, get base64 audio back. Choose between expressive multi-speaker Gemini TTS, xAI voices and open-source CosyVoice, all billed per character or minute.',
    kind: 'tts',
    endpoint: '/audio/speech',
    defaultModel: 'xai-tts',
    studioPath: '/tools/google-tts',
    studioLabel: 'Open the Gemini TTS studio',
    select: m => m.tags.includes('audio') && /tts/.test(m.id),
    sort: 'price',
    faq: [{ q: 'What does the response look like?', a: 'JSON with a base64 audio field and a format field. Decode it and write it to a file, or play it from a data URL.' }, FAQ_KEY.signup, FAQ_KEY.pricing],
    related: ['speech-to-text', 'music-generation', 'chat-completions'],
  },
  {
    slug: 'speech-to-text',
    path: '/api/speech-to-text',
    name: 'Speech to Text',
    h1: 'Speech-to-Text API (Transcription)',
    tagline: 'OpenAI-compatible /v1/audio/transcriptions',
    seoTitle: 'Speech-to-Text API - OpenAI-Compatible Transcription | OpenPaths',
    seoDescription: 'Transcribe audio with an OpenAI-compatible /v1/audio/transcriptions endpoint. Per-hour pricing, upload-and-transcribe demo and code snippets.',
    intro: 'Upload an audio file and get text back using the same multipart request shape as the OpenAI transcription API, billed per audio hour.',
    kind: 'stt',
    endpoint: '/audio/transcriptions',
    defaultModel: 'xai-stt',
    select: m => m.tags.includes('audio') && /stt|transcribe/.test(m.id),
    sort: 'price',
    faq: [FAQ_KEY.openai, { q: 'Which formats are accepted?', a: 'Common audio containers such as mp3, wav, m4a, webm and ogg, up to the gateway request size limit.' }, FAQ_KEY.pricing],
    related: ['text-to-speech', 'chat-completions', 'embeddings'],
  },
  {
    slug: 'image-generation',
    path: '/api/image-generation',
    name: 'Image Generation',
    h1: 'AI Image Generation API (OpenAI-Compatible)',
    tagline: 'FLUX, GPT Image, RA1 and more',
    seoTitle: 'AI Image Generation API - OpenAI-Compatible, 25+ Models | OpenPaths',
    seoDescription: 'Generate images with GPT Image 2, FLUX, RA1, Grok Imagine and more through /v1/images/generations. Per-image pricing, live generator and snippets.',
    intro: 'The standard OpenAI images endpoint backed by every major text-to-image model. Use openpaths/auto-image to let the router pick, or pin a model for consistency.',
    kind: 'image',
    endpoint: '/images/generations',
    defaultModel: 'openpaths/auto-image',
    studioPath: '/text-to-image',
    studioLabel: 'Open the text-to-image studio',
    select: m => m.tags.includes('text-to-image'),
    sort: 'price',
    faq: [FAQ_KEY.openai, { q: 'Can I edit images too?', a: 'Yes. POST /v1/images/edits accepts OpenAI multipart or JSON image URLs; see the image edit studio.' }, FAQ_KEY.pricing],
    related: ['image-editing', 'video-generation', '3d-generation', 'vision-llm'],
    vsLimit: 5,
  },
  {
    slug: 'image-editing',
    path: '/api/image-editing',
    name: 'Image Editing',
    h1: 'AI Image Editing API',
    tagline: 'Style transfer, outpainting and edits',
    seoTitle: 'AI Image Editing API - Edit, Outpaint, Style Transfer | OpenPaths',
    seoDescription: 'Edit images with GPT Image 2, FLUX Kontext, Qwen and RA2 edit through /v1/images/edits. Compare prices and open the editor.',
    intro: 'Describe a change and send an image: instruction edits, outpainting, virtual try-on and style transfer share one /v1/images/edits endpoint.',
    kind: 'studio',
    endpoint: '/images/edits',
    defaultModel: 'flux-kontext-pro',
    studioPath: '/image-edit',
    studioLabel: 'Open the image edit studio',
    select: m => m.tags.includes('image-to-image') && !m.tags.includes('image-to-3d'),
    sort: 'price',
    sampleBody: { model: 'flux-kontext-pro', prompt: 'Make it a watercolor painting', image_url: 'https://example.com/photo.jpg' },
    faq: [FAQ_KEY.openai, FAQ_KEY.pricing],
    related: ['image-generation', 'video-generation', '3d-generation'],
    vsLimit: 5,
  },
  {
    slug: 'video-generation',
    path: '/api/video-generation',
    name: 'Video Generation',
    h1: 'AI Video Generation API',
    tagline: 'Text and image to video with native audio',
    seoTitle: 'AI Video Generation API - Wan, Seedance, FLUX, Grok Video | OpenPaths',
    seoDescription: 'One API for text-to-video and image-to-video: Wan 3.0, Seedance, FLUX Video, MiniMax and more. Per-second price comparison and code snippets.',
    intro: 'Video jobs are asynchronous: POST to /v1/videos/generations, then poll the job until the MP4 is ready. Pricing is per generated second.',
    kind: 'studio',
    endpoint: '/videos/generations',
    defaultModel: 'wan-3.0-text-to-video',
    studioPath: '/text-to-video',
    studioLabel: 'Open the text-to-video studio',
    select: m => m.tags.includes('video generation'),
    sort: 'price',
    sampleBody: { model: 'wan-3.0-text-to-video', prompt: 'A paper boat drifting down a rain-soaked street at dusk', resolution: '720p', aspect_ratio: '16:9', duration: 5, generate_audio: true },
    faq: [{ q: 'How do I get the finished video?', a: 'The create call returns a job_id. Poll GET /v1/videos/generations/{job_id} until status is completed and read the video URL.' }, FAQ_KEY.signup, FAQ_KEY.pricing],
    related: ['image-generation', 'video-relighting', 'music-generation', '3d-generation'],
    vsLimit: 6,
  },
  {
    slug: 'video-relighting',
    path: '/api/video-relighting',
    name: 'Video Relighting',
    h1: 'AI Video Relighting API',
    tagline: 'Re-light a clip from a reference sphere',
    seoTitle: 'AI Video Relighting API - MiniMax H3 Max Relight | OpenPaths',
    seoDescription: 'Relight any video with MiniMax H3 Max Relight: pass a clip and a sphere render of the target lighting to /v1/videos/generations. From $0.055 per second.',
    intro: 'Send a source video (up to 15 seconds, at least about 2.3) and a reference sphere image that shows the lighting you want: warm key, neon rim, golden hour. The model re-lights the footage and keeps the motion. Jobs are asynchronous; billing is per second of clip length, 3 to 15 seconds.',
    kind: 'studio',
    endpoint: '/videos/generations',
    defaultModel: 'minimax-h3-max-relight',
    studioPath: '/models/minimax-h3-max-relight',
    studioLabel: 'Open the relighting space',
    select: m => m.id === 'minimax-h3-max-relight',
    sort: 'price',
    sampleBody: { model: 'minimax-h3-max-relight', video_url: 'https://example.com/clip.mp4', reference_image_url: 'https://example.com/sphere.png', resolution: '768P', aspect_ratio: '16:9' },
    faq: [
      { q: 'What is the reference sphere?', a: 'A render of a shaded sphere lit the way you want the video lit. The space generates presets in the browser, or you can upload your own sphere image.' },
      { q: 'How is relighting billed?', a: '$0.055 per second at 480P, $0.088 at 768P, $0.176 at 1080P and $0.352 at 2K, rounded up to whole seconds and clamped to 3 to 15 seconds.' },
      FAQ_KEY.signup,
    ],
    related: ['video-generation', 'image-editing', 'image-generation'],
  },
  {
    slug: 'music-generation',
    path: '/api/music-generation',
    name: 'Music Generation',
    h1: 'AI Music Generation API',
    tagline: 'Songs, loops and instrumentals from a prompt',
    seoTitle: 'AI Music Generation API - Lyria 3 & MiniMax Music | OpenPaths',
    seoDescription: 'Generate songs and instrumentals with Google Lyria 3 and MiniMax through /v1/music/generations. Per-request pricing and code snippets.',
    intro: 'Describe genre, mood and structure and receive an audio file. Lyria 3 Clip makes 30-second clips; Lyria 3 Pro produces full-length songs.',
    kind: 'studio',
    endpoint: '/music/generations',
    defaultModel: 'lyria-3-clip-preview',
    studioPath: '/tools/lyria',
    studioLabel: 'Open the Lyria music studio',
    select: m => m.tags.includes('audio') && /lyria|music/.test(m.id),
    sort: 'price',
    sampleBody: { model: 'lyria-3-clip-preview', prompt: 'Warm lo-fi hip hop loop with vinyl crackle and soft electric piano' },
    staticRows: [{ id: 'mg-music', name: 'MiniMax-Music3', provider: 'ManifoldGen', price: 'See model page' }],
    faq: [FAQ_KEY.signup, FAQ_KEY.pricing],
    related: ['text-to-speech', 'video-generation', 'image-generation'],
  },
  {
    slug: '3d-generation',
    path: '/api/3d-generation',
    name: '3D Generation',
    h1: 'Image and Text to 3D Model API',
    tagline: 'Textured GLB models via one API',
    seoTitle: 'Image to 3D API - Pixal3D, Meshy, Tripo GLB Generation | OpenPaths',
    seoDescription: 'Turn an image or text prompt into a textured GLB with Pixal3D, Meshy v6 and Tripo through one API. Per-request pricing, rigging and retexture endpoints.',
    intro: 'Submit an image (or a text prompt that is rendered first) and receive a textured GLB. Rigging and retexturing are separate endpoints.',
    kind: 'studio',
    endpoint: '/3d/generations',
    defaultModel: 'pixal3d-image-to-3d',
    studioPath: '/image-to-3d',
    studioLabel: 'Open the image-to-3D studio',
    select: m => m.tags.includes('image-to-3d'),
    sort: 'price',
    sampleBody: { model: 'pixal3d-image-to-3d', image_url: 'https://example.com/character.png' },
    faq: [FAQ_KEY.signup, FAQ_KEY.pricing],
    related: ['image-generation', 'video-generation', 'image-editing'],
  },
  {
    slug: 'web-search',
    path: '/api/web-search',
    name: 'Web Search',
    h1: 'Web Search API for LLMs and Agents',
    tagline: 'Neural search, grounded answers, research papers',
    seoTitle: 'Web Search API for AI Agents - Exa, Gemini, Papers | OpenPaths',
    seoDescription: 'Search the web and research papers with Exa, Gemini grounding, OpenAI web search, Grok and Papers behind one /v1/search endpoint. Per-request pricing.',
    intro: 'One endpoint returns ranked results with page contents or a grounded answer with citations, billed per request.',
    kind: 'studio',
    endpoint: '/search',
    defaultModel: 'exa-search',
    studioPath: '/search',
    studioLabel: 'Open the search playground',
    select: () => false,
    sampleBody: { query: 'latest research on speculative decoding', provider: 'exa', numResults: 5 },
    staticRows: [
      { id: 'exa-search', name: 'Exa Search', provider: 'Exa', price: '$0.0077 / request' },
      { id: 'papers-search', name: 'Papers Search', provider: 'Papers', price: '$0.001 / search' },
      { id: 'gemini', name: 'Gemini grounded answer', provider: 'Google', price: '$0.004 / request' },
      { id: 'openai', name: 'OpenAI web search', provider: 'OpenAI', price: '$0.01 / request' },
      { id: 'grok', name: 'Grok live search', provider: 'xAI', price: '$0.01 / request' },
    ],
    faq: [FAQ_KEY.signup, { q: 'Which providers are supported?', a: 'provider accepts exa (default), papers, gemini, openai and grok.' }, FAQ_KEY.pricing],
    related: ['chat-completions', 'embeddings', 'free-llm'],
  },
];

export const capabilityBySlug = new Map(CAPABILITIES.map(c => [c.slug, c]));

/** Models for a capability, deduplicated and ordered. */
export function capabilityModels(cap: Capability): Model[] {
  const list = models.filter(m => cap.select(m) && (m.priceInput > 0 || m.priceOutput > 0 || FREE_SET.has(m.id)));
  const seen = new Set<string>();
  const unique = list.filter(m => (seen.has(m.id) ? false : (seen.add(m.id), true)));
  if (cap.sort === 'price') return [...unique].sort((a, b) => a.priceInput - b.priceInput || a.popularity - b.popularity);
  return [...unique].sort((a, b) => a.popularity - b.popularity);
}

export function capabilityForModel(m: Model): Capability | undefined {
  if (m.tags.includes('embedding')) return capabilityBySlug.get('embeddings');
  if (m.tags.includes('audio') && /tts/.test(m.id)) return capabilityBySlug.get('text-to-speech');
  if (m.tags.includes('audio') && /stt|transcribe/.test(m.id)) return capabilityBySlug.get('speech-to-text');
  if (m.provider === 'Anthropic' && isChatModel(m)) return capabilityBySlug.get('anthropic-messages');
  if (isChatModel(m)) return capabilityBySlug.get('chat-completions');
  return undefined;
}

/* ---------- head-to-head pairs ---------- */

export interface VsPair {
  slug: string;
  a: Model;
  b: Model;
  cap: Capability;
}

export const modelSlug = (m: Model) => slugify(m.id);

function buildPairs(): VsPair[] {
  const out: VsPair[] = [];
  const seen = new Set<string>();
  for (const cap of CAPABILITIES) {
    if (!cap.vsLimit) continue;
    const byPopularity = [...capabilityModels(cap)].sort((a, b) => a.popularity - b.popularity);
    const providers = new Set<string>();
    const top = byPopularity.filter(m => (providers.has(m.provider) ? false : (providers.add(m.provider), true))).slice(0, cap.vsLimit);
    for (let i = 0; i < top.length; i += 1) {
      for (let j = i + 1; j < top.length; j += 1) {
        const [a, b] = [top[i], top[j]];
        const slug = `${modelSlug(a)}-vs-${modelSlug(b)}`;
        const rev = `${modelSlug(b)}-vs-${modelSlug(a)}`;
        if (seen.has(slug) || seen.has(rev)) continue;
        seen.add(slug);
        out.push({ slug, a, b, cap });
      }
    }
  }
  return out;
}

export const VS_PAIRS: VsPair[] = buildPairs();
export const vsBySlug = new Map(VS_PAIRS.map(p => [p.slug, p]));

export function pairsForModel(m: Model) {
  return VS_PAIRS.filter(p => p.a.id === m.id || p.b.id === m.id);
}

/* ---------- snippets ---------- */

export type Lang = 'python' | 'javascript' | 'curl';

const KEY_PLACEHOLDER = 'op-...';

export function buildSnippets(cap: Capability, model: string, input: string, apiKey = ''): Record<Lang, string> {
  const key = apiKey || KEY_PLACEHOLDER;
  const text = JSON.stringify(input);
  switch (cap.kind) {
    case 'chat':
      return {
        python: `from openai import OpenAI

client = OpenAI(base_url="${API_BASE}", api_key="${key}")

resp = client.chat.completions.create(
    model="${model}",
    messages=[{"role": "user", "content": ${text}}],
)
print(resp.choices[0].message.content)`,
        javascript: `import OpenAI from "openai";

const client = new OpenAI({ baseURL: "${API_BASE}", apiKey: "${key}" });

const resp = await client.chat.completions.create({
  model: "${model}",
  messages: [{ role: "user", content: ${text} }],
});
console.log(resp.choices[0].message.content);`,
        curl: `curl ${API_BASE}/chat/completions \\
  -H "Authorization: Bearer ${key}" \\
  -H "Content-Type: application/json" \\
  -d '${JSON.stringify({ model, messages: [{ role: 'user', content: input }] })}'`,
      };
    case 'messages':
      return {
        python: `import anthropic

client = anthropic.Anthropic(base_url="https://openpaths.io", api_key="${key}")

msg = client.messages.create(
    model="${model}",
    max_tokens=512,
    messages=[{"role": "user", "content": ${text}}],
)
print(msg.content[0].text)`,
        javascript: `import Anthropic from "@anthropic-ai/sdk";

const client = new Anthropic({ baseURL: "https://openpaths.io", apiKey: "${key}" });

const msg = await client.messages.create({
  model: "${model}",
  max_tokens: 512,
  messages: [{ role: "user", content: ${text} }],
});
console.log(msg.content[0].text);`,
        curl: `curl ${API_BASE}/messages \\
  -H "x-api-key: ${key}" \\
  -H "anthropic-version: 2023-06-01" \\
  -H "Content-Type: application/json" \\
  -d '${JSON.stringify({ model, max_tokens: 512, messages: [{ role: 'user', content: input }] })}'`,
      };
    case 'embeddings':
      return {
        python: `from openai import OpenAI

client = OpenAI(base_url="${API_BASE}", api_key="${key}")

resp = client.embeddings.create(model="${model}", input=${text})
print(len(resp.data[0].embedding))`,
        javascript: `import OpenAI from "openai";

const client = new OpenAI({ baseURL: "${API_BASE}", apiKey: "${key}" });

const resp = await client.embeddings.create({ model: "${model}", input: ${text} });
console.log(resp.data[0].embedding.length);`,
        curl: `curl ${API_BASE}/embeddings \\
  -H "Authorization: Bearer ${key}" \\
  -H "Content-Type: application/json" \\
  -d '${JSON.stringify({ model, input })}'`,
      };
    case 'tts':
      return {
        python: `import base64
from openai import OpenAI

client = OpenAI(base_url="${API_BASE}", api_key="${key}")

result = client.post("/audio/speech", body={"model": "${model}", "input": ${text}}, cast_to=dict)
open("speech.mp3", "wb").write(base64.b64decode(result["audio"]))`,
        javascript: `import fs from "node:fs";
import OpenAI from "openai";

const client = new OpenAI({ baseURL: "${API_BASE}", apiKey: "${key}" });

const result = await client.post("/audio/speech", { body: { model: "${model}", input: ${text} } });
fs.writeFileSync("speech.mp3", Buffer.from(result.audio, "base64"));`,
        curl: `curl ${API_BASE}/audio/speech \\
  -H "Authorization: Bearer ${key}" \\
  -H "Content-Type: application/json" \\
  -d '${JSON.stringify({ model, input })}'`,
      };
    case 'stt':
      return {
        python: `from openai import OpenAI

client = OpenAI(base_url="${API_BASE}", api_key="${key}")

with open("audio.mp3", "rb") as f:
    resp = client.audio.transcriptions.create(model="${model}", file=f)
print(resp.text)`,
        javascript: `import fs from "node:fs";
import OpenAI from "openai";

const client = new OpenAI({ baseURL: "${API_BASE}", apiKey: "${key}" });

const resp = await client.audio.transcriptions.create({
  model: "${model}",
  file: fs.createReadStream("audio.mp3"),
});
console.log(resp.text);`,
        curl: `curl ${API_BASE}/audio/transcriptions \\
  -H "Authorization: Bearer ${key}" \\
  -F model=${model} \\
  -F file=@audio.mp3`,
      };
    case 'image':
      return {
        python: `from openai import OpenAI

client = OpenAI(base_url="${API_BASE}", api_key="${key}")

result = client.images.generate(model="${model}", prompt=${text}, size="1024x1024", n=1)
print(result.data[0].url or result.data[0].b64_json[:32])`,
        javascript: `import OpenAI from "openai";

const client = new OpenAI({ baseURL: "${API_BASE}", apiKey: "${key}" });

const result = await client.images.generate({ model: "${model}", prompt: ${text}, size: "1024x1024", n: 1 });
console.log(result.data[0].url ?? result.data[0].b64_json.slice(0, 32));`,
        curl: `curl ${API_BASE}/images/generations \\
  -H "Authorization: Bearer ${key}" \\
  -H "Content-Type: application/json" \\
  -d '${JSON.stringify({ model, prompt: input, size: '1024x1024', n: 1 })}'`,
      };
    default: {
      const body = JSON.stringify({ ...(cap.sampleBody || {}), ...(cap.sampleBody && 'model' in cap.sampleBody ? { model } : {}) }, null, 2);
      return {
        python: `import json
from openai import OpenAI

client = OpenAI(base_url="${API_BASE}", api_key="${key}")

result = client.post("${cap.endpoint}", body=json.loads(r'''${body}'''), cast_to=dict)
print(result)`,
        javascript: `import OpenAI from "openai";

const client = new OpenAI({ baseURL: "${API_BASE}", apiKey: "${key}" });

const result = await client.post("${cap.endpoint}", { body: ${body} });
console.log(result);`,
        curl: `curl ${API_BASE}${cap.endpoint} \\
  -H "Authorization: Bearer ${key}" \\
  -H "Content-Type: application/json" \\
  -d '${body}'`,
      };
    }
  }
}

export const DEFAULT_INPUT: Record<TryKind, string> = {
  chat: 'Explain what an API gateway does in two sentences.',
  messages: 'Explain what an API gateway does in two sentences.',
  embeddings: 'OpenPaths routes one API key to every model.',
  tts: 'Welcome to OpenPaths, one API for every model.',
  stt: '',
  image: 'A beige ceramic coffee mug on a wooden table, soft window light, editorial product photo',
  studio: '',
};
