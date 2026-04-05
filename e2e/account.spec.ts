import { test, expect } from '@playwright/test';

const TEST_API_KEY = process.env.TEST_API_KEY!;
const TEST_USER = {
  id: process.env.TEST_USER_ID!,
  email: process.env.TEST_USER_EMAIL!,
  name: process.env.TEST_USER_NAME!,
};

// Shared mock helpers
function mockAccountAPIs(page: any, overrides: { balance?: any; transactions?: any[] } = {}) {
  const balance = overrides.balance ?? { balance_cents: 425000, balance_usd: 42.50, balance_usd_exact: '42.5000' };
  const transactions = overrides.transactions ?? [
    { id: '1', tx_type: 'deposit', description: 'Stripe checkout cs_123 ($25.00)', amount_cents: 2500000, balance_after: 4250000, created_at: '2024-01-15T00:00:00Z' },
    { id: '2', tx_type: 'usage_deduction', description: 'Model: gpt-4o, in: 1000, out: 500', amount_cents: -5000, balance_after: 4245000, created_at: '2024-01-14T00:00:00Z' },
  ];
  page.route('**/account/balance', (route: any) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(balance) })
  );
  page.route('**/account/transactions*', (route: any) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ transactions }) })
  );
  page.route('**/account/keys', (route: any) => {
    if (route.request().method() === 'GET') {
      return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ keys: [
        { id: 'k1', name: 'Default', key_prefix: 'op-abc12345' },
      ] }) });
    }
    return route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify({ key: 'op-newkey123456', id: 'k2', name: 'New', prefix: 'op-newkey1' }) });
  });
  page.route('**/account/stripe/config', (route: any) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ publishable_key: 'pk_test_123' }) })
  );
}

