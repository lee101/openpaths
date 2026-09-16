import { expect, test } from '@playwright/test';

test.use({ permissions: ['microphone'], launchOptions: { args: ['--use-fake-device-for-media-stream', '--use-fake-ui-for-media-stream'] } });

for (const model of ['gemini-3.8-live-extended-thinking', 'gpt-realtime-2.1-mini']) {
  test(`${model}: microphone, streamed playback, interruption, mute, and cleanup`, async ({ page }) => {
    let micFrames = 0;
    let setup: any;
    let socket: any;
    const google = model.startsWith('gemini');
    const pcm = Buffer.alloc(4800);
    for (let i = 0; i < 2400; i++) pcm.writeInt16LE(Math.sin(i * 0.1) * 3000, i * 2);
    await page.routeWebSocket('**/v1/realtime?**', ws => {
      socket = ws;
      if (!google) ws.send(JSON.stringify({ type: 'session.created' }));
      ws.onMessage(message => {
        const e = JSON.parse(message.toString());
        if (e.setup || e.type === 'session.update') {
          setup = e;
          ws.send(JSON.stringify(google ? { setupComplete: {} } : { type: 'session.updated' }));
        }
        if (e.realtimeInput?.audio || e.type === 'input_audio_buffer.append') {
          micFrames++;
          if (micFrames === 3) {
            ws.send(JSON.stringify(google ? { serverContent: { interactionStatus: 'IN_PROGRESS', modelTurn: { parts: [{ inlineData: { mimeType: 'audio/pcm;rate=24000', data: pcm.toString('base64') } }] }, outputTranscription: { text: 'Hello from your voice assistant.' } } } : { type: 'response.output_audio.delta', item_id: 'item1', delta: pcm.toString('base64') }));
            if (!google) ws.send(JSON.stringify({ type: 'response.output_audio_transcript.delta', delta: 'Hello from your voice assistant.' }));
          }
        }
      });
    });
    await page.goto(`/tools/live-voice?model=${model}`);
    await page.getByLabel('OpenPaths API key').fill('sk-op-test-voice');
    await page.getByRole('button', { name: 'Start call' }).click();
    await expect.poll(() => micFrames).toBeGreaterThan(3);
    await expect(page.getByTestId('received-frames')).toContainText('1 audio frames');
    await expect(page.getByRole('region', { name: 'Live transcript' })).toContainText('Hello from your voice assistant.');
    if (google) {
      expect(setup.setup.model).toBe(`models/${model}`);
      expect(setup.setup.generationConfig.thinkingConfig.thinkingLevel).toBe('LOW');
      socket.send(JSON.stringify({ serverContent: { turnComplete: true, interactionStatus: 'IN_PROGRESS' } }));
      await expect(page.getByRole('status')).toHaveText('Thinking');
      socket.send(JSON.stringify({ serverContent: { interactionStatus: 'IDLE' } }));
    } else {
      expect(setup.session.audio.input.format.rate).toBe(24000);
      socket.send(JSON.stringify({ type: 'input_audio_buffer.speech_started' }));
    }
    await expect(page.getByRole('status')).toHaveText('Listening');
    await page.getByRole('button', { name: 'Mute microphone' }).click();
    const mutedAt = micFrames;
    await page.waitForTimeout(200);
    expect(micFrames).toBeLessThanOrEqual(mutedAt + 1);
    await page.getByRole('button', { name: 'Unmute microphone' }).click();
    await expect.poll(() => micFrames).toBeGreaterThan(mutedAt + 2);
    await page.getByRole('button', { name: 'End call' }).click();
    await expect(page.getByRole('status')).toHaveText('Ready to talk');
    await expect(page.getByLabel('Voice model')).toBeEnabled();
    const stoppedAt = micFrames;
    await page.waitForTimeout(200);
    expect(micFrames).toBe(stoppedAt);
  });
}

test('provider errors end the call and allow retry', async ({ page }) => {
  await page.routeWebSocket('**/v1/realtime?**', ws => ws.onMessage(() => ws.send(JSON.stringify({ error: { message: 'Insufficient provider credits' } }))));
  await page.goto('/tools/live-voice');
  await page.getByLabel('OpenPaths API key').fill('sk-op-test-voice');
  await page.getByRole('button', { name: 'Start call' }).click();
  await expect(page.getByRole('alert')).toHaveText('Insufficient provider credits');
  await expect(page.getByRole('button', { name: 'Start call' })).toBeEnabled();
});
