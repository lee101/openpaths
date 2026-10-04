import { expect, test } from '@playwright/test';

test('FLUX 3 controls send native editing inputs and resolution', async ({ page }) => {
  await page.goto('/models/flux-3-image');
  await expect(page.getByRole('heading', {name:'FLUX 3 Image',exact:true})).toBeVisible();
  await page.getByTestId('mp-image-references').fill('https://example.com/a.jpg\nhttps://example.com/b.jpg');
  await page.getByTestId('mp-image-resolution').selectOption('4k');
  await page.getByTestId('mp-image-safety').selectOption('0');
  await page.getByTestId('mp-image-grounding').click();
  await page.getByTestId('mp-image-api-key').fill('op-test');
  let input:any;
  await page.route('**/v1/images/edits', async route => {
    input=route.request().postDataJSON();
    await route.fulfill({json:{data:[{url:'https://example.com/result.webp'}]}});
  });
  await page.getByTestId('mp-image-generate').click();
  await expect.poll(()=>input?.resolution).toBe('4k');
  expect(input.images).toEqual(['https://example.com/a.jpg','https://example.com/b.jpg']);
  expect(input.grounding).toBe(false);
  expect(input.safety_tolerance).toBe(0);
  expect(input.size).toBeUndefined();
  await expect(page.locator('pre').last()).toContainText('"resolution": "4k"');
});

test('homepage features FLUX 3 Image and H3 Max Recast',async ({page})=>{
  await page.goto('/');
  await expect(page.locator('a[href^="/models/flux-3-image"]').first()).toBeAttached();
  await expect(page.locator('a[href^="/models/minimax-h3-max-recast"]').first()).toBeAttached();
  await expect(page.locator('video[src$="cat-recast.webm"]').first()).toBeAttached();
});
