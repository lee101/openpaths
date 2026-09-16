import React, { useEffect, useRef, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { AudioLines, Mic, MicOff, Phone, PhoneOff } from 'lucide-react';
import { Seo } from '../components/Seo';
import { getApiKey, onAuthChange } from '../lib/api';
import { LIVE_MODELS, LiveVoiceSession, type VoiceState, type VoiceStats } from '../lib/liveVoice';

const EMPTY_STATS: VoiceStats = { sentFrames: 0, receivedFrames: 0, playedSeconds: 0, firstAudioMs: null };
const LABELS: Record<VoiceState, string> = { idle: 'Ready to talk', connecting: 'Connecting…', listening: 'Listening', thinking: 'Thinking', speaking: 'Speaking' };

export function LiveVoice() {
  const [params] = useSearchParams();
  const [modelID, setModelID] = useState(LIVE_MODELS.find(m => m.id === params.get('model'))?.id || LIVE_MODELS[0].id);
  const model = LIVE_MODELS.find(m => m.id === modelID)!;
  const [key, setKey] = useState(getApiKey);
  const [voice, setVoice] = useState(model.provider === 'google' ? 'Zephyr' : 'marin');
  const [instructions, setInstructions] = useState('You are a friendly voice assistant. Speak naturally and keep answers brief. Respond in the language the user speaks.');
  const [state, setState] = useState<VoiceState>('idle');
  const [muted, setMuted] = useState(false);
  const [error, setError] = useState('');
  const [stats, setStats] = useState<VoiceStats>(EMPTY_STATS);
  const [transcript, setTranscript] = useState<{ role: 'user' | 'assistant'; text: string; finished: boolean }[]>([]);
  const session = useRef<LiveVoiceSession | undefined>(undefined);
  const active = state !== 'idle';

  useEffect(() => onAuthChange(() => setKey(getApiKey())), []);
  useEffect(() => () => session.current?.stop(), []);

  function start() {
    setError(''); setMuted(false); setTranscript([]); setStats(EMPTY_STATS);
    const call = new LiveVoiceSession(modelID, {
      state: setState, error: setError, stats: setStats,
      transcript: (role, text, finished) => setTranscript(previous => {
        const last = previous[previous.length - 1];
        if (last?.role === role && !last.finished) return [...previous.slice(0, -1), { role, text: last.text + text, finished }];
        return text ? [...previous, { role, text, finished }] : previous;
      }),
    });
    session.current = call;
    void call.start(key.trim(), voice, instructions);
  }

  const fieldClass = 'w-full rounded-lg border border-white/15 bg-[#141414] px-3 py-3 text-sm outline-none focus:border-white/50 disabled:opacity-50';
  return <>
    <Seo title="Live Voice — Gemini & OpenAI Speech to Speech | OpenPaths" description="Talk naturally with Gemini 3.8 Live Extended Thinking and GPT Realtime. Streaming voice, interruptions, and transparent provider pricing plus 5%." path="/tools/live-voice" />
    <main className="mx-auto max-w-5xl px-4 py-12 sm:px-6">
      <Link to="/tools" className="text-sm text-white/50 hover:text-white">← All tools</Link>
      <div className="mt-7 flex items-center gap-3"><AudioLines className="h-8 w-8 text-emerald-300" /><h1 className="text-3xl font-semibold tracking-tight sm:text-4xl">Live voice</h1></div>
      <p className="mt-3 max-w-2xl text-white/60">Speak naturally with Gemini or OpenAI. Hear replies as they arrive, and interrupt whenever you need to.</p>
      <div className="mt-8 grid gap-6 md:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]">
        <section className="space-y-5 rounded-2xl border border-white/10 bg-white/[0.025] p-5 sm:p-6" aria-label="Voice settings">
          <label className="block space-y-2"><span className="text-sm text-white/70">Model</span>
            <select aria-label="Voice model" className={fieldClass} value={modelID} disabled={active} onChange={e => { const m = LIVE_MODELS.find(m => m.id === e.target.value)!; setModelID(m.id); setVoice(m.provider === 'google' ? 'Zephyr' : 'marin'); }}>
              {LIVE_MODELS.map(m => <option key={m.id} value={m.id}>{m.name}</option>)}
            </select>
          </label>
          <label className="block space-y-2"><span className="text-sm text-white/70">Voice</span>
            <select aria-label="Voice" className={fieldClass} value={voice} disabled={active} onChange={e => setVoice(e.target.value)}>
              {(model.provider === 'google' ? ['Zephyr', 'Puck', 'Kore', 'Aoede', 'Charon', 'Fenrir'] : ['marin', 'cedar', 'alloy', 'ash', 'coral', 'sage', 'verse']).map(v => <option key={v}>{v}</option>)}
            </select>
          </label>
          <label className="block space-y-2"><span className="text-sm text-white/70">Instructions</span>
            <textarea aria-label="Voice instructions" className={fieldClass} rows={3} value={instructions} disabled={active} onChange={e => setInstructions(e.target.value)} />
          </label>
          <label className="block space-y-2"><span className="text-sm text-white/70">OpenPaths API key</span>
            <input aria-label="OpenPaths API key" type="password" autoComplete="off" className={fieldClass} placeholder="sk-op-…" value={key} disabled={active} onChange={e => setKey(e.target.value)} />
          </label>
          {!key && <p className="text-sm text-white/50"><Link className="underline" to="/account">Sign in or get an API key</Link> to start a call.</p>}
          <div className="rounded-lg border border-white/10 p-3 text-xs leading-6 text-white/60">
            <p className="font-medium text-white/80">Provider price + 5% · per 1M tokens</p>
            <p>Audio: ${model.audioIn.toFixed(2)} input / ${model.audioOut.toFixed(2)} output</p>
            <p>Text: ${model.textIn} input / ${model.textOut} output</p>
            <p className="mt-1 text-white/40">Includes conversation context and reasoning. Usage is charged to your OpenPaths balance.</p>
          </div>
        </section>
        <section className="flex min-h-[420px] flex-col rounded-2xl border border-white/10 bg-white/[0.025] p-5 sm:p-6" aria-label="Voice conversation">
          <div className="flex flex-1 flex-col items-center justify-center py-8">
            <div className={`flex h-28 w-28 items-center justify-center rounded-full border ${active ? 'border-emerald-300/40 bg-emerald-300/10 text-emerald-300' : 'border-white/15 bg-white/5 text-white/40'} ${state === 'speaking' ? 'animate-pulse' : ''}`}><AudioLines className="h-12 w-12" /></div>
            <p role="status" className="mt-6 text-lg font-medium">{muted && active ? 'Microphone muted' : LABELS[state]}</p>
            <p className="mt-2 text-center text-sm text-white/40">{active ? 'You can speak over the assistant to interrupt.' : 'Allow microphone access when your browser asks.'}</p>
            <div className="mt-6 flex gap-3">
              {active ? <>
                <button aria-label={muted ? 'Unmute microphone' : 'Mute microphone'} onClick={() => { session.current?.setMuted(!muted); setMuted(!muted); }} className="rounded-full border border-white/20 p-3">{muted ? <MicOff className="h-5 w-5" /> : <Mic className="h-5 w-5" />}</button>
                <button onClick={() => session.current?.stop()} className="flex items-center gap-2 rounded-full bg-red-500/20 px-6 py-3 text-red-200"><PhoneOff className="h-4 w-4" /> End call</button>
              </> : <button disabled={!key.trim()} onClick={start} className="flex items-center gap-2 rounded-full bg-emerald-300 px-7 py-3 font-medium text-black disabled:opacity-40"><Phone className="h-4 w-4" /> Start call</button>}
            </div>
          </div>
          {error && <p role="alert" className="mb-4 rounded-lg border border-red-400/30 bg-red-400/10 p-3 text-sm text-red-200">{error}</p>}
          <div className="flex flex-wrap justify-center gap-x-5 gap-y-2 border-t border-white/10 pt-4 text-xs text-white/45">
            <span data-testid="sent-frames">{stats.sentFrames} mic frames sent</span>
            <span data-testid="received-frames">{stats.receivedFrames} audio frames received</span>
            {stats.firstAudioMs !== null && <span>Response: ~{stats.firstAudioMs} ms</span>}
          </div>
        </section>
      </div>
      {transcript.length > 0 && <section aria-label="Live transcript" className="mt-6 space-y-3 rounded-2xl border border-white/10 p-5">
        <h2 className="text-sm font-medium text-white/50">Live transcript</h2>
        {transcript.map((line, i) => <p key={i} className="text-sm leading-6"><span className="mr-2 text-white/40">{line.role === 'user' ? 'You' : 'Assistant'}:</span>{line.text}</p>)}
      </section>}
    </main>
  </>;
}
