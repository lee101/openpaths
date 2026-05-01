import { test, expect } from '@playwright/test';

test.describe('Blog', () => {
  test('agent integrations post renders and links', async ({ page }) => {
    await page.goto('/blog');
    await page.getByRole('link', { name: /OpenPaths Agent Integrations/ }).click();

    await expect(page).toHaveURL('/blog/openpaths-agent-integrations-hermes-openclaw');
    await expect(page.getByRole('heading', { name: 'OpenPaths Agent Integrations: Hermes Agent and OpenClaw' })).toBeVisible();
    await expect(page.getByRole('article').getByRole('link', { name: 'Integrations' })).toBeVisible();
  });

  test('new hosting post renders and links', async ({ page }) => {
    await page.goto('/blog');
    await page.getByRole('link', { name: /How OpenPaths Is Hosted on Codex Infinity/ }).click();

    await expect(page).toHaveURL('/blog/how-openpaths-is-hosted-on-codex-infinity');
    await expect(page.getByRole('heading', { name: 'How OpenPaths Is Hosted on Codex Infinity' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Codex Infinity' })).toBeVisible();
  });
});
