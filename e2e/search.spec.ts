import { test, expect } from '@playwright/test';

test.describe('Search Playground', () => {
  test('renders from nav and generates Exa curl', async ({ page }) => {
    await page.goto('/');
    await page.click('nav >> text=Search');
    await expect(page).toHaveURL('/search');
    await expect(page.locator('h1')).toContainText('Exa search playground');
    await expect(page.locator('text=Search API')).toBeVisible();
    await expect(page.locator('text=Estimated cost')).toBeVisible();
    await expect(page.locator('pre')).toContainText('/v1/search');
    await expect(page.locator('pre')).toContainText('"query": "Latest news on Nvidia"');
  });

  test('submits mocked search and renders results', async ({ page }) => {
    let requestBody: any = null;
    await page.route('**/v1/search', async route => {
      requestBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          requestId: 'pw_req_1',
          resolvedSearchType: 'auto',
          results: [
            {
              id: 'https://example.test/nvidia',
              title: 'Nvidia announces new platform',
              url: 'https://example.test/nvidia',
              highlights: ['Nvidia introduced a new AI platform in the latest company update.'],
              favicon: 'https://example.test/favicon.ico',
            },
          ],
        }),
      });
    });

    await page.goto('/search');
    await page.locator('input[placeholder="op-..."]').fill('op-search-test-key');
    await page.getByRole('button', { name: 'Fast 450ms' }).click();
    await page.locator('select').first().selectOption('news article');
    await page.getByRole('button', { name: 'Run Search' }).click();

    await expect(page.locator('text=requestId: pw_req_1')).toBeVisible();
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
});
