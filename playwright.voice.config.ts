import 'dotenv/config';
import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  testMatch: '**/live-voice-paid.spec.ts',
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 60000,
  reporter: 'list',
  use: {
    baseURL: process.env.TARGET_URL || 'http://localhost:3099',
    permissions: ['microphone'],
    // Authentication travels in WS subprotocols: don't put it in traces or videos.
    trace: 'off', video: 'off',
    launchOptions: { args: [
      '--use-fake-device-for-media-stream', '--use-fake-ui-for-media-stream',
      ...(process.env.LIVE_VOICE_AUDIO_FILE ? [`--use-file-for-fake-audio-capture=${process.env.LIVE_VOICE_AUDIO_FILE}`] : []),
    ] },
  },
  projects: [{ name: 'chromium', use: { browserName: 'chromium' } }],
});
