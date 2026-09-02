import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  AudioLines,
  Check,
  ChevronDown,
  Copy,
  Download,
  Loader2,
  Mic2,
  Play,
  Search,
  Sparkles,
  Users,
  WandSparkles,
  X,
} from 'lucide-react';
import { CodeBlock } from '../components/CodeBlock';
import { ToolSeo } from '../components/ToolSeo';

const MODEL_ID = 'gemini-3.1-flash-tts-preview';
const API_BASE = 'https://openpaths.io/v1';

const VOICES = [
  { name: 'Achernar', style: 'Soft', pitch: 'Higher pitch' },
  { name: 'Achird', style: 'Friendly', pitch: 'Lower middle pitch' },
  { name: 'Algenib', style: 'Gravelly', pitch: 'Lower pitch' },
  { name: 'Algieba', style: 'Smooth', pitch: 'Lower pitch' },
  { name: 'Alnilam', style: 'Firm', pitch: 'Lower middle pitch' },
  { name: 'Aoede', style: 'Breezy', pitch: 'Middle pitch' },
  { name: 'Autonoe', style: 'Bright', pitch: 'Middle pitch' },
  { name: 'Callirrhoe', style: 'Easy-going', pitch: 'Middle pitch' },
  { name: 'Charon', style: 'Informative', pitch: 'Lower pitch' },
  { name: 'Despina', style: 'Smooth', pitch: 'Middle pitch' },
  { name: 'Enceladus', style: 'Breathy', pitch: 'Lower pitch' },
  { name: 'Erinome', style: 'Clear', pitch: 'Middle pitch' },
  { name: 'Fenrir', style: 'Excitable', pitch: 'Lower middle pitch' },
  { name: 'Gacrux', style: 'Mature', pitch: 'Middle pitch' },
  { name: 'Iapetus', style: 'Clear', pitch: 'Lower middle pitch' },
  { name: 'Kore', style: 'Firm', pitch: 'Middle pitch' },
  { name: 'Laomedeia', style: 'Upbeat', pitch: 'Higher pitch' },
  { name: 'Leda', style: 'Youthful', pitch: 'Higher pitch' },
  { name: 'Orus', style: 'Firm', pitch: 'Lower middle pitch' },
  { name: 'Puck', style: 'Upbeat', pitch: 'Middle pitch' },
  { name: 'Pulcherrima', style: 'Forward', pitch: 'Middle pitch' },
  { name: 'Rasalgethi', style: 'Informative', pitch: 'Middle pitch' },
  { name: 'Sadachbia', style: 'Lively', pitch: 'Lower pitch' },
  { name: 'Sadaltager', style: 'Knowledgeable', pitch: 'Middle pitch' },
  { name: 'Schedar', style: 'Even', pitch: 'Lower middle pitch' },
  { name: 'Sulafat', style: 'Warm', pitch: 'Middle pitch' },
  { name: 'Umbriel', style: 'Easy-going', pitch: 'Lower middle pitch' },
  { name: 'Vindemiatrix', style: 'Gentle', pitch: 'Middle pitch' },
  { name: 'Zephyr', style: 'Bright', pitch: 'Higher pitch' },
  { name: 'Zubenelgenubi', style: 'Casual', pitch: 'Lower middle pitch' },
] as const;

const STYLES = ['Natural', 'Deadpan', 'Empathetic', 'Dramatic', 'Whispering', 'Excited', 'Calm', 'Authoritative', 'Playful', 'Suspicious'];
const PACES = ['Natural', 'Slow', 'Measured', 'Fast', 'Staccato', 'Urgent'];
const ACCENTS = ['American (Gen)', 'British (RP)', 'Neutral', 'Australian', 'Indian English', 'Irish', 'Scottish'];
const EMOTIONS = ['[shouting]', '[whispers]', '[caution]', '[determination]', '[pensive]', '[suspicion]', '[urgency]', '[warmly]'];
const LANGUAGES = [
  ['en-US', 'English (US)'], ['en-GB', 'English (UK)'], ['es-US', 'Spanish'], ['fr-FR', 'French'],
  ['de-DE', 'German'], ['it-IT', 'Italian'], ['pt-BR', 'Portuguese'], ['ja-JP', 'Japanese'],
  ['ko-KR', 'Korean'], ['hi-IN', 'Hindi'], ['nl-NL', 'Dutch'], ['ru-RU', 'Russian'],
] as const;

