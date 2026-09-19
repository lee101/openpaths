import { getStoredAPIKey, getStoredToken } from '../lib/session';

// Client for the skills marketplace API (/v1/skills*). Public reads; writes
// carry the stored op key as a Bearer token (same convention as
// src/lib/agents.ts authHeaders).

export interface Skill {
  id: string;
  slug: string;
  name: string;
  description: string;
  body?: string;
  source: string;
  sourceRepo?: string;
  category?: string;
  tags: string[];
  setupPreamble?: string;
  createdAt?: string;
  updatedAt?: string;
  ownerId: string;
  currentVersion: string;
  versionCount: number;
  setupScript?: string;
  setupPrompt?: string;
  skillPrompt?: string;
  versions?: string[];
  files?: SkillFileManifest[];
  score?: number;
}

export interface SkillFileManifest {
  path: string;
  mime: string;
  size: number;
}


export interface SkillListResult {
  skills: Skill[];
  count: number;
  semantic: boolean;
}

export interface SkillGetResult {
  skill: Skill;
  markdown: string;
  versions: string[];
  files: SkillFileManifest[];
}

export interface SkillsMeta {
  total: number;
  sources: { id: string; label: string; repo: string; count: number }[];
  categories: { name: string; count: number }[];
}

export interface SkillInput {
  slug?: string;
  name?: string;
  description?: string;
  body?: string;
  source?: string;
  sourceRepo?: string;
  category?: string;
  tags?: string[];
  setupScript?: string;
  setupPrompt?: string;
  skillPrompt?: string;
  version?: string;
  files?: { path: string; content: string }[];
}

function authHeaders(json = true): Record<string, string> {
  const h: Record<string, string> = {};
  if (json) h['Content-Type'] = 'application/json';
  const tok = getStoredToken() || getStoredAPIKey();
  if (tok) h.Authorization = `Bearer ${tok}`;
  return h;
}

function enc(slug: string): string {
  return encodeURIComponent(slug);
}

async function jsonOrThrow<T>(res: Response): Promise<T> {
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    const msg =
      (body as { error?: { message?: string } | string })?.error != null
        ? typeof (body as { error?: unknown }).error === 'string'
          ? ((body as unknown as { error: string }).error)
          : ((body as { error?: { message?: string } }).error?.message ?? `request failed (${res.status})`)
        : `request failed (${res.status})`;
    const err = new Error(msg) as Error & { status?: number };
    err.status = res.status;
    throw err;
  }
  return body as T;
}

function versionQs(version?: string): string {
  return version ? `?version=${encodeURIComponent(version)}` : '';
}

export async function listSkills(params: { q?: string; source?: string; category?: string; version?: string; limit?: number } = {}): Promise<SkillListResult> {
  const qs = new URLSearchParams();
  if (params.q) qs.set('q', params.q);
  if (params.source) qs.set('source', params.source);
  if (params.category) qs.set('category', params.category);
  if (params.version) qs.set('version', params.version);
  if (params.limit) qs.set('limit', String(params.limit));
  const q = qs.toString() ? `?${qs.toString()}` : '';
  const b = await jsonOrThrow<{ skills?: Skill[]; count?: number; semantic?: boolean }>(
    await fetch(`/v1/skills${q}`, { headers: { Accept: 'application/json' } }),
  );
  return { skills: b.skills ?? [], count: b.count ?? (b.skills ?? []).length, semantic: !!b.semantic };
}

export async function fetchSkillsMeta(): Promise<SkillsMeta | null> {
  try {
    const b = await jsonOrThrow<{ total?: number; sources?: SkillsMeta['sources']; categories?: SkillsMeta['categories'] }>(
      await fetch('/v1/skills/meta', { headers: { Accept: 'application/json' } }),
    );
    return { total: b.total ?? 0, sources: b.sources ?? [], categories: b.categories ?? [] };
  } catch {
    return null;
  }
}

