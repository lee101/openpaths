import { test, expect } from '@playwright/test';

test.describe('Integrations Page', () => {
  test('renders supported SDK guides', async ({ page }) => {
    await page.goto('/integrations');

    await expect(page.locator('h1')).toContainText('Integrate OpenPaths With Your Agent Stack');
    await expect(page.getByTestId('integrations-base-url')).toContainText('/v1');

    for (const id of ['openai-agents-sdk', 'claude-code-gateway', 'anthropic-agent-sdk', 'hermes-agent', 'openclaw', 'langchain', 'vercel-ai-sdk', 'pydantic-ai', 'mastra', 'langfuse', 'livekit']) {
      await expect(page.getByTestId(`integration-${id}`)).toBeVisible();
    }

    await expect(page.getByTestId('code-openai-agents-sdk')).toContainText('/v1');
    await expect(page.getByTestId('code-anthropic-agent-sdk')).toContainText('https://openpaths.io');
    await expect(page.getByTestId('code-claude-code-gateway')).toContainText('nvidia/deepseek-v4-pro');
    await expect(page.getByTestId('code-claude-code-gateway')).not.toContainText('openpaths.io/v1');
  });

  test('auto-populates stored API key in SDK examples', async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('op_api_key', 'op-test-integrations-key');
    });

    await page.goto('/integrations');

    await expect(page.getByTestId('integrations-api-key')).toContainText('op-test-integrations-key');
    await expect(page.getByTestId('code-openai-agents-sdk')).toContainText('op-test-integrations-key');
    await expect(page.getByTestId('code-claude-code-gateway')).toContainText('op-test-integrations-key');
    await expect(page.getByTestId('code-anthropic-agent-sdk')).toContainText('op-test-integrations-key');
    await expect(page.getByTestId('code-hermes-agent')).toContainText('op-test-integrations-key');
    await expect(page.getByTestId('code-openclaw')).toContainText('op-test-integrations-key');
    await expect(page.getByTestId('code-langchain')).toContainText('op-test-integrations-key');
    await expect(page.getByTestId('code-vercel-ai-sdk')).toContainText('op-test-integrations-key');
    await expect(page.getByTestId('code-pydantic-ai')).toContainText('op-test-integrations-key');
    await expect(page.getByTestId('code-mastra')).toContainText('op-test-integrations-key');
    await expect(page.getByTestId('code-langfuse')).toContainText('op-test-integrations-key');
    await expect(page.getByTestId('code-livekit')).toContainText('op-test-integrations-key');
  });
});
