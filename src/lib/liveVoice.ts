export const LIVE_MODELS = [
  { id: 'gemini-3.8-live-extended-thinking', name: 'Gemini 3.8 Live · Extended Thinking', provider: 'google', audioIn: 3.15, audioOut: 12.60, textIn: 0.7875, textOut: 4.725 },
  { id: 'gpt-realtime-2.1-mini', name: 'GPT Realtime 2.1 Mini', provider: 'openai', audioIn: 10.50, audioOut: 21, textIn: 0.63, textOut: 2.52 },
  { id: 'gpt-realtime-2.1', name: 'GPT Realtime 2.1', provider: 'openai', audioIn: 33.60, audioOut: 67.20, textIn: 4.20, textOut: 25.20 },
] as const;

export type VoiceState = 'idle' | 'connecting' | 'listening' | 'thinking' | 'speaking';
export type VoiceStats = { sentFrames: number; receivedFrames: number; playedSeconds: number; firstAudioMs: number | null };
type Callbacks = {
  state: (state: VoiceState) => void;
  transcript: (role: 'user' | 'assistant', text: string, finished: boolean) => void;
  error: (message: string) => void;
  stats: (stats: VoiceStats) => void;
};

function base64(buffer: ArrayBuffer): string {
  return btoa(String.fromCharCode(...new Uint8Array(buffer)));
}

/** Native provider events over our authenticated relay. Audio is never collected into a file. */
export class LiveVoiceSession {
  private socket?: WebSocket;
  private context?: AudioContext;
  private stream?: MediaStream;
  private capture?: AudioWorkletNode;
  private source?: MediaStreamAudioSourceNode;
  private output = new Set<AudioBufferSourceNode>();
  private nextPlayAt = 0;
  private closed = false;
  private ready = false;
  private muted = false;
  private timer?: ReturnType<typeof setTimeout>;
  private statsTimer?: ReturnType<typeof setInterval>;
  private responseStartedAt = 0;
  private itemID = '';
  private itemStart = 0;
  private itemDuration = 0;
  private busy = false;
  private stats: VoiceStats = { sentFrames: 0, receivedFrames: 0, playedSeconds: 0, firstAudioMs: null };
  private google: boolean;

  constructor(private model: string, private callbacks: Callbacks) {
    this.google = model.startsWith('gemini-');
  }

