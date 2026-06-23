// Client for the private saved-response search + settings endpoints.
// Auth mirrors src/pages/Account.tsx: Bearer of the stored API key, same origin.

export interface SavedResponse {
  id: string;
  kind: 'text' | 'image';
  model: string;
  provider: string;
  prompt: string;
  input?: string;
  output?: string;
  image_url?: string;
  thumb_url?: string;
  width?: number;
  height?: number;
  tokens_in?: number;
  tokens_out?: number;
  cost_cents?: number;
  created_at: string;
  score?: number;
}

export interface SearchResponse {
  results: SavedResponse[];
  count: number;
  total: number;
  mode: 'semantic' | 'trigram' | 'browse';
}

export interface UsageSettings {
  text_enabled: boolean;
  image_enabled: boolean;
  available?: boolean;
}

function getAPIKey(): string {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem('op_api_key') || '';
}

export function isAuthenticated(): boolean {
  return !!getAPIKey();
}

async function api(path: string, opts: RequestInit = {}): Promise<Response> {
  const key = getAPIKey();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(opts.headers as Record<string, string>),
  };
  if (key) headers.Authorization = `Bearer ${key}`;
  return fetch(path, { ...opts, headers });
}

export async function getUsageSettings(): Promise<UsageSettings> {
  const res = await api('/account/usage/settings');
  if (!res.ok) throw new Error('Failed to load settings');
  return res.json();
}

export async function updateUsageSettings(s: { text_enabled: boolean; image_enabled: boolean }): Promise<UsageSettings> {
  const res = await api('/account/usage/settings', { method: 'POST', body: JSON.stringify(s) });
  if (!res.ok) throw new Error('Failed to update settings');
  return res.json();
}

export async function searchSavedResponses(
  kind: 'text' | 'image',
  query: string,
  limit = 48,
  offset = 0,
): Promise<SearchResponse> {
  const params = new URLSearchParams({ kind, q: query, limit: String(limit), offset: String(offset) });
  const res = await api(`/account/usage/responses?${params.toString()}`);
  if (!res.ok) throw new Error('Search failed');
  return res.json();
}

export async function getSavedResponse(id: string): Promise<{ item: SavedResponse; similar: SavedResponse[] }> {
  const res = await api(`/account/usage/responses/${encodeURIComponent(id)}`);
  if (!res.ok) throw new Error('Not found');
  return res.json();
}
