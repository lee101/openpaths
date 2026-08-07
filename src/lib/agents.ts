import { getStoredToken, getStoredAPIKey } from './session';

export interface AgentConfig {
  tools: string[];
  max_steps?: number;
  temperature?: number;
  image_model?: string;
  video_model?: string;
}

export interface AgentSource {
  id: string;
  agent_id: string;
  kind: string;
  name: string;
  status: string;
  meta?: Record<string, any>;
  chunk_count: number;
}

export interface Agent {
  id: string;
  name: string;
  description?: string;
  system_prompt?: string;
  model: string;
  config: AgentConfig;
  preset?: string;
  sources?: AgentSource[];
}

export interface ToolSpec {
  name: string;
  label: string;
  description: string;
  experimental?: boolean;
}

export interface Preset {
  key: string;
  name: string;
  description: string;
  system_prompt: string;
  model: string;
  config: AgentConfig;
  example_prompts?: string[];
}

export interface RunStep {
  type: string;
  tool?: string;
  input?: string;
  output?: string;
  model?: string;
  elapsed_ms?: number;
}

export interface AgentRun {
  id: string;
  input: string;
  output: string;
  steps: RunStep[];
  status: string;
  cost_cents: number;
  created_at?: string;
}

export interface KnowledgeResult {
  id: string;
  data_source_id: string;
  title?: string;
  chunk_index: number;
  content: string;
  score?: number;
}

function authHeaders(json = true): Record<string, string> {
  const h: Record<string, string> = {};
  if (json) h['Content-Type'] = 'application/json';
  const tok = getStoredToken() || getStoredAPIKey();
  if (tok) h.Authorization = `Bearer ${tok}`;
  return h;
}

async function jsonOrThrow(r: Response) {
  const body = await r.json().catch(() => ({}));
  if (!r.ok) throw new Error(body?.error?.message || body?.error || `request failed (${r.status})`);
  return body;
}

export async function fetchPresets(): Promise<{ presets: Preset[]; tools: ToolSpec[] }> {
  return jsonOrThrow(await fetch('/v1/agents/presets', { headers: { Accept: 'application/json' } }));
}

export async function listAgents(): Promise<Agent[]> {
  const b = await jsonOrThrow(await fetch('/v1/agents', { headers: authHeaders(false) }));
  return b.agents || [];
}

export async function createAgent(body: Partial<Agent> & { preset?: string }): Promise<Agent> {
  return jsonOrThrow(await fetch('/v1/agents', { method: 'POST', headers: authHeaders(), body: JSON.stringify(body) }));
}

export async function getAgent(id: string): Promise<Agent> {
  return jsonOrThrow(await fetch(`/v1/agents/${id}`, { headers: authHeaders(false) }));
}

export async function updateAgent(id: string, body: Partial<Agent>): Promise<Agent> {
  return jsonOrThrow(await fetch(`/v1/agents/${id}`, { method: 'PATCH', headers: authHeaders(), body: JSON.stringify(body) }));
}

export async function deleteAgent(id: string): Promise<void> {
  await jsonOrThrow(await fetch(`/v1/agents/${id}`, { method: 'DELETE', headers: authHeaders(false) }));
}

export async function addSource(id: string, body: Record<string, any>): Promise<AgentSource> {
  return jsonOrThrow(await fetch(`/v1/agents/${id}/sources`, { method: 'POST', headers: authHeaders(), body: JSON.stringify(body) }));
}

export async function uploadSource(id: string, file: File): Promise<AgentSource> {
  const fd = new FormData();
  fd.append('file', file);
  return jsonOrThrow(await fetch(`/v1/agents/${id}/sources/upload`, { method: 'POST', headers: authHeaders(false), body: fd }));
}

export async function deleteSource(id: string, sid: string): Promise<void> {
  await jsonOrThrow(await fetch(`/v1/agents/${id}/sources/${sid}`, { method: 'DELETE', headers: authHeaders(false) }));
}

export async function listAgentRuns(id: string): Promise<AgentRun[]> {
  const body = await jsonOrThrow(await fetch(`/v1/agents/${id}/runs`, { headers: authHeaders(false) }));
  return body.runs || [];
}

export async function searchAgentKnowledge(id: string, query: string): Promise<KnowledgeResult[]> {
  const body = await jsonOrThrow(await fetch(`/v1/agents/${id}/search?q=${encodeURIComponent(query)}&limit=8`, { headers: authHeaders(false) }));
  return body.results || [];
}

export async function runAgentStream(
  id: string,
  input: string,
  onStep: (s: RunStep) => void,
  onDone: (r: AgentRun) => void,
  onError: (e: string) => void,
): Promise<void> {
  const resp = await fetch(`/v1/agents/${id}/run/stream`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify({ input }),
  });
  if (!resp.ok || !resp.body) {
    const body = await resp.json().catch(() => ({}));
    onError(body?.error?.message || body?.error || `run failed (${resp.status})`);
    return;
  }
  const reader = resp.body.getReader();
  const dec = new TextDecoder();
  let buf = '';
  let terminalEvent = false;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += dec.decode(value, { stream: true });
    const events = buf.split('\n\n');
    buf = events.pop() || '';
    for (const ev of events) {
      let event = 'message';
      let data = '';
      for (const line of ev.split('\n')) {
        if (line.startsWith('event:')) event = line.slice(6).trim();
        else if (line.startsWith('data:')) data += line.slice(5).trim();
      }
      if (!data) continue;
      try {
        const parsed = JSON.parse(data);
        if (event === 'step') onStep(parsed);
        else if (event === 'done') { terminalEvent = true; onDone(parsed); }
        else if (event === 'error') { terminalEvent = true; onError(parsed.error || 'error'); }
      } catch { /* ignore partial */ }
    }
  }
  if (!terminalEvent) onError('The run ended before a final response was received.');
}