  async start(key: string, voice: string, instructions: string) {
    try {
      const context = this.context = new AudioContext({ latencyHint: 'interactive' });
      await context.resume();
      this.callbacks.state('connecting');
      const stream = await navigator.mediaDevices.getUserMedia({ audio: { channelCount: 1, echoCancellation: true, noiseSuppression: true, autoGainControl: true } });
      if (this.closed) { stream.getTracks().forEach(track => track.stop()); return; }
      this.stream = stream;
      await context.audioWorklet.addModule('/audio/pcm-capture.js');
      if (this.closed) return;
      const inputRate = this.google ? 16000 : 24000;
      this.source = context.createMediaStreamSource(stream);
      this.capture = new AudioWorkletNode(context, 'pcm-capture', { processorOptions: { sampleRate: inputRate } });
      this.capture.port.onmessage = ({ data }: MessageEvent<ArrayBuffer>) => {
        if (!this.ready || this.closed || this.muted) return;
        if ((this.socket?.bufferedAmount || 0) > 128 * 1024) {
          this.fail('Your connection cannot keep up with live audio. Please reconnect.');
          return;
        }
        // Approximate end-of-speech on the capture side. Transcript events can
        // arrive alongside the reply, so timing from them understates latency.
        const samples = new Int16Array(data);
        let energy = 0;
        for (const sample of samples) energy += (sample / 32768) ** 2;
        if (energy / samples.length > 0.0001) this.responseStartedAt = performance.now();
        const audio = base64(data);
        this.send(this.google ? { realtimeInput: { audio: { data: audio, mimeType: 'audio/pcm;rate=16000' } } } : { type: 'input_audio_buffer.append', audio });
        this.stats.sentFrames++;
      };
      this.source.connect(this.capture);
      // Worklet output is silence. Connecting it keeps capture running in all browsers.
      this.capture.connect(context.destination);
      const url = new URL('/v1/realtime', window.location.origin);
      url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
      url.searchParams.set('model', this.model);
      const socket = this.socket = new WebSocket(url, ['openpaths-realtime', `openpaths-api-key.${key}`]);
      socket.binaryType = 'arraybuffer';
      this.timer = setTimeout(() => this.fail('The voice session did not connect. Please try again.'), 25000);
      this.statsTimer = setInterval(() => this.callbacks.stats({ ...this.stats }), 250);
      socket.onopen = () => {
        if (this.google) {
          this.send({ setup: {
            model: `models/${this.model}`,
            generationConfig: { responseModalities: ['AUDIO'], thinkingConfig: { thinkingLevel: 'LOW' }, speechConfig: { voiceConfig: { prebuiltVoiceConfig: { voiceName: voice } } } },
            systemInstruction: { parts: [{ text: instructions }] },
            inputAudioTranscription: {}, outputAudioTranscription: {},
            contextWindowCompression: { triggerTokens: 104857, slidingWindow: { targetTokens: 52428 } },
          } });
        }
      };
      socket.onmessage = event => {
        try {
          const raw = event.data instanceof ArrayBuffer ? new TextDecoder().decode(event.data) : event.data;
          if (!this.closed) this.receive(JSON.parse(raw), voice, instructions);
        } catch { this.fail('The voice service sent an unreadable response.'); }
      };
      socket.onerror = () => this.fail('Unable to connect. Check your OpenPaths key and credit balance, then try again.');
      socket.onclose = event => {
        if (!this.closed) {
          if (event.code !== 1000) this.callbacks.error('The voice connection ended unexpectedly. Please reconnect.');
          this.stop();
        }
      };
    } catch (error) {
      if (!this.closed) this.fail(error instanceof Error ? error.message : 'Unable to start your microphone.');
    }
  }

  private send(event: unknown) {
    if (this.socket?.readyState === WebSocket.OPEN) this.socket.send(JSON.stringify(event));
  }

  private connected() {
    clearTimeout(this.timer);
    this.ready = true;
    this.callbacks.state('listening');
  }

  private receive(event: any, voice: string, instructions: string) {
    if (event.error) { this.fail(event.error.message || 'The voice service returned an error.'); return; }
    if (this.google) {
      if (event.setupComplete) this.connected();
      const content = event.serverContent;
      if (content?.interrupted) this.interrupt();
      if (content?.inputTranscription?.text) {
        this.callbacks.transcript('user', content.inputTranscription.text, !!content.inputTranscription.finished);
      }
      if (content?.outputTranscription?.text) this.callbacks.transcript('assistant', content.outputTranscription.text, !!content.outputTranscription.finished);
      for (const part of content?.modelTurn?.parts || []) {
        if (part.inlineData?.mimeType?.startsWith('audio/pcm')) this.play(part.inlineData.data);
      }
      // Extended Thinking can speak several times per turn. Only interactionStatus marks idle.
      const status = content?.interactionStatus || event.interactionStatus || event.interaction_status;
      if (status) this.busy = status === 'IN_PROGRESS';
      if (content?.turnComplete) this.callbacks.transcript('assistant', '', true);
      if (!this.output.size) this.callbacks.state(this.busy ? 'thinking' : 'listening');
      if (event.goAway) this.fail('This voice session is ending. Start a new call to continue.');
      return;
    }
    switch (event.type) {
      case 'session.created':
        this.send({ type: 'session.update', session: {
          type: 'realtime', instructions, output_modalities: ['audio'], max_output_tokens: 1024,
          audio: {
            input: { format: { type: 'audio/pcm', rate: 24000 }, turn_detection: { type: 'server_vad', silence_duration_ms: 350, prefix_padding_ms: 300, create_response: true, interrupt_response: true } },
            output: { format: { type: 'audio/pcm', rate: 24000 }, voice },
          },
        } });
        break;
      case 'session.updated': this.connected(); break;
      case 'input_audio_buffer.speech_started': this.interrupt(); break;
      case 'input_audio_buffer.speech_stopped':
        this.callbacks.state('thinking');
        break;
      case 'response.created': this.busy = true; break;
      case 'response.output_audio.delta':
      case 'response.audio.delta':
        if (this.itemID !== event.item_id) {
          this.itemID = event.item_id;
          this.itemStart = Math.max(this.context!.currentTime, this.nextPlayAt);
          this.itemDuration = 0;
        }
        this.play(event.delta);
        break;
      case 'response.output_audio_transcript.delta':
      case 'response.audio_transcript.delta': this.callbacks.transcript('assistant', event.delta, false); break;
      case 'response.output_audio_transcript.done':
      case 'response.audio_transcript.done': this.callbacks.transcript('assistant', '', true); break;
      case 'conversation.item.input_audio_transcription.completed': this.callbacks.transcript('user', event.transcript, true); break;
      case 'response.done':
        this.busy = false;
        if (event.response?.status === 'failed') this.fail(event.response.status_details?.error?.message || 'Voice generation failed.');
        else if (!this.output.size) this.callbacks.state('listening');
        break;
    }
  }

