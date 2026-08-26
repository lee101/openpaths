import React, { useEffect, useMemo, useState } from 'react';
import { Copy, Download, Loader2, Music, Sparkles } from 'lucide-react';
import { Seo } from '../components/Seo';
import { CodeBlock } from '../components/CodeBlock';

const MODEL_ID = 'mg-music';
const DEFAULT_PROMPT =
  'An upbeat indie rock track with warm electric guitars, driving drums, and a catchy wordless chorus hook.';
const MIN_DURATION = 30;
const MAX_DURATION = 300;

type APIErrorBody = { error?: { message?: string } | string };
type MusicResponse = { data?: { audio?: string }; base_resp?: { status_msg?: string } };

function getStoredAPIKey() {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem('op_api_key') || '';
}

async function parseAPIResponse(resp: Response) {
  const text = await resp.text();
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch {
    throw new Error(resp.ok ? 'The API returned invalid JSON.' : text.slice(0, 240));
  }
}

function getAPIErrorMessage(data: unknown, fallback: string) {
  const error = (data as APIErrorBody | null)?.error;
  if (typeof error === 'string') return error;
  return error?.message || fallback;
}

export function MusicGenerator() {
  const [apiKey, setApiKey] = useState(getStoredAPIKey);
  const [prompt, setPrompt] = useState(DEFAULT_PROMPT);
  const [lyrics, setLyrics] = useState('');
  const [duration, setDuration] = useState('60');
  const [audioUrl, setAudioUrl] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [activeSnippet, setActiveSnippet] = useState<'python' | 'javascript' | 'curl'>('python');
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (apiKey) localStorage.setItem('op_api_key', apiKey);
  }, [apiKey]);

  const durationSeconds = Math.min(MAX_DURATION, Math.max(MIN_DURATION, Math.round(Number(duration) || 60)));
  const body = useMemo(() => {
    const parsed: Record<string, unknown> = {
      model: MODEL_ID,
      prompt: prompt.trim(),
      duration: durationSeconds,
    };
    if (lyrics.trim()) parsed.lyrics = lyrics.trim();
    return parsed;
  }, [prompt, lyrics, durationSeconds]);

  const apiBase = 'https://openpaths.io/v1';
  const bearer = apiKey.trim() || 'op-...';
  const snippets = useMemo(() => {
    const json = JSON.stringify(body, null, 2);
    return {
      python: `import json
from openai import OpenAI

client = OpenAI(
    api_key="${bearer}",
    base_url="${apiBase}",
)

result = client.post(
    "/music/generations",
    body=json.loads(r'''${json}'''),
    cast_to=dict,
)

print(result["data"]["audio"])`,
      javascript: `import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "${bearer}",
  baseURL: "${apiBase}",
});

const result = await client.post("/music/generations", {
  body: ${json},
});

console.log(result.data.audio);`,
      curl: `curl "${apiBase}/music/generations" \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${bearer}" \\
  -d @- <<'JSON'
${json}
JSON`,
    };
  }, [body, bearer]);

  async function generate() {
    if (!apiKey) { setError('Set an OpenPaths API key first.'); return; }
    if (prompt.trim().length < 10) { setError('Describe the track in at least 10 characters.'); return; }
    setLoading(true);
    setError('');
    setAudioUrl('');
    try {
      const resp = await fetch('/v1/music/generations', {
        method: 'POST',
        headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data = await parseAPIResponse(resp) as MusicResponse;
      if (!resp.ok) throw new Error(getAPIErrorMessage(data, 'Music generation failed'));
      const audio = data?.data?.audio || '';
      if (!audio) throw new Error('No audio returned');
      setAudioUrl(audio);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Music generation failed');
    } finally {
      setLoading(false);
    }
  }

  async function copySnippet() {
    await navigator.clipboard.writeText(snippets[activeSnippet]);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  }

  const price = Math.max(0.35, 0.25 + 0.15 * (durationSeconds / 60));

  return (
    <>
      <Seo
        title="Music Generator — OpenPaths"
        description="Generate full songs with vocals from a prompt and optional lyrics through MiniMax-Music3 on ManifoldGen, OpenPaths' first-party GPU studio. Pin any length from 30 to 300 seconds."
        path="/music-generator"
      />
      <div className="px-6 py-10">
        <div className="mx-auto max-w-7xl">
          <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <div className="mb-3 inline-flex items-center gap-2 rounded border border-white/20 bg-white/[0.06] px-3 py-1 text-xs font-mono text-white/45">
                <Music className="h-3.5 w-3.5" /> MiniMax-Music3 · powered by ManifoldGen
              </div>
              <h1 className="max-w-3xl text-4xl font-bold tracking-tight md:text-6xl">Music Generator</h1>
              <p className="mt-4 max-w-2xl text-base leading-relaxed text-white/55">
                Describe a track and get a full song with vocals — add your own lyrics with [Verse]/[Chorus] markers,
                or let the model improvise. Powered by ManifoldGen, OpenPaths' first-party GPU studio.
              </p>
            </div>
            <div className="grid grid-cols-1 gap-2 text-xs font-mono text-white/45">
              <span className="rounded border border-white/20 px-3 py-2">$0.25 base + $0.15 / output minute · $0.35 minimum</span>
            </div>
          </div>

          <div className="grid gap-6 lg:grid-cols-[430px_minmax(0,1fr)]">
            <section className="rounded-lg border border-white/20 bg-white/[0.05] p-5">
              <h2 className="mb-4 font-mono text-sm font-bold uppercase tracking-wider text-white/70">Compose</h2>
              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">OpenPaths API key</span>
                <input value={apiKey} onChange={e => setApiKey(e.target.value)} placeholder="op-..." className="w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none" />
              </label>
              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Track prompt (min 10 characters)</span>
                <textarea value={prompt} onChange={e => setPrompt(e.target.value)} rows={4} placeholder="Genre, instruments, mood, tempo..." className="w-full resize-y rounded border border-white/20 bg-black px-3 py-2 text-sm text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none" />
              </label>
              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Lyrics (optional · [Verse] / [Chorus])</span>
                <textarea value={lyrics} onChange={e => setLyrics(e.target.value)} rows={6} placeholder={'[Verse]\nLine one\n\n[Chorus]\nHook line'} className="w-full resize-y rounded border border-white/20 bg-black px-3 py-2 text-sm text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none" />
              </label>
              <label className="block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Duration · {durationSeconds}s</span>
                <input type="range" min={MIN_DURATION} max={MAX_DURATION} step={5} value={durationSeconds} onChange={e => setDuration(e.target.value)} className="w-full accent-white" />
                <span className="mt-1 flex justify-between text-[10px] font-mono text-white/40"><span>30s</span><span>300s</span></span>
              </label>
              <button type="button" onClick={generate} disabled={loading} className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded bg-white px-4 py-3 text-sm font-mono font-bold text-black transition-colors hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60">
                {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
                {loading ? 'Composing...' : `Generate song (~$${price.toFixed(2)})`}
              </button>
              <p className="mt-2 text-center text-[11px] font-mono text-white/40">Billed at $0.25 base + $0.15 per output minute, $0.35 minimum.</p>
              {error && <p className="mt-3 rounded border border-red-400/20 bg-red-400/10 px-3 py-2 text-xs font-mono text-red-200">{error}</p>}
            </section>

            <section className="overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
              <div className="border-b border-white/20 px-4 py-3">
                <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">Result</h2>
              </div>
              <div className="flex min-h-[430px] items-center justify-center p-5">
                {audioUrl ? (
                  <div className="w-full overflow-hidden rounded border border-white/20 bg-black p-5">
                    <audio controls src={audioUrl} className="w-full" />
                    <a href={audioUrl} download="openpaths-mini-max-music3.mp3" className="mt-4 flex items-center justify-center gap-2 border-t border-white/20 pt-4 text-xs font-mono text-white/50 hover:text-white"><Download className="h-3.5 w-3.5" /> Download audio</a>
                  </div>
                ) : loading ? (
                  <p className="text-sm font-mono text-white/45">Composing your track...</p>
                ) : (
                  <p className="max-w-md text-center text-xs font-mono text-white/50">Describe a song above — MiniMax-Music3 renders full vocals and instrumentation here.</p>
                )}
              </div>
            </section>
          </div>

          <section className="mt-6 min-w-0 overflow-hidden rounded-lg border border-white/20 bg-white/[0.05] p-5">
            <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
              <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">API code</h2>
              <div className="flex items-center gap-2">
                {(['python', 'javascript', 'curl'] as const).map(snippet => (
                  <button key={snippet} type="button" onClick={() => setActiveSnippet(snippet)} className={`rounded border px-2.5 py-1.5 text-[10px] font-mono uppercase tracking-[0.14em] transition-colors ${activeSnippet === snippet ? 'border-white/30 bg-white/10 text-white' : 'border-white/20 text-white/50 hover:text-white'}`}>
                    {snippet === 'javascript' ? 'JS' : snippet}
                  </button>
                ))}
                <button type="button" onClick={copySnippet} className="inline-flex items-center gap-1.5 rounded border border-white/20 px-2.5 py-1.5 text-[10px] font-mono uppercase tracking-[0.14em] text-white/45 hover:text-white">
                  <Copy className="h-3 w-3" /> {copied ? 'Copied' : 'Copy'}
                </button>
              </div>
            </div>
            <CodeBlock language={activeSnippet === 'curl' ? 'bash' : activeSnippet === 'javascript' ? 'javascript' : 'python'} label={activeSnippet === 'curl' ? 'cURL' : activeSnippet === 'javascript' ? 'JavaScript' : 'Python'} code={snippets[activeSnippet]} containerClassName="overflow-hidden rounded-lg border border-white/20 bg-black/40" headerClassName="bg-white/[0.06]" preClassName="text-xs" />
          </section>
        </div>
      </div>
    </>
  );
}
