import { test, expect } from '@playwright/test';

const fixtureItems = [
  {
    id: 'koi-1',
    slug: 'lantern-koi-station-koi-1',
    title: 'Lantern Koi Station',
    prompt: 'Anime illustration of a lantern train station above a koi pond at blue hour',
    imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/landing/art-playground/netwrck/zimage-lantern-koi-station.webp',
    thumbUrl: 'https://openpathsstatic.openpaths.io/static/uploads/landing/art-playground/netwrck/zimage-lantern-koi-station.webp',
    model: 'zimage',
    steps: 20,
    tags: ['anime', 'lantern', 'koi'],
  },
  {
    id: 'city-1',
    slug: 'neon-city-city-1',
    title: 'Neon City',
    prompt: 'A rainy neon city street with reflective signs and cinematic anime lighting',
    imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/landing/art-playground/netwrck/zimage-lantern-koi-station.webp',
    model: 'zimage',
    steps: 20,
    tags: ['city', 'neon'],
  },
];

test.describe('ZImage Art Search', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('https://openpathsstatic.openpaths.io/static/data/zimage-art/manifest.json', route => route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({
        version: 1,
        kind: 'zimage-art-index',
        model: 'zimage',
        count: fixtureItems.length,
        generatedCount: fixtureItems.length,
        publicBase: 'https://openpathsstatic.openpaths.io/static/data/zimage-art',
        chunks: [{ path: 'chunks/chunk-0000.json', count: fixtureItems.length }],
      }),
    }));
    await page.route('https://openpathsstatic.openpaths.io/static/data/zimage-art/chunks/chunk-0000.json', route => route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify(fixtureItems),
    }));
    await page.route('/art/search**', route => route.fulfill({ status: 503, contentType: 'application/json', body: '{"error":"not ready"}' }));
  });

  test('loads indexed art and links prompts into the image playground', async ({ page }) => {
    await page.goto('/art');

    await expect(page.getByRole('heading', { name: 'Search generated art by prompt' })).toBeVisible();
    await expect(page.getByText('2 indexed prompts')).toBeVisible();

    await page.getByPlaceholder('Search prompts, style, scene, character, mood...').fill('koi lantern');
    await expect(page.getByRole('article').filter({ hasText: 'Anime illustration of a lantern train station' })).toBeVisible();

    await page.getByRole('link', { name: 'Try this prompt' }).click();
    await expect(page).toHaveURL(/\/playground\?model=zimage&prompt=/);
  });
});