  private play(encoded: string) {
    const context = this.context;
    if (!context || this.closed) return;
    const raw = atob(encoded);
    const bytes = Uint8Array.from(raw, c => c.charCodeAt(0));
    const view = new DataView(bytes.buffer);
    const buffer = context.createBuffer(1, Math.floor(bytes.length / 2), 24000);
    const samples = buffer.getChannelData(0);
    for (let i = 0; i < samples.length; i++) samples[i] = view.getInt16(i * 2, true) / 32768;
    if (!this.stats.receivedFrames && this.responseStartedAt) this.stats.firstAudioMs = Math.round(performance.now() - this.responseStartedAt);
    this.stats.receivedFrames++;
    const source = context.createBufferSource();
    source.buffer = buffer;
    source.connect(context.destination);
    const start = Math.max(context.currentTime + 0.015, this.nextPlayAt);
    this.nextPlayAt = start + buffer.duration;
    this.itemDuration += buffer.duration;
    this.output.add(source);
    source.onended = () => {
      this.stats.playedSeconds += buffer.duration;
      this.output.delete(source);
      source.disconnect();
      if (!this.closed && !this.output.size) this.callbacks.state(this.busy ? 'thinking' : 'listening');
    };
    source.start(start);
    this.callbacks.state('speaking');
  }

  private interrupt() {
    if (!this.google && this.itemID && this.context && this.output.size) {
      this.send({ type: 'conversation.item.truncate', item_id: this.itemID, content_index: 0, audio_end_ms: Math.max(0, Math.floor(Math.min(this.itemDuration, this.context.currentTime - this.itemStart) * 1000)) });
    }
    for (const source of this.output) { source.onended = null; source.stop(); source.disconnect(); }
    this.output.clear();
    this.nextPlayAt = 0;
    this.itemID = '';
    this.busy = false;
    this.callbacks.transcript('assistant', '', true);
    this.callbacks.state('listening');
  }

  setMuted(muted: boolean) {
    this.muted = muted;
    this.stream?.getAudioTracks().forEach(track => { track.enabled = !muted; });
    if (muted && this.google) this.send({ realtimeInput: { audioStreamEnd: true } });
  }

  private fail(message: string) { this.callbacks.error(message); this.stop(); }

  stop() {
    this.closed = true;
    this.ready = false;
    clearTimeout(this.timer);
    clearInterval(this.statsTimer);
    this.interrupt();
    if (this.socket) {
      this.socket.onclose = null;
      this.socket.onerror = null;
      this.socket.onmessage = null;
      this.socket.close(1000, 'Call ended');
    }
    if (this.capture) { this.capture.port.onmessage = null; this.capture.disconnect(); }
    this.source?.disconnect();
    this.stream?.getTracks().forEach(track => track.stop());
    void this.context?.close().catch(() => {});
    this.callbacks.stats({ ...this.stats });
    this.callbacks.state('idle');
  }
}
