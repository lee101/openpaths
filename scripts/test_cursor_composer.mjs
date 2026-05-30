#!/usr/bin/env node
/**
 * Smoke test Cursor Composer 2.5 Fast via the Cursor SDK (local agent).
 *
 * This is NOT OpenAI-style /v1/chat/completions. Composer runs through the
 * Cursor Agent SDK, which uses the same agent runtime as the IDE/CLI.
 *
 * Model: composer-2.5 with param fast=true (Composer 2.5 Fast).
 * Auth: CURSOR_API_KEY from https://cursor.com/dashboard/integrations
 *
 * Cloud REST alternative: POST https://api.cursor.com/v1/agents with
 *   { "model": { "id": "composer-2.5", "params": [{ "id": "fast", "value": "true" }] } }
 * Requires Cursor "storage mode" enabled for cloud agents.
 */

import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const scriptDir = dirname(fileURLToPath(import.meta.url));
const sdkEntry = join(scriptDir, '.cursor-sdk-deps/node_modules/@cursor/sdk/dist/esm/index.js');
const { Agent } = await import(pathToFileURL(sdkEntry).href);

const apiKey = process.env.CURSOR_API_KEY;
if (!apiKey) {
  console.error('CURSOR_API_KEY not set');
  process.exit(1);
}

const cwd = process.env.CURSOR_TEST_CWD || process.cwd();
const prompt = process.env.CURSOR_TEST_PROMPT || 'say hi nothing else';

const result = await Agent.prompt(prompt, {
  apiKey,
  model: {
    id: 'composer-2.5',
    params: [{ id: 'fast', value: 'true' }],
  },
  local: { cwd },
});

console.log('status:', result.status);
console.log('result:', result.result);

if (result.status !== 'finished') {
  process.exit(2);
}
