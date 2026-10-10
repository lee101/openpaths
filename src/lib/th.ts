type ThProps = Record<string, string | number | boolean | undefined>;

declare global {
  interface Window {
    th?: (...args: unknown[]) => void;
  }
}

export function track(event: string, props?: ThProps) {
  if (typeof window === 'undefined') return;
  try {
    window.th?.('track', event, props);
  } catch {}
}

export function identify(userId?: string | number | null) {
  if (typeof window === 'undefined' || userId == null || userId === '') return;
  try {
    window.th?.('identify', String(userId));
  } catch {}
}

export function trackAuthStarted(mode: 'login' | 'register', surface: string) {
  track(mode === 'register' ? 'sign_up_started' : 'sign_in_started', { method: 'email', surface });
}

export function trackAuthCompleted(mode: 'login' | 'register', surface: string, userId?: string | number | null) {
  identify(userId);
  track(mode === 'register' ? 'sign_up_completed' : 'sign_in_completed', { method: 'email', surface });
}
