import { test, expect } from '@playwright/test';

const fusedBody = JSON.stringify({
  id: 'chatcmpl_fusion_e2e',
  object: 'chat.completion',
  model: 'gemini-latest',
  choices: [
    {
      index: 0,
      message: { role: 'assistant', content: 'Fused answer: experts disagree mainly on revenue recycling.' },
      finish_reason: 'stop',
    },
  ],
  usage: { prompt_tokens: 50, completion_tokens: 20, total_tokens: 70 },
});

test.describe('Model Fusion space', () => {
  test('renders from nav with the quality preset and a request preview', async ({ page }) => {
    await page.goto('/');
    await page.click('nav >> text=Fusion');
    await expect(page).toHaveURL('/fusion');

    await expect(page.getByRole('heading', { name: /Fuse multiple model answers into one stronger result/i })).toBeVisible();

    // Backend + preset toggles present, OpenPaths is the default backend.
    await expect(page.getByRole('button', { name: 'openpaths' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'openrouter' })).toBeVisible();
    await expect(page.getByText('OpenPaths API key')).toBeVisible();

    // Quality preset selects the three quality models (panel count 3/8).
    await expect(page.getByText('3/8 panel models')).toBeVisible();

    // Reveal the request preview and confirm it targets the OpenPaths endpoint
    // with the openpaths fusion payload.
    await page.getByRole('button', { name: /Show request/i }).click();
    const pre = page.locator('pre');
    await expect(pre).toContainText('/v1/chat/completions');
    await expect(pre).toContainText('"type": "openpaths"');
    await expect(pre).toContainText('"analysis_models"');
    // Quality panel resolves to native OpenPaths model ids.
    await expect(pre).toContainText('claude-opus-latest');
    await expect(pre).toContainText('gpt-5.5');
  });

  test('switching to the OpenRouter backend swaps endpoint and payload shape', async ({ page }) => {
    await page.goto('/fusion');
    await page.getByRole('button', { name: 'openrouter' }).click();

    await expect(page.getByText('OpenRouter API key')).toBeVisible();
    await page.getByRole('button', { name: /Show request/i }).click();

    const pre = page.locator('pre');
    await expect(pre).toContainText('https://openrouter.ai/api/v1/chat/completions');
    await expect(pre).toContainText('openrouter:fusion');
    // OpenRouter uses native provider-prefixed ids, not OpenPaths aliases.
    await expect(pre).toContainText('~anthropic/claude-opus-latest');
    await expect(pre).toContainText('"tool_choice": "required"');
  });

  test('budget preset narrows the panel to the budget models', async ({ page }) => {
    await page.goto('/fusion');
    await page.getByRole('button', { name: 'budget' }).click();

    await expect(page.getByText('3/8 panel models')).toBeVisible();
    await page.getByRole('button', { name: /Show request/i }).click();
    const pre = page.locator('pre');
    await expect(pre).toContainText('gemini-3.5-flash');
    await expect(pre).toContainText('deepseek-v4-flash');
    await expect(pre).toContainText('kimi-k2.5');
  });

  test('runs a mocked OpenPaths fusion and renders the fused answer', async ({ page }) => {
    let requestBody: any = null;
    await page.route('**/v1/chat/completions', async route => {
      requestBody = route.request().postDataJSON();
      await route.fulfill({ status: 200, contentType: 'application/json', body: fusedBody });
    });

    await page.goto('/fusion');
    await page.locator('input[placeholder="op-..."]').fill('op-fusion-e2e-key');
    await page.getByRole('button', { name: /Run Fusion/i }).click();

    await expect(page.getByText(/Fused answer: experts disagree/i)).toBeVisible();

    // The browser sent the OpenPaths fusion payload with native panel ids. With
    // "Fuse with: auto" the judge is the first panel model (claude-opus-latest).
    expect(requestBody).toMatchObject({
      model: 'claude-opus-latest',
      fusion: {
        type: 'openpaths',
        model: 'claude-opus-latest',
        analysis_models: ['claude-opus-latest', 'gpt-5.5', 'gemini-latest'],
      },
    });
  });

  test('adds the same model multiple times as separate panel variants', async ({ page }) => {
    await page.goto('/fusion');
    // Default quality preset selects 3 models; add GLM-5.2 three times on top via
    // its row "+" button (addModelCopy appends a copy each click).
    const addGlm = page.getByRole('button', { name: 'Add another GLM-5.2' });
    await addGlm.click();
    await addGlm.click();
    await addGlm.click();

    // The row shows a 3x count badge and the panel counter reads 6/8 (3 + 3).
    await expect(page.getByText('3x')).toBeVisible();
    await expect(page.getByText('6/8 panel models')).toBeVisible();

    await page.getByRole('button', { name: /Show request/i }).click();
    const pre = page.locator('pre');
    await expect(pre).toContainText('"analysis_models"');
    // glm-5.2 appears three times in the panel.
    const text = await pre.innerText();
    const occurrences = text.split('"glm-5.2"').length - 1;
    expect(occurrences).toBeGreaterThanOrEqual(3);
  });

  test('requires an API key before running', async ({ page }) => {
    await page.goto('/fusion');
    // Quality preset is selected by default, so the only missing input is the key.
    await page.getByRole('button', { name: /Run Fusion/i }).click();
    await expect(page.getByText(/Add an OpenPaths API key to run fusion/i)).toBeVisible();
  });

  test('surfaces upstream errors from the fusion endpoint', async ({ page }) => {
    await page.route('**/v1/chat/completions', async route => {
      await route.fulfill({
        status: 502,
        contentType: 'application/json',
        body: JSON.stringify({ error: { message: 'all fusion panel models failed' } }),
      });
    });

    await page.goto('/fusion');
    await page.locator('input[placeholder="op-..."]').fill('op-fusion-e2e-key');
    await page.getByRole('button', { name: /Run Fusion/i }).click();

    await expect(page.getByText(/all fusion panel models failed/i)).toBeVisible();
  });
});