const FANTASY_TRANSCRIPT = `Speaker 1: [shouting] Halt, traveler! The northern pass is sealed by order of the council.
Speaker 2: [determination] I carry a message for the elder. Step aside, or I will force my way through.
Speaker 1: [caution] No one passes. [pensive] The elder is... he's no longer receiving visitors.
Speaker 2: [suspicion] What do you mean? We don't have time for games.
Speaker 1: It's too late. [whispers] The shadow... it reached him first. [urgency] You need to leave. [shouting] Now.`;

type Mode = 'single' | 'multi';
type CodeLanguage = 'rest' | 'python' | 'javascript';
type SpeechResponse = { audio?: string; audio_url?: string; format?: string; characters?: number; error?: { message?: string } | string };

function storedAPIKey() {
  return typeof window === 'undefined' ? '' : localStorage.getItem('op_api_key') || '';
}

function FieldLabel({ children }: { children: React.ReactNode }) {
  return <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-[0.16em] text-white/45">{children}</span>;
}

function DirectionSelect({ label, value, options, onChange }: { label: string; value: string; options: readonly string[]; onChange: (value: string) => void }) {
  return (
    <label className="min-w-0 flex-1">
      <FieldLabel>{label}</FieldLabel>
      <div className="relative">
        <select value={value} onChange={event => onChange(event.target.value)} className="w-full appearance-none rounded border border-white/15 bg-black px-3 py-2.5 pr-8 text-xs text-white outline-none transition-colors focus:border-white/40">
          {options.map(option => <option key={option}>{option}</option>)}
        </select>
        <ChevronDown className="pointer-events-none absolute right-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-white/40" />
      </div>
    </label>
  );
}

