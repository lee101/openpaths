import { test, expect } from '@playwright/test';

test.describe('Global Navigation', () => {
  test('navbar links render and navigate correctly', async ({ page }) => {
    await page.goto('/');
    const nav = page.locator('nav');
    await expect(nav).toBeVisible();
    await expect(nav.locator('text=OpenPath')).toBeVisible();
    await expect(nav.locator('text=Models')).toBeVisible();
    await expect(nav.locator('text=Playground')).toBeVisible();
    await expect(nav.locator('text=Dashboard')).toBeVisible();
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

  test('navigate to /account via Dashboard button', async ({ page }) => {
    await page.goto('/');
    await page.click('nav >> text=Dashboard');
    await expect(page).toHaveURL('/account');
    await expect(page.locator('h2:has-text("Account")')).toBeVisible();
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
    await expect(footer.locator('text=OpenPath')).toBeVisible();
  });
});
