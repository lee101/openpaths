import { expect, test } from '@playwright/test';

test('builds and renders a multi-speaker Gemini TTS request', async ({ page }) => {
  let payload: Record<string, unknown> | undefined;
  await page.addInitScript(() => localStorage.setItem('op_api_key', 'op-test'));
  await page.route('**/v1/audio/speech', async route => {
    payload = route.request().postDataJSON();
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        audio: 'UklGRiQAAABXQVZFZm10IBAAAAABAAEAwF0AAIC7AAACABAAZGF0YQAAAAA=',
        format: 'wav',
        characters: 480,
      }),
    });
  });

  await page.goto('/tools/google-tts');
  await expect(page.getByRole('heading', { name: 'Direct every voice.' })).toBeVisible();
  await page.getByRole('button', { name: 'Generate speech' }).click();

  await expect.poll(() => payload).toBeTruthy();
  expect(payload?.model).toBe('gemini-3.1-flash-tts-preview');
  expect(payload?.speaker_voices).toEqual([
    { speaker: 'Speaker 1', voice: 'Fenrir' },
    { speaker: 'Speaker 2', voice: 'Puck' },
  ]);
  expect(payload?.input).toContain('# Audio Profile');
  expect(payload?.input).toContain('## Scene:');
  await expect(page.getByText('Ready', { exact: true })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Download WAV' })).toBeVisible();
});

test('switches modes, inserts direction, and updates generated code', async ({ page }) => {
  await page.goto('/tools/google-tts');
  await page.getByRole('button', { name: 'Single speaker' }).click();
  await expect(page.getByText('A calm, intimate documentary narrator')).toBeVisible();

  const transcript = page.getByLabel('Transcript');
  await transcript.fill('This is a test.');
  await page.getByRole('button', { name: '[urgency]', exact: true }).click();
  await expect(transcript).toHaveValue(/\[urgency\]/);

  await page.getByRole('button', { name: 'JS', exact: true }).click();
  await expect(page.getByText('const result = await client.post')).toBeVisible();
  await expect(page.locator('pre')).toContainText('"voice": "Zephyr"');
});
