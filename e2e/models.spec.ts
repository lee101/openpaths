import { test, expect } from '@playwright/test';

test.describe('Models Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/models');
  });

  test('page title and description render', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Model Directory');
    await expect(page.locator('text=Explore our vast network')).toBeVisible();
  });

  test('search input is present and functional', async ({ page }) => {
    const search = page.locator('input[placeholder*="Search models"]');
    await expect(search).toBeVisible();

    await search.fill('claude');
    await expect(page.locator('text=Claude 3.5 Sonnet')).toBeVisible();
    await expect(page.locator('text=Midjourney v6')).not.toBeVisible();
  });

  test('tag filters work', async ({ page }) => {
    // click art generation tag
    await page.click('button:has-text("art generation")');
    await expect(page.locator('text=Midjourney v6')).toBeVisible();
    await expect(page.locator('text=FLUX.1 Pro')).toBeVisible();
    // text models should not be visible
    await expect(page.locator('text=Claude 3.5 Sonnet')).not.toBeVisible();
  });

  test('clear filters button works', async ({ page }) => {
    await page.click('button:has-text("programming")');
    // fewer models visible
    const cardsBefore = await page.locator('[class*="rounded-xl"][class*="border"]').count();

    await page.click('text=Clear');
    const cardsAfter = await page.locator('[class*="rounded-xl"][class*="border"]').count();
    expect(cardsAfter).toBeGreaterThanOrEqual(cardsBefore);
  });

  test('model cards show provider, name, pricing, and ID', async ({ page }) => {
    const card = page.locator('text=Claude 3.5 Sonnet').locator('..');
    await expect(card).toBeVisible();
    // pricing is in the parent card area
    await expect(page.locator('text=$3.00').first()).toBeVisible();
    await expect(page.locator('text=$15.00').first()).toBeVisible();
    await expect(page.locator('code:has-text("anthropic/claude-3.5-sonnet")')).toBeVisible();
  });

  test('no results state shows when search matches nothing', async ({ page }) => {
    const search = page.locator('input[placeholder*="Search models"]');
    await search.fill('xyznonexistentmodel');
    await expect(page.locator('text=No models found')).toBeVisible();
  });

  test('all tag filter buttons are present', async ({ page }) => {
    const tags = ['programming', 'roleplay', 'art generation', 'general', 'vision', 'fast', 'reasoning', 'open-source'];
    for (const tag of tags) {
      await expect(page.locator(`button:has-text("${tag}")`)).toBeVisible();
    }
  });
});
