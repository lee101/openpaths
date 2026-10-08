import { expect, test } from '@playwright/test';

test('Relight Space presets, pricing and request stay in sync', async ({ page }) => {
  await page.goto('/models/minimax-h3-max-relight');
  await expect(page.getByRole('heading', { name: 'H3 Max Relight', exact: true })).toBeVisible();
  await expect(page.getByTestId('mp-relight-source')).toHaveAttribute('src', /relight\/source.mp4$/);
  await expect(page.getByTestId('mp-relight-pricing')).toContainText('$0.055');
  await page.getByTestId('mp-relight-resolution').selectOption('1080P');
  await expect(page.getByTestId('mp-relight-pricing')).toContainText('$0.176');
  await page.getByTestId('mp-relight-resolution').selectOption('480P');
  await page.getByTestId('relight-preset-neon').click();
  await page.getByTestId('relight-preset-custom').click();
  await expect(page.getByTestId('mp-relight-custom')).toBeVisible();
  await expect(page.locator('pre').last()).toContainText('reference_image_url');
  await expect(page.locator('pre').last()).not.toContainText('"prompt"');
  await page.getByTestId('mp-relight-api-key').fill('op-test');
  let input: any;
  await page.route('**/v1/videos/generations?async=true', async route => {
    input = route.request().postDataJSON();
    await route.fulfill({ json: { video_url: 'https://example.com/relit.mp4' } });
  });
  await page.getByTestId('mp-relight-generate').click();
  await expect(page.getByTestId('mp-relight-output')).toHaveAttribute('src', 'https://example.com/relit.mp4');
  expect(input.reference_image_url).toMatch(/^data:image\/png;base64,/);
  expect(input.resolution).toBe('480P');
  expect(input.video_url).toContain('relight/source.mp4');
  expect(input.prompt).toBeUndefined();
  expect(input.duration).toBeUndefined();
});
