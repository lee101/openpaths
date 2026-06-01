/**
 * Real server e2e tests — hit the actual Go backend at port 8090.
 * Run with: npx playwright test e2e/auth-real.spec.ts --config=playwright.real.config.ts
 */
import { test, expect } from '@playwright/test';

function uniqueEmail() {
  return `e2e-${Date.now()}-${Math.random().toString(36).slice(2)}@test.com`;
}

const BASE = 'http://localhost:8090';

test.describe('Real server auth', () => {
  // ========== REGISTER ==========

  test('register creates account and returns api_key', async ({ request }) => {
    const email = uniqueEmail();
    const res = await request.post(`${BASE}/auth/register`, {
      data: { email, password: 'password123', name: 'E2E Test' },
    });
    expect(res.status()).toBe(201);
    const body = await res.json();
    expect(body.api_key).toMatch(/^op-/);
    expect(body.token).toBe(body.api_key);
    expect(body.user.email).toBe(email);
  });

  test('register rejects duplicate email', async ({ request }) => {
    const email = uniqueEmail();
    await request.post(`${BASE}/auth/register`, {
      data: { email, password: 'password123', name: 'First' },
    });
    const res = await request.post(`${BASE}/auth/register`, {
      data: { email, password: 'password456', name: 'Second' },
    });
    expect(res.status()).toBe(409);
    const body = await res.json();
    expect(body.error.message).toMatch(/already registered/i);
  });

  test('register rejects short password', async ({ request }) => {
    const res = await request.post(`${BASE}/auth/register`, {
      data: { email: uniqueEmail(), password: 'short', name: 'Test' },
    });
    expect(res.status()).toBe(400);
  });

  test('register rejects missing email', async ({ request }) => {
    const res = await request.post(`${BASE}/auth/register`, {
      data: { password: 'password123' },
    });
    expect(res.status()).toBe(400);
  });

  // ========== LOGIN ==========

  test('login returns JWT when user already has API keys', async ({ request }) => {
    const email = uniqueEmail();
    await request.post(`${BASE}/auth/register`, {
      data: { email, password: 'mypassword', name: 'Login Test' },
    });

    const res = await request.post(`${BASE}/auth/login`, {
      data: { email, password: 'mypassword' },
    });
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.api_key).toBeUndefined();
    expect(body.token).toMatch(/^eyJ/);
    expect(body.user.email).toBe(email);
  });

  test('login rejects wrong password', async ({ request }) => {
    const email = uniqueEmail();
    await request.post(`${BASE}/auth/register`, {
      data: { email, password: 'rightpassword', name: 'Test' },
    });

    const res = await request.post(`${BASE}/auth/login`, {
      data: { email, password: 'wrongpassword' },
    });
    expect(res.status()).toBe(401);
    const body = await res.json();
    expect(body.error.message).toMatch(/invalid/i);
  });

  test('login rejects unknown email', async ({ request }) => {
    const res = await request.post(`${BASE}/auth/login`, {
      data: { email: 'nobody@nowhere.invalid', password: 'password123' },
    });
    expect(res.status()).toBe(401);
  });

  // ========== API KEY WORKS FOR ACCOUNT ROUTES ==========

  test('api key from register grants access to balance', async ({ request }) => {
    const email = uniqueEmail();
    const reg = await request.post(`${BASE}/auth/register`, {
      data: { email, password: 'password123', name: 'Test' },
    });
    const { api_key } = await reg.json();

    const res = await request.get(`${BASE}/account/balance`, {
      headers: { Authorization: `Bearer ${api_key}` },
    });
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body).toHaveProperty('balance_usd');
  });

  test('login JWT grants access to balance', async ({ request }) => {
    const email = uniqueEmail();
    await request.post(`${BASE}/auth/register`, {
      data: { email, password: 'password123', name: 'Test' },
    });
    const login = await request.post(`${BASE}/auth/login`, {
      data: { email, password: 'password123' },
    });
    const { token } = await login.json();

    const res = await request.get(`${BASE}/account/balance`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body).toHaveProperty('balance_usd');
  });

  test('login JWT grants access to account keys list', async ({ request }) => {
    const email = uniqueEmail();
    await request.post(`${BASE}/auth/register`, {
      data: { email, password: 'password123', name: 'Test' },
    });
    const login = await request.post(`${BASE}/auth/login`, {
      data: { email, password: 'password123' },
    });
    const { token } = await login.json();

    const res = await request.get(`${BASE}/account/keys`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(Array.isArray(body.keys)).toBe(true);
    expect(body.keys.length).toBe(1);
  });

  test('invalid api key returns 401', async ({ request }) => {
    const res = await request.get(`${BASE}/account/balance`, {
      headers: { Authorization: 'Bearer op-fakekeynotreal' },
    });
    expect(res.status()).toBe(401);
  });

  test('no auth header returns 401', async ({ request }) => {
    const res = await request.get(`${BASE}/account/balance`);
    expect(res.status()).toBe(401);
  });

  // ========== UI FLOW ==========

  test('register via UI flow stores api_key and shows dashboard', async ({ page }) => {
    const email = uniqueEmail();

    // Mock nothing — hit real server via vite proxy at port 3099
    await page.goto('http://localhost:3099/account');
    await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();

    await page.getByTestId('auth-toggle').click();
    await expect(page.locator('h1:has-text("Create Account")')).toBeVisible();

    await page.getByPlaceholder('Name').fill('E2E User');
    await page.getByTestId('auth-email').fill(email);
    await page.getByTestId('auth-password').fill('password123');
    await page.getByTestId('auth-submit').click();

    await expect(page.locator('h2:has-text("Account")')).toBeVisible({ timeout: 10000 });

    const storedKey = await page.evaluate(() => localStorage.getItem('op_api_key'));
    expect(storedKey).toMatch(/^op-/);
  });

  test('login via UI flow stores session token and shows dashboard', async ({ page }) => {
    // Register via API first
    const email = uniqueEmail();
    await page.request.post(`${BASE}/auth/register`, {
      data: { email, password: 'password123', name: 'E2E User' },
    });

    await page.goto('http://localhost:3099/account');
    await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();

    await page.getByTestId('auth-email').fill(email);
    await page.getByTestId('auth-password').fill('password123');
    await page.getByTestId('auth-submit').click();

    await expect(page.locator('h2:has-text("Account")')).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId('new-key-banner')).toHaveCount(0);

    const storedKey = await page.evaluate(() => localStorage.getItem('op_api_key'));
    expect(storedKey).toMatch(/^eyJ/);
  });

  test('wrong password via UI shows error without redirect', async ({ page }) => {
    const email = uniqueEmail();
    await page.request.post(`${BASE}/auth/register`, {
      data: { email, password: 'password123', name: 'Test' },
    });

    await page.goto('http://localhost:3099/account');
    await page.getByTestId('auth-email').fill(email);
    await page.getByTestId('auth-password').fill('wrongpassword');
    await page.getByTestId('auth-submit').click();

    await expect(page.locator('text=Invalid email or password')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();
  });

  test('re-login via UI after logout works', async ({ page }) => {
    const email = uniqueEmail();
    await page.request.post(`${BASE}/auth/register`, {
      data: { email, password: 'password123', name: 'Test' },
    });

    await page.goto('http://localhost:3099/account');
    await page.getByTestId('auth-email').fill(email);
    await page.getByTestId('auth-password').fill('password123');
    await page.getByTestId('auth-submit').click();
    await expect(page.locator('h2:has-text("Account")')).toBeVisible({ timeout: 10000 });

    await page.getByTestId('logout-btn').click();
    await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();

    // Re-login
    await page.getByTestId('auth-email').fill(email);
    await page.getByTestId('auth-password').fill('password123');
    await page.getByTestId('auth-submit').click();
    await expect(page.locator('h2:has-text("Account")')).toBeVisible({ timeout: 10000 });
  });
});
