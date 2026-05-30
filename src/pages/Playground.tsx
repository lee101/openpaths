import React, { useState, useRef, useEffect, useCallback, useMemo } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { Send, Plus, X, Settings, ChevronDown, Loader2, Trash2, Square, Copy, Check, Zap, RotateCcw, Code2, Share2, Wallet, Eye, Wrench, Volume2, Bookmark, BookmarkCheck } from 'lucide-react';
import { CodeBlock as HighlightedCodeBlock } from '../components/CodeBlock';
import { VIDEO_DEMOS, type VideoDemo } from '../data/videoDemos';
import { prepareUploadFile } from '../lib/imageUpload';
import { normalizeUploadedAssetUrl } from '../lib/uploadUrls';
import { fetchPrompts, loadSavedPrompts, removeSavedPrompt, savePrompt, type SavedPrompt } from '../lib/promptLibrary';
import type { LibraryPrompt } from '../data/promptLibrary';

interface Message {
  role: 'system' | 'user' | 'assistant';
  content: string;
  imageB64?: string;
  imageUrl?: string;
  videoUrl?: string;
  audioB64?: string;
  audioUrl?: string;
  audioFormat?: string;
}

interface ModelPricing {
  input_per_1m_tokens?: number;
  input_cache_hit_per_1m_tokens?: number;
  output_per_1m_tokens?: number;
  per_request?: number;
  per_image?: number;
  per_megapixel?: number;
  first_megapixel?: number;
  extra_megapixel?: number;
  per_video?: number;
  per_second?: number;
  per_second_with_video_input?: number;
}

interface ModelCapabilities {
  streaming?: boolean;
  tools?: boolean;
  vision?: boolean;
}

interface CatalogModel {
  id: string;
  label: string;
  provider: string;
  pricing?: ModelPricing;
  capabilities?: ModelCapabilities;
}

interface ModelPane {
  id: string;
  modelId: string;
  messages: Message[];
  streaming: boolean;
  error: string | null;
  latencyMs: number | null;
  tokensUsed: number | null;
  promptTokens: number | null;
  completionTokens: number | null;
  costUsd: number | null;
}

type CodeLanguage = 'python' | 'js' | 'go' | 'curl';
type PromptExample = string | { label: string; prompt: string };
type ReferenceUploadTarget = 'image' | 'end-image' | 'images' | 'video' | 'audio';

async function pollVideoJob(baseUrl: string, apiKey: string, jobId: string, signal: AbortSignal) {
  const deadline = Date.now() + 15 * 60 * 1000;
  while (Date.now() < deadline) {
    await new Promise<void>((resolve, reject) => {
      const timer = window.setTimeout(resolve, 2000);
      signal.addEventListener('abort', () => {
        window.clearTimeout(timer);
        reject(new DOMException('Aborted', 'AbortError'));
      }, { once: true });
    });
    const resp = await fetch(`${baseUrl}/v1/videos/generations/${encodeURIComponent(jobId)}`, {
      headers: { Authorization: `Bearer ${apiKey}` },
      signal,
    });
    if (!resp.ok) {
      const errText = await resp.text();
      throw new Error(`${resp.status}: ${errText.slice(0, 200)}`);
    }
    const data = await resp.json();
    if (data?.status === 'completed' && data?.result?.video_url) return data.result;
    if (data?.status === 'failed') throw new Error(data?.error?.message || 'Video generation failed');
  }
  throw new Error('Video generation timed out');
}

const IMAGE_SIZES = ['1024x1024', '1152x768', '768x1152', '1360x768', '768x1360', '1280x720', '720x1280'] as const;
const IMAGE_ASPECT_RATIOS = ['auto', '1:1', '16:9', '9:16', '4:3', '3:4', '3:2', '2:3', '2:1', '1:2'] as const;
const IMAGE_QUALITIES = ['standard', 'high'] as const;
const IMAGE_RESPONSE_FORMATS = ['url', 'b64_json'] as const;
const VIDEO_RESOLUTIONS = ['480p', '720p', '1080p'] as const;
const VIDEO_DURATIONS = ['auto', '4', '5', '6', '7', '8', '9', '10', '11', '12', '13', '14', '15'] as const;
const VIDEO_ASPECT_RATIOS = ['auto', '21:9', '16:9', '4:3', '1:1', '3:4', '9:16'] as const;
const SPEECH_VOICES = ['eve', 'ara', 'rex', 'sal', 'leo'];
const GEMINI_TTS_VOICE_INFO = [
  { name: 'Achernar', style: 'Soft', pitch: 'Higher pitch' },
  { name: 'Achird', style: 'Friendly', pitch: 'Lower middle pitch' },
  { name: 'Algenib', style: 'Gravelly', pitch: 'Lower pitch' },
  { name: 'Algieba', style: 'Smooth', pitch: 'Lower pitch' },
  { name: 'Alnilam', style: 'Firm', pitch: 'Lower middle pitch' },
  { name: 'Aoede', style: 'Breezy', pitch: 'Middle pitch' },
  { name: 'Autonoe', style: 'Bright', pitch: 'Middle pitch' },
  { name: 'Callirrhoe', style: 'Easy-going', pitch: 'Middle pitch' },
  { name: 'Charon', style: 'Informative', pitch: 'Lower pitch' },
  { name: 'Despina', style: 'Smooth', pitch: 'Middle pitch' },
  { name: 'Enceladus', style: 'Breathy', pitch: 'Lower pitch' },
  { name: 'Erinome', style: 'Clear', pitch: 'Middle pitch' },
  { name: 'Fenrir', style: 'Excitable', pitch: 'Lower middle pitch' },
  { name: 'Gacrux', style: 'Mature', pitch: 'Middle pitch' },
  { name: 'Iapetus', style: 'Clear', pitch: 'Lower middle pitch' },
  { name: 'Kore', style: 'Firm', pitch: 'Middle pitch' },
  { name: 'Laomedeia', style: 'Upbeat', pitch: 'Higher pitch' },
  { name: 'Leda', style: 'Youthful', pitch: 'Higher pitch' },
  { name: 'Orus', style: 'Firm', pitch: 'Lower middle pitch' },
  { name: 'Puck', style: 'Upbeat', pitch: 'Middle pitch' },
  { name: 'Pulcherrima', style: 'Forward', pitch: 'Middle pitch' },
  { name: 'Rasalgethi', style: 'Informative', pitch: 'Middle pitch' },
  { name: 'Sadachbia', style: 'Lively', pitch: 'Lower pitch' },
  { name: 'Sadaltager', style: 'Knowledgeable', pitch: 'Middle pitch' },
  { name: 'Schedar', style: 'Even', pitch: 'Lower middle pitch' },
  { name: 'Sulafat', style: 'Warm', pitch: 'Middle pitch' },
  { name: 'Umbriel', style: 'Easy-going', pitch: 'Lower middle pitch' },
  { name: 'Vindemiatrix', style: 'Gentle', pitch: 'Middle pitch' },
  { name: 'Zephyr', style: 'Bright', pitch: 'Higher pitch' },
  { name: 'Zubenelgenubi', style: 'Casual', pitch: 'Lower middle pitch' },
] as const;
const GEMINI_TTS_VOICES = GEMINI_TTS_VOICE_INFO.map(v => v.name);
const TTS_STYLES = ['Natural', 'Deadpan', 'Empathetic', 'Dramatic', 'Whispering', 'Excited', 'Calm', 'Authoritative', 'Playful', 'Suspicious'] as const;
const TTS_PACES = ['Natural', 'Slow', 'Measured', 'Fast', 'Staccato', 'Urgent'] as const;
const TTS_ACCENTS = ['American (Gen)', 'British (RP)', 'Neutral', 'Australian', 'Indian English', 'Irish', 'Scottish'] as const;
const SPEECH_LANGUAGES = ['en', 'es', 'fr', 'de', 'it', 'pt', 'ja', 'ko', 'zh'] as const;

const FALLBACK_MODELS: CatalogModel[] = [
  { id: 'openpaths/auto', label: 'OpenPaths Auto (hero)', provider: 'OpenPaths' },
  { id: 'openpaths/auto-code', label: 'OpenPaths Auto Code', provider: 'OpenPaths' },
  { id: 'openpaths/auto-fast', label: 'OpenPaths Auto Fast', provider: 'OpenPaths' },
  { id: 'openpaths/auto-cheap', label: 'OpenPaths Auto Cheap', provider: 'OpenPaths' },
  { id: 'openpaths/auto-reasoning', label: 'OpenPaths Auto Reasoning', provider: 'OpenPaths' },
  { id: 'openpaths/auto-vision', label: 'OpenPaths Auto Vision', provider: 'OpenPaths' },
  { id: 'openpaths/auto-image', label: 'OpenPaths Auto Image', provider: 'OpenPaths', pricing: { per_image: 0.211 } },
  { id: 'gemini-latest', label: 'Gemini Latest', provider: 'Google', pricing: { input_per_1m_tokens: 1.50, output_per_1m_tokens: 9.00 } },
  { id: 'gemini-3.5-flash', label: 'Gemini 3.5 Flash', provider: 'Google', pricing: { input_per_1m_tokens: 1.50, output_per_1m_tokens: 9.00 } },
  { id: 'gemini-3.1-flash-lite', label: 'Gemini 3.1 Flash Lite', provider: 'Google', pricing: { input_per_1m_tokens: 0.25, output_per_1m_tokens: 1.50 } },
  { id: 'gemini-2.5-pro', label: 'Gemini 2.5 Pro', provider: 'Google' },
  { id: 'gemini-2.5-flash', label: 'Gemini 2.5 Flash', provider: 'Google' },
  { id: 'gemini-3.1-flash-tts-preview', label: 'Gemini 3.1 Flash TTS Preview', provider: 'Google', pricing: { input_per_1m_tokens: 1.00, output_per_1m_tokens: 20.00 } },
  { id: 'lyria-3-pro-preview', label: 'Lyria 3 Pro Preview', provider: 'Google', pricing: { per_request: 0.08 } },
  { id: 'lyria-3-clip-preview', label: 'Lyria 3 Clip Preview', provider: 'Google', pricing: { per_request: 0.04 } },
  { id: 'gpt-5.5', label: 'GPT-5.5', provider: 'OpenAI' },
  { id: 'gpt-5.4', label: 'GPT-5.4', provider: 'OpenAI' },
  { id: 'o3', label: 'o3', provider: 'OpenAI' },
  { id: 'o4-mini', label: 'o4-mini', provider: 'OpenAI' },
  { id: 'gpt-4o', label: 'GPT-4o', provider: 'OpenAI' },
  { id: 'gpt-4o-mini', label: 'GPT-4o Mini', provider: 'OpenAI' },
  { id: 'claude-sonnet-latest', label: 'Claude Sonnet (latest)', provider: 'Anthropic' },
  { id: 'claude-opus-latest', label: 'Claude Opus (latest)', provider: 'Anthropic' },
  { id: 'claude-haiku-4-5-20251001', label: 'Claude Haiku', provider: 'Anthropic' },
  { id: 'grok-4.3', label: 'Grok 4.3', provider: 'xAI' },
  { id: 'grok-latest', label: 'Grok Latest', provider: 'xAI' },
  { id: 'grok-4.20-non-reasoning', label: 'Grok 4.20 Non-Reasoning', provider: 'xAI' },
  { id: 'grok-3-mini', label: 'Grok 3 Mini', provider: 'xAI' },
  { id: 'xai-tts', label: 'xAI Text to Speech', provider: 'xAI', pricing: { input_per_1m_tokens: 15.00 } },
  { id: 'grok-imagine-image', label: 'Grok Imagine Image', provider: 'xAI', pricing: { per_image: 0.02 } },
  { id: 'deepseek-v4-flash', label: 'DeepSeek V4 Flash', provider: 'DeepSeek', pricing: { input_per_1m_tokens: 0.14, input_cache_hit_per_1m_tokens: 0.0028, output_per_1m_tokens: 0.28 } },
  { id: 'deepseek-v4-pro', label: 'DeepSeek V4 Pro', provider: 'DeepSeek', pricing: { input_per_1m_tokens: 0.435, input_cache_hit_per_1m_tokens: 0.003625, output_per_1m_tokens: 0.87 } },
  { id: 'nvidia/deepseek-v4-pro', label: 'DeepSeek V4 Pro Free', provider: 'NVIDIA' },
  { id: 'deepseek-chat', label: 'DeepSeek Chat', provider: 'DeepSeek' },
  { id: 'deepseek-reasoner', label: 'DeepSeek Reasoner', provider: 'DeepSeek' },
  { id: 'mistral-large-latest', label: 'Mistral Large', provider: 'Mistral' },
  { id: 'llama-3.3-70b-versatile', label: 'Llama 3.3 70B', provider: 'Groq' },
  { id: 'glm-5', label: 'GLM-5', provider: 'Together' },
  { id: 'qwen3.5-397b', label: 'Qwen 3.5 397B', provider: 'Together' },
  { id: 'minimax-m2.7', label: 'MiniMax M2.7', provider: 'MiniMax' },
  { id: 'minimax-m2.5-direct', label: 'MiniMax M2.5', provider: 'MiniMax' },
  { id: 'kimi-k2.5', label: 'Kimi K2.5', provider: 'Together' },
  { id: 'ra1', label: 'RA1 Art Generator', provider: 'Netwrck', pricing: { per_image: 0.04 } },
  { id: 'zimage', label: 'ZImage', provider: 'Netwrck', pricing: { per_image: 0.007 } },
  { id: 'klein', label: 'FLUX Klein 4B', provider: 'Fal', pricing: { per_image: 0.02 } },
  { id: 'flux-pro', label: 'FLUX.1 Pro', provider: 'Fal', pricing: { per_image: 0.04 } },
  { id: 'flux-dev', label: 'FLUX Dev', provider: 'Fal', pricing: { per_image: 0.025 } },
  { id: 'flux-schnell', label: 'FLUX Schnell', provider: 'Fal', pricing: { per_image: 0.003 } },
  { id: 'stable-diffusion-3', label: 'Stable Diffusion 3', provider: 'Together', pricing: { per_image: 0.002 } },
  { id: 'glm-image', label: 'GLM Image', provider: 'Z.AI', pricing: { per_image: 0.015 } },
  { id: 'fal-gpt-image-2', label: 'GPT Image 2 via Fal', provider: 'Fal', pricing: { per_image: 0.22 } },
  { id: 'hidream-o1-image-dev', label: 'HiDream O1 Image Dev', provider: 'Fal', pricing: { per_megapixel: 0.011 } },
  { id: 'fal-ai/hidream-o1-image/edit', label: 'HiDream O1 Image Edit', provider: 'Fal', pricing: { per_megapixel: 0.011 } },
  { id: 'fal-ai/flux-2-pro/outpaint', label: 'FLUX 2 Pro Outpaint', provider: 'Fal', pricing: { first_megapixel: 0.033, extra_megapixel: 0.0165 } },
  { id: 'seedance-2.0-fast-text-to-video', label: 'Seedance 2.0 Fast Text to Video', provider: 'Fal', pricing: { per_second: 0.26609 } },
  { id: 'seedance-2.0-text-to-video', label: 'Seedance 2.0 Text to Video', provider: 'Fal', pricing: { per_second: 0.33374 } },
  { id: 'seedance-2.0-image-to-video', label: 'Seedance 2.0 Image to Video', provider: 'Fal', pricing: { per_second: 0.33264 } },
  { id: 'seedance-2.0-fast-reference-to-video', label: 'Seedance 2.0 Fast Reference to Video', provider: 'Fal', pricing: { per_second: 0.26609, per_second_with_video_input: 0.15972 } },
  { id: 'seedance-2.0-reference-to-video', label: 'Seedance 2.0 Reference to Video', provider: 'Fal', pricing: { per_second: 0.33264, per_second_with_video_input: 0.19954 } },
  { id: 'alibaba/happy-horse/image-to-video', label: 'Happy Horse Image to Video', provider: 'Alibaba', pricing: { per_second: 0.28 } },
];

const QUICK_PROMPTS: PromptExample[] = [
  'Explain quantum computing in simple terms',
  'Write a Python function to find prime numbers',
  'Compare REST vs GraphQL APIs',
  'Create a React hook for debouncing',
];

const IMAGE_QUICK_PROMPTS: PromptExample[] = [
  'A cinematic product photo of a translucent glass pathfinder compass on a matte black desk, thin luminous routing lines inside the glass, soft window light, shallow depth of field, premium AI infrastructure product photography, no text, no logo.',
  'A beige ceramic coffee mug on a wooden table, natural light',
  'Isometric illustration of a futuristic city at sunset',
  'Photorealistic gray tabby cat hugging an otter with an orange scarf',
  'Minimal line-art logo of a mountain over a flowing river',
];

const SPEECH_QUICK_PROMPTS: PromptExample[] = [
  'Welcome to OpenPaths. Your one API key can route text, image, video, and speech models.',
  'This is a quick voice check for Grok text to speech in the OpenPaths playground.',
  'Read this in a warm, concise product-demo voice with natural pacing.',
  'The model is billed per input character, so this short sample stays inexpensive.',
];

