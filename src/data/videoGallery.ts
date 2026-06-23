export type VideoGalleryItem = {
  slug: string;
  title: string;
  provider: string;
  model: string;
  prompt: string;
  videoUrl: string;
  originalVideoUrl?: string;
  duration: number;
  resolution: '480p' | '720p';
  aspectRatio: '16:9' | '9:16' | '1:1' | '4:3' | '3:4';
};

export const videoGallery: VideoGalleryItem[] = [
  {
    slug: "grok-imagine-video-data-observatory",
    title: "Data Observatory",
    provider: "xAI",
    model: "grok-imagine-video",
    prompt: "A cinematic macro shot of a glass observatory filled with glowing data streams orbiting like constellations, slow dolly forward, realistic reflections, premium AI infrastructure mood, no readable text.",
    videoUrl: "https://openpathsstatic.openpaths.io/static/uploads/landing/video-gallery/grok-imagine-video-data-observatory.webm",
    duration: 6,
    resolution: "480p",
    aspectRatio: "16:9",
  },
  {
    slug: "grok-imagine-video-routing-garden",
    title: "Routing Garden",
    provider: "xAI",
    model: "grok-imagine-video",
    prompt: "A miniature indoor garden where fiber optic vines connect small model cards across black stone, soft rain on glass, gentle camera orbit, elegant product demo lighting, no text or logos.",
    videoUrl: "https://openpathsstatic.openpaths.io/static/uploads/landing/video-gallery/grok-imagine-video-routing-garden.webm",
    duration: 6,
    resolution: "480p",
    aspectRatio: "16:9",
  },
  {
    slug: "grok-imagine-video-agent-workshop",
    title: "Agent Workshop",
    provider: "xAI",
    model: "grok-imagine-video",
    prompt: "A quiet futuristic workshop where autonomous software agents appear as small luminous tools assembling a clean interface in midair, warm practical lights, shallow depth of field, slow push-in, no readable text.",
    videoUrl: "https://openpathsstatic.openpaths.io/static/uploads/landing/video-gallery/grok-imagine-video-agent-workshop.webm",
    duration: 6,
    resolution: "480p",
    aspectRatio: "16:9",
  },
];
