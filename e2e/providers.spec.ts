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
});
