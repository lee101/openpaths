import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  testMatch: 'funnel-regressions.spec.ts',
  fullyParallel: true,
  reporter: 'list',
  use: { baseURL: 'http://127.0.0.1:3107', viewport: { width: 390, height: 844 } },
  webServer: {
    command: 'npx vite --port=3107 --host=127.0.0.1 --strictPort',
    url: 'http://127.0.0.1:3107',
    reuseExistingServer: false,
    timeout: 60000,
  },
  projects: [{ name: 'chromium', use: { browserName: 'chromium' } }],
});