const GEMINI_TTS_QUICK_PROMPTS: PromptExample[] = [
  {
    label: 'Fantasy RPG two-speaker scene',
    prompt: `Read the following transcript based on the audio profile and director's note.

# Audio Profile
For Speaker 1: A stern and weary gatekeeper
For Speaker 2: A determined and courageous traveler seeking answers.

# Director's note
For Speaker 1: Style: Deadpan. Pace: Natural. Accent: British (RP).
For Speaker 2: Style: Empathetic. Pace: Staccato. Accent: American (Gen).

## Scene:
A dark, crumbling dungeon with dripping water echoing in the distance.

## Sample Context:
Fantasy RPG style. Pacing is measured, snapping into urgency at the end. Tone is tense and cautious.

## Transcript:
Speaker 1: [shouting] Halt, traveler! The northern pass is sealed by order of the council.
Speaker 2: [determination] I carry a message for the elder. Step aside, or I will force my way through.
Speaker 1: [caution] No one passes. [pensive] The elder is... he's no longer receiving visitors.
Speaker 2: [suspicion] What do you mean? We don't have time for games.`,
  },
  {
    label: 'Podcast cold open',
    prompt: 'Read this as a polished documentary podcast cold open. [measured] The storm reached the harbor just after midnight. By morning, every clock in town had stopped, but one lighthouse was still flashing a signal no one could decode.',
  },
  {
    label: 'Product demo narration',
    prompt: 'Read in a confident, warm product-demo voice with clean pacing and slight emphasis on bracketed tags. [clear] OpenPaths routes text, image, video, music, and speech models through one API. [upbeat] Switch models without rewriting your app.',
  },
  {
    label: 'Audiobook suspense',
    prompt: 'Narrate with a low, intimate audiobook tone. [whispering] The hallway light flickered twice. Mira held her breath, because the second shadow on the wall did not belong to her.',
  },
];

const MUSIC_QUICK_PROMPTS: PromptExample[] = [
  {
    label: 'Peak-time EDM full song',
    prompt: `Again and Again!: Composition Breakdown
[0:00 - 0:20] Intro: Intensity: 4/10. A driving, atmospheric EDM intro with a punchy four-on-the-floor kick, crisp sixteenth-note hi-hats, and a muffled rhythmic synth pluck that brightens as a low-pass filter opens. No vocals. Minor key, repetitive two-bar loop, expectant underground club atmosphere.
[0:20 - 0:50] Verse: Intensity: 6/10. Lyrics: "Over and over, the feeling is new / Over and over, it's taking its cue / Over and over, the rhythm is true / Over and over, I'm finding it here." Hypnotic Tech House groove with deep side-chained sub-bass, snapping claps on beats two and four, metallic woodblock-like synth syncopation, and a bright crystalline female soprano chanting rhythmically.
[0:50 - 1:30] Chorus: Intensity: 9/10. Lyrics: "Again and again! / Again and again! / Again and again! / Again and again!" Massive Big Room EDM drop with distorted kick, thunderous sub-bass, rapid snare rolls, crashing open hi-hats, supersaw anthem chords, heavy sidechain pumping, wide choral vocal layers, and euphoric festival energy.
[1:30 - 2:00] Verse: Intensity: 6/10. Lyrics: "Over and over, the focus is clear / Over and over, I'm holding it near / Over and over, the sound is the cure / Over and over, it's making me pure." Return to tight Tech House percussion, digital shakers, rim-shots, metallic staccato synth lead, precise rhythmic vocal cadence, and relentless groove.
[2:00 - 2:40] Chorus: Intensity: 10/10. Lyrics: "Again and again! / Again and again! / Again and again! / Again and again!" Final full-frequency Big Room drop with saturated low end, double-time snare hits, triumphant high-register supersaws, powerful belted female vocal, and total peak-time immersion.
[2:40 - 3:04] Outro: Intensity: 4/10. Lyrics: "Again and again. / Again and again." Gradual subtraction of melodic layers, steady kick and hi-hats, fading synth echoes, filtered sub-bass, sparse dry whispered vocal repetitions, clean DJ-friendly mix-out.`,
  },
  {
    label: 'Lo-fi house night drive',
    prompt: 'A 90-second atmospheric lo-fi house track with warm vinyl texture, soft side-chained pads, a round kick, brushed percussion, dusty tape wobble, and a mellow Rhodes hook. Structure: 8-bar filtered intro, 16-bar groove, 8-bar breakdown with rain ambience, final 16-bar groove with a subtle vocal chop.',
  },
  {
    label: 'Cinematic trailer cue',
    prompt: 'A cinematic orchestral trailer cue that starts with quiet piano pulses and low cello drones, builds with staccato strings and taiko drums, then resolves into a bright brass theme. Use three clear acts: suspenseful intro, escalating midsection, heroic final climax with choir and cymbal swells.',
  },
  {
    label: 'Cyberpunk drum and bass',
    prompt: 'A dark cyberpunk drum and bass track at 174 BPM with razor-sharp breakbeats, reese bass, glitch percussion, metallic impacts, and distant vocoder phrases. Start with neon ambience, drop into a rolling bassline, add a half-time breakdown, then return with denser drums and distorted synth stabs.',
  },
  {
    label: 'Indie folk duet',
    prompt: 'A sparse acoustic indie folk song with fingerpicked guitar, intimate male and female harmony vocals, subtle cello swells, brushed snare, and a bittersweet chorus about coming home after years away. Keep the arrangement organic and close-mic, with a small lift in the final chorus.',
  },
  {
    label: 'Latin pop summer hook',
    prompt: 'A bright Latin pop song with nylon guitar, reggaeton-inspired dembow percussion, warm bass, hand claps, and a catchy bilingual chorus. Female lead vocal, confident and sunny. Build from a guitar intro into a danceable chorus, include a short percussion break, then finish with layered ad-libs.',
  },
  {
    label: 'Synthwave credits theme',
    prompt: 'A nostalgic 1980s synthwave end-credits theme with gated drums, pulsing analog bass, shimmering Juno-style pads, and a soaring lead melody. Medium tempo, cinematic and bittersweet, with a two-part structure: restrained first half, wide emotional final refrain.',
  },
  {
    label: 'Afro house sunrise',
    prompt: 'A warm Afro house track with organic hand percussion, deep kick, marimba-like plucks, airy vocal chants, and a rolling bass groove. Start minimal and sunrise-like, introduce call-and-response vocal fragments, then open into a spacious melodic drop with polyrhythmic percussion.',
  },
];

function promptExampleLabel(example: PromptExample): string {
  return typeof example === 'string' ? example : example.label;
}

function promptExampleText(example: PromptExample): string {
  return typeof example === 'string' ? example : example.prompt;
}

interface ImageDemo {
  prompt: string;
  outputUrl: string;
  size: typeof IMAGE_SIZES[number];
  imageUrl?: string;
  imageSize?: string;
  numInferenceSteps?: number;
  guidanceScale?: number;
  outputFormat?: 'jpeg' | 'png' | 'webp';
  safetyChecker?: boolean;
  outpaint?: {
    expandTop: number;
    expandBottom: number;
    expandLeft: number;
    expandRight: number;
    outputFormat: 'jpeg' | 'png';
  };
}

const IMAGE_DEMOS: Record<string, ImageDemo> = {
  'hidream-o1-image-dev': {
    prompt: 'A cinematic product photo of a translucent glass pathfinder compass on a matte black desk, thin luminous routing lines inside the glass, soft window light, shallow depth of field, premium AI infrastructure product photography, no text, no logo.',
    outputUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/hidream/hidream-o1-image-dev-demo.png',
    size: '1024x1024',
  },
  'fal-ai/hidream-o1-image/edit': {
    prompt: 'Replace the perfume bottle with a lipstick',
    imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/hidream-edit/perfume.jpg',
    outputUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/hidream-edit/lipstick.png',
    size: '1360x768',
    imageSize: 'landscape_16_9',
    numInferenceSteps: 50,
    guidanceScale: 5,
    outputFormat: 'png',
    safetyChecker: false,
  },
  'fal-ai/flux-2-pro/outpaint': {
    prompt: 'Expand this image naturally beyond the frame.',
    imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/flux-outpaint/input.png',
    outputUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/flux-outpaint/output.jpg',
    size: '1024x1024',
    outpaint: {
      expandTop: 0,
      expandBottom: 200,
      expandLeft: 200,
      expandRight: 200,
      outputFormat: 'jpeg',
    },
  },
};

const MODELS_CACHE_KEY = 'op_models_cache_v1';
const MODELS_CACHE_TTL_MS = 60 * 60 * 1000; // 1 hour
const PANE_HISTORY_PREFIX = 'op_pg_pane_';
// Models excluded from the chat selector. Image models remain available since
// the playground now routes them to /v1/images/generations automatically.
const NON_CHAT_PATTERNS = /^(whisper|xai-stt|grok-voice|text-embedding|openpaths-embed|modernbert|mistral-embed|codestral-embed|nemotron-embed|gemini-embedding-001|gemini-embedding-2-preview|gemini-embedding-2|gpt-4o-transcribe|gpt-4o-mini-transcribe|distil-whisper|whisper-v3)/i;
const IMAGE_MODEL_PATTERNS = /^(openpaths\/auto-image|auto-image|flux|klein|ra1|zimage|glm-image|grok-imagine-image|gpt-image|fal-gpt-image|hidream|dall-e|stable-diffusion|sd3|ideogram|fal-ai\/flux-2-pro\/outpaint)/i;
const VIDEO_MODEL_PATTERNS = /^(auto-video|wan|ltx|hailuo|kling|luma|ra2v|sora|seedance)/i;
const SPEECH_MODEL_PATTERNS = /(tts|speech-)/i;
const MUSIC_MODEL_PATTERNS = /^(music-|lyria-)/i;

function isImageModel(m: CatalogModel | undefined): boolean {
  if (!m) return false;
  if (MUSIC_MODEL_PATTERNS.test(m.id)) return false;
  if (m.pricing?.per_image && m.pricing.per_image > 0) return true;
  if (m.pricing?.per_megapixel && m.pricing.per_megapixel > 0) return true;
  if (m.pricing?.first_megapixel && m.pricing.first_megapixel > 0) return true;
  return IMAGE_MODEL_PATTERNS.test(m.id);
}

function isVideoModel(m: CatalogModel | undefined): boolean {
  if (!m) return false;
  if ((m.pricing?.per_video && m.pricing.per_video > 0) || (m.pricing?.per_second && m.pricing.per_second > 0)) return true;
  return VIDEO_MODEL_PATTERNS.test(m.id);
}

function isSpeechModel(m: CatalogModel | undefined): boolean {
  return !!m && SPEECH_MODEL_PATTERNS.test(m.id);
}

function isGeminiSpeechModel(m: CatalogModel | undefined): boolean {
  return !!m && /gemini.*tts/i.test(m.id);
}

function hasTwoSpeakerTranscript(text: string): boolean {
  return /Speaker\s*1\s*:/i.test(text) && /Speaker\s*2\s*:/i.test(text);
}

function geminiVoiceInfo(name: string) {
  return GEMINI_TTS_VOICE_INFO.find(v => v.name.toLowerCase() === name.toLowerCase());
}

function isMusicModel(m: CatalogModel | undefined): boolean {
  return !!m && MUSIC_MODEL_PATTERNS.test(m.id);
}

function isImageToVideoModel(m: CatalogModel | undefined): boolean {
  return !!m && /image-to-video|i2v/i.test(m.id);
}

function isReferenceToVideoModel(m: CatalogModel | undefined): boolean {
  return !!m && /reference-to-video|reference/i.test(m.id);
}

function isHappyHorseVideoModel(m: CatalogModel | undefined): boolean {
  return m?.id === 'alibaba/happy-horse/image-to-video';
}

function isOutpaintImageModel(m: CatalogModel | undefined): boolean {
  return m?.id === 'fal-ai/flux-2-pro/outpaint';
}

function parseImageInputUrls(value: string): string[] {
  return value
    .split(/[\n,]+/)
    .map(v => v.trim())
    .filter(Boolean);
}

function normalizeUploadUrlText(value: string): string {
  return value
    .split('\n')
    .map(line => normalizeUploadedAssetUrl(line))
    .join('\n');
}

function ImagePreviewStrip({ urls, label }: { urls: string[]; label: string }) {
  if (urls.length === 0) return null;
  return (
    <div className="mt-2 flex gap-2 overflow-x-auto pb-1" data-testid={`preview-${label}`}>
      {urls.slice(0, 6).map((url, idx) => (
        <a key={`${url}-${idx}`} href={url} target="_blank" rel="noreferrer" className="group relative h-16 w-16 flex-none overflow-hidden rounded border border-white/10 bg-black">
          <img src={url} alt={`${label} ${idx + 1}`} className="h-full w-full object-cover transition-opacity group-hover:opacity-80" loading="lazy" />
        </a>
      ))}
    </div>
  );
}

function SingleImagePreview({ url, label }: { url: string; label: string }) {
  const urls = parseImageInputUrls(url);
  return <ImagePreviewStrip urls={urls.slice(0, 1)} label={label} />;
}

function resizeTextarea(el: HTMLTextAreaElement) {
  el.style.height = 'auto';
  const next = Math.min(el.scrollHeight, 200);
  el.style.height = `${next}px`;
  el.style.overflowY = el.scrollHeight > 200 ? 'auto' : 'hidden';
}

let paneCounter = 0;
function makePane(modelId: string): ModelPane {
  return {
    id: `pane-${++paneCounter}`,
    modelId,
    messages: [],
    streaming: false,
    error: null,
    latencyMs: null,
    tokensUsed: null,
    promptTokens: null,
    completionTokens: null,
    costUsd: null,
  };
}

function loadPaneHistory(modelId: string): Message[] {
  try {
    const raw = localStorage.getItem(PANE_HISTORY_PREFIX + modelId);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (Array.isArray(parsed)) return parsed;
  } catch {}
  return [];
}

function savePaneHistory(modelId: string, messages: Message[]) {
  try {
    if (messages.length === 0) {
      localStorage.removeItem(PANE_HISTORY_PREFIX + modelId);
    } else {
      localStorage.setItem(PANE_HISTORY_PREFIX + modelId, JSON.stringify(messages));
    }
  } catch {}
}

