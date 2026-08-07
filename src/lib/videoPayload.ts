// Single source of truth for building a /v1/videos/generations request body.
// Used by the playground (live request + code snippet) and the model space page.
import {
  getVideoParamSpec,
  type VideoAdvancedValues,
  type VideoParamSpec,
} from '../data/videoModelParams';

export type VideoBaseValues = {
  prompt: string;
  resolution: string;
  duration: string;
  aspectRatio: string;
  generateAudio: boolean;
  imageUrl?: string;
  endImageUrl?: string;
  videoUrl?: string;
  imageUrls?: string[];
  videoUrls?: string[];
  audioUrls?: string[];
};

function num(v: string): number | undefined {
  const t = v.trim();
  if (t === '') return undefined;
  const n = Number(t);
  return Number.isFinite(n) ? n : undefined;
}

export function buildVideoPayload(
  modelId: string,
  base: VideoBaseValues,
  advanced: Partial<VideoAdvancedValues> = {},
  spec: VideoParamSpec = getVideoParamSpec(modelId),
): Record<string, unknown> {
  const body: Record<string, unknown> = {
    model: modelId,
    prompt: base.prompt,
    resolution: base.resolution,
    duration: base.duration,
    aspect_ratio: base.aspectRatio,
  };

  if (spec.enableSafetyChecker) {
    body.duration = base.duration === 'auto' ? 5 : Number(base.duration) || 5;
    body.enable_safety_checker = true;
  } else {
    body.generate_audio = base.generateAudio;
  }

  if (spec.safetyTolerance !== undefined) body.safety_tolerance = spec.safetyTolerance;

  if (spec.inputModes) {
    if (base.imageUrl?.trim()) body.image_url = base.imageUrl.trim();
    if (base.imageUrl?.trim() && base.endImageUrl?.trim()) body.end_image_url = base.endImageUrl.trim();
    if (base.videoUrl?.trim()) body.video_url = base.videoUrl.trim();
  }

  if (!spec.inputModes && spec.imageToVideo) {
    if (base.imageUrl?.trim()) body.image_url = base.imageUrl.trim();
    if (spec.endImage && base.endImageUrl?.trim()) body.end_image_url = base.endImageUrl.trim();
  }
  if (!spec.inputModes && spec.referenceToVideo) {
    if (base.imageUrls?.length) body.image_urls = base.imageUrls;
    if (base.videoUrls?.length) body.video_urls = base.videoUrls;
    if (base.audioUrls?.length) body.audio_urls = base.audioUrls;
  }

  // Advanced args: only included when supported by the model AND set by the user.
  const a = advanced;
  if (spec.negativePrompt && a.negativePrompt?.trim()) body.negative_prompt = a.negativePrompt.trim();
  if (spec.seed && a.seed !== undefined) {
    const s = num(a.seed);
    if (s !== undefined) body.seed = Math.trunc(s);
  }
  if (spec.numFrames && a.numFrames !== undefined) {
    const n = num(a.numFrames);
    if (n !== undefined) body.num_frames = Math.trunc(n);
  }
  if (spec.framesPerSecond && a.framesPerSecond !== undefined) {
    const n = num(a.framesPerSecond);
    if (n !== undefined) body.frames_per_second = Math.trunc(n);
  }
  if (spec.guidanceScale && a.guidanceScale !== undefined) {
    const n = num(a.guidanceScale);
    if (n !== undefined) body.guidance_scale = n;
  }
  if (spec.numInferenceSteps && a.numInferenceSteps !== undefined) {
    const n = num(a.numInferenceSteps);
    if (n !== undefined) body.num_inference_steps = Math.trunc(n);
  }
  if (spec.outputFormats && a.outputFormat?.trim()) body.output_format = a.outputFormat.trim();

  return body;
}
