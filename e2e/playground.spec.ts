import { test, expect } from '@playwright/test';

test.describe('Playground Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/playground');
  });

  test('toolbar renders with Settings, Add Model, Clear buttons', async ({ page }) => {
    await expect(page.locator('button:has-text("Settings")')).toBeVisible();
    await expect(page.locator('button:has-text("Add Model")')).toBeVisible();
    await expect(page.locator('button:has-text("Clear")')).toBeVisible();
  });

  test('shows 2 default model panes', async ({ page }) => {
    await expect(page.locator('text=2/4 models')).toBeVisible();
  });

  test('settings panel toggles', async ({ page }) => {
    await expect(page.locator('label:has-text("API Key")')).not.toBeVisible();
    await page.click('button:has-text("Settings")');
    await expect(page.locator('label:has-text("API Key")')).toBeVisible();
    await expect(page.locator('label:has-text("System Prompt")')).toBeVisible();
  });

  test('add model pane up to 4', async ({ page }) => {
    await page.click('button:has-text("Add Model")');
    await expect(page.locator('text=3/4 models')).toBeVisible();

    await page.click('button:has-text("Add Model")');
    await expect(page.locator('text=4/4 models')).toBeVisible();

    // button should be disabled at 4
    const addBtn = page.locator('button:has-text("Add Model")');
    await expect(addBtn).toBeDisabled();
  });

  test('input is disabled without API key', async ({ page }) => {
    const textarea = page.locator('textarea[placeholder*="Set your API key"]');
    await expect(textarea).toBeDisabled();
  });

  test('input enables with API key', async ({ page }) => {
    await page.click('button:has-text("Settings")');
    await page.locator('input[placeholder="op-..."]').fill('test-key');
    const textarea = page.locator('textarea[placeholder*="Send a message"]');
    await expect(textarea).toBeEnabled();
  });

  test('model selector dropdown opens and shows providers', async ({ page }) => {
    // click first model selector
    const firstSelector = page.locator('button:has-text("Auto")').first();
    await firstSelector.click();
    await expect(page.locator('text=OpenAI').first()).toBeVisible();
    await expect(page.locator('text=Anthropic').first()).toBeVisible();
    await expect(page.locator('text=Google').first()).toBeVisible();
  });

  test('clear button resets messages', async ({ page }) => {
    await page.click('button:has-text("Clear")');
    await expect(page.locator('text=Send a message to start').first()).toBeVisible();
  });
});
