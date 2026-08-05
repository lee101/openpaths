import { test, expect } from '@playwright/test';

test.describe('Providers Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/providers');
  });

  test('jump links scroll to a provider section', async ({ page }) => {
    await expect(page.getByText('Jump to provider')).toBeVisible();

    await page.locator('a[href="#openai"]').click();
    await expect(page).toHaveURL('/providers#openai');
    await expect(page.locator('#openai')).toBeVisible();
  });

  test('provider docs button opens the provider docs page', async ({ page }) => {
    const openaiCard = page.locator('#openai');
    await openaiCard.getByRole('link', { name: 'Docs' }).click();
    await expect(page).toHaveURL('/openai/docs');
    await expect(page.locator('h1')).toContainText('OpenAI');
  });

  test('Black Forest Labs has family spaces and a working video calculator', async ({ page }) => {
    await page.goto('/providers/black-forest-labs');

    await expect(page.locator('h1')).toHaveText('Black Forest Labs');
    await expect(page.locator('#flux-3-video')).toContainText('FLUX 3 Video');
    await expect(page.locator('#flux-2')).toContainText('FLUX.2');
    await expect(page.locator('#flux-tools')).toContainText('FLUX Tools');
    await expect(page.locator('#flux-1')).toContainText('FLUX.1');
    await expect(page.getByText('$0.30', { exact: true })).toBeVisible();

    await page.getByRole('button', { name: 'Full quality' }).click();
    await page.getByRole('button', { name: 'FHD' }).click();
    await expect(page.getByText('$1.45', { exact: true })).toBeVisible();
  });
});
