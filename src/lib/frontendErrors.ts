type FrontendErrorEvent = {
  source: 'window.error' | 'unhandledrejection' | 'console.error';
  message: string;
  stack?: string;
  filename?: string;
  lineno?: number;
  colno?: number;
  route: string;
  userAgent: string;
  occurredAt: string;
};

const endpoint = '/monitoring/frontend-errors';
let sentCount = 0;
let lastSentAt = 0;

function stringifyReason(reason: unknown): {message: string; stack?: string} {
  if (reason instanceof Error) {
    return {message: reason.message, stack: reason.stack};
  }
  if (typeof reason === 'string') {
    return {message: reason};
  }
  try {
    return {message: JSON.stringify(reason)};
  } catch {
    return {message: String(reason)};
  }
}

function report(event: Omit<FrontendErrorEvent, 'route' | 'userAgent' | 'occurredAt'>) {
  const now = Date.now();
  if (sentCount >= 20 || now - lastSentAt < 2000) {
    return;
  }
  sentCount += 1;
  lastSentAt = now;

  const payload: FrontendErrorEvent = {
    ...event,
    route: `${window.location.pathname}${window.location.search}${window.location.hash}`,
    userAgent: navigator.userAgent,
    occurredAt: new Date().toISOString(),
  };
  const body = JSON.stringify(payload);

  if (navigator.sendBeacon) {
    const blob = new Blob([body], {type: 'application/json'});
    if (navigator.sendBeacon(endpoint, blob)) {
      return;
    }
  }

  void fetch(endpoint, {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body,
    keepalive: true,
    credentials: 'same-origin',
  }).catch(() => {});
}

export function installFrontendErrorReporting() {
  window.addEventListener('error', (event) => {
    report({
      source: 'window.error',
      message: event.message,
      stack: event.error instanceof Error ? event.error.stack : undefined,
      filename: event.filename,
      lineno: event.lineno,
      colno: event.colno,
    });
  });

  window.addEventListener('unhandledrejection', (event) => {
    report({
      source: 'unhandledrejection',
      ...stringifyReason(event.reason),
    });
  });

  const originalConsoleError = console.error.bind(console);
  console.error = (...args: unknown[]) => {
    originalConsoleError(...args);
    const message = args.map((arg) => stringifyReason(arg).message).join(' ');
    const firstError = args.find((arg): arg is Error => arg instanceof Error);
    report({
      source: 'console.error',
      message,
      stack: firstError?.stack,
    });
  };
}
