import { expect, test } from '@playwright/test';

test('FLUX 3 Max space covers text, image and keyframes with a live price and request', async ({ page }) => {
  await page.goto('/models/flux-3-max-video');
  await expect(page.getByRole('heading', { name: 'FLUX 3 Max Video', exact: true })).toBeVisible();
  await expect(page.getByTestId('mp-video-output')).toHaveAttribute('src', /flux-3-max\/text-output\.mp4$/);
  await expect(page.getByTestId('flux3-price')).toContainText('$0.85');

  await page.getByTestId('mp-video-resolution').selectOption('1080p');
  await expect(page.getByTestId('flux3-price')).toContainText('$1.45');
  await page.getByTestId('mp-video-duration').fill('10');
  await expect(page.getByTestId('flux3-price')).toContainText('$2.90');
  await page.getByTestId('flux3-reasoning').selectOption('low');
  await page.getByTestId('mp-video-generate-audio').click();
  await page.getByTestId('mp-video-seed').fill('77');
  const snippet = page.locator('pre').last();
  await expect(snippet).toContainText('"model": "flux-3-max-video"');
  await expect(snippet).toContainText('"reasoning_level": "low"');
  await expect(snippet).toContainText('"generate_audio": false');
  await expect(snippet).toContainText('"seed": 77');
  await expect(snippet).not.toContainText('image_url');

  await page.getByTestId('flux3-example-image').click();
  await page.getByTestId('flux3-mode-image').click();
  await expect(page.getByTestId('flux3-start-url')).toHaveValue(/flux-3-max\/image-input\.jpg$/);
  await expect(page.getByTestId('mp-video-output')).toHaveAttribute('src', /flux-3-max\/image-output\.mp4$/);
  await expect(snippet).toContainText('"image_url"');

  await page.getByTestId('flux3-example-keyframes').click();
  await expect(page.getByTestId('flux3-kf0-url')).toHaveValue(/keyframes-input-1\.jpg$/);
  await expect(page.getByTestId('flux3-kf1-index')).toHaveValue('120');
  await page.getByTestId('flux3-add-keyframe').click();
  await expect(page.getByTestId('flux3-keyframe-2')).toBeVisible();
  await expect(snippet).toContainText('"frame_index": 120');
  await expect(snippet).not.toContainText('"image_url": "https://openpathsstatic.openpaths.io/static/video-gallery/flux-3-max/image-input.jpg"');

  await page.getByTestId('mp-video-api-key').fill('op-test');
  await page.getByTestId('flux3-kf2-url').fill('https://example.com/mid.jpg');
  let input: any;
  await page.route('**/v1/videos/generations?async=true', async route => {
    input = route.request().postDataJSON();
    await route.fulfill({ json: { video_url: 'https://example.com/flux3.mp4' } });
  });
  await page.getByTestId('mp-video-generate').click();
  await expect(page.getByTestId('mp-video-output')).toHaveAttribute('src', 'https://example.com/flux3.mp4');
  expect(input.model).toBe('flux-3-max-video');
  expect(input.keyframes).toHaveLength(3);
  expect(input.keyframes[1]).toEqual({ image_url: expect.stringContaining('keyframes-input-2.jpg'), frame_index: 120 });
  expect(input.image_url).toBeUndefined();
  expect(input.duration).toBe(5);
});
