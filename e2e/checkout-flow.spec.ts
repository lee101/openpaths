import { test, expect } from '@playwright/test';

// Full e2e flow against live backend on openpaths.local:8080
// Tests: register -> logout -> login -> open stripe checkout dialog
// All in a single test, no page reloads

const BASE = 'http://openpaths.local:8090';
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

    // Should land on the dashboard overview (no reload)
    await expect(page.locator('h1:has-text("Overview")')).toBeVisible({ timeout: 10000 });
    await expect(page.locator(`text=${email}`)).toBeVisible();

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

    // Should be back on overview
    await expect(page.locator('h1:has-text("Overview")')).toBeVisible({ timeout: 10000 });
    await expect(page.locator(`text=${email}`)).toBeVisible();

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

    // Verify amount buttons
    for (const amt of [10, 25, 50, 100]) {
      await expect(page.getByTestId(`amount-${amt}`)).toBeVisible();
    }

    // Default is $25
    await expect(page.getByTestId('stripe-checkout-btn')).toContainText('Pay $25');

    // Change to $10
    await page.getByTestId('amount-10').click();
    await expect(page.getByTestId('stripe-checkout-btn')).toContainText('Pay $10');

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
    expect(page.url()).toBe(BASE + '/account');
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
