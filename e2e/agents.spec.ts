import { expect, test } from '@playwright/test';

const presets = [
  {
    key: 'researcher',
    name: 'Document Researcher',
    description: 'Answers questions grounded in your connected documents and databases, with citations.',
    system_prompt: 'Use connected sources and cite them.',
    model: 'claude-sonnet-latest',
    config: { tools: ['search_documents', 'query_database', 'call_model'], max_steps: 8 },
    example_prompts: ['Summarize the main decisions across these files and cite the sources.'],
  },
  {
    key: 'generalist',
    name: 'General Assistant',
    description: 'A blank multi-tool agent.',
    system_prompt: 'Be helpful.',
    model: 'auto',
    config: { tools: ['search_documents'], max_steps: 8 },
    example_prompts: ['Research these notes and recommend a next step.'],
  },
];

const tools = [
  { name: 'search_documents', label: 'Document search', description: 'Search connected files.' },
  { name: 'query_database', label: 'Database query', description: 'Run read-only queries.' },
  { name: 'call_model', label: 'Call another model', description: 'Delegate a sub-task.' },
];

test.beforeEach(async ({ page }) => {
  await page.route('**/v1/agents/presets', route => route.fulfill({ json: { presets, tools } }));
});

test('preset cards explain how an agent works before sign in', async ({ page }) => {
  await page.goto('/agents');

  await expect(page.getByRole('heading', { name: 'Give a model a job, tools, and knowledge.' })).toBeVisible();
  await page.getByRole('button', { name: /Document Researcher/ }).click();

  const dialog = page.getByRole('dialog');
  await expect(dialog.getByRole('heading', { name: 'Document Researcher' })).toBeVisible();
  await expect(dialog.getByText('Summarize the main decisions across these files and cite the sources.')).toBeVisible();
  await expect(dialog.getByRole('button', { name: /Document search/ })).toBeVisible();
  await expect(dialog.getByRole('link', { name: /Sign in to build/ })).toHaveAttribute('href', /\/account\?next=.*researcher/);
});

test('custom builder accepts knowledge files before creation', async ({ page }) => {
  await page.goto('/agents');
  await page.getByRole('button', { name: /Build your own/ }).first().click();

  const dialog = page.getByRole('dialog');
  await dialog.getByLabel('Name').fill('Policy assistant');
  await dialog.locator('input[type="file"]').setInputFiles({
    name: 'handbook.md',
    mimeType: 'text/markdown',
    buffer: Buffer.from('# Handbook\n\nUse citations.'),
  });

  await expect(dialog.getByText('handbook.md')).toBeVisible();
  await expect(dialog.getByText('Files become structured Markdown and searchable chunks.')).toBeVisible();
});

test('signed-in users can create a working agent from a preset', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('op_token', 'test-dashboard-token'));
  await page.route('**/v1/agents', async route => {
    if (route.request().method() === 'GET') {
      await route.fulfill({ json: { agents: [] } });
      return;
    }
    await route.fulfill({ json: {
      id: 'agent-1',
      name: 'Document Researcher',
      model: 'claude-sonnet-latest',
      config: { tools: ['search_documents', 'query_database', 'call_model'], max_steps: 8 },
    } });
  });
  await page.route('**/v1/agents/agent-1', route => route.fulfill({ json: {
    id: 'agent-1',
    name: 'Document Researcher',
    model: 'claude-sonnet-latest',
    config: { tools: ['search_documents', 'query_database', 'call_model'], max_steps: 8 },
    sources: [],
  } }));
  await page.goto('/agents');
  await page.getByRole('button', { name: /Document Researcher/ }).click();

  const creation = page.waitForRequest(request => request.url().endsWith('/v1/agents') && request.method() === 'POST');
  await page.getByRole('dialog').getByRole('button', { name: 'Create agent' }).click();
  const request = await creation;

  expect(request.postDataJSON()).toMatchObject({ preset: 'researcher', name: 'Document Researcher' });
  await expect(page).toHaveURL(/\/agents\/agent-1$/);
  await expect(page.getByRole('button', { name: 'Configure' })).toBeVisible();
});

test('agent workspace separates configuration, knowledge retrieval, and run history', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('op_token', 'test-dashboard-token'));
  await page.route('**/v1/agents/agent-detail/runs', route => route.fulfill({ json: { runs: [{
    id: 'run-1', input: 'Summarize the handbook', output: 'Summary', steps: [], status: 'complete', cost_cents: 2, created_at: '2026-07-13T10:00:00Z',
  }] } }));
  await page.route('**/v1/agents/agent-detail/search?**', route => route.fulfill({ json: { results: [{
    id: 'chunk-1', data_source_id: 'source-1', title: 'handbook.md', chunk_index: 0, content: 'Vacation requests require manager approval.', score: 0.91,
  }] } }));
  await page.route('**/v1/agents/agent-detail', route => route.fulfill({ json: {
    id: 'agent-detail', name: 'Policy assistant', description: 'Answers handbook questions.', model: 'auto', preset: 'researcher',
    config: { tools: ['search_documents'], max_steps: 8 },
    sources: [{ id: 'source-1', agent_id: 'agent-detail', kind: 'document', name: 'handbook.md', status: 'ready', chunk_count: 12, meta: { parser: 'markitdown' } }],
  } }));

  await page.goto('/agents/agent-detail?tab=knowledge');
  await expect(page.getByRole('heading', { name: 'Connected knowledge' })).toBeVisible();
  await page.getByPlaceholder('Search connected knowledge').fill('vacation approval');
  await page.getByPlaceholder('Search connected knowledge').press('Enter');
  await expect(page.getByText('Vacation requests require manager approval.')).toBeVisible();

  await page.getByRole('button', { name: 'Test & runs' }).click();
  await expect(page.getByRole('heading', { name: 'Run agent' })).toBeVisible();
  await expect(page.getByRole('button', { name: /Summarize the handbook/ })).toBeVisible();
});
