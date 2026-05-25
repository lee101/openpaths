import { ZIMAGE_ART_MANIFEST_URL, fallbackZImageArt, type ZImageArtItem, type ZImageArtManifest } from '../data/zimageArt';

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

export async function searchZImageArt(query: string, limit: number): Promise<ZImageArtItem[] | null> {
  const q = query.trim();
  if (!q) return null;
  try {
    const resp = await fetch(`/art/search?q=${encodeURIComponent(q)}&limit=${limit}`, { headers: { Accept: 'application/json' } });
    if (!resp.ok) return null;
    const data = await resp.json();
    if (!Array.isArray(data.results)) return null;
    return data.results.filter(isUsableItem);
  } catch {
    return null;
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
