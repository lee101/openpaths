import { expect, test } from '@playwright/test';

test('Recast Space defaults and request stay in sync', async ({ page }) => {
  await page.goto('/models/minimax-h3-max-recast');
  await expect(page.getByRole('heading', { name: 'H3 Max Recast', exact: true })).toBeVisible();
  await expect(page.getByTestId('mp-video-output')).toHaveAttribute('src', /cat-recast.webm$/);
  await expect(page.getByTestId('mp-video-source-preview')).toHaveAttribute('src', /dance-source.webm$/);
  await expect(page.getByTestId('mp-video-reference-photos')).toHaveValue(/cat-reference.webp$/);
  await expect(page.getByTestId('mp-video-recast-pricing')).toContainText('$0.30');
  await expect(page.getByTestId('mp-video-duration')).toHaveCount(0);
  await page.getByTestId('mp-video-resolution').selectOption('768P');
  await page.getByTestId('mp-video-reference-photos').fill('https://example.com/a.png\nhttps://example.com/b.png');
  await expect(page.locator('pre').last()).toContainText('reference_image_urls');
  await expect(page.locator('pre').last()).toContainText('768P');
  await expect(page.locator('pre').last()).not.toContainText('generate_audio');
  await page.getByTestId('mp-video-api-key').fill('op-test');
  let input: any;
  await page.route('**/v1/videos/generations?async=true', async route => {
    input = route.request().postDataJSON();
    await route.fulfill({ json: { video_url: 'https://example.com/recast.mp4' } });
  });
  await page.getByTestId('mp-video-generate').click();
  await expect(page.getByTestId('mp-video-output')).toHaveAttribute('src', 'https://example.com/recast.mp4');
  expect(input.reference_image_urls).toEqual(['https://example.com/a.png','https://example.com/b.png']);
  expect(input.resolution).toBe('768P');
  expect(input.video_url).toContain('dance-source.webm');
  expect(input.prompt).toBeUndefined();
  expect(input.duration).toBeUndefined();
});
