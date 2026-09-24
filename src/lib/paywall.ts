import { getApiKey, setApiKey } from './api';

export const AUTH_REQUIRED_EVENT = 'op-auth-required';
export const DEFAULT_SUBSCRIBE_URL = '/pricing';

export type Paywall = { kind: 'login' | 'subscribe'; message: string; subscribeUrl: string };

export function requestSignIn() {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new CustomEvent(AUTH_REQUIRED_EVENT));
}

function sessionToken(): string {
  if (typeof window === 'undefined') return '';
  return window.userData?.secret || localStorage.getItem('op_token') || '';
}

export function hasSession(): boolean {
  return !!(getApiKey() || sessionToken() || (typeof window !== 'undefined' && window.userData?.authenticated));
}

// Returns an API key for in-page spaces, minting a "Spaces" key for dashboard sessions that only hold a JWT.
export async function ensureSpaceApiKey(): Promise<string> {
  const existing = getApiKey();
  if (existing) return existing;
  const token = sessionToken();
  if (!token) return '';
  if (token.startsWith('op_') || token.startsWith('sk-op-')) {
    setApiKey(token);
    return token;
  }
  try {
    const res = await fetch('/account/keys', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      credentials: 'include',
      body: JSON.stringify({ name: 'Spaces' }),
    });
    const data = await res.json().catch(() => ({}));
    if (res.ok && data?.key) {
      setApiKey(data.key);
      return data.key;
    }
  } catch {}
  return '';
}

export function paywallFromResponse(resp: Response, data: any): Paywall | null {
  if (resp.status !== 401 && resp.status !== 402) return null;
  const err = typeof data?.error === 'object' ? data.error : {};
  const subscribeUrl = err.subscribe_url || resp.headers.get('X-Subscribe-URL') || DEFAULT_SUBSCRIBE_URL;
  if (resp.status === 401) return { kind: 'login', message: err.message || 'Sign in to run this model.', subscribeUrl };
  return { kind: 'subscribe', message: err.message || 'Add credits to run this model.', subscribeUrl };
}

export function paywallBeforeRun(): Paywall | null {
  return hasSession() ? null : { kind: 'login', message: 'Sign in to run this model.', subscribeUrl: DEFAULT_SUBSCRIBE_URL };
}
