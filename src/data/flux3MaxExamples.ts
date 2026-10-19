export type Flux3MaxMode = 'text' | 'image' | 'keyframes';

export const FLUX3_MAX_MODEL_ID = 'flux-3-max-video';
export const FLUX3_MAX_RATES = { '720p': 0.17, '1080p': 0.29 } as const;
export const FLUX3_MAX_ASPECTS = ['auto', '21:9', '2:1', '16:9', '4:3', '1:1', '3:4', '9:16'] as const;
export const FLUX3_MAX_FPS = 24;
export const FLUX3_MAX_MAX_KEYFRAMES = 10;
export const FLUX3_MAX_MIN_SECONDS = 5;
export const FLUX3_MAX_MAX_SECONDS = 20;

const BASE = 'https://openpathsstatic.openpaths.io/static/video-gallery/flux-3-max';

export type Flux3MaxExample = {
  prompt: string;
  outputUrl: string;
  posterUrl: string;
  imageUrl?: string;
  keyframes?: { imageUrl: string; frameIndex: number }[];
};

export const FLUX3_MAX_EXAMPLES: Record<Flux3MaxMode, Flux3MaxExample> = {
  text: {
    prompt: 'A red panda walks along a mossy log in a sunlit forest, pausing to sniff the air before climbing over a fallen branch. Soft morning light filters through the canopy. Ambient birdsong and rustling leaves.',
    outputUrl: `${BASE}/text-output.mp4`,
    posterUrl: `${BASE}/text-poster.jpg`,
  },
  image: {
    prompt: 'Slow cinematic push-in toward the lighthouse as waves roll against the rocks and clouds drift across the golden sky. Gulls cry in the distance with the sound of surf.',
    outputUrl: `${BASE}/image-output.mp4`,
    posterUrl: `${BASE}/image-poster.jpg`,
    imageUrl: `${BASE}/image-input.jpg`,
  },
  keyframes: {
    prompt: 'The woman picks up a brown felt fedora, lifts it onto her head with both hands and adjusts the brim, smiling at the camera. Soft studio light against a plain gray wall.',
    outputUrl: `${BASE}/keyframes-output.mp4`,
    posterUrl: `${BASE}/keyframes-poster.jpg`,
    keyframes: [
      { imageUrl: `${BASE}/keyframes-input-1.jpg`, frameIndex: 0 },
      { imageUrl: `${BASE}/keyframes-input-2.jpg`, frameIndex: 120 },
    ],
  },
};
