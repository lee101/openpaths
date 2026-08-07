import { defineConfig } from '@playwright/test';

/**
 * Config for real-server e2e tests.
 * Requires the Go backend running at localhost:8090.
 *
 * Local uses vite dev (port 3099) which proxies API calls to the Go server.
 * Live runs directly against TARGET_URL.
 *
 * Local:   npx playwright test --config=playwright.real.config.ts
 * Live:    TARGET_URL=https://openpaths.io npx playwright test --config=playwright.real.config.ts
 * Single:  npx playwright test e2e/full-flow.spec.ts --config=playwright.real.config.ts
 */
const isLive = !!process.env.TARGET_URL;

export default defineConfig({
  testDir: './e2e',
  testMatch: ['**/auth-real.spec.ts', '**/full-flow.spec.ts', '**/gpt54-models.spec.ts', '**/gpt56-models.spec.ts', '**/minimax-m27.spec.ts', '**/new-models-live.spec.ts', '**/xai-video-live.spec.ts'],
  fullyParallel: false,
  retries: 0,
  timeout: 60000,
  reporter: 'list',
  use: {
    baseURL: process.env.TARGET_URL || 'http://localhost:3099',
    trace: 'on-first-retry',
  },
  ...(isLive ? {} : {
    webServer: {
      command: 'npx vite --port=3099 --host=0.0.0.0',
      port: 3099,
      reuseExistingServer: true,
      timeout: 30000,
    },
  }),
  projects: [
    { name: 'chromium', use: { browserName: 'chromium' } },
  ],
});