export async function searchSkills(q: string, version?: string, limit = 40): Promise<SkillListResult> {
  const qs = new URLSearchParams({ q, k: String(limit) });
  if (version) qs.set('version', version);
  const b = await jsonOrThrow<{ results?: Skill[]; skills?: Skill[]; count?: number; semantic?: boolean }>(
    await fetch(`/v1/skills/search?${qs.toString()}`, { headers: { Accept: 'application/json' } }),
  );
  const skills = b.results ?? b.skills ?? [];
  return { skills, count: b.count ?? skills.length, semantic: !!b.semantic };
}

export async function getSkill(slug: string, version?: string): Promise<SkillGetResult> {
  return jsonOrThrow<SkillGetResult>(
    await fetch(`/v1/skills/${enc(slug)}${versionQs(version)}`, { headers: { Accept: 'application/json' } }),
  );
}

export async function listSkillFiles(slug: string, version?: string): Promise<SkillFileManifest[]> {
  const b = await jsonOrThrow<{ files?: SkillFileManifest[] }>(
    await fetch(`/v1/skills/${enc(slug)}/files${versionQs(version)}`, { headers: { Accept: 'application/json' } }),
  );
  return b.files ?? [];
}

export function skillFileRawUrl(slug: string, path: string, version?: string): string {
  const qs = version ? `?version=${encodeURIComponent(version)}` : '';
  return `/v1/skills/${enc(slug)}/files/${path.split('/').map(encodeURIComponent).join('/')}${qs}`;
}

export async function fetchSkillFileText(slug: string, path: string, version?: string): Promise<string> {
  const res = await fetch(skillFileRawUrl(slug, path, version), { headers: { Accept: '*/*' } });
  if (!res.ok) throw new Error(`file fetch failed (${res.status})`);
  return res.text();
}

export async function createSkill(input: SkillInput): Promise<Skill> {
  const b = await jsonOrThrow<{ skill: Skill }>(
    await fetch('/v1/skills', { method: 'POST', headers: authHeaders(), body: JSON.stringify(input) }),
  );
  return b.skill;
}

export async function updateSkill(slug: string, input: SkillInput): Promise<Skill> {
  const b = await jsonOrThrow<{ skill: Skill }>(
    await fetch(`/v1/skills/${enc(slug)}`, { method: 'PUT', headers: authHeaders(), body: JSON.stringify(input) }),
  );
  return b.skill;
}

export async function deleteSkill(slug: string): Promise<void> {
  await jsonOrThrow<unknown>(
    await fetch(`/v1/skills/${enc(slug)}`, { method: 'DELETE', headers: authHeaders(false) }),
  );
}

export function isLoggedIn(): boolean {
  return !!(getStoredToken() || getStoredAPIKey());
}

export function mimeFromPath(path: string): string {
  const ext = path.split('.').pop()?.toLowerCase() ?? '';
  switch (ext) {
    case 'png': return 'image/png';
    case 'jpg':
    case 'jpeg': return 'image/jpeg';
    case 'gif': return 'image/gif';
    case 'webp': return 'image/webp';
    case 'svg': return 'image/svg+xml';
    case 'mp3': return 'audio/mpeg';
    case 'wav': return 'audio/wav';
    case 'ogg': return 'audio/ogg';
    case 'mp4': return 'video/mp4';
    case 'webm': return 'video/webm';
    case 'pdf': return 'application/pdf';
    case 'md':
    case 'markdown': return 'text/markdown';
    case 'json': return 'application/json';
    case 'html': return 'text/html';
    default: return 'text/plain';
  }
}

export function langFromPath(path: string): string {
  const ext = path.split('.').pop()?.toLowerCase() ?? '';
  const map: Record<string, string> = {
    js: 'javascript', jsx: 'javascript', ts: 'typescript', tsx: 'typescript',
    py: 'python', sh: 'bash', shell: 'bash', yml: 'yaml', yaml: 'yaml',
    md: 'markdown', markdown: 'markdown', json: 'json', html: 'html', css: 'css',
    go: 'go', rs: 'rust', sql: 'sql',
  };
  return map[ext] ?? ext;
}