test.describe('Account Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/account');
  });

  // ========== AUTH FORM DISPLAY ==========

  test('shows login form when not authenticated', async ({ page }) => {
    await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();
    await expect(page.getByTestId('auth-email')).toBeVisible();
    await expect(page.getByTestId('auth-password')).toBeVisible();
    await expect(page.getByTestId('auth-submit')).toBeVisible();
  });

  test('can toggle between login and register', async ({ page }) => {
    await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();
    await page.getByTestId('auth-toggle').click();
    await expect(page.locator('h1:has-text("Create Account")')).toBeVisible();
    await page.getByTestId('auth-toggle').click();
    await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();
  });

  test('password field has show/hide toggle', async ({ page }) => {
    const pw = page.getByTestId('auth-password');
    await expect(pw).toHaveAttribute('type', 'password');
    // Click the eye icon button (sibling of password input)
    await page.locator('button[type="button"]').first().click();
    await expect(pw).toHaveAttribute('type', 'text');
    await page.locator('button[type="button"]').first().click();
    await expect(pw).toHaveAttribute('type', 'password');
  });

  // ========== REGISTER FLOW ==========

  test.describe('Register flow', () => {
    test.beforeEach(async ({ page }) => {
      await page.route('**/auth/register', (route) =>
        route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({ token: TEST_API_KEY, api_key: TEST_API_KEY, user: TEST_USER }),
        })
      );
      mockAccountAPIs(page);
    });

    test('creates account and lands on dashboard', async ({ page }) => {
      await page.getByTestId('auth-toggle').click();
      await expect(page.locator('h1:has-text("Create Account")')).toBeVisible();

      await page.getByPlaceholder('Name').fill('Test User');
      await page.getByTestId('auth-email').fill('test@example.com');
      await page.getByTestId('auth-password').fill('password123');
      await page.getByTestId('auth-submit').click();

      // Should land on authenticated dashboard (keys tab shown first)
      await expect(page.locator('h2:has-text("Account")')).toBeVisible({ timeout: 5000 });
      await expect(page.locator('text=test@example.com')).toBeVisible();
    });

    test('register posts correct JSON body', async ({ page }) => {
      let capturedBody: any = null;
      await page.route('**/auth/register', (route) => {
        capturedBody = route.request().postDataJSON();
        return route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({ token: TEST_API_KEY, api_key: TEST_API_KEY, user: TEST_USER }),
        });
      });

      await page.getByTestId('auth-toggle').click();
      await page.getByPlaceholder('Name').fill('Test User');
      await page.getByTestId('auth-email').fill('newuser@example.com');
      await page.getByTestId('auth-password').fill('securepass');
      await page.getByTestId('auth-submit').click();

      await expect(page.locator('h2:has-text("Account")')).toBeVisible({ timeout: 5000 });
      expect(capturedBody).toMatchObject({ email: 'newuser@example.com', password: 'securepass', name: 'Test User' });
    });

    test('register stores api_key in localStorage', async ({ page }) => {
      await page.getByTestId('auth-toggle').click();
      await page.getByPlaceholder('Name').fill('Test User');
      await page.getByTestId('auth-email').fill('test@example.com');
      await page.getByTestId('auth-password').fill('password123');
      await page.getByTestId('auth-submit').click();

      await expect(page.locator('h2:has-text("Account")')).toBeVisible({ timeout: 5000 });
      const stored = await page.evaluate(() => localStorage.getItem('op_api_key'));
      expect(stored).toBe(TEST_API_KEY);
    });

    test('register shows error on duplicate email', async ({ page }) => {
      await page.route('**/auth/register', (route) =>
        route.fulfill({
          status: 409,
          contentType: 'application/json',
          body: JSON.stringify({ error: { message: 'Email already registered' } }),
        })
      );

      await page.getByTestId('auth-toggle').click();
      await page.getByPlaceholder('Name').fill('Test User');
      await page.getByTestId('auth-email').fill('existing@example.com');
      await page.getByTestId('auth-password').fill('password123');
      await page.getByTestId('auth-submit').click();

      await expect(page.locator('text=Email already registered')).toBeVisible({ timeout: 5000 });
      await expect(page.locator('h1:has-text("Create Account")')).toBeVisible();
    });
  });

  // ========== LOGIN FLOW ==========

  test.describe('Login flow', () => {
    test.beforeEach(async ({ page }) => {
      await page.route('**/auth/login', (route) =>
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ token: TEST_API_KEY, api_key: TEST_API_KEY, user: TEST_USER }),
        })
      );
      mockAccountAPIs(page);
    });

    test('signs in and lands on dashboard', async ({ page }) => {
      await page.getByTestId('auth-email').fill('test@example.com');
      await page.getByTestId('auth-password').fill('password123');
      await page.getByTestId('auth-submit').click();

      await expect(page.locator('h2:has-text("Account")')).toBeVisible({ timeout: 5000 });
      await expect(page.locator('text=test@example.com')).toBeVisible();
    });

    test('login posts correct JSON body', async ({ page }) => {
      let capturedBody: any = null;
      await page.route('**/auth/login', (route) => {
        capturedBody = route.request().postDataJSON();
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ token: TEST_API_KEY, api_key: TEST_API_KEY, user: TEST_USER }),
        });
      });

      await page.getByTestId('auth-email').fill('test@example.com');
      await page.getByTestId('auth-password').fill('mypassword');
      await page.getByTestId('auth-submit').click();

      await expect(page.locator('h2:has-text("Account")')).toBeVisible({ timeout: 5000 });
      expect(capturedBody).toMatchObject({ email: 'test@example.com', password: 'mypassword' });
    });

    test('login stores api_key in localStorage', async ({ page }) => {
      await page.getByTestId('auth-email').fill('test@example.com');
      await page.getByTestId('auth-password').fill('password123');
      await page.getByTestId('auth-submit').click();

      await expect(page.locator('h2:has-text("Account")')).toBeVisible({ timeout: 5000 });
      const stored = await page.evaluate(() => localStorage.getItem('op_api_key'));
      expect(stored).toBe(TEST_API_KEY);
    });

    test('login shows error on wrong password', async ({ page }) => {
      await page.route('**/auth/login', (route) =>
        route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({ error: { message: 'Invalid email or password' } }),
        })
      );

      await page.getByTestId('auth-email').fill('test@example.com');
      await page.getByTestId('auth-password').fill('wrongpass');
      await page.getByTestId('auth-submit').click();

      await expect(page.locator('text=Invalid email or password')).toBeVisible({ timeout: 5000 });
      await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();
    });

    test('re-login after logout works', async ({ page }) => {
      // First login
      await page.getByTestId('auth-email').fill('test@example.com');
      await page.getByTestId('auth-password').fill('password123');
      await page.getByTestId('auth-submit').click();
      await expect(page.locator('h2:has-text("Account")')).toBeVisible({ timeout: 5000 });

      // Logout
      await page.getByTestId('logout-btn').click();
      await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();

      // Verify localStorage is cleared
      const key = await page.evaluate(() => localStorage.getItem('op_api_key'));
      expect(key).toBeNull();

      // Re-login
      await page.getByTestId('auth-email').fill('test@example.com');
      await page.getByTestId('auth-password').fill('password123');
      await page.getByTestId('auth-submit').click();
      await expect(page.locator('h2:has-text("Account")')).toBeVisible({ timeout: 5000 });
    });
  });

  // ========== AUTHENTICATED DASHBOARD ==========

  test.describe('Authenticated', () => {
    test.beforeEach(async ({ page }) => {
      await page.addInitScript(({ key, user }: { key: string; user: any }) => {
        localStorage.setItem('op_api_key', key);
        localStorage.setItem('op_user', JSON.stringify(user));
      }, { key: TEST_API_KEY, user: TEST_USER });

      mockAccountAPIs(page);
      await page.goto('/account');
    });

    test('page loads with sidebar and overview tab active', async ({ page }) => {
      await expect(page.locator('h2:has-text("Account")')).toBeVisible();
      await expect(page.locator('text=test@example.com')).toBeVisible();
      await expect(page.locator('h1:has-text("Overview")')).toBeVisible();
    });

    test('sidebar has all three tabs', async ({ page }) => {
      await expect(page.getByTestId('tab-overview')).toBeVisible();
      await expect(page.getByTestId('tab-keys')).toBeVisible();
      await expect(page.getByTestId('tab-billing')).toBeVisible();
    });

    test('overview shows balance from API', async ({ page }) => {
      await expect(page.getByTestId('balance')).toContainText('$42.50');
    });

    test('overview preserves sub-cent balance precision', async ({ page }) => {
      await page.unroute('**/account/balance');
      await page.route('**/account/balance', (route: any) =>
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ balance_cents: 99999, balance_usd: 9.9999, balance_usd_exact: '9.9999' }),
        })
      );

      await page.reload();

      await expect(page.getByTestId('balance')).toContainText('$9.9999');
      await page.getByTestId('tab-billing').click();
      await expect(page.getByTestId('billing-balance')).toContainText('$9.9999');
    });

    test('overview shows transactions from API', async ({ page }) => {
      const table = page.getByTestId('activity-table');
      await expect(table).toBeVisible();
      await expect(table.locator('text=deposit')).toBeVisible();
      await expect(table.locator('text=usage_deduction')).toBeVisible();
    });

    test('Add Funds button switches to billing tab', async ({ page }) => {
      await page.click('button:has-text("Add Funds")');
      await expect(page.locator('h1:has-text("Billing & Payments")')).toBeVisible();
    });

    test('keys tab renders with API keys from backend', async ({ page }) => {
      await page.getByTestId('tab-keys').click();
      await expect(page.locator('h1:has-text("API Keys")')).toBeVisible();
      await expect(page.getByTestId('api-key-card')).toBeVisible();
      await expect(page.getByTestId('key-status')).toContainText('Active');
    });

    test('keys tab shows create key button', async ({ page }) => {
      await page.getByTestId('tab-keys').click();
      await expect(page.getByTestId('create-key-btn')).toBeVisible();
      await expect(page.getByTestId('create-key-btn')).toContainText('Create Key');
    });

    test('keys tab shows security warning', async ({ page }) => {
      await page.getByTestId('tab-keys').click();
      await expect(page.locator('text=Do not share your API key')).toBeVisible();
    });

    test('billing tab renders', async ({ page }) => {
      await page.getByTestId('tab-billing').click();
      await expect(page.locator('h1:has-text("Billing & Payments")')).toBeVisible();
    });

    test('billing tab shows balance', async ({ page }) => {
      await page.getByTestId('tab-billing').click();
      await expect(page.getByTestId('billing-balance')).toContainText('$42.50');
    });

    test('billing tab shows Stripe and Solana payment cards', async ({ page }) => {
      await page.getByTestId('tab-billing').click();
      await expect(page.getByTestId('stripe-card')).toBeVisible();
      await expect(page.getByTestId('solana-card')).toBeVisible();
    });

    test('billing tab shows transaction history', async ({ page }) => {
      await page.getByTestId('tab-billing').click();
      const table = page.getByTestId('payment-history-table');
      await expect(table).toBeVisible();
      await expect(table.locator('text=deposit')).toBeVisible();
    });

    test('Add Funds with Stripe opens checkout modal', async ({ page }) => {
      await page.getByTestId('tab-billing').click();
      await page.getByTestId('add-funds-stripe-btn').click();
      await expect(page.getByTestId('stripe-modal')).toBeVisible();
      await expect(page.locator('h2:has-text("Add Funds")')).toBeVisible();
    });

    test('stripe modal shows amount selection buttons', async ({ page }) => {
      await page.getByTestId('tab-billing').click();
      await page.getByTestId('add-funds-stripe-btn').click();
      for (const amount of [10, 25, 50, 100]) {
        await expect(page.getByTestId(`amount-${amount}`)).toBeVisible();
      }
    });

    test('stripe modal amount selection updates checkout button', async ({ page }) => {
      await page.getByTestId('tab-billing').click();
      await page.getByTestId('add-funds-stripe-btn').click();
      await expect(page.getByTestId('stripe-checkout-btn')).toContainText('Pay $25');
      await page.getByTestId('amount-100').click();
      await expect(page.getByTestId('stripe-checkout-btn')).toContainText('Pay $100');
      await page.getByTestId('amount-10').click();
      await expect(page.getByTestId('stripe-checkout-btn')).toContainText('Pay $10');
    });

    test('stripe modal custom amount input works', async ({ page }) => {
      await page.getByTestId('tab-billing').click();
      await page.getByTestId('add-funds-stripe-btn').click();
      const input = page.getByTestId('custom-amount-input');
      await input.fill('75');
      await expect(page.getByTestId('stripe-checkout-btn')).toContainText('Pay $75');
    });

    test('stripe modal closes on X button', async ({ page }) => {
      await page.getByTestId('tab-billing').click();
      await page.getByTestId('add-funds-stripe-btn').click();
      await expect(page.getByTestId('stripe-modal')).toBeVisible();
      await page.getByTestId('stripe-modal-close').click();
      await expect(page.getByTestId('stripe-modal')).not.toBeVisible();
    });

    test('stripe modal closes on backdrop click', async ({ page }) => {
      await page.getByTestId('tab-billing').click();
      await page.getByTestId('add-funds-stripe-btn').click();
      await expect(page.getByTestId('stripe-modal')).toBeVisible();
      await page.getByTestId('stripe-modal-backdrop').click({ position: { x: 10, y: 10 } });
      await expect(page.getByTestId('stripe-modal')).not.toBeVisible();
    });

    test('stripe checkout calls backend and transitions to embedded checkout', async ({ page }) => {
      let checkoutCalled = false;
      await page.route('**/account/stripe/checkout', (route) => {
        checkoutCalled = true;
        return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ client_secret: 'cs_test_secret_123' }) });
      });

      await page.getByTestId('tab-billing').click();
      await page.getByTestId('add-funds-stripe-btn').click();
      await page.getByTestId('stripe-checkout-btn').click();

      await expect(page.getByTestId('embedded-checkout-container')).toBeAttached({ timeout: 5000 });
      await expect(page.getByTestId('stripe-checkout-btn')).not.toBeVisible();
      expect(checkoutCalled).toBe(true);
    });

    test('stripe checkout sends Authorization header with API key', async ({ page }) => {
      let capturedAuthHeader = '';
      await page.route('**/account/stripe/checkout', (route) => {
        capturedAuthHeader = route.request().headers()['authorization'] || '';
        return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ client_secret: 'cs_test_secret_456' }) });
      });

      await page.getByTestId('tab-billing').click();
      await page.getByTestId('add-funds-stripe-btn').click();
      await page.getByTestId('stripe-checkout-btn').click();

      await expect(page.getByTestId('embedded-checkout-container')).toBeAttached({ timeout: 5000 });
      expect(capturedAuthHeader).toBe(`Bearer ${TEST_API_KEY}`);
    });

    test('stripe checkout sends correct amount_usd in request body', async ({ page }) => {
      let capturedBody: any = null;
      await page.route('**/account/stripe/checkout', (route) => {
        capturedBody = route.request().postDataJSON();
        return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ client_secret: 'cs_test_secret_789' }) });
      });

      await page.getByTestId('tab-billing').click();
      await page.getByTestId('add-funds-stripe-btn').click();
      await page.getByTestId('amount-50').click();
      await page.getByTestId('stripe-checkout-btn').click();

      await expect(page.getByTestId('embedded-checkout-container')).toBeAttached({ timeout: 5000 });
      expect(capturedBody).toEqual({ amount_usd: 50 });
    });

    test('stripe checkout shows error on API failure', async ({ page }) => {
      await page.route('**/account/stripe/checkout', (route) =>
        route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: { message: 'Stripe is down' } }) })
      );

      await page.getByTestId('tab-billing').click();
      await page.getByTestId('add-funds-stripe-btn').click();
      await page.getByTestId('stripe-checkout-btn').click();

      await expect(page.locator('text=Stripe is down')).toBeVisible({ timeout: 5000 });
      await expect(page.getByTestId('stripe-checkout-btn')).toBeVisible();
    });

    test('tab switching preserves correct content', async ({ page }) => {
      await expect(page.locator('h1:has-text("Overview")')).toBeVisible();
      await page.getByTestId('tab-keys').click();
      await expect(page.locator('h1:has-text("API Keys")')).toBeVisible();
      await expect(page.locator('h1:has-text("Overview")')).not.toBeVisible();
      await page.getByTestId('tab-billing').click();
      await expect(page.locator('h1:has-text("Billing & Payments")')).toBeVisible();
      await expect(page.locator('h1:has-text("API Keys")')).not.toBeVisible();
      await page.getByTestId('tab-overview').click();
      await expect(page.locator('h1:has-text("Overview")')).toBeVisible();
    });

    test('Connect Wallet button is visible on billing tab', async ({ page }) => {
      await page.getByTestId('tab-billing').click();
      await expect(page.getByTestId('connect-wallet-btn')).toBeVisible();
      await expect(page.getByTestId('connect-wallet-btn')).toContainText('Connect Wallet');
    });

    test('logout clears auth and shows login form', async ({ page }) => {
      await expect(page.getByTestId('logout-btn')).toBeVisible();
      await page.getByTestId('logout-btn').click();
      await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();
      const key = await page.evaluate(() => localStorage.getItem('op_api_key'));
      expect(key).toBeNull();
    });

    test('401 response clears localStorage and triggers page reload', async ({ page }) => {
      await page.route('**/account/stripe/checkout', (route) =>
        route.fulfill({ status: 401, contentType: 'application/json', body: JSON.stringify({ error: { message: 'Invalid key', code: 'invalid_api_key' } }) })
      );

      await page.getByTestId('tab-billing').click();
      await page.getByTestId('add-funds-stripe-btn').click();
      await page.getByTestId('stripe-checkout-btn').click();

      await page.waitForURL('**/account', { timeout: 10000 });
    });
  });
});
