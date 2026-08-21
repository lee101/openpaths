export type VideoGalleryItem = {
  slug: string;
  title: string;
  provider: string;
  model: string;
  prompt: string;
  videoUrl: string;
  posterUrl?: string;
  originalVideoUrl?: string;
  duration: number;
  resolution: '480p' | '720p' | 'HD' | 'FHD';
  aspectRatio: '16:9' | '9:16' | '1:1' | '4:3' | '3:4';
};

export const videoGallery: VideoGalleryItem[] = [
  {
    slug: "manifoldgen-kfold-lighthouse",
    title: "Obsidian Lighthouse",
    provider: "ManifoldGen",
    model: "K-Fold Video Generator",
    prompt: "An obsidian lighthouse fractures moonlight into spectral fog while black waves climb upward, slow impossible crane shot; sub-bass surf, distant glass harmonics.",
    videoUrl: "https://manifoldgenstatic.manifoldgen.com/gallery/03475ad6-41a/videos/add2e0dd-9f8f-4d6d-b0dc-41a210fecaa3.webm",
    duration: 5,
    resolution: "HD",
    aspectRatio: "16:9",
  },
  {
    slug: "manifoldgen-kfold-cartographer",
    title: "Cartographer in Motion",
    provider: "ManifoldGen",
    model: "K-Fold Video Generator",
    prompt: "A cartographer moves through a luminous paper landscape, ink rivers flowing into new constellations, graceful camera drift, tactile cinematic detail.",
    videoUrl: "https://manifoldgenstatic.manifoldgen.com/gallery/videos/wan_animate_cartographer_standard_5s_20260816.mp4",
    duration: 5,
    resolution: "HD",
    aspectRatio: "16:9",
  },
  {
    slug: "manifoldgen-kfold-temple-spin",
    title: "Temple Chiffon Spin",
    provider: "ManifoldGen",
    model: "K-Fold Video Generator",
    prompt: "A dancer in layered chiffon turns through a sunlit temple, fabric tracing wide arcs through warm dust and quiet beams of light.",
    videoUrl: "https://manifoldgenstatic.manifoldgen.com/gallery/videos/temple-chiffon-spin.webm",
    duration: 5,
    resolution: "HD",
    aspectRatio: "16:9",
  },
  {
    slug: "flux-3-video-routing-forest-draft",
    title: "Routing Forest — Draft",
    provider: "Black Forest Labs",
    model: "FLUX 3 Video Draft",
    prompt: "A cinematic macro journey through a miniature Black Forest at night where luminous fiber-optic paths weave between moss-covered stones like an intelligent routing network. The camera glides slowly forward at ground level; cool moonlight, warm bioluminescent pulses, light fog, realistic depth of field. Natural forest ambience and subtle electronic tones, no speech, no readable text, no logos.",
    videoUrl: "/static/video-gallery/bfl/flux-3-routing-forest-draft.webm",
    posterUrl: "/static/video-gallery/bfl/flux-3-routing-forest-draft-poster.webp",
    duration: 5,
    resolution: "HD",
    aspectRatio: "16:9",
  },
  {
    slug: "flux-3-video-routing-terrarium",
    title: "Living Routing Terrarium",
    provider: "Black Forest Labs",
    model: "FLUX 3 Video",
    prompt: "A precision glass terrarium sits on a dark studio desk, containing a living miniature forest whose glowing root network reroutes pulses of light around fallen branches in real time. Slow cinematic orbit, physically accurate reflections, rich moss detail, cool blue and warm amber lighting, soft mechanical room tone blended with forest ambience, no speech, no readable text, no logos.",
    videoUrl: "/static/video-gallery/bfl/flux-3-routing-terrarium-full.webm",
    posterUrl: "/static/video-gallery/bfl/flux-3-routing-terrarium-full-poster.webp",
    duration: 5,
    resolution: "HD",
    aspectRatio: "16:9",
  },
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
