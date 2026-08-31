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
    await expect(page.getByText('AI Image Gallery')).toBeVisible();
    await expect(page.getByRole('heading', { name: '100% OpenAI Compatible' })).toBeVisible();
  });

  test('renders the full image and video galleries without load-more controls', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'The Routing Engine' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Desert Cog Portal' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Glass Hummingbird' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Rain Tram — Native HD' })).toBeVisible();
    await expect(page.getByRole('button', { name: /load more/i })).toHaveCount(0);
  });

  test('homepage frontier roster and evals stay current', async ({ page }) => {
    await expect(page.getByText('Route from measurements, not model lore.')).toBeVisible();
    await expect(page.getByText('GPT-5.6 Sol', { exact: false }).first()).toBeVisible();
    await expect(page.getByText('Gemini 3.7 Flash', { exact: false }).first()).toBeVisible();
    await expect(page.getByText('GLM-5.3', { exact: false }).first()).toBeVisible();
    await expect(page.locator('main')).not.toContainText('GPT-5.5');
    await expect(page.locator('main')).not.toContainText('Gemini 2.5 Flash');
  });

  test('Black Forest Labs gallery videos and posters are available', async ({ page, request }) => {
    await expect(page.getByText('AI Video Gallery')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Routing Forest — Draft' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Living Routing Terrarium' })).toBeVisible();

    const draftVideo = await request.get('/static/video-gallery/bfl/flux-3-routing-forest-draft.webm');
    expect(draftVideo.ok()).toBeTruthy();
    expect(draftVideo.headers()['content-type']).toContain('video/webm');
    const fullPoster = await request.get('/static/video-gallery/bfl/flux-3-routing-terrarium-full-poster.webp');
    expect(fullPoster.ok()).toBeTruthy();
    expect(fullPoster.headers()['content-type']).toContain('image/webp');
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
    await expect(page.getByRole('heading', { name: 'OpenPaths Auto' })).toBeVisible();
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