function loadCachedModels(): CatalogModel[] | null {
  try {
    const raw = localStorage.getItem(MODELS_CACHE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    if (parsed && typeof parsed === 'object' && Array.isArray(parsed.models) && Date.now() - parsed.ts < MODELS_CACHE_TTL_MS) {
      return parsed.models as CatalogModel[];
    }
  } catch {}
  return null;
}

function saveCachedModels(models: CatalogModel[]) {
  try {
    localStorage.setItem(MODELS_CACHE_KEY, JSON.stringify({ ts: Date.now(), models }));
  } catch {}
}

function humanProvider(ownedBy: string): string {
  const p = (ownedBy || '').toLowerCase();
  const map: Record<string, string> = {
    openai: 'OpenAI', anthropic: 'Anthropic', google: 'Google', xai: 'xAI',
    deepseek: 'DeepSeek', mistral: 'Mistral', groq: 'Groq', together: 'Together',
    fireworks: 'Fireworks', minimax: 'MiniMax', zai: 'Z.AI', nous: 'Nous',
    openrouter: 'OpenRouter', fal: 'fal', netwrck: 'Netwrck', gobed: 'GoBed',
    openpaths: 'OpenPaths',
  };
  if (map[p]) return map[p];
  if (!p) return 'Other';
  return p.charAt(0).toUpperCase() + p.slice(1);
}

function fmtCost(usd: number): string {
  if (usd < 0.0001) return '<$0.0001';
  if (usd < 0.01) return `$${usd.toFixed(4)}`;
  if (usd < 1) return `$${usd.toFixed(3)}`;
  return `$${usd.toFixed(2)}`;
}

function estimateCost(model: CatalogModel | undefined, promptTok: number, completionTok: number): number | null {
  const p = model?.pricing;
  if (!p || (!p.input_per_1m_tokens && !p.output_per_1m_tokens)) return null;
  const inCost = (p.input_per_1m_tokens || 0) * promptTok / 1_000_000;
  const outCost = (p.output_per_1m_tokens || 0) * completionTok / 1_000_000;
  return inCost + outCost;
}

function fmtBalance(cents: number): string {
  const usd = cents / 100;
  if (usd >= 1) return `$${usd.toFixed(2)}`;
  if (usd > 0) return `$${usd.toFixed(4)}`;
  return '$0.00';
}

// --- Minimal markdown renderer ---

function renderMarkdown(text: string): React.ReactNode[] {
  const nodes: React.ReactNode[] = [];
  const lines = text.split('\n');
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];

    // Fenced code block
    if (line.startsWith('```')) {
      const lang = line.slice(3).trim();
      const codeLines: string[] = [];
      i++;
      while (i < lines.length && !lines[i].startsWith('```')) {
        codeLines.push(lines[i]);
        i++;
      }
      i++; // skip closing ```
      const code = codeLines.join('\n');
      nodes.push(<MarkdownCodeBlock key={nodes.length} code={code} lang={lang} />);
      continue;
    }

    // Heading
    const headingMatch = line.match(/^(#{1,3})\s+(.+)/);
    if (headingMatch) {
      const level = headingMatch[1].length;
      const text = headingMatch[2];
      const cls = level === 1 ? 'text-lg font-bold mt-4 mb-2' : level === 2 ? 'text-base font-bold mt-3 mb-1.5' : 'text-sm font-bold mt-2 mb-1';
      nodes.push(<div key={nodes.length} className={cls}>{renderInline(text)}</div>);
      i++;
      continue;
    }

    // Bullet list
    if (line.match(/^[\s]*[-*]\s/)) {
      const items: string[] = [];
      while (i < lines.length && lines[i].match(/^[\s]*[-*]\s/)) {
        items.push(lines[i].replace(/^[\s]*[-*]\s/, ''));
        i++;
      }
      nodes.push(
        <ul key={nodes.length} className="list-disc list-inside space-y-0.5 my-1">
          {items.map((item, j) => <li key={j} className="text-sm leading-relaxed">{renderInline(item)}</li>)}
        </ul>
      );
      continue;
    }

    // Numbered list
    if (line.match(/^[\s]*\d+\.\s/)) {
      const items: string[] = [];
      while (i < lines.length && lines[i].match(/^[\s]*\d+\.\s/)) {
        items.push(lines[i].replace(/^[\s]*\d+\.\s/, ''));
        i++;
      }
      nodes.push(
        <ol key={nodes.length} className="list-decimal list-inside space-y-0.5 my-1">
          {items.map((item, j) => <li key={j} className="text-sm leading-relaxed">{renderInline(item)}</li>)}
        </ol>
      );
      continue;
    }

    // Empty line
    if (line.trim() === '') {
      i++;
      continue;
    }

    // Regular paragraph - collect consecutive non-empty lines
    const paraLines: string[] = [];
    while (i < lines.length && lines[i].trim() !== '' && !lines[i].startsWith('```') && !lines[i].match(/^#{1,3}\s/) && !lines[i].match(/^[\s]*[-*]\s/) && !lines[i].match(/^[\s]*\d+\.\s/)) {
      paraLines.push(lines[i]);
      i++;
    }
    nodes.push(
      <p key={nodes.length} className="text-sm leading-relaxed my-1">
        {renderInline(paraLines.join('\n'))}
      </p>
    );
  }

  return nodes;
}

function renderInline(text: string): React.ReactNode[] {
  const parts: React.ReactNode[] = [];
  // Match: **bold**, `code`, *italic*
  const regex = /(\*\*(.+?)\*\*|`([^`]+)`|\*(.+?)\*)/g;
  let lastIndex = 0;
  let match;

  while ((match = regex.exec(text)) !== null) {
    if (match.index > lastIndex) {
      parts.push(text.slice(lastIndex, match.index));
    }
    if (match[2]) {
      parts.push(<strong key={parts.length} className="font-semibold">{match[2]}</strong>);
    } else if (match[3]) {
      parts.push(<code key={parts.length} className="bg-white/10 px-1.5 py-0.5 rounded text-[13px] font-mono">{match[3]}</code>);
    } else if (match[4]) {
      parts.push(<em key={parts.length}>{match[4]}</em>);
    }
    lastIndex = match.index + match[0].length;
  }

  if (lastIndex < text.length) {
    parts.push(text.slice(lastIndex));
  }

  return parts.length > 0 ? parts : [text];
}

function MarkdownCodeBlock({ code, lang }: { code: string; lang: string; key?: React.Key }) {
  const [copied, setCopied] = useState(false);

  const copy = () => {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="my-2 rounded-lg border border-white/10 overflow-hidden bg-white/[0.03]">
      <div className="flex items-center justify-between px-3 py-1.5 border-b border-white/10 bg-white/[0.02]">
        <span className="text-[10px] font-mono text-white/30 uppercase">{lang || 'code'}</span>
        <button onClick={copy} className="text-white/30 hover:text-white/60 transition-colors p-0.5">
          {copied ? <Check className="w-3 h-3 text-green-400" /> : <Copy className="w-3 h-3" />}
        </button>
      </div>
      <HighlightedCodeBlock
        code={code}
        language={lang}
        hideLabel
        preClassName="p-3 overflow-x-auto text-[13px] font-mono leading-relaxed text-white/80"
      />
    </div>
  );
}

// --- Main component ---

export function Playground() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [apiKey, setApiKey] = useState(() => localStorage.getItem('op_api_key') || '');
  const [systemPrompt, setSystemPrompt] = useState('You are a helpful assistant.');
  const [temperature, setTemperature] = useState(0.7);
  const [maxTokens, setMaxTokens] = useState(4096);
  const [showSettings, setShowSettings] = useState(false);
  const [showCode, setShowCode] = useState(false);
  const [codeLang, setCodeLang] = useState<CodeLanguage>('python');
  const [codeCopied, setCodeCopied] = useState(false);
  const [input, setInput] = useState('');
  const [shareCopied, setShareCopied] = useState(false);
  const [balanceCents, setBalanceCents] = useState<number | null>(null);
  const [imageSize, setImageSize] = useState<typeof IMAGE_SIZES[number]>('1024x1024');
  const [imageQuality, setImageQuality] = useState<typeof IMAGE_QUALITIES[number]>('standard');
  const [imageCount, setImageCount] = useState(1);
  const [imageResponseFormat, setImageResponseFormat] = useState<typeof IMAGE_RESPONSE_FORMATS[number]>('url');
  const [imageAspectRatio, setImageAspectRatio] = useState<typeof IMAGE_ASPECT_RATIOS[number]>('auto');
  const [imageInputUrls, setImageInputUrls] = useState('');
  const [outpaintTop, setOutpaintTop] = useState(0);
  const [outpaintBottom, setOutpaintBottom] = useState(200);
  const [outpaintLeft, setOutpaintLeft] = useState(200);
  const [outpaintRight, setOutpaintRight] = useState(200);
  const [videoResolution, setVideoResolution] = useState<typeof VIDEO_RESOLUTIONS[number]>('720p');
  const [videoDuration, setVideoDuration] = useState<typeof VIDEO_DURATIONS[number]>('10');
  const [videoAspectRatio, setVideoAspectRatio] = useState<typeof VIDEO_ASPECT_RATIOS[number]>('16:9');
  const [videoGenerateAudio, setVideoGenerateAudio] = useState(true);
  const [videoImageUrl, setVideoImageUrl] = useState('');
  const [videoEndImageUrl, setVideoEndImageUrl] = useState('');
  const [videoImageUrls, setVideoImageUrls] = useState('');
  const [videoVideoUrls, setVideoVideoUrls] = useState('');
  const [videoAudioUrls, setVideoAudioUrls] = useState('');
  const [speechVoice, setSpeechVoice] = useState('eve');
  const [speechLanguage, setSpeechLanguage] = useState<typeof SPEECH_LANGUAGES[number]>('en');
  const [ttsSpeaker1Profile, setTtsSpeaker1Profile] = useState('A stern and weary gatekeeper');
  const [ttsSpeaker2Profile, setTtsSpeaker2Profile] = useState('A determined and courageous traveler seeking answers.');
  const [ttsSpeaker1Style, setTtsSpeaker1Style] = useState<typeof TTS_STYLES[number]>('Deadpan');
  const [ttsSpeaker2Style, setTtsSpeaker2Style] = useState<typeof TTS_STYLES[number]>('Empathetic');
  const [ttsSpeaker1Pace, setTtsSpeaker1Pace] = useState<typeof TTS_PACES[number]>('Natural');
  const [ttsSpeaker2Pace, setTtsSpeaker2Pace] = useState<typeof TTS_PACES[number]>('Staccato');
  const [ttsSpeaker1Accent, setTtsSpeaker1Accent] = useState<typeof TTS_ACCENTS[number]>('British (RP)');
  const [ttsSpeaker2Accent, setTtsSpeaker2Accent] = useState<typeof TTS_ACCENTS[number]>('American (Gen)');
  const [ttsSpeaker1Voice, setTtsSpeaker1Voice] = useState('Fenrir');
  const [ttsSpeaker2Voice, setTtsSpeaker2Voice] = useState('Puck');
  const [speechAutoEmotion, setSpeechAutoEmotion] = useState(false);
  const [uploadingRefs, setUploadingRefs] = useState(false);
  const [dragTarget, setDragTarget] = useState<ReferenceUploadTarget | null>(null);
  const [dynamicModels, setDynamicModels] = useState<CatalogModel[] | null>(() => loadCachedModels());
  const [panes, setPanes] = useState<ModelPane[]>(() => {
    const modelParam = searchParams.get('model');
    const initial = makePane(modelParam || 'auto');
    initial.messages = loadPaneHistory(initial.modelId);
    return [initial];
  });
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const abortRefs = useRef<Map<string, AbortController>>(new Map());
  const autoRanRef = useRef(false);
  const lastAppliedVideoDemoPromptRef = useRef('');
  const lastAppliedImageDemoPromptRef = useRef('');

  const catalog: CatalogModel[] = dynamicModels || FALLBACK_MODELS;
  const chatCatalog = useMemo(() => catalog.filter(m => !NON_CHAT_PATTERNS.test(m.id)), [catalog]);
  const modelIndex = useMemo(() => {
    const m = new Map<string, CatalogModel>();
    for (const mod of catalog) m.set(mod.id, mod);
    return m;
  }, [catalog]);

  const anyStreaming = panes.some(p => p.streaming);
  const primaryModel = modelIndex.get(panes[0]?.modelId || '');
  const primaryIsImage = isImageModel(primaryModel) || IMAGE_MODEL_PATTERNS.test(panes[0]?.modelId || '');
  const primaryIsVideo = isVideoModel(primaryModel) || VIDEO_MODEL_PATTERNS.test(panes[0]?.modelId || '');
  const primaryIsSpeech = isSpeechModel(primaryModel) || SPEECH_MODEL_PATTERNS.test(panes[0]?.modelId || '');
  const primaryIsGeminiSpeech = isGeminiSpeechModel(primaryModel);
  const primaryIsMusic = isMusicModel(primaryModel) || MUSIC_MODEL_PATTERNS.test(panes[0]?.modelId || '');
  const primaryIsImageToVideo = isImageToVideoModel(primaryModel);
  const primaryIsReferenceToVideo = isReferenceToVideoModel(primaryModel);
  const primaryIsHappyHorseVideo = isHappyHorseVideoModel(primaryModel);
  const primaryIsOutpaintImage = isOutpaintImageModel(primaryModel) || panes[0]?.modelId === 'fal-ai/flux-2-pro/outpaint';
  const primaryImageDemo = IMAGE_DEMOS[panes[0]?.modelId || ''];
  const primaryVideoDemo = VIDEO_DEMOS[panes[0]?.modelId || ''];

  useEffect(() => {
    const t = setTimeout(() => inputRef.current?.focus(), 100);
    return () => clearTimeout(t);
  }, []);

  useEffect(() => {
    if (inputRef.current) resizeTextarea(inputRef.current);
  }, [input, primaryIsImage, primaryIsSpeech, primaryIsVideo]);

  useEffect(() => {
    if (apiKey) {
      localStorage.setItem('op_api_key', apiKey);
    }
  }, [apiKey]);

  useEffect(() => {
    if (!primaryImageDemo) return;
    setImageSize(primaryImageDemo.size);
    if (primaryImageDemo.imageUrl) setImageInputUrls(primaryImageDemo.imageUrl);
    if (primaryImageDemo.outpaint) {
      setOutpaintTop(primaryImageDemo.outpaint.expandTop);
      setOutpaintBottom(primaryImageDemo.outpaint.expandBottom);
      setOutpaintLeft(primaryImageDemo.outpaint.expandLeft);
      setOutpaintRight(primaryImageDemo.outpaint.expandRight);
    }
    setInput(prev => {
      if (prev.trim() && prev !== lastAppliedImageDemoPromptRef.current) return prev;
      lastAppliedImageDemoPromptRef.current = primaryImageDemo.prompt;
      return primaryImageDemo.prompt;
    });
  }, [primaryImageDemo]);

  useEffect(() => {
    if (!primaryVideoDemo) return;
    setVideoResolution(primaryVideoDemo.resolution);
    setVideoDuration(primaryVideoDemo.duration);
    setVideoAspectRatio(primaryVideoDemo.aspectRatio);
    setVideoGenerateAudio(primaryVideoDemo.generateAudio);
    setVideoImageUrl(primaryVideoDemo.imageUrl || '');
    setVideoEndImageUrl(primaryVideoDemo.endImageUrl || '');
    setVideoImageUrls((primaryVideoDemo.imageUrls || []).join('\n'));
    setVideoVideoUrls((primaryVideoDemo.videoUrls || []).join('\n'));
    setVideoAudioUrls((primaryVideoDemo.audioUrls || []).join('\n'));
    setInput(prev => {
      if (prev.trim() && prev !== lastAppliedVideoDemoPromptRef.current) return prev;
      lastAppliedVideoDemoPromptRef.current = primaryVideoDemo.prompt;
      return primaryVideoDemo.prompt;
    });
  }, [primaryVideoDemo]);

  useEffect(() => {
    if (primaryIsGeminiSpeech && !GEMINI_TTS_VOICES.includes(speechVoice)) {
      setSpeechVoice('Puck');
    } else if (!primaryIsGeminiSpeech && !SPEECH_VOICES.includes(speechVoice)) {
      setSpeechVoice('eve');
    }
  }, [primaryIsGeminiSpeech, speechVoice]);

  const buildGeminiTTSPrompt = useCallback((text: string) => {
    if (/#\s*Audio Profile/i.test(text) || /#\s*Director's note/i.test(text)) return text;
    if (!hasTwoSpeakerTranscript(text)) return text;
    return `Read the following transcript based on the audio profile and director's note.

# Audio Profile
For Speaker 1: ${ttsSpeaker1Profile}
For Speaker 2: ${ttsSpeaker2Profile}

# Director's note
For Speaker 1: Style: ${ttsSpeaker1Style}. Pace: ${ttsSpeaker1Pace}. Accent: ${ttsSpeaker1Accent}.
For Speaker 2: Style: ${ttsSpeaker2Style}. Pace: ${ttsSpeaker2Pace}. Accent: ${ttsSpeaker2Accent}.

## Transcript:
${text}`;
  }, [ttsSpeaker1Accent, ttsSpeaker1Pace, ttsSpeaker1Profile, ttsSpeaker1Style, ttsSpeaker2Accent, ttsSpeaker2Pace, ttsSpeaker2Profile, ttsSpeaker2Style]);

  const currentGeminiSpeakerVoices = useCallback((text: string) => {
    if (!hasTwoSpeakerTranscript(text)) return undefined;
    return [
      { speaker: 'Speaker 1', voice: ttsSpeaker1Voice },
      { speaker: 'Speaker 2', voice: ttsSpeaker2Voice },
    ];
  }, [ttsSpeaker1Voice, ttsSpeaker2Voice]);

  // Fetch model catalog from /v1/models (works with or without API key — endpoint is auth-gated).
  useEffect(() => {
    if (!apiKey) return;
    let aborted = false;
    (async () => {
      try {
        const resp = await fetch(`${window.location.origin}/v1/models`, {
          headers: { 'Authorization': `Bearer ${apiKey}` },
        });
        if (!resp.ok) return;
        const data = await resp.json();
        if (aborted || !Array.isArray(data.data)) return;
        const mapped: CatalogModel[] = data.data.map((m: any) => ({
          id: m.id,
          label: m.id,
          provider: humanProvider(m.owned_by),
          pricing: m.pricing,
          capabilities: m.capabilities,
        }));
        // Merge with fallback so curated labels survive for built-in ids.
        const byId = new Map<string, CatalogModel>();
        for (const f of FALLBACK_MODELS) byId.set(f.id, f);
        for (const m of mapped) {
          const existing = byId.get(m.id);
          byId.set(m.id, {
            ...m,
            label: existing?.label || m.label,
            provider: existing?.provider || m.provider,
          });
        }
        const merged = Array.from(byId.values());
        setDynamicModels(merged);
        saveCachedModels(merged);
      } catch {}
    })();
    return () => { aborted = true; };
  }, [apiKey]);

  // Fetch account balance when signed in.
  const refreshBalance = useCallback(async () => {
    if (!apiKey) return;
    try {
      const resp = await fetch(`${window.location.origin}/account/balance`, {
        headers: { 'Authorization': `Bearer ${apiKey}` },
      });
      if (!resp.ok) return;
      const d = await resp.json();
      if (typeof d.balance_cents === 'number') setBalanceCents(d.balance_cents);
      else if (typeof d.balance_usd === 'number') setBalanceCents(Math.round(d.balance_usd * 100));
    } catch {}
  }, [apiKey]);

  useEffect(() => { refreshBalance(); }, [refreshBalance]);

  const baseUrl = window.location.origin;

  const uploadReferenceFiles = useCallback(async (files: FileList | File[], target: ReferenceUploadTarget) => {
    if (!apiKey || files.length === 0) return;
    setUploadingRefs(true);
    try {
      const urls: string[] = [];
      for (const file of Array.from(files)) {
        const uploadFile = await prepareUploadFile(file);
        const form = new FormData();
        form.append('file', uploadFile);
        const resp = await fetch(`${baseUrl}/v1/files/upload`, {
          method: 'POST',
          headers: { 'Authorization': `Bearer ${apiKey}` },
          body: form,
        });
        if (!resp.ok) throw new Error(await resp.text());
        const data = await resp.json();
        if (data.url) urls.push(normalizeUploadedAssetUrl(data.url));
      }
      const append = (prev: string) => [prev.trim(), ...urls].filter(Boolean).join('\n');
      if (target === 'image') setVideoImageUrl(urls[0] || '');
      if (target === 'end-image') setVideoEndImageUrl(urls[0] || '');
      if (target === 'images') setVideoImageUrls(append);
      if (target === 'video') setVideoVideoUrls(append);
      if (target === 'audio') setVideoAudioUrls(append);
    } finally {
      setUploadingRefs(false);
    }
  }, [apiKey, baseUrl]);

  const referenceDropHandlers = useCallback((target: ReferenceUploadTarget) => ({
    onDragEnter: (event: React.DragEvent<HTMLElement>) => {
      event.preventDefault();
      event.stopPropagation();
      if (!uploadingRefs) setDragTarget(target);
    },
    onDragOver: (event: React.DragEvent<HTMLElement>) => {
      event.preventDefault();
      event.stopPropagation();
      event.dataTransfer.dropEffect = uploadingRefs ? 'none' : 'copy';
      if (!uploadingRefs) setDragTarget(target);
    },
    onDragLeave: (event: React.DragEvent<HTMLElement>) => {
      event.preventDefault();
      event.stopPropagation();
      if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
        setDragTarget(current => current === target ? null : current);
      }
    },
    onDrop: (event: React.DragEvent<HTMLElement>) => {
      event.preventDefault();
      event.stopPropagation();
      setDragTarget(null);
      const files = Array.from<File>(event.dataTransfer.files).filter(file => file.type.startsWith('image/'));
      if (!uploadingRefs && files.length > 0) {
        void uploadReferenceFiles(files, target);
      }
    },
  }), [uploadReferenceFiles, uploadingRefs]);

  useEffect(() => {
    if (!primaryIsImageToVideo || !apiKey) return;
    const handlePaste = (event: ClipboardEvent) => {
      if (uploadingRefs) return;
      const files = Array.from(event.clipboardData?.files || []).filter(file => file.type.startsWith('image/'));
      if (files.length === 0) return;
      event.preventDefault();
      void uploadReferenceFiles(files.slice(0, 1), 'image');
    };
    window.addEventListener('paste', handlePaste);
    return () => window.removeEventListener('paste', handlePaste);
  }, [apiKey, primaryIsImageToVideo, uploadReferenceFiles, uploadingRefs]);

  const sendToImageModel = useCallback(async (paneId: string, modelId: string, prompt: string) => {
    const controller = new AbortController();
    abortRefs.current.set(paneId, controller);
    const start = performance.now();

    setPanes(prev => prev.map(p =>
      p.id === paneId ? { ...p, streaming: true, error: null, latencyMs: null, tokensUsed: null, promptTokens: null, completionTokens: null, costUsd: null } : p
    ));

    try {
      const inputUrls = parseImageInputUrls(imageInputUrls);
      const isOutpaint = modelId === 'fal-ai/flux-2-pro/outpaint';
      const imageDemo = IMAGE_DEMOS[modelId];
      const isEdit = inputUrls.length > 0 && !isOutpaint;
      const body: Record<string, unknown> = {
        model: modelId,
        prompt,
        n: imageCount,
        size: imageSize,
        quality: imageQuality,
        response_format: imageResponseFormat,
      };
      if (isEdit) {
        body.images = inputUrls.map(url => ({ type: 'image_url', url }));
        body.reference_image_urls = inputUrls;
        body.aspect_ratio = imageAspectRatio;
      }
      if (imageDemo?.imageSize) body.image_size = imageDemo.imageSize;
      if (imageDemo?.numInferenceSteps) body.num_inference_steps = imageDemo.numInferenceSteps;
      if (imageDemo?.guidanceScale !== undefined) body.guidance_scale = imageDemo.guidanceScale;
      if (imageDemo?.outputFormat) body.output_format = imageDemo.outputFormat;
      if (imageDemo?.safetyChecker !== undefined) body.enable_safety_checker = imageDemo.safetyChecker;
      if (imageDemo && isEdit) body.num_images = imageCount;
      if (isOutpaint) {
        body.image_url = inputUrls[0] || IMAGE_DEMOS['fal-ai/flux-2-pro/outpaint'].imageUrl;
        body.expand_top = outpaintTop;
        body.expand_bottom = outpaintBottom;
        body.expand_left = outpaintLeft;
        body.expand_right = outpaintRight;
        body.enable_safety_checker = true;
        body.output_format = 'jpeg';
        delete body.n;
        delete body.size;
        delete body.quality;
        delete body.response_format;
      }

      const resp = await fetch(`${baseUrl}/v1/images/${isEdit ? 'edits' : 'generations'}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${apiKey}`,
        },
        body: JSON.stringify(body),
        signal: controller.signal,
      });
      if (!resp.ok) {
        const errText = await resp.text();
        throw new Error(`${resp.status}: ${errText.slice(0, 200)}`);
      }
      const data = await resp.json();
      const first = data?.data?.[0];
      if (!first) throw new Error('No image returned');
      const latency = Math.round(performance.now() - start);
      const modelCfg = modelIndex.get(modelId);
      const requestedSize = imageSize.match(/^(\d+)x(\d+)$/);
      const megapixels = requestedSize ? (Number(requestedSize[1]) * Number(requestedSize[2])) / 1_000_000 : 0.512 * 0.512;
      const cost = modelCfg?.pricing?.first_megapixel && modelCfg?.pricing?.extra_megapixel
        ? modelCfg.pricing.first_megapixel + (modelCfg.pricing.extra_megapixel * 2)
        : modelCfg?.pricing?.per_image
        ? modelCfg.pricing.per_image * Math.max(1, imageCount)
        : modelCfg?.pricing?.per_megapixel
          ? modelCfg.pricing.per_megapixel * megapixels * Math.max(1, imageCount)
          : null;

      setPanes(prev => prev.map(p => {
        if (p.id !== paneId) return p;
        const msgs = [...p.messages];
        msgs.push({
          role: 'assistant',
          content: first.revised_prompt || '',
          imageB64: first.b64_json,
          imageUrl: first.url,
        });
        savePaneHistory(modelId, msgs);
        return {
          ...p,
          messages: msgs,
          streaming: false,
          latencyMs: latency,
          tokensUsed: null,
          promptTokens: null,
          completionTokens: null,
          costUsd: cost,
        };
      }));
      refreshBalance();
    } catch (err: any) {
      if (err.name === 'AbortError') {
        setPanes(prev => prev.map(p => p.id === paneId ? { ...p, streaming: false } : p));
        return;
      }
      setPanes(prev => prev.map(p =>
        p.id === paneId ? { ...p, streaming: false, error: err.message } : p
      ));
    } finally {
      abortRefs.current.delete(paneId);
    }
  }, [apiKey, baseUrl, imageAspectRatio, imageCount, imageInputUrls, imageQuality, imageResponseFormat, imageSize, modelIndex, outpaintBottom, outpaintLeft, outpaintRight, outpaintTop, refreshBalance]);

  const sendToVideoModel = useCallback(async (paneId: string, modelId: string, prompt: string) => {
    const controller = new AbortController();
    abortRefs.current.set(paneId, controller);
    const start = performance.now();

    setPanes(prev => prev.map(p =>
      p.id === paneId ? { ...p, streaming: true, error: null, latencyMs: null, tokensUsed: null, promptTokens: null, completionTokens: null, costUsd: null } : p
    ));

    try {
      const imageUrls = parseImageInputUrls(videoImageUrls);
      const videoUrls = parseImageInputUrls(videoVideoUrls);
      const audioUrls = parseImageInputUrls(videoAudioUrls);
      const body: Record<string, unknown> = {
        model: modelId,
        prompt,
        resolution: videoResolution,
        duration: videoDuration,
        aspect_ratio: videoAspectRatio,
      };
      const selectedModel = modelIndex.get(modelId);
      if (isHappyHorseVideoModel(selectedModel)) {
        body.duration = videoDuration === 'auto' ? 5 : Number(videoDuration) || 5;
        body.enable_safety_checker = true;
      } else {
        body.generate_audio = videoGenerateAudio;
      }
      if (isImageToVideoModel(selectedModel)) {
        if (videoImageUrl.trim()) body.image_url = videoImageUrl.trim();
        if (videoEndImageUrl.trim()) body.end_image_url = videoEndImageUrl.trim();
      }
      if (isReferenceToVideoModel(selectedModel)) {
        if (imageUrls.length > 0) body.image_urls = imageUrls;
        if (videoUrls.length > 0) body.video_urls = videoUrls;
        if (audioUrls.length > 0) body.audio_urls = audioUrls;
      }

      const resp = await fetch(`${baseUrl}/v1/videos/generations`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${apiKey}`,
        },
        body: JSON.stringify(body),
        signal: controller.signal,
      });
      if (!resp.ok) {
        const errText = await resp.text();
        throw new Error(`${resp.status}: ${errText.slice(0, 200)}`);
      }
      let data = await resp.json();
      if (data?.result?.video_url) {
        data = data.result;
      } else if (data?.id && data?.status && data.status !== 'completed' && data.status !== 'failed') {
        data = await pollVideoJob(baseUrl, apiKey, data.id, controller.signal);
      } else if (data?.status === 'completed' && data?.result?.video_url) {
        data = data.result;
      } else if (data?.status === 'failed') {
        throw new Error(data?.error?.message || 'Video generation failed');
      }
      if (!data?.video_url) throw new Error('No video returned');
      const latency = Math.round(performance.now() - start);
      const modelCfg = modelIndex.get(modelId);
      const durationSeconds = videoDuration === 'auto' ? 10 : Number(videoDuration) || 10;
      const perSecond = videoUrls.length > 0 && modelCfg?.pricing?.per_second_with_video_input
        ? modelCfg.pricing.per_second_with_video_input
        : modelCfg?.pricing?.per_second;
      const cost = perSecond ? perSecond * durationSeconds : modelCfg?.pricing?.per_video || null;

      setPanes(prev => prev.map(p => {
        if (p.id !== paneId) return p;
        const msgs = [...p.messages, { role: 'assistant' as const, content: '', videoUrl: data.video_url }];
        savePaneHistory(modelId, msgs);
        return { ...p, messages: msgs, streaming: false, latencyMs: latency, costUsd: cost, tokensUsed: null, promptTokens: null, completionTokens: null };
      }));
      refreshBalance();
    } catch (err: any) {
      if (err.name === 'AbortError') {
        setPanes(prev => prev.map(p => p.id === paneId ? { ...p, streaming: false } : p));
        return;
      }
      setPanes(prev => prev.map(p => p.id === paneId ? { ...p, streaming: false, error: err.message } : p));
    } finally {
      abortRefs.current.delete(paneId);
    }
  }, [apiKey, baseUrl, modelIndex, refreshBalance, videoAspectRatio, videoAudioUrls, videoDuration, videoEndImageUrl, videoGenerateAudio, videoImageUrl, videoImageUrls, videoResolution, videoVideoUrls]);

  const sendToSpeechModel = useCallback(async (paneId: string, modelId: string, text: string) => {
    const controller = new AbortController();
    abortRefs.current.set(paneId, controller);
    const start = performance.now();

    setPanes(prev => prev.map(p =>
      p.id === paneId ? { ...p, streaming: true, error: null, latencyMs: null, tokensUsed: null, promptTokens: null, completionTokens: null, costUsd: null } : p
    ));

    try {
      const selectedModel = modelIndex.get(modelId);
      const isGeminiTTS = isGeminiSpeechModel(selectedModel);
      const speechInput = isGeminiTTS ? buildGeminiTTSPrompt(text) : text;
      const body: Record<string, unknown> = {
        model: modelId,
        input: speechInput,
        voice: speechVoice,
        language: speechLanguage,
      };
      if (speechAutoEmotion && isGeminiTTS) body.auto_emotion = true;
      if (isGeminiTTS) {
        body.temperature = 1;
        const speakerVoices = currentGeminiSpeakerVoices(text);
        if (speakerVoices) body.speaker_voices = speakerVoices;
      }

      const resp = await fetch(`${baseUrl}/v1/audio/speech`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${apiKey}`,
        },
        body: JSON.stringify(body),
        signal: controller.signal,
      });
      if (!resp.ok) {
        const errText = await resp.text();
        throw new Error(`${resp.status}: ${errText.slice(0, 200)}`);
      }
      const data = await resp.json();
      if (!data?.audio && !data?.audio_url) throw new Error('No audio returned');
      const latency = Math.round(performance.now() - start);
      const chars = typeof data.characters === 'number' ? data.characters : text.length;
      const modelCfg = modelIndex.get(modelId);
      const charPrice = modelCfg?.pricing?.input_per_1m_tokens;
      const cost = charPrice ? charPrice * chars / 1_000_000 : null;

      setPanes(prev => prev.map(p => {
        if (p.id !== paneId) return p;
        const msgs = [...p.messages, {
          role: 'assistant' as const,
          content: `${chars.toLocaleString()} characters synthesized with ${speechVoice}.`,
          audioB64: data.audio,
          audioUrl: data.audio_url,
          audioFormat: data.format || 'mp3',
        }];
        savePaneHistory(modelId, msgs);
        return {
          ...p,
          messages: msgs,
          streaming: false,
          latencyMs: latency,
          tokensUsed: null,
          promptTokens: chars,
          completionTokens: null,
          costUsd: cost,
        };
      }));
      refreshBalance();
    } catch (err: any) {
      if (err.name === 'AbortError') {
        setPanes(prev => prev.map(p => p.id === paneId ? { ...p, streaming: false } : p));
        return;
      }
      setPanes(prev => prev.map(p => paneId === p.id ? { ...p, streaming: false, error: err.message } : p));
    } finally {
      abortRefs.current.delete(paneId);
    }
  }, [apiKey, baseUrl, buildGeminiTTSPrompt, currentGeminiSpeakerVoices, modelIndex, refreshBalance, speechAutoEmotion, speechLanguage, speechVoice]);

  const sendToMusicModel = useCallback(async (paneId: string, modelId: string, text: string) => {
    const controller = new AbortController();
    abortRefs.current.set(paneId, controller);
    const start = performance.now();

    setPanes(prev => prev.map(p =>
      p.id === paneId ? { ...p, streaming: true, error: null, latencyMs: null, tokensUsed: null, promptTokens: null, completionTokens: null, costUsd: null } : p
    ));

    try {
      const resp = await fetch(`${baseUrl}/v1/music/generations`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${apiKey}`,
        },
        body: JSON.stringify({
          model: modelId,
          prompt: text,
          output_format: 'b64_json',
        }),
        signal: controller.signal,
      });
      if (!resp.ok) {
        const errText = await resp.text();
        throw new Error(`${resp.status}: ${errText.slice(0, 200)}`);
      }
      const data = await resp.json();
      const audio = data?.data?.audio;
      if (!audio) throw new Error('No audio returned');
      const latency = Math.round(performance.now() - start);
      const modelCfg = modelIndex.get(modelId);
      const cost = modelCfg?.pricing?.per_request || modelCfg?.pricing?.per_image || null;

      setPanes(prev => prev.map(p => {
        if (p.id !== paneId) return p;
        const msgs = [...p.messages, {
          role: 'assistant' as const,
          content: 'Generated music.',
          audioB64: audio,
          audioFormat: 'mpeg',
        }];
        savePaneHistory(modelId, msgs);
        return {
          ...p,
          messages: msgs,
          streaming: false,
          latencyMs: latency,
          tokensUsed: null,
          promptTokens: null,
          completionTokens: null,
          costUsd: cost,
        };
      }));
      refreshBalance();
    } catch (err: any) {
      if (err.name === 'AbortError') {
        setPanes(prev => prev.map(p => p.id === paneId ? { ...p, streaming: false } : p));
        return;
      }
      setPanes(prev => prev.map(p => paneId === p.id ? { ...p, streaming: false, error: err.message } : p));
    } finally {
      abortRefs.current.delete(paneId);
    }
  }, [apiKey, baseUrl, modelIndex, refreshBalance]);

  const sendToModel = useCallback(async (paneId: string, modelId: string, messages: Message[]) => {
    // Image models get routed to /v1/images/generations.
    if (isMusicModel(modelIndex.get(modelId))) {
      const lastUser = [...messages].reverse().find(m => m.role === 'user');
      if (!lastUser) return;
      return sendToMusicModel(paneId, modelId, lastUser.content);
    }
    if (isImageModel(modelIndex.get(modelId))) {
      const lastUser = [...messages].reverse().find(m => m.role === 'user');
      if (!lastUser) return;
      return sendToImageModel(paneId, modelId, lastUser.content);
    }
    if (isVideoModel(modelIndex.get(modelId))) {
      const lastUser = [...messages].reverse().find(m => m.role === 'user');
      if (!lastUser) return;
      return sendToVideoModel(paneId, modelId, lastUser.content);
    }
    if (isSpeechModel(modelIndex.get(modelId))) {
      const lastUser = [...messages].reverse().find(m => m.role === 'user');
      if (!lastUser) return;
      return sendToSpeechModel(paneId, modelId, lastUser.content);
    }

    const controller = new AbortController();
    abortRefs.current.set(paneId, controller);

    const start = performance.now();

    setPanes(prev => prev.map(p =>
      p.id === paneId ? { ...p, streaming: true, error: null, latencyMs: null, tokensUsed: null, promptTokens: null, completionTokens: null, costUsd: null } : p
    ));

    try {
      const body = {
        model: modelId,
        messages: [
          ...(systemPrompt ? [{ role: 'system', content: systemPrompt }] : []),
          ...messages,
        ],
        stream: true,
        temperature,
        max_tokens: maxTokens,
      };

      const resp = await fetch(`${baseUrl}/v1/chat/completions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${apiKey}`,
        },
        body: JSON.stringify(body),
        signal: controller.signal,
      });

      if (!resp.ok) {
        const errText = await resp.text();
        throw new Error(`${resp.status}: ${errText.slice(0, 200)}`);
      }

      const reader = resp.body?.getReader();
      if (!reader) throw new Error('No response body');

      const decoder = new TextDecoder();
      let assistantContent = '';
      let firstToken = true;
      let totalTokens: number | null = null;
      let promptTokens: number | null = null;
      let completionTokens: number | null = null;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const chunk = decoder.decode(value, { stream: true });
        const lines = chunk.split('\n');

        for (const line of lines) {
          if (!line.startsWith('data: ')) continue;
          const data = line.slice(6);
          if (data === '[DONE]') continue;

          try {
            const parsed = JSON.parse(data);
            const delta = parsed.choices?.[0]?.delta?.content;
            if (parsed.usage) {
              if (typeof parsed.usage.total_tokens === 'number') totalTokens = parsed.usage.total_tokens;
              if (typeof parsed.usage.prompt_tokens === 'number') promptTokens = parsed.usage.prompt_tokens;
              if (typeof parsed.usage.completion_tokens === 'number') completionTokens = parsed.usage.completion_tokens;
            }
            if (delta) {
              if (firstToken) {
                firstToken = false;
                const ttft = Math.round(performance.now() - start);
                setPanes(prev => prev.map(p =>
                  p.id === paneId ? { ...p, latencyMs: ttft } : p
                ));
              }
              assistantContent += delta;
              setPanes(prev => prev.map(p => {
                if (p.id !== paneId) return p;
                const msgs = [...p.messages];
                const last = msgs[msgs.length - 1];
                if (last && last.role === 'assistant') {
                  msgs[msgs.length - 1] = { ...last, content: assistantContent };
                } else {
                  msgs.push({ role: 'assistant', content: assistantContent });
                }
                return { ...p, messages: msgs };
              }));
            }
          } catch {}
        }
      }

      setPanes(prev => prev.map(p => {
        if (p.id !== paneId) return p;
        const cost = estimateCost(modelIndex.get(p.modelId), promptTokens || 0, completionTokens || 0);
        return { ...p, streaming: false, tokensUsed: totalTokens, promptTokens, completionTokens, costUsd: cost };
      }));
      savePaneHistory(modelId, [...messages, { role: 'assistant', content: assistantContent }]);
      refreshBalance();
    } catch (err: any) {
      if (err.name === 'AbortError') {
        setPanes(prev => prev.map(p =>
          p.id === paneId ? { ...p, streaming: false } : p
        ));
        return;
      }
      setPanes(prev => prev.map(p =>
        p.id === paneId ? { ...p, streaming: false, error: err.message } : p
      ));
    } finally {
      abortRefs.current.delete(paneId);
    }
  }, [apiKey, baseUrl, systemPrompt, temperature, maxTokens, modelIndex, refreshBalance, sendToImageModel, sendToVideoModel, sendToSpeechModel, sendToMusicModel]);

  function generateCode(lang: CodeLanguage): string {
    const pane = panes[0];
    const model = pane.modelId;
    const allMessages: Message[] = [
      ...(systemPrompt ? [{ role: 'system' as const, content: systemPrompt }] : []),
      ...pane.messages,
    ];
    const baseUrl = `${window.location.origin}/v1`;
    const exampleApiKey = apiKey || 'op-...';
    const isImageCode = primaryIsImage;
    const isVideoCode = primaryIsVideo;
    const isSpeechCode = primaryIsSpeech;
    const isMusicCode = primaryIsMusic;
    const payload = {
      model,
      messages: allMessages.map(({ role, content }) => ({ role, content })),
      temperature,
      max_tokens: maxTokens,
    };
    const imagePayload = {
      model,
      prompt: input.trim() || [...allMessages].reverse().find(m => m.role === 'user')?.content || 'A cinematic product photo of a matte black espresso machine on a marble counter',
      n: imageCount,
      size: imageSize,
      quality: imageQuality,
      response_format: imageResponseFormat,
    };
    const inputUrls = parseImageInputUrls(imageInputUrls);
    const imageIsOutpaint = model === 'fal-ai/flux-2-pro/outpaint';
    const imageDemo = IMAGE_DEMOS[model];
    const imageRequestPath = inputUrls.length > 0 && !imageIsOutpaint ? 'images/edits' : 'images/generations';
    const imageCodePayload = imageIsOutpaint
      ? {
          model,
          image_url: inputUrls[0] || IMAGE_DEMOS['fal-ai/flux-2-pro/outpaint'].imageUrl,
          expand_top: outpaintTop,
          expand_bottom: outpaintBottom,
          expand_left: outpaintLeft,
          expand_right: outpaintRight,
          enable_safety_checker: true,
          output_format: 'jpeg',
        }
      : inputUrls.length > 0
      ? {
          model,
          prompt: imagePayload.prompt,
          images: inputUrls.map(url => ({ type: 'image_url', url })),
          reference_image_urls: inputUrls,
          aspect_ratio: imageAspectRatio,
          ...(imageDemo?.imageSize ? { image_size: imageDemo.imageSize } : {}),
          ...(imageDemo?.numInferenceSteps ? { num_inference_steps: imageDemo.numInferenceSteps } : {}),
          ...(imageDemo?.guidanceScale !== undefined ? { guidance_scale: imageDemo.guidanceScale } : {}),
          ...(imageDemo ? { num_images: imageCount } : {}),
          ...(imageDemo?.outputFormat ? { output_format: imageDemo.outputFormat } : {}),
          ...(imageDemo?.safetyChecker !== undefined ? { enable_safety_checker: imageDemo.safetyChecker } : {}),
        }
      : imagePayload;
    const videoPrompt = input.trim() || [...allMessages].reverse().find(m => m.role === 'user')?.content || 'A cinematic handheld shot of a rainy neon street at night';
    const videoCodePayload: Record<string, unknown> = {
      model,
      prompt: videoPrompt,
      resolution: videoResolution,
      duration: videoDuration,
      aspect_ratio: videoAspectRatio,
    };
    if (primaryIsHappyHorseVideo) {
      videoCodePayload.duration = videoDuration === 'auto' ? 5 : Number(videoDuration) || 5;
      videoCodePayload.enable_safety_checker = true;
    } else {
      videoCodePayload.generate_audio = videoGenerateAudio;
    }
    const videoImages = parseImageInputUrls(videoImageUrls);
    const videoVideos = parseImageInputUrls(videoVideoUrls);
    const videoAudios = parseImageInputUrls(videoAudioUrls);
    if (primaryIsImageToVideo) {
      if (videoImageUrl.trim()) videoCodePayload.image_url = videoImageUrl.trim();
      if (videoEndImageUrl.trim()) videoCodePayload.end_image_url = videoEndImageUrl.trim();
    }
    if (primaryIsReferenceToVideo) {
      if (videoImages.length > 0) videoCodePayload.image_urls = videoImages;
      if (videoVideos.length > 0) videoCodePayload.video_urls = videoVideos;
      if (videoAudios.length > 0) videoCodePayload.audio_urls = videoAudios;
    }
    const rawSpeechText = input.trim() || [...allMessages].reverse().find(m => m.role === 'user')?.content || (primaryIsGeminiSpeech ? promptExampleText(GEMINI_TTS_QUICK_PROMPTS[0]) : 'Hello from Grok text to speech.');
    const speechText = primaryIsGeminiSpeech ? buildGeminiTTSPrompt(rawSpeechText) : rawSpeechText;
    const speechCodePayload: Record<string, unknown> = {
      model,
      input: speechText,
      voice: speechVoice,
      language: speechLanguage,
    };
    if (speechAutoEmotion && primaryIsGeminiSpeech) speechCodePayload.auto_emotion = true;
    if (primaryIsGeminiSpeech) {
      speechCodePayload.temperature = 1;
      const speakerVoices = currentGeminiSpeakerVoices(rawSpeechText);
      if (speakerVoices) speechCodePayload.speaker_voices = speakerVoices;
    }
    const musicPrompt = input.trim() || [...allMessages].reverse().find(m => m.role === 'user')?.content || promptExampleText(MUSIC_QUICK_PROMPTS[0]);
    const musicCodePayload = {
      model,
      prompt: musicPrompt,
      output_format: 'b64_json',
    };

    if (lang === 'python') {
      if (isMusicCode) {
        return `from openai import OpenAI
import base64
import pathlib

client = OpenAI(
    api_key="${exampleApiKey}",
    base_url="${baseUrl}",
)

result = client.post(
    "/music/generations",
    body=${JSON.stringify(musicCodePayload, null, 4)},
    cast_to=dict,
)

pathlib.Path("openpaths-music.mp3").write_bytes(base64.b64decode(result["data"]["audio"]))
print("wrote openpaths-music.mp3")`;
      }
      if (isSpeechCode) {
        return `from openai import OpenAI
import base64
import pathlib

client = OpenAI(
    api_key="${exampleApiKey}",
    base_url="${baseUrl}",
)

result = client.post(
    "/audio/speech",
    body=${JSON.stringify(speechCodePayload, null, 4)},
    cast_to=dict,
)

pathlib.Path("openpaths-speech.mp3").write_bytes(base64.b64decode(result["audio"]))
print("wrote openpaths-speech.mp3")`;
      }
      if (isVideoCode) {
        return `from openai import OpenAI

client = OpenAI(
    api_key="${exampleApiKey}",
    base_url="${baseUrl}",
)

result = client.post(
    "/videos/generations",
    body=${JSON.stringify(videoCodePayload, null, 4)},
    cast_to=dict,
)

print(result["video_url"])`;
      }
      if (isImageCode) {
        return `from openai import OpenAI
import base64
import pathlib
import requests

client = OpenAI(
    api_key="${exampleApiKey}",
    base_url="${baseUrl}",
)

result = client.post(
    "/${imageRequestPath}",
    body=${JSON.stringify(imageCodePayload, null, 4)},
    cast_to=dict,
)

image = result["data"][0]
if image.get("b64_json"):
    pathlib.Path("openpaths-image.png").write_bytes(base64.b64decode(image["b64_json"]))
else:
    pathlib.Path("openpaths-image.png").write_bytes(requests.get(image["url"]).content)

print("wrote openpaths-image.png")`;
      }
      const messagesStr = allMessages
        .map(m => `        {"role": "${m.role}", "content": ${JSON.stringify(m.content)}},`)
        .join('\n');
      return `from openai import OpenAI

client = OpenAI(
    api_key="${exampleApiKey}",
    base_url="${baseUrl}",
)

completion = client.chat.completions.create(
    model="${model}",
    messages=[
${messagesStr}
    ],
    temperature=${temperature},
    max_tokens=${maxTokens},
)

print(completion.choices[0].message.content)`;
    } else if (lang === 'js') {
      if (isMusicCode) {
        return `import OpenAI from "openai";
import { writeFile } from "node:fs/promises";

const client = new OpenAI({
  apiKey: "${exampleApiKey}",
  baseURL: "${baseUrl}",
});

const result = await client.post("/music/generations", {
  body: ${JSON.stringify(musicCodePayload, null, 2)},
  cast_to: Object,
});

await writeFile("openpaths-music.mp3", Buffer.from(result.data.audio, "base64"));
console.log("wrote openpaths-music.mp3");`;
      }
      if (isSpeechCode) {
        return `import OpenAI from "openai";
import { writeFile } from "node:fs/promises";

const client = new OpenAI({
  apiKey: "${exampleApiKey}",
  baseURL: "${baseUrl}",
});

const result = await client.post("/audio/speech", {
  body: ${JSON.stringify(speechCodePayload, null, 2)},
  cast_to: Object,
});

await writeFile("openpaths-speech.mp3", Buffer.from(result.audio, "base64"));
console.log("wrote openpaths-speech.mp3");`;
      }
      if (isVideoCode) {
        return `import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "${exampleApiKey}",
  baseURL: "${baseUrl}",
});

const result = await client.post("/videos/generations", {
  body: ${JSON.stringify(videoCodePayload, null, 2)},
  cast_to: Object,
});

console.log(result.video_url);`;
      }
      if (isImageCode) {
        return `import OpenAI from "openai";
import { writeFile } from "node:fs/promises";

const client = new OpenAI({
  apiKey: "${exampleApiKey}",
  baseURL: "${baseUrl}",
});

const result = await client.post("/${imageRequestPath}", {
  body: ${JSON.stringify(imageCodePayload, null, 2)},
  cast_to: Object,
});

const image = result.data[0];
if (image.b64_json) {
  await writeFile("openpaths-image.png", Buffer.from(image.b64_json, "base64"));
} else {
  const response = await fetch(image.url);
  await writeFile("openpaths-image.png", Buffer.from(await response.arrayBuffer()));
}

console.log("wrote openpaths-image.png");`;
      }
      const messagesStr = allMessages
        .map(m => `    { role: "${m.role}", content: ${JSON.stringify(m.content)} },`)
        .join('\n');
      return `import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "${exampleApiKey}",
  baseURL: "${baseUrl}",
});

const completion = await client.chat.completions.create({
  model: "${model}",
  messages: [
${messagesStr}
  ],
  temperature: ${temperature},
  max_tokens: ${maxTokens},
});

console.log(completion.choices[0].message.content);`;
    } else if (lang === 'go') {
      if (isMusicCode) {
        return `package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	payload := ${JSON.stringify(JSON.stringify(musicCodePayload, null, 2))}

	req, err := http.NewRequest("POST", "${baseUrl}/music/generations", bytes.NewBufferString(payload))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ${exampleApiKey}")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode >= 400 {
		panic(fmt.Sprintf("%s: %s", resp.Status, body))
	}

	var result struct {
		Data struct {
			Audio string \`json:"audio"\`
		} \`json:"data"\`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		panic(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(result.Data.Audio)
	if err != nil {
		panic(err)
	}
	os.WriteFile("openpaths-music.mp3", decoded, 0644)
}`;
      }
      if (isSpeechCode) {
        return `package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	payload := ${JSON.stringify(JSON.stringify(speechCodePayload, null, 2))}

	req, err := http.NewRequest("POST", "${baseUrl}/audio/speech", bytes.NewBufferString(payload))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ${exampleApiKey}")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode >= 400 {
		panic(fmt.Sprintf("%s: %s", resp.Status, body))
	}

	var result struct {
		Audio string \`json:"audio"\`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		panic(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(result.Audio)
	if err != nil {
		panic(err)
	}
	os.WriteFile("openpaths-speech.mp3", decoded, 0644)
}`;
      }
      if (isVideoCode) {
        return `package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	payload := ${JSON.stringify(JSON.stringify(videoCodePayload, null, 2))}

	req, err := http.NewRequest("POST", "${baseUrl}/videos/generations", bytes.NewBufferString(payload))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ${exampleApiKey}")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode >= 400 {
		panic(fmt.Sprintf("%s: %s", resp.Status, body))
	}

	var result struct {
		VideoURL string \`json:"video_url"\`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		panic(err)
	}
	fmt.Println(result.VideoURL)
}`;
      }
      if (isImageCode) {
        return `package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	payload := ${JSON.stringify(JSON.stringify(imageCodePayload, null, 2))}

	req, err := http.NewRequest("POST", "${baseUrl}/${imageRequestPath}", bytes.NewBufferString(payload))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ${exampleApiKey}")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode >= 400 {
		panic(fmt.Sprintf("%s: %s", resp.Status, body))
	}

	var result struct {
		Data []struct {
			URL     string \`json:"url"\`
			B64JSON string \`json:"b64_json"\`
		} \`json:"data"\`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		panic(err)
	}
	if len(result.Data) == 0 {
		panic("no image returned")
	}
	if result.Data[0].B64JSON != "" {
		decoded, err := base64.StdEncoding.DecodeString(result.Data[0].B64JSON)
		if err != nil {
			panic(err)
		}
		os.WriteFile("openpaths-image.png", decoded, 0644)
	} else {
		fmt.Println(result.Data[0].URL)
	}
}`;
      }
      return `package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	payload := ${JSON.stringify(JSON.stringify(payload, null, 2))}

	req, err := http.NewRequest("POST", "${baseUrl}/chat/completions", bytes.NewBufferString(payload))
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ${exampleApiKey}")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode >= 400 {
		panic(fmt.Sprintf("%s: %s", resp.Status, body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string \`json:"content"\`
			} \`json:"message"\`
		} \`json:"choices"\`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		panic(err)
	}
	if len(result.Choices) > 0 {
		fmt.Println(result.Choices[0].Message.Content)
	}
}`;
    } else {
      if (isMusicCode) {
        return `curl "${baseUrl}/music/generations" \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${exampleApiKey}" \\
  -d @- <<'JSON'
${JSON.stringify(musicCodePayload, null, 2)}
JSON`;
      }
      if (isSpeechCode) {
        return `curl "${baseUrl}/audio/speech" \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${exampleApiKey}" \\
  -d @- <<'JSON'
${JSON.stringify(speechCodePayload, null, 2)}
JSON`;
      }
      if (isVideoCode) {
        return `curl "${baseUrl}/videos/generations" \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${exampleApiKey}" \\
  -d @- <<'JSON'
${JSON.stringify(videoCodePayload, null, 2)}
JSON`;
      }
      if (isImageCode) {
        return `curl "${baseUrl}/${imageRequestPath}" \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${exampleApiKey}" \\
  -d @- <<'JSON'
${JSON.stringify(imageCodePayload, null, 2)}
JSON`;
      }
      return `curl "${baseUrl}/chat/completions" \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${exampleApiKey}" \\
  -d @- <<'JSON'
${JSON.stringify(payload, null, 2)}
JSON`;
    }
  }

  useEffect(() => {
    const promptParam = searchParams.get('prompt');
    if (!promptParam || input.trim() || autoRanRef.current) return;
    setInput(promptParam);
  }, [input, searchParams]);

  function copyCode() {
    navigator.clipboard.writeText(generateCode(codeLang));
    setCodeCopied(true);
    setTimeout(() => setCodeCopied(false), 2000);
  }

  const handleSend = useCallback((text?: string) => {
    const msg = (text || input).trim();
    if (!msg || !apiKey) return;

    const userMsg: Message = { role: 'user', content: msg };
    if (!text) setInput('');
    setTimeout(() => inputRef.current?.focus(), 0);

    const updatedPanes = panes.map(p => ({
      ...p,
      messages: [...p.messages, userMsg],
    }));
    setPanes(updatedPanes);

    for (const pane of updatedPanes) {
      savePaneHistory(pane.modelId, pane.messages);
      sendToModel(pane.id, pane.modelId, pane.messages);
    }
  }, [input, apiKey, panes, sendToModel]);

  // Auto-run a prompt passed via ?prompt= once the API key is ready.
  useEffect(() => {
    if (autoRanRef.current) return;
    const promptParam = searchParams.get('prompt');
    if (!promptParam || !apiKey) return;
    autoRanRef.current = true;
    handleSend(promptParam);
    // Strip the prompt param so refreshing doesn't re-trigger.
    const next = new URLSearchParams(searchParams);
    next.delete('prompt');
    setSearchParams(next, { replace: true });
  }, [searchParams, apiKey, handleSend, setSearchParams]);

  function copyShareLink() {
    const firstPane = panes[0];
    const lastUser = [...firstPane.messages].reverse().find(m => m.role === 'user');
    const u = new URL(window.location.origin + '/playground');
    u.searchParams.set('model', firstPane.modelId);
    if (lastUser) u.searchParams.set('prompt', lastUser.content);
    navigator.clipboard.writeText(u.toString());
    setShareCopied(true);
    setTimeout(() => setShareCopied(false), 2000);
  }

  function stopAll() {
    for (const ctrl of abortRefs.current.values()) ctrl.abort();
    abortRefs.current.clear();
  }

  function addPane() {
    if (panes.length >= 4) return;
    const usedModels = new Set(panes.map(p => p.modelId));
    const next = chatCatalog.find(m => !usedModels.has(m.id)) || chatCatalog[0];
    const pane = makePane(next.id);
    pane.messages = loadPaneHistory(next.id);
    setPanes(prev => [...prev, pane]);
  }

  function removePane(id: string) {
    if (panes.length <= 1) return;
    const ctrl = abortRefs.current.get(id);
    if (ctrl) ctrl.abort();
    setPanes(prev => prev.filter(p => p.id !== id));
  }

  function changeModel(paneId: string, modelId: string) {
    setPanes(prev => prev.map(p => {
      if (p.id !== paneId) return p;
      // Swap in persisted history for the new model (if any).
      return { ...p, modelId, messages: loadPaneHistory(modelId), error: null, latencyMs: null, tokensUsed: null, promptTokens: null, completionTokens: null, costUsd: null };
    }));
  }

  function clearAll() {
    stopAll();
    setPanes(prev => prev.map(p => {
      savePaneHistory(p.modelId, []);
      return { ...p, messages: [], streaming: false, error: null, latencyMs: null, tokensUsed: null, promptTokens: null, completionTokens: null, costUsd: null };
    }));
  }

  function retryLast(paneId: string) {
    const pane = panes.find(p => p.id === paneId);
    if (!pane) return;
    // Remove last assistant message and re-send
    const msgs = pane.messages.filter((_, i) => !(i === pane.messages.length - 1 && pane.messages[i].role === 'assistant'));
    setPanes(prev => prev.map(p => p.id === paneId ? { ...p, messages: msgs, error: null } : p));
    savePaneHistory(pane.modelId, msgs);
    sendToModel(paneId, pane.modelId, msgs);
  }

  // Regenerate the turn at `index`. Whether the clicked message is the user
  // prompt or the assistant reply, we re-send the conversation up to and
  // including the preceding user message, dropping that turn's old output and
  // anything after it (standard chat "regenerate" behaviour).
  function retryMessage(paneId: string, index: number) {
    const pane = panes.find(p => p.id === paneId);
    if (!pane) return;
    const ctrl = abortRefs.current.get(paneId);
    if (ctrl) ctrl.abort();
    const target = pane.messages[index];
    if (!target) return;
    // Cut point: keep everything strictly before the assistant reply. If the
    // user clicked a user message, keep that message and drop what follows.
    const cut = target.role === 'user' ? index + 1 : index;
    const msgs = pane.messages.slice(0, cut);
    if (!msgs.some(m => m.role === 'user')) return;
    setPanes(prev => prev.map(p => p.id === paneId ? { ...p, messages: msgs, error: null } : p));
    savePaneHistory(pane.modelId, msgs);
    sendToModel(paneId, pane.modelId, msgs);
  }

  function deleteMessage(paneId: string, index: number) {
    const pane = panes.find(p => p.id === paneId);
    if (!pane) return;
    const msgs = pane.messages.filter((_, i) => i !== index);
    setPanes(prev => prev.map(p => p.id === paneId ? { ...p, messages: msgs } : p));
    savePaneHistory(pane.modelId, msgs);
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  const hasMessages = panes.some(p => p.messages.length > 0);

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="border-b border-white/10 px-4 py-2.5 flex items-center gap-2 bg-white/[0.02] overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden [&>*]:shrink-0">
        <button
          onClick={() => setShowSettings(!showSettings)}
          className={`flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-mono rounded border transition-colors ${showSettings ? 'border-white/30 bg-white/10 text-white' : 'border-white/10 text-white/60 hover:text-white hover:border-white/20'}`}
        >
          <Settings className="w-3.5 h-3.5" /> Settings
        </button>
        <button
          onClick={addPane}
          disabled={panes.length >= 4}
          className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-mono rounded border border-white/10 text-white/60 hover:text-white hover:border-white/20 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
        >
          <Plus className="w-3.5 h-3.5" /> Compare
        </button>
        {hasMessages && (
          <button
            onClick={clearAll}
            className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-mono rounded border border-white/10 text-white/60 hover:text-white hover:border-white/20 transition-colors"
          >
            <Trash2 className="w-3.5 h-3.5" /> Clear
          </button>
        )}
        {(hasMessages || primaryIsImage || primaryIsVideo || primaryIsSpeech || primaryIsMusic) && (
          <button
            onClick={() => setShowCode(!showCode)}
            className={`flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-mono rounded border transition-colors ${showCode ? 'border-white/30 bg-white/10 text-white' : 'border-white/10 text-white/60 hover:text-white hover:border-white/20'}`}
          >
            <Code2 className="w-3.5 h-3.5" /> Copy Code
          </button>
        )}
        {hasMessages && (
          <button
            onClick={copyShareLink}
            className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-mono rounded border border-white/10 text-white/60 hover:text-white hover:border-white/20 transition-colors"
            title="Copy a link that replays this prompt"
          >
            {shareCopied ? <><Check className="w-3.5 h-3.5 text-green-400" /> Copied</> : <><Share2 className="w-3.5 h-3.5" /> Share</>}
          </button>
        )}
        <div className="ml-auto flex items-center gap-3">
          {apiKey && balanceCents !== null && (
            balanceCents <= 0 ? (
              <a href="/account" className="flex items-center gap-1.5 px-2.5 py-1.5 text-[11px] font-mono rounded border border-red-400/30 bg-red-400/5 text-red-300 hover:bg-red-400/10 transition-colors">
                <Wallet className="w-3.5 h-3.5" /> Top up credits
              </a>
            ) : (
              <a href="/account" className="flex items-center gap-1.5 text-[11px] font-mono text-white/50 hover:text-white/80 transition-colors" title="Account balance">
                <Wallet className="w-3.5 h-3.5" /> {fmtBalance(balanceCents)}
              </a>
            )
          )}
          {panes.length > 1 && (
            <span className="text-[10px] font-mono text-white/25">{panes.length}/4 models</span>
          )}
          <span className="text-[10px] font-mono text-white/25">
            {primaryIsMusic ? 'music generation' : primaryIsSpeech ? `${speechVoice} | ${speechLanguage}` : primaryIsVideo ? `${videoResolution} | ${videoDuration}s | ${videoAspectRatio}` : primaryIsImage ? `${imageSize} | ${imageQuality} | ${imageCount} img` : `temp ${temperature} | max ${maxTokens}`}
          </span>
        </div>
      </div>

      {/* Settings Panel */}
      {showSettings && (
        <div className="border-b border-white/10 px-4 py-4 bg-white/[0.02]">
          <div className="max-w-2xl grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">API Key</label>
              <input
                type="password"
                value={apiKey}
                onChange={e => setApiKey(e.target.value)}
                placeholder="op-..."
                className="w-full bg-black border border-white/10 rounded px-3 py-2 text-sm font-mono text-white placeholder:text-white/20 focus:outline-none focus:border-white/30"
              />
            </div>
            <div>
              <label className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">System Prompt</label>
              <textarea
                value={systemPrompt}
                onChange={e => setSystemPrompt(e.target.value)}
                rows={2}
                className="w-full bg-black border border-white/10 rounded px-3 py-2 text-sm font-mono text-white placeholder:text-white/20 focus:outline-none focus:border-white/30 resize-none"
                placeholder="You are a helpful assistant."
              />
            </div>
            <div>
              <label className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">
                Temperature <span className="text-white/60">{temperature}</span>
              </label>
              <input
                type="range"
                min="0"
                max="2"
                step="0.1"
                value={temperature}
                onChange={e => setTemperature(parseFloat(e.target.value))}
                className="w-full accent-white h-1"
              />
              <div className="flex justify-between text-[9px] font-mono text-white/20 mt-0.5">
                <span>Precise</span><span>Creative</span>
              </div>
            </div>
            <div>
              <label className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">
                Max Tokens <span className="text-white/60">{maxTokens.toLocaleString()}</span>
              </label>
              <input
                type="range"
                min="256"
                max="16384"
                step="256"
                value={maxTokens}
                onChange={e => setMaxTokens(parseInt(e.target.value))}
                className="w-full accent-white h-1"
              />
              <div className="flex justify-between text-[9px] font-mono text-white/20 mt-0.5">
                <span>256</span><span>16,384</span>
              </div>
            </div>
          </div>
        </div>
      )}

      {primaryIsImage && (
        <div className="border-b border-white/10 px-4 py-3 bg-black/40">
          <div className="max-w-5xl grid grid-cols-2 md:grid-cols-6 gap-3">
            {!primaryIsOutpaintImage && <label className="block">
              <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">Size</span>
              <select
                value={imageSize}
                onChange={e => setImageSize(e.target.value as typeof IMAGE_SIZES[number])}
                className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-white/30"
                data-testid="image-size"
              >
                {IMAGE_SIZES.map(size => <option key={size} value={size} className="bg-black text-white">{size}</option>)}
              </select>
            </label>}
            {!primaryIsOutpaintImage && <label className="block">
              <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">Quality</span>
              <select
                value={imageQuality}
                onChange={e => setImageQuality(e.target.value as typeof IMAGE_QUALITIES[number])}
                className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-white/30"
                data-testid="image-quality"
              >
                {IMAGE_QUALITIES.map(q => <option key={q} value={q} className="bg-black text-white">{q}</option>)}
              </select>
            </label>}
            {!primaryIsOutpaintImage && <label className="block">
              <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">Images</span>
              <input
                type="number"
                min="1"
                max="4"
                value={imageCount}
                onChange={e => setImageCount(Math.max(1, Math.min(4, Number(e.target.value) || 1)))}
                className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-white/30"
                data-testid="image-count"
              />
            </label>}
            {!primaryIsOutpaintImage && <label className="block">
              <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">Response</span>
              <select
                value={imageResponseFormat}
                onChange={e => setImageResponseFormat(e.target.value as typeof IMAGE_RESPONSE_FORMATS[number])}
                className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-white/30"
                data-testid="image-response-format"
              >
                {IMAGE_RESPONSE_FORMATS.map(format => <option key={format} value={format} className="bg-black text-white">{format}</option>)}
              </select>
            </label>}
            {!primaryIsOutpaintImage && <label className="block">
              <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">Aspect</span>
              <select
                value={imageAspectRatio}
                onChange={e => setImageAspectRatio(e.target.value as typeof IMAGE_ASPECT_RATIOS[number])}
                className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-white/30"
                data-testid="image-aspect-ratio"
              >
                {IMAGE_ASPECT_RATIOS.map(ratio => <option key={ratio} value={ratio} className="bg-black text-white">{ratio}</option>)}
              </select>
            </label>}
            {primaryIsOutpaintImage && [
              ['Top', outpaintTop, setOutpaintTop],
              ['Bottom', outpaintBottom, setOutpaintBottom],
              ['Left', outpaintLeft, setOutpaintLeft],
              ['Right', outpaintRight, setOutpaintRight],
            ].map(([label, value, setter]) => (
              <label key={label as string} className="block">
                <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">{label as string}</span>
                <input
                  type="number"
                  min="0"
                  max="2048"
                  value={value as number}
                  onChange={e => (setter as React.Dispatch<React.SetStateAction<number>>)(Math.max(0, Math.min(2048, Number(e.target.value) || 0)))}
                  className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-white/30"
                />
              </label>
            ))}
            <label className="block col-span-2 md:col-span-1">
              <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">{primaryIsOutpaintImage ? 'Source image' : 'Input URLs'}</span>
              <textarea
                value={imageInputUrls}
                onChange={e => setImageInputUrls(normalizeUploadUrlText(e.target.value))}
                rows={1}
                placeholder={primaryIsOutpaintImage ? 'Required source image URL' : 'Optional image URLs'}
                className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white placeholder:text-white/25 focus:outline-none focus:border-white/30 resize-none"
                data-testid="image-input-urls"
              />
              <ImagePreviewStrip urls={parseImageInputUrls(imageInputUrls)} label="image-input-urls" />
            </label>
          </div>
        </div>
      )}

      {primaryIsVideo && (
        <div className="border-b border-white/10 px-4 py-3 bg-black/40">
          <div className="max-w-5xl grid grid-cols-2 md:grid-cols-7 gap-3">
            <label className="block">
              <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">Resolution</span>
              <select value={videoResolution} onChange={e => setVideoResolution(e.target.value as typeof VIDEO_RESOLUTIONS[number])} className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-white/30" data-testid="video-resolution">
                {VIDEO_RESOLUTIONS.map(v => <option key={v} value={v} className="bg-black text-white">{v}</option>)}
              </select>
            </label>
            <label className="block">
              <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">Duration</span>
              <select value={videoDuration} onChange={e => setVideoDuration(e.target.value as typeof VIDEO_DURATIONS[number])} className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-white/30" data-testid="video-duration">
                {VIDEO_DURATIONS.map(v => <option key={v} value={v} className="bg-black text-white">{v}</option>)}
              </select>
            </label>
            <label className="block">
              <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">Aspect</span>
              <select value={videoAspectRatio} onChange={e => setVideoAspectRatio(e.target.value as typeof VIDEO_ASPECT_RATIOS[number])} className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-white/30" data-testid="video-aspect-ratio">
                {VIDEO_ASPECT_RATIOS.map(v => <option key={v} value={v} className="bg-black text-white">{v}</option>)}
              </select>
            </label>
            <label className="block">
              <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">{primaryIsHappyHorseVideo ? 'Safety' : 'Audio'}</span>
              <button type="button" onClick={() => !primaryIsHappyHorseVideo && setVideoGenerateAudio(v => !v)} className={`w-full border rounded px-3 py-2 text-xs font-mono transition-colors ${(primaryIsHappyHorseVideo || videoGenerateAudio) ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-200' : 'border-white/10 bg-black text-white/50'}`} data-testid="video-generate-audio">
                {primaryIsHappyHorseVideo || videoGenerateAudio ? 'on' : 'off'}
              </button>
            </label>
            {primaryIsImageToVideo ? (
              <>
                <label
                  className={`block col-span-2 md:col-span-2 rounded border p-2 transition-colors ${dragTarget === 'image' ? 'border-emerald-400/60 bg-emerald-400/10' : 'border-white/10 bg-white/[0.015]'}`}
                  data-testid="video-start-image-dropzone"
                  {...referenceDropHandlers('image')}
                >
                  <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">Start image</span>
                  <input value={videoImageUrl} onChange={e => setVideoImageUrl(normalizeUploadedAssetUrl(e.target.value))} placeholder="https://..." className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white placeholder:text-white/25 focus:outline-none focus:border-white/30" data-testid="video-image-url" />
                  <input type="file" accept="image/*" disabled={uploadingRefs} onChange={e => e.target.files && uploadReferenceFiles(e.target.files, 'image')} className="mt-1 block w-full text-[10px] font-mono text-white/35 file:mr-2 file:rounded file:border-0 file:bg-white/10 file:px-2 file:py-1 file:text-white/60" />
                  <SingleImagePreview url={videoImageUrl} label="video-start-image" />
                </label>
                <label
                  className={`block col-span-2 md:col-span-1 rounded border p-2 transition-colors ${dragTarget === 'end-image' ? 'border-emerald-400/60 bg-emerald-400/10' : 'border-white/10 bg-white/[0.015]'}`}
                  data-testid="video-end-image-dropzone"
                  {...referenceDropHandlers('end-image')}
                >
                  <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">End image</span>
                  <input value={videoEndImageUrl} onChange={e => setVideoEndImageUrl(normalizeUploadedAssetUrl(e.target.value))} placeholder="optional URL" className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white placeholder:text-white/25 focus:outline-none focus:border-white/30" data-testid="video-end-image-url" />
                  <input type="file" accept="image/*" disabled={uploadingRefs} onChange={e => e.target.files && uploadReferenceFiles(e.target.files, 'end-image')} className="mt-1 block w-full text-[10px] font-mono text-white/35 file:mr-2 file:rounded file:border-0 file:bg-white/10 file:px-2 file:py-1 file:text-white/60" />
                  <SingleImagePreview url={videoEndImageUrl} label="video-end-image" />
                </label>
              </>
            ) : primaryIsReferenceToVideo ? (
              <>
                <label className="block col-span-2 md:col-span-1">
                  <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">Images</span>
                  <textarea value={videoImageUrls} onChange={e => setVideoImageUrls(normalizeUploadUrlText(e.target.value))} rows={1} placeholder="image URLs" className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white placeholder:text-white/25 focus:outline-none focus:border-white/30 resize-none" data-testid="video-image-urls" />
                  <input type="file" accept="image/*" multiple disabled={uploadingRefs} onChange={e => e.target.files && uploadReferenceFiles(e.target.files, 'images')} className="mt-1 block w-full text-[10px] font-mono text-white/35 file:mr-2 file:rounded file:border-0 file:bg-white/10 file:px-2 file:py-1 file:text-white/60" />
                  <ImagePreviewStrip urls={parseImageInputUrls(videoImageUrls)} label="video-image-urls" />
                </label>
                <label className="block col-span-2 md:col-span-1">
                  <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">Videos</span>
                  <textarea value={videoVideoUrls} onChange={e => setVideoVideoUrls(e.target.value)} rows={1} placeholder="video URLs" className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white placeholder:text-white/25 focus:outline-none focus:border-white/30 resize-none" data-testid="video-video-urls" />
                  <input type="file" accept="video/mp4,video/quicktime,video/*" multiple disabled={uploadingRefs} onChange={e => e.target.files && uploadReferenceFiles(e.target.files, 'video')} className="mt-1 block w-full text-[10px] font-mono text-white/35 file:mr-2 file:rounded file:border-0 file:bg-white/10 file:px-2 file:py-1 file:text-white/60" />
                </label>
                <label className="block col-span-2 md:col-span-1">
                  <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">Audio refs</span>
                  <textarea value={videoAudioUrls} onChange={e => setVideoAudioUrls(e.target.value)} rows={1} placeholder="audio URLs" className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white placeholder:text-white/25 focus:outline-none focus:border-white/30 resize-none" data-testid="video-audio-urls" />
                  <input type="file" accept="audio/*" multiple disabled={uploadingRefs} onChange={e => e.target.files && uploadReferenceFiles(e.target.files, 'audio')} className="mt-1 block w-full text-[10px] font-mono text-white/35 file:mr-2 file:rounded file:border-0 file:bg-white/10 file:px-2 file:py-1 file:text-white/60" />
                </label>
              </>
            ) : null}
          </div>
        </div>
      )}

      {primaryIsSpeech && (
        <div className="border-b border-white/10 px-4 py-3 bg-black/40">
          {primaryIsGeminiSpeech ? (
            <div className="max-w-5xl space-y-3">
              <button
                type="button"
                onClick={() => setSpeechAutoEmotion(v => !v)}
                className={`inline-flex items-center gap-2 border rounded px-3 py-2 text-xs font-mono transition-colors ${speechAutoEmotion ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-200' : 'border-white/10 bg-black text-white/50 hover:text-white/75'}`}
                data-testid="speech-auto-emotion"
              >
                <span className={`w-2 h-2 rounded-full ${speechAutoEmotion ? 'bg-emerald-300' : 'bg-white/20'}`} />
                Auto emotion
              </button>
              <datalist id="gemini-tts-voices">
                {GEMINI_TTS_VOICE_INFO.map(voice => <option key={voice.name} value={voice.name}>{voice.style} · {voice.pitch}</option>)}
              </datalist>
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                <GeminiSpeakerControls
                  label="Speaker 1"
                  profile={ttsSpeaker1Profile}
                  onProfile={setTtsSpeaker1Profile}
                  style={ttsSpeaker1Style}
                  onStyle={v => setTtsSpeaker1Style(v as typeof TTS_STYLES[number])}
                  pace={ttsSpeaker1Pace}
                  onPace={v => setTtsSpeaker1Pace(v as typeof TTS_PACES[number])}
                  accent={ttsSpeaker1Accent}
                  onAccent={v => setTtsSpeaker1Accent(v as typeof TTS_ACCENTS[number])}
                  voice={ttsSpeaker1Voice}
                  onVoice={setTtsSpeaker1Voice}
                />
                <GeminiSpeakerControls
                  label="Speaker 2"
                  profile={ttsSpeaker2Profile}
                  onProfile={setTtsSpeaker2Profile}
                  style={ttsSpeaker2Style}
                  onStyle={v => setTtsSpeaker2Style(v as typeof TTS_STYLES[number])}
                  pace={ttsSpeaker2Pace}
                  onPace={v => setTtsSpeaker2Pace(v as typeof TTS_PACES[number])}
                  accent={ttsSpeaker2Accent}
                  onAccent={v => setTtsSpeaker2Accent(v as typeof TTS_ACCENTS[number])}
                  voice={ttsSpeaker2Voice}
                  onVoice={setTtsSpeaker2Voice}
                />
              </div>
            </div>
          ) : (
            <div className="max-w-xl grid grid-cols-2 gap-3">
              <label className="block">
                <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">Voice</span>
                <select
                  value={speechVoice}
                  onChange={e => setSpeechVoice(e.target.value)}
                  className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-white/30"
                  data-testid="speech-voice"
                >
                  {SPEECH_VOICES.map(voice => <option key={voice} value={voice} className="bg-black text-white">{voice}</option>)}
                </select>
              </label>
              <label className="block">
                <span className="text-[10px] font-mono text-white/40 uppercase tracking-wider block mb-1.5">Language</span>
                <select
                  value={speechLanguage}
                  onChange={e => setSpeechLanguage(e.target.value as typeof SPEECH_LANGUAGES[number])}
                  className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-white/30"
                  data-testid="speech-language"
                >
                  {SPEECH_LANGUAGES.map(language => <option key={language} value={language} className="bg-black text-white">{language}</option>)}
                </select>
              </label>
            </div>
          )}
          <p className="mt-2 text-[10px] font-mono text-white/30">{primaryIsGeminiSpeech ? '$1.00 input / $20.00 output per 1M tokens' : '$15.00 / 1M input characters'}</p>
        </div>
      )}

      {/* Code Panel */}
      {showCode && (
        <div className="border-b border-white/10 bg-white/[0.02]">
          <div className="flex items-center gap-1 px-4 pt-3 pb-0">
            {(['python', 'js', 'go', 'curl'] as const).map(lang => (
              <button
                key={lang}
                onClick={() => setCodeLang(lang)}
                className={`px-3 py-1.5 text-xs font-mono rounded-t border-t border-l border-r transition-colors ${codeLang === lang ? 'border-white/20 bg-black text-white' : 'border-transparent text-white/40 hover:text-white/70'}`}
              >
                {lang === 'python' ? 'Python' : lang === 'js' ? 'JavaScript' : lang === 'go' ? 'Go' : 'cURL'}
              </button>
            ))}
            <button
              onClick={copyCode}
              className="ml-auto flex items-center gap-1.5 px-3 py-1.5 text-xs font-mono border border-white/10 rounded text-white/60 hover:text-white hover:border-white/25 transition-colors mb-0.5"
            >
              {codeCopied ? <><Check className="w-3 h-3 text-green-400" /> Copied</> : <><Copy className="w-3 h-3" /> Copy</>}
            </button>
          </div>
          <HighlightedCodeBlock
            code={generateCode(codeLang)}
            language={codeLang === 'js' ? 'javascript' : codeLang === 'curl' ? 'bash' : codeLang}
            preClassName="px-4 pb-4 overflow-x-auto text-[12px] font-mono leading-relaxed text-white/80 bg-black mx-4 mb-3 rounded-b rounded-tr border border-white/10 pt-3"
            testId="playground-generated-code"
          />
        </div>
      )}

      {/* Model Panes */}
      <div className="flex-1 flex overflow-hidden">
        {panes.map(pane => (
          <div key={pane.id} className="flex-1 flex flex-col border-r border-white/10 last:border-r-0 min-w-0">
            {/* Pane Header */}
            <div className="flex items-center gap-2 px-3 py-2 border-b border-white/10 bg-white/[0.02]">
              <ModelSelect
                value={pane.modelId}
                onChange={m => changeModel(pane.id, m)}
                models={chatCatalog}
              />
              <div className="flex items-center gap-2 shrink-0 ml-auto">
                {pane.latencyMs !== null && (
                  <span className="text-[10px] font-mono text-green-400/70" title="Time to first token">
                    <Zap className="w-2.5 h-2.5 inline mr-0.5" />{pane.latencyMs}ms
                  </span>
                )}
                {pane.tokensUsed !== null && !pane.streaming && (
                  <span className="text-[10px] font-mono text-white/30" title="Total tokens">
                    {pane.tokensUsed.toLocaleString()} tok
                  </span>
                )}
                {pane.costUsd !== null && !pane.streaming && (
                  <span className="text-[10px] font-mono text-amber-300/70" title={`Prompt ${pane.promptTokens ?? '?'} + completion ${pane.completionTokens ?? '?'} tok`}>
                    {fmtCost(pane.costUsd)}
                  </span>
                )}
                {pane.streaming && (
                  <Loader2 className="w-3 h-3 animate-spin text-white/40" />
                )}
                {panes.length > 1 && (
                  <button onClick={() => removePane(pane.id)} className="text-white/20 hover:text-white/60">
                    <X className="w-3.5 h-3.5" />
                  </button>
                )}
              </div>
            </div>

            {/* Messages */}
            <div className="flex-1 overflow-y-auto">
              {pane.messages.length === 0 && !pane.error ? (
                <EmptyState
                  onPrompt={handleSend}
                  hasApiKey={!!apiKey}
                  isImage={isImageModel(modelIndex.get(pane.modelId))}
                  isSpeech={isSpeechModel(modelIndex.get(pane.modelId))}
                  isGeminiSpeech={isGeminiSpeechModel(modelIndex.get(pane.modelId))}
                  isMusic={isMusicModel(modelIndex.get(pane.modelId))}
                  imageDemo={IMAGE_DEMOS[pane.modelId]}
                  videoDemo={VIDEO_DEMOS[pane.modelId]}
                />
              ) : (
                <div className={`p-3 space-y-4 ${panes.length === 1 ? 'max-w-3xl mx-auto w-full' : ''}`}>
                  {pane.messages.map((msg, i) => (
                    <MessageBubble
                      key={i}
                      message={msg}
                      streaming={pane.streaming}
                      isLast={i === pane.messages.length - 1}
                      onRetry={() => retryMessage(pane.id, i)}
                      onDelete={() => deleteMessage(pane.id, i)}
                    />
                  ))}
                  {pane.error && (
                    <div className="flex items-start gap-2">
                      <div className="flex-1 text-xs font-mono text-red-400/80 bg-red-400/5 rounded-lg p-3 border border-red-400/10">
                        {pane.error}
                      </div>
                      <button onClick={() => retryLast(pane.id)} className="shrink-0 p-1.5 text-white/30 hover:text-white/60 transition-colors" title="Retry">
                        <RotateCcw className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  )}
                  <ScrollAnchor messages={pane.messages} streaming={pane.streaming} />
                </div>
              )}
            </div>
          </div>
        ))}
      </div>

      {/* Input */}
      <div className="border-t border-white/10 p-3 bg-white/[0.02]">
        <div className={`mx-auto flex gap-2 items-end ${panes.length === 1 ? 'max-w-3xl' : 'max-w-4xl'}`}>
          <textarea
            ref={inputRef}
            value={input}
            onChange={e => {
              setInput(e.target.value);
              resizeTextarea(e.target);
            }}
            onKeyDown={handleKeyDown}
            rows={1}
            placeholder={apiKey ? (primaryIsSpeech ? 'Enter text to synthesize...' : primaryIsVideo ? 'Describe the video you want to generate...' : primaryIsImage ? 'Describe the image you want to generate...' : 'Send a message... (Shift+Enter for newline)') : 'Set your API key in Settings first'}
            disabled={!apiKey}
            autoFocus
            data-testid="chat-input"
            className="flex-1 bg-black border border-white/10 rounded-lg px-4 py-3 text-sm text-white placeholder:text-white/20 focus:outline-none focus:border-white/30 focus:ring-1 focus:ring-white/10 resize-none overflow-y-hidden disabled:opacity-30 transition-colors"
            style={{ minHeight: '44px', maxHeight: '200px' }}
          />
          <SavePromptButton
            text={input}
            modelId={panes[0]?.modelId}
            modality={primaryIsImage ? 'image' : primaryIsMusic ? 'music' : primaryIsVideo ? 'video' : 'text'}
          />
          {anyStreaming ? (
            <button
              onClick={stopAll}
              data-testid="chat-stop"
              className="bg-red-500/20 text-red-400 border border-red-500/30 px-4 py-3 rounded-lg font-mono text-sm font-bold hover:bg-red-500/30 transition-colors shrink-0"
              title="Stop generation"
            >
              <Square className="w-4 h-4" />
            </button>
          ) : (
            <button
              onClick={() => handleSend()}
              disabled={!input.trim() || !apiKey}
              data-testid="chat-send"
              className="bg-white text-black px-4 py-3 rounded-lg font-mono text-sm font-bold hover:bg-white/90 transition-colors disabled:opacity-30 disabled:cursor-not-allowed shrink-0"
            >
              <Send className="w-4 h-4" />
            </button>
          )}
        </div>
        {!apiKey && (
          <p className="text-center text-xs font-mono text-white/30 mt-2">
            <button onClick={() => setShowSettings(true)} className="underline hover:text-white/60 transition-colors">Set your API key</button> or <a href="/account" className="underline hover:text-white/60 transition-colors">create an account</a> to get started
          </p>
        )}
      </div>
    </div>
  );
}

