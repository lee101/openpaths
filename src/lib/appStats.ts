export type AppModelUsage = {
  model: string;
  provider: string;
  requests: number;
  tokens_in: number;
  tokens_out: number;
  total_tokens: number;
  source: string;
};

export type AppUsageStats = {
  app_id: string;
  slug: string;
  name: string;
  url: string;
  description: string;
  favicon_url: string;
  categories: string[];
  source: string;
  total_requests: number;
  total_tokens: number;
  models: AppModelUsage[];
};

export function formatTokens(value: number) {
  if (value >= 1e12) return `${(value / 1e12).toFixed(2)}T`;
  if (value >= 1e9) return `${(value / 1e9).toFixed(2)}B`;
  if (value >= 1e6) return `${(value / 1e6).toFixed(2)}M`;
  if (value >= 1e3) return `${(value / 1e3).toFixed(1)}K`;
  return value.toLocaleString('en-US');
}

export function host(url: string) {
  try {
    return new URL(url).hostname.replace(/^www\./, '');
  } catch {
    return url;
  }
}

export function sourceLabel(source: string) {
  if (source === 'openrouter') return 'OpenRouter';
  if (source === 'openpaths') return 'OpenPaths';
  return source || 'tracked';
}

export function appOgImage(slug: string) {
  return `/og/apps/${encodeURIComponent(slug)}.svg`;
}
