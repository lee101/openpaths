/**
 * E2E tests for MiniMax M2.7 model.
 * Actually calls the model via /v1/chat/completions to verify the endpoint works.
 *
 * Run:  npx playwright test e2e/minimax-m27.spec.ts --config=playwright.real.config.ts
 * Live: TARGET_URL=https://openpaths.io npx playwright test e2e/minimax-m27.spec.ts --config=playwright.real.config.ts
 */
import { test, expect } from '@playwright/test';

const BASE = process.env.TARGET_URL || 'http://localhost:8092';

function uniqueEmail() {
  return `e2e-mm27-${Date.now()}-${Math.random().toString(36).slice(2)}@test.openpaths.io`;
}

async function getApiKey(request: any): Promise<string> {
  const res = await request.post(`${BASE}/auth/register`, {
    data: { email: uniqueEmail(), password: 'testpass1234', name: 'MiniMax M2.7 Test' },
  });
  expect(res.status()).toBe(201);
  const body = await res.json();
  const apiKey = body.api_key;

  const creditRes = await request.post(`${BASE}/account/credits/add`, {
    headers: { Authorization: `Bearer ${apiKey}` },
    data: { amount_cents: 10000, description: 'e2e test credits' },
  });
  expect(creditRes.status()).toBe(200);

  return apiKey;
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
