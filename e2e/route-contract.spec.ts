import { expect, test } from '@playwright/test';
import { seedApps } from '../src/data/seedApps';

const BASE_URL = 'https://openpaths.io';

test.describe('Route Contract', () => {
  test('sitemap app routes use deployed canonical URLs', async ({ request }) => {
    const response = await request.get('/sitemap.xml');
    expect(response.ok()).toBeTruthy();

    const xml = await response.text();
    expect(xml).toContain(`${BASE_URL}/apps/`);
    for (const app of seedApps) {
      expect(xml).toContain(`${BASE_URL}/apps/${encodeURIComponent(app.slug)}/`);
    }
  });

  test('every sitemap route is served by the frontend host', async ({ request }) => {
    const response = await request.get('/sitemap.xml');
    expect(response.ok()).toBeTruthy();

    const xml = await response.text();
    const routes = [...xml.matchAll(/<loc>([^<]+)<\/loc>/g)]
      .map(match => match[1])
      .filter((loc): loc is string => Boolean(loc))
      .map(loc => new URL(loc).pathname);

    expect(routes.length).toBeGreaterThan(50);

    const failures: string[] = [];
    for (const route of routes) {
      const routeResponse = await request.get(route);
      if (!routeResponse.ok()) {
        failures.push(`${route} -> ${routeResponse.status()}`);
      }
    }

    expect(failures).toEqual([]);
  });

  test('browser-rendered sitemap routes do not miss React routing', async ({ page }) => {
    const routeErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error' && msg.text().includes('No routes matched location')) {
        routeErrors.push(msg.text());
      }
    });

    await page.route('**/stats/apps?*', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ period: '30d', limit: 100, apps: [] }),
    }));
    await page.route('**/stats/apps/hermes-agent?*', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        period: '30d',
        app: {
          app_id: 'app-hermes',
          slug: 'hermes-agent',
          name: 'Hermes Agent',
          url: 'https://nousresearch.com',
          description: 'Hermes Agent route smoke test.',
          favicon_url: '/favicon.ico',
          categories: ['cli-agents'],
          source: 'openrouter',
          total_requests: 1,
          total_tokens: 8_520_000_000_000,
          models: [],
        },
      }),
    }));

    const routes = [
      '/',
      '/models',
      '/providers',
      '/docs',
      '/integrations',
      '/pricing',
      '/stats',
      '/apps/',
      '/apps/hermes-agent/',
      '/evals',
      '/compare',
      '/blog',
      '/alternatives',
    ];

    for (const route of routes) {
      await page.goto(route);
      await page.waitForLoadState('networkidle');
    }

    expect(routeErrors).toEqual([]);
  });
});
