import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import {
  artificialAnalysisModels,
  findArtificialAnalysisModel,
} from '../src/lib/artificialAnalysis';
import { seedApps } from '../src/data/seedApps';
import { appOgImage } from '../src/lib/appStats';

const BASE_URL = 'https://openpaths.io';
const DIST_DIR = 'dist';

type StaticMeta = {
  path: string;
  title: string;
  description: string;
  image?: string;
};

const indexHtml = readFileSync(join(DIST_DIR, 'index.html'), 'utf8');

const routes: StaticMeta[] = [
  {
    path: '/art',
    title: 'ZImage Prompt Search | OpenPaths',
    description: 'Browse and search a large ZImage generated-art prompt index, then try any prompt against OpenPaths image generation models.',
  },
  {
    path: '/evals',
    title: 'AI Model Evals, Pricing, and Speed | OpenPaths',
    description: 'Compare frontier AI model intelligence, coding, agentic performance, speed, and token pricing using the OpenPaths Artificial Analysis benchmark snapshot.',
  },
  {
    path: '/compare',
    title: 'Compare AI Models by Evals, Speed, and Price | OpenPaths',
    description: 'Compare frontier AI models head-to-head using Artificial Analysis evals, pricing, context windows, speed, and benchmark run costs.',
  },
  {
    path: '/stats',
    title: 'OpenPaths Stats | AI Model Usage',
    description: 'Daily model usage and public OpenPaths request breakdowns.',
  },
  {
    path: '/apps/',
    title: 'Apps | OpenPaths App Usage Stats',
    description: 'Opt-in app and agent usage stats across OpenPaths and OpenRouter, including model and token breakdowns.',
    image: appOgImage('openpaths-apps'),
  },
  ...seedApps.map(app => ({
    path: `/apps/${app.slug}/`,
    title: `${app.name} usage stats | OpenPaths Apps`,
    description: `${app.name} model and token usage on OpenPaths, including top models, total tokens, and opt-in attribution stats.`,
    image: appOgImage(app.slug),
  })),
  ...compareRoutes(),
];

const seen = new Set<string>();
let written = 0;
const manifest: Array<{ path: string; source: string; key: string }> = [];

for (const route of routes) {
  if (seen.has(route.path)) continue;
  seen.add(route.path);
  const html = withMeta(indexHtml, route);
  manifest.push(writeRouteFile(route.path, html));
  written += 1;
}

writeFileSync(join(DIST_DIR, 'prerender-manifest.json'), JSON.stringify(manifest, null, 2));
console.log(`Prerendered static metadata for ${written} routes`);

function compareRoutes(): StaticMeta[] {
  const models = artificialAnalysisModels.filter(model => typeof model.evaluations.intelligenceIndex === 'number');
  const anchor = models[0];
  if (!anchor) return [];

  return [
    compareMeta('/compare/gpt-5.5-vs-opus4.7'),
    ...models.slice(1, 12).map(model => compareMeta(`/compare/${anchor.slug}-vs-${model.slug}`)),
  ];
}

function compareMeta(path: string): StaticMeta {
  const names = path.replace(/^\/compare\//, '')
    .split('-vs-')
    .flatMap(token => {
      const name = findArtificialAnalysisModel(token)?.shortName;
      return name ? [String(name)] : [];
    });
  const subject = names.length >= 2 ? formatNameList(names) : 'AI models';

  return {
    path,
    title: `${subject} | AI Model Comparison | OpenPaths`,
    description: `Compare ${subject} using Artificial Analysis evals, speed, context, and token pricing.`,
  };
}

function writeRouteFile(path: string, html: string) {
  const key = path.replace(/^\//, '') || 'index.html';
  const sourcePath = join(DIST_DIR, 'prerender-routes', `${key.replace(/[^a-z0-9.-]+/gi, '__')}.html`);
  mkdirSync(dirname(sourcePath), { recursive: true });
  writeFileSync(sourcePath, html);

  const indexPath = join(DIST_DIR, key, 'index.html');
  mkdirSync(dirname(indexPath), { recursive: true });
  writeFileSync(indexPath, html);

  return {
    path,
    source: sourcePath,
    key,
  };
}

function withMeta(html: string, meta: StaticMeta) {
  const url = `${BASE_URL}${meta.path}`;
  const image = meta.image ? absoluteUrl(meta.image) : `${BASE_URL}/og-image.png`;
  let out = html;
  const replacements: Array<[RegExp, string]> = [
    [/<title>.*?<\/title>/is, `<title>${escapeHtml(meta.title)}</title>`],
    [/<meta\s+name=["']description["']\s+content=["'][^"']*["']\s*\/?>/is, `<meta name="description" content="${escapeHtml(meta.description)}" />`],
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
      ? out.replace(pattern, replacement)
      : out.replace('</head>', `${replacement}\n</head>`);
  }

  return out;
}

function absoluteUrl(value: string) {
  if (/^https?:\/\//i.test(value)) return value;
  return `${BASE_URL}${value.startsWith('/') ? value : `/${value}`}`;
}

function escapeHtml(value: string) {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&#39;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;');
}

function formatNameList(names: string[]) {
  if (names.length <= 2) return names.join(' and ');
  return `${names.slice(0, -1).join(', ')}, and ${names[names.length - 1]}`;
}
