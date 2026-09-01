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
const paidApiKey = process.env.PAID_MODEL_API_KEY || '';
const shouldRun = process.env.RUN_PAID_MODEL_E2E === '1' && !!targetURL && !!paidApiKey;
const defaultModels = [
  // glm-5.3 always thinks: 'none' below is normalized server-side to 'low',
  // which keeps the reply to ~3 completion tokens and 0 reasoning tokens.
  'glm-5.3-flash',
  'ox-alpha',
  'glm-5.3',
  'or/glm-5.3',
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

test.describe('new model live smoke checks', () => {
  test.describe.configure({ mode: 'serial' });
  test.skip(!shouldRun, 'set RUN_PAID_MODEL_E2E=1, TARGET_URL, and PAID_MODEL_API_KEY to run paid model checks');

  let apiKey = '';

  test.beforeAll(async ({ request }) => {
    apiKey = paidApiKey;
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
