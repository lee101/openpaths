export const SITE_URL = 'https://openpaths.io';

type Json = Record<string, unknown>;

const SEGMENT_LABELS: Record<string, string> = {
  models: 'Models',
  docs: 'Docs',
  'image-generation': 'Image Generation',
  providers: 'Providers',
  tools: 'Tools',
  blog: 'Blog',
  'use-cases': 'Use Cases',
  prompts: 'Prompts',
  compare: 'Compare',
  apps: 'Apps',
  art: 'Art',
  mcp: 'MCP',
  byok: 'BYOK',
  faq: 'FAQ',
};

export function segmentLabel(segment: string) {
  let decoded = segment;
  try { decoded = decodeURIComponent(segment); } catch { /* keep raw */ }
  return SEGMENT_LABELS[decoded] || decoded.replace(/[-_]+/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
}

export function breadcrumbLd(path: string, leafName?: string): Json | null {
  const clean = path.split('?')[0].replace(/\/+$/, '');
  const segments = clean.split('/').filter(Boolean);
  if (segments.length === 0) return null;
  const items: Json[] = [{ '@type': 'ListItem', position: 1, name: 'OpenPaths', item: `${SITE_URL}/` }];
  segments.forEach((segment, i) => {
    const last = i === segments.length - 1;
    items.push({
      '@type': 'ListItem',
      position: i + 2,
      name: last && leafName ? leafName : segmentLabel(segment),
      item: `${SITE_URL}/${segments.slice(0, i + 1).join('/')}`,
    });
  });
  return { '@context': 'https://schema.org', '@type': 'BreadcrumbList', itemListElement: items };
}

export function faqLd(items: Array<{ q: string; a: string }>): Json {
  return {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: items.map(item => ({
      '@type': 'Question',
      name: item.q,
      acceptedAnswer: { '@type': 'Answer', text: item.a },
    })),
  };
}

export function softwareApplicationLd(opts: {
  name: string;
  description: string;
  path: string;
  category?: string;
  price?: number;
  image?: string;
}): Json {
  return {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: opts.name,
    description: opts.description,
    url: `${SITE_URL}${opts.path}`,
    applicationCategory: opts.category || 'DeveloperApplication',
    operatingSystem: 'Web',
    ...(opts.image ? { image: opts.image } : {}),
    offers: {
      '@type': 'Offer',
      price: String(opts.price ?? 0),
      priceCurrency: 'USD',
    },
  };
}

export function hasType(data: unknown, type: string): boolean {
  if (Array.isArray(data)) return data.some(item => hasType(item, type));
  if (data && typeof data === 'object') {
    const t = (data as Json)['@type'];
    if (t === type || (Array.isArray(t) && t.includes(type))) return true;
    const graph = (data as Json)['@graph'];
    return Array.isArray(graph) && graph.some(item => hasType(item, type));
  }
  return false;
}

export function withBreadcrumbs(jsonLd: unknown, path: string, leafName?: string): unknown[] {
  const list = jsonLd ? (Array.isArray(jsonLd) ? jsonLd : [jsonLd]) : [];
  if (hasType(list, 'BreadcrumbList')) return list;
  const crumbs = breadcrumbLd(path, leafName);
  return crumbs ? [...list, crumbs] : list;
}
