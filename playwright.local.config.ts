import { defineConfig } from '@playwright/test';
import base from './playwright.config';

export default defineConfig({
  ...(base as any),
  use: { ...(base as any).use, baseURL: 'http://127.0.0.1:3101' },
  webServer: undefined,
});
