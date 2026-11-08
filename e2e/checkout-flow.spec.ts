import { test, expect } from '@playwright/test';

// Full e2e flow against live backend on openpaths.local:8080
// Tests: register -> logout -> login -> open stripe checkout dialog
// All in a single test, no page reloads

const BASE = process.env.TARGET_URL || 'http://openpaths.local:8090';
const unique = () => `e2e_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;

test.describe('Full Checkout Flow (live backend)', () => {
  test.use({ baseURL: BASE });

  test('register, logout, login, open stripe dialog - no reloads', async ({ page }) => {
    const email = `${unique()}@test.openpaths.io`;
    const password = 'testpass123';
    const name = 'E2E Tester';

    // Navigate to account page
    await page.goto('/account');
    await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();

    // -- REGISTER --
    await page.getByTestId('auth-toggle').click();
    await expect(page.locator('h1:has-text("Create Account")')).toBeVisible();

    await page.locator('input[placeholder="Name"]').fill(name);
    await page.getByTestId('auth-email').fill(email);
    await page.getByTestId('auth-password').fill(password);
    await page.getByTestId('auth-submit').click();

    // Should land on the signed-in dashboard (no reload). The landing tab has changed over time, so assert the
    // account sidebar + email, then open Overview explicitly for the balance check.
    await expect(page.getByRole('heading', { name: 'Account', level: 2 })).toBeVisible({ timeout: 10000 });
    await expect(page.locator(`text=${email}`).first()).toBeVisible();
    await page.getByRole('button', { name: 'Overview' }).click();

    // Balance should show $0.00 for new account
    await expect(page.getByTestId('balance')).toContainText('$0.00');

    // Verify we have a JWT stored
    const token = await page.evaluate(() => localStorage.getItem('op_token'));
    expect(token).toBeTruthy();
    expect(token!.split('.').length).toBe(3); // JWT format

    // -- LOGOUT (no reload) --
    await page.getByTestId('logout-btn').click();
    await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();

    // Token should be cleared
    const tokenAfterLogout = await page.evaluate(() => localStorage.getItem('op_token'));
    expect(tokenAfterLogout).toBeNull();

    // -- LOGIN (no reload) --
    await page.getByTestId('auth-email').fill(email);
    await page.getByTestId('auth-password').fill(password);
    await page.getByTestId('auth-submit').click();

    // Should be back on the signed-in dashboard
    await expect(page.getByRole('heading', { name: 'Account', level: 2 })).toBeVisible({ timeout: 10000 });
    await expect(page.locator(`text=${email}`).first()).toBeVisible();

    // Verify we got a valid token back
    const newToken = await page.evaluate(() => localStorage.getItem('op_token'));
    expect(newToken).toBeTruthy();
    expect(newToken!.split('.').length).toBe(3);

    // -- NAVIGATE TO BILLING (no reload) --
    await page.getByTestId('tab-billing').click();
    await expect(page.locator('h1:has-text("Billing & Payments")')).toBeVisible();

    // Balance visible on billing page too
    await expect(page.getByTestId('billing-balance')).toContainText('$0.00');

    // Stripe and Solana cards visible
    await expect(page.getByTestId('stripe-card')).toBeVisible();
    await expect(page.getByTestId('solana-card')).toBeVisible();

    // -- OPEN STRIPE DIALOG (no reload) --
    await page.getByTestId('add-funds-stripe-btn').click();
    await expect(page.getByTestId('stripe-modal')).toBeVisible();
    await expect(page.locator('h2:has-text("Add Funds")')).toBeVisible();

    // Credit packs are data-driven (amounts change); assert there are packs, then pick the smallest one.
    const packs = page.locator('[data-testid^="amount-"]');
    await expect(packs.first()).toBeVisible();
    const ids = await packs.evaluateAll(els => els.map(e => e.getAttribute('data-testid') || ''));
    const amounts = ids.map(i => Number(i.replace('amount-', ''))).filter(n => n > 0).sort((x, y) => x - y);
    expect(amounts.length).toBeGreaterThan(1);
    const pick = amounts[0];
    await page.getByTestId(`amount-${pick}`).click();
    await expect(page.getByTestId('stripe-checkout-btn')).toContainText(`Pay $${pick}`);

    // Click checkout - this hits the real Stripe API
    await page.getByTestId('stripe-checkout-btn').click();

    // Should transition to embedded checkout (Stripe iframe loads)
    await expect(page.getByTestId('embedded-checkout-container')).toBeAttached({ timeout: 10000 });

    // The amount buttons should be gone (replaced by Stripe form)
    await expect(page.getByTestId('stripe-checkout-btn')).not.toBeVisible();

    // Stripe iframes should appear (Stripe loads multiple)
    const stripeFrames = page.locator('iframe[src*="stripe.com"]');
    await expect(stripeFrames.first()).toBeAttached({ timeout: 15000 });

    // -- CLOSE AND VERIFY NO RELOAD --
    await page.getByTestId('stripe-modal-close').click();
    await expect(page.getByTestId('stripe-modal')).not.toBeVisible();

    // Still on billing tab, no reload happened
    await expect(page.locator('h1:has-text("Billing & Payments")')).toBeVisible();
    await expect(page.locator(`text=${email}`)).toBeVisible();

    // Verify no navigation/reloads by checking URL is still /account
    expect(new URL(page.url()).pathname).toBe('/account');
  });

  test('login with wrong password shows error', async ({ page }) => {
    await page.goto('/account');
    await page.getByTestId('auth-email').fill('nonexistent@test.openpaths.io');
    await page.getByTestId('auth-password').fill('wrongpassword');
    await page.getByTestId('auth-submit').click();
    await expect(page.locator('text=Invalid email or password')).toBeVisible({ timeout: 5000 });
  });

  test('register with short password is blocked by validation', async ({ page }) => {
    await page.goto('/account');
    await page.getByTestId('auth-toggle').click();
    await page.getByTestId('auth-email').fill('short@test.openpaths.io');
    await page.getByTestId('auth-password').fill('short');
    await page.getByTestId('auth-submit').click();
    // HTML5 minlength validation prevents submission - still on register form
    await expect(page.locator('h1:has-text("Create Account")')).toBeVisible();
    // Should NOT have navigated to dashboard
    const token = await page.evaluate(() => localStorage.getItem('op_token'));
    expect(token).toBeNull();
  });
});
