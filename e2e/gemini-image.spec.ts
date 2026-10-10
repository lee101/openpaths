import { expect, test } from '@playwright/test';

test('Nano Banana 2 Lite sends edit with image_url and aspect ratio', async ({ page }) => {
  await page.goto('/models/gemini-3.1-flash-lite-image');
  await expect(page.getByRole('heading', { name: 'Nano Banana 2 Lite', exact: true })).toBeVisible();
  await page.getByTestId('mp-image-aspect-ratio').selectOption('9:16');
  await page.getByTestId('mp-image-api-key').fill('op-test');
  let input: any;
  await page.route('**/v1/images/edits', async route => {
    input = route.request().postDataJSON();
    await route.fulfill({ json: { data: [{ url: 'https://example.com/result.webp' }] } });
  });
  await page.getByTestId('mp-image-generate').click();
  await expect.poll(() => input?.model).toBe('gemini-3.1-flash-lite-image');
  expect(input.aspect_ratio).toBe('9:16');
  expect(input.image_url).toContain('fort-source.webp');
  expect(input.size).toBeUndefined();
});

test('Nano Banana 2.1 text-to-image works without an input image', async ({ page }) => {
  await page.goto('/models/gemini-nano-banana-2.1');
  await page.getByTestId('mp-image-api-key').fill('op-test');
  await page.getByTestId('mp-image-input-url').fill('');
  let input: any;
  await page.route('**/v1/images/generations', async route => {
    input = route.request().postDataJSON();
    await route.fulfill({ json: { data: [{ url: 'https://example.com/result.webp' }] } });
  });
  await page.getByTestId('mp-image-generate').click();
  await expect.poll(() => input?.model).toBe('gemini-nano-banana-2.1');
  expect(input.image_url).toBeUndefined();
});

test('blog post renders', async ({ page }) => {
  await page.goto('/blog/gemini-image-models-game-dev-concept-art');
  await expect(page.getByRole('heading', { name: /Nano Banana 2 Lite and 2.1/ }).first()).toBeVisible();
  await expect(page.locator('img[src$="fort-winter.webp"]')).toBeVisible();
});
