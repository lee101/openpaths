export const API_KEY_STORAGE_KEY = 'op_api_key';
export const AUTH_EVENT = 'op-auth-changed';
export const CREDITS_REQUIRED_EVENT = 'op-credits-required';

export function getApiKey(): string {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem(API_KEY_STORAGE_KEY) || '';
}

export function setApiKey(key: string) {
  if (typeof window === 'undefined' || !key) return;
  localStorage.setItem(API_KEY_STORAGE_KEY, key);
  window.dispatchEvent(new CustomEvent(AUTH_EVENT));
}

export function clearApiKey() {
  if (typeof window === 'undefined') return;
  localStorage.removeItem(API_KEY_STORAGE_KEY);
  localStorage.removeItem('op_user');
  localStorage.removeItem('op_token');
  window.dispatchEvent(new CustomEvent(AUTH_EVENT));
}

export function onAuthChange(fn: () => void): () => void {
  if (typeof window === 'undefined') return () => {};
  window.addEventListener(AUTH_EVENT, fn);
  return () => window.removeEventListener(AUTH_EVENT, fn);
}

export function requestCreditTopUp(surface?: unknown) {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new CustomEvent(CREDITS_REQUIRED_EVENT, { detail: { surface: typeof surface === 'string' ? surface : 'credits_required' } }));
}

export async function api(path: string, opts: RequestInit = {}): Promise<Response> {
  const key = getApiKey();
  const headers: Record<string, string> = { 'Content-Type': 'application/json', ...(opts.headers as Record<string, string>) };
  if (key) headers.Authorization = `Bearer ${key}`;
  const res = await fetch(path, { ...opts, headers });
  // The stored key was rejected: drop the dead session so the UI shows signed-out instead of a stuck "Invalid key".
  if (res.status === 401 && key && !path.startsWith('/auth/') && !(opts.headers as Record<string, string> | undefined)?.Authorization) {
    if (typeof window !== 'undefined') window.userData = undefined;
    clearApiKey();
    window.dispatchEvent(new Event('auth-change'));
  }
  return res;
}
