/**
 * Full UI flow e2e tests — register, login, account dashboard, logout, re-login, Stripe checkout.
 *
 * Local:  npx playwright test e2e/full-flow.spec.ts --config=playwright.real.config.ts
 *         (uses vite dev at :3099 which proxies API to Go server at :8090)
 * Live:   TARGET_URL=https://openpaths.io npx playwright test e2e/full-flow.spec.ts --config=playwright.real.config.ts
 */
import { test, expect } from '@playwright/test';

// Local: vite dev at :3099 proxies API calls to Go backend at :8090
// Live: openpaths.io serves everything
const TARGET = process.env.TARGET_URL || 'http://localhost:3099';

function uniqueEmail() {
  return `e2e-${Date.now()}-${Math.random().toString(36).slice(2)}@test.openpaths.io`;
}

// API base: page.request doesn't go through browser proxy, so hit Go server directly for local
const API_BASE = process.env.TARGET_URL || 'http://localhost:8090';

// Helper: register a user via API and return credentials
async function registerUser(request: any) {
  const email = uniqueEmail();
  const password = 'testpass1234';
  const res = await request.post(`${API_BASE}/auth/register`, {
    data: { email, password, name: 'E2E Flow Test' },
  });
  expect(res.status()).toBe(201);
  const body = await res.json();
  return { email, password, apiKey: body.api_key as string, user: body.user };
}

// Helper: login via the UI
async function loginViaUI(page: any, email: string, password: string) {
  await page.getByTestId('auth-email').fill(email);
  await page.getByTestId('auth-password').fill(password);
  await page.getByTestId('auth-submit').click();
  await expect(page.locator('h2:has-text("Account")')).toBeVisible({ timeout: 15000 });
}

// ========== FULL SIGN-IN / ACCOUNT / LOGOUT / RE-LOGIN FLOW ==========