// --- Sub-components ---

function GeminiSpeakerControls({
  label,
  profile,
  onProfile,
  style,
  onStyle,
  pace,
  onPace,
  accent,
  onAccent,
  voice,
  onVoice,
}: {
  label: string;
  profile: string;
  onProfile: (value: string) => void;
  style: string;
  onStyle: (value: string) => void;
  pace: string;
  onPace: (value: string) => void;
  accent: string;
  onAccent: (value: string) => void;
  voice: string;
  onVoice: (value: string) => void;
}) {
  const info = geminiVoiceInfo(voice);
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-[10px] font-mono text-white/50 uppercase tracking-wider">{label}</span>
        {info && <span className="text-[10px] font-mono text-white/30">{info.style} · {info.pitch}</span>}
      </div>
      <label className="block">
        <span className="text-[10px] font-mono text-white/35 uppercase tracking-wider block mb-1">Audio profile</span>
        <input
          value={profile}
          onChange={e => onProfile(e.target.value)}
          className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white placeholder:text-white/25 focus:outline-none focus:border-white/30"
        />
      </label>
      <div className="grid grid-cols-3 gap-2">
        <label className="block">
          <span className="text-[10px] font-mono text-white/35 uppercase tracking-wider block mb-1">Style</span>
          <select value={style} onChange={e => onStyle(e.target.value)} className="w-full bg-black border border-white/10 rounded px-2 py-2 text-xs font-mono text-white focus:outline-none focus:border-white/30">
            {TTS_STYLES.map(v => <option key={v} value={v} className="bg-black text-white">{v}</option>)}
          </select>
        </label>
        <label className="block">
          <span className="text-[10px] font-mono text-white/35 uppercase tracking-wider block mb-1">Pace</span>
          <select value={pace} onChange={e => onPace(e.target.value)} className="w-full bg-black border border-white/10 rounded px-2 py-2 text-xs font-mono text-white focus:outline-none focus:border-white/30">
            {TTS_PACES.map(v => <option key={v} value={v} className="bg-black text-white">{v}</option>)}
          </select>
        </label>
        <label className="block">
          <span className="text-[10px] font-mono text-white/35 uppercase tracking-wider block mb-1">Accent</span>
          <select value={accent} onChange={e => onAccent(e.target.value)} className="w-full bg-black border border-white/10 rounded px-2 py-2 text-xs font-mono text-white focus:outline-none focus:border-white/30">
            {TTS_ACCENTS.map(v => <option key={v} value={v} className="bg-black text-white">{v}</option>)}
          </select>
        </label>
      </div>
      <label className="block">
        <span className="text-[10px] font-mono text-white/35 uppercase tracking-wider block mb-1">Voice</span>
        <input
          list="gemini-tts-voices"
          value={voice}
          onChange={e => onVoice(e.target.value)}
          placeholder="Search voices"
          className="w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white placeholder:text-white/25 focus:outline-none focus:border-white/30"
        />
      </label>
    </div>
  );
}

