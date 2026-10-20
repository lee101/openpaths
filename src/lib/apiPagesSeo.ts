import type { Model } from '../data/models';
import { CAPABILITIES, VS_PAIRS, capabilityBySlug, capabilityModels, type Capability, type VsPair } from '../data/capabilities';
import { formatCurrency, isFreeModel, isTokenPriced, priceSummary } from './modelPricing';
import { modelPath } from './paths';

export const BASE_URL = 'https://openpaths.io';

export const HUB_SEO = {
  path: '/api',
  title: 'OpenAI-Compatible AI APIs: Chat, Image, Video, Audio, Search | OpenPaths',
  description: 'One OpenAI- and Anthropic-compatible key for chat, embeddings, image, video, music, speech, 3D and search APIs. Live pricing for every model and free LLM options.',
};

export function workload() {
  return {
    label: '10M input + 2M output tokens',
    cost: (m: Model) => (10 * m.priceInput) + (2 * m.priceOutput),
  };
}

export function vsSeo(p: VsPair) {
  return {
    path: `/vs/${p.slug}`,
    title: `${p.a.name} vs ${p.b.name}: API Pricing & Comparison | OpenPaths`,
    description: `${p.a.name} vs ${p.b.name} on one OpenAI-compatible API: ${vsVerdict(p.a, p.b)} Compare context, price and try both live.`.slice(0, 300),
  };
}

export function vsVerdict(a: Model, b: Model) {
  if (isFreeModel(a) || isFreeModel(b)) {
    const [free, paid] = isFreeModel(a) ? [a, b] : [b, a];
    return isFreeModel(paid)
      ? `Both ${a.name} and ${b.name} bill $0 per token.`
      : `${free.name} bills $0 per token while ${paid.name} costs ${priceSummary(paid)}.`;
  }
  if (isTokenPriced(a) && isTokenPriced(b)) {
    const w = workload();
    const [ca, cb] = [w.cost(a), w.cost(b)];
    const [cheap, dear, cc, dc] = ca <= cb ? [a, b, ca, cb] : [b, a, cb, ca];
    const pct = dc > 0 ? Math.round((1 - cc / dc) * 100) : 0;
    const ctx = a.contextLength !== 'N/A' && b.contextLength !== 'N/A' ? ` Context windows: ${a.name} ${a.contextLength}, ${b.name} ${b.contextLength}.` : '';
    return pct > 0
      ? `For ${w.label}, ${cheap.name} costs ${formatCurrency(cc)} versus ${formatCurrency(dc)} for ${dear.name} (${pct}% cheaper).${ctx}`
      : `${a.name} and ${b.name} cost the same for ${w.label}.${ctx}`;
  }
  return `${a.name} is ${priceSummary(a)}; ${b.name} is ${priceSummary(b)}.`;
}

export function fromPrice(cap: Capability) {
  const paid = capabilityModels(cap).filter(m => !isFreeModel(m));
  if (!paid.length) return '';
  const cheapest = [...paid].sort((x, y) => x.priceInput - y.priceInput)[0];
  return `${formatCurrency(cheapest.priceInput)} ${cheapest.pricingType && cheapest.pricingType !== 'token' ? priceSummary(cheapest).replace(/^\S+ /, '') : '/ 1M tokens'}`;
}

const crumb = (items: Array<[string, string]>) => ({
  '@context': 'https://schema.org',
  '@type': 'BreadcrumbList',
  itemListElement: items.map(([name, path], i) => ({ '@type': 'ListItem', position: i + 1, name, item: `${BASE_URL}${path}` })),
});

export function hubJsonLd() {
  return [
    crumb([['Home', '/'], ['API', '/api']]),
    {
      '@context': 'https://schema.org',
      '@type': 'ItemList',
      name: 'OpenPaths APIs',
      itemListElement: CAPABILITIES.map((c, i) => ({ '@type': 'ListItem', position: i + 1, url: `${BASE_URL}${c.path}`, name: c.name })),
    },
  ];
}

export function capabilityJsonLd(cap: Capability) {
  return [
    crumb([['Home', '/'], ['API', '/api'], [cap.name, cap.path]]),
    {
      '@context': 'https://schema.org',
      '@type': 'FAQPage',
      mainEntity: cap.faq.map(f => ({ '@type': 'Question', name: f.q, acceptedAnswer: { '@type': 'Answer', text: f.a } })),
    },
    {
      '@context': 'https://schema.org',
      '@type': 'ItemList',
      name: `${cap.name} models`,
      itemListElement: capabilityModels(cap).slice(0, 25).map((m, i) => ({ '@type': 'ListItem', position: i + 1, url: `${BASE_URL}${modelPath(m.id)}`, name: m.name })),
    },
  ];
}

