/**
 * Real speech in/out through Chromium and the deployed relay. No API/audio mocks.
 * RUN_LIVE_VOICE_E2E=1 TARGET_URL=https://openpaths.io LIVE_VOICE_AUDIO_FILE=/tmp/voice.wav
 * PAID_MODEL_API_KEY=... npx playwright test --config=playwright.voice.config.ts
 * The WAV should say "pineapple" or ask the assistant to repeat that word, with
 * five seconds of leading silence and at least 30 seconds of trailing silence.
 */
import { expect, test } from '@playwright/test';
import { writeFile } from 'node:fs/promises';

const key = process.env.PAID_MODEL_API_KEY || process.env.TEST_API_KEY || '';
const models = (process.env.LIVE_VOICE_MODELS || 'gemini-3.8-live-extended-thinking,gpt-realtime-2.1-mini,gpt-realtime-2.1').split(',');
test.skip(process.env.RUN_LIVE_VOICE_E2E !== '1' || !key || !process.env.LIVE_VOICE_AUDIO_FILE, 'requires explicit paid voice opt-in, API key, and speech WAV');

for (const model of models) {
  test(`${model}: real microphone speech produces audible streamed reply`, async ({ page, request }, testInfo) => {
    const before = await request.get('/account/balance', { headers: { Authorization: `Bearer ${key}` } });
    expect(before.ok()).toBeTruthy();
    const initialBalance = (await before.json()).balance_cents;
    const errors: string[] = [];
    let audioChunks = 0;
    let inputFrames = 0;
    let outputSamples = 0;
    let sumSquares = 0;
    const usage: unknown[] = [];
    const started = Date.now();
    let firstAudioAt = 0;
    page.on('websocket', socket => {
      if (!socket.url().includes('/v1/realtime')) return;
      socket.on('socketerror', error => errors.push(String(error)));
      socket.on('framesent', frame => {
        const e = JSON.parse(frame.payload.toString());
        if (e.type === 'input_audio_buffer.append' || e.realtimeInput?.audio) inputFrames++;
      });
      socket.on('framereceived', frame => {
        const e = JSON.parse(frame.payload.toString());
        if (e.error) errors.push(e.error.message);
        if (e.usageMetadata || e.response?.usage) usage.push(e.usageMetadata || e.response.usage);
        const encoded: string[] = e.type === 'response.output_audio.delta' || e.type === 'response.audio.delta' ? [e.delta] :
          (e.serverContent?.modelTurn?.parts || []).filter((p: any) => p.inlineData?.mimeType?.startsWith('audio/pcm')).map((p: any) => p.inlineData.data);
        for (const data of encoded) {
          if (!firstAudioAt) firstAudioAt = Date.now();
          audioChunks++;
          const pcm = Buffer.from(data, 'base64');
          for (let i = 0; i + 1 < pcm.length; i += 2) { sumSquares += (pcm.readInt16LE(i) / 32768) ** 2; outputSamples++; }
        }
      });
    });
    // Observe actual Web Audio playback without replacing any audio implementation.
    await page.addInitScript(() => {
      (window as any).__voicePlayback = { starts: 0, ended: 0 };
      const original = AudioBufferSourceNode.prototype.start;
      AudioBufferSourceNode.prototype.start = function (...args: Parameters<typeof original>) {
        (window as any).__voicePlayback.starts++;
        this.addEventListener('ended', () => { (window as any).__voicePlayback.ended++; });
        return original.apply(this, args);
      };
    });
    await page.goto(`/tools/live-voice?model=${model}`);
    await page.getByLabel('OpenPaths API key').fill(key);
    await page.getByLabel('Voice instructions').fill('Respond in English. Follow the spoken request exactly; if the user says a single word, repeat it in English. Reply with one word only.');
    await page.getByRole('button', { name: 'Start call' }).click();
    try {
      await expect.poll(() => errors.length > 0 || audioChunks > 1, { timeout: 30000 }).toBeTruthy();
      expect(errors, 'Provider must accept this real speech session').toEqual([]);
      expect(inputFrames).toBeGreaterThan(20);
      expect(outputSamples).toBeGreaterThan(2400);
      expect(Math.sqrt(sumSquares / outputSamples)).toBeGreaterThan(0.001);
      await expect(page.getByRole('region', { name: 'Live transcript' }).locator('p').filter({ hasText: /^Assistant:/ }).filter({ hasText: /pineapple/i }).first()).toBeVisible({ timeout: 10000 });
      await expect.poll(() => page.evaluate(() => (window as any).__voicePlayback.ended)).toBeGreaterThan(0);
      await expect.poll(() => usage.length, { timeout: 10000 }).toBeGreaterThan(0);
      await expect.poll(() => page.evaluate(() => {
        const playback = (window as any).__voicePlayback;
        return playback.starts > 0 && playback.ended === playback.starts;
      })).toBeTruthy();
      await page.getByRole('button', { name: 'End call' }).click();
      await expect(page.getByRole('status')).toHaveText('Ready to talk');
      expect(errors).toEqual([]);
      await expect.poll(async () => {
        const res = await request.get('/account/balance', { headers: { Authorization: `Bearer ${key}` } });
        return (await res.json()).balance_cents;
      }, { timeout: 10000 }).toBeLessThan(initialBalance);
      // Remove the credential from the screenshot and never record protocol headers.
      await page.getByLabel('OpenPaths API key').fill('');
      await page.screenshot({ path: testInfo.outputPath('voice-success.png'), fullPage: true });
      const evidence = JSON.stringify({ model, inputFrames, audioChunks, audioSeconds: outputSamples / 24000, rms: Math.sqrt(sumSquares / outputSamples), firstAudioFromStartMs: firstAudioAt - started, usage }, null, 2);
      await writeFile(testInfo.outputPath('voice-evidence.json'), evidence);
      await testInfo.attach('voice-evidence', { body: evidence, contentType: 'application/json' });
    } finally {
      if (await page.getByRole('button', { name: 'End call' }).isVisible()) await page.getByRole('button', { name: 'End call' }).click();
      await page.getByLabel('OpenPaths API key').fill('');
    }
  });
}
