import { test, expect } from '@playwright/test';

test.describe('Landing Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('hero section renders with title and CTA', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('The Open Source');
    await expect(page.locator('h1')).toContainText('Model Router');
    await expect(page.locator('text=Explore Models')).toBeVisible();
    await expect(page.locator('text=View Source')).toBeVisible();
  });

  test('version badge is visible', async ({ page }) => {
    await expect(page.locator('text=v1.1.0 is now live')).toBeVisible();
  });

  test('stats bar shows metrics', async ({ page }) => {
    await expect(page.locator('text=99.99% Uptime')).toBeVisible();
    await expect(page.locator('text=100+ Models')).toBeVisible();
    await expect(page.locator('text=<50ms Latency')).toBeVisible();
    await expect(page.locator('text=Intelligent Fallbacks')).toBeVisible();
  });

  test('code snippet tabs toggle between python and curl', async ({ page }) => {
    const codeSection = page.locator('#api');
    await expect(codeSection.locator('text=100% OpenAI Compatible')).toBeVisible();

    // python tab active by default
    await expect(codeSection.locator('text=import')).toBeVisible();

    // switch to curl
    await codeSection.locator('button:has-text("cURL")').click();
    await expect(codeSection.locator('pre >> text=curl')).toBeVisible();

    // switch back
    await codeSection.locator('button:has-text("Python")').click();
    await expect(codeSection.locator('text=import')).toBeVisible();
  });

  test('feature cards render', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Millisecond Routing' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Universal API' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Stripe Integration' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Solana Payments' })).toBeVisible();
  });

  test('CTA section at bottom', async ({ page }) => {
    await expect(page.locator('text=Ready to find your path?')).toBeVisible();
    await expect(page.locator('text=Create Free Account')).toBeVisible();
  });

  test('Explore Models link navigates to /models', async ({ page }) => {
    await page.click('text=Explore Models');
    await expect(page).toHaveURL('/models');
  });

  test('Create Free Account navigates to /account', async ({ page }) => {
    await page.click('text=Create Free Account');
    await expect(page).toHaveURL('/account');
  });
});
