import { breadcrumbLd, faqLd, softwareApplicationLd, SITE_URL } from '../../src/lib/seoSchema';

export type StaticMeta = {
  path: string;
  title: string;
  description: string;
  image?: string;
  // Full static shell (rich body) to prerender instead of the generic staticBody.
  html?: string;
  jsonLd?: unknown[];
  links?: Array<{ href: string; label: string }>;
  type?: 'website' | 'article';
};

const CORE_LINKS = [
  { href: '/models', label: 'Models' },
  { href: '/docs', label: 'API docs' },
  { href: '/docs/image-generation', label: 'Image generation API' },
  { href: '/pricing', label: 'Pricing' },
  { href: '/providers', label: 'Providers' },
  { href: '/tools', label: 'Tools' },
];

export function absoluteUrl(value: string) {
  if (/^https?:\/\//i.test(value)) return value;
  return `${SITE_URL}${value.startsWith('/') ? value : `/${value}`}`;
}

export function escapeHtml(value: string) {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;');
}

export function pageHeading(title: string) {
  return title.replace(/\s*\|\s*OpenPaths\s*$/i, '').trim() || title;
}

export function staticBody(meta: StaticMeta) {
  const links = [...(meta.links || []), ...CORE_LINKS].filter((l, i, all) => l.href !== meta.path && all.findIndex(o => o.href === l.href) === i);
  return `<main id="seo-static"><h1>${escapeHtml(pageHeading(meta.title))}</h1><p>${escapeHtml(meta.description)}</p><nav aria-label="Related pages"><ul>${links.map(l => `<li><a href="${escapeHtml(l.href)}">${escapeHtml(l.label)}</a></li>`).join('')}</ul></nav></main>`;
}

export function jsonLdScript(data: unknown[]) {
  const json = JSON.stringify(data).replaceAll('</', '<\\/');
  return `<script type="application/ld+json" id="seo-jsonld">${json}</script>`;
}

export function withMeta(html: string, meta: StaticMeta) {
  const url = `${SITE_URL}${meta.path}`;
  const image = meta.image ? absoluteUrl(meta.image) : `${SITE_URL}/og-image.png`;
  let out = html;
  const replacements: Array<[RegExp, string]> = [
    [/<title>.*?<\/title>/is, `<title>${escapeHtml(meta.title)}</title>`],
    [/<meta\s+name=["']description["']\s+content=["'][^"']*["']\s*\/?>/is, `<meta name="description" content="${escapeHtml(meta.description)}" />`],
    [/<meta\s+property=["']og:type["']\s+content=["'][^"']*["']\s*\/?>/is, `<meta property="og:type" content="${meta.type || 'website'}" />`],
    [/<meta\s+property=["']og:url["']\s+content=["'][^"']*["']\s*\/?>/is, `<meta property="og:url" content="${escapeHtml(url)}" />`],
    [/<meta\s+property=["']og:title["']\s+content=["'][^"']*["']\s*\/?>/is, `<meta property="og:title" content="${escapeHtml(meta.title)}" />`],
    [/<meta\s+property=["']og:description["']\s+content=["'][^"']*["']\s*\/?>/is, `<meta property="og:description" content="${escapeHtml(meta.description)}" />`],
    [/<meta\s+property=["']og:image["']\s+content=["'][^"']*["']\s*\/?>/is, `<meta property="og:image" content="${escapeHtml(image)}" />`],
    [/<meta\s+name=["']twitter:title["']\s+content=["'][^"']*["']\s*\/?>/is, `<meta name="twitter:title" content="${escapeHtml(meta.title)}" />`],
    [/<meta\s+name=["']twitter:description["']\s+content=["'][^"']*["']\s*\/?>/is, `<meta name="twitter:description" content="${escapeHtml(meta.description)}" />`],
    [/<meta\s+name=["']twitter:image["']\s+content=["'][^"']*["']\s*\/?>/is, `<meta name="twitter:image" content="${escapeHtml(image)}" />`],
    [/<link\s+rel=["']canonical["']\s+href=["'][^"']*["']\s*\/?>/is, `<link rel="canonical" href="${escapeHtml(url)}" />`],
  ];

  for (const [pattern, replacement] of replacements) {
    out = pattern.test(out)
      ? out.replace(pattern, () => replacement)
      : out.replace('</head>', () => `${replacement}\n</head>`);
  }

  const graph = [...(meta.jsonLd || [])];
  if (!graph.some(item => (item as { '@type'?: string })['@type'] === 'BreadcrumbList')) {
    const crumbs = breadcrumbLd(meta.path);
    if (crumbs) graph.push(crumbs);
  }
  if (graph.length) out = out.replace('</head>', () => `${jsonLdScript(graph)}\n</head>`);

  if (meta.path !== '/') {
    const start = out.indexOf('<div id="root">');
    const end = out.indexOf('<script>if(location.pathname');
    if (start >= 0 && end > start) {
      out = `${out.slice(0, start)}<div id="root">${meta.html ?? staticBody(meta)}</div>\n    ${out.slice(end)}`;
    }
  }

  return out;
}

export function modelJsonLd(meta: { id: string; name: string; description: string; price: number; pricingLabel: string }) {
  const path = `/models/${encodeURIComponent(meta.id)}`;
  return [
    softwareApplicationLd({ name: meta.name, description: meta.description, path, price: meta.price }),
    faqLd([
      { q: `What is the model ID for ${meta.name} on OpenPaths?`, a: `Use ${meta.id} as the model field in the OpenPaths API.` },
      { q: `How much does ${meta.name} cost?`, a: `${meta.pricingLabel}, with near-zero OpenPaths markup.` },
    ]),
    breadcrumbLd(path, meta.name),
  ];
}
