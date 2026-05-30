// Client for the OpenPaths prompt library API (/v1/prompts*), with lexical
// fallback over the bundled seed when the backend is unavailable. Mirrors the
// shape of src/lib/zimageArt.ts.

import {
  fallbackPrompts,
  type LibraryPrompt,
  type PromptMeta,
} from '../data/promptLibrary';

export interface PromptFilters {
  q?: string;
  category?: string;
  model?: string;
  type?: string;
  free?: boolean;
  limit?: number;
  offset?: number;
}

export interface PromptListResult {
  results: LibraryPrompt[];
  total: number;
  semantic: boolean;
  source: 'api' | 'fallback';
}

async function fetchJson<T>(url: string, timeoutMs = 6000): Promise<T> {
  const ctrl = new AbortController();
  const timer = window.setTimeout(() => ctrl.abort(), timeoutMs);
  try {
    const resp = await fetch(url, { headers: { Accept: 'application/json' }, signal: ctrl.signal });
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    return (await resp.json()) as T;
  } finally {
    window.clearTimeout(timer);
  }
}

function buildQuery(filters: PromptFilters): string {
  const params = new URLSearchParams();
  if (filters.q) params.set('q', filters.q);
  if (filters.category) params.set('category', filters.category);
  if (filters.model) params.set('model', filters.model);
  if (filters.type) params.set('type', filters.type);
  if (filters.free) params.set('free', '1');
  if (filters.limit) params.set('limit', String(filters.limit));
  if (filters.offset) params.set('offset', String(filters.offset));
  const qs = params.toString();
  return qs ? `?${qs}` : '';
}

export async function fetchPrompts(filters: PromptFilters = {}): Promise<PromptListResult> {
  try {
    const data = await fetchJson<{ results: LibraryPrompt[]; total: number; semantic: boolean }>(
      `/v1/prompts${buildQuery(filters)}`,
    );
    if (Array.isArray(data.results)) {
      return { results: data.results, total: data.total ?? data.results.length, semantic: !!data.semantic, source: 'api' };
    }
  } catch {
    /* fall through to local */
  }
  const local = lexicalPromptSearch(fallbackPrompts, filters);
  return { results: local, total: local.length, semantic: false, source: 'fallback' };
}

export async function fetchPromptMeta(): Promise<PromptMeta | null> {
  try {
    return await fetchJson<PromptMeta>('/v1/prompts/meta');
  } catch {
    return null;
  }
}

export async function fetchPrompt(
  slug: string,
): Promise<{ prompt: LibraryPrompt; related: LibraryPrompt[] } | null> {
  try {
    const data = await fetchJson<{ prompt: LibraryPrompt; related: LibraryPrompt[] }>(
      `/v1/prompts/${encodeURIComponent(slug)}`,
    );
    if (data?.prompt) return { prompt: data.prompt, related: data.related || [] };
  } catch {
    /* fall through to local */
  }
  const prompt = fallbackPrompts.find(p => p.slug === slug);
  if (!prompt) return null;
  const related = fallbackPrompts
    .filter(p => p.slug !== slug && (p.categorySlug === prompt.categorySlug || p.modality === prompt.modality))
    .slice(0, 4);
  return { prompt, related };
}

// Deep-link a prompt into the playground (model id + prompt text). The playground
// already reads ?model= and ?prompt= and auto-sends.
export function promptPlaygroundHref(prompt: LibraryPrompt): string {
  const params = new URLSearchParams({ model: prompt.modelSlug, prompt: prompt.prompt });
  return `/playground?${params.toString()}`;
}

function tokenize(value: string): string[] {
  return value
    .toLowerCase()
    .split(/[^a-z0-9]+/)
    .filter(t => t.length > 1);
}

// Lexical search + filtering over an in-memory prompt list (fallback path).
export function lexicalPromptSearch(items: LibraryPrompt[], filters: PromptFilters): LibraryPrompt[] {
  let list = items.filter(p => {
    if (filters.category && p.categorySlug !== filters.category) return false;
    if (filters.model && p.modelSlug !== filters.model) return false;
    if (filters.free && !p.isFree) return false;
    if (filters.type) {
      if (filters.type === 'free') {
        if (!p.isFree) return false;
      } else if (p.modality !== filters.type) {
        return false;
      }
    }
    return true;
  });

  const q = (filters.q || '').trim();
  if (q) {
    const terms = tokenize(q);
    const raw = q.toLowerCase();
    list = list
      .map(p => {
        const haystack = `${p.title} ${p.summary} ${p.prompt} ${p.tags.join(' ')}`.toLowerCase();
        let score = 0;
        for (const t of terms) {
          if (haystack.includes(t)) score += t.length > 4 ? 3 : 1;
          if (p.tags.some(tag => tag.toLowerCase() === t)) score += 4;
        }
        if (p.title.toLowerCase().includes(raw)) score += 8;
        else if (p.summary.toLowerCase().includes(raw)) score += 4;
        if (p.featured) score += 1;
        return { p, score };
      })
      .filter(r => r.score > 0)
      .sort((a, b) => b.score - a.score || b.p.popularity - a.p.popularity)
      .map(r => r.p);
  } else {
    list = [...list].sort((a, b) => Number(b.featured) - Number(a.featured) || b.popularity - a.popularity);
  }

  const offset = filters.offset || 0;
  const limit = filters.limit || list.length;
  return list.slice(offset, offset + limit);
}

// Saved prompts (playground "save a prompt / chat"). localStorage-backed MVP.
const SAVED_KEY = 'op_pg_saved_prompts';

export interface SavedPrompt {
  id: string;
  title: string;
  prompt: string;
  modelId?: string;
  modality?: string;
  savedAt: number;
}

export function loadSavedPrompts(): SavedPrompt[] {
  if (typeof window === 'undefined') return [];
  try {
    const raw = window.localStorage.getItem(SAVED_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? (parsed as SavedPrompt[]) : [];
  } catch {
    return [];
  }
}

export function savePrompt(entry: Omit<SavedPrompt, 'id' | 'savedAt'>): SavedPrompt[] {
  const list = loadSavedPrompts();
  const id = `${entry.modelId || 'prompt'}-${list.length}-${entry.title.slice(0, 24)}`;
  const next = [{ ...entry, id, savedAt: nextStamp(list) }, ...list].slice(0, 50);
  persistSaved(next);
  return next;
}

export function removeSavedPrompt(id: string): SavedPrompt[] {
  const next = loadSavedPrompts().filter(p => p.id !== id);
  persistSaved(next);
  return next;
}

// Monotonic-ish stamp without relying on Date in a way that breaks tests.
function nextStamp(list: SavedPrompt[]): number {
  const max = list.reduce((m, p) => Math.max(m, p.savedAt || 0), 0);
  return max + 1;
}

function persistSaved(list: SavedPrompt[]): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(SAVED_KEY, JSON.stringify(list));
  } catch {
    /* ignore quota errors */
  }
}
