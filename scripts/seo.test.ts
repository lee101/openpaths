import { afterAll, beforeAll, describe, expect, test } from 'bun:test';
import { cpSync, mkdtempSync, readFileSync, rmSync, mkdirSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { breadcrumbLd, faqLd, hasType, withBreadcrumbs } from '../src/lib/seoSchema';
import { staticBody, withMeta } from './lib/staticMeta';

let dist = '';
let manifest: Array<{ path: string; key: string }> = [];

const read = (path: string) => readFileSync(join(dist, decodeURIComponent(path).replace(/^\//, ''), 'index.html'), 'utf8');
const tag = (html: string, re: RegExp) => html.match(re)?.[1];
const title = (html: string) => tag(html, /<title>(.*?)<\/title>/s);
const canonical = (html: string) => tag(html, /<link rel="canonical" href="([^"]*)"/);
const ldTypes = (html: string) => {
  const types: string[] = [];
  for (const m of html.matchAll(/<script type="application\/ld\+json"[^>]*>(.*?)<\/script>/gs)) {
    const data = JSON.parse(m[1]);
    for (const item of Array.isArray(data) ? data : data['@graph'] || [data]) types.push(item['@type']);
  }
  return types;
};
const h1Count = (html: string) => (html.match(/<h1[ >]/g) || []).length;

beforeAll(() => {
  dist = mkdtempSync(join(tmpdir(), 'seo-dist-'));
  cpSync('index.html', join(dist, 'index.html'));
  const proc = Bun.spawnSync(['bun', 'scripts/prerender-static-meta.ts'], { env: { ...process.env, DIST_DIR: dist }, stderr: 'pipe' });
  if (proc.exitCode !== 0) throw new Error(proc.stderr.toString());
  manifest = JSON.parse(readFileSync(join(dist, 'prerender-manifest.json'), 'utf8'));
});

afterAll(() => rmSync(dist, { recursive: true, force: true }));

describe('prerendered routes', () => {
  test('sample routes carry unique title, canonical, og and twitter tags', () => {
    for (const path of ['/pricing', '/docs', '/docs/image-generation', '/models/gemini-nano-banana-2.1', '/models/gpt-image-2', '/providers/google', '/tools', '/text-to-image', '/blog/gemini-image-models-game-dev-concept-art']) {
      const html = read(path);
      const t = title(html)!;
      expect(t.length).toBeGreaterThan(10);
      expect(canonical(html)).toBe(`https://openpaths.io${path}`);
      expect(tag(html, /<meta property="og:url" content="([^"]*)"/)).toBe(`https://openpaths.io${path}`);
      expect(tag(html, /<meta property="og:title" content="([^"]*)"/)).toContain(t.slice(0, 20));
      expect(tag(html, /<meta name="twitter:title" content="([^"]*)"/)).toBeTruthy();
      expect(tag(html, /<meta name="twitter:image" content="(https:\/\/[^"]*)"/)).toBeTruthy();
      expect(tag(html, /<meta name="description" content="([^"]{40,})"/)).toBeTruthy();
    }
  });

  test('titles and canonicals are unique across every prerendered route', () => {
    const titles = new Map<string, string>();
    const canonicals = new Set<string>();
    for (const { path } of manifest) {
      const html = read(path);
      const t = title(html)!;
      expect(titles.has(t), `duplicate title "${t}" on ${path} and ${titles.get(t)}`).toBe(false);
      titles.set(t, path);
      const c = canonical(html)!;
      expect(canonicals.has(c), `duplicate canonical ${c}`).toBe(false);
      canonicals.add(c);
    }
  });

  test('every non-home route has exactly one h1 in the static body', () => {
    for (const { path } of manifest) expect(h1Count(read(path)), path).toBe(1);
  });

  test('JSON-LD: breadcrumbs everywhere, software application and FAQ on model pages', () => {
    for (const { path } of manifest) expect(ldTypes(read(path)), path).toContain('BreadcrumbList');
    const model = ldTypes(read('/models/gemini-nano-banana-2.1'));
    expect(model).toContain('SoftwareApplication');
    expect(model).toContain('FAQPage');
  });

  test('image docs page lists the gemini image models as internal links', () => {
    const html = read('/docs/image-generation');
    expect(html).toContain('href="/models/gemini-nano-banana-2.1"');
    expect(html).toContain('href="/models/gemini-3.1-flash-lite-image"');
  });

  test('the repo sitemap lists the image docs and every catalog model', () => {
    Bun.spawnSync(['bun', 'scripts/generate-sitemap.ts']);
    const xml = readFileSync('public/sitemap.xml', 'utf8');
    expect(xml).toContain('<loc>https://openpaths.io/docs/image-generation</loc>');
    expect(xml).toContain('<loc>https://openpaths.io/models/gemini-nano-banana-2.1</loc>');
  });
});

describe('withMeta', () => {
  const base = readFileSync('index.html', 'utf8');

  test('home keeps its hero and gets no static body replacement', () => {
    const html = withMeta(base, { path: '/', title: 'Home | OpenPaths', description: 'd' });
    expect(html).toContain('id="static-shell"');
    expect(h1Count(html)).toBe(1);
  });

  test('escapes html in meta and body', () => {
    const html = withMeta(base, { path: '/x', title: '"><script>alert(1)</script>', description: '<b>' });
    expect(html).not.toContain('<script>alert(1)</script>');
    expect(staticBody({ path: '/x', title: '<i>t</i>', description: '<b>' })).not.toContain('<i>');
  });

  test('JSON-LD cannot break out of its script tag', () => {
    const html = withMeta(base, { path: '/x', title: 't', description: 'd', jsonLd: [{ '@type': 'Thing', name: '</script><script>alert(1)</script>' }] });
    expect(html).not.toContain('</script><script>alert(1)');
  });
});

describe('seoSchema', () => {
  test('breadcrumbs follow the path', () => {
    const crumbs = breadcrumbLd('/models/gpt-image-2', 'GPT Image 2') as any;
    expect(crumbs.itemListElement.map((i: any) => i.name)).toEqual(['OpenPaths', 'Models', 'GPT Image 2']);
    expect(crumbs.itemListElement[2].item).toBe('https://openpaths.io/models/gpt-image-2');
    expect(breadcrumbLd('/')).toBeNull();
  });

  test('withBreadcrumbs does not duplicate an existing BreadcrumbList', () => {
    const own = breadcrumbLd('/docs')!;
    expect(withBreadcrumbs(own, '/docs')).toHaveLength(1);
    expect(withBreadcrumbs(faqLd([{ q: 'q', a: 'a' }]), '/docs')).toHaveLength(2);
    expect(hasType([faqLd([])], 'FAQPage')).toBe(true);
  });
});
