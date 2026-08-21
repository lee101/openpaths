/**
 * Opt-in live xAI video E2E.
 *
 * This spends real provider credits and can take several minutes:
 *   RUN_XAI_VIDEO_E2E=1 XAI_API_KEY=... npx playwright test e2e/xai-video-live.spec.ts --config=playwright.real.config.ts
 */
import { expect, test } from '@playwright/test';

const BASE = process.env.TARGET_URL || 'http://localhost:8090';
const paidApiKey = process.env.PAID_MODEL_API_KEY || '';
const shouldRun = process.env.RUN_XAI_VIDEO_E2E === '1' && !!process.env.XAI_API_KEY && !!paidApiKey;

async function getApiKey(request: any): Promise<string> {
  return paidApiKey;
}

test.describe('xAI Grok Imagine video live', () => {
  test.skip(!shouldRun, 'set RUN_XAI_VIDEO_E2E=1, XAI_API_KEY, and PAID_MODEL_API_KEY to run the live video test');

  test('generates and backend-reencodes a Grok video to WebM', async ({ request }) => {
    test.setTimeout(20 * 60 * 1000);
    const apiKey = await getApiKey(request);
    const res = await request.post(`${BASE}/v1/videos/generations`, {
      timeout: 20 * 60 * 1000,
      headers: {
        Authorization: `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      data: {
        model: 'grok-imagine-video',
        prompt: 'A compact luminous AI routing console on a matte black desk, glass panels, slow cinematic push-in, realistic reflections, no readable text.',
        duration: 6,
        resolution: '480p',
        aspect_ratio: '16:9',
        output_format: 'webm',
      },
    });
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.video_url).toMatch(/\.webm($|\?)/);
    expect(body.original_video_url).toMatch(/^https?:\/\//);
    expect(body.output_format).toBe('webm');
    expect(body.bytes).toBeGreaterThan(0);
    expect(body.original_bytes).toBeGreaterThan(0);
  });
});
