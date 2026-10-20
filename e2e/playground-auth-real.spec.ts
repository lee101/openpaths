/**
 * Real-user playground auth e2e: register, sign out, sign back in, and use the
 * playground with a real account against a live server.
 *
 * TARGET_URL=https://openpaths.io RUN_PLAYGROUND_AUTH_E2E=1 \
 *   npx playwright test e2e/playground-auth-real.spec.ts --config=playwright.playground-auth.config.ts
 * Optional: E2E_EMAIL / E2E_PASSWORD to use an existing user instead of registering one.
 */
import { expect, test, type Page } from '@playwright/test';

const target = process.env.TARGET_URL || '';
test.skip(process.env.RUN_PLAYGROUND_AUTH_E2E !== '1' || !target, 'set RUN_PLAYGROUND_AUTH_E2E=1 and TARGET_URL');

const email = process.env.E2E_EMAIL || `e2e-pg-${Date.now()}-${Math.random().toString(36).slice(2, 7)}@test.com`;
const password = process.env.E2E_PASSWORD || 'password123-e2e';
const url = '/playground?model=grok-4.7';

async function expectSignedInPlayground(page: Page) {
  await expect(page.getByTestId('nav-signin')).toHaveCount(0);
  await expect(page.getByText('Sign in / sign up')).toHaveCount(0);
  await expect(page.getByText('Set your API key')).toHaveCount(0);
  await expect(page.getByTestId('playground-session-loading')).toHaveCount(0, { timeout: 15000 });
  await expect(page.getByTestId('chat-send')).toBeVisible();
  await expect.poll(() => page.evaluate(() => localStorage.getItem('op_api_key') || ''), { timeout: 15000 }).toMatch(/^(sk-op-|op_)/);
  await expect(page.locator('textarea').first()).toBeEnabled();
}

test('sign up, sign out, sign in again, and use the playground', async ({ page }) => {
  test.setTimeout(120000);
  if (!process.env.E2E_EMAIL) {
    const res = await page.request.post(`${target}/auth/register`, { data: { email, password, name: 'E2E Playground' } });
    expect(res.status()).toBe(201);
  }

  // Signed out: prompt is shown.
  await page.goto(url);
  await expect(page.getByText('Sign in / sign up')).toBeVisible();

  // Sign in through the nav modal.
  const signIn = async () => {
    await page.getByTestId('nav-signin').click();
    await expect(page.getByTestId('auth-modal')).toBeVisible();
    await page.getByTestId('auth-modal-email').fill(email);
    await page.getByTestId('auth-modal-password').fill(password);
    await page.getByTestId('auth-modal-submit').click();
  };
  await signIn();
  await expectSignedInPlayground(page);

  // Survive reload as a JWT-only session (drop the stored API key; playground must re-mint it).
  await page.evaluate(() => localStorage.removeItem('op_api_key'));
  await page.reload();
  await expectSignedInPlayground(page);

  // Use it.
  await page.locator('textarea').first().fill('Reply with the single word: pong');
  await page.getByTestId('chat-send').click();
  await expect(page.getByText(/pong/i).last()).toBeVisible({ timeout: 60000 });

  // Real sign out via the account page, then sign in again.
  await page.goto('/account');
  await page.getByTestId('logout-btn').click();
  await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();
  await page.goto(url);
  await expect(page.getByTestId('nav-signin')).toBeVisible();
  expect(await page.evaluate(() => localStorage.getItem('op_token'))).toBeNull();
  await signIn();
  await expectSignedInPlayground(page);
});
