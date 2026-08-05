import 'dotenv/config';
import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  testIgnore: [
    '**/auth-real.spec.ts',
    '**/checkout-flow.spec.ts',
    '**/full-flow.spec.ts',
    '**/gpt54-models.spec.ts',
    '**/gpt56-models.spec.ts',
    '**/minimax-m27.spec.ts',
    '**/new-models-live.spec.ts',
    '**/xai-video-live.spec.ts',
  ],
  fullyParallel: true,
  retries: 0,
  reporter: 'list',
  use: {
    baseURL: 'http://localhost:3099',
    trace: 'on-first-retry',
  },
  webServer: {
    command: 'npx vite preview --port=3099 --host=0.0.0.0',
    port: 3099,
    reuseExistingServer: true,
    timeout: 30000,
  },
  projects: [
    { name: 'chromium', use: { browserName: 'chromium' } },
  ],
});
