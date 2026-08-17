import { getStoredAPIKey, getAppOrigin } from './session';

export interface ArtifactFile {
  path: string;
  content: string;
}

export interface Artifact {
  id: string;
  slug: string;
  user_id: string;
  title: string;
  description: string;
  image_url: string;
  entry: string;
  visibility: 'private' | 'public' | 'unlisted';
  tags: string[];
  view_count: number;
  fork_of?: string;
  files?: ArtifactFile[];
  created_at: number;
  updated_at: number;
  published_at?: number;
}

export interface ArtifactInput {
  title: string;
  description: string;
  image_url: string;
  files: ArtifactFile[];
  entry: string;
  visibility: 'private' | 'public' | 'unlisted';
  tags: string[];
  fork_of?: string;
}

function authHeaders(): Record<string, string> {
  const key = getStoredAPIKey();
  const h: Record<string, string> = { 'Content-Type': 'application/json' };
  if (key) h.Authorization = `Bearer ${key}`;
  return h;
}

const base = () => getAppOrigin();

async function asJson(res: Response) {
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data?.error?.message || `Request failed (${res.status})`);
  return data;
}

export async function listMine(): Promise<Artifact[]> {
  const res = await fetch(`${base()}/account/artifacts`, { headers: authHeaders() });
  return (await asJson(res)).artifacts || [];
}

export async function listPublic(limit = 48): Promise<Artifact[]> {
  const res = await fetch(`${base()}/v1/artifacts?limit=${limit}`);
  return (await asJson(res)).artifacts || [];
}

export async function searchMine(q: string): Promise<Artifact[]> {
  const res = await fetch(`${base()}/account/artifacts/search?q=${encodeURIComponent(q)}`, { headers: authHeaders() });
  return (await asJson(res)).artifacts || [];
}

// Public, billed $1/1000.
export async function searchPublic(q: string, limit = 24): Promise<Artifact[]> {
  const res = await fetch(`${base()}/v1/artifacts/search?q=${encodeURIComponent(q)}&limit=${limit}`, { headers: authHeaders() });
  return (await asJson(res)).artifacts || [];
}

export async function getMine(id: string): Promise<Artifact> {
  const res = await fetch(`${base()}/account/artifacts/${id}`, { headers: authHeaders() });
  return asJson(res);
}

export async function getPublic(id: string): Promise<Artifact> {
  const res = await fetch(`${base()}/v1/artifacts/${encodeURIComponent(id)}`, { headers: authHeaders() });
  return asJson(res);
}

export async function createArtifact(input: ArtifactInput): Promise<Artifact> {
  const res = await fetch(`${base()}/account/artifacts`, { method: 'POST', headers: authHeaders(), body: JSON.stringify(input) });
  return asJson(res);
}

export async function updateArtifact(id: string, input: ArtifactInput): Promise<Artifact> {
  const res = await fetch(`${base()}/account/artifacts/${id}`, { method: 'PUT', headers: authHeaders(), body: JSON.stringify(input) });
  return asJson(res);
}

export async function deleteArtifact(id: string): Promise<void> {
  const res = await fetch(`${base()}/account/artifacts/${id}`, { method: 'DELETE', headers: authHeaders() });
  await asJson(res);
}

export function isLoggedIn(): boolean {
  return !!getStoredAPIKey();
}

export function langFromPath(path: string): string {
  const ext = path.split('.').pop()?.toLowerCase() || '';
  if (ext === 'html' || ext === 'htm') return 'html';
  if (ext === 'css') return 'css';
  if (ext === 'json') return 'json';
  if (ext === 'js' || ext === 'jsx' || ext === 'mjs' || ext === 'ts' || ext === 'tsx') return 'javascript';
  return 'text';
}

// Build a single srcdoc HTML string for the preview iframe by inlining local css/js
// referenced from the entry HTML file. Falls back to the entry file as-is.
export function buildPreviewDoc(files: ArtifactFile[], entry: string): string {
  const byPath = new Map(files.map(f => [f.path.replace(/^\.?\//, ''), f]));
  const entryFile = byPath.get(entry.replace(/^\.?\//, '')) || files.find(f => /\.html?$/.test(f.path));
  if (!entryFile) {
    // No HTML: show the first file as plain text.
    const first = files[0];
    return `<pre style="color:#ddd;background:#0b0b0f;padding:16px;font-family:monospace;white-space:pre-wrap">${escapeHtml(first?.content || 'Empty artifact')}</pre>`;
  }
  let html = entryFile.content;
  // Inline <link rel=stylesheet href=local.css>
  html = html.replace(/<link[^>]*href=["']([^"']+)["'][^>]*>/gi, (m, href) => {
    const f = byPath.get(String(href).replace(/^\.?\//, ''));
    return f ? `<style>${f.content}</style>` : m;
  });
  // Inline <script src=local.js>
  html = html.replace(/<script[^>]*src=["']([^"']+)["'][^>]*>\s*<\/script>/gi, (m, src) => {
    const f = byPath.get(String(src).replace(/^\.?\//, ''));
    return f ? `<script>${f.content}</script>` : m;
  });
  return html;
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}

// Parse agent output for file blocks. Supports fenced blocks annotated with a path:
//   ```html path=index.html      or      ```js title="app.js"
// Returns the files found (later duplicates win) and the prose with file blocks removed.
export function parseAgentFiles(text: string): { files: ArtifactFile[]; prose: string } {
  const files: ArtifactFile[] = [];
  const seen = new Map<string, number>();
  const fence = /```[^\n]*?(?:path|title|file)\s*=\s*["']?([^\s"'`]+)["']?[^\n]*\n([\s\S]*?)```/gi;
  let prose = text;
  let m: RegExpExecArray | null;
  while ((m = fence.exec(text)) !== null) {
    const path = m[1].replace(/^\.?\//, '').trim();
    const content = m[2].replace(/\n$/, '');
    if (!path) continue;
    if (seen.has(path)) {
      files[seen.get(path)!] = { path, content };
    } else {
      seen.set(path, files.length);
      files.push({ path, content });
    }
  }
  prose = text.replace(fence, '').trim();
  return { files, prose };
}

// Merge agent-produced files into the existing set (by path).
export function mergeFiles(existing: ArtifactFile[], incoming: ArtifactFile[]): ArtifactFile[] {
  const map = new Map(existing.map(f => [f.path, f]));
  for (const f of incoming) map.set(f.path, f);
  return Array.from(map.values());
}

export const ARTIFACT_SYSTEM_PROMPT = `You are an expert frontend engineer building a self-contained web artifact.
Output complete files. For EVERY file you create or change, emit a fenced code block whose info string includes the path, like:

\`\`\`html path=index.html
<!doctype html>...
\`\`\`

Rules:
- Build a single-page, self-contained app. Prefer one index.html; you may add style.css and app.js and reference them with relative paths.
- Always return the FULL content of any file you touch (no diffs, no ellipses).
- Keep it dependency-free or use CDN <script> tags. No build step.
- After the file blocks, add one or two sentences describing what you built or changed.`;
