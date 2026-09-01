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

const moreRelatedItem = {
  id: 'more-1',
  slug: 'city-neon-more-1',
  title: 'Neon City Portrait',
  prompt: 'Anime portrait in a neon city with glowing signs and cinematic light',
  imageUrl: 'https://openpathsstatic.openpaths.io/static/uploads/landing/art-playground/netwrck/zimage-lantern-koi-station.webp',
  thumbUrl: 'https://openpathsstatic.openpaths.io/static/uploads/landing/art-playground/netwrck/zimage-lantern-koi-station.webp',
  model: 'zimage',
  steps: 20,
  tags: ['city', 'neon', 'portrait'],
};

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
    // The DB-backed endpoints are unavailable in this fixture, so the page falls
    // back to the static manifest index (mocked above).
    await page.route('/v1/art/search**', route => {
      const query = new URL(route.request().url()).searchParams.get('q');
      if (query === 'city' || query === 'neon') {
        return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ results: [moreRelatedItem] }) });
      }
      return route.fulfill({ status: 503, contentType: 'application/json', body: '{"error":"not ready"}' });
    });
    await page.route('/v1/art/list**', route => route.fulfill({ status: 503, contentType: 'application/json', body: '{"error":"not ready"}' }));
    await page.route('/v1/art/tags**', route => route.fulfill({ status: 200, contentType: 'application/json', body: '{"tags":[]}' }));
    await page.route('/v1/art/item**', route => {
      const slug = new URL(route.request().url()).searchParams.get('slug');
      if (slug !== fixtureItems[0].slug) {
        return route.fulfill({ status: 404, contentType: 'application/json', body: '{"error":"not found"}' });
      }
      return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ item: fixtureItems[0], related: [fixtureItems[1]] }) });
    });
  });

  test('loads indexed art and links prompts into the image playground', async ({ page }) => {
    await page.goto('/art');

    await expect(page.getByRole('heading', { name: 'Search AI art by prompt' })).toBeVisible();
    await expect(page.getByRole('article').filter({ hasText: 'Anime illustration of a lantern train station' })).toBeVisible();

    await page.getByPlaceholder('Search prompts, style, scene, character, mood...').fill('koi lantern');
    await expect(page.getByRole('article').filter({ hasText: 'Anime illustration of a lantern train station' })).toBeVisible();

    await page.getByRole('link', { name: 'Try prompt' }).first().click();
    await expect(page).toHaveURL(/\/playground\?model=zimage&prompt=/);
  });

  test('loads more detail-page art from tags on related items', async ({ page }) => {
    await page.goto(`/art/i/${fixtureItems[0].slug}`);

    await expect(page.getByRole('heading', { name: 'Related art' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Load more related art' })).toBeVisible();

    await page.getByRole('button', { name: 'Load more related art' }).click();
    await expect(page.getByAltText(moreRelatedItem.prompt)).toBeVisible();
    await expect(page.getByRole('button', { name: 'Load more related art' })).toHaveCount(0);
  });
});
