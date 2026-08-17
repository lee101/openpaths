export const SEEDANCE_LOGO_URL = 'https://openpaths.io/logo-512.webp';
export const HAPPY_HORSE_RAP_IMAGE_URL = 'https://openpathsstatic.openpaths.io/static/uploads/playground/happy-horse/rap.png';
const HH11_BASE = 'https://openpathsstatic.openpaths.io/static/uploads/playground/happy-horse-v1.1';
export const HAPPY_HORSE_V11_IMAGE_URL = `${HH11_BASE}/rap.png`;
const SYNC_LIPSYNC_BASE = 'https://openpathsstatic.openpaths.io/static/uploads/playground/sync-lipsync';
export const SYNC_LIPSYNC_AVATAR_URL = `${SYNC_LIPSYNC_BASE}/avatar.jpg`;
const FAL_VIDEO_BASE = 'https://openpathsstatic.openpaths.io/static/uploads/playground/fal-video';
const GEMINI_OMNI_BASE = 'https://openpathsstatic.openpaths.io/static/uploads/playground/gemini-omni';
export const GEMINI_OMNI_PERFUME_IMAGE_URL = 'https://openpathsstatic.openpaths.io/static/uploads/playground/hidream-edit/perfume.jpg';

export type VideoDemo = {
  prompt: string;
  outputUrl: string;
  resolution: '480p' | '720p' | '768P' | '1080p' | '2K' | 'HD' | 'FHD';
  duration: 'auto' | '4' | '5' | '6' | '7' | '8' | '9' | '10' | '11' | '12' | '13' | '14' | '15' | '16' | '17' | '18' | '19' | '20';
  aspectRatio: 'auto' | '21:9' | '16:9' | '4:3' | '1:1' | '3:4' | '9:16';
  generateAudio: boolean;
  imageUrl?: string;
  endImageUrl?: string;
  imageUrls?: string[];
  videoUrls?: string[];
  audioUrls?: string[];
};

