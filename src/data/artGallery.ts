export interface ArtGalleryItem {
  slug: string;
  provider: string;
  model: string;
  providerModelId: string;
  title: string;
  prompt: string;
  imageUrl: string;
}

export const artGallery: ArtGalleryItem[] = [
  {
    slug: 'fal-klein-archive-chapel',
    provider: 'Fal',
    model: 'FLUX Klein 4B',
    providerModelId: 'fal-ai/flux-2/klein/4b/base',
    title: 'Archive Chapel',
    prompt: 'A brutalist chapel grown from translucent salt crystals on the edge of a frozen black lake, thin dawn light, long shadows, hyper-detailed editorial architecture photography, no people',
    imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/landing/art-playground/fal/fal-klein-archive-chapel.webp',
  },
  {
    slug: 'fal-schnell-gravity-orchard',
    provider: 'Fal',
    model: 'FLUX Schnell',
    providerModelId: 'fal-ai/flux/schnell',
    title: 'Gravity Orchard',
    prompt: 'An orchard where every fruit is a tiny moon with its own orbit, farmers on ladders harvesting by magnetic lantern light, surreal realism, cinematic depth, atmospheric fog',
    imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/landing/art-playground/fal/fal-schnell-gravity-orchard.webp',
  },
  {
    slug: 'fal-dev-paper-nautilus',
    provider: 'Fal',
    model: 'FLUX Dev',
    providerModelId: 'fal-ai/flux/dev',
    title: 'Paper Nautilus',
    prompt: 'A giant nautilus made of folded maps and train tickets swimming through a dry museum atrium, sunbeams, dust particles, whimsical but physically grounded, museum-grade photography',
    imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/landing/art-playground/fal/fal-dev-paper-nautilus.webp',
  },
  {
    slug: 'fal-pro-monsoon-teahouse',
    provider: 'Fal',
    model: 'FLUX Pro 1.1',
    providerModelId: 'fal-ai/flux-pro/v1.1',
    title: 'Monsoon Teahouse',
    prompt: 'A floating teahouse drifting through a tropical monsoon above a neon city, rain slanting sideways, paper lamps glowing amber, cinematic illustration with rich texture',
    imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/landing/art-playground/fal/fal-pro-monsoon-teahouse.webp',
  },
  {
    slug: 'together-sd3-ocean-typewriter',
    provider: 'Together AI',
    model: 'Stable Diffusion 3 Medium',
    providerModelId: 'stabilityai/stable-diffusion-3-medium',
    title: 'Ocean Typewriter',
    prompt: 'A vintage typewriter resting on the seafloor, each keypress releasing schools of silver fish shaped like punctuation marks, teal water, volumetric light, dreamlike macro photography',
    imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/landing/art-playground/together-ai/together-sd3-ocean-typewriter.webp',
  },
  {
    slug: 'zai-glm-clockmaker-desert',
    provider: 'Z.AI',
    model: 'GLM Image',
    providerModelId: 'glm-image',
    title: 'Clockmaker Desert',
    prompt: 'A desert workshop where watch gears are half-buried like fossils and a lone mechanic tunes time with copper tools, warm dusk palette, tactile realism, intricate details',
    imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/landing/art-playground/z-ai/zai-glm-clockmaker-desert.webp',
  },
  {
    slug: 'ra1-archive-chapel',
    provider: 'Netwrck',
    model: 'RA1',
    providerModelId: 'ra1',
    title: 'Archive Chapel',
    prompt: 'A brutalist chapel grown from translucent salt crystals on the edge of a frozen black lake, thin dawn light, long shadows, hyper-detailed editorial architecture photography, no people',
    imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/landing/art-playground/netwrck/ra1-archive-chapel.webp',
  },
  {
    slug: 'zimage-lantern-koi-station',
    provider: 'Netwrck',
    model: 'ZImage',
    providerModelId: 'zimage',
    title: 'Lantern Koi Station',
    prompt: 'Anime illustration of a quiet train platform floating above a koi pond at blue hour, paper lanterns reflected in the water, detailed character silhouette waiting with a satchel, cinematic composition, no text',
    imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/landing/art-playground/netwrck/zimage-lantern-koi-station.webp',
  },
];
