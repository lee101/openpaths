import { test, expect } from '@playwright/test';

test.describe('Models Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/models');
  });

  test('page title and description render', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Model Directory');
    await expect(page.locator('text=models from leading providers')).toBeVisible();
  });

  test('search input is present and functional', async ({ page }) => {
    const search = page.locator('input[placeholder*="Search models"]');
    await expect(search).toBeVisible();

    await search.fill('claude');
    await expect(page.locator('text=Claude Sonnet 5')).toBeVisible();
    await expect(page.locator('text=RA1 Art Generator')).not.toBeVisible();
  });

  test('newest Qwen model is listed with its OpenRouter route', async ({ page }) => {
    const search = page.locator('input[placeholder*="Search models"]');
    await search.fill('qwen 3.8');
    await expect(page.getByText('Qwen 3.8 Max')).toBeVisible();
    await expect(page.locator('code:has-text("or/qwen3.8-max")')).toBeVisible();
    await expect(page.getByText('$2.00').first()).toBeVisible();
    await expect(page.getByText('$6.00').first()).toBeVisible();
  });

  test('tag filters work', async ({ page }) => {
    await page.click('button:has-text("art generation")');
    await expect(page.locator('text=RA1 Art Generator')).toBeVisible();
    await expect(page.locator('text=FLUX.1 Pro')).toBeVisible();
    await expect(page.locator('text=Claude Sonnet 5')).not.toBeVisible();
  });

  test('clear filters button works', async ({ page }) => {
    await page.click('button:has-text("programming")');
    const cardsBefore = await page.locator('[class*="rounded-xl"][class*="border"]').count();

    await page.click('text=Clear');
    const cardsAfter = await page.locator('[class*="rounded-xl"][class*="border"]').count();
    expect(cardsAfter).toBeGreaterThanOrEqual(cardsBefore);
  });

  test('model cards show provider, name, pricing, and ID', async ({ page }) => {
    const card = page.locator('text=Claude Sonnet 5').locator('..');
    await expect(card).toBeVisible();
    await expect(page.locator('text=$2.00').first()).toBeVisible();
    await expect(page.locator('text=$10.00').first()).toBeVisible();
    await expect(page.locator('code:has-text("claude-sonnet-5")')).toBeVisible();
  });

  test('no results state shows when search matches nothing', async ({ page }) => {
    const search = page.locator('input[placeholder*="Search models"]');
    await search.fill('xyznonexistentmodel');
    await expect(page.locator('text=No models found')).toBeVisible();
  });

  test('all tag filter buttons are present', async ({ page }) => {
    const tags = ['programming', 'reasoning', 'general', 'vision', 'fast', 'open-source', 'free', 'art generation', 'video generation', 'roleplay'];
    for (const tag of tags) {
      await expect(page.locator(`button:has-text("${tag}")`)).toBeVisible();
    }
  });

  test('sort dropdown is present and functional', async ({ page }) => {
    const sortSelect = page.locator('select');
    await expect(sortSelect).toBeVisible();

    await sortSelect.selectOption('newest');
    const firstCard = page.locator('[class*="rounded-xl"][class*="border"]').first();
    await expect(firstCard).toBeVisible();

    await sortSelect.selectOption('price-low');
    await expect(firstCard).toBeVisible();
  });

  test('model count is displayed', async ({ page }) => {
    await expect(page.getByText(/^\d+ models?$/)).toBeVisible();
  });

  test('image models launch the image playground', async ({ page }) => {
    await page.evaluate(() => {
      localStorage.setItem('op_api_key', 'op-models-image-key');
    });
    await page.reload();
    await page.locator('input[placeholder*="Search models"]').fill('RA1');
    await page.locator('text=RA1 Art Generator').waitFor();
    await page.locator('button:has-text("Generate")').first().click();

    await expect(page).toHaveURL(/\/playground\?model=ra1&mode=image/);
    await expect(page.locator('button:has-text("RA1 Art Generator")')).toBeVisible();
    await expect(page.getByTestId('image-size')).toBeVisible();
    await expect(page.locator('textarea[placeholder*="Describe the image"]')).toBeVisible();
    await expect(page.locator('button:has-text("Copy Code")')).toBeVisible();
  });
});
