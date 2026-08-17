import { expect, test } from '@playwright/test';

const seedanceModels = [
  'gemini-omni-flash-preview',
  'seedance-2.0-fast-text-to-video',
  'seedance-2.0-text-to-video',
  'seedance-2.0-image-to-video',
  'seedance-2.0-fast-reference-to-video',
  'seedance-2.0-reference-to-video',
  'alibaba/happy-horse/image-to-video',
  'ltx-2.3-image-to-video',
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
    await expect(nav.locator('a[href="/tools"]').last()).toBeVisible();
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
    await expect(page.locator('input[type="file"]').first()).toHaveAttribute('accept', 'image/*');
    await expect(page.getByText(/Preview your own 3D file/i)).toBeVisible();

    await expect(page.getByRole('button', { name: 'python' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'JS' })).toBeVisible();
    await expect(page.locator('button').filter({ hasText: /^curl$/ }).last()).toBeVisible();
    await expect(page.getByText('/v1/3d/generations')).toBeVisible();
    await expect(page.getByText('Model: pixal3d-image-to-3d')).toBeVisible();
  });

  test('tools hub lists first-party tools and links to each page', async ({ page }) => {
    await page.goto('/tools');

    await expect(page.getByRole('heading', { name: /OpenPaths Tools/i })).toBeVisible();
    await expect(page.locator('a[href="/text-to-image"]').first()).toBeVisible();
    await expect(page.locator('a[href="/image-to-3d"]').first()).toBeVisible();
    await expect(page.locator('a[href="/text-to-3d"]').first()).toBeVisible();

    await page.locator('a[href="/text-to-3d"]').first().click();
    await expect(page).toHaveURL('/text-to-3d');
    await expect(page.getByRole('heading', { name: /^Text to 3D$/i })).toBeVisible();
  });

  test('text-to-3d space exposes prompt controls, viewer, and snippets', async ({ page }) => {
    await page.goto('/text-to-3d');

    await expect(page.getByRole('heading', { name: /^Text to 3D$/i })).toBeVisible();
    await expect(page.locator('model-viewer')).toHaveAttribute('src', /sword-pixal3d\.glb/);
    await expect(page.getByText('/v1/3d/text-generations')).toBeVisible();
    await expect(page.getByRole('button', { name: /Generate 3D/i })).toBeVisible();
  });

  test('text-to-image space exposes prompt, model select, and snippets', async ({ page }) => {
    await page.goto('/text-to-image');

    await expect(page.getByRole('heading', { name: /^Text to Image$/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /^Generate/i })).toBeVisible();
    // Default snippet tab is Python (images.generate); switch to cURL to see the raw endpoint.
    await expect(page.getByText('images.generate')).toBeVisible();
    await page.getByRole('button', { name: /^curl$/ }).click();
    await expect(page.getByText('/v1/images/generations')).toBeVisible();
  });

  for (const modelId of seedanceModels) {
    test(`video model space renders demo output and API snippets for ${modelId}`, async ({ page }) => {
      await page.goto(`/models/${encodeURIComponent(modelId)}`);

      await expect(page.getByRole('button', { name: /Generate video/i })).toBeVisible();
      await expect(page.getByText('Video API example')).toBeVisible();
      await expect(page.locator('video')).toHaveCount(1);
      await expect(page.getByText('/videos/generations').first()).toBeVisible();
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

  test('PixVerse image-to-video page is a complete on-page workspace with a live request', async ({ page }) => {
    const modelId = 'fal-ai/pixverse/v5.6/image-to-video';
    await page.goto(`/models/${encodeURIComponent(modelId)}`);

    const panel = page.getByTestId('mp-video-panel');
    await expect(panel).toBeVisible();
    await expect(page.getByTestId('mp-video-output')).toHaveAttribute('src', /video-tips\/coast\.mp4/);
    await expect(page.getByTestId('mp-video-input-preview')).toHaveAttribute('src', /video-tips\/coast-poster\.webp/);
    await expect(page.getByTestId('mp-video-image-url')).toHaveValue(/video-tips\/coast-poster\.webp/);
    await expect(page.getByTestId('mp-video-prompt')).toHaveValue(/Slow cinematic aerial push-in/);
    await expect(panel.locator('pre')).toContainText(`"model": "${modelId}"`);
    await expect(panel.locator('pre')).toContainText('"image_url": "https://openpaths.io/static/blog/video-tips/coast-poster.webp"');

    await page.getByTestId('mp-video-prompt').fill('Locked-off shot; ocean waves move gently and clouds drift east.');
    await expect(panel.locator('pre')).toContainText('Locked-off shot; ocean waves move gently and clouds drift east.');
    await expect(page.locator(`a[href^="/playground?model=${encodeURIComponent(modelId)}"]`)).toHaveCount(0);

    await page.getByRole('button', { name: 'JavaScript' }).click();
    await expect(panel.locator('pre')).not.toContainText('cast_to');
    await page.getByRole('button', { name: 'Python' }).click();
    await expect(panel.locator('pre')).toContainText("body=json.loads(r'''");
  });

  test('model video workspace generates and replaces the preview without navigating away', async ({ page }) => {
    const modelId = 'fal-ai/pixverse/v5.6/image-to-video';
    await page.route('**/v1/videos/generations?async=true', route => route.fulfill({
      status: 202,
      json: { id: 'video-job-e2e', status: 'queued' },
    }));
    await page.route('**/v1/videos/generations/video-job-e2e', route => route.fulfill({
      json: {
        id: 'video-job-e2e',
        status: 'completed',
        result: { video_url: 'https://cdn.example.com/generated-pixverse.mp4' },
      },
    }));
    await page.goto('/');
    await page.evaluate(() => localStorage.setItem('op_api_key', 'op-model-page-e2e'));
    await page.goto(`/models/${encodeURIComponent(modelId)}`);

    await page.getByTestId('mp-video-generate').click();
    await expect(page.getByTestId('mp-video-output')).toHaveAttribute('src', 'https://cdn.example.com/generated-pixverse.mp4', { timeout: 10_000 });
    await expect(page).toHaveURL(`/models/${encodeURIComponent(modelId)}`);
    await expect(page.getByText('Generated output')).toBeVisible();
  });

  test('image model page has a live on-page workspace and replaces its starter output', async ({ page }) => {
    await page.route('**/v1/images/generations', route => route.fulfill({
      json: { data: [{ url: 'https://cdn.example.com/generated-ra1.webp' }] },
    }));
    await page.goto('/');
    await page.evaluate(() => localStorage.setItem('op_api_key', 'op-image-page-e2e'));
    await page.goto('/models/ra1');

    const panel = page.getByTestId('mp-image-panel');
    await expect(panel).toBeVisible();
    await expect(page.getByTestId('mp-image-prompt')).toHaveValue(/red fox wearing a tiny astronaut helmet/);
    await expect(panel.locator('pre')).toContainText('"model": "ra1"');
    await expect(page.getByRole('button', { name: /Chat in playground/i })).toHaveCount(0);

    await page.getByTestId('mp-image-prompt').fill('A cobalt teapot on a marble plinth, soft north light.');
    await expect(panel.locator('pre')).toContainText('A cobalt teapot on a marble plinth, soft north light.');
    await page.getByTestId('mp-image-generate').click();
    await expect(page.getByTestId('mp-image-output')).toHaveAttribute('src', 'https://cdn.example.com/generated-ra1.webp');
    await expect(page).toHaveURL('/models/ra1');
  });

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
    await expect(page.getByTestId('video-image-url')).toHaveValue('https://openpaths.io/logo-512.webp');
    await expect(page.getByTestId('preview-video-start-image').locator('img')).toBeVisible();
    const promptInput = page.locator('textarea[placeholder*="Describe the video"]');
    await expect(promptInput).toHaveValue(/Animate the supplied OpenPaths logo/);
    await expect(promptInput).not.toHaveCSS('height', '44px');

    await page.getByRole('button', { name: /Copy Code/i }).click();
    const generatedCode = page.getByTestId('playground-generated-code');
    await expect(generatedCode).toContainText('op-video-e2e-key');
    await expect(generatedCode).toContainText('/videos/generations');
    await expect(generatedCode).toContainText('"model": "seedance-2.0-image-to-video"');
    await expect(generatedCode).toContainText('"image_url": "https://openpaths.io/logo-512.webp"');

    await page.getByRole('button', { name: 'cURL' }).click();
    await expect(generatedCode).toContainText('Authorization: Bearer op-video-e2e-key');
  });

  test('server-injected session authenticates spaces without manual api key', async ({ page }) => {
    // Simulate the server injecting window.userData (op_session cookie) before page scripts run.
    await page.addInitScript(() => {
      (window as any).userData = {
        id: 'u_e2e',
        email: 'e2e@openpaths.io',
        name: 'E2E',
        secret: 'op-injected-session-key',
        authenticated: true,
      };
    });

    await page.goto('/text-to-image');
    await expect(page.getByRole('heading', { name: /^Text to Image$/i })).toBeVisible();

    // main.tsx must bridge window.userData.secret -> localStorage['op_api_key'].
    await expect.poll(() => page.evaluate(() => localStorage.getItem('op_api_key'))).toBe('op-injected-session-key');

    // Snippets must use the injected key, never 'op-...' placeholder or 'undefined'.
    await page.getByRole('button', { name: /^curl$/ }).click();
    await expect(page.getByText('Bearer op-injected-session-key')).toBeVisible();
    await expect(page.getByText('Bearer undefined')).toHaveCount(0);
    await expect(page.getByText('Bearer op-...')).toHaveCount(0);
  });

  test('model space video panel exposes open-weight args and reflects them in the snippet', async ({ page }) => {
    await page.goto('/models/ltx-2.3-image-to-video');
    const panel = page.getByTestId('mp-video-panel');
    await expect(panel).toBeVisible();
    await expect(page.getByTestId('mp-video-resolution')).toBeVisible();
    await expect(page.getByTestId('mp-video-duration')).toBeVisible();

    await page.getByTestId('mp-video-advanced-toggle').click();
    // LTX is open-weight: frame + diffusion controls are available.
    await expect(page.getByTestId('mp-video-seed')).toBeVisible();
    await expect(page.getByTestId('mp-video-negative-prompt')).toBeVisible();
    await expect(page.getByTestId('mp-video-num-frames')).toBeVisible();
    await expect(page.getByTestId('mp-video-fps')).toBeVisible();
    await expect(page.getByTestId('mp-video-guidance-scale')).toBeVisible();
    await expect(page.getByTestId('mp-video-num-inference-steps')).toBeVisible();

    await page.getByTestId('mp-video-seed').fill('12345');
    await page.getByTestId('mp-video-negative-prompt').fill('blurry, distorted');
    await page.getByTestId('mp-video-guidance-scale').fill('4.5');
    const code = panel.locator('pre');
    await expect(code).toContainText('"seed": 12345');
    await expect(code).toContainText('"negative_prompt": "blurry, distorted"');
    await expect(code).toContainText('"guidance_scale": 4.5');
  });

  test('model space video panel hides frame args for closed-source seedance but keeps seed/negative', async ({ page }) => {
    await page.goto('/models/seedance-2.0-text-to-video');
    await page.getByTestId('mp-video-advanced-toggle').click();
    await expect(page.getByTestId('mp-video-seed')).toBeVisible();
    await expect(page.getByTestId('mp-video-negative-prompt')).toBeVisible();
    await expect(page.getByTestId('mp-video-num-frames')).toHaveCount(0);
    await expect(page.getByTestId('mp-video-num-inference-steps')).toHaveCount(0);
  });

  test('FLUX 3 playground exposes native modes and keeps safety tolerance fixed at 4', async ({ page }) => {
    await page.goto('/playground?model=flux-3-video&mode=video');

    const mode = page.getByTestId('video-input-mode');
    await expect(mode).toBeVisible();
    await expect(mode.locator('option')).toHaveCount(3);
    await expect(page.getByTestId('video-safety-tolerance')).toContainText('4 · fixed');
    await expect(page.getByTestId('video-resolution').locator('option')).toHaveText(['HD', 'FHD']);
    await expect(page.getByTestId('video-duration').locator('option').first()).toHaveText('auto');
    await expect(page.getByTestId('video-aspect-ratio').locator('option', { hasText: '2:1' })).toHaveCount(1);

    await mode.selectOption('image-to-video');
    await page.getByTestId('video-image-url').fill('https://cdn.example.com/start.webp');
    await page.getByTestId('video-end-image-url').fill('https://cdn.example.com/end.webp');
    await page.getByRole('button', { name: /Copy Code/i }).click();
    const generatedCode = page.getByTestId('playground-generated-code');
    await expect(generatedCode).toContainText('"image_url": "https://cdn.example.com/start.webp"');
    await expect(generatedCode).toContainText('"end_image_url": "https://cdn.example.com/end.webp"');
    await expect(generatedCode).toContainText('"safety_tolerance": 4');

    await mode.selectOption('video-to-video');
    await page.getByTestId('video-video-urls').fill('https://cdn.example.com/source.mp4');
    await expect(generatedCode).toContainText('"video_url": "https://cdn.example.com/source.mp4"');
    await expect(generatedCode).not.toContainText('"image_url"');
  });

  test('FLUX 2 Pro playground exposes image controls and prompt upsampling', async ({ page }) => {
    await page.goto('/playground?model=flux-2-pro-preview&mode=image');

    await expect(page.getByTestId('image-size')).toBeVisible();
    await expect(page.getByTestId('image-size').locator('option', { hasText: '2048x2048' })).toHaveCount(1);
    await expect(page.getByTestId('image-output-format')).toHaveValue('webp');
    await expect(page.getByTestId('image-prompt-upsampling')).toContainText('on · richer prompt');
    await expect(page.getByTestId('image-safety-tolerance')).toContainText('5 · fixed');

    await page.getByRole('button', { name: /Copy Code/i }).click();
    const generatedCode = page.getByTestId('playground-generated-code');
    await expect(generatedCode).toContainText('"output_format": "webp"');
    await expect(generatedCode).toContainText('"safety_tolerance": 5');
    await expect(generatedCode).toContainText('"disable_pup": false');

    await page.getByTestId('image-prompt-upsampling').click();
    await expect(generatedCode).toContainText('"disable_pup": true');
  });

  test('video playground exposes advanced args and emits them in generated code', async ({ page }) => {
    await page.goto('/');
    await page.evaluate(() => localStorage.setItem('op_api_key', 'op-adv-e2e-key'));
    await page.goto('/playground?model=seedance-2.0-image-to-video&mode=video');

    await expect(page.getByTestId('video-resolution')).toBeVisible();
    await page.getByTestId('video-advanced-toggle').click();
    await expect(page.getByTestId('video-seed')).toBeVisible();
    await expect(page.getByTestId('video-negative-prompt')).toBeVisible();
    await page.getByTestId('video-seed').fill('98765');
    await page.getByTestId('video-negative-prompt').fill('low quality');

    await page.getByRole('button', { name: /Copy Code/i }).click();
    const generatedCode = page.getByTestId('playground-generated-code');
    await expect(generatedCode).toContainText('"seed": 98765');
    await expect(generatedCode).toContainText('"negative_prompt": "low quality"');
    await expect(generatedCode).toContainText('"model": "seedance-2.0-image-to-video"');
  });

  test('gallery renders indexed video generations as playable cards and detail', async ({ page }) => {
    const videoItem = {
      id: 'video-seedance-2-0-text-to-video',
      slug: 'video-seedance-2-0-text-to-video',
      title: 'Seedance 2.0 Text To Video',
      prompt: 'A polished studio macro shot of an AI infrastructure dashboard.',
      mediaType: 'video',
      videoUrl: 'https://openpathsstatic.openpaths.io/static/uploads/playground/seedance/seedance-text-to-video.mp4',
      posterUrl: '',
      durationSeconds: 4,
      aspect: 'wide',
      model: 'seedance-2.0-text-to-video',
      source: 'openpaths-gen',
      tags: ['video', 'video generation'],
    };
    await page.route('**/v1/art/list*', route =>
      route.fulfill({ json: { results: [videoItem], total: 1, aspects: { wide: 1 } } }),
    );
    await page.route('**/v1/art/tags*', route => route.fulfill({ json: [] }));
    await page.route('**/v1/art/item*', route =>
      route.fulfill({ json: { item: videoItem, related: [] } }),
    );

    await page.goto('/art');
    const card = page.getByTestId('art-card-video').first();
    await expect(card).toBeVisible();
    await expect(card).toHaveAttribute('src', /seedance-text-to-video\.mp4/);

    await page.goto('/art/i/video-seedance-2-0-text-to-video');
    const detail = page.getByTestId('art-detail-video');
    await expect(detail).toBeVisible();
    await expect(detail).toHaveAttribute('src', /seedance-text-to-video\.mp4/);
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
