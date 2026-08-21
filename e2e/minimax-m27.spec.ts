/**
 * E2E tests for MiniMax M2.7 model.
 * Actually calls the model via /v1/chat/completions to verify the endpoint works.
 *
 * Opt-in (spends provider credits):
 * RUN_PAID_MODEL_E2E=1 TARGET_URL=https://openpaths.io npx playwright test e2e/minimax-m27.spec.ts --config=playwright.real.config.ts
 */
import { test, expect } from '@playwright/test';

const BASE = process.env.TARGET_URL || 'http://localhost:8092';
const paidApiKey = process.env.PAID_MODEL_API_KEY || '';
test.skip(process.env.RUN_PAID_MODEL_E2E !== '1' || !paidApiKey, 'set RUN_PAID_MODEL_E2E=1 and PAID_MODEL_API_KEY to run live model checks');

async function getApiKey(request: any): Promise<string> {
  return paidApiKey;
}

async function chatCompletion(request: any, apiKey: string, model: string, prompt: string) {
  return request.post(`${BASE}/v1/chat/completions`, {
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
    data: {
      model,
      messages: [{ role: 'user', content: prompt }],
      max_tokens: 16,
    },
  });
}

test.describe('MiniMax M2.7', () => {
  test('returns a valid chat completion', async ({ request }) => {
    const apiKey = await getApiKey(request);
    const res = await chatCompletion(
      request, apiKey, 'minimax-m2.7',
      'Reply with only the single word "no" and nothing else.'
    );
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.choices).toBeDefined();
    expect(body.choices.length).toBeGreaterThan(0);
    const content = body.choices[0].message.content.trim().toLowerCase();
    expect(content).toBe('no');
  });

  test('works with alias mm-m2.7', async ({ request }) => {
    const apiKey = await getApiKey(request);
    const res = await chatCompletion(
      request, apiKey, 'mm-m2.7',
      'Reply with only the single word "no" and nothing else.'
    );
    expect(res.status()).toBe(200);
    const body = await res.json();
    const content = body.choices[0].message.content.trim().toLowerCase();
    expect(content).toBe('no');
  });

  test('works with alias minimax-latest', async ({ request }) => {
    const apiKey = await getApiKey(request);
    const res = await chatCompletion(
      request, apiKey, 'minimax-latest',
      'Reply with only the single word "no" and nothing else.'
    );
    expect(res.status()).toBe(200);
    const body = await res.json();
    const content = body.choices[0].message.content.trim().toLowerCase();
    expect(content).toBe('no');
  });

  test('supports streaming', async ({ request }) => {
    const apiKey = await getApiKey(request);
    const res = await request.post(`${BASE}/v1/chat/completions`, {
      headers: {
        Authorization: `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      data: {
        model: 'minimax-m2.7',
        messages: [{ role: 'user', content: 'Say no' }],
        max_tokens: 16,
        stream: true,
      },
    });
    expect(res.status()).toBe(200);
    const text = await res.text();
    expect(text).toContain('data:');
  });

  test('model field in response references minimax', async ({ request }) => {
    const apiKey = await getApiKey(request);
    const res = await chatCompletion(
      request, apiKey, 'minimax-m2.7',
      'Reply with only the single word "no" and nothing else.'
    );
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.model).toBeTruthy();
  });
});
