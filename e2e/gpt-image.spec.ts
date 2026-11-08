import { expect, test } from '@playwright/test';

test('gpt-image tool: model tabs, examples and edit request', async ({ page }) => {
  await page.goto('/gpt-image');
  await expect(page.getByRole('heading', { name: 'GPT Image 2.6', exact: true })).toBeVisible();
  await expect(page.getByTestId('gpt-image-model-tab')).toHaveCount(4);
  await page.getByTestId('gpt-image-model-tab').nth(1).click();
  await page.getByTestId('mp-image-example').nth(3).click();
  await expect(page.getByTestId('mp-image-output')).toHaveAttribute('src', /gpt-image-2\.6-flare-perfume-edit\.webp/);
  await page.getByTestId('mp-image-api-key').fill('op-test');
  let input: any;
  await page.route('**/v1/images/edits', async route => {
    input = route.request().postDataJSON();
    await route.fulfill({ json: { data: [{ url: 'https://example.com/result.webp' }] } });
  });
  await page.getByTestId('mp-image-quality').selectOption('medium');
  await page.getByTestId('mp-image-generate').click();
  await expect.poll(() => input?.model).toBe('gpt-image-2.6-flare');
  expect(input.quality).toBe('medium');
  expect(input.image_url).toContain('perfume.jpg');
});

test('gpt-image model page shows examples and text-to-image works', async ({ page }) => {
  await page.goto('/models/gpt-image-2.5-sunburst');
  await expect(page.getByTestId('mp-image-example')).toHaveCount(4);
  await page.getByTestId('mp-image-api-key').fill('op-test');
  let input: any;
  await page.route('**/v1/images/generations', async route => {
    input = route.request().postDataJSON();
    await route.fulfill({ json: { data: [{ url: 'https://example.com/result.webp' }] } });
  });
  await page.getByTestId('mp-image-size').selectOption('1536x1024');
  await page.getByTestId('mp-image-generate').click();
  await expect.poll(() => input?.model).toBe('gpt-image-2.5-sunburst');
  expect(input.size).toBe('1536x1024');
  expect(input.image_url).toBeUndefined();
});
