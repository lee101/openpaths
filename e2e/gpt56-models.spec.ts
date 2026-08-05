/**
 * E2E tests for the GPT-5.6 tier family (Sol, Terra, Luna).
 * Calls the models via /v1/chat/completions; while the upstream org lacks
 * gpt-5.6 access these are served by the configured fallback chain, so the
 * tests assert a valid completion rather than a specific upstream model.
 *
 * Opt-in (spends provider credits):
 * RUN_PAID_MODEL_E2E=1 TARGET_URL=https://openpaths.io npx playwright test e2e/gpt56-models.spec.ts --config=playwright.real.config.ts
 */
import { test, expect } from '@playwright/test';

const BASE = process.env.TARGET_URL || 'http://localhost:8090';
test.skip(process.env.RUN_PAID_MODEL_E2E !== '1', 'set RUN_PAID_MODEL_E2E=1 to spend credits on live model checks');

function uniqueEmail() {
  return `e2e-gpt56-${Date.now()}-${Math.random().toString(36).slice(2)}@test.openpaths.io`;
}

async function getApiKey(request: any): Promise<string> {
  const res = await request.post(`${BASE}/auth/register`, {
    data: { email: uniqueEmail(), password: 'testpass1234', name: 'GPT56 Test' },
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
  const res = await request.post(`${BASE}/v1/chat/completions`, {
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
    data: {
      model,
      messages: [{ role: 'user', content: prompt }],
      max_tokens: 512,
    },
  });
  return res;
}

for (const tier of ['sol', 'terra', 'luna'] as const) {
  const id = `gpt-5.6-${tier}`;

  test.describe(`GPT-5.6 ${tier[0].toUpperCase()}${tier.slice(1)}`, () => {
    test('returns a valid chat completion', async ({ request }) => {
      const apiKey = await getApiKey(request);
      const res = await chatCompletion(request, apiKey, id, 'Say hello in one sentence.');
      expect(res.status()).toBe(200);
      const body = await res.json();
      expect(body.choices).toBeDefined();
      expect(body.choices.length).toBeGreaterThan(0);
      expect(body.choices[0].message.content.length).toBeGreaterThan(0);
    });

    test(`works with passthrough alias gpt5.6-${tier}`, async ({ request }) => {
      const apiKey = await getApiKey(request);
      const res = await chatCompletion(request, apiKey, `gpt5.6-${tier}`, 'What is 2+2? Answer with just the number.');
      expect(res.status()).toBe(200);
      const body = await res.json();
      expect(body.choices[0].message.content).toMatch(/4/);
    });
  });
}

test.describe('GPT-5.6 Luna streaming', () => {
  test('supports streaming', async ({ request }) => {
    const apiKey = await getApiKey(request);
    const res = await request.post(`${BASE}/v1/chat/completions`, {
      headers: {
        Authorization: `Bearer ${apiKey}`,
        'Content-Type': 'application/json',
      },
      data: {
        model: 'gpt-5.6-luna',
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
