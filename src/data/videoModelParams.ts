// Per-model fal video argument specs. Declares which optional args each video
// model exposes in the UI + their ranges/options. Advanced args are opt-in:
// they are only sent in the request when the user provides a value, so default
// behavior is unchanged for every model.

export type NumRange = { min: number; max: number; step?: number; placeholder?: string };
export type VideoInputMode = 'text-to-video' | 'image-to-video' | 'video-to-video';

export type VideoParamSpec = {
  resolutions: string[];
  durations: string[];
  aspectRatios: string[];
  generateAudio: boolean; // audio toggle supported (false => hidden / forced)
  enableSafetyChecker: boolean; // forces enable_safety_checker:true (happy-horse)
  safetyTolerance?: number; // fixed provider value included in the request
  // input modes
  inputModes?: VideoInputMode[];
  imageToVideo: boolean;
  endImage: boolean;
  referenceToVideo: boolean;
  // advanced optional args (opt-in)
  negativePrompt: boolean;
  seed: boolean;
  numFrames?: NumRange;
  framesPerSecond?: { options: number[] };
  guidanceScale?: NumRange;
  numInferenceSteps?: NumRange;
  outputFormats?: string[];
};

export const VIDEO_RESOLUTIONS = ['480p', '720p', '1080p'];
export const VIDEO_DURATIONS = ['auto', '4', '5', '6', '7', '8', '9', '10', '11', '12', '13', '14', '15'];
export const VIDEO_ASPECT_RATIOS = ['auto', '21:9', '16:9', '4:3', '1:1', '3:4', '9:16'];
export const VIDEO_OUTPUT_FORMATS = ['mp4', 'webm'];

const BASE: VideoParamSpec = {
  resolutions: VIDEO_RESOLUTIONS,
  durations: VIDEO_DURATIONS,
  aspectRatios: VIDEO_ASPECT_RATIOS,
  generateAudio: true,
  enableSafetyChecker: false,
  imageToVideo: false,
  endImage: false,
  referenceToVideo: false,
  negativePrompt: true,
  seed: true,
  outputFormats: VIDEO_OUTPUT_FORMATS,
};

