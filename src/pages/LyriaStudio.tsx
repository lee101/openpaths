import React, { useEffect, useMemo, useState } from 'react';
import {
  Activity,
  ChevronDown,
  CircleDollarSign,
  Copy,
  Disc3,
  Download,
  Gauge,
  Guitar,
  Headphones,
  Loader2,
  Music2,
  Play,
  SlidersHorizontal,
  Sparkles,
  Waves,
} from 'lucide-react';
import { CodeBlock } from '../components/CodeBlock';
import { Seo } from '../components/Seo';

const API_BASE = 'https://openpaths.io/v1';
const MODELS = [
  { id: 'lyria-3-pro-preview', name: 'Lyria 3 Pro', description: 'Full songs · complex structure', duration: '1–3 min', price: 0.08 },
  { id: 'lyria-3-clip-preview', name: 'Lyria 3 Clip', description: 'Hooks, loops · fast iteration', duration: '30 sec', price: 0.04 },
] as const;
const GENRES = ['Cinematic', 'Indie pop', 'Electronic', 'Ambient', 'Hip-hop', 'Jazz', 'Rock', 'R&B', 'Folk', 'Classical', 'Experimental'];
const MOODS = ['Bittersweet', 'Euphoric', 'Mysterious', 'Hopeful', 'Melancholic', 'Tense', 'Dreamy', 'Playful', 'Triumphant', 'Intimate'];
const ENERGY = ['Low', 'Slow build', 'Medium', 'Driving', 'Explosive'];
const TEMPOS = ['Very slow · 60 BPM', 'Slow · 78 BPM', 'Midtempo · 100 BPM', 'Upbeat · 124 BPM', 'Fast · 145 BPM'];
const DURATIONS = ['60', '90', '120', '180'];

const PRESETS = [
  {
    name: 'Neon afterglow', genre: 'Electronic', mood: 'Bittersweet', energy: 'Driving', tempo: 'Upbeat · 124 BPM', mode: 'instrumental' as const,
    brief: 'A late-night synthwave drive through rain-lit streets, nostalgic but propulsive, with a huge final lift.',
    instruments: 'Pulsing analog bass, gated drums, shimmering Juno pads, glassy arpeggios, soaring lead synth',
    structure: '[0:00] Filtered pulse and distant pads\n[0:20] Beat and bass enter\n[0:50] Wide melodic hook\n[1:20] Half-time breakdown\n[1:40] Full final refrain and clean ending',
  },
  {
    name: 'Open road chorus', genre: 'Indie pop', mood: 'Hopeful', energy: 'Driving', tempo: 'Midtempo · 100 BPM', mode: 'vocal' as const,
    brief: 'Warm, human indie pop about leaving the city at dawn and choosing an uncertain new beginning.',
    instruments: 'Muted electric guitar, live drums, warm bass, handclaps, subtle Mellotron, gang vocals in the final chorus',
    structure: 'Short guitar intro, intimate verse, rising pre-chorus, memorable chorus, second verse, bridge, double final chorus',
  },
  {
    name: 'A world awakens', genre: 'Cinematic', mood: 'Triumphant', energy: 'Slow build', tempo: 'Slow · 78 BPM', mode: 'instrumental' as const,
    brief: 'An orchestral main title for the first sunrise over an undiscovered world; wonder becomes resolve.',
    instruments: 'Solo piano, low strings, French horns, evolving choir, taiko, full symphonic strings',
    structure: '[0:00] Sparse piano motif\n[0:25] Low strings reveal harmony\n[0:55] Horn theme\n[1:20] Percussion and choir build\n[1:50] Full orchestral climax\n[2:10] Resolve on solo piano',
  },
] as const;

type TrackMode = 'instrumental' | 'vocal' | 'lyrics';
type CodeLanguage = 'rest' | 'go' | 'python' | 'javascript';
type MusicResponse = {
  data?: { audio?: string; format?: string; mime_type?: string; status?: number };
  extra_info?: { music_size?: number; music_duration?: number };
  analysis_info?: { mime_type?: string };
  error?: string | { message?: string };
};

function storedAPIKey() {
  return typeof window === 'undefined' ? '' : localStorage.getItem('op_api_key') || '';
}

function Label({ children }: { children: React.ReactNode }) {
  return <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-[0.16em] text-white/40">{children}</span>;
}

