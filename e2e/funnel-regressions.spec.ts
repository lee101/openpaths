import { test, expect } from '@playwright/test';

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    (window as any).__funnelEvents = [];
    window.th = (...args: unknown[]) => (window as any).__funnelEvents.push(args);
  });
  await page.route('**/*', route => {
    if (new URL(route.request().url()).hostname !== '127.0.0.1') return route.abort();
    return route.continue();
  });
  await page.route('**/account/stripe/config', route => route.fulfill({ json: { publishable_key: 'pk_test_local_fixture' } }));
});

test('mobile sign-in remains on the originating page and records the funnel', async ({ page }) => {
  await page.route('**/auth/login', route => route.fulfill({ json: { token: 'fixture-session', api_key: 'op_fixture', user: { id: 'user-fixture', email: 'fixture@example.test' } } }));
  await page.goto('/pricing');
  const originalUrl = page.url();
  await page.evaluate(() => window.dispatchEvent(new CustomEvent('op-auth-required')));
  await expect(page.getByTestId('auth-modal')).toBeVisible();
  await page.getByTestId('auth-modal-email').fill('fixture@example.test');
  await page.getByTestId('auth-modal-password').fill('fixturepassword');
  await page.getByTestId('auth-modal-submit').click();
  await expect(page.getByTestId('auth-modal')).not.toBeVisible();
  expect(page.url()).toBe(originalUrl);
  expect(await page.evaluate(() => localStorage.getItem('op_api_key'))).toBe('op_fixture');
  const events = await page.evaluate(() => (window as any).__funnelEvents);
  expect(events.some((e: any[]) => e[1] === 'sign_in_started')).toBe(true);
  expect(events.some((e: any[]) => e[1] === 'sign_in_completed')).toBe(true);
  expect(JSON.stringify(events)).not.toContain('fixturepassword');
});

test('invalid credit amounts never create checkout sessions', async ({ page }) => {
  let requests = 0;
  await page.route('**/account/stripe/checkout', route => { requests++; return route.fulfill({ json: { error: { message: 'fixture' } } }); });
  await page.goto('/pricing');
  await page.evaluate(() => window.dispatchEvent(new CustomEvent('op-credits-required', { detail: { surface: 'test' } })));
  await expect(page.getByRole('dialog', { name: 'Add prepaid credits' })).toBeVisible();
  for (const invalid of ['0', '501', '1.5', '']) {
    await page.getByTestId('custom-amount-input').fill(invalid);
    await expect(page.getByTestId('stripe-checkout-btn')).toBeDisabled();
  }
  await page.getByTestId('custom-amount-input').fill('25');
  await expect(page.getByTestId('stripe-checkout-btn')).toBeEnabled();
  expect(requests).toBe(0);
});

test('missing checkout secret is recoverable and never reports begin_checkout', async ({ page }) => {
  await page.route('**/account/stripe/checkout', route => route.fulfill({ json: {} }));
  await page.goto('/pricing');
  const originalUrl = page.url();
  await page.evaluate(() => window.dispatchEvent(new CustomEvent('op-credits-required', { detail: { surface: 'test' } })));
  await expect(page.getByTestId('stripe-modal')).toBeVisible();
  await page.getByTestId('stripe-checkout-btn').click();
  await expect(page.getByRole('alert')).toHaveText('Failed to create checkout');
  await expect(page.getByTestId('stripe-checkout-btn')).toBeEnabled();
  const events = await page.evaluate(() => (window as any).__funnelEvents);
  expect(events.some((e: any[]) => e[1] === 'checkout_failed')).toBe(true);
  expect(events.some((e: any[]) => e[1] === 'begin_checkout')).toBe(false);
  await page.getByRole('button', { name: 'Close checkout' }).click();
  await expect(page.getByTestId('stripe-modal')).not.toBeVisible();
  expect(page.url()).toBe(originalUrl);
});

test('routing experiment article is discoverable and has canonical product CTAs', async ({ page }) => {
  await page.goto('/blog');
  await page.getByRole('link', { name: /LLM Routing Experiments: A Cost, Latency and Quality Checklist/ }).click();
  const canonical = 'https://openpaths.io/blog/llm-routing-cost-latency-quality-experiment-checklist';
  await expect(page).toHaveURL(/\/blog\/llm-routing-cost-latency-quality-experiment-checklist$/);
  await expect(page.getByRole('heading', { name: 'LLM Routing Experiments: A Cost, Latency and Quality Checklist' })).toBeVisible();
  await expect(page.locator('link[rel="canonical"]')).toHaveAttribute('href', canonical);
  await expect(page.locator('meta[name="description"]')).toHaveAttribute('content', /reproducible workload/);
  const article = page.getByRole('article');
  for (const [label, href] of [
    ['Explore the model catalog', '/models'], ['Read the API docs', '/docs'],
    ['try a prompt in the Playground', '/playground'], ['review current pricing', '/pricing'],
    ['open your account', '/account'],
  ]) {
    await expect(article.getByRole('link', { name: label })).toHaveAttribute('href', href);
  }
  expect(await page.locator('script[type="application/ld+json"]').allTextContents()).toEqual(expect.arrayContaining([expect.stringContaining(canonical)]));
});