// Family overrides keyed by a detector. First match wins (order matters).
const FAMILIES: Array<{ test: RegExp; spec: Partial<VideoParamSpec> }> = [
  {
    // Wan 3.0: resolution tiers 480p-1080p, smart or fixed 2-30s duration,
    // native audio. No negative prompt / guidance / steps in its schema.
    test: /^wan-3\.0/i,
    spec: {
      resolutions: ['480p', '720p', '1080p'],
      durations: ['auto', ...Array.from({ length: 29 }, (_, index) => String(index + 2))],
      aspectRatios: ['auto', '16:9', '4:3', '1:1', '3:4', '9:16'],
      negativePrompt: false,
      seed: true,
      outputFormats: undefined,
    },
  },
  {
    test: /^flux-3-video-draft$/i,
    spec: {
      resolutions: ['HD'],
      durations: ['auto', ...Array.from({ length: 16 }, (_, index) => String(index + 5))],
      aspectRatios: ['auto', '21:9', '2:1', '16:9', '4:3', '1:1', '3:4', '9:16'],
      inputModes: ['text-to-video', 'image-to-video'],
      safetyTolerance: 4,
      negativePrompt: false,
      seed: false,
      outputFormats: undefined,
    },
  },
  {
    test: /^flux-3-video$/i,
    spec: {
      resolutions: ['HD', 'FHD'],
      durations: ['auto', ...Array.from({ length: 16 }, (_, index) => String(index + 5))],
      aspectRatios: ['auto', '21:9', '2:1', '16:9', '4:3', '1:1', '3:4', '9:16'],
      inputModes: ['text-to-video', 'image-to-video', 'video-to-video'],
      safetyTolerance: 4,
      negativePrompt: false,
      seed: false,
      outputFormats: undefined,
    },
  },
  {
    test: /^minimax-h3$/i,
    spec: {
      resolutions: ['768P', '2K'],
      durations: ['5', '6', '7', '8', '9', '10', '11', '12', '13', '14', '15'],
      aspectRatios: ['16:9', '21:9', '4:3', '1:1', '3:4', '9:16'],
      generateAudio: false,
      negativePrompt: false,
      seed: false,
      outputFormats: ['mp4', 'webm'],
    },
  },
  {
    test: /^minimax-h3-max$/i,
    spec: {
      resolutions: ['480P', '768P'],
      durations: ['5', '6', '7', '8', '9', '10', '11', '12', '13', '14', '15'],
      aspectRatios: ['16:9', '21:9', '4:3', '1:1', '3:4', '9:16'],
      inputModes: ['text-to-video', 'image-to-video'],
      generateAudio: false,
      negativePrompt: false,
      seed: true,
      outputFormats: ['mp4', 'webm'],
    },
  },
  {
    test: /^minimax-h3-max-image-to-video$/i,
    spec: {
      resolutions: ['480P', '768P'],
      durations: ['5', '6', '7', '8', '9', '10', '11', '12', '13', '14', '15'],
      aspectRatios: ['16:9'],
      generateAudio: false,
      negativePrompt: false,
      seed: true,
      outputFormats: ['mp4', 'webm'],
    },
  },
  {
    test: /gemini-omni/i,
    spec: {
      resolutions: ['720p'],
      durations: ['3', '4', '5', '6', '7', '8', '9', '10'],
      aspectRatios: ['16:9'],
      generateAudio: false,
      negativePrompt: false,
      seed: false,
      outputFormats: ['mp4'],
    },
  },
  {
    // Open-weight LTX: exposes frame-level + diffusion controls.
    test: /ltx/i,
    spec: {
      numFrames: { min: 9, max: 161, step: 8, placeholder: '121' },
      framesPerSecond: { options: [24, 25, 30] },
      guidanceScale: { min: 1, max: 10, step: 0.5, placeholder: '3' },
      numInferenceSteps: { min: 10, max: 50, placeholder: '30' },
    },
  },
  {
    test: /wan\//i,
    spec: {
      guidanceScale: { min: 1, max: 10, step: 0.5, placeholder: '5' },
      numInferenceSteps: { min: 10, max: 50, placeholder: '30' },
    },
  },
  {
    test: /^flux-video-upscale$/i,
    spec: {
      inputModes: ['video-to-video'],
      resolutions: ['auto'],
      durations: ['auto'],
      aspectRatios: ['auto'],
      generateAudio: false,
      negativePrompt: false,
      seed: false,
      outputFormats: ['mp4'],
    },
  },
  {
    test: /hunyuan/i,
    spec: {
      guidanceScale: { min: 1, max: 10, step: 0.5, placeholder: '6' },
      numInferenceSteps: { min: 10, max: 50, placeholder: '30' },
    },
  },
  {
    test: /cogvideo/i,
    spec: {
      guidanceScale: { min: 1, max: 10, step: 0.5, placeholder: '7' },
      numInferenceSteps: { min: 10, max: 50, placeholder: '50' },
    },
  },
];

function detectInputs(modelId: string): Pick<VideoParamSpec, 'imageToVideo' | 'endImage' | 'referenceToVideo'> {
  const isImageToVideo = /image-to-video|i2v/i.test(modelId);
  const isReference = /reference-to-video|reference/i.test(modelId);
  // Image-to-video models accept an optional end frame.
  return { imageToVideo: isImageToVideo, endImage: isImageToVideo, referenceToVideo: isReference };
}

export function getVideoParamSpec(modelId: string): VideoParamSpec {
  const family = FAMILIES.find(f => f.test.test(modelId));
  const isHappyHorse = modelId === 'alibaba/happy-horse/image-to-video';
  return {
    ...BASE,
    ...detectInputs(modelId),
    // Happy Horse forces the safety checker and always emits audio.
    enableSafetyChecker: isHappyHorse,
    generateAudio: !isHappyHorse,
    ...(family?.spec ?? {}),
  };
}

// Advanced arg values entered in the UI. Empty string => unset (not sent).
export type VideoAdvancedValues = {
  negativePrompt: string;
  seed: string;
  numFrames: string;
  framesPerSecond: string;
  guidanceScale: string;
  numInferenceSteps: string;
  outputFormat: string;
};

export const EMPTY_ADVANCED: VideoAdvancedValues = {
  negativePrompt: '',
  seed: '',
  numFrames: '',
  framesPerSecond: '',
  guidanceScale: '',
  numInferenceSteps: '',
  outputFormat: '',
};