function VoicePicker({ label, value, onChange }: { label: string; value: string; onChange: (voice: string) => void }) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const selected = VOICES.find(voice => voice.name === value) || VOICES[0];
  const filtered = VOICES.filter(voice => `${voice.name} ${voice.style} ${voice.pitch}`.toLowerCase().includes(query.toLowerCase()));

  return (
    <div className="relative">
      <FieldLabel>{label}</FieldLabel>
      <button type="button" onClick={() => setOpen(current => !current)} className="flex w-full items-center justify-between rounded border border-white/15 bg-black px-3 py-2.5 text-left transition-colors hover:border-white/35">
        <span className="flex min-w-0 items-center gap-2.5">
          <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-white/10 text-white/65"><AudioLines className="h-3.5 w-3.5" /></span>
          <span className="min-w-0">
            <span className="block text-sm font-medium">{selected.name}</span>
            <span className="block truncate text-[10px] font-mono text-white/40">{selected.style} · {selected.pitch}</span>
          </span>
        </span>
        <ChevronDown className={`h-4 w-4 text-white/40 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>
      {open && (
        <div className="absolute z-30 mt-2 w-full min-w-[280px] overflow-hidden rounded-lg border border-white/20 bg-[#0a0a0a] shadow-2xl shadow-black/70">
          <div className="flex items-center gap-2 border-b border-white/15 px-3 py-2">
            <Search className="h-3.5 w-3.5 text-white/35" />
            <input autoFocus value={query} onChange={event => setQuery(event.target.value)} placeholder="Search voices" className="min-w-0 flex-1 bg-transparent py-1 text-xs outline-none placeholder:text-white/30" />
            <button type="button" onClick={() => setOpen(false)} aria-label="Close voice picker"><X className="h-3.5 w-3.5 text-white/40" /></button>
          </div>
          <div className="max-h-64 overflow-y-auto p-1.5">
            {filtered.map(voice => (
              <button key={voice.name} type="button" onClick={() => { onChange(voice.name); setOpen(false); setQuery(''); }} className={`flex w-full items-center justify-between rounded px-3 py-2 text-left hover:bg-white/[0.07] ${voice.name === value ? 'bg-white/[0.08]' : ''}`}>
                <span><span className="block text-xs font-medium">{voice.name}</span><span className="text-[10px] font-mono text-white/40">{voice.style} · {voice.pitch}</span></span>
                {voice.name === value && <Check className="h-3.5 w-3.5 text-white/65" />}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function Waveform({ active = false }: { active?: boolean }) {
  const bars = [18, 34, 56, 28, 70, 45, 82, 36, 64, 92, 48, 76, 30, 58, 86, 42, 68, 24, 52, 78, 38, 62, 28, 46, 72, 34, 56, 22, 44, 68, 30, 50, 20, 38, 26, 18];
  return (
    <div className="flex h-24 items-center justify-center gap-1" aria-hidden="true">
      {bars.map((height, index) => <span key={index} className={`w-1 rounded-full bg-white transition-opacity ${active ? 'opacity-65' : 'opacity-15'}`} style={{ height: `${height}%` }} />)}
    </div>
  );
}

export function GoogleTTS() {
  const [apiKey, setApiKey] = useState(storedAPIKey);
  const [mode, setMode] = useState<Mode>('multi');
  const [transcript, setTranscript] = useState(FANTASY_TRANSCRIPT);
  const [scene, setScene] = useState('A dark, crumbling dungeon with dripping water echoing in the distance.');
  const [context, setContext] = useState('Fantasy RPG style. Pacing is measured, snapping into urgency at the end. Tone is tense and cautious.');
  const [singleProfile, setSingleProfile] = useState('A calm, intimate documentary narrator');
  const [singleStyle, setSingleStyle] = useState('Natural');
  const [singlePace, setSinglePace] = useState('Measured');
  const [singleAccent, setSingleAccent] = useState('Neutral');
  const [singleVoice, setSingleVoice] = useState('Zephyr');
  const [speaker1Profile, setSpeaker1Profile] = useState('A stern and weary gatekeeper');
  const [speaker2Profile, setSpeaker2Profile] = useState('A determined and courageous traveler seeking answers.');
  const [speaker1Style, setSpeaker1Style] = useState('Deadpan');
  const [speaker2Style, setSpeaker2Style] = useState('Empathetic');
  const [speaker1Pace, setSpeaker1Pace] = useState('Natural');
  const [speaker2Pace, setSpeaker2Pace] = useState('Staccato');
  const [speaker1Accent, setSpeaker1Accent] = useState('British (RP)');
  const [speaker2Accent, setSpeaker2Accent] = useState('American (Gen)');
  const [speaker1Voice, setSpeaker1Voice] = useState('Fenrir');
  const [speaker2Voice, setSpeaker2Voice] = useState('Puck');
  const [language, setLanguage] = useState('en-US');
  const [temperature, setTemperature] = useState(1);
  const [autoEmotion, setAutoEmotion] = useState(false);
  const [advanced, setAdvanced] = useState(false);
  const [audioSrc, setAudioSrc] = useState('');
  const [audioFormat, setAudioFormat] = useState('wav');
  const [generatedCharacters, setGeneratedCharacters] = useState(0);
  const [latency, setLatency] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [codeLanguage, setCodeLanguage] = useState<CodeLanguage>('rest');
  const [copied, setCopied] = useState(false);
  const transcriptRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (apiKey) localStorage.setItem('op_api_key', apiKey);
  }, [apiKey]);

  const generatedPrompt = useMemo(() => {
    const setting = `${scene.trim() ? `\n\n## Scene:\n${scene.trim()}` : ''}${context.trim() ? `\n\n## Sample Context:\n${context.trim()}` : ''}`;
    if (mode === 'single') {
      return `Read the following transcript based on the audio profile and director's note.\n\n# Audio Profile\n${singleProfile.trim()}\n\n# Director's note\nStyle: ${singleStyle}. Pace: ${singlePace}. Accent: ${singleAccent}.${setting}\n\n## Transcript:\n${transcript.trim()}`;
    }
    return `Read the following transcript based on the audio profile and director's note.\n\n# Audio Profile\nFor Speaker 1: ${speaker1Profile.trim()}\nFor Speaker 2: ${speaker2Profile.trim()}\n\n# Director's note\nFor Speaker 1: Style: ${speaker1Style}. Pace: ${speaker1Pace}. Accent: ${speaker1Accent}.\nFor Speaker 2: Style: ${speaker2Style}. Pace: ${speaker2Pace}. Accent: ${speaker2Accent}.${setting}\n\n## Transcript:\n${transcript.trim()}`;
  }, [context, mode, scene, singleAccent, singlePace, singleProfile, singleStyle, speaker1Accent, speaker1Pace, speaker1Profile, speaker1Style, speaker2Accent, speaker2Pace, speaker2Profile, speaker2Style, transcript]);

  const requestBody = useMemo(() => {
    const body: Record<string, unknown> = {
      model: MODEL_ID,
      input: generatedPrompt,
      language,
      temperature,
    };
    if (mode === 'single') body.voice = singleVoice;
    else body.speaker_voices = [{ speaker: 'Speaker 1', voice: speaker1Voice }, { speaker: 'Speaker 2', voice: speaker2Voice }];
    if (autoEmotion) body.auto_emotion = true;
    return body;
  }, [autoEmotion, generatedPrompt, language, mode, singleVoice, speaker1Voice, speaker2Voice, temperature]);

  const snippets = useMemo(() => {
    const body = JSON.stringify(requestBody, null, 2);
    const key = apiKey.trim() || 'op-...';
    return {
      rest: `curl "${API_BASE}/audio/speech" \\\n+  -H "Authorization: Bearer ${key}" \\\n+  -H "Content-Type: application/json" \\\n+  -d @- <<'JSON'\n${body}\nJSON`,
      python: `import base64\nimport json\nfrom openai import OpenAI\n\nclient = OpenAI(\n    api_key="${key}",\n    base_url="${API_BASE}",\n)\n\nresult = client.post(\n    "/audio/speech",\n    body=json.loads(r'''${body}'''),\n    cast_to=dict,\n)\n\nwith open("speech.wav", "wb") as f:\n    f.write(base64.b64decode(result["audio"]))`,
      javascript: `import fs from "node:fs";\nimport OpenAI from "openai";\n\nconst client = new OpenAI({\n  apiKey: "${key}",\n  baseURL: "${API_BASE}",\n});\n\nconst result = await client.post("/audio/speech", {\n  body: ${body},\n});\n\nfs.writeFileSync("speech.wav", Buffer.from(result.audio, "base64"));`,
    };
  }, [apiKey, requestBody]);

  function addEmotion(tag: string) {
    const textarea = transcriptRef.current;
    const start = textarea?.selectionStart ?? transcript.length;
    const end = textarea?.selectionEnd ?? transcript.length;
    const before = transcript.slice(0, start);
    const spacer = before && !/\s$/.test(before) ? ' ' : '';
    const next = `${before}${spacer}${tag} ${transcript.slice(end)}`;
    setTranscript(next);
    requestAnimationFrame(() => {
      const cursor = start + spacer.length + tag.length + 1;
      textarea?.focus();
      textarea?.setSelectionRange(cursor, cursor);
    });
  }

  async function generate() {
    if (!apiKey.trim()) { setError('Add an OpenPaths API key to generate speech.'); return; }
    if (!transcript.trim()) { setError('Add a transcript first.'); return; }
    if (mode === 'multi' && (!/Speaker 1:/i.test(transcript) || !/Speaker 2:/i.test(transcript))) {
      setError('Multi-speaker transcripts need both “Speaker 1:” and “Speaker 2:” labels.');
      return;
    }
    setLoading(true);
    setError('');
    setAudioSrc('');
    const started = performance.now();
    try {
      const response = await fetch('/v1/audio/speech', {
        method: 'POST',
        headers: { Authorization: `Bearer ${apiKey.trim()}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(requestBody),
      });
      const text = await response.text();
      let data: SpeechResponse = {};
      try { data = text ? JSON.parse(text) : {}; } catch { throw new Error(text.slice(0, 240) || 'The API returned invalid JSON.'); }
      if (!response.ok) {
        const message = typeof data.error === 'string' ? data.error : data.error?.message;
        throw new Error(message || `Speech generation failed (${response.status})`);
      }
      if (!data.audio && !data.audio_url) throw new Error('No audio was returned.');
      const format = data.format || 'wav';
      setAudioFormat(format);
      setAudioSrc(data.audio_url || `data:audio/${format};base64,${data.audio}`);
      setGeneratedCharacters(data.characters || generatedPrompt.length);
      setLatency(Math.round(performance.now() - started));
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Speech generation failed.');
    } finally {
      setLoading(false);
    }
  }

  async function copyCode() {
    await navigator.clipboard.writeText(snippets[codeLanguage]);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  }

  function loadNarrationSample() {
    setMode('single');
    setSingleProfile('A warm, thoughtful nature documentary narrator');
    setSingleStyle('Calm');
    setSinglePace('Measured');
    setSingleAccent('British (RP)');
    setSingleVoice('Sulafat');
    setScene('Early morning on a remote coastline as the fog lifts from the water.');
    setContext('Intimate documentary narration. Quiet wonder, precise diction, with a gentle lift on the final sentence.');
    setTranscript('[softly] At first light, the cliffs seem almost weightless. Seabirds trace the edge of the fog, and below them, an ocean shaped by a thousand winters begins another day.');
  }

  return (
    <>
      <ToolSeo slug="google-tts" />

      <div className="min-h-screen bg-[radial-gradient(circle_at_75%_0%,rgba(255,255,255,0.07),transparent_28%)] px-4 py-8 sm:px-6 lg:py-10">
        <div className="mx-auto max-w-[1500px]">
          <header className="mb-7 flex flex-col gap-5 xl:flex-row xl:items-end xl:justify-between">
            <div>
              <div className="mb-3 inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/[0.06] px-3 py-1.5 text-[10px] font-mono uppercase tracking-[0.16em] text-white/55">
                <span className="h-1.5 w-1.5 rounded-full bg-emerald-300 shadow-[0_0_10px_rgba(110,231,183,0.8)]" /> Google · Gemini 3.1 Flash TTS Preview
              </div>
              <h1 className="max-w-4xl text-4xl font-bold tracking-[-0.04em] sm:text-5xl lg:text-6xl">Direct every voice.</h1>
              <p className="mt-3 max-w-2xl text-sm leading-relaxed text-white/50 sm:text-base">Build expressive narration and two-person scenes with natural-language direction. Shape the performance, choose from 30 voices, then ship the exact request.</p>
            </div>
            <div className="flex flex-wrap gap-2 text-[10px] font-mono uppercase tracking-[0.13em] text-white/45">
              <span className="rounded-full border border-white/15 px-3 py-2">30 voices</span>
              <span className="rounded-full border border-white/15 px-3 py-2">2 speakers</span>
              <span className="rounded-full border border-white/15 px-3 py-2">24 kHz WAV</span>
            </div>
          </header>

          <div className="grid gap-4 xl:grid-cols-[minmax(320px,0.78fr)_minmax(420px,1.25fr)_minmax(320px,0.82fr)]">
            <section className="rounded-xl border border-white/15 bg-white/[0.035] p-4 sm:p-5">
              <div className="mb-5 flex items-center justify-between">
                <div><div className="text-[10px] font-mono uppercase tracking-[0.18em] text-white/35">01 / Cast</div><h2 className="mt-1 text-lg font-semibold">Speaker settings</h2></div>
                <Users className="h-4 w-4 text-white/35" />
              </div>

              <div className="mb-5 grid grid-cols-2 rounded-lg border border-white/15 bg-black p-1">
                {(['single', 'multi'] as const).map(option => (
                  <button key={option} type="button" onClick={() => setMode(option)} className={`rounded-md px-3 py-2 text-xs font-mono capitalize transition-colors ${mode === option ? 'bg-white text-black' : 'text-white/45 hover:text-white'}`}>{option} speaker</button>
                ))}
              </div>

              {mode === 'single' ? (
                <div className="space-y-4">
                  <label className="block"><FieldLabel>Audio profile</FieldLabel><input value={singleProfile} onChange={event => setSingleProfile(event.target.value)} className="w-full rounded border border-white/15 bg-black px-3 py-2.5 text-xs outline-none focus:border-white/40" /></label>
                  <div className="grid grid-cols-2 gap-3"><DirectionSelect label="Style" value={singleStyle} options={STYLES} onChange={setSingleStyle} /><DirectionSelect label="Pace" value={singlePace} options={PACES} onChange={setSinglePace} /></div>
                  <DirectionSelect label="Accent" value={singleAccent} options={ACCENTS} onChange={setSingleAccent} />
                  <VoicePicker label="Voice" value={singleVoice} onChange={setSingleVoice} />
                </div>
              ) : (
                <div className="space-y-4">
                  <div className="rounded-lg border border-white/15 bg-black/50 p-3.5">
                    <div className="mb-3 flex items-center gap-2"><span className="flex h-5 w-5 items-center justify-center rounded-full bg-white text-[10px] font-bold text-black">1</span><span className="text-xs font-semibold">Speaker 1</span></div>
                    <label className="mb-3 block"><FieldLabel>Audio profile</FieldLabel><input value={speaker1Profile} onChange={event => setSpeaker1Profile(event.target.value)} className="w-full rounded border border-white/15 bg-black px-3 py-2 text-xs outline-none focus:border-white/40" /></label>
                    <div className="mb-3 grid grid-cols-2 gap-2"><DirectionSelect label="Style" value={speaker1Style} options={STYLES} onChange={setSpeaker1Style} /><DirectionSelect label="Pace" value={speaker1Pace} options={PACES} onChange={setSpeaker1Pace} /></div>
                    <div className="space-y-3"><DirectionSelect label="Accent" value={speaker1Accent} options={ACCENTS} onChange={setSpeaker1Accent} /><VoicePicker label="Voice" value={speaker1Voice} onChange={setSpeaker1Voice} /></div>
                  </div>
                  <div className="rounded-lg border border-white/15 bg-black/50 p-3.5">
                    <div className="mb-3 flex items-center gap-2"><span className="flex h-5 w-5 items-center justify-center rounded-full border border-white/30 text-[10px] font-bold">2</span><span className="text-xs font-semibold">Speaker 2</span></div>
                    <label className="mb-3 block"><FieldLabel>Audio profile</FieldLabel><input value={speaker2Profile} onChange={event => setSpeaker2Profile(event.target.value)} className="w-full rounded border border-white/15 bg-black px-3 py-2 text-xs outline-none focus:border-white/40" /></label>
                    <div className="mb-3 grid grid-cols-2 gap-2"><DirectionSelect label="Style" value={speaker2Style} options={STYLES} onChange={setSpeaker2Style} /><DirectionSelect label="Pace" value={speaker2Pace} options={PACES} onChange={setSpeaker2Pace} /></div>
                    <div className="space-y-3"><DirectionSelect label="Accent" value={speaker2Accent} options={ACCENTS} onChange={setSpeaker2Accent} /><VoicePicker label="Voice" value={speaker2Voice} onChange={setSpeaker2Voice} /></div>
                  </div>
                </div>
              )}
            </section>

            <section className="rounded-xl border border-white/15 bg-white/[0.035] p-4 sm:p-5">
              <div className="mb-5 flex items-center justify-between">
                <div><div className="text-[10px] font-mono uppercase tracking-[0.18em] text-white/35">02 / Script</div><h2 className="mt-1 text-lg font-semibold">Scene & transcript</h2></div>
                <div className="flex gap-2"><button type="button" onClick={loadNarrationSample} className="rounded border border-white/15 px-2.5 py-1.5 text-[10px] font-mono text-white/45 hover:text-white">Narration sample</button><button type="button" onClick={() => { setMode('multi'); setTranscript(FANTASY_TRANSCRIPT); }} className="rounded border border-white/15 px-2.5 py-1.5 text-[10px] font-mono text-white/45 hover:text-white">Scene sample</button></div>
              </div>

              <label className="mb-4 block"><FieldLabel>Scene</FieldLabel><input value={scene} onChange={event => setScene(event.target.value)} placeholder="Where is this performance happening?" className="w-full rounded border border-white/15 bg-black px-3 py-2.5 text-xs outline-none placeholder:text-white/25 focus:border-white/40" /></label>
              <label className="mb-4 block"><FieldLabel>Sample context / director's note</FieldLabel><textarea value={context} onChange={event => setContext(event.target.value)} rows={3} placeholder="Genre, overall energy, timing, emotional arc..." className="w-full resize-y rounded border border-white/15 bg-black px-3 py-2.5 text-xs leading-relaxed outline-none placeholder:text-white/25 focus:border-white/40" /></label>
              <label className="block">
                <div className="mb-1.5 flex items-center justify-between"><FieldLabel>Transcript</FieldLabel><span className="text-[10px] font-mono text-white/30">{transcript.length.toLocaleString()} characters</span></div>
                <textarea ref={transcriptRef} value={transcript} onChange={event => setTranscript(event.target.value)} rows={15} spellCheck className="w-full resize-y rounded-lg border border-white/15 bg-black px-4 py-3 text-sm leading-7 text-white/80 outline-none placeholder:text-white/25 focus:border-white/40" placeholder={mode === 'multi' ? 'Speaker 1: Welcome...\nSpeaker 2: Thanks...' : 'Enter the words to speak...'} />
              </label>
              <div className="mt-3">
                <div className="mb-2 flex items-center gap-1.5 text-[10px] font-mono uppercase tracking-[0.15em] text-white/35"><WandSparkles className="h-3 w-3" /> Insert expression</div>
                <div className="flex flex-wrap gap-1.5">{EMOTIONS.map(emotion => <button key={emotion} type="button" onClick={() => addEmotion(emotion)} className="rounded-full border border-white/15 bg-white/[0.03] px-2.5 py-1.5 text-[10px] font-mono text-white/45 hover:border-white/35 hover:text-white">{emotion}</button>)}</div>
              </div>

              <button type="button" onClick={() => setAdvanced(current => !current)} className="mt-5 flex w-full items-center justify-between border-t border-white/10 pt-4 text-xs font-mono text-white/45 hover:text-white"><span>Advanced controls</span><ChevronDown className={`h-3.5 w-3.5 transition-transform ${advanced ? 'rotate-180' : ''}`} /></button>
              {advanced && (
                <div className="mt-4 grid gap-4 rounded-lg border border-white/15 bg-black/45 p-3.5 sm:grid-cols-2">
                  <label><FieldLabel>Language</FieldLabel><div className="relative"><select value={language} onChange={event => setLanguage(event.target.value)} className="w-full appearance-none rounded border border-white/15 bg-black px-3 py-2.5 pr-8 text-xs text-white outline-none focus:border-white/40">{LANGUAGES.map(([code, name]) => <option key={code} value={code}>{name} · {code}</option>)}</select><ChevronDown className="pointer-events-none absolute right-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-white/40" /></div></label>
                  <label><FieldLabel>Temperature · {temperature.toFixed(1)}</FieldLabel><input type="range" min="0" max="2" step="0.1" value={temperature} onChange={event => setTemperature(Number(event.target.value))} className="mt-2 w-full accent-white" /></label>
                  <label className="flex items-start gap-2.5 sm:col-span-2"><input type="checkbox" checked={autoEmotion} onChange={event => setAutoEmotion(event.target.checked)} className="mt-0.5 accent-white" /><span><span className="block text-xs text-white/70">Auto-mark emotion</span><span className="mt-0.5 block text-[10px] leading-relaxed text-white/35">Let OpenPaths add expressive direction tags before synthesis. Your transcript remains unchanged.</span></span></label>
                </div>
              )}
            </section>

            <section className="flex flex-col rounded-xl border border-white/15 bg-white/[0.035] p-4 sm:p-5">
              <div className="mb-5 flex items-center justify-between">
                <div><div className="text-[10px] font-mono uppercase tracking-[0.18em] text-white/35">03 / Render</div><h2 className="mt-1 text-lg font-semibold">Generate audio</h2></div>
                <Mic2 className="h-4 w-4 text-white/35" />
              </div>

              <label className="mb-4 block"><FieldLabel>OpenPaths API key</FieldLabel><input type="password" value={apiKey} onChange={event => setApiKey(event.target.value)} placeholder="op-..." autoComplete="off" className="w-full rounded border border-white/15 bg-black px-3 py-2.5 text-xs font-mono outline-none placeholder:text-white/25 focus:border-white/40" /><span className="mt-1.5 block text-[10px] text-white/30">Stored only in this browser.</span></label>

              <button type="button" onClick={generate} disabled={loading} className="flex w-full items-center justify-center gap-2 rounded-lg bg-white px-4 py-3.5 text-sm font-mono font-bold text-black transition-all hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60">
                {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}{loading ? 'Directing performance…' : 'Generate speech'}
              </button>
              <div className="mt-2 flex justify-between text-[10px] font-mono text-white/30"><span>{mode === 'multi' ? 'Two-speaker scene' : 'Single-speaker narration'}</span><span>{MODEL_ID.replace('-preview', '')}</span></div>
              {error && <div className="mt-4 rounded-lg border border-red-400/20 bg-red-400/[0.08] px-3 py-2.5 text-xs leading-relaxed text-red-200">{error}</div>}

              <div className="mt-5 overflow-hidden rounded-xl border border-white/15 bg-black">
                <div className="flex items-center justify-between border-b border-white/10 px-4 py-3"><span className="text-[10px] font-mono uppercase tracking-[0.15em] text-white/40">Output</span><span className={`flex items-center gap-1.5 text-[10px] font-mono ${audioSrc ? 'text-emerald-300/70' : 'text-white/25'}`}><span className={`h-1.5 w-1.5 rounded-full ${audioSrc ? 'bg-emerald-300' : 'bg-white/20'}`} />{audioSrc ? 'Ready' : loading ? 'Rendering' : 'Waiting'}</span></div>
                <div className="p-4">
                  <Waveform active={Boolean(audioSrc) || loading} />
                  {audioSrc ? (
                    <>
                      <audio controls src={audioSrc} className="mt-1 w-full" />
                      <div className="mt-4 grid grid-cols-2 gap-px overflow-hidden rounded border border-white/10 bg-white/10 text-center text-[10px] font-mono"><div className="bg-black px-2 py-2"><span className="block text-white/30">Latency</span><span className="mt-0.5 block text-white/65">{(latency / 1000).toFixed(1)}s</span></div><div className="bg-black px-2 py-2"><span className="block text-white/30">Characters</span><span className="mt-0.5 block text-white/65">{generatedCharacters.toLocaleString()}</span></div></div>
                      <a href={audioSrc} download={`openpaths-gemini-tts.${audioFormat}`} className="mt-3 flex w-full items-center justify-center gap-2 rounded border border-white/15 px-3 py-2.5 text-xs font-mono text-white/55 hover:border-white/35 hover:text-white"><Download className="h-3.5 w-3.5" /> Download {audioFormat.toUpperCase()}</a>
                    </>
                  ) : (
                    <div className="pb-3 text-center"><span className="mx-auto flex h-9 w-9 items-center justify-center rounded-full border border-white/10 text-white/25"><Play className="ml-0.5 h-3.5 w-3.5" /></span><p className="mt-2 text-[10px] font-mono text-white/25">Your performance will appear here</p></div>
                  )}
                </div>
              </div>

              <div className="mt-5 rounded-lg border border-white/10 bg-white/[0.025] p-3 text-[10px] leading-relaxed text-white/35"><span className="font-mono uppercase tracking-wider text-white/55">Tip</span><p className="mt-1">Direction is part of the prompt. Keep transcript labels identical to the speaker names and use bracketed cues exactly where the delivery should change.</p></div>
            </section>
          </div>

          <section className="mt-4 overflow-hidden rounded-xl border border-white/15 bg-white/[0.035]">
            <div className="flex flex-col gap-3 border-b border-white/15 px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:px-5">
              <div><div className="text-[10px] font-mono uppercase tracking-[0.18em] text-white/35">Get code</div><h2 className="mt-0.5 text-sm font-semibold">Use this exact performance in your app</h2></div>
              <div className="flex items-center gap-1.5">
                {(['rest', 'python', 'javascript'] as const).map(languageOption => <button key={languageOption} type="button" onClick={() => setCodeLanguage(languageOption)} className={`rounded px-3 py-1.5 text-[10px] font-mono uppercase tracking-[0.13em] ${codeLanguage === languageOption ? 'bg-white text-black' : 'border border-white/15 text-white/45 hover:text-white'}`}>{languageOption === 'javascript' ? 'JS' : languageOption}</button>)}
                <button type="button" onClick={copyCode} className="ml-1 inline-flex items-center gap-1.5 rounded border border-white/15 px-3 py-1.5 text-[10px] font-mono uppercase tracking-[0.13em] text-white/45 hover:text-white"><Copy className="h-3 w-3" />{copied ? 'Copied' : 'Copy'}</button>
              </div>
            </div>
            <CodeBlock language={codeLanguage === 'rest' ? 'bash' : codeLanguage === 'javascript' ? 'javascript' : 'python'} label={codeLanguage === 'rest' ? 'REST / cURL' : codeLanguage === 'javascript' ? 'JavaScript' : 'Python'} code={snippets[codeLanguage]} containerClassName="border-0 bg-black/50" headerClassName="hidden" preClassName="max-h-[520px] text-xs" />
          </section>
        </div>
      </div>
    </>
  );
}
