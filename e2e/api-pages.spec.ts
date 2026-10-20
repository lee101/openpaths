import { expect, test } from '@playwright/test';

const withKey = (page: import('@playwright/test').Page) => page.addInitScript(() => localStorage.setItem('op_api_key', 'op-test'));

test('hub lists every capability and comparisons', async ({ page }) => {
  await page.goto('/api');
  await expect(page.getByRole('heading', { name: 'One API key for every model' })).toBeVisible();
  for (const name of ['Chat Completions', 'Free LLM API', 'Embeddings', 'Video Generation', 'Web Search']) {
    await expect(page.getByRole('link', { name: new RegExp(name) }).first()).toBeVisible();
  }
  await expect(page.locator('a[href^="/vs/"]').first()).toBeVisible();
});

test('embeddings try-it sends the selected model and renders dimensions', async ({ page }) => {
  await withKey(page);
  let payload: Record<string, unknown> | undefined;
  await page.route('**/v1/embeddings', async route => {
    payload = route.request().postDataJSON();
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ data: [{ embedding: [0.1, 0.2, 0.3] }], usage: { total_tokens: 7 } }) });
  });
  await page.goto('/api/embeddings');
  await expect(page.getByRole('heading', { level: 1 })).toContainText('Embeddings API');
  await page.getByTestId('tryit-run').click();
  await expect.poll(() => payload).toBeTruthy();
  expect(payload?.model).toBe('openpaths-embed');
  await expect(page.getByTestId('tryit-result')).toContainText('3 dimensions');
  await page.getByRole('button', { name: 'JS', exact: true }).click();
  await expect(page.locator('pre')).toContainText('client.embeddings.create');
});

test('anonymous visitors are asked to sign in and no API call is made', async ({ page }) => {
  let calls = 0;
  await page.route('**/v1/**', async route => {
    calls += 1;
    await route.abort();
  });
  await page.goto('/api/free-llm');
  await expect(page.getByTestId('tryit-model')).toHaveValue('or/nemotron-nano');
  await page.getByTestId('tryit-run').click();
  await expect(page.getByTestId('subscribe-prompt')).toBeVisible();
  expect(calls).toBe(0);
});

test('free page only offers $0 models', async ({ page }) => {
  await page.goto('/api/free-llm');
  const options = await page.getByTestId('tryit-model').locator('option').evaluateAll(els => els.map(e => (e as HTMLOptionElement).value));
  expect(options.sort()).toEqual(['glm-4.6v-flash', 'nvidia/deepseek-v4-flash', 'or/nemotron-nano']);
});

test('empty balance on a paid model shows the add-credits prompt', async ({ page }) => {
  await withKey(page);
  await page.route('**/v1/chat/completions', route => route.fulfill({
    status: 402,
    contentType: 'application/json',
    body: JSON.stringify({ error: { code: 'subscription_required', message: 'Add credits', subscribe_url: '/pricing' } }),
  }));
  await page.goto('/api/chat-completions');
  await page.getByTestId('tryit-run').click();
  await expect(page.getByTestId('subscribe-prompt-topup')).toBeVisible();
});

test('chat try-it renders the reply', async ({ page }) => {
  await withKey(page);
  let payload: any;
  await page.route('**/v1/chat/completions', async route => {
    payload = route.request().postDataJSON();
    await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ choices: [{ message: { content: 'hello there' } }], usage: { prompt_tokens: 3, completion_tokens: 2 } }) });
  });
  await page.goto('/api/chat-completions');
  await page.getByTestId('tryit-input').fill('say hi');
  await page.getByTestId('tryit-run').click();
  await expect(page.getByTestId('tryit-result')).toContainText('hello there');
  expect(payload.messages[0].content).toBe('say hi');
});

test('model vs page compares two models and links back', async ({ page }) => {
  await page.goto('/api/chat-completions');
  const href = await page.locator('a[href^="/vs/"]').first().getAttribute('href');
  expect(href).toBeTruthy();
  await page.goto(href as string);
  await expect(page.getByRole('heading', { level: 1 })).toContainText(' vs ');
  await expect(page.locator('table')).toBeVisible();
  await expect(page.getByTestId('tryit-model')).toBeVisible();
});

test('model page exposes the API try-it for embedding models', async ({ page }) => {
  await page.goto('/models/openpaths-embed');
  await expect(page.getByRole('heading', { name: /via API/ })).toBeVisible();
  await expect(page.getByTestId('capability-tryit')).toBeVisible();
});

test('unknown capability redirects to the hub', async ({ page }) => {
  await page.goto('/api/nope');
  await expect(page).toHaveURL(/\/api$/);
});
