import { ARTIFICIAL_ANALYSIS_IMAGE_SNAPSHOT } from '../data/artificialAnalysisImages';

export interface ImageLeaderboardModel {
  id: string;
  name: string;
  rank: number;
  elo: number | null;
  appearances: number | null;
  winRate: number | null;
  pricePer1kImages: number | null;
  released: string | null;
  openWeights: boolean;
  modelUrl: string | null;
  ciLower: number | null;
  ciUpper: number | null;
  creator: { name: string; logoUrl: string | null; color: string };
}

export const artificialAnalysisImageSnapshot = ARTIFICIAL_ANALYSIS_IMAGE_SNAPSHOT;
export const textToImageLeaderboard = ARTIFICIAL_ANALYSIS_IMAGE_SNAPSHOT.textToImage as readonly unknown[] as ImageLeaderboardModel[];
export const imageEditingLeaderboard = ARTIFICIAL_ANALYSIS_IMAGE_SNAPSHOT.editing as readonly unknown[] as ImageLeaderboardModel[];

// Generators OpenPaths actually serves, with the example image we generated for each
// (single shared prompt) and the substrings used to match them into the AA leaderboard.
// `onLeaderboard: false` means the generator isn't ranked by Artificial Analysis (e.g. RA1).
export interface OpenPathsImageModel {
  slug: string;            // example image filename + internal id
  label: string;
  openpathsId: string;     // model id to call on the OpenPaths API
  provider: string;
  image: string;           // generated example image (absolute CDN URL)
  blurb: string;
  aaMatch: string[];       // substrings that identify this model on the AA leaderboard
  onLeaderboard: boolean;
}

export const OPENPATHS_IMAGE_MODELS: OpenPathsImageModel[] = [
  {
    slug: 'ra1',
    label: 'RA1',
    openpathsId: 'ra1',
    provider: 'Netwrck',
    image: 'https://openpathsstatic.openpaths.io/static/blog/image-eval/ra1.webp',
    blurb: 'Netwrck’s flagship art generator. Policy-flexible, fast, and the OpenPaths fallback when other providers refuse a prompt.',
    aaMatch: [],
    onLeaderboard: false,
  },
  {
    slug: 'zimage',
    label: 'Z-Image Turbo',
    openpathsId: 'zimage',
    provider: 'Netwrck',
    image: 'https://openpathsstatic.openpaths.io/static/blog/image-eval/zimage.webp',
    blurb: 'Stylized/anime-leaning open-weights model. The cheapest generator on OpenPaths at well under a cent per image.',
    aaMatch: ['Z-Image Turbo'],
    onLeaderboard: true,
  },
  {
    slug: 'gpt-image-2',
    label: 'GPT Image 2',
    openpathsId: 'gpt-image-2',
    provider: 'OpenAI',
    image: 'https://openpathsstatic.openpaths.io/static/blog/image-eval/gpt-image-2.webp',
    blurb: 'OpenAI’s top image model and #1 on the Artificial Analysis Text-to-Image Arena - best-in-class prompt following and legible text.',
    aaMatch: ['GPT Image 2'],
    onLeaderboard: true,
  },
  {
    slug: 'flux-pro',
    label: 'FLUX1.1 [pro]',
    openpathsId: 'flux-pro',
    provider: 'Black Forest Labs (fal)',
    image: 'https://openpathsstatic.openpaths.io/static/blog/image-eval/flux-pro.webp',
    blurb: 'Black Forest Labs’ production model - a reliable, photoreal workhorse with strong composition.',
    aaMatch: ['FLUX1.1 [pro]'],
    onLeaderboard: true,
  },
  {
    slug: 'flux-dev',
    label: 'FLUX [dev]',
    openpathsId: 'flux-dev',
    provider: 'Black Forest Labs (fal)',
    image: 'https://openpathsstatic.openpaths.io/static/blog/image-eval/flux-dev.webp',
    blurb: 'The open-weights FLUX dev checkpoint - great quality-per-dollar for everyday generation.',
    aaMatch: ['FLUX.2 [dev]', 'FLUX.1 [dev]'],
    onLeaderboard: true,
  },
  {
    slug: 'klein',
    label: 'FLUX.2 [klein]',
    openpathsId: 'klein',
    provider: 'Black Forest Labs (fal)',
    image: 'https://openpathsstatic.openpaths.io/static/blog/image-eval/klein.webp',
    blurb: 'The small, efficient FLUX.2 variant - fast and cheap while staying surprisingly sharp.',
    aaMatch: ['FLUX.2 [klein]'],
    onLeaderboard: true,
  },
  {
    slug: 'grok-imagine-image',
    label: 'Grok Imagine',
    openpathsId: 'grok-imagine-image',
    provider: 'xAI',
    image: 'https://openpathsstatic.openpaths.io/static/blog/image-eval/grok-imagine-image.webp',
    blurb: 'xAI’s image model - strong on dramatic, stylized scenes and a top-10 Arena finisher.',
    aaMatch: ['grok-imagine-image'],
    onLeaderboard: true,
  },
];

const matchers: Array<{ model: OpenPathsImageModel; needles: string[] }> = OPENPATHS_IMAGE_MODELS
  .filter(model => model.aaMatch.length > 0)
  .map(model => ({ model, needles: model.aaMatch.map(s => s.toLowerCase()) }));

/** Returns the OpenPaths generator that backs a given AA leaderboard row, if we serve it. */
export function matchOpenPathsModel(aaName: string): OpenPathsImageModel | undefined {
  const name = aaName.toLowerCase();
  return matchers.find(m => m.needles.some(needle => name.includes(needle)))?.model;
}

export function displayElo(value: number | null | undefined): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return 'N/A';
  return Math.round(value).toLocaleString('en-US');
}

export function displayWinRate(value: number | null | undefined): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return 'N/A';
  return `${Math.round(value * 100)}%`;
}

export function formatPricePer1k(value: number | null | undefined): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return 'N/A';
  return `$${value.toLocaleString('en-US', { maximumFractionDigits: value < 10 ? 2 : 0 })}`;
}

export function topImageModels(limit = 20): ImageLeaderboardModel[] {
  return textToImageLeaderboard.slice(0, limit);
}
