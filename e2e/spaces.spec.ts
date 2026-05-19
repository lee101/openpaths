import { expect, test } from '@playwright/test';

const seedanceModels = [
  'seedance-2.0-fast-text-to-video',
  'seedance-2.0-text-to-video',
  'seedance-2.0-image-to-video',
  'seedance-2.0-fast-reference-to-video',
  'seedance-2.0-reference-to-video',
  'alibaba/happy-horse/image-to-video',
];

test.describe('Public Spaces', () => {
  test('mobile header exposes hidden navigation through the burger menu', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/');

    const nav = page.locator('nav');
    await expect(nav.locator('a[href="/models"]').first()).toBeHidden();
    await page.getByRole('button', { name: 'Open menu' }).click();

    await expect(nav.locator('a[href="/models"]').last()).toBeVisible();
    await expect(nav.locator('a[href="/alternatives"]').last()).toBeVisible();
    await expect(nav.locator('a[href="/image-to-3d"]').last()).toBeVisible();
    await expect(nav.locator('a[href="/#api"]').last()).toBeVisible();

    await nav.locator('a[href="/alternatives"]').last().click();
    await expect(page).toHaveURL('/alternatives');
    await expect(page.getByRole('heading', { name: /Compare OpenPaths with other AI gateways/i })).toBeVisible();
  });

  test('alternatives space lists comparison guides and opens alternative posts', async ({ page }) => {
    await page.goto('/alternatives');

    await expect(page.getByRole('heading', { name: /Compare OpenPaths with other AI gateways/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /OpenRouter Alternative/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /Together AI Alternative/i })).toBeVisible();

    await page.getByRole('link', { name: /OpenRouter Alternative/i }).click();
    await expect(page).toHaveURL('/alternatives/openrouter');
    await expect(page.getByRole('heading', { name: /OpenRouter Alternative/i })).toBeVisible();
    await expect(page.getByRole('link', { name: /All Alternatives/i })).toBeVisible();
  });

  test('image-to-3d space exposes default asset, upload controls, and copyable snippets', async ({ page }) => {
    await page.goto('/image-to-3d');

    await expect(page.getByRole('heading', { name: /Image to 3D/i })).toBeVisible();
    await expect(page.locator('model-viewer')).toHaveAttribute('src', /sword-pixal3d\.glb/);
    await expect(page.getByText('Upload, paste, or drop reference image')).toBeVisible();
    await expect(page.locator('input[type="file"]')).toHaveAttribute('accept', 'image/*');

    await expect(page.getByRole('button', { name: 'python' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'JS' })).toBeVisible();
    await expect(page.locator('button').filter({ hasText: /^curl$/ }).last()).toBeVisible();
    await expect(page.getByText('/v1/3d/generations')).toBeVisible();
    await expect(page.getByText('Model: pixal3d-image-to-3d')).toBeVisible();
  });

  for (const modelId of seedanceModels) {
    test(`video model space renders demo output and API snippets for ${modelId}`, async ({ page }) => {
      await page.goto(`/models/${encodeURIComponent(modelId)}`);

      await expect(page.getByRole('button', { name: /Generate video/i })).toBeVisible();
      await expect(page.getByText('Video API example')).toBeVisible();
      await expect(page.locator('video')).toHaveCount(1);
      await expect(page.getByText('/videos/generations')).toBeVisible();
      await expect(page.getByText(`"model": "${modelId}"`)).toBeVisible();
      if (modelId === 'alibaba/happy-horse/image-to-video') {
        await expect(page.getByText('"enable_safety_checker": true')).toBeVisible();
        await expect(page.getByText('"image_url": "https://openpathsstatic.openpaths.io/static/uploads/playground/happy-horse/rap.png"')).toBeVisible();
      }

      await page.getByRole('button', { name: 'JavaScript' }).click();
      await expect(page.getByText('client.post("/videos/generations"')).toBeVisible();

      await page.getByRole('button', { name: 'cURL' }).click();
      await expect(page.getByText('Authorization: Bearer op-...')).toBeVisible();
      await expect(page.getByText(`"model": "${modelId}"`)).toBeVisible();
    });
  }

  test('outpaint model space renders before-after demo and image API snippets', async ({ page }) => {
    await page.goto(`/models/${encodeURIComponent('fal-ai/flux-2-pro/outpaint')}`);

    await expect(page.getByRole('heading', { name: /FLUX 2 Pro Outpaint/i })).toBeVisible();
    await expect(page.getByText('Image API example')).toBeVisible();
    await expect(page.getByAltText(/FLUX 2 Pro Outpaint input/i)).toHaveAttribute('src', /flux-outpaint\/input\.png/);
    await expect(page.getByAltText(/FLUX 2 Pro Outpaint output/i)).toHaveAttribute('src', /flux-outpaint\/output\.jpg/);
    await expect(page.getByText('/images/generations')).toBeVisible();
    await expect(page.getByText('"model": "fal-ai/flux-2-pro/outpaint"')).toBeVisible();
    await expect(page.getByText('"expand_bottom": 200')).toBeVisible();

    await page.getByRole('button', { name: 'JavaScript' }).click();
    await expect(page.getByText('client.post("/images/generations"')).toBeVisible();
  });

  test('hidream edit model space renders reference edit demo and image API snippets', async ({ page }) => {
    await page.goto(`/models/${encodeURIComponent('fal-ai/hidream-o1-image/edit')}`);

    await expect(page.getByRole('heading', { name: /HiDream O1 Image Edit/i })).toBeVisible();
    await expect(page.getByText('Image API example')).toBeVisible();
    await expect(page.getByAltText(/HiDream O1 Image Edit input/i)).toHaveAttribute('src', /hidream-edit\/perfume\.jpg/);
    await expect(page.getByAltText(/HiDream O1 Image Edit output/i)).toHaveAttribute('src', /hidream-edit\/lipstick\.png/);
    await expect(page.getByText('/images/generations')).toBeVisible();
    await expect(page.getByText('"model": "fal-ai/hidream-o1-image/edit"')).toBeVisible();
    await expect(page.getByText('"image_size": "landscape_16_9"')).toBeVisible();
    await expect(page.getByText('"enable_safety_checker": false')).toBeVisible();
  });

  test('models can be searched by media task tags', async ({ page }) => {
    await page.goto('/models?q=image%20to%203d');
    await expect(page.getByRole('link', { name: /Pixal3D Image to 3D/i })).toBeVisible();

    await page.goto('/models?q=image%20to%20image');
    await expect(page.getByRole('link', { name: /FLUX 2 Pro Outpaint/i })).toBeVisible();
  });

  test('video playground opens from model query params and generates copyable video code', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.setItem('op_api_key', 'op-video-e2e-key');
    });
    await page.goto('/playground?model=seedance-2.0-image-to-video&mode=video');

    await expect(page.getByRole('button', { name: /Seedance 2.0 Image to Video/i })).toBeVisible();
    await expect(page.getByTestId('video-resolution')).toBeVisible();
    await expect(page.getByTestId('video-duration')).toHaveValue('4');
    await expect(page.getByTestId('video-start-image-dropzone')).toBeVisible();
    await expect(page.getByTestId('video-end-image-dropzone')).toBeVisible();
    await expect(page.locator('input[type="file"][accept="image/*"]').first()).toBeVisible();
    await expect(page.getByTestId('video-image-url')).toHaveValue('https://openpathsstatic.openpaths.io/static/uploads/playground/seedance/openpaths-logo.webp');
    await expect(page.getByTestId('preview-video-start-image').locator('img')).toBeVisible();
    const promptInput = page.locator('textarea[placeholder*="Describe the video"]');
    await expect(promptInput).toHaveValue(/Animate the supplied OpenPaths logo/);
    await expect(promptInput).not.toHaveCSS('height', '44px');

    await page.getByRole('button', { name: /Copy Code/i }).click();
    const generatedCode = page.getByTestId('playground-generated-code');
    await expect(generatedCode).toContainText('op-video-e2e-key');
    await expect(generatedCode).toContainText('/videos/generations');
    await expect(generatedCode).toContainText('"model": "seedance-2.0-image-to-video"');
    await expect(generatedCode).toContainText('"image_url": "https://openpathsstatic.openpaths.io/static/uploads/playground/seedance/openpaths-logo.webp"');

    await page.getByRole('button', { name: 'cURL' }).click();
    await expect(generatedCode).toContainText('Authorization: Bearer op-video-e2e-key');
  });

  test('outpaint playground opens with default image and expansion controls', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.setItem('op_api_key', 'op-image-e2e-key');
    });
    await page.goto(`/playground?model=${encodeURIComponent('fal-ai/flux-2-pro/outpaint')}&mode=image`);

    await expect(page.getByRole('button', { name: /FLUX 2 Pro Outpaint/i })).toBeVisible();
    await expect(page.getByTestId('image-input-urls')).toHaveValue(/flux-outpaint\/input\.png/);
    await expect(page.getByText('Bottom')).toBeVisible();
    await expect(page.getByText('Left')).toBeVisible();
    await expect(page.getByText('Right')).toBeVisible();

    await page.getByRole('button', { name: /Copy Code/i }).click();
    const generatedCode = page.getByTestId('playground-generated-code');
    await expect(generatedCode).toContainText('/images/generations');
    await expect(generatedCode).toContainText('"model": "fal-ai/flux-2-pro/outpaint"');
    await expect(generatedCode).toContainText('"image_url": "https://openpathsstatic.openpaths.io/static/uploads/playground/flux-outpaint/input.png"');
    await expect(generatedCode).toContainText('"expand_bottom": 200');
  });

  test('hidream edit playground opens with reference image and edit settings', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => {
      localStorage.setItem('op_api_key', 'op-image-e2e-key');
    });
    await page.goto(`/playground?model=${encodeURIComponent('fal-ai/hidream-o1-image/edit')}&mode=image`);

    await expect(page.getByRole('button', { name: /HiDream O1 Image Edit/i })).toBeVisible();
    await expect(page.getByTestId('image-input-urls')).toHaveValue(/hidream-edit\/perfume\.jpg/);
    await expect(page.getByTestId('image-input-urls').locator('..').getByRole('img')).toBeVisible();
    await expect(page.getByTestId('chat-input')).toHaveValue(/Replace the perfume bottle with a lipstick/);

    await page.getByRole('button', { name: /Copy Code/i }).click();
    const generatedCode = page.getByTestId('playground-generated-code');
    await expect(generatedCode).toContainText('/images/edits');
    await expect(generatedCode).toContainText('"model": "fal-ai/hidream-o1-image/edit"');
    await expect(generatedCode).toContainText('"reference_image_urls"');
    await expect(generatedCode).toContainText('"image_size": "landscape_16_9"');
    await expect(generatedCode).toContainText('"enable_safety_checker": false');
  });
});
