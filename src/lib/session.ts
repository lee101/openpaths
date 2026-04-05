export const TOKEN_STORAGE_KEY = 'op_token';
export const USER_STORAGE_KEY = 'op_user';
export const API_KEY_STORAGE_KEY = 'op_api_key';
export const DEFAULT_API_KEY_NAME = 'Dashboard';

export function getStoredToken(): string {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem(TOKEN_STORAGE_KEY) || '';
}

export function getStoredUser<T = any>(): T | null {
  if (typeof window === 'undefined') return null;
  const raw = localStorage.getItem(USER_STORAGE_KEY);
  if (!raw) return null;

  try {
    return JSON.parse(raw) as T;
  } catch {
    return null;
  }
}

export function persistAuth(token: string, user: any) {
  if (typeof window === 'undefined') return;
  localStorage.setItem(TOKEN_STORAGE_KEY, token);
  localStorage.setItem(USER_STORAGE_KEY, JSON.stringify(user));
}

export function clearStoredSession() {
  if (typeof window === 'undefined') return;
  localStorage.removeItem(TOKEN_STORAGE_KEY);
  localStorage.removeItem(USER_STORAGE_KEY);
  localStorage.removeItem(API_KEY_STORAGE_KEY);
}

export function getStoredAPIKey(): string {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem(API_KEY_STORAGE_KEY) || '';
}

export function storeAPIKey(apiKey: string) {
  if (typeof window === 'undefined' || !apiKey) return;
  localStorage.setItem(API_KEY_STORAGE_KEY, apiKey);
}

export function getAppOrigin(): string {
  if (typeof window === 'undefined') return 'https://openpaths.io';
  return window.location.origin;
}

export function getAPIBaseURL(): string {
  return `${getAppOrigin()}/v1`;
}
