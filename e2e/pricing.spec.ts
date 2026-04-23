import { expect, test } from '@playwright/test';

test.describe('Pricing Page', () => {
  test('loads directly at /pricing', async ({ page }) => {
    await page.goto('/pricing');

    await expect(page).toHaveURL('/pricing');
    await expect(page).toHaveTitle(/OpenPaths Pricing/i);
    await expect(page.locator('h1')).toContainText('Pricing built to stay as close to 0 markup as possible');
    await expect(page.getByText('Text Generation and Reasoning', { exact: true })).toBeVisible();
    await expect(page.getByText('Image Generation', { exact: true }).first()).toBeVisible();
    await expect(page.getByText('Video Generation', { exact: true }).first()).toBeVisible();
  });
});
