/**
 * Cost-capped, explicitly opt-in production smoke tests for newly added model
 * routes. Default and real-server suites skip this file unless both gates are
 * supplied:
 *
 * RUN_PAID_MODEL_E2E=1 TARGET_URL=https://openpaths.io npm run test:e2e:paid-models
 *
 * Optionally limit the run further:
 * PAID_MODEL_E2E_MODELS=qwen-latest,or/gpt-5.6-luna
 */
import { expect, test } from '@playwright/test';

const targetURL = process.env.TARGET_URL;
const shouldRun = process.env.RUN_PAID_MODEL_E2E === '1' && !!targetURL;
const defaultModels = [
  'qwen-latest',
  'or/gpt-5.6-sol',
  'or/gpt-5.6-terra',
  'or/gpt-5.6-luna',
  'or/gpt-5-codex',
];
const modelIDs = (process.env.PAID_MODEL_E2E_MODELS || defaultModels.join(','))
  .split(',')
  .map(model => model.trim())
  .filter(Boolean);

function uniqueEmail() {
  return `e2e-new-models-${Date.now()}-${Math.random().toString(36).slice(2)}@test.openpaths.io`;
}

test.describe('new model live smoke checks', () => {
  test.describe.configure({ mode: 'serial' });
  test.skip(!shouldRun, 'set RUN_PAID_MODEL_E2E=1 and TARGET_URL to run paid model checks');

  let apiKey = '';

  test.beforeAll(async ({ request }) => {
    const registration = await request.post(`${targetURL}/auth/register`, {
      data: {
        email: uniqueEmail(),
        password: 'testpass1234',
        name: 'New Models E2E',
      },
    });
    expect(registration.status()).toBe(201);
    apiKey = (await registration.json()).api_key;

    // Ten cents is far above this suite's capped few-token spend, while much
    // smaller than the historical $1 test grants.
    const credit = await request.post(`${targetURL}/account/credits/add`, {
      headers: { Authorization: `Bearer ${apiKey}` },
      data: { amount_cents: 1000, description: 'opt-in new-model smoke test' },
    });
    expect(credit.status()).toBe(200);
  });

  for (const model of modelIDs) {
    test(`${model} replies only hi`, async ({ request }) => {
      const response = await request.post(`${targetURL}/v1/chat/completions`, {
        headers: {
          Authorization: `Bearer ${apiKey}`,
          'Content-Type': 'application/json',
        },
        data: {
          model,
          messages: [{ role: 'user', content: 'say hi nothing else' }],
          max_tokens: 16,
          reasoning_effort: 'none',
        },
      });
      const body = await response.json();
      expect(response.status(), JSON.stringify(body)).toBe(200);
      expect(body.choices?.[0]?.message?.content?.trim()).toMatch(/^hi[!.]?$/i);
    });
  }
});
