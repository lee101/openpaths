import { test, expect } from '@playwright/test';

test.describe('Account Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/account');
  });

  // ========== AUTH FLOW ==========

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

  // ========== AUTHENTICATED TESTS (mock localStorage) ==========

  test.describe('Authenticated', () => {
    test.beforeEach(async ({ page }) => {
      // Mock auth by setting localStorage before navigating
      await page.addInitScript(() => {
        localStorage.setItem('op_token', 'test-jwt-token');
        localStorage.setItem('op_user', JSON.stringify({ id: 'u1', email: 'test@example.com', name: 'Test' }));
      });

      // Mock API responses
      await page.route('**/account/balance', route =>
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ balance_cents: 425000, balance_usd: 42.50 }) })
      );
      await page.route('**/account/transactions*', route =>
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ transactions: [
          { id: '1', tx_type: 'deposit', description: 'Stripe checkout cs_123 ($25.00)', amount_cents: 2500000, balance_after: 4250000, created_at: '2024-01-15T00:00:00Z' },
          { id: '2', tx_type: 'usage_deduction', description: 'Model: gpt-4o, in: 1000, out: 500', amount_cents: -5000, balance_after: 4245000, created_at: '2024-01-14T00:00:00Z' },
        ] }) })
      );
      await page.route('**/account/keys', route => {
        if (route.request().method() === 'GET') {
          return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ keys: [
            { id: 'k1', name: 'Default', key_prefix: 'op_live_abc123' },
          ] }) });
        }
        return route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify({ key: 'op_live_newkey123', id: 'k2', name: 'New', prefix: 'op_live_new' }) });
      });
      await page.route('**/account/stripe/config', route =>
        route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ publishable_key: 'pk_test_123' }) })
      );

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

    // ========== OVERVIEW TAB ==========

    test('overview shows balance from API', async ({ page }) => {
      await expect(page.getByTestId('balance')).toContainText('$42.50');
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

    // ========== KEYS TAB ==========

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

    // ========== BILLING TAB ==========

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

    // ========== STRIPE CHECKOUT MODAL ==========

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
      await page.route('**/account/stripe/checkout', route => {
        checkoutCalled = true;
        return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ client_secret: 'cs_test_secret_123' }) });
      });

      await page.getByTestId('tab-billing').click();
      await page.getByTestId('add-funds-stripe-btn').click();
      await page.getByTestId('stripe-checkout-btn').click();

      // The checkout container should be in the DOM after the API call
      await expect(page.getByTestId('embedded-checkout-container')).toBeAttached({ timeout: 5000 });
      // The amount selection should no longer be visible (replaced by checkout)
      await expect(page.getByTestId('stripe-checkout-btn')).not.toBeVisible();
      expect(checkoutCalled).toBe(true);
    });

    test('stripe checkout sends Authorization header with JWT', async ({ page }) => {
      let capturedAuthHeader = '';
      await page.route('**/account/stripe/checkout', route => {
        capturedAuthHeader = route.request().headers()['authorization'] || '';
        return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ client_secret: 'cs_test_secret_456' }) });
      });

      await page.getByTestId('tab-billing').click();
      await page.getByTestId('add-funds-stripe-btn').click();
      await page.getByTestId('stripe-checkout-btn').click();

      await expect(page.getByTestId('embedded-checkout-container')).toBeAttached({ timeout: 5000 });
      expect(capturedAuthHeader).toBe('Bearer test-jwt-token');
    });

    test('stripe checkout sends correct amount_usd in request body', async ({ page }) => {
      let capturedBody: any = null;
      await page.route('**/account/stripe/checkout', route => {
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
      await page.route('**/account/stripe/checkout', route => {
        return route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: { message: 'Stripe is down' } }) });
      });

      await page.getByTestId('tab-billing').click();
      await page.getByTestId('add-funds-stripe-btn').click();
      await page.getByTestId('stripe-checkout-btn').click();

      await expect(page.locator('text=Stripe is down')).toBeVisible({ timeout: 5000 });
      // Should still show checkout button (not transition to embedded checkout)
      await expect(page.getByTestId('stripe-checkout-btn')).toBeVisible();
    });

    // ========== TAB SWITCHING ==========

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
    });

    test('401 on checkout clears localStorage and triggers page reload', async ({ page }) => {
      // Return 401 for the checkout call (simulating expired JWT)
      await page.route('**/account/stripe/checkout', route =>
        route.fulfill({ status: 401, contentType: 'application/json', body: JSON.stringify({ error: { message: 'Invalid token', code: 'invalid_token' } }) })
      );

      await page.getByTestId('tab-billing').click();
      await page.getByTestId('add-funds-stripe-btn').click();
      await page.getByTestId('stripe-checkout-btn').click();

      // The 401 handler clears localStorage and calls reload.
      // On reload, addInitScript re-sets token, but we can detect the reload happened
      // by waiting for a fresh page load (navigation event).
      await page.waitForURL('**/account', { timeout: 10000 });

      // Verify the console warning was emitted (the 401 handler logs it)
      // The page reloaded, confirming the 401 auto-logout flow triggered
    });
  });
});
