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
  window.dispatchEvent(new CustomEvent(AUTH_EVENT));
}

export function onAuthChange(fn: () => void): () => void {
  if (typeof window === 'undefined') return () => {};
  window.addEventListener(AUTH_EVENT, fn);
  return () => window.removeEventListener(AUTH_EVENT, fn);
}

export function requestCreditTopUp() {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new CustomEvent(CREDITS_REQUIRED_EVENT));
}

export async function api(path: string, opts: RequestInit = {}): Promise<Response> {
  const key = getApiKey();
  const headers: Record<string, string> = { 'Content-Type': 'application/json', ...(opts.headers as Record<string, string>) };
  if (key) headers.Authorization = `Bearer ${key}`;
  return fetch(path, { ...opts, headers });
}
