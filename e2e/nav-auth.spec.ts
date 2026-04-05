import { test, expect } from '@playwright/test';

const TEST_API_KEY = process.env.TEST_API_KEY!;
const TEST_USER = {
  id: process.env.TEST_USER_ID!,
  email: process.env.TEST_USER_EMAIL!,
  name: process.env.TEST_USER_NAME!,
};

test.describe('Nav links when NOT logged in', () => {
  test('shows Sign In and Get Started, not Account or Dashboard', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByTestId('nav-signin')).toBeVisible();
    await expect(page.getByTestId('nav-get-started')).toBeVisible();
    await expect(page.getByTestId('nav-account')).not.toBeAttached();
    await expect(page.getByTestId('nav-dashboard')).not.toBeAttached();
  });

  test('Sign In link goes to /account', async ({ page }) => {
    await page.goto('/');
    await page.getByTestId('nav-get-started').click();
    await expect(page).toHaveURL(/\/account/);
  });
});

test.describe('Nav links when logged in', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(({ key, user }: { key: string; user: any }) => {
      localStorage.setItem('op_api_key', key);
      localStorage.setItem('op_user', JSON.stringify(user));
    }, { key: TEST_API_KEY, user: TEST_USER });
  });

  test('shows Account and Dashboard, not Sign In or Get Started', async ({ page }) => {
    // Mock account APIs so the page doesn't error
    page.route('**/account/balance', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ balance_usd: 0 }) })
    );
    page.route('**/account/transactions*', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ transactions: [] }) })
    );
    page.route('**/account/keys', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ keys: [] }) })
    );
    page.route('**/account/stripe/config', (route) =>
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({}) })
    );

    await page.goto('/');
    await expect(page.getByTestId('nav-account')).toBeVisible();
    await expect(page.getByTestId('nav-dashboard')).toBeVisible();
    await expect(page.getByTestId('nav-signin')).not.toBeAttached();
    await expect(page.getByTestId('nav-get-started')).not.toBeAttached();
  });
});
