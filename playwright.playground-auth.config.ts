import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  testMatch: 'playground-auth-real.spec.ts',
  retries: 0,
  reporter: 'list',
  use: { baseURL: process.env.TARGET_URL, viewport: { width: 1440, height: 900 } },
  projects: [{ name: 'chromium', use: { browserName: 'chromium' } }],
});
