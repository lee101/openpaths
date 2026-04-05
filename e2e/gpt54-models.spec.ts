/**
 * E2E tests for GPT-5.4 Mini and GPT-5.4 Nano models.
 * Actually calls the models via /v1/chat/completions to verify they work end-to-end.
 *
 * Run:  npx playwright test e2e/gpt54-models.spec.ts --config=playwright.real.config.ts
 * Live: TARGET_URL=https://openpaths.io npx playwright test e2e/gpt54-models.spec.ts --config=playwright.real.config.ts
 */
import { test, expect } from '@playwright/test';

const BASE = process.env.TARGET_URL || 'http://localhost:8090';

function uniqueEmail() {
  return `e2e-gpt54-${Date.now()}-${Math.random().toString(36).slice(2)}@test.openpaths.io`;
}

async function getApiKey(request: any): Promise<string> {
  const res = await request.post(`${BASE}/auth/register`, {
    data: { email: uniqueEmail(), password: 'testpass1234', name: 'GPT54 Test' },
  });
  expect(res.status()).toBe(201);
  const body = await res.json();
  const apiKey = body.api_key;

  // Add test credits ($1.00 = 10000 hundredths-of-a-cent)
  const creditRes = await request.post(`${BASE}/account/credits/add`, {
    headers: { Authorization: `Bearer ${apiKey}` },
    data: { amount_cents: 10000, description: 'e2e test credits' },
  });
  expect(creditRes.status()).toBe(200);

  return apiKey;
}

async function chatCompletion(request: any, apiKey: string, model: string, prompt: string) {
  const res = await request.post(`${BASE}/v1/chat/completions`, {
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
    data: {
      model,
      messages: [{ role: 'user', content: prompt }],
      max_tokens: 128,
    },
  });
  return res;
}

test.describe('GPT-5.4 Mini', () => {
  test('returns a valid chat completion', async ({ request }) => {
    const apiKey = await getApiKey(request);
    const res = await chatCompletion(request, apiKey, 'gpt-5.4-mini', 'Say hello in one sentence.');
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.choices).toBeDefined();
    expect(body.choices.length).toBeGreaterThan(0);
    expect(body.choices[0].message.content.length).toBeGreaterThan(0);
    expect(body.model).toContain('gpt-5.4-mini');
  });

  test('works with alias gpt54-mini', async ({ request }) => {
    const apiKey = await getApiKey(request);
    const res = await chatCompletion(request, apiKey, 'gpt54-mini', 'What is 2+2? Answer with just the number.');
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.choices[0].message.content).toMatch(/4/);
  });

  test('supports streaming', async ({ request }) => {
    const apiKey = await getApiKey(request);
    const res = await request.post(`${BASE}/v1/chat/completions`, {
      headers: {
        Authorization: `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      data: {
        model: 'gpt-5.4-mini',
        messages: [{ role: 'user', content: 'Say hi' }],
        max_tokens: 32,
        stream: true,
      },
    });
    expect(res.status()).toBe(200);
    const text = await res.text();
    expect(text).toContain('data:');
  });
});

test.describe('GPT-5.4 Nano', () => {
  test('returns a valid chat completion', async ({ request }) => {
    const apiKey = await getApiKey(request);
    const res = await chatCompletion(request, apiKey, 'gpt-5.4-nano', 'Say hello in one sentence.');
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.choices).toBeDefined();
    expect(body.choices.length).toBeGreaterThan(0);
    expect(body.choices[0].message.content.length).toBeGreaterThan(0);
    expect(body.model).toContain('gpt-5.4-nano');
  });

  test('works with alias gpt54-nano', async ({ request }) => {
    const apiKey = await getApiKey(request);
    const res = await chatCompletion(request, apiKey, 'gpt54-nano', 'What is 3+3? Answer with just the number.');
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.choices[0].message.content).toMatch(/6/);
  });

  test('handles classification task efficiently', async ({ request }) => {
    const apiKey = await getApiKey(request);
    const res = await chatCompletion(
      request,
      apiKey,
      'gpt-5.4-nano',
      'Classify this sentiment as positive, negative, or neutral: "I love this product!" Reply with one word.'
    );
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.choices[0].message.content.toLowerCase()).toMatch(/positive/);
  });
});

test.describe('Auto routing includes new models', () => {
  test('auto-easy-task resolves and returns a completion', async ({ request }) => {
    const apiKey = await getApiKey(request);
    const res = await chatCompletion(request, apiKey, 'auto-easy-task', 'Classify this as a fruit or vegetable: banana');
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.choices[0].message.content.length).toBeGreaterThan(0);
  });

  test('auto-medium-task resolves and returns a completion', async ({ request }) => {
    const apiKey = await getApiKey(request);
    const res = await chatCompletion(request, apiKey, 'auto-medium-task', 'Write a simple Python function to reverse a string');
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.choices[0].message.content).toMatch(/def|return|reverse/i);
  });

  test('auto-think resolves a trivial prompt with a completion', async ({ request }) => {
    const apiKey = await getApiKey(request);
    const res = await chatCompletion(request, apiKey, 'auto-think', 'Say yes.');
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.choices[0].message.content.toLowerCase()).toContain('yes');
  });
});
