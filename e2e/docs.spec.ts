import { test, expect } from '@playwright/test';

test.describe('Docs Page', () => {
  test('shows sign-in guidance when no API key is stored', async ({ page }) => {
    await page.goto('/docs');
    await expect(page.locator('h1')).toContainText('OpenAI And Anthropic SDK Compatible Docs');
    await expect(page.getByTestId('docs-base-url')).toContainText('/v1');
    await expect(page.locator('text=Sign in on the account page')).toBeVisible();
  });

  test('auto-populates the stored API key in examples', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('op_api_key', 'op-test-docs-key');
    });

    await page.goto('/docs');

    await expect(page.getByTestId('docs-api-key')).toContainText('op-test-docs-key');
    await expect(page.getByTestId('docs-curl')).toContainText('op-test-docs-key');
    await expect(page.getByTestId('docs-python')).toContainText('op-test-docs-key');
    await expect(page.getByTestId('docs-javascript')).toContainText('op-test-docs-key');
    await expect(page.getByTestId('docs-go')).toContainText('op-test-docs-key');
  });
});
