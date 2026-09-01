import { expect, test } from '@playwright/test';

test('generates a Lyria Pro request with Opus output', async ({ page }) => {
  let payload: Record<string, unknown> | undefined;
  await page.addInitScript(() => localStorage.setItem('op_api_key', 'op-test'));
  await page.route('**/v1/music/generations', async route => {
    payload = route.request().postDataJSON();
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        data: {
          status: 2,
          audio: 'T2dnUw==',
          format: 'opus',
          mime_type: 'audio/ogg;codecs=opus',
        },
        extra_info: { music_size: 524288 },
      }),
    });
  });

  await page.goto('/tools/lyria');
  await expect(page.getByRole('heading', { name: 'From direction to record.' })).toBeVisible();
  await page.getByRole('button', { name: /Generate track/ }).click();

  await expect.poll(() => payload).toBeTruthy();
  expect(payload?.model).toBe('lyria-3-pro-preview');
  expect(payload?.output_format).toBe('opus');
  expect(payload?.prompt).toContain('Instrumental only');
  expect(payload?.prompt).toContain('## Structure');
  await expect(page.getByText('Ready', { exact: true })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Download .opus' })).toBeVisible();
});

test('supports Clip, custom lyrics, format selection, and Go code', async ({ page }) => {
  await page.goto('/tools/lyria');
  await page.getByRole('button', { name: /Lyria 3 Clip/ }).click();
  await expect(page.getByText('30s · opus')).toBeVisible();

  await page.getByRole('button', { name: 'My lyrics' }).click();
  await expect(page.getByLabel(/Lyrics/)).toBeVisible();
  await page.getByLabel('File format').selectOption('mp3');

  await page.getByRole('button', { name: 'go', exact: true }).click();
  const code = page.locator('pre').last();
  await expect(code).toContainText('package main');
  await expect(code).toContainText('"output_format": "mp3"');
  await expect(code).toContainText('os.WriteFile');
});