export function vsJsonLd(p: VsPair) {
  return [
    crumb([['Home', '/'], ['API', '/api'], [p.cap.name, p.cap.path], [`${p.a.name} vs ${p.b.name}`, `/vs/${p.slug}`]]),
  ];
}

/* ---------- static HTML for prerendered routes (crawlers without JS) ---------- */

const esc = (v: string) => v.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;');
const link = (href: string, label: string) => `<a href="${esc(href)}" class="underline">${esc(label)}</a>`;
const shell = (inner: string) => `<div id="static-shell" class="bg-black text-white font-sans min-h-screen"><main class="mx-auto max-w-5xl px-6 py-12">${inner}</main></div>`;

function modelTable(list: Model[], limit = 40) {
  const rows = list.slice(0, limit).map(m => `<tr><td>${link(modelPath(m.id), m.name)} <code>${esc(m.id)}</code></td><td>${esc(m.provider)}</td><td>${esc(m.contextLength)}</td><td>${esc(priceSummary(m))}</td></tr>`).join('');
  return `<table><thead><tr><th>Model</th><th>Provider</th><th>Context</th><th>Price</th></tr></thead><tbody>${rows}</tbody></table>`;
}

export interface PrerenderRoute {
  path: string;
  title: string;
  description: string;
  html: string;
}

export function apiPrerenderRoutes(): PrerenderRoute[] {
  const out: PrerenderRoute[] = [];
  out.push({
    path: HUB_SEO.path,
    title: HUB_SEO.title,
    description: HUB_SEO.description,
    html: shell(`<h1>One API key for every model</h1><p>${esc(HUB_SEO.description)}</p><ul>${CAPABILITIES.map(c => `<li>${link(c.path, c.name)} - ${esc(c.tagline)}</li>`).join('')}</ul><p>${link('/pricing', 'Pricing')} ${link('/models', 'All models')} ${link('/docs', 'Docs')}</p>`),
  });
  for (const cap of CAPABILITIES) {
    const list = capabilityModels(cap);
    const pairs = VS_PAIRS.filter(p => p.cap.slug === cap.slug).slice(0, 12);
    const static_ = (cap.staticRows || []).map(r => `<li>${esc(r.name)} (${esc(r.provider)}): ${esc(r.price)}</li>`).join('');
    out.push({
      path: cap.path,
      title: cap.seoTitle,
      description: cap.seoDescription,
      html: shell(
        `<nav>${link('/', 'Home')} / ${link('/api', 'API')} / ${esc(cap.name)}</nav><h1>${esc(cap.h1)}</h1><p>${esc(cap.intro)}</p>`
        + (list.length ? `<h2>${esc(cap.name)} models and pricing</h2>${modelTable(list)}` : '')
        + (static_ ? `<ul>${static_}</ul>` : '')
        + (pairs.length ? `<h2>Comparisons</h2><ul>${pairs.map(p => `<li>${link(`/vs/${p.slug}`, `${p.a.name} vs ${p.b.name}`)}</li>`).join('')}</ul>` : '')
        + `<h2>FAQ</h2>${cap.faq.map(f => `<h3>${esc(f.q)}</h3><p>${esc(f.a)}</p>`).join('')}`
        + `<p>${cap.related.map(s => capabilityBySlug.get(s)).filter(Boolean).map(r => link(r!.path, r!.name)).join(' ')}</p>`,
      ),
    });
  }
  for (const p of VS_PAIRS) {
    const seo = vsSeo(p);
    out.push({
      path: seo.path,
      title: seo.title,
      description: seo.description,
      html: shell(
        `<nav>${link('/', 'Home')} / ${link('/api', 'API')} / ${link(p.cap.path, p.cap.name)}</nav><h1>${esc(p.a.name)} vs ${esc(p.b.name)}</h1><p>${esc(vsVerdict(p.a, p.b))}</p>`
        + modelTable([p.a, p.b])
        + `<p>${link(modelPath(p.a.id), p.a.name)} ${link(modelPath(p.b.id), p.b.name)} ${link(p.cap.path, `${p.cap.name} API`)}</p>`,
      ),
    });
  }
  return out;
}

export function apiSitemapPaths(): Array<{ path: string; priority: string }> {
  return [
    { path: HUB_SEO.path, priority: '0.8' },
    ...CAPABILITIES.map(c => ({ path: c.path, priority: '0.8' })),
    ...VS_PAIRS.map(p => ({ path: `/vs/${p.slug}`, priority: '0.6' })),
  ];
}
