import { test, expect } from '@playwright/test';

const appsPayload = {
  period: '30d',
  limit: 100,
  apps: [
    {
      app_id: 'app-hermes',
      slug: 'hermes-agent',
      name: 'Hermes Agent',
      url: 'https://nousresearch.com',
      description: 'An open-source persistent AI agent with memory and reusable skills.',
      favicon_url: '/favicon.ico',
      categories: ['personal-agents', 'cli-agents'],
      source: 'openrouter',
      total_requests: 4200,
      total_tokens: 8_520_000_000_000,
      models: [
        {
          model: 'openai/gpt-oss-120b',
          provider: 'openrouter',
          requests: 0,
          tokens_in: 0,
          tokens_out: 0,
          total_tokens: 2_500_000_000_000,
          source: 'openrouter',
        },
      ],
    },
    {
      app_id: 'app-openpaths',
      slug: 'openpaths-test-client',
      name: 'OpenPaths Test Client',
      url: 'https://example.test/app',
      description: 'An opt-in client sending attribution headers.',
      favicon_url: '/favicon.ico',
      categories: ['programming-app'],
      source: 'openpaths',
      total_requests: 12,
      total_tokens: 34_500,
      models: [
        {
          model: 'gpt-5.4',
          provider: 'openai',
          requests: 12,
          tokens_in: 12000,
          tokens_out: 22500,
          total_tokens: 34500,
          source: 'openpaths',
        },
      ],
    },
  ],
};

test.describe('Apps Page', () => {
  test.beforeEach(({ page }) => {
    page.on('console', msg => {
      if (msg.type() === 'error' && msg.text().includes('No routes matched location')) {
        throw new Error(msg.text());
      }
    });
  });

  test('renders app usage stats from /stats/apps', async ({ page }) => {
    const requestedPeriods: string[] = [];
    await page.route('**/stats/apps?*', async route => {
      const url = new URL(route.request().url());
      requestedPeriods.push(url.searchParams.get('period') || '');
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ ...appsPayload, period: url.searchParams.get('period') || '30d' }),
      });
    });

    await page.goto('/apps');

    await expect(page.locator('h1')).toContainText('Apps And Agents');
    await expect(page.getByText('Opt-in usage tracking')).toBeVisible();
    await expect(page.getByText('Hermes Agent')).toBeVisible();
    await expect(page.getByText('nousresearch.com')).toBeVisible();
    await expect(page.getByText('OpenPaths Test Client')).toBeVisible();
    await expect(page.getByText('8.52T').first()).toBeVisible();
    await expect(page.getByText('openai/gpt-oss-120b')).toBeVisible();
    await expect(page.getByText('2.50T')).toBeVisible();
    await expect(page.getByText('gpt-5.4')).toBeVisible();
    await expect(page.getByText('34.5K').first()).toBeVisible();

    await page.getByRole('button', { name: '7d' }).click();
    await expect.poll(() => requestedPeriods.at(-1)).toBe('7d');
  });

  test('shows empty state when there is no app usage', async ({ page }) => {
    await page.route('**/stats/apps?*', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ period: '30d', limit: 100, apps: [] }),
    }));

    await page.goto('/apps');

    await expect(page.locator('h1')).toContainText('Apps And Agents');
    await expect(page.getByText('No app usage has been recorded for this period.')).toBeVisible();
  });

  test('renders app detail page with app-specific metadata', async ({ page }) => {
    await page.route('**/stats/apps/hermes-agent?*', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ period: '30d', app: appsPayload.apps[0] }),
    }));

    await page.goto('/apps/hermes-agent');

    await expect(page.locator('h1')).toContainText('Hermes Agent');
    await expect(page.getByText('nousresearch.com')).toBeVisible();
    await expect(page.getByText('8.52T')).toBeVisible();
    await expect(page.getByText('openai/gpt-oss-120b')).toBeVisible();
    await expect(page.locator('meta[property="og:image"]')).toHaveAttribute('content', 'https://openpaths.io/og/apps/hermes-agent.svg');
    await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', 'https://openpaths.io/apps/hermes-agent/');
  });

  test('supports the deployed trailing-slash app URLs', async ({ page }) => {
    await page.route('**/stats/apps?*', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify(appsPayload),
    }));
    await page.route('**/stats/apps/hermes-agent?*', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ period: '30d', app: appsPayload.apps[0] }),
    }));

    await page.goto('/apps/');
    await expect(page).toHaveURL('/apps/');
    await expect(page.locator('h1')).toContainText('Apps And Agents');

    await page.getByRole('link', { name: /Hermes Agent/i }).first().click();
    await expect(page).toHaveURL('/apps/hermes-agent/');
    await expect(page.locator('h1')).toContainText('Hermes Agent');

    await page.goto('/apps/hermes-agent/');
    await expect(page.locator('h1')).toContainText('Hermes Agent');
    await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', 'https://openpaths.io/apps/hermes-agent/');
  });
});
