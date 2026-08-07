import { expect, test } from '@playwright/test';

test('dashboard JWT injection preserves an existing API key', async ({ page }) => {
  await page.addInitScript(() => {
    localStorage.setItem('op_api_key', 'op_existing_api_key');
    (window as any).userData = {
      id: 'user-1',
      email: 'user@example.com',
      name: 'User',
      secret: 'eyJhbGciOiJIUzI1NiJ9.dashboard.jwt',
      authenticated: true,
    };
  });

  await page.goto('/');
  await expect.poll(() => page.evaluate(() => localStorage.getItem('op_api_key'))).toBe('op_existing_api_key');
});

test('organization invitation page accepts a valid token', async ({ page }) => {
  let accepted = false;
  await page.route('**/account/orgs/acme/join?token=abc123', async route => {
    accepted = route.request().method() === 'POST';
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ org: { id: 'org-1', name: 'Acme', slug: 'acme', role: 'member' } }),
    });
  });

  await page.goto('/orgs/acme/join?token=abc123');
  await expect(page.getByRole('heading', { name: 'Invitation accepted' })).toBeVisible();
  await expect(page.getByText('You joined Acme.')).toBeVisible();
  expect(accepted).toBe(true);
});

test('organization invitation page prompts unauthenticated users to sign in', async ({ page }) => {
  await page.route('**/account/orgs/acme/join?token=abc123', route =>
    route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({ error: { message: 'Sign in required' } }),
    }),
  );

  await page.goto('/orgs/acme/join?token=abc123');
  await expect(page.getByRole('button', { name: 'Sign in to accept' })).toBeVisible();
  await expect(page.getByText('invited email address')).toBeVisible();
});
