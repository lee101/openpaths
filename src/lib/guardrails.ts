import { getStoredAPIKey, getAppOrigin } from './session';

export type ResetInterval = 'daily' | 'weekly' | 'monthly';

export interface PromptInjectionConfig {
  enabled: boolean;
  action: 'block' | 'email' | 'flag';
  patterns?: string[];
}

export interface SensitiveFilter {
  slug: 'email' | 'phone' | 'ssn' | 'credit-card' | 'ip-address';
  action: 'block' | 'redact' | 'email';
}

export interface CustomFilter {
  name: string;
  pattern: string;
  action: 'block' | 'redact' | 'email';
}

export interface GuardrailAssignment {
  guardrail_id: string;
  target_type: 'api_key' | 'user';
  target_id: string;
  created_at: string;
}

export interface Guardrail {
  id: string;
  user_id: string;
  name: string;
  limit_cents: number | null;
  reset_interval: ResetInterval | null;
  budget_actions: string[];
  allowed_models: string[];
  allowed_providers: string[];
  prompt_injection: PromptInjectionConfig;
  sensitive_info: { filters: SensitiveFilter[] };
  custom_filters: CustomFilter[];
  assignments?: GuardrailAssignment[];
  created_at: string;
  updated_at: string;
}

export interface GuardrailInput {
  name: string;
  limit_cents: number | null;
  reset_interval: ResetInterval | null;
  budget_actions: string[];
  allowed_models: string[];
  allowed_providers: string[];
  prompt_injection: PromptInjectionConfig;
  sensitive_info: { filters: SensitiveFilter[] };
  custom_filters: CustomFilter[];
}

export const PII_SLUGS: SensitiveFilter['slug'][] = ['email', 'phone', 'ssn', 'credit-card', 'ip-address'];

export const PROVIDER_OPTIONS = [
  'openai', 'anthropic', 'google', 'deepseek', 'xai', 'mistral', 'netwrck', 'fal', 'bfl',
  'minimax', 'cutedsl', 'together', 'fireworks', 'groq', 'nvidia', 'sakana', 'openrouter', 'cursor',
];

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

function normalize(g: any): Guardrail {
  const pi = typeof g.prompt_injection === 'string' ? JSON.parse(g.prompt_injection || '{}') : (g.prompt_injection || {});
  const si = typeof g.sensitive_info === 'string' ? JSON.parse(g.sensitive_info || '{}') : (g.sensitive_info || {});
  const cf = typeof g.custom_filters === 'string' ? JSON.parse(g.custom_filters || '[]') : (g.custom_filters || []);
  return {
    ...g,
    prompt_injection: {
      enabled: !!pi.enabled,
      action: pi.action || 'block',
      patterns: pi.patterns || [],
    },
    sensitive_info: { filters: si.filters || [] },
    custom_filters: Array.isArray(cf) ? cf : [],
    budget_actions: g.budget_actions || [],
    allowed_models: g.allowed_models || [],
    allowed_providers: g.allowed_providers || [],
    assignments: g.assignments || [],
  };
}

export function emptyGuardrailInput(): GuardrailInput {
  return {
    name: 'New guardrail',
    limit_cents: null,
    reset_interval: 'daily',
    budget_actions: ['block'],
    allowed_models: [],
    allowed_providers: [],
    prompt_injection: { enabled: false, action: 'block', patterns: [] },
    sensitive_info: { filters: [] },
    custom_filters: [],
  };
}

export async function listGuardrails(): Promise<Guardrail[]> {
  const res = await fetch(`${base()}/account/guardrails`, { headers: authHeaders() });
  const data = await asJson(res);
  return (data.guardrails || []).map(normalize);
}

export async function createGuardrail(input: GuardrailInput): Promise<Guardrail> {
  const res = await fetch(`${base()}/account/guardrails`, {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify(input),
  });
  return normalize(await asJson(res));
}

export async function updateGuardrail(id: string, input: GuardrailInput): Promise<Guardrail> {
  const res = await fetch(`${base()}/account/guardrails/${id}`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify(input),
  });
  return normalize(await asJson(res));
}

export async function deleteGuardrail(id: string): Promise<void> {
  const res = await fetch(`${base()}/account/guardrails/${id}`, {
    method: 'DELETE',
    headers: authHeaders(),
  });
  await asJson(res);
}

export async function setAssignments(id: string, apiKeyIds: string[], userDefault: boolean): Promise<Guardrail> {
  const res = await fetch(`${base()}/account/guardrails/${id}/assignments`, {
    method: 'PUT',
    headers: authHeaders(),
    body: JSON.stringify({ api_key_ids: apiKeyIds, user_default: userDefault }),
  });
  return normalize(await asJson(res));
}

/** Convert USD to our hundredths-of-a-cent units. */
export function usdToCents(usd: number): number {
  return Math.round(usd * 10000);
}

export function centsToUsd(cents: number): number {
  return cents / 10000;
}

export function summarize(g: Guardrail): string[] {
  const tags: string[] = [];
  if (g.limit_cents != null && g.limit_cents > 0) {
    tags.push(`$${centsToUsd(g.limit_cents).toFixed(0)} / ${g.reset_interval || 'daily'}`);
  }
  if (g.allowed_models.length || g.allowed_providers.length) tags.push('Access');
  if (g.prompt_injection?.enabled) tags.push('Injection');
  if (g.sensitive_info?.filters?.length) tags.push('PII');
  if (g.custom_filters?.length) tags.push('Regex');
  if (!tags.length) tags.push('Empty');
  return tags;
}
