import { artGallery } from './artGallery';

export type ZImageArtItem = {
  id: string;
  slug: string;
  title?: string;
  prompt: string;
  imageUrl: string;
  thumbUrl?: string;
  width?: number;
  height?: number;
  aspect?: string; // 'square' | 'portrait' | 'wide'
  model: string;
  seed?: number;
  steps?: number;
  source?: string;
  tags?: string[];
  createdAt?: string;
  score?: number;
  mediaType?: string; // 'image' | 'video'
  videoUrl?: string;
  posterUrl?: string;
  durationSeconds?: number;
};

export type ArtTagFacet = { tag: string; count: number };

export type ArtListResponse = {
  results: ZImageArtItem[];
  total: number;
  aspects: Record<string, number>;
};

export type ZImageArtChunk = {
  path: string;
  url?: string;
  count: number;
};

export type ZImageArtManifest = {
  version: number;
  kind: string;
  model: string;
  count: number;
  generatedCount: number;
  publicBase?: string;
  generatedAt?: string;
  chunks: ZImageArtChunk[];
};

export const ZIMAGE_ART_PUBLIC_BASE = 'https://openpathsstatic.openpaths.io/static/data/zimage-art';
export const ZIMAGE_ART_MANIFEST_URL = `${ZIMAGE_ART_PUBLIC_BASE}/manifest.json`;

export const fallbackZImageArt: ZImageArtItem[] = artGallery
  .filter(item => item.providerModelId === 'zimage')
  .map(item => ({
    id: item.slug,
    slug: item.slug,
    title: item.title,
    prompt: item.prompt,
    imageUrl: item.imageUrl,
    thumbUrl: item.imageUrl,
    width: 1024,
    height: 1024,
    model: item.providerModelId,
    steps: 20,
    source: 'openpaths-seed',
    tags: ['anime', 'illustration', 'zimage'],
  }));