export const VIDEO_DEMOS: Record<string, VideoDemo> = {
  'flux-3-video-draft': {
    prompt: 'A cinematic macro journey through a miniature Black Forest at night where luminous fiber-optic paths weave between moss-covered stones like an intelligent routing network. The camera glides slowly forward at ground level; cool moonlight, warm bioluminescent pulses, light fog, realistic depth of field. Natural forest ambience and subtle electronic tones, no speech, no readable text, no logos.',
    outputUrl: '/static/video-gallery/bfl/flux-3-routing-forest-draft.webm',
    resolution: 'HD',
    duration: '5',
    aspectRatio: '16:9',
    generateAudio: true,
  },
  'flux-3-video': {
    prompt: 'A precision glass terrarium sits on a dark studio desk, containing a living miniature forest whose glowing root network reroutes pulses of light around fallen branches in real time. Slow cinematic orbit, physically accurate reflections, rich moss detail, cool blue and warm amber lighting, soft mechanical room tone blended with forest ambience, no speech, no readable text, no logos.',
    outputUrl: '/static/video-gallery/bfl/flux-3-routing-terrarium-full.webm',
    resolution: 'HD',
    duration: '5',
    aspectRatio: '16:9',
    generateAudio: true,
  },
  'gemini-omni-flash-preview': {
    prompt: '[# Sources <FIRST_FRAME>@Image1] Continuous single-shot premium product video: the perfume bottle remains the hero object from the input image, the camera makes a slow cinematic push-in, soft studio lights sweep across the glass, subtle reflections move on the surface, no scene cuts, no added text, no dialogue. Use Image1 as the starting frame.',
    outputUrl: `${GEMINI_OMNI_BASE}/gemini-omni-perfume-image-to-video.mp4`,
    resolution: '720p',
    duration: '4',
    aspectRatio: '16:9',
    generateAudio: false,
    imageUrl: GEMINI_OMNI_PERFUME_IMAGE_URL,
  },
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
  'alibaba/happy-horse/v1.1/text-to-video': {
    prompt: 'A cinematic 5-second shot of a luminous AI model-routing console on a dark studio desk, glowing fiber-optic paths connecting model nodes across a glass interface, slow handheld push-in, realistic reflections, premium product lighting, no readable text.',
    outputUrl: `${HH11_BASE}/text-to-video.mp4`,
    resolution: '1080p',
    duration: '5',
    aspectRatio: '16:9',
    generateAudio: true,
  },
  'alibaba/happy-horse/v1.1/image-to-video': {
    prompt: 'Bring the scene in the image to life with natural motion and ambient audio.',
    outputUrl: `${HH11_BASE}/image-to-video.mp4`,
    resolution: '1080p',
    duration: '5',
    aspectRatio: '16:9',
    generateAudio: true,
    imageUrl: HAPPY_HORSE_V11_IMAGE_URL,
  },
  'alibaba/happy-horse/v1.1/reference-to-video': {
    prompt: 'character1 performs an energetic rap on a neon-lit stage, cinematic lighting, smooth camera movement, synchronized lip-sync.',
    outputUrl: `${HH11_BASE}/reference-to-video.mp4`,
    resolution: '1080p',
    duration: '5',
    aspectRatio: '16:9',
    generateAudio: true,
    imageUrls: [HAPPY_HORSE_V11_IMAGE_URL],
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
  'fal-ai/kling-video/v3/pro/text-to-video': {
    prompt: 'A lone astronaut drifts weightless inside a derelict space station, dust motes glittering in shafts of golden sunlight through cracked viewports, slow cinematic dolly push, anamorphic lens flare, volumetric god rays, hyper-detailed.',
    outputUrl: `${FAL_VIDEO_BASE}/kling-v3-pro-text-to-video.mp4`,
    resolution: '1080p',
    duration: '5',
    aspectRatio: '16:9',
    generateAudio: true,
  },
  'fal-ai/wan/v2.7/text-to-video': {
    prompt: 'Cinematic aerial shot soaring over a neon-drenched futuristic megacity at dusk, sleek glass towers reflecting magenta and cyan light, flying vehicles streaking between skyscrapers, volumetric fog, ultra-smooth glide.',
    outputUrl: `${FAL_VIDEO_BASE}/wan-2.7-text-to-video.mp4`,
    resolution: '1080p',
    duration: '5',
    aspectRatio: '16:9',
    generateAudio: true,
  },
  wan: {
    prompt: 'Cinematic aerial shot soaring over a neon-drenched futuristic megacity at dusk, sleek glass towers reflecting magenta and cyan light, flying vehicles streaking between skyscrapers, volumetric fog, ultra-smooth glide.',
    outputUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/manifoldgen/wan-2.7-text-to-video-example.mp4',
    resolution: '720p',
    duration: '5',
    aspectRatio: '16:9',
    generateAudio: false,
  },
  'fal-ai/pika/v2.2/text-to-video': {
    prompt: 'Cinematic dolly through a bioluminescent jungle at night, glowing blue spores drifting in the air, mist curling around ancient vines, a waterfall shimmering in moonlight, ultra-detailed, lush atmospheric color grade.',
    outputUrl: `${FAL_VIDEO_BASE}/pika-2.2-text-to-video.mp4`,
    resolution: '1080p',
    duration: '5',
    aspectRatio: '16:9',
    generateAudio: false,
  },
  'fal-ai/cogvideox-5b': {
    prompt: 'Cinematic tracking shot following a sleek silver sports car carving through a coastal mountain road at sunset, ocean glittering below, lens flare, motion blur, dramatic golden light, ultra-detailed.',
    outputUrl: `${FAL_VIDEO_BASE}/cogvideox-text-to-video.mp4`,
    resolution: '720p',
    duration: '5',
    aspectRatio: '16:9',
    generateAudio: false,
  },
  'fal-ai/hunyuan-video-v1.5/text-to-video': {
    prompt: 'Cinematic slow-motion close-up of molten gold liquid swirling into elegant ribbons against a black void, caustic light refractions, hyper-detailed fluid simulation, dramatic spotlight, luxurious, shallow focus.',
    outputUrl: `${FAL_VIDEO_BASE}/hunyuan-1.5-text-to-video.mp4`,
    resolution: '720p',
    duration: '5',
    aspectRatio: '16:9',
    generateAudio: false,
  },
  'fal-ai/minimax/hailuo-2.3/standard/text-to-video': {
    prompt: 'Sweeping cinematic drone shot gliding over neon-lit futuristic city canyons at dusk, holographic light trails reflecting on glass towers, volumetric fog, dramatic teal-and-amber color grade.',
    outputUrl: `${FAL_VIDEO_BASE}/hailuo-2.3-text-to-video.mp4`,
    resolution: '720p',
    duration: '6',
    aspectRatio: '16:9',
    generateAudio: false,
  },
  'fal-ai/veo3.1/fast/image-to-video': {
    prompt: 'Bring the portrait to life: subtle head turn and blink, gentle breeze in the hair, soft volumetric light drifting across the face, cinematic shallow focus, ambient room tone.',
    outputUrl: `${FAL_VIDEO_BASE}/veo-3.1-fast-image-to-video.mp4`,
    resolution: '1080p',
    duration: '5',
    aspectRatio: '16:9',
    generateAudio: true,
    imageUrl: HAPPY_HORSE_V11_IMAGE_URL,
  },
  'fal-ai/luma-dream-machine/ray-2/image-to-video': {
    prompt: 'The scene comes alive with gentle cinematic motion, soft parallax depth, drifting light and dust particles, photoreal slow push-in.',
    outputUrl: `${FAL_VIDEO_BASE}/luma-ray-2-image-to-video.mp4`,
    resolution: '720p',
    duration: '5',
    aspectRatio: '16:9',
    generateAudio: false,
    imageUrl: HAPPY_HORSE_V11_IMAGE_URL,
  },
  'sync-lipsync-v3': {
    prompt: 'Lip-sync the character in the image to the supplied voice track.',
    outputUrl: `${SYNC_LIPSYNC_BASE}/sync-lipsync-v3-demo.mp4`,
    resolution: '1080p',
    duration: '10',
    aspectRatio: '1:1',
    generateAudio: false,
    imageUrl: SYNC_LIPSYNC_AVATAR_URL,
    audioUrls: [`${SYNC_LIPSYNC_BASE}/voice.wav`],
  },
};
