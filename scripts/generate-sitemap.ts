import { writeFileSync } from 'node:fs';
import { posts } from '../src/data/blog';
import { models } from '../src/data/models';
import { providers } from '../src/data/providers';
import { artificialAnalysisModels } from '../src/lib/artificialAnalysis';
import { seedApps } from '../src/data/seedApps';

const BASE_URL = 'https://openpaths.io';

type SitemapEntry = {
  path: string;
  changefreq: 'daily' | 'weekly' | 'monthly' | 'yearly';
  priority: string;
};

const entries: SitemapEntry[] = [
  { path: '/', changefreq: 'weekly', priority: '1.0' },
  { path: '/models', changefreq: 'weekly', priority: '0.9' },
  { path: '/pricing', changefreq: 'weekly', priority: '0.9' },
  { path: '/providers', changefreq: 'weekly', priority: '0.8' },
  { path: '/docs', changefreq: 'weekly', priority: '0.8' },
  { path: '/integrations', changefreq: 'weekly', priority: '0.8' },
  { path: '/playground', changefreq: 'monthly', priority: '0.5' },
  { path: '/image-to-3d', changefreq: 'monthly', priority: '0.7' },
  { path: '/search', changefreq: 'monthly', priority: '0.7' },
  { path: '/stats', changefreq: 'weekly', priority: '0.6' },
  { path: '/apps/', changefreq: 'daily', priority: '0.7' },
  { path: '/evals', changefreq: 'weekly', priority: '0.8' },
  { path: '/compare', changefreq: 'weekly', priority: '0.8' },
  { path: '/blog', changefreq: 'weekly', priority: '0.8' },
  { path: '/alternatives', changefreq: 'weekly', priority: '0.8' },
  ...artificialAnalysisModels.slice(1, 12).map(model => ({
    path: `/compare/${encodeURIComponent(artificialAnalysisModels[0].slug)}-vs-${encodeURIComponent(model.slug)}`,
    changefreq: 'weekly' as const,
    priority: '0.7',
  })),
  ...providers.map(provider => ({
    path: `/providers/${encodeURIComponent(provider.slug)}`,
    changefreq: 'weekly' as const,
    priority: provider.featured ? '0.8' : '0.7',
  })),
  ...providers.map(provider => ({
    path: `/${encodeURIComponent(provider.slug)}/docs`,
    changefreq: 'weekly' as const,
    priority: provider.featured ? '0.8' : '0.7',
  })),
  ...seedApps.map(app => ({
    path: `/apps/${encodeURIComponent(app.slug)}/`,
    changefreq: 'daily' as const,
    priority: app.total_tokens >= 1_000_000_000_000 ? '0.8' : '0.7',
  })),
  ...models.map(model => ({
    path: `/models/${encodeURIComponent(model.id)}`,
    changefreq: 'weekly' as const,
    priority: model.popularity <= 0 ? '0.7' : '0.6',
  })),
  ...posts.map(post => ({
    path: `/blog/${encodeURIComponent(post.slug)}`,
    changefreq: 'monthly' as const,
    priority: '0.7',
  })),
  ...posts.flatMap(post => post.alternativePath ? [{
    path: post.alternativePath,
    changefreq: 'monthly' as const,
    priority: '0.7',
  }] : []),
];

const seen = new Set<string>();
const uniqueEntries = entries.filter(entry => {
  if (seen.has(entry.path)) return false;
  seen.add(entry.path);
  return true;
});

const xml = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${uniqueEntries.map(entry => `  <url>
    <loc>${escapeXml(`${BASE_URL}${entry.path}`)}</loc>
    <changefreq>${entry.changefreq}</changefreq>
    <priority>${entry.priority}</priority>
  </url>`).join('\n')}
</urlset>
`;

writeFileSync('public/sitemap.xml', xml);
console.log(`Wrote public/sitemap.xml with ${uniqueEntries.length} URLs`);

function escapeXml(value: string) {
  return value
    .replaceAll('&', '&amp;')
    .replaceAll('"', '&quot;')
    .replaceAll("'", '&apos;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;');
}
