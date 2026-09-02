import {
  ZIMAGE_ART_MANIFEST_URL,
  fallbackZImageArt,
  type ArtListResponse,
  type ArtTagFacet,
  type ZImageArtItem,
  type ZImageArtManifest,
} from '../data/zimageArt';

export type ArtSearchFilters = {
  aspect?: string;
  tag?: string;
  limit?: number;
  offset?: number;
};

function artQueryString(filters: ArtSearchFilters): string {
  const params = new URLSearchParams();
  if (filters.aspect) params.set('aspect', filters.aspect);
  if (filters.tag) params.set('tag', filters.tag);
  if (filters.limit) params.set('limit', String(filters.limit));
  if (filters.offset) params.set('offset', String(filters.offset));
  const qs = params.toString();
  return qs ? `&${qs}` : '';
}

type LoadResult = {
  manifest: ZImageArtManifest | null;
  items: ZImageArtItem[];
  source: 'remote' | 'fallback';
};

export async function loadZImageArtIndex(manifestUrl = ZIMAGE_ART_MANIFEST_URL): Promise<LoadResult> {
  try {
    const manifest = await fetchJson<ZImageArtManifest>(manifestUrl);
    const base = manifest.publicBase || manifestUrl.slice(0, manifestUrl.lastIndexOf('/'));
    const chunks = await Promise.all(
      manifest.chunks.map(async chunk => {
        const chunkUrl = chunk.url || `${base.replace(/\/$/, '')}/${chunk.path.replace(/^\//, '')}`;
        return await fetchJson<ZImageArtItem[]>(chunkUrl);
      }),
    );
    const items = chunks.flat().filter(isUsableItem);
    return {
      manifest: { ...manifest, count: manifest.count || items.length },
      items: items.length ? items : fallbackZImageArt,
      source: items.length ? 'remote' : 'fallback',
    };
  } catch {
    return { manifest: null, items: fallbackZImageArt, source: 'fallback' };
  }
}

async function fetchJson<T>(url: string): Promise<T> {
  const ctrl = new AbortController();
  const timeout = window.setTimeout(() => ctrl.abort(), 5000);
  try {
    const resp = await fetch(url, { headers: { Accept: 'application/json' }, signal: ctrl.signal });
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    return await resp.json() as T;
  } finally {
    window.clearTimeout(timeout);
  }
}

export async function searchZImageArt(query: string, limit: number, filters: ArtSearchFilters = {}): Promise<ZImageArtItem[] | null> {
  const q = query.trim();
  if (!q) return null;
  try {
    const resp = await fetch(`/v1/art/search?q=${encodeURIComponent(q)}&limit=${limit}${artQueryString(filters)}`, { headers: { Accept: 'application/json' } });
    if (!resp.ok) return null;
    const data = await resp.json();
    if (!Array.isArray(data.results)) return null;
    return data.results.filter(isUsableItem);
  } catch {
    return null;
  }
}

// fetchArtList browses the DB-backed gallery (no query) with aspect/tag filters + totals.
export async function fetchArtList(filters: ArtSearchFilters = {}): Promise<ArtListResponse | null> {
  try {
    const resp = await fetch(`/v1/art/list?${artQueryString(filters).replace(/^&/, '')}`, { headers: { Accept: 'application/json' } });
    if (!resp.ok) return null;
    const data = await resp.json();
    if (!Array.isArray(data.results)) return null;
    return {
      results: data.results.filter(isUsableItem),
      total: typeof data.total === 'number' ? data.total : data.results.length,
      aspects: data.aspects || {},
    };
  } catch {
    return null;
  }
}

export async function fetchArtItem(slug: string): Promise<{ item: ZImageArtItem; related: ZImageArtItem[] } | null> {
  try {
    const resp = await fetch(`/v1/art/item?slug=${encodeURIComponent(slug)}`, { headers: { Accept: 'application/json' } });
    if (!resp.ok) return null;
    const data = await resp.json();
    if (!data.item) return null;
    return { item: data.item, related: Array.isArray(data.related) ? data.related.filter(isUsableItem) : [] };
  } catch {
    return null;
  }
}

// Rank tags that appear across the existing neighbors so detail pages can
// continue browsing through the semantic search index without another API shape.
export function deriveRelatedArtTags(item: ZImageArtItem, related: ZImageArtItem[], limit = 4): string[] {
  if (limit <= 0) return [];

  const ownTags = new Set((item.tags || []).map(tag => tag.trim().toLowerCase()).filter(Boolean));
  const counts = new Map<string, { label: string; count: number }>();
  for (const candidate of related) {
    const seenOnCandidate = new Set<string>();
    for (const rawTag of candidate.tags || []) {
      const label = rawTag.trim();
      const key = label.toLowerCase();
      if (!key || seenOnCandidate.has(key)) continue;
      seenOnCandidate.add(key);
      const current = counts.get(key);
      if (current) {
        current.count += 1;
      } else {
        counts.set(key, { label, count: 1 });
      }
    }
  }

  return [...counts.entries()]
    .sort(([aKey, a], [bKey, b]) => Number(ownTags.has(aKey)) - Number(ownTags.has(bKey)) || b.count - a.count || aKey.localeCompare(bKey))
    .slice(0, limit)
    .map(([, value]) => value.label);
}

export async function fetchArtTags(limit = 60): Promise<ArtTagFacet[]> {
  try {
    const resp = await fetch(`/v1/art/tags?limit=${limit}`, { headers: { Accept: 'application/json' } });
    if (!resp.ok) return [];
    const data = await resp.json();
    return Array.isArray(data.tags) ? data.tags : [];
  } catch {
    return [];
  }
}

export function lexicalZImageSearch(items: ZImageArtItem[], query: string, limit = 48): ZImageArtItem[] {
  const terms = tokenize(query);
  if (terms.length === 0) return items.slice(0, limit);
  return items
    .map(item => ({ item, score: scoreItem(item, terms) }))
    .filter(row => row.score > 0)
    .sort((a, b) => b.score - a.score)
    .slice(0, limit)
    .map(row => ({ ...row.item, score: row.score }));
}

export function artPromptPlaygroundHref(item: ZImageArtItem): string {
  const params = new URLSearchParams({
    model: 'zimage',
    prompt: item.prompt,
  });
  return `/playground?${params.toString()}`;
}

function isUsableItem(item: ZImageArtItem): boolean {
  return Boolean(item && item.prompt && item.model);
}

function scoreItem(item: ZImageArtItem, terms: string[]): number {
  const haystack = `${item.title || ''} ${item.prompt} ${(item.tags || []).join(' ')}`.toLowerCase();
  let score = 0;
  for (const term of terms) {
    if (haystack.includes(term)) score += term.length > 4 ? 3 : 1;
    if ((item.tags || []).some(tag => tag.toLowerCase() === term)) score += 4;
  }
  return score;
}

function tokenize(value: string): string[] {
  return value
    .toLowerCase()
    .split(/[^a-z0-9]+/)
    .map(term => term.trim())
    .filter(term => term.length > 1);
}