function Select({ label, value, options, onChange }: { label: string; value: string; options: readonly string[]; onChange: (value: string) => void }) {
  return (
    <label className="min-w-0">
      <Label>{label}</Label>
      <div className="relative">
        <select value={value} onChange={event => onChange(event.target.value)} className="w-full appearance-none rounded border border-white/15 bg-black px-3 py-2.5 pr-8 text-xs outline-none focus:border-white/40">
          {options.map(option => <option key={option}>{option}</option>)}
        </select>
        <ChevronDown className="pointer-events-none absolute right-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-white/35" />
      </div>
    </label>
  );
}

function Spectrum({ active }: { active: boolean }) {
  const heights = [20, 35, 48, 72, 43, 86, 58, 32, 67, 91, 52, 75, 38, 62, 82, 45, 68, 29, 54, 88, 61, 36, 73, 48, 80, 57, 31, 65, 84, 44, 70, 52, 27, 59, 76, 40, 64, 30, 50, 22];
  return <div className="flex h-28 items-end justify-center gap-1">{heights.map((height, index) => <span key={index} className={`w-1 rounded-t-sm bg-white transition-opacity ${active ? 'opacity-65' : 'opacity-12'}`} style={{ height: `${height}%` }} />)}</div>;
}

function formatBytes(bytes: number) {
  if (!bytes) return '—';
  return bytes > 1024 * 1024 ? `${(bytes / 1024 / 1024).toFixed(1)} MB` : `${Math.round(bytes / 1024)} KB`;
}