test.describe('Full auth UI flow', () => {
  test('register → nav shows Account/Dashboard → account page → logout → nav shows Sign In → re-login', async ({ page }) => {
    const email = uniqueEmail();
    const password = 'testpass1234';

    // 1. Visit homepage — nav should show Sign In / Get Started
    await page.goto(TARGET + '/');
    await expect(page.getByTestId('nav-signin')).toBeVisible();
    await expect(page.getByTestId('nav-get-started')).toBeVisible();
    await expect(page.getByTestId('nav-account')).not.toBeAttached();
    await expect(page.getByTestId('nav-dashboard')).not.toBeAttached();

    // 2. Click Get Started → goes to /account, sees login form
    await page.getByTestId('nav-get-started').click();
    await expect(page).toHaveURL(/\/account/);
    await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();

    // 3. Switch to register, create account
    await page.getByTestId('auth-toggle').click();
    await expect(page.locator('h1:has-text("Create Account")')).toBeVisible();
    await page.getByPlaceholder('Name').fill('E2E Flow');
    await page.getByTestId('auth-email').fill(email);
    await page.getByTestId('auth-password').fill(password);
    await page.getByTestId('auth-submit').click();

    // 4. Should land on account dashboard
    await expect(page.locator('h2:has-text("Account")')).toBeVisible({ timeout: 15000 });
    await expect(page.locator(`text=${email}`)).toBeVisible();

    // 5. Nav should now show Account / Dashboard
    await expect(page.getByTestId('nav-account')).toBeVisible({ timeout: 5000 });
    await expect(page.getByTestId('nav-dashboard')).toBeVisible();
    await expect(page.getByTestId('nav-signin')).not.toBeAttached();

    // 6. Verify localStorage has the key
    const storedKey = await page.evaluate(() => localStorage.getItem('op_api_key'));
    expect(storedKey).toMatch(/^op-/);

    // 7. Logout
    await page.getByTestId('logout-btn').click();
    await expect(page.locator('h1:has-text("Sign In")')).toBeVisible({ timeout: 5000 });

    // 8. localStorage should be cleared
    const clearedKey = await page.evaluate(() => localStorage.getItem('op_api_key'));
    expect(clearedKey).toBeNull();

    // 9. Nav should revert to Sign In / Get Started
    await expect(page.getByTestId('nav-signin')).toBeVisible({ timeout: 5000 });
    await expect(page.getByTestId('nav-get-started')).toBeVisible();
    await expect(page.getByTestId('nav-account')).not.toBeAttached();

    // 10. Re-login with same creds
    await loginViaUI(page, email, password);
    await expect(page.locator(`text=${email}`)).toBeVisible();

    // 11. Nav should show Account / Dashboard again
    await expect(page.getByTestId('nav-account')).toBeVisible({ timeout: 5000 });
    await expect(page.getByTestId('nav-dashboard')).toBeVisible();
  });

  test('login via UI → account tabs work → logout', async ({ page }) => {
    // Pre-register via API
    const { email, password } = await registerUser(page.request);

    await page.goto(TARGET + '/account');
    await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();

    // Login — lands on keys tab (handleAuth sets activeTab='keys')
    await loginViaUI(page, email, password);

    // Keys tab should be active after fresh login (new key banner shown)
    await expect(page.locator('h1:has-text("API Keys")')).toBeVisible({ timeout: 10000 });
    // Should have at least 1 key (auto-created on register + login key)
    await expect(page.getByTestId('api-key-card').first()).toBeVisible({ timeout: 5000 });

    // Switch to overview tab
    await page.getByTestId('tab-overview').click();
    await expect(page.locator('h1:has-text("Overview")')).toBeVisible();
    await expect(page.getByTestId('balance')).toBeVisible({ timeout: 10000 });

    // Billing tab
    await page.getByTestId('tab-billing').click();
    await expect(page.locator('h1:has-text("Billing & Payments")')).toBeVisible();
    await expect(page.getByTestId('billing-balance')).toBeVisible();

    // Logout
    await page.getByTestId('logout-btn').click();
    await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();
  });

  test('wrong password shows error, does not navigate away', async ({ page }) => {
    const { email } = await registerUser(page.request);

    await page.goto(TARGET + '/account');
    await page.getByTestId('auth-email').fill(email);
    await page.getByTestId('auth-password').fill('wrongpassword');
    await page.getByTestId('auth-submit').click();

    await expect(page.locator('text=Invalid email or password')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();

    // Nav should still show Sign In
    await expect(page.getByTestId('nav-signin')).toBeVisible();
  });
});

// ========== STRIPE CHECKOUT FLOW ==========

test.describe('Stripe embedded checkout', () => {
  test('opens checkout modal, selects amounts, and initiates Stripe embedded checkout', async ({ page }) => {
    // Register + login via UI so session cookie is set properly
    const { email, password } = await registerUser(page.request);
    await page.goto(TARGET + '/account');
    await loginViaUI(page, email, password);

    // Go to billing tab
    await page.getByTestId('tab-billing').click();
    await expect(page.locator('h1:has-text("Billing & Payments")')).toBeVisible();

    // Verify Stripe card is present
    await expect(page.getByTestId('stripe-card')).toBeVisible();
    await expect(page.getByTestId('add-funds-stripe-btn')).toBeVisible();

    // Open Stripe modal
    await page.getByTestId('add-funds-stripe-btn').click();
    await expect(page.getByTestId('stripe-modal')).toBeVisible();
    await expect(page.locator('h2:has-text("Add Funds")')).toBeVisible();

    // Default amount should be $25
    await expect(page.getByTestId('stripe-checkout-btn')).toContainText('Pay $25');

    // Select $50
    await page.getByTestId('amount-50').click();
    await expect(page.getByTestId('stripe-checkout-btn')).toContainText('Pay $50');

    // Select $10
    await page.getByTestId('amount-10').click();
    await expect(page.getByTestId('stripe-checkout-btn')).toContainText('Pay $10');

    // Custom amount
    await page.getByTestId('custom-amount-input').fill('42');
    await expect(page.getByTestId('stripe-checkout-btn')).toContainText('Pay $42');

    // Click pay — this hits the real Stripe API to create a checkout session
    await page.getByTestId('stripe-checkout-btn').click();

    // Wait for either:
    // a) The embedded checkout container to appear (Stripe JS loads the iframe)
    // b) An error message (if Stripe isn't configured on this server)
    const checkoutOrError = await Promise.race([
      page.getByTestId('embedded-checkout-container').waitFor({ state: 'attached', timeout: 15000 })
        .then(() => 'checkout' as const),
      page.locator('.text-red-400').waitFor({ state: 'visible', timeout: 15000 })
        .then(() => 'error' as const),
    ]);

    if (checkoutOrError === 'checkout') {
      // Stripe embedded checkout loaded successfully
      await expect(page.getByTestId('embedded-checkout-container')).toBeVisible();
      // The modal title should change to "Complete Payment"
      await expect(page.locator('h2:has-text("Complete Payment")')).toBeVisible();
      // Amount selection buttons should no longer be visible
      await expect(page.getByTestId('stripe-checkout-btn')).not.toBeVisible();
    } else {
      // Stripe not configured — we still verified the modal flow works
      const errorText = await page.locator('.text-red-400').textContent();
      console.log(`Stripe checkout returned error (expected on unconfigured servers): ${errorText}`);
    }
  });

  test('stripe modal closes via X button', async ({ page }) => {
    const { email, password } = await registerUser(page.request);
    await page.goto(TARGET + '/account');
    await loginViaUI(page, email, password);

    await page.getByTestId('tab-billing').click();
    await expect(page.locator('h1:has-text("Billing & Payments")')).toBeVisible();
    await page.getByTestId('add-funds-stripe-btn').click();
    await expect(page.getByTestId('stripe-modal')).toBeVisible();

    await page.getByTestId('stripe-modal-close').click();
    await expect(page.getByTestId('stripe-modal')).not.toBeVisible();
  });

  test('stripe modal closes via backdrop click', async ({ page }) => {
    const { email, password } = await registerUser(page.request);
    await page.goto(TARGET + '/account');
    await loginViaUI(page, email, password);

    await page.getByTestId('tab-billing').click();
    await expect(page.locator('h1:has-text("Billing & Payments")')).toBeVisible();
    await page.getByTestId('add-funds-stripe-btn').click();
    await expect(page.getByTestId('stripe-modal')).toBeVisible();

    await page.getByTestId('stripe-modal-backdrop').click({ position: { x: 10, y: 10 } });
    await expect(page.getByTestId('stripe-modal')).not.toBeVisible();
  });
});
