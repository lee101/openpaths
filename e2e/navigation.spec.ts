import { test, expect } from '@playwright/test';

test.describe('Global Navigation', () => {
  test('navbar links render and navigate correctly', async ({ page }) => {
    await page.goto('/');
    const nav = page.locator('nav');
    await expect(nav).toBeVisible();
    await expect(nav.locator('text=OpenPath')).toBeVisible();
    await expect(nav.locator('text=Models')).toBeVisible();
    await expect(nav.locator('text=Apps')).toBeVisible();
    await expect(nav.locator('text=Integrations')).toBeVisible();
    await expect(nav.locator('text=Playground')).toBeVisible();
    await expect(nav.getByTestId('nav-get-started').or(nav.getByTestId('nav-dashboard'))).toBeVisible();
  });

  test('navigate to /models', async ({ page }) => {
    await page.goto('/');
    await page.click('nav >> text=Models');
    await expect(page).toHaveURL('/models');
    await expect(page.locator('h1')).toContainText('Model Directory');
  });

  test('navigate to /playground', async ({ page }) => {
    await page.goto('/');
    await page.click('nav >> text=Playground');
    await expect(page).toHaveURL('/playground');
  });

  test('navigate to /integrations', async ({ page }) => {
    await page.goto('/');
    await page.click('nav >> text=Integrations');
    await expect(page).toHaveURL('/integrations');
    await expect(page.locator('h1')).toContainText('Integrate OpenPaths');
  });

  test('navigate to /apps', async ({ page }) => {
    await page.route('**/stats/apps?*', route => route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ period: '30d', limit: 100, apps: [] }),
    }));

    await page.goto('/');
    await page.click('nav >> text=Apps');
    await expect(page).toHaveURL('/apps/');
    await expect(page.locator('h1')).toContainText('Apps And Agents');
  });

  test('navigate to /account via primary account button', async ({ page }) => {
    await page.goto('/');
    const primaryAccountButton = page.getByTestId('nav-get-started').or(page.getByTestId('nav-dashboard'));
    await primaryAccountButton.click();
    await expect(page).toHaveURL('/account');
    // Shows login form when not authenticated
    await expect(page.locator('h1:has-text("Sign In")')).toBeVisible();
  });

  test('logo navigates home', async ({ page }) => {
    await page.goto('/models');
    await page.click('nav >> a:has-text("OpenPath")');
    await expect(page).toHaveURL('/');
  });

  test('footer renders', async ({ page }) => {
    await page.goto('/');
    const footer = page.locator('footer');
    await expect(footer).toBeVisible();
    await expect(footer.getByText('OpenPaths', { exact: true }).first()).toBeVisible();
  });
});
