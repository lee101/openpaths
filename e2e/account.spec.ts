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
            { id: 'k1', name: 'Default', key_prefix: 'op-abc123' },
          ] }) });
        }
        return route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify({ key: 'op-newkey123', id: 'k2', name: 'New', prefix: 'op-newkey' }) });
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

    // ========== AUTO TOP-UP ==========

    test.describe('Auto Top-Up', () => {
      test.beforeEach(async ({ page }) => {
        await page.route('**/account/autotopup/settings', route => {
          if (route.request().method() === 'GET') {
            return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({
              enabled: false,
              threshold_cents: 50000,
              threshold_usd: 5,
              amount_cents: 100000,
              amount_usd: 10,
              has_payment_method: false,
            }) });
          }
          return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({
            enabled: true, threshold_cents: 50000, amount_cents: 100000,
          }) });
        });
        await page.route('**/account/stripe/payment-methods', route =>
          route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ payment_methods: [], default_payment_method_id: null }) })
        );
      });

      test('billing tab shows auto top-up section', async ({ page }) => {
        await page.getByTestId('tab-billing').click();
        await expect(page.locator('text=Auto Top-Up')).toBeVisible();
        await expect(page.locator('text=Payment Methods')).toBeVisible();
      });

      test('shows add card button when no cards saved', async ({ page }) => {
        await page.getByTestId('tab-billing').click();
        await expect(page.locator('button:has-text("Add Card")')).toBeVisible();
        await expect(page.locator('text=No card saved')).toBeVisible();
      });

      test('shows enable toggle when card is saved', async ({ page }) => {
        await page.route('**/account/stripe/payment-methods', route =>
          route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({
            payment_methods: [{ id: 'pm_123', brand: 'visa', last4: '4242', exp_month: 12, exp_year: 2027 }],
            default_payment_method_id: 'pm_123',
          }) })
        );
        await page.getByTestId('tab-billing').click();
        await expect(page.locator('text=VISA **** 4242')).toBeVisible();
        await expect(page.locator('text=Enable Auto Top-Up')).toBeVisible();
        await expect(page.locator('text=When balance drops below')).toBeVisible();
        await expect(page.locator('text=Top up amount')).toBeVisible();
      });

      test('displays saved card with delete button', async ({ page }) => {
        await page.route('**/account/stripe/payment-methods', route =>
          route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({
            payment_methods: [{ id: 'pm_456', brand: 'mastercard', last4: '5555', exp_month: 3, exp_year: 2028 }],
            default_payment_method_id: 'pm_456',
          }) })
        );
        await page.getByTestId('tab-billing').click();
        await expect(page.locator('text=MASTERCARD **** 5555')).toBeVisible();
      });

      test('threshold and amount dropdowns have correct options', async ({ page }) => {
        await page.route('**/account/stripe/payment-methods', route =>
          route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({
            payment_methods: [{ id: 'pm_123', brand: 'visa', last4: '4242', exp_month: 12, exp_year: 2027 }],
            default_payment_method_id: 'pm_123',
          }) })
        );
        await page.getByTestId('tab-billing').click();

        const thresholdSelect = page.locator('select').first();
        const thresholdOptions = await thresholdSelect.locator('option').allTextContents();
        expect(thresholdOptions).toEqual(['$1', '$5', '$10', '$25', '$50']);

        const amountSelect = page.locator('select').nth(1);
        const amountOptions = await amountSelect.locator('option').allTextContents();
        expect(amountOptions).toEqual(['$5', '$10', '$25', '$50', '$100']);
      });

      test('toggling auto-topup sends settings update', async ({ page }) => {
        let settingsPosted = false;
        await page.route('**/account/stripe/payment-methods', route =>
          route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({
            payment_methods: [{ id: 'pm_123', brand: 'visa', last4: '4242', exp_month: 12, exp_year: 2027 }],
            default_payment_method_id: 'pm_123',
          }) })
        );
        await page.route('**/account/autotopup/settings', route => {
          if (route.request().method() === 'POST') {
            settingsPosted = true;
            return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ enabled: true, threshold_cents: 50000, amount_cents: 100000 }) });
          }
          return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({
            enabled: false, threshold_cents: 50000, amount_cents: 100000, has_payment_method: true,
          }) });
        });

        await page.getByTestId('tab-billing').click();
        // Click the toggle
        const toggle = page.locator('button.w-12');
        await toggle.click();
        expect(settingsPosted).toBe(true);
      });

      test('shows rate limit note when enabled', async ({ page }) => {
        await page.route('**/account/stripe/payment-methods', route =>
          route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({
            payment_methods: [{ id: 'pm_123', brand: 'visa', last4: '4242', exp_month: 12, exp_year: 2027 }],
            default_payment_method_id: 'pm_123',
          }) })
        );
        await page.route('**/account/autotopup/settings', route =>
          route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({
            enabled: true, threshold_cents: 50000, amount_cents: 100000, has_payment_method: true,
          }) })
        );
        await page.getByTestId('tab-billing').click();
        await expect(page.locator('text=Max once per 60s')).toBeVisible();
      });

      test('add card button shows card form', async ({ page }) => {
        await page.getByTestId('tab-billing').click();
        await page.locator('button:has-text("Add Card")').click();
        // Card form should appear with Save and Cancel buttons
        await expect(page.locator('button:has-text("Save Card")')).toBeVisible();
        await expect(page.locator('button:has-text("Cancel")')).toBeVisible();
      });

      test('cancel button hides card form', async ({ page }) => {
        await page.getByTestId('tab-billing').click();
        await page.locator('button:has-text("Add Card")').click();
        await expect(page.locator('button:has-text("Save Card")')).toBeVisible();
        await page.locator('button:has-text("Cancel")').click();
        await expect(page.locator('button:has-text("Save Card")')).not.toBeVisible();
        await expect(page.locator('button:has-text("Add Card")')).toBeVisible();
      });

      test('delete card calls API and refreshes', async ({ page }) => {
        let deleteCalled = false;
        await page.route('**/account/stripe/payment-methods', route => {
          if (route.request().method() === 'DELETE') {
            deleteCalled = true;
            return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ message: 'removed' }) });
          }
          return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({
            payment_methods: deleteCalled ? [] : [{ id: 'pm_123', brand: 'visa', last4: '4242', exp_month: 12, exp_year: 2027 }],
            default_payment_method_id: deleteCalled ? null : 'pm_123',
          }) });
        });

        await page.getByTestId('tab-billing').click();
        await expect(page.locator('text=VISA **** 4242')).toBeVisible();
        // Click the trash icon
        await page.locator('text=VISA **** 4242').locator('..').locator('button').click();
        expect(deleteCalled).toBe(true);
      });
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
  });
});
