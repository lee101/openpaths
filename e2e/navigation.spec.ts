import { test, expect } from '@playwright/test';

// The top bar carries only the core destinations; everything else lives behind
// the menu button. Use this for any link that is not in BAR_LABELS.
async function openNavMenu(page: import('@playwright/test').Page) {
  await page.getByRole('button', { name: 'Open menu' }).click();
}

test.describe('Global Navigation', () => {
  test('navbar links render and navigate correctly', async ({ page }) => {
    await page.goto('/');
    const nav = page.locator('nav');
    await expect(nav).toBeVisible();
    await expect(nav.locator('text=OpenPath')).toBeVisible();
    await expect(nav.locator('text=Models')).toBeVisible();
    await expect(nav.locator('text=Playground')).toBeVisible();
    await expect(nav.getByTestId('nav-get-started').or(nav.getByTestId('nav-dashboard'))).toBeVisible();

    // The rest of the destinations are reachable through the menu.
    await openNavMenu(page);
    await expect(nav.locator('a[href="/apps/"]').last()).toBeVisible();
    await expect(nav.locator('a[href="/integrations"]').last()).toBeVisible();
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
    await openNavMenu(page);
    await page.locator('nav a[href="/integrations"]').last().click();
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
    await openNavMenu(page);
    await page.locator('nav a[href="/apps/"]').last().click();
    await expect(page).toHaveURL('/apps/');
    await expect(page.locator('h1')).toContainText('Apps And Agents');
  });

  test('primary account button opens sign-in', async ({ page }) => {
    await page.goto('/');
    const primaryAccountButton = page.getByTestId('nav-get-started').or(page.getByTestId('nav-dashboard'));
    await primaryAccountButton.click();
    await expect(page.getByTestId('auth-modal')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Sign In' })).toBeVisible();
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