export function LyriaStudio() {
  const [apiKey, setApiKey] = useState(storedAPIKey);
  const [model, setModel] = useState<(typeof MODELS)[number]['id']>('lyria-3-pro-preview');
  const [trackMode, setTrackMode] = useState<TrackMode>('instrumental');
  const [brief, setBrief] = useState(PRESETS[0].brief);
  const [genre, setGenre] = useState(PRESETS[0].genre);
  const [mood, setMood] = useState(PRESETS[0].mood);
  const [energy, setEnergy] = useState(PRESETS[0].energy);
  const [tempo, setTempo] = useState(PRESETS[0].tempo);
  const [instruments, setInstruments] = useState(PRESETS[0].instruments);
  const [vocalDirection, setVocalDirection] = useState('Intimate lead vocal in the verses, opening into layered harmonies in the chorus. Clear diction, emotionally restrained.');
  const [lyrics, setLyrics] = useState('[Verse 1]\nCity sleeping in the rear-view glow\nMorning breaking on an open road\n\n[Chorus]\nWe are heading where the skyline ends\nNothing certain, everything begins');
  const [structure, setStructure] = useState(PRESETS[0].structure);
  const [avoid, setAvoid] = useState('No abrupt cutoff, no spoken-word intro, no overly bright mastering');
  const [duration, setDuration] = useState('120');
  const [outputFormat, setOutputFormat] = useState('opus');
  const [advanced, setAdvanced] = useState(true);
  const [audioSrc, setAudioSrc] = useState('');
  const [actualFormat, setActualFormat] = useState('opus');
  const [actualMime, setActualMime] = useState('audio/ogg;codecs=opus');
  const [audioSize, setAudioSize] = useState(0);
  const [latency, setLatency] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [codeLanguage, setCodeLanguage] = useState<CodeLanguage>('rest');
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (apiKey) localStorage.setItem('op_api_key', apiKey);
  }, [apiKey]);

  const selectedModel = MODELS.find(item => item.id === model) || MODELS[0];
  const resolvedDuration = model === 'lyria-3-clip-preview' ? '30' : duration;
  const generatedPrompt = useMemo(() => {
    const vocalLine = trackMode === 'instrumental'
      ? 'Instrumental only. Do not include vocals, chanting, or spoken words.'
      : trackMode === 'vocal'
        ? `Include original sung lyrics. Vocal direction: ${vocalDirection.trim()}`
        : `Use the supplied lyrics exactly where natural. Vocal direction: ${vocalDirection.trim()}\n\n## Lyrics\n${lyrics.trim()}`;
    return `Create a ${resolvedDuration}-second ${genre.toLowerCase()} track.\n\n## Creative brief\n${brief.trim()}\n\n## Musical direction\nMood: ${mood}. Energy: ${energy}. Tempo: ${tempo}.\nInstrumentation: ${instruments.trim()}.\n${vocalLine}\n\n## Structure\n${structure.trim()}${avoid.trim() ? `\n\n## Avoid\n${avoid.trim()}` : ''}\n\nFinish with a deliberate musical ending; do not cut off mid-phrase.`;
  }, [avoid, brief, energy, genre, instruments, lyrics, mood, resolvedDuration, structure, tempo, trackMode, vocalDirection]);

  const requestBody = useMemo(() => ({ model, prompt: generatedPrompt, output_format: outputFormat }), [generatedPrompt, model, outputFormat]);
  const snippets = useMemo(() => {
    const body = JSON.stringify(requestBody, null, 2);
    const key = apiKey.trim() || 'op-...';
    return {
      rest: `curl "${API_BASE}/music/generations" \\\n+  -H "Authorization: Bearer ${key}" \\\n+  -H "Content-Type: application/json" \\\n+  -d @- <<'JSON'\n${body}\nJSON`,
      go: `package main\n\nimport (\n\t"encoding/base64"\n\t"encoding/json"\n\t"io"\n\t"net/http"\n\t"os"\n\t"strings"\n)\n\nfunc main() {\n\tbody := strings.NewReader(\`${body}\`)\n\treq, _ := http.NewRequest("POST", "${API_BASE}/music/generations", body)\n\treq.Header.Set("Authorization", "Bearer ${key}")\n\treq.Header.Set("Content-Type", "application/json")\n\n\tresp, err := http.DefaultClient.Do(req)\n\tif err != nil { panic(err) }\n\tdefer resp.Body.Close()\n\tif resp.StatusCode >= 300 {\n\t\tmessage, _ := io.ReadAll(resp.Body)\n\t\tpanic(string(message))\n\t}\n\n\tvar result struct {\n\t\tData struct {\n\t\t\tAudio string \`json:"audio"\`\n\t\t\tFormat string \`json:"format"\`\n\t\t} \`json:"data"\`\n\t}\n\tif err := json.NewDecoder(resp.Body).Decode(&result); err != nil { panic(err) }\n\taudio, _ := base64.StdEncoding.DecodeString(result.Data.Audio)\n\t_ = os.WriteFile("lyria."+result.Data.Format, audio, 0644)\n}`,
      python: `import base64\nimport json\nfrom openai import OpenAI\n\nclient = OpenAI(api_key="${key}", base_url="${API_BASE}")\nresult = client.post(\n    "/music/generations",\n    body=json.loads(r'''${body}'''),\n    cast_to=dict,\n)\n\nwith open(f"lyria.{result['data']['format']}", "wb") as f:\n    f.write(base64.b64decode(result["data"]["audio"]))`,
      javascript: `import fs from "node:fs";\nimport OpenAI from "openai";\n\nconst client = new OpenAI({ apiKey: "${key}", baseURL: "${API_BASE}" });\nconst result = await client.post("/music/generations", { body: ${body} });\nconst audio = Buffer.from(result.data.audio, "base64");\nfs.writeFileSync(\`lyria.\${result.data.format}\`, audio);`,
    };
  }, [apiKey, requestBody]);

  function loadPreset(index: number) {
    const preset = PRESETS[index];
    setGenre(preset.genre);
    setMood(preset.mood);
    setEnergy(preset.energy);
    setTempo(preset.tempo);
    setTrackMode(preset.mode);
    setBrief(preset.brief);
    setInstruments(preset.instruments);
    setStructure(preset.structure);
  }

  async function generate() {
    if (!apiKey.trim()) { setError('Add an OpenPaths API key to generate music.'); return; }
    if (brief.trim().length < 12) { setError('Give Lyria a more detailed creative brief.'); return; }
    if (trackMode === 'lyrics' && !lyrics.trim()) { setError('Add lyrics or choose another vocal mode.'); return; }
    setLoading(true);
    setError('');
    setAudioSrc('');
    const started = performance.now();
    try {
      const response = await fetch('/v1/music/generations', {
        method: 'POST',
        headers: { Authorization: `Bearer ${apiKey.trim()}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(requestBody),
      });
      const text = await response.text();
      let data: MusicResponse = {};
      try { data = text ? JSON.parse(text) : {}; } catch { throw new Error(text.slice(0, 240) || 'The API returned invalid JSON.'); }
      if (!response.ok) {
        const message = typeof data.error === 'string' ? data.error : data.error?.message;
        throw new Error(message || `Music generation failed (${response.status})`);
      }
      const audio = data.data?.audio;
      if (!audio) throw new Error('No audio was returned.');
      const format = data.data?.format || outputFormat;
      const mime = data.data?.mime_type || data.analysis_info?.mime_type || (format === 'opus' ? 'audio/ogg;codecs=opus' : `audio/${format === 'mp3' ? 'mpeg' : format}`);
      setActualFormat(format);
      setActualMime(mime);
      setAudioSize(data.extra_info?.music_size || Math.round(audio.length * 0.75));
      setAudioSrc(`data:${mime};base64,${audio}`);
      setLatency(Math.round(performance.now() - started));
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Music generation failed.');
    } finally {
      setLoading(false);
    }
  }

  async function copyCode() {
    await navigator.clipboard.writeText(snippets[codeLanguage]);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  }

  return (
    <>
      <Seo
        title="Lyria 3 Music Studio — AI Song & Instrumental Generator | OpenPaths"
        description="Generate complete songs, instrumentals, loops, and 30-second clips with Google Lyria 3 Pro and Clip. Direct genre, mood, structure, instruments, vocals, lyrics, and export as Opus."
        path="/tools/lyria"
        jsonLd={{ '@context': 'https://schema.org', '@type': 'SoftwareApplication', name: 'OpenPaths Lyria 3 Music Studio', applicationCategory: 'MultimediaApplication', operatingSystem: 'Web' }}
      />

      <div className="min-h-screen bg-[radial-gradient(circle_at_18%_0%,rgba(255,255,255,0.07),transparent_30%)] px-4 py-8 sm:px-6 lg:py-10">
        <div className="mx-auto max-w-[1450px]">
          <header className="mb-8 flex flex-col gap-5 xl:flex-row xl:items-end xl:justify-between">
            <div>
              <div className="mb-3 inline-flex items-center gap-2 rounded-full border border-white/15 bg-white/[0.055] px-3 py-1.5 text-[10px] font-mono uppercase tracking-[0.16em] text-white/55"><span className="h-1.5 w-1.5 rounded-full bg-violet-300 shadow-[0_0_10px_rgba(196,181,253,0.8)]" /> Google · Lyria 3</div>
              <h1 className="max-w-4xl text-4xl font-bold tracking-[-0.045em] sm:text-5xl lg:text-6xl">From direction to record.</h1>
              <p className="mt-3 max-w-2xl text-sm leading-relaxed text-white/50 sm:text-base">Compose full songs, cinematic scores, and tight loops. Give Lyria the musical intent and structure; OpenPaths returns a compact, ready-to-ship Opus file.</p>
            </div>
            <div className="grid grid-cols-3 gap-px overflow-hidden rounded-lg border border-white/15 bg-white/15 text-center text-[10px] font-mono uppercase tracking-[0.12em]"><div className="bg-black px-4 py-3"><span className="block text-white/30">Audio</span><span className="mt-1 block text-white/65">44.1 kHz</span></div><div className="bg-black px-4 py-3"><span className="block text-white/30">Default</span><span className="mt-1 block text-white/65">Opus</span></div><div className="bg-black px-4 py-3"><span className="block text-white/30">From</span><span className="mt-1 block text-white/65">$0.04</span></div></div>
          </header>

          <div className="grid gap-4 xl:grid-cols-[minmax(310px,0.78fr)_minmax(430px,1.24fr)_minmax(330px,0.84fr)]">
            <section className="rounded-xl border border-white/15 bg-white/[0.035] p-4 sm:p-5">
              <div className="mb-5 flex items-center justify-between"><div><div className="text-[10px] font-mono uppercase tracking-[0.18em] text-white/35">01 / Sound</div><h2 className="mt-1 text-lg font-semibold">Musical direction</h2></div><SlidersHorizontal className="h-4 w-4 text-white/35" /></div>

              <div className="mb-5 space-y-2">
                {MODELS.map(item => <button key={item.id} type="button" onClick={() => setModel(item.id)} className={`flex w-full items-center justify-between rounded-lg border p-3 text-left transition-colors ${model === item.id ? 'border-white/45 bg-white/[0.09]' : 'border-white/12 bg-black/40 hover:border-white/25'}`}><span className="flex items-center gap-3"><span className={`flex h-9 w-9 items-center justify-center rounded-full ${model === item.id ? 'bg-white text-black' : 'bg-white/[0.07] text-white/45'}`}><Disc3 className={`h-4 w-4 ${model === item.id && loading ? 'animate-spin' : ''}`} /></span><span><span className="block text-xs font-semibold">{item.name}</span><span className="mt-0.5 block text-[10px] font-mono text-white/35">{item.description}</span></span></span><span className="text-right text-[10px] font-mono"><span className="block text-white/60">${item.price.toFixed(2)}</span><span className="text-white/30">{item.duration}</span></span></button>)}
              </div>

              <div className="mb-4 grid grid-cols-3 rounded-lg border border-white/15 bg-black p-1">
                {([['instrumental', 'Instrumental'], ['vocal', 'AI vocals'], ['lyrics', 'My lyrics']] as const).map(([value, label]) => <button key={value} type="button" onClick={() => setTrackMode(value)} className={`rounded-md px-2 py-2 text-[10px] font-mono transition-colors ${trackMode === value ? 'bg-white text-black' : 'text-white/40 hover:text-white'}`}>{label}</button>)}
              </div>

              <div className="grid grid-cols-2 gap-3"><Select label="Genre" value={genre} options={GENRES} onChange={setGenre} /><Select label="Mood" value={mood} options={MOODS} onChange={setMood} /><Select label="Energy" value={energy} options={ENERGY} onChange={setEnergy} /><Select label="Tempo" value={tempo} options={TEMPOS} onChange={setTempo} /></div>
              <label className="mt-4 block"><Label>Instrumentation</Label><textarea value={instruments} onChange={event => setInstruments(event.target.value)} rows={4} className="w-full resize-y rounded border border-white/15 bg-black px-3 py-2.5 text-xs leading-relaxed outline-none focus:border-white/40" /></label>

              {trackMode !== 'instrumental' && <label className="mt-4 block"><Label>Vocal direction</Label><textarea value={vocalDirection} onChange={event => setVocalDirection(event.target.value)} rows={3} className="w-full resize-y rounded border border-white/15 bg-black px-3 py-2.5 text-xs leading-relaxed outline-none focus:border-white/40" /></label>}
              {trackMode === 'lyrics' && <label className="mt-4 block"><Label>Lyrics · use [Verse] / [Chorus] / [Bridge]</Label><textarea value={lyrics} onChange={event => setLyrics(event.target.value)} rows={9} className="w-full resize-y rounded border border-white/15 bg-black px-3 py-2.5 font-mono text-xs leading-relaxed outline-none focus:border-white/40" /></label>}
            </section>

            <section className="rounded-xl border border-white/15 bg-white/[0.035] p-4 sm:p-5">
              <div className="mb-5 flex items-center justify-between"><div><div className="text-[10px] font-mono uppercase tracking-[0.18em] text-white/35">02 / Compose</div><h2 className="mt-1 text-lg font-semibold">Creative brief</h2></div><Guitar className="h-4 w-4 text-white/35" /></div>
              <div className="mb-4 grid grid-cols-3 gap-2">{PRESETS.map((preset, index) => <button key={preset.name} type="button" onClick={() => loadPreset(index)} className="rounded-lg border border-white/12 bg-black/40 px-3 py-3 text-left transition-colors hover:border-white/35 hover:bg-white/[0.05]"><Music2 className="mb-2 h-3.5 w-3.5 text-white/35" /><span className="block text-[11px] font-medium">{preset.name}</span><span className="mt-1 block text-[9px] font-mono text-white/30">{preset.genre} · {preset.mood}</span></button>)}</div>

              <label className="block"><Label>What should this feel like?</Label><textarea value={brief} onChange={event => setBrief(event.target.value)} rows={6} className="w-full resize-y rounded-lg border border-white/15 bg-black px-4 py-3 text-sm leading-7 outline-none focus:border-white/40" /></label>
              <label className="mt-4 block"><div className="mb-1.5 flex items-center justify-between"><Label>Arrangement & timeline</Label><span className="text-[9px] font-mono text-white/25">Timestamps supported</span></div><textarea value={structure} onChange={event => setStructure(event.target.value)} rows={8} className="w-full resize-y rounded-lg border border-white/15 bg-black px-4 py-3 font-mono text-xs leading-6 outline-none focus:border-white/40" /></label>

              <button type="button" onClick={() => setAdvanced(current => !current)} className="mt-5 flex w-full items-center justify-between border-t border-white/10 pt-4 text-xs font-mono text-white/40 hover:text-white"><span>Output & advanced</span><ChevronDown className={`h-3.5 w-3.5 transition-transform ${advanced ? 'rotate-180' : ''}`} /></button>
              {advanced && <div className="mt-4 rounded-lg border border-white/12 bg-black/40 p-3.5"><div className="grid grid-cols-2 gap-3"><label><Label>Duration</Label><div className="relative"><select disabled={model === 'lyria-3-clip-preview'} value={resolvedDuration} onChange={event => setDuration(event.target.value)} className="w-full appearance-none rounded border border-white/15 bg-black px-3 py-2.5 pr-8 text-xs outline-none disabled:text-white/35">{(model === 'lyria-3-clip-preview' ? ['30'] : DURATIONS).map(seconds => <option key={seconds} value={seconds}>{seconds} seconds</option>)}</select><ChevronDown className="pointer-events-none absolute right-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-white/35" /></div></label><label><Label>File format</Label><div className="relative"><select value={outputFormat} onChange={event => setOutputFormat(event.target.value)} className="w-full appearance-none rounded border border-white/15 bg-black px-3 py-2.5 pr-8 text-xs outline-none"><option value="opus">Opus · recommended</option><option value="mp3">MP3 · source</option><option value="wav">WAV · lossless</option></select><ChevronDown className="pointer-events-none absolute right-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-white/35" /></div></label></div><label className="mt-3 block"><Label>Avoid</Label><textarea value={avoid} onChange={event => setAvoid(event.target.value)} rows={2} className="w-full resize-y rounded border border-white/15 bg-black px-3 py-2.5 text-xs outline-none focus:border-white/40" /></label><p className="mt-2 text-[9px] leading-relaxed text-white/25">Lyria emits MP3; OpenPaths encodes the default Opus export at 192 kbps VBR. WAV is also available.</p></div>}

              <div className="mt-4 rounded-lg border border-white/10 bg-white/[0.02] p-3"><button type="button" onClick={() => navigator.clipboard.writeText(generatedPrompt)} className="mb-2 flex w-full items-center justify-between text-[10px] font-mono uppercase tracking-[0.15em] text-white/35 hover:text-white"><span>Compiled Lyria prompt</span><Copy className="h-3 w-3" /></button><pre className="max-h-40 overflow-auto whitespace-pre-wrap text-[10px] leading-5 text-white/35">{generatedPrompt}</pre></div>
            </section>

            <section className="flex flex-col rounded-xl border border-white/15 bg-white/[0.035] p-4 sm:p-5">
              <div className="mb-5 flex items-center justify-between"><div><div className="text-[10px] font-mono uppercase tracking-[0.18em] text-white/35">03 / Master</div><h2 className="mt-1 text-lg font-semibold">Render track</h2></div><Headphones className="h-4 w-4 text-white/35" /></div>
              <label className="mb-4 block"><Label>OpenPaths API key</Label><input type="password" value={apiKey} onChange={event => setApiKey(event.target.value)} placeholder="op-..." autoComplete="off" className="w-full rounded border border-white/15 bg-black px-3 py-2.5 text-xs font-mono outline-none placeholder:text-white/25 focus:border-white/40" /><span className="mt-1.5 block text-[10px] text-white/25">Stored only in this browser.</span></label>
              <button type="button" onClick={generate} disabled={loading} className="flex w-full items-center justify-center gap-2 rounded-lg bg-white px-4 py-3.5 text-sm font-mono font-bold text-black hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60">{loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}{loading ? 'Composing…' : `Generate track · $${selectedModel.price.toFixed(2)}`}</button>
              <div className="mt-2 flex justify-between text-[10px] font-mono text-white/25"><span>{selectedModel.name}</span><span>{resolvedDuration}s · {outputFormat}</span></div>
              {error && <div className="mt-4 rounded-lg border border-red-400/20 bg-red-400/[0.08] px-3 py-2.5 text-xs leading-relaxed text-red-200">{error}</div>}

              <div className="mt-5 overflow-hidden rounded-xl border border-white/15 bg-black">
                <div className="flex items-center justify-between border-b border-white/10 px-4 py-3"><span className="text-[10px] font-mono uppercase tracking-[0.15em] text-white/35">Master output</span><span className={`flex items-center gap-1.5 text-[10px] font-mono ${audioSrc ? 'text-violet-200/75' : 'text-white/25'}`}><span className={`h-1.5 w-1.5 rounded-full ${audioSrc ? 'bg-violet-300' : 'bg-white/20'}`} />{audioSrc ? 'Ready' : loading ? 'Generating' : 'Waiting'}</span></div>
                <div className="p-4"><Spectrum active={Boolean(audioSrc) || loading} />{audioSrc ? <><audio controls src={audioSrc} className="mt-3 w-full" /><div className="mt-4 grid grid-cols-3 gap-px overflow-hidden rounded border border-white/10 bg-white/10 text-center text-[9px] font-mono"><div className="bg-black px-1 py-2"><span className="block text-white/25">Render</span><span className="mt-1 block text-white/60">{(latency / 1000).toFixed(1)}s</span></div><div className="bg-black px-1 py-2"><span className="block text-white/25">Size</span><span className="mt-1 block text-white/60">{formatBytes(audioSize)}</span></div><div className="bg-black px-1 py-2"><span className="block text-white/25">Codec</span><span className="mt-1 block uppercase text-white/60">{actualFormat}</span></div></div><a href={audioSrc} download={`openpaths-lyria.${actualFormat}`} className="mt-3 flex w-full items-center justify-center gap-2 rounded border border-white/15 px-3 py-2.5 text-xs font-mono text-white/55 hover:border-white/35 hover:text-white"><Download className="h-3.5 w-3.5" /> Download .{actualFormat}</a><p className="mt-2 truncate text-center text-[9px] font-mono text-white/20">{actualMime}</p></> : <div className="pb-4 text-center"><span className="mx-auto flex h-10 w-10 items-center justify-center rounded-full border border-white/10 text-white/20"><Play className="ml-0.5 h-4 w-4" /></span><p className="mt-2 text-[10px] font-mono text-white/25">Your track will appear here</p></div>}</div>
              </div>

              <div className="mt-5 grid grid-cols-3 gap-2 text-center"><div className="rounded-lg border border-white/10 p-2.5"><Gauge className="mx-auto h-3.5 w-3.5 text-white/30" /><span className="mt-1.5 block text-[9px] font-mono text-white/30">Structured</span></div><div className="rounded-lg border border-white/10 p-2.5"><Waves className="mx-auto h-3.5 w-3.5 text-white/30" /><span className="mt-1.5 block text-[9px] font-mono text-white/30">Stereo</span></div><div className="rounded-lg border border-white/10 p-2.5"><CircleDollarSign className="mx-auto h-3.5 w-3.5 text-white/30" /><span className="mt-1.5 block text-[9px] font-mono text-white/30">Per render</span></div></div>
            </section>
          </div>

          <section className="mt-4 overflow-hidden rounded-xl border border-white/15 bg-white/[0.035]">
            <div className="flex flex-col gap-3 border-b border-white/15 px-4 py-3 sm:flex-row sm:items-center sm:justify-between sm:px-5"><div><div className="flex items-center gap-1.5 text-[10px] font-mono uppercase tracking-[0.18em] text-white/35"><Activity className="h-3 w-3" /> Get code</div><h2 className="mt-0.5 text-sm font-semibold">The same track request, ready for production</h2></div><div className="flex items-center gap-1.5">{(['rest', 'go', 'python', 'javascript'] as const).map(language => <button key={language} type="button" onClick={() => setCodeLanguage(language)} className={`rounded px-3 py-1.5 text-[10px] font-mono uppercase tracking-[0.13em] ${codeLanguage === language ? 'bg-white text-black' : 'border border-white/15 text-white/40 hover:text-white'}`}>{language === 'javascript' ? 'JS' : language}</button>)}<button type="button" onClick={copyCode} className="ml-1 inline-flex items-center gap-1.5 rounded border border-white/15 px-3 py-1.5 text-[10px] font-mono uppercase tracking-[0.13em] text-white/40 hover:text-white"><Copy className="h-3 w-3" />{copied ? 'Copied' : 'Copy'}</button></div></div>
            <CodeBlock language={codeLanguage === 'rest' ? 'bash' : codeLanguage === 'javascript' ? 'javascript' : codeLanguage} label={codeLanguage === 'rest' ? 'REST / cURL' : codeLanguage === 'javascript' ? 'JavaScript' : codeLanguage === 'go' ? 'Go' : 'Python'} code={snippets[codeLanguage]} containerClassName="border-0 bg-black/50" headerClassName="hidden" preClassName="max-h-[560px] text-xs" />
          </section>
        </div>
      </div>
    </>
  );
}
