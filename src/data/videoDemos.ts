export const SEEDANCE_LOGO_URL = 'https://openpathsstatic.openpaths.io/static/uploads/playground/seedance/openpaths-logo.webp';
export const HAPPY_HORSE_RAP_IMAGE_URL = 'https://openpathsstatic.openpaths.io/static/uploads/playground/happy-horse/rap.png';

export type VideoDemo = {
  prompt: string;
  outputUrl: string;
  resolution: '480p' | '720p' | '1080p';
  duration: 'auto' | '4' | '5' | '6' | '7' | '8' | '9' | '10' | '11' | '12' | '13' | '14' | '15';
  aspectRatio: 'auto' | '21:9' | '16:9' | '4:3' | '1:1' | '3:4' | '9:16';
  generateAudio: boolean;
  imageUrl?: string;
  endImageUrl?: string;
  imageUrls?: string[];
  videoUrls?: string[];
  audioUrls?: string[];
};

export const VIDEO_DEMOS: Record<string, VideoDemo> = {
  'seedance-2.0-fast-text-to-video': {
    prompt: 'A cinematic 4-second shot of a compact AI routing console on a dark workstation, luminous paths connecting model nodes across a glass interface, slow handheld push-in, realistic reflections, premium product demo lighting, no readable text.',
    outputUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/seedance/seedance-fast-text-to-video.mp4',
    resolution: '720p',
    duration: '4',
    aspectRatio: '16:9',
    generateAudio: false,
  },
  'seedance-2.0-text-to-video': {
    prompt: 'A polished studio macro shot of an AI infrastructure dashboard represented as glowing fiber-optic routes inside a transparent cube, slow orbiting camera, cinematic depth of field, clean black background, no readable text.',
    outputUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/seedance/seedance-text-to-video.mp4',
    resolution: '720p',
    duration: '4',
    aspectRatio: '16:9',
    generateAudio: false,
  },
  'seedance-2.0-image-to-video': {
    prompt: 'Animate the supplied OpenPaths logo as a premium product mark: subtle camera push-in, soft light sweep across the surface, tiny particles moving around it, clean dark studio background, elegant motion, no added text.',
    outputUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/seedance/seedance-image-to-video.mp4',
    resolution: '720p',
    duration: '4',
    aspectRatio: '1:1',
    generateAudio: false,
    imageUrl: SEEDANCE_LOGO_URL,
  },
  'seedance-2.0-fast-reference-to-video': {
    prompt: 'Use @Image1 as the exact brand mark on a small illuminated badge mounted to a matte black server rack. Slow dolly-in, shallow depth of field, cool white rim light, subtle cable movement, premium infrastructure commercial, no extra text.',
    outputUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/seedance/seedance-fast-reference-to-video.mp4',
    resolution: '720p',
    duration: '4',
    aspectRatio: '16:9',
    generateAudio: false,
    imageUrls: [SEEDANCE_LOGO_URL],
    audioUrls: [],
  },
  'seedance-2.0-reference-to-video': {
    prompt: '@Image1 is projected as a crisp holographic interface element above a developer desk. Camera slides left to right, soft reflections on glass, realistic workstation lighting, cinematic product demo, no additional words or watermarks.',
    outputUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/seedance/seedance-reference-to-video.mp4',
    resolution: '720p',
    duration: '4',
    aspectRatio: '16:9',
    generateAudio: false,
    imageUrls: [SEEDANCE_LOGO_URL],
    audioUrls: [],
  },
  'alibaba/happy-horse/image-to-video': {
    prompt: 'Bring the scene in the image to life.',
    outputUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/happy-horse/happy-horse-image-to-video.mp4',
    resolution: '1080p',
    duration: '5',
    aspectRatio: '16:9',
    generateAudio: true,
    imageUrl: HAPPY_HORSE_RAP_IMAGE_URL,
  },
  'ltx-2.3-image-to-video': {
    prompt: 'A polished real-estate listing still becomes a smooth slow zoom-in video, subtle parallax, stable architecture, natural lighting, no text overlays.',
    outputUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/happy-horse/happy-horse-image-to-video.mp4',
    resolution: '1080p',
    duration: '6',
    aspectRatio: 'auto',
    generateAudio: false,
    imageUrl: SEEDANCE_LOGO_URL,
  },
};
