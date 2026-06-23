import { test, expect } from '@playwright/test';

test.describe('Playground Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/playground');
  });

  test('toolbar renders with Settings and Compare buttons', async ({ page }) => {
    await expect(page.locator('button:has-text("Settings")')).toBeVisible();
    await expect(page.locator('button:has-text("Compare")')).toBeVisible();
    await expect(page.locator('button:has-text("Clear")')).not.toBeVisible();
  });

  test('shows 1 default model pane', async ({ page }) => {
    await expect(page.locator('button:has-text("Auto (intelligent routing)")')).toBeVisible();
  });

  test('settings panel toggles', async ({ page }) => {
    await expect(page.locator('label:has-text("API Key")')).not.toBeVisible();
    await page.click('button:has-text("Settings")');
    await expect(page.locator('label:has-text("API Key")')).toBeVisible();
    await expect(page.locator('label:has-text("System Prompt")')).toBeVisible();
    await expect(page.getByTestId('playground-export-attachments')).toBeVisible();
    await expect(page.getByTestId('playground-auto-archive-days')).toBeVisible();
  });

  test('compare model pane up to 4', async ({ page }) => {
    await page.click('button:has-text("Compare")');
    await expect(page.locator('text=2/4 models')).toBeVisible();

    await page.click('button:has-text("Compare")');
    await expect(page.locator('text=3/4 models')).toBeVisible();

    await page.click('button:has-text("Compare")');
    await expect(page.locator('text=4/4 models')).toBeVisible();

    // button should be disabled at 4
    await expect(page.locator('button:has-text("Compare")')).toBeDisabled();
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

  test('attaches uploaded files to chat requests', async ({ page }) => {
    let chatBody: any = null;
    await page.route('**/v1/files/upload', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ url: 'https://openpathsstatic.openpaths.io/static/uploads/e2e-notes.txt' }),
      });
    });
    await page.route('**/v1/chat/completions', async (route) => {
      chatBody = route.request().postDataJSON();
      await route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: 'data: {"choices":[{"delta":{"content":"attached ok"}}]}\n\ndata: [DONE]\n\n',
      });
    });

    await page.evaluate(() => {
      localStorage.setItem('op_api_key', 'op-attachment-test-key');
    });
    await page.reload();

    await page.locator('input[type="file"][multiple]').setInputFiles({
      name: 'e2e-notes.txt',
      mimeType: 'text/plain',
      buffer: Buffer.from('OpenPaths attachment test content.'),
    });

    await expect(page.getByText('e2e-notes.txt')).toBeVisible();
    await expect(page.getByTestId('chat-send')).toBeEnabled();
    await page.getByTestId('chat-send').click();

    await expect(page.getByText('attached ok')).toBeVisible();
    const userMessage = chatBody?.messages?.find((m: any) => m.role === 'user');
    expect(Array.isArray(userMessage?.content)).toBe(true);
    expect(JSON.stringify(userMessage.content)).toContain('OpenPaths attachment test content.');
    expect(JSON.stringify(userMessage.content)).toContain('https://openpathsstatic.openpaths.io/static/uploads/e2e-notes.txt');
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
    await page.evaluate(() => {
      localStorage.setItem('op_pg_pane_auto', JSON.stringify([{ role: 'user', content: 'hello' }]));
    });
    await page.reload();
    await page.click('button:has-text("Clear")');
    await expect(page.locator('button:has-text("Clear")')).not.toBeVisible();
    await expect(page.locator('text=Sign in to start comparing models').first()).toBeVisible();
  });

  test('delete button removes a single message from the conversation', async ({ page }) => {
    await page.evaluate(() => {
      localStorage.setItem('op_api_key', 'op-test-key');
      localStorage.setItem('op_pg_pane_auto', JSON.stringify([
        { role: 'user', content: 'first question' },
        { role: 'assistant', content: 'first answer' },
      ]));
    });
    await page.reload();

    await expect(page.locator('text=first answer')).toBeVisible();
    // Delete the assistant reply (second message).
    await page.getByTestId('msg-delete').last().click();
    await expect(page.locator('text=first answer')).not.toBeVisible();
    await expect(page.locator('text=first question')).toBeVisible();
    // Deletion persists to localStorage.
    const stored = await page.evaluate(() => localStorage.getItem('op_pg_pane_auto'));
    expect(stored).not.toContain('first answer');
    expect(stored).toContain('first question');
  });

  test('retry on an assistant reply drops the old reply before regenerating', async ({ page }) => {
    await page.evaluate(() => {
      localStorage.setItem('op_api_key', 'op-test-key');
      localStorage.setItem('op_pg_pane_auto', JSON.stringify([
        { role: 'user', content: 'keep this prompt' },
        { role: 'assistant', content: 'stale reply to regenerate' },
      ]));
    });
    await page.reload();

    await expect(page.locator('text=stale reply to regenerate')).toBeVisible();
    await page.getByTestId('msg-retry').last().click();
    // The old assistant reply is removed; the user prompt is preserved.
    await expect(page.locator('text=stale reply to regenerate')).not.toBeVisible();
    await expect(page.locator('text=keep this prompt')).toBeVisible();
  });

  test('copy code panel highlights snippets and injects stored API key', async ({ page }) => {
    await page.evaluate(() => {
      localStorage.setItem('op_api_key', 'op-playground-code-key');
      localStorage.setItem('op_pg_pane_auto', JSON.stringify([{ role: 'user', content: 'hello from test' }]));
    });
    await page.reload();

    await page.click('button:has-text("Copy Code")');

    const generatedCode = page.getByTestId('playground-generated-code');
    await expect(generatedCode).toContainText('op-playground-code-key');
    await expect(generatedCode).not.toContainText('YOUR_OPENPATHS_API_KEY');
    await expect(generatedCode.locator('.hljs-string').first()).toBeVisible();

    await page.click('button:has-text("JavaScript")');
    await expect(generatedCode).toContainText('apiKey: "op-playground-code-key"');

    await page.click('button:has-text("Go")');
    await expect(generatedCode).toContainText('Bearer op-playground-code-key');

    await page.click('button:has-text("cURL")');
    await expect(generatedCode).toContainText('Authorization: Bearer op-playground-code-key');
  });

  test('image playground exposes generation settings and image code', async ({ page }) => {
    await page.evaluate(() => {
      localStorage.setItem('op_api_key', 'op-image-code-key');
    });
    await page.goto('/playground?model=ra1&mode=image');

    await expect(page.locator('button:has-text("RA1 Art Generator")')).toBeVisible();
    await expect(page.getByTestId('image-size')).toBeVisible();
    await page.getByTestId('image-size').selectOption('1360x768');
    await page.getByTestId('image-quality').selectOption('high');
    await page.getByTestId('image-count').fill('2');
    await page.getByTestId('image-response-format').selectOption('b64_json');

    await page.locator('textarea[placeholder*="Describe the image"]').fill('A black espresso machine on a marble counter');
    await page.click('button:has-text("Copy Code")');

    const generatedCode = page.getByTestId('playground-generated-code');
    await expect(generatedCode).toContainText('op-image-code-key');
    await expect(generatedCode).toContainText('client.post');
    await expect(generatedCode).toContainText('/images/generations');
    await expect(generatedCode).toContainText('"model": "ra1"');
    await expect(generatedCode).toContainText('"size": "1360x768"');
    await expect(generatedCode).toContainText('"quality": "high"');

    await page.click('button:has-text("cURL")');
    await expect(generatedCode).toContainText('/images/generations');
    await expect(generatedCode).toContainText('Authorization: Bearer op-image-code-key');
    await expect(generatedCode).toContainText('"response_format": "b64_json"');
  });

  test('speech playground exposes voice settings and speech code', async ({ page }) => {
    await page.evaluate(() => {
      localStorage.setItem('op_api_key', 'op-speech-code-key');
    });
    await page.goto('/playground?model=xai-tts');

    await expect(page.getByTestId('speech-voice')).toBeVisible();
    await page.getByTestId('speech-voice').selectOption('ara');
    await page.getByTestId('speech-language').selectOption('en');
    await expect(page.locator('text=$15.00 / 1M input characters')).toBeVisible();

    await page.locator('textarea[placeholder*="Enter text to synthesize"]').fill('Hello from Grok text to speech.');
    await page.click('button:has-text("Copy Code")');

    const generatedCode = page.getByTestId('playground-generated-code');
    await expect(generatedCode).toContainText('op-speech-code-key');
    await expect(generatedCode).toContainText('/audio/speech');
    await expect(generatedCode).toContainText('"model": "xai-tts"');
    await expect(generatedCode).toContainText('"voice": "ara"');

    await page.click('button:has-text("cURL")');
    await expect(generatedCode).toContainText('Authorization: Bearer op-speech-code-key');
    await expect(generatedCode).toContainText('"input": "Hello from Grok text to speech."');
  });
});
