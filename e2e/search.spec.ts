import { test, expect } from '@playwright/test';

test.describe('Search Playground', () => {
  test('renders from nav with Gemini default and a code example', async ({ page }) => {
    await page.goto('/');
    // Search lives behind the nav menu button, not the core top bar.
    await page.getByRole('button', { name: 'Open menu' }).click();
    await page.locator('nav a[href="/search"]').last().click();
    await expect(page).toHaveURL('/search');
    await expect(page.locator('h1')).toContainText('Search the web with AI');
    // Provider tabs are present and Gemini is the default engine.
    await expect(page.getByRole('button', { name: 'Gemini Flash' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Exa' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Papers' })).toBeVisible();
    // Code example reflects the Gemini request body.
    await expect(page.locator('pre')).toContainText('/v1/search');
    await expect(page.locator('pre')).toContainText('"provider": "gemini"');
    await expect(page.locator('pre')).toContainText('"query": "Latest news on Nvidia"');
  });

  test('runs a mocked Gemini answer with sources', async ({ page }) => {
    let requestBody: any = null;
    await page.route('**/v1/search', async route => {
      requestBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          provider: 'gemini',
          model: 'gemini-3.5-flash',
          query: 'Latest news on Nvidia',
          answer: 'Nvidia announced a new AI platform this week.',
          searchQueries: ['nvidia latest news'],
          citations: [{ title: 'Nvidia News', url: 'https://example.test/nvidia' }],
          results: [{ title: 'Nvidia News', url: 'https://example.test/nvidia' }],
        }),
      });
    });

    await page.goto('/search');
    await page.locator('input[placeholder="op-..."]').fill('op-search-test-key');
    await page.getByRole('button', { name: 'Search', exact: true }).click();

    await expect(page.locator('text=Nvidia announced a new AI platform')).toBeVisible();
    await expect(page.locator('text=Sources')).toBeVisible();
    await expect(page.locator('a', { hasText: 'Nvidia News' })).toBeVisible();
    expect(requestBody).toMatchObject({ provider: 'gemini', query: 'Latest news on Nvidia' });
  });

  test('switches to Exa and renders mocked results', async ({ page }) => {
    let requestBody: any = null;
    await page.route('**/v1/search', async route => {
      requestBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          requestId: 'pw_req_1',
          resolvedSearchType: 'fast',
          results: [
            {
              id: 'https://example.test/nvidia',
              title: 'Nvidia announces new platform',
              url: 'https://example.test/nvidia',
              highlights: ['Nvidia introduced a new AI platform in the latest company update.'],
            },
          ],
        }),
      });
    });

    await page.goto('/search');
    await page.locator('input[placeholder="op-..."]').fill('op-search-test-key');
    await page.getByRole('button', { name: 'Exa', exact: true }).click();
    await page.getByRole('button', { name: 'Fast 450ms' }).click();
    await page.locator('select').first().selectOption('news article');
    await page.getByRole('button', { name: 'Search', exact: true }).click();

    await expect(page.locator('text=Nvidia announces new platform')).toBeVisible();
    await expect(page.locator('text=Nvidia introduced a new AI platform')).toBeVisible();
    expect(requestBody).toMatchObject({
      query: 'Latest news on Nvidia',
      numResults: 10,
      type: 'fast',
      category: 'news article',
      contents: { highlights: true },
    });
  });

  test('switches to Papers and renders markdown', async ({ page }) => {
    let requestBody: any = null;
    await page.route('**/v1/search', async route => {
      requestBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        contentType: 'text/markdown; charset=utf-8',
        body: '# Papers Search\n\n- Query: diffusion transformers\n- Count: 1\n',
      });
    });

    await page.goto('/search');
    await page.locator('input[placeholder="op-..."]').fill('op-search-test-key');
    await page.getByRole('button', { name: 'Papers', exact: true }).click();
    await page.locator('input[aria-label="Search query"]').fill('diffusion transformers');
    await page.getByRole('button', { name: 'Search', exact: true }).click();

    await expect(page.locator('pre').filter({ hasText: 'Papers Search' })).toBeVisible();
    expect(requestBody).toMatchObject({
      provider: 'papers',
      query: 'diffusion transformers',
      numResults: 10,
      type: 'papers',
      format: 'markdown',
    });
  });
});
