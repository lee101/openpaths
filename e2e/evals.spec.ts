import { test, expect } from '@playwright/test';

test.describe('evals page', () => {
  test('renders live evals section with empty-state fallback', async ({ page }) => {
    await page.goto('/evals');

    // The page always renders the external benchmark shell.
    await expect(page.getByText('Artificial Analysis model data')).toBeVisible();

    // Live evals section renders either the snapshot or the pre-sweep fallback.
    const live = page.getByText('OpenPaths Auto vs the frontier');
    const pending = page.getByText('first sweep pending');
    await expect(live.or(pending).first()).toBeVisible();
  });

  test('live snapshot drives tabs, cards, and charts', async ({ page }) => {
    await page.route('**/v1/evals/results', route =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          ran_at: new Date().toISOString(),
          models: [
            {
              model: 'openpaths/auto',
              by_suite: {
                coding: { avg_score: 0.9, pass_rate: 0.9, cases: 6, median_ttft_ms: 200, avg_tps: 50, cost_per_case_micro_usd: 5000 },
                agentic: { avg_score: 0.8, pass_rate: 0.8, cases: 5, median_ttft_ms: 240, avg_tps: 45, cost_per_case_micro_usd: 5800 },
                creative: { avg_score: 0.95, pass_rate: 0.95, cases: 4, median_ttft_ms: 360, avg_tps: 35, cost_per_case_micro_usd: 12000 },
              },
              overall: { avg_score: 0.88, pass_rate: 0.88, cases: 15, median_ttft_ms: 260, avg_tps: 44, cost_per_case_micro_usd: 7600 },
            },
            {
              model: 'gpt-5.6',
              by_suite: {
                coding: { avg_score: 0.92, pass_rate: 0.92, cases: 6, median_ttft_ms: 420, avg_tps: 38, cost_per_case_micro_usd: 24000 },
                agentic: { avg_score: 0.88, pass_rate: 0.88, cases: 5, median_ttft_ms: 500, avg_tps: 34, cost_per_case_micro_usd: 27600 },
                creative: { avg_score: 0.94, pass_rate: 0.94, cases: 4, median_ttft_ms: 750, avg_tps: 26, cost_per_case_micro_usd: 57600 },
              },
              overall: { avg_score: 0.91, pass_rate: 0.91, cases: 15, median_ttft_ms: 550, avg_tps: 33, cost_per_case_micro_usd: 36400 },
            },
          ],
          cases: [
            {
              suite: 'coding',
              case_id: 'mod-arithmetic',
              results: {
                'openpaths/auto': { score: 1, passed: true, ttft_ms: 210, total_ms: 3200, tokens_per_sec: 50, cost_micro_usd: 5000, answer_preview: '{"answer": 9}', error: null },
                'gpt-5.6': { score: 1, passed: true, ttft_ms: 430, total_ms: 5100, tokens_per_sec: 38, cost_micro_usd: 24000, answer_preview: '{"answer": 9}', error: null },
              },
            },
          ],
          auto_vs_best: {
            __overall__: { best_model: 'gpt-5.6', best_score: 0.91, auto_score: 0.88, best_cost_per_case_micro_usd: 36400, auto_cost_per_case_micro_usd: 7600 },
            coding: { best_model: 'gpt-5.6', best_score: 0.92, auto_score: 0.9 },
            agentic: { best_model: 'gpt-5.6', best_score: 0.88, auto_score: 0.8 },
            creative: { best_model: 'gpt-5.6', best_score: 0.94, auto_score: 0.95 },
          },
        }),
      }),
    );

    await page.goto('/evals');

    // Headline auto-vs-best cards render with the hero overall card.
    await expect(page.getByText('OpenPaths Auto vs the frontier')).toBeVisible();
    await expect(page.getByText('best single model:').first()).toBeVisible();

    // Tabs exist and switching to Coding shows the per-case matrix.
    for (const tab of ['Overview', 'Coding', 'Agentic', 'Creative SVG', 'Speed', 'Economics']) {
      await expect(page.getByRole('tab', { name: tab })).toBeVisible();
    }
    await page.getByRole('tab', { name: 'Coding' }).click();
    await expect(page.getByText('Per-case results')).toBeVisible();
    await expect(page.getByText('Modular arithmetic')).toBeVisible();

    // Speed tab shows both metric charts.
    await page.getByRole('tab', { name: 'Speed' }).click();
    await expect(page.getByText('Median TTFT')).toBeVisible();

    // Economics tab shows cost + quality-per-dollar charts.
    await page.getByRole('tab', { name: 'Economics' }).click();
    await expect(page.getByText('Cost per case', { exact: false })).toBeVisible();
  });

  test('has no horizontal overflow on mobile viewport', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/evals');
    await page.getByText('Artificial Analysis model data').waitFor();
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
    expect(overflow).toBeLessThanOrEqual(0);
  });
});