function EmptyState({ onPrompt, hasApiKey, isImage, isSpeech, isGeminiSpeech, isMusic, imageDemo, videoDemo }: { onPrompt: (text: string) => void; hasApiKey: boolean; isImage?: boolean; isSpeech?: boolean; isGeminiSpeech?: boolean; isMusic?: boolean; imageDemo?: ImageDemo; videoDemo?: VideoDemo }) {
  const fallback = isMusic ? MUSIC_QUICK_PROMPTS : isGeminiSpeech ? GEMINI_TTS_QUICK_PROMPTS : isSpeech ? SPEECH_QUICK_PROMPTS : isImage ? IMAGE_QUICK_PROMPTS : QUICK_PROMPTS;
  const libType = isImage ? 'image' : isMusic ? 'music' : 'text';

  const [libPrompts, setLibPrompts] = useState<LibraryPrompt[]>([]);
  const [saved, setSaved] = useState<SavedPrompt[]>([]);

  useEffect(() => {
    let cancelled = false;
    fetchPrompts({ type: libType, limit: 24 }).then(res => {
      if (!cancelled) setLibPrompts(res.results);
    });
    setSaved(loadSavedPrompts());
    return () => {
      cancelled = true;
    };
  }, [libType]);

  // Library examples take priority; fall back to the bundled quick prompts.
  const examples: { label: string; text: string }[] = libPrompts.length
    ? libPrompts.map(p => ({ label: p.title, text: p.prompt }))
    : fallback.map(p => ({ label: promptExampleLabel(p), text: promptExampleText(p) }));

  return (
    <div className="h-full flex flex-col items-center justify-center px-6 py-12">
      <div className="text-white/10 mb-6">
        <Zap className="w-10 h-10" />
      </div>
      {hasApiKey ? (
        <>
          {imageDemo && (
            <div className="w-full max-w-sm mb-6 rounded-xl overflow-hidden border border-white/10 bg-black/40">
              <img src={imageDemo.outputUrl} alt="Verified HiDream generated sample" className="w-full h-auto block" />
              <button
                onClick={() => onPrompt(imageDemo.prompt)}
                className="w-full px-3 py-2 text-left text-xs font-mono text-white/45 hover:text-white/75 border-t border-white/10 transition-colors"
              >
                Run this verified prompt
              </button>
            </div>
          )}
          {videoDemo && (
            <div className="w-full max-w-lg mb-6 rounded-xl overflow-hidden border border-white/10 bg-black/40">
              <video src={videoDemo.outputUrl} controls muted loop playsInline className="w-full h-auto block" />
              <button
                onClick={() => onPrompt(videoDemo.prompt)}
                className="w-full px-3 py-2 text-left text-xs font-mono text-white/45 hover:text-white/75 border-t border-white/10 transition-colors"
              >
                Run this verified prompt
              </button>
            </div>
          )}
          <p className="text-sm font-mono text-white/20 mb-6">
            {imageDemo || videoDemo ? 'Verified sample output is loaded' : isMusic ? 'Describe a song or clip to generate' : isSpeech ? 'Enter text to synthesize' : isImage ? 'Describe an image to generate' : 'Try a prompt to get started'}
          </p>
          {!imageDemo && !videoDemo && (
            <div className="w-full max-w-md flex flex-col items-center gap-3">
              {/* Examples select — sourced from the prompt library */}
              <select
                value=""
                onChange={e => {
                  const ex = examples[Number(e.target.value)];
                  if (ex) onPrompt(ex.text);
                }}
                data-testid="example-select"
                className="w-full bg-black border border-white/10 rounded-lg px-3 py-2 text-xs font-mono text-white/60 focus:outline-none focus:border-white/30"
              >
                <option value="" disabled>Load an example prompt…</option>
                {examples.map((ex, i) => (
                  <option key={i} value={i}>{ex.label}</option>
                ))}
              </select>

              <div className="flex flex-wrap gap-2 justify-center">
                {examples.slice(0, 6).map((ex, i) => (
                  <button
                    key={i}
                    onClick={() => onPrompt(ex.text)}
                    className="text-xs font-mono text-white/40 border border-white/10 rounded-lg px-3 py-2 hover:border-white/25 hover:text-white/60 hover:bg-white/[0.02] transition-colors text-left"
                  >
                    {ex.label}
                  </button>
                ))}
              </div>

              {saved.length > 0 && (
                <div className="w-full">
                  <p className="text-[11px] font-mono uppercase tracking-wide text-white/25 mb-1.5">Saved prompts</p>
                  <div className="flex flex-wrap gap-2 justify-center">
                    {saved.slice(0, 8).map(sp => (
                      <span key={sp.id} className="group inline-flex items-center gap-1 text-xs font-mono text-white/45 border border-white/10 rounded-lg pl-3 pr-1.5 py-2 hover:border-white/25">
                        <button onClick={() => onPrompt(sp.prompt)} className="hover:text-white/70">{sp.title}</button>
                        <button
                          onClick={() => setSaved(removeSavedPrompt(sp.id))}
                          className="text-white/20 hover:text-white/60"
                          title="Remove saved prompt"
                        >
                          <X className="w-3 h-3" />
                        </button>
                      </span>
                    ))}
                  </div>
                </div>
              )}

              <Link to={`/prompts${libType === 'text' ? '' : `/type/${libType}`}`} className="text-[11px] font-mono text-white/30 hover:text-white/60 transition-colors">
                Browse the full prompt library →
              </Link>
            </div>
          )}
        </>
      ) : (
        <>
          <p className="text-sm font-mono text-white/60 mb-2">Sign in to start comparing models</p>
          <p className="text-xs font-mono text-white/30 mb-6 max-w-sm text-center leading-relaxed">
            One API key for every major model — GPT, Claude, Gemini, Grok, DeepSeek, Llama, and more.
          </p>
          <div className="flex gap-2">
            <a
              href="/account"
              className="text-xs font-mono bg-white text-black rounded-lg px-4 py-2 font-bold hover:bg-white/90 transition-colors"
            >
              Sign in / sign up
            </a>
            <a
              href="/models"
              className="text-xs font-mono text-white/50 border border-white/10 rounded-lg px-4 py-2 hover:text-white hover:border-white/25 transition-colors"
            >
              Browse models
            </a>
          </div>
        </>
      )}
    </div>
  );
}

