import { test, expect } from '@playwright/test';

test.describe('Landing Page', () => {
  let pageErrors: Error[] = [];

  test.beforeEach(async ({ page }) => {
    pageErrors = [];
    page.on('pageerror', error => {
      pageErrors.push(error);
    });
    await page.goto('/');
  });

  test.afterEach(async () => {
    expect(pageErrors, pageErrors.map(error => error.message).join('\n')).toHaveLength(0);
  });

  test('hero section renders with title and CTA', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('The Open Source');
    await expect(page.locator('h1')).toContainText('Model Router');
    await expect(page.locator('text=Explore Models')).toBeVisible();
    await expect(page.locator('text=View Source')).toBeVisible();
  });

  test('art playground and api sections render', async ({ page }) => {
    await expect(page.getByText('Live Art Playground')).toBeVisible();
    await expect(page.getByRole('heading', { name: '100% OpenAI Compatible' })).toBeVisible();
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

    // openai python tab active by default
    await expect(codeSection.locator('text=import')).toBeVisible();
    await expect(codeSection.locator('text=openai.OpenAI')).toBeVisible();

    // switch to cURL
    await codeSection.locator('button:has-text("cURL")').click();
    await expect(codeSection.locator('pre >> text=curl')).toBeVisible();

    // switch back to Python
    await codeSection.locator('button:has-text("Python")').click();
    await expect(codeSection.locator('text=openai.OpenAI')).toBeVisible();
  });

  test('feature cards render', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Millisecond Routing' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Universal API' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Auto Models' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Solana Payments' })).toBeVisible();
  });

  test('feature cards navigate to related pages', async ({ page }) => {
    await page.getByRole('link', { name: /Millisecond Routing/ }).click();
    await expect(page).toHaveURL('/models');

    await page.goto('/');
    await page.getByRole('link', { name: /Solana Payments/ }).click();
    await expect(page).toHaveURL('/pricing');
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
