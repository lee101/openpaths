import { test, expect } from '@playwright/test';

test.describe('Account Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/account');
  });

  test('page loads with sidebar and overview tab active', async ({ page }) => {
    await expect(page.locator('h2:has-text("Account")')).toBeVisible();
    await expect(page.locator('text=user@example.com')).toBeVisible();
    await expect(page.locator('h1:has-text("Overview")')).toBeVisible();
  });

  test('sidebar has all three tabs', async ({ page }) => {
    await expect(page.getByTestId('tab-overview')).toBeVisible();
    await expect(page.getByTestId('tab-keys')).toBeVisible();
    await expect(page.getByTestId('tab-billing')).toBeVisible();
  });

  // ========== OVERVIEW TAB ==========

  test('overview shows balance and request metrics', async ({ page }) => {
    await expect(page.getByTestId('balance')).toContainText('$42.50');
    await expect(page.getByTestId('requests-count')).toContainText('1.2M');
    await expect(page.locator('text=+14% from last month')).toBeVisible();
  });

  test('overview shows usage graph over time', async ({ page }) => {
    const graph = page.getByTestId('usage-graph');
    await expect(graph).toBeVisible();
    await expect(graph.locator('text=Usage Over Time')).toBeVisible();
    await expect(graph.locator('text=Last 9 weeks')).toBeVisible();
    // SVG chart renders
    await expect(graph.locator('svg[role="img"]')).toBeVisible();
    // data points render (9 circles for 9 weeks)
    const circles = graph.locator('svg circle');
    await expect(circles).toHaveCount(9);
  });

  test('usage graph shows tooltip on hover', async ({ page }) => {
    const graph = page.getByTestId('usage-graph');
    const circles = graph.locator('svg circle');
    // hover over a data point
    await circles.nth(4).hover();
    // tooltip with request count should appear
    await expect(graph.locator('svg text:has-text("req")')).toBeVisible();
  });

  test('usage graph has axis labels', async ({ page }) => {
    const graph = page.getByTestId('usage-graph');
    // Y-axis labels (k values)
    await expect(graph.locator('svg text:has-text("k")')).toHaveCount(5);
    // X-axis date labels
    await expect(graph.locator('svg text:has-text("Jan")')).toHaveCount(5);
    await expect(graph.locator('svg text:has-text("Feb")')).toHaveCount(4);
  });

  test('overview shows recent activity table', async ({ page }) => {
    const table = page.getByTestId('activity-table');
    await expect(table).toBeVisible();
    await expect(table.locator('th:has-text("Model")')).toBeVisible();
    await expect(table.locator('th:has-text("Requests")')).toBeVisible();
    await expect(table.locator('th:has-text("Cost")')).toBeVisible();
    await expect(table.locator('text=anthropic/claude-3.5-sonnet')).toBeVisible();
    await expect(table.locator('text=openai/gpt-4o')).toBeVisible();
    await expect(table.locator('text=meta-llama/llama-3.1-70b-instruct')).toBeVisible();
  });

  test('Add Funds button switches to billing tab', async ({ page }) => {
    await page.click('button:has-text("Add Funds")');
    await expect(page.locator('h1:has-text("Billing & Payments")')).toBeVisible();
  });

  // ========== KEYS TAB ==========

  test('keys tab renders', async ({ page }) => {
    await page.getByTestId('tab-keys').click();
    await expect(page.locator('h1:has-text("API Keys")')).toBeVisible();
  });

  test('keys tab shows create key button', async ({ page }) => {
    await page.getByTestId('tab-keys').click();
    await expect(page.getByTestId('create-key-btn')).toBeVisible();
    await expect(page.getByTestId('create-key-btn')).toContainText('Create Key');
  });

  test('keys tab shows API key with active status', async ({ page }) => {
    await page.getByTestId('tab-keys').click();
    const card = page.getByTestId('api-key-card');
    await expect(card).toBeVisible();
    await expect(card.locator('text=Default Project Key')).toBeVisible();
    await expect(page.getByTestId('key-status')).toContainText('Active');
    await expect(page.getByTestId('api-key-value')).toContainText('op_live_');
  });

  test('copy key button works', async ({ page, context }) => {
    await context.grantPermissions(['clipboard-read', 'clipboard-write']);
    await page.getByTestId('tab-keys').click();
    await page.getByTestId('copy-key-btn').click();
    // check icon changes to checkmark (green)
    await expect(page.locator('[data-testid="copy-key-btn"] svg.text-green-400')).toBeVisible();
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

  test('billing tab shows Stripe and Solana payment cards', async ({ page }) => {
    await page.getByTestId('tab-billing').click();
    await expect(page.getByTestId('stripe-card')).toBeVisible();
    await expect(page.getByTestId('solana-card')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Credit Card' })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Solana Payment' })).toBeVisible();
  });

  test('billing tab shows payment history', async ({ page }) => {
    await page.getByTestId('tab-billing').click();
    const table = page.getByTestId('payment-history-table');
    await expect(table).toBeVisible();
    await expect(table.locator('text=Solana (USDC)')).toBeVisible();
    await expect(table.locator('text=Stripe (...4242)')).toBeVisible();
    await expect(table.locator('text=$50.00')).toBeVisible();
    await expect(table.locator('text=$25.00')).toBeVisible();
    await expect(table.locator('text=Completed').first()).toBeVisible();
  });

  // ========== STRIPE PORTAL LINK ==========

  test('Manage Billing button opens Stripe portal in new tab', async ({ page, context }) => {
    await page.getByTestId('tab-billing').click();
    const portalBtn = page.getByTestId('stripe-portal-link');
    await expect(portalBtn).toBeVisible();
    await expect(portalBtn).toContainText('Manage Billing');

    const [newPage] = await Promise.all([
      context.waitForEvent('page'),
      portalBtn.click(),
    ]);
    expect(newPage.url()).toContain('billing.stripe.com');
    await newPage.close();
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

    // default is $25
    await expect(page.getByTestId('stripe-checkout-btn')).toContainText('Pay $25');

    // select $100
    await page.getByTestId('amount-100').click();
    await expect(page.getByTestId('stripe-checkout-btn')).toContainText('Pay $100');

    // select $10
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

    // click the backdrop (outside the modal)
    await page.getByTestId('stripe-modal-backdrop').click({ position: { x: 10, y: 10 } });
    await expect(page.getByTestId('stripe-modal')).not.toBeVisible();
  });

  test('stripe checkout button opens Stripe in new tab', async ({ page, context }) => {
    await page.getByTestId('tab-billing').click();
    await page.getByTestId('add-funds-stripe-btn').click();

    const [newPage] = await Promise.all([
      context.waitForEvent('page'),
      page.getByTestId('stripe-checkout-btn').click(),
    ]);
    expect(newPage.url()).toContain('stripe.com');
    await newPage.close();
  });

  // ========== TAB SWITCHING ==========

  test('tab switching preserves correct content', async ({ page }) => {
    // overview is default
    await expect(page.locator('h1:has-text("Overview")')).toBeVisible();

    // switch to keys
    await page.getByTestId('tab-keys').click();
    await expect(page.locator('h1:has-text("API Keys")')).toBeVisible();
    await expect(page.locator('h1:has-text("Overview")')).not.toBeVisible();

    // switch to billing
    await page.getByTestId('tab-billing').click();
    await expect(page.locator('h1:has-text("Billing & Payments")')).toBeVisible();
    await expect(page.locator('h1:has-text("API Keys")')).not.toBeVisible();

    // back to overview
    await page.getByTestId('tab-overview').click();
    await expect(page.locator('h1:has-text("Overview")')).toBeVisible();
  });

  test('Connect Wallet button is visible on billing tab', async ({ page }) => {
    await page.getByTestId('tab-billing').click();
    await expect(page.getByTestId('connect-wallet-btn')).toBeVisible();
    await expect(page.getByTestId('connect-wallet-btn')).toContainText('Connect Wallet');
  });
});
