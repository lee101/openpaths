# Live speech to speech

Browser: `/tools/live-voice`. Choose Gemini 3.8 Live Extended Thinking,
GPT Realtime 2.1 Mini, or GPT Realtime 2.1, then start a call.
The browser captures mono PCM in 20 ms AudioWorklet frames, streams them over
WebSocket, and schedules returned 24 kHz audio immediately. Interruptions clear
queued playback; OpenAI also receives the played-audio truncation position.
Ending a call closes the socket, microphone tracks, and AudioContext.

## API

`wss://openpaths.io/v1/realtime?model=MODEL_ID`

Server clients authenticate with `Authorization: Bearer OPENPATHS_API_KEY`.
Browsers use subprotocols `openpaths-realtime` and
`openpaths-api-key.OPENPATHS_API_KEY`. The credential is never put in the URL or
echoed in the negotiated subprotocol. Provider credentials remain server-side.

Use native OpenAI Realtime events for GPT models and native Gemini Live events
for Gemini. The setup model must match the selected model; sessions cannot switch
to another model. Gemini input is 16 kHz PCM, OpenAI input is 24 kHz, and both
return 24 kHz PCM. Gemini uses LOW thinking and context compression. For Extended
Thinking, `turnComplete` ends an utterance; `interactionStatus` determines whether
background work is still running. Keep receiving throughout the session.

## Pricing

USD per million tokens; provider list price plus 5%:

| Model | Text input | Text output / thinking | Audio input | Audio output |
| --- | ---: | ---: | ---: | ---: |
| Gemini 3.8 Live Extended Thinking | 0.7875 | 4.725 | 3.15 | 12.60 |
| GPT Realtime 2.1 Mini | 0.63 | 2.52 | 10.50 | 21.00 |
| GPT Realtime 2.1 | 4.20 | 25.20 | 33.60 | 67.20 |

OpenAI cached input and image rates also include 5%. Billing uses the provider's
reported text/audio/image breakdown. Gemini reasoning tokens use the text output
rate. Existing conversation context contributes to input usage on later turns.
The ledger uses $0.0001 units, with the existing minimum charge and rounding.

Sources: [Google pricing](https://ai.google.dev/gemini-api/docs/pricing),
[GPT Realtime 2.1 Mini](https://developers.openai.com/api/docs/models/gpt-realtime-2.1-mini),
[GPT Realtime 2.1](https://developers.openai.com/api/docs/models/gpt-realtime-2.1).

## Verification

`npx playwright test e2e/live-voice.spec.ts` exercises microphone capture,
streamed playback, interruption state, mute/unmute, errors, and teardown with
mocked provider messages.

For paid production verification, use a recorded or naturally synthesized WAV
saying "pineapple" (or asking the assistant to repeat that word). Save it as
`/tmp/voice-request.wav`, pad it with ffmpeg, then run the browser suite:

```sh
ffmpeg -y -i /tmp/voice-request.wav -af 'adelay=5000,apad=pad_dur=35' -ar 48000 -ac 1 -c:a pcm_s16le /tmp/voice.wav
RUN_LIVE_VOICE_E2E=1 TARGET_URL=https://openpaths.io \
LIVE_VOICE_AUDIO_FILE=/tmp/voice.wav \
npx playwright test --config=playwright.voice.config.ts
```

Set `PAID_MODEL_API_KEY` (or `TEST_API_KEY` in `.env`) to a funded test account.
Optional `LIVE_VOICE_MODELS` is a comma-separated selection of model IDs.
This test uses real Chromium microphone capture with the spoken WAV, real provider
WebSockets, decoded audio amplitude, Web Audio playback completion, the spoken
answer transcript, provider usage, and a balance deduction. Traces are disabled
to keep authentication headers out of artifacts. Calls end even on failure.

## Production verification — 2026-09-16

Gemini 3.8 Live Extended Thinking passed the paid Chromium speech round trip on
openpaths.io, including a spoken "pineapple" reply, non-silent decoded PCM,
completed Web Audio playback, reported usage, and a balance deduction. A natural
speech fixture was used; an eSpeak sentence was not transcribed reliably.

Both OpenAI realtime models reached OpenAI through the deployed relay but returned
`credit_balance_exhausted`. Their real audio verification needs the OpenAI provider
account funded (or a funded BYOK account); mocked protocol and playback tests pass.
