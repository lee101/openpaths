export interface ToolEntry {
  slug: string;
  path: string;
  name: string;
  tagline: string;
  description: string;
  price: string;
  keywords: string;
  seoTitle: string;
  seoDescription: string;
}

export const BASE_URL = 'https://openpaths.io';

/** Branded 1200x630 social card. */
export const toolOgImage = (slug: string) => `${BASE_URL}/og/tools/${slug}.webp`;

/** Unbranded 3:2 art the card is composited from, used by the /tools grid. */
export const toolArtImage = (slug: string) => `/og/tools/art/${slug}.webp`;

export const TOOLS_INDEX_SLUG = 'index';

export const TOOLS: ToolEntry[] = [
  {
    slug: 'google-tts',
    path: '/tools/google-tts',
    name: 'Gemini Flash TTS',
    tagline: '30 voices · multi-speaker',
    description: 'Direct expressive narration and two-person scenes with control over voice, style, pace, accent, context, and emotion.',
    price: 'Gemini token pricing',
    keywords: 'speech audio voice narration tts text to speech gemini dialogue podcast',
    seoTitle: 'Gemini Flash TTS Studio - Multi-speaker AI Voice | OpenPaths',
    seoDescription: 'Generate steerable single- and multi-speaker speech with Gemini 3.1 Flash TTS. Direct voice, style, pace, accent, scene, and emotion, then play, download, or copy the API code.',
  },
  {
    slug: 'lyria',
    path: '/tools/lyria',
    name: 'Lyria 3 Music Studio',
    tagline: 'Pro songs · 30s clips · Opus',
    description: 'Compose full songs, scores, and loops with precise control over genre, mood, arrangement, vocals, lyrics, and structure.',
    price: '$0.04 clip · $0.08 pro',
    keywords: 'music song audio soundtrack score loop lyria compose instrumental',
    seoTitle: 'Lyria 3 Music Studio - AI Song & Instrumental Generator | OpenPaths',
    seoDescription: 'Generate complete songs, instrumentals, loops, and 30-second clips with Google Lyria 3 Pro and Clip. Direct genre, mood, structure, instruments, vocals, lyrics, and export as Opus.',
  },
  {
    slug: 'text-to-image',
    path: '/text-to-image',
    name: 'Text to Image',
    tagline: 'Auto image endpoint',
    description: 'Generate an image from a prompt. The auto image route picks the best model (GPT Image 2, RA1, Flux), or pin your own.',
    price: 'from ~$0.04 / image',
    keywords: 'image art picture generate flux gpt image ra1 draw illustration',
    seoTitle: 'Text to Image API | OpenPaths',
    seoDescription: 'Generate images from text with the OpenPaths auto image endpoint - routes to GPT Image 2, RA1, Flux, and more with near-zero markup.',
  },
  {
    slug: 'image-edit',
    path: '/image-edit',
    name: 'Image Style Transfer',
    tagline: 'GPT Image 2 · routed edits',
    description: 'Upload a source image and describe the new visual world. The shared image-edit route handles GPT Image 2 and fallbacks.',
    price: '~$0.30 / edit',
    keywords: 'image edit style transfer restyle photo inpaint modify',
    seoTitle: 'AI Image Style Transfer | OpenPaths',
    seoDescription: 'Upload an image and describe a new visual direction. OpenPaths routes the edit through GPT Image 2 and image-editing fallbacks.',
  },
  {
    slug: 'text-to-video',
    path: '/text-to-video',
    name: 'Text to Video',
    tagline: 'Wan 3.0 · native audio',
    description: 'Describe a shot and get up to 30 seconds of cinematic video with native audio. Smart duration or pin exact seconds, per-resolution pricing.',
    price: '$0.05 - $0.20 / sec',
    keywords: 'video clip film cinematic wan motion generate movie',
    seoTitle: 'Text to Video API - Wan 3.0 | OpenPaths',
    seoDescription: 'Generate up-to-30s cinematic video with native audio from one prompt through Wan 3.0 on OpenPaths. Per-second pricing by resolution.',
  },
  {
    slug: 'image-to-video',
    path: '/image-to-video',
    name: 'Image to Video',
    tagline: 'Wan 3.0 · first + last frame',
    description: 'Animate any still with native audio, optionally driving it to an end frame. Smart duration or pin exact seconds.',
    price: '$0.05 - $0.20 / sec',
    keywords: 'video animate still photo motion wan frame keyframe',
    seoTitle: 'Image to Video API - Wan 3.0 | OpenPaths',
    seoDescription: 'Animate a still into up-to-30s video with native audio through Wan 3.0 on OpenPaths. Optional end frame, per-second pricing by resolution.',
  },
  {
    slug: 'video-extension',
    path: '/video-extension',
    name: 'Video Extension',
    tagline: 'Grok Imagine Video',
    description: 'Upload or paste an MP4, describe what happens next, and extend it through the Grok Imagine video API.',
    price: '~$0.08 / sec',
    keywords: 'video extend continue longer grok imagine mp4 clip',
    seoTitle: 'Video Edit and Extension API | OpenPaths',
    seoDescription: 'Edit or extend an existing MP4 with Grok Imagine Video through OpenPaths. Describe the next beat and continue any clip.',
  },
  {
    slug: 'character-animator',
    path: '/character-animator',
    name: 'Character Animator',
    tagline: 'Wan-Animate',
    description: 'Animate a reference character with a driving performance video. Standard, fast, and x-fast latency lanes.',
    price: '$0.15 - $0.60 / sec',
    keywords: 'character animate avatar performance motion capture puppet wan animate video',
    seoTitle: 'Character Animator - Wan-Animate API | OpenPaths',
    seoDescription: "Animate a reference character with a driving performance video through Wan-Animate on ManifoldGen, OpenPaths' own GPU studio. Standard, fast, and x-fast latency lanes with per-second pricing.",
  },
  {
    slug: 'music-generator',
    path: '/music-generator',
    name: 'Music Generator',
    tagline: 'MiniMax-Music3',
    description: 'Full songs with vocals from a prompt and optional lyrics. Pin 30-300 seconds.',
    price: 'from $0.35 / track',
    keywords: 'music song vocals lyrics audio track minimax generate',
    seoTitle: 'AI Music Generator - MiniMax-Music3 API | OpenPaths',
    seoDescription: "Generate full songs with vocals from a prompt and optional lyrics through MiniMax-Music3 on ManifoldGen, OpenPaths' own GPU studio. Pin any length from 30 to 300 seconds.",
  },
  {
    slug: 'remove-video-background',
    path: '/remove-video-background',
    name: 'Background Remover',
    tagline: 'Transparent video',
    description: 'Key any clip to alpha-transparent WebM, optionally compositing a solid backdrop color.',
    price: '~$0.005 / sec',
    keywords: 'video background remove alpha transparent matte key chroma greenscreen',
    seoTitle: 'Remove Video Background - Transparent WebM API | OpenPaths',
    seoDescription: "Key any clip to alpha-transparent WebM through ManifoldGen, OpenPaths' own GPU studio. Optionally composite a solid backdrop color and keep the original audio.",
  },
  {
    slug: 'image-to-3d',
    path: '/image-to-3d',
    name: 'Image to 3D',
    tagline: 'Pixal3D · Meshy v6 · Tripo p1',
    description: 'Upload or paste an object image and get back a textured GLB you can preview, rotate, and download. Choose Pixal3D (fast), Meshy v6 (quality), or Tripo p1 (sharp geometry).',
    price: '$0.30 - $0.80',
    keywords: '3d model mesh glb object photogrammetry meshy tripo pixal3d',
    seoTitle: 'Image to 3D API | OpenPaths',
    seoDescription: 'Generate textured GLB models from a single image through OpenPaths with Pixal3D, Meshy v6, and Tripo p1.',
  },
  {
    slug: 'text-to-3d',
    path: '/text-to-3d',
    name: 'Text to 3D',
    tagline: 'Auto Image + Pixal3D',
    description: 'Type a prompt: OpenPaths generates an image then converts it to a textured GLB in one call. Pay image + 3D price.',
    price: 'image + 3D price',
    keywords: '3d model mesh glb prompt asset game generate',
    seoTitle: 'Text to 3D API | OpenPaths',
    seoDescription: 'Generate textured GLB models straight from a text prompt. OpenPaths auto-generates an image then converts it to 3D with Pixal3D.',
  },
  {
    slug: 'rig-3d',
    path: '/rig-3d',
    name: '3D Auto-Rigging',
    tagline: 'Fal Meshy Rigging',
    description: 'Upload a humanoid GLB and get back a rigged character (GLB + FBX) with walk/run animations - preview it right in the browser.',
    price: '$0.20 - $0.32',
    keywords: '3d rig skeleton bones character animation fbx glb walk run',
    seoTitle: '3D Auto-Rigging API | OpenPaths',
    seoDescription: 'Upload a humanoid GLB and get back a rigged character (GLB + FBX) with optional walk/run animation, via OpenPaths and Fal Meshy.',
  },
  {
    slug: 'retexture-3d',
    path: '/retexture-3d',
    name: '3D Retexture',
    tagline: 'Fal Trellis-2',
    description: 'Upload an existing mesh plus a reference image and re-skin it with a fresh texture - preview and download the new GLB.',
    price: '$0.20 - $0.24',
    keywords: '3d texture retexture material skin mesh glb trellis paint',
    seoTitle: '3D Retexture API | OpenPaths',
    seoDescription: 'Re-texture an existing 3D mesh from a reference image - upload a GLB and a style image, get back a textured GLB, via OpenPaths and Fal Trellis-2.',
  },
  {
    slug: 'playground',
    path: '/playground',
    name: 'Playground',
    tagline: 'Chat · image · video · audio',
    description: 'Multi-pane studio to test any model across modalities side by side, with live cost and latency.',
    price: 'pay per request',
    keywords: 'playground chat test compare models multimodal studio sandbox',
    seoTitle: 'Playground - Test Any AI Model Side by Side | OpenPaths',
    seoDescription: 'Multi-pane studio for chat, image, video, and audio models. Compare any two models on the same prompt with live cost and latency, then copy the API snippet.',
  },
  {
    slug: 'fusion',
    path: '/fusion',
    name: 'Model Fusion',
    tagline: 'OpenRouter fusion beta',
    description: 'Run multiple models side by side, analyze consensus and contradictions, then fuse the best result into one answer.',
    price: 'panel + judge cost',
    keywords: 'fusion ensemble consensus compare judge multiple models merge',
    seoTitle: 'Model Fusion Beta | OpenPaths',
    seoDescription: 'Run multiple models side by side with OpenRouter fusion, analyze the panel, and fuse the strongest result into one answer.',
  },
];

export const TOOLS_INDEX_SEO = {
  path: '/tools',
  seoTitle: 'AI Tools - Image, Video, Music, Speech, 3D | OpenPaths',
  seoDescription: 'Hands-on studios for every OpenPaths generation endpoint: text to image, video, music, speech, 3D, model fusion, and the multi-model playground. Each tool is a thin wrapper over the API.',
};

export const toolBySlug = new Map(TOOLS.map(tool => [tool.slug, tool]));
