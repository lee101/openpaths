import { expect, test, type Page } from '@playwright/test';

const ART_ITEM = {
  id: 'art-e2e-1',
  slug: 'anime-eyes-e2e',
  title: 'Anime eyes',
  prompt: 'Close-up anime portrait with luminous violet eyes and intricate reflections',
  imageUrl: 'https://cdn.example.com/anime-eyes.webp',
  thumbUrl: 'https://cdn.example.com/anime-eyes-thumb.webp',
  width: 1024,
  height: 1024,
  aspect: 'square',
  model: 'zimage',
  tags: ['anime', 'portrait'],
};

async function mockArt(page: Page) {
  await page.route('**/v1/art/item*', route => route.fulfill({
    json: { item: ART_ITEM, related: [] },
  }));
}

async function signInLocally(page: Page) {
  await page.goto('/');
  await page.evaluate(() => localStorage.setItem('op_api_key', 'op-art-generation-e2e'));
}

test.describe('Art detail generation studio', () => {
  test('offers image and video creation on the artwork page and asks guests to sign in', async ({ page }) => {
    await mockArt(page);
    await page.goto('/art/i/anime-eyes-e2e');

    await expect(page.getByTestId('art-generation-studio')).toBeVisible();
    await expect(page.getByTestId('art-image-controls')).toBeVisible();
    await expect(page.getByTestId('art-generate-submit')).toHaveText(/Sign in to generate/i);

    await page.getByRole('button', { name: /Animate this image/i }).click();
    await expect(page.getByTestId('art-video-controls')).toBeVisible();
    await expect(page.getByText('First frame locked')).toBeVisible();

    await page.getByTestId('art-generate-submit').click();
    await expect(page.getByTestId('auth-modal')).toBeVisible();
  });

  test('generates an image variation inline with the selected price and options', async ({ page }) => {
    await mockArt(page);
    await page.route('**/account/balance', route => route.fulfill({ json: { balance_usd: 12.5, balance_cents: 125000 } }));
    let requestBody: Record<string, unknown> | null = null;
    await page.route('**/v1/images/generations', route => {
      requestBody = route.request().postDataJSON();
      return route.fulfill({ json: { data: [{ url: 'https://cdn.example.com/generated-variation.webp' }] } });
    });
    await signInLocally(page);
    await page.goto('/art/i/anime-eyes-e2e');

    await expect(page.getByTestId('art-generation-balance')).toContainText('$12.50');
    await expect(page.getByTestId('art-generate-submit')).toContainText('$0.007');
    await page.getByTestId('art-generate-submit').click();

    await expect(page.getByAltText('Generated variation 1')).toHaveAttribute('src', 'https://cdn.example.com/generated-variation.webp');
    expect(requestBody).toMatchObject({
      model: 'zimage',
      prompt: ART_ITEM.prompt,
      size: '1024x1024',
      n: 1,
    });
  });

  test('animates the artwork inline and polls the async video job', async ({ page }) => {
    await mockArt(page);
    await page.route('**/account/balance', route => route.fulfill({ json: { balance_usd: 12.5, balance_cents: 125000 } }));
    let requestBody: Record<string, unknown> | null = null;
    await page.route('**/v1/videos/generations', route => {
      if (route.request().method() !== 'POST') return route.continue();
      requestBody = route.request().postDataJSON();
      return route.fulfill({ status: 202, json: { id: 'art-video-job', status: 'queued' } });
    });
    await page.route('**/v1/videos/generations/art-video-job', route => route.fulfill({
      json: { id: 'art-video-job', status: 'completed', result: { video_url: 'https://cdn.example.com/animated-art.mp4' } },
    }));
    await signInLocally(page);
    await page.goto('/art/i/anime-eyes-e2e');

    await page.getByTestId('art-generate-video-tab').click();
    await expect(page.getByTestId('art-generate-submit')).toContainText('$0.40');
    await page.getByTestId('art-generate-submit').click();

    await expect(page.getByTestId('art-generated-video')).toHaveAttribute('src', 'https://cdn.example.com/animated-art.mp4', { timeout: 10_000 });
    expect(requestBody).toMatchObject({
      model: 'minimax-h3-max-image-to-video',
      image_url: ART_ITEM.imageUrl,
      resolution: '768p',
      duration: 5,
      aspect_ratio: 'auto',
      generate_audio: true,
      async: true,
    });
  });

  test('opens the credit purchase flow before calling generation for an empty balance', async ({ page }) => {
    await mockArt(page);
    await page.route('**/account/balance', route => route.fulfill({ json: { balance_usd: 0, balance_cents: 0 } }));
    await page.route('**/account/stripe/config', route => route.fulfill({ json: { publishable_key: 'pk_test_art' } }));
    let generationCalls = 0;
    await page.route('**/v1/images/generations', route => {
      generationCalls += 1;
      return route.fulfill({ json: { data: [] } });
    });
    await signInLocally(page);
    await page.goto('/art/i/anime-eyes-e2e');

    await expect(page.getByTestId('art-generate-submit')).toHaveText(/Add credits to generate/i);
    await page.getByTestId('art-generate-submit').click();
    await expect(page.getByTestId('stripe-modal')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Add funds' })).toBeVisible();
    expect(generationCalls).toBe(0);
  });
});