// Saves the current composer text to the local "saved prompts" list, surfaced
// back in the EmptyState examples and the prompt library.
function SavePromptButton({ text, modelId, modality }: { text: string; modelId?: string; modality?: string }) {
  const [done, setDone] = useState(false);
  const disabled = !text.trim();
  return (
    <button
      onClick={() => {
        if (disabled) return;
        const title = (text.trim().split('\n')[0] || 'Saved prompt').slice(0, 48);
        savePrompt({ title, prompt: text.trim(), modelId, modality });
        setDone(true);
        window.setTimeout(() => setDone(false), 1600);
      }}
      disabled={disabled}
      data-testid="chat-save-prompt"
      title="Save this prompt"
      className="border border-white/10 text-white/40 px-3 py-3 rounded-lg hover:text-white/70 hover:border-white/25 transition-colors disabled:opacity-30 disabled:cursor-not-allowed shrink-0"
    >
      {done ? <BookmarkCheck className="w-4 h-4" /> : <Bookmark className="w-4 h-4" />}
    </button>
  );
}

function MessageBubble({
  message,
  streaming = false,
  isLast = false,
  onRetry,
  onDelete,
}: {
  message: Message;
  streaming?: boolean;
  isLast?: boolean;
  onRetry?: () => void;
  onDelete?: () => void;
  key?: React.Key;
}) {
  const [copied, setCopied] = useState(false);
  const isUser = message.role === 'user';
  const imageSrc = message.imageB64
    ? `data:image/png;base64,${message.imageB64}`
    : message.imageUrl || null;
  const audioSrc = message.audioB64
    ? `data:audio/${message.audioFormat || 'mpeg'};base64,${message.audioB64}`
    : message.audioUrl || null;
  // Hide actions on the message that's still streaming in.
  const isStreamingThis = streaming && isLast && !isUser;
  const hasCopyable = !!message.content;

  const copy = () => {
    navigator.clipboard.writeText(message.content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  // Action bar: visible by default on touch (no hover), reveal on hover for
  // pointer devices. Keeps Retry/Delete reachable on mobile.
  const actions = !isStreamingThis && (onRetry || onDelete || hasCopyable) ? (
    <div
      className={`flex items-center gap-0.5 pt-1 opacity-100 transition-opacity sm:opacity-0 sm:group-hover:opacity-100 ${isUser ? 'justify-end' : ''}`}
    >
      {hasCopyable && (
        <button
          onClick={copy}
          className="p-1.5 rounded-md text-white/30 hover:text-white hover:bg-white/10 transition-colors"
          title="Copy message"
          aria-label="Copy message"
          data-testid="msg-copy"
        >
          {copied ? <Check className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
        </button>
      )}
      {onRetry && (
        <button
          onClick={onRetry}
          disabled={streaming}
          className="p-1.5 rounded-md text-white/30 hover:text-white hover:bg-white/10 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
          title={isUser ? 'Resend from here' : 'Regenerate response'}
          aria-label={isUser ? 'Resend from here' : 'Regenerate response'}
          data-testid="msg-retry"
        >
          <RotateCcw className="w-3.5 h-3.5" />
        </button>
      )}
      {onDelete && (
        <button
          onClick={onDelete}
          disabled={streaming}
          className="p-1.5 rounded-md text-white/30 hover:text-red-400 hover:bg-red-400/10 transition-colors disabled:opacity-30 disabled:cursor-not-allowed"
          title="Delete message"
          aria-label="Delete message"
          data-testid="msg-delete"
        >
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      )}
    </div>
  ) : null;

  return (
    <div className={`group ${isUser ? 'flex flex-col items-end' : ''}`}>
      <div className={`${isUser ? 'max-w-[85%]' : 'w-full'}`}>
        {isUser ? (
          <div className="bg-white/10 rounded-2xl rounded-br-sm px-4 py-2.5 text-sm leading-relaxed whitespace-pre-wrap break-words">
            {message.content}
          </div>
        ) : (
          <div className="text-sm leading-relaxed text-white/90 space-y-3">
            {imageSrc && (
              <a
                href={imageSrc}
                target="_blank"
                rel="noopener noreferrer"
                className="block rounded-xl overflow-hidden border border-white/10 bg-black/40 max-w-lg"
              >
                <img src={imageSrc} alt={message.content || 'Generated image'} className="w-full h-auto block" />
              </a>
            )}
            {message.videoUrl && (
              <div className="rounded-xl overflow-hidden border border-white/10 bg-black/40 max-w-2xl">
                <video src={message.videoUrl} controls className="w-full h-auto block" />
                <a href={message.videoUrl} target="_blank" rel="noopener noreferrer" className="block px-3 py-2 text-xs font-mono text-white/40 hover:text-white/70 border-t border-white/10">
                  Open video
                </a>
              </div>
            )}
            {audioSrc && (
              <div className="rounded-xl border border-white/10 bg-black/40 max-w-xl p-3">
                <div className="flex items-center gap-2 mb-2 text-xs font-mono text-white/45">
                  <Volume2 className="w-3.5 h-3.5" /> Generated speech
                </div>
                <audio src={audioSrc} controls className="w-full" />
              </div>
            )}
            {message.content && renderMarkdown(message.content)}
          </div>
        )}
      </div>
      {actions}
    </div>
  );
}

function ModelSelect({ value, onChange, models }: { value: string; onChange: (v: string) => void; models: CatalogModel[] }) {
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState('');
  const ref = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, []);

  useEffect(() => {
    if (open) {
      setSearch('');
      setTimeout(() => searchRef.current?.focus(), 50);
    }
  }, [open]);

  const current = models.find(m => m.id === value);

  const filtered = useMemo(() => {
    if (!search) return models;
    const q = search.toLowerCase();
    return models.filter(m => m.label.toLowerCase().includes(q) || m.provider.toLowerCase().includes(q) || m.id.toLowerCase().includes(q));
  }, [search, models]);

  const grouped: Record<string, CatalogModel[]> = useMemo(() => {
    return filtered.reduce<Record<string, CatalogModel[]>>((acc, m) => {
      (acc[m.provider] ||= []).push(m);
      return acc;
    }, {});
  }, [filtered]);

  return (
    <div ref={ref} className="relative min-w-0">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1 text-xs font-mono text-white/80 hover:text-white truncate"
      >
        <span className="truncate">{current?.label || value}</span>
        <ChevronDown className={`w-3 h-3 shrink-0 text-white/30 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>
      {open && (
        <div className="absolute top-full left-0 mt-1 w-96 max-h-96 overflow-hidden bg-black border border-white/10 rounded-lg shadow-xl z-50 flex flex-col">
          <div className="p-2 border-b border-white/10">
            <input
              ref={searchRef}
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search models..."
              className="w-full bg-white/5 border-none rounded px-2.5 py-1.5 text-xs font-mono text-white placeholder:text-white/20 focus:outline-none"
            />
          </div>
          <div className="overflow-y-auto flex-1">
            {Object.keys(grouped).length === 0 ? (
              <div className="px-3 py-4 text-xs font-mono text-white/30 text-center">No models found</div>
            ) : (
              Object.entries(grouped).map(([provider, provModels]) => (
                <div key={provider}>
                  <div className="px-3 py-1.5 text-[10px] font-mono text-white/30 uppercase tracking-wider sticky top-0 bg-black">
                    {provider}
                  </div>
                  {provModels.map(m => {
                    const pin = m.pricing?.input_per_1m_tokens;
                    const pout = m.pricing?.output_per_1m_tokens;
                    const speech = isSpeechModel(m);
                    return (
                      <button
                        key={m.id}
                        onClick={() => { onChange(m.id); setOpen(false); }}
                        className={`w-full text-left px-3 py-1.5 text-xs font-mono hover:bg-white/5 transition-colors flex items-center gap-2 ${m.id === value ? 'text-white bg-white/5' : 'text-white/70'}`}
                      >
                        <span className="truncate flex-1">{m.label}</span>
                        <span className="flex items-center gap-1 shrink-0 text-white/30">
                          {m.capabilities?.vision && <Eye className="w-3 h-3" aria-label="vision" />}
                          {m.capabilities?.tools && <Wrench className="w-3 h-3" aria-label="tools" />}
                        </span>
                        {(pin !== undefined || pout !== undefined) && (
                          <span className="text-[10px] font-mono text-white/30 shrink-0 tabular-nums">
                            {speech ? `$${pin?.toFixed(2) ?? '?'}/chars` : `$${pin?.toFixed(2) ?? '?'}/${pout?.toFixed(2) ?? '?'}`}
                          </span>
                        )}
                      </button>
                    );
                  })}
                </div>
              ))
            )}
          </div>
          <div className="px-3 py-1.5 border-t border-white/10 bg-white/[0.02]">
            <p className="text-[9px] font-mono text-white/25">Token models show $ per 1M input / output tokens. Speech shows $ per 1M characters.</p>
          </div>
        </div>
      )}
    </div>
  );
}

function ScrollAnchor({ messages, streaming }: { messages: Message[]; streaming: boolean }) {
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    ref.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages.length, messages[messages.length - 1]?.content, streaming]);
  return <div ref={ref} />;
}
