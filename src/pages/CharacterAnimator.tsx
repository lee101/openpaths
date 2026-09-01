import React, { useEffect, useMemo, useState } from 'react';
import { Copy, Download, ExternalLink, Loader2, PersonStanding, Sparkles, Upload } from 'lucide-react';
import { ToolSeo } from '../components/ToolSeo';
import { CodeBlock } from '../components/CodeBlock';
import { normalizeUploadedAssetUrl } from '../lib/uploadUrls';

const TIERS = [
  { id: 'wan-animate', label: 'Standard', rate: 0.15 },
  { id: 'wan-animate-fast', label: 'Fast', rate: 0.3 },
  { id: 'wan-animate-xfast', label: 'X-Fast', rate: 0.6 },
];
const DEFAULT_PROMPT = '';

type APIErrorBody = { error?: { message?: string } | string };
type VideoResponse = { video_url?: string; result?: { video_url?: string }; id?: string; status?: string };

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

async function pollJob(apiKey: string, jobId: string) {
  const sleep = (ms: number) => {
    const { promise, resolve } = Promise.withResolvers<void>();
    setTimeout(resolve, ms);
    return promise;
  };
  for (let i = 0; i < 240; i++) {
    await sleep(2000);
    const resp = await fetch(`/v1/videos/generations/${encodeURIComponent(jobId)}`, {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    const data = await parseAPIResponse(resp) as VideoResponse;
    if (!resp.ok) throw new Error(getAPIErrorMessage(data, `Video status request failed (${resp.status})`));
    if (data?.status === 'completed') return data?.video_url || data?.result?.video_url || '';
    if (data?.status === 'failed') throw new Error(getAPIErrorMessage(data, 'Video generation failed'));
  }
  throw new Error('Timed out waiting for video generation');
}

export function CharacterAnimator() {
  const [apiKey, setApiKey] = useState(getStoredAPIKey);
  const [characterUrl, setCharacterUrl] = useState('');
  const [drivingUrl, setDrivingUrl] = useState('');
  const [model, setModel] = useState('wan-animate');
  const [duration, setDuration] = useState('5');
  const [prompt, setPrompt] = useState(DEFAULT_PROMPT);
  const [resultUrl, setResultUrl] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState<'character' | 'driving' | ''>('');
  const [activeSnippet, setActiveSnippet] = useState<'python' | 'javascript' | 'curl'>('python');
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (apiKey) localStorage.setItem('op_api_key', apiKey);
  }, [apiKey]);

  const durationSeconds = Math.min(8, Math.max(1, Math.round(Number(duration) || 5)));
  const body = useMemo(() => {
    const parsed: Record<string, unknown> = {
      model,
      image_url: characterUrl.trim(),
      video_url: drivingUrl.trim(),
      duration: durationSeconds,
    };
    if (prompt.trim()) parsed.prompt = prompt.trim();
    return parsed;
  }, [model, characterUrl, drivingUrl, durationSeconds, prompt]);

  const apiBase = 'https://openpaths.io/v1';
  const bearer = apiKey.trim() || 'op-...';
  const snippets = useMemo(() => {
    const json = JSON.stringify(body, null, 2);
    return {
      python: `import json
import time
from openai import OpenAI

client = OpenAI(
    api_key="${bearer}",
    base_url="${apiBase}",
)

result = client.post(
    "/videos/generations",
    body=json.loads(r'''${json}'''),
    cast_to=dict,
)

while result.get("status") not in (None, "completed", "failed"):
    time.sleep(2)
    result = client.get(
        f"/videos/generations/{result['id']}",
        cast_to=dict,
    )

if result.get("status") == "failed":
    raise RuntimeError(result.get("error", {}).get("message", "Video generation failed"))

video = result.get("result") or result
print(video["video_url"])`,
      javascript: `import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "${bearer}",
  baseURL: "${apiBase}",
});

let result = await client.post("/videos/generations", {
  body: ${json},
});

while (result.status && !["completed", "failed"].includes(result.status)) {
  await new Promise((resolve) => setTimeout(resolve, 2000));
  result = await client.get(\`/videos/generations/\${result.id}\`);
}

if (result.status === "failed") {
  throw new Error(result.error?.message || "Video generation failed");
}

console.log((result.result ?? result).video_url);`,
      curl: `curl "${apiBase}/videos/generations" \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${bearer}" \\
  -d @- <<'JSON'
${json}
JSON`,
    };
  }, [body, bearer]);

  async function uploadMedia(file: File, apply: (url: string) => void) {
    setUploading(apply === setCharacterUrl ? 'character' : 'driving');
    setError('');
    try {
      const form = new FormData();
      form.append('file', file);
      const resp = await fetch('/v1/files/upload', {
        method: 'POST',
        headers: { Authorization: `Bearer ${apiKey}` },
        body: form,
      });
      const data = await parseAPIResponse(resp) as { url?: string };
      if (!resp.ok || !data?.url) throw new Error(getAPIErrorMessage(data, 'Upload failed'));
      apply(normalizeUploadedAssetUrl(data.url));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      setUploading('');
    }
  }

  async function generate() {
    if (!apiKey) { setError('Set an OpenPaths API key first.'); return; }
    if (!characterUrl.trim()) { setError('Paste a character image URL or upload one.'); return; }
    if (!drivingUrl.trim()) { setError('Paste a driving performance video URL or upload one.'); return; }
    setLoading(true);
    setError('');
    setResultUrl('');
    try {
      const resp = await fetch('/v1/videos/generations', {
        method: 'POST',
        headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data = await parseAPIResponse(resp) as VideoResponse;
      if (!resp.ok) throw new Error(getAPIErrorMessage(data, 'Video generation failed'));
      const url = data?.video_url || data?.result?.video_url || (data?.id ? await pollJob(apiKey, data.id) : '');
      if (!url) throw new Error('No video URL returned');
      setResultUrl(url);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Video generation failed');
    } finally {
      setLoading(false);
    }
  }

  async function copySnippet() {
    await navigator.clipboard.writeText(snippets[activeSnippet]);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  }

  const rate = TIERS.find(t => t.id === model)?.rate ?? 0.15;
  const price = rate * durationSeconds;

  return (
    <>
      <ToolSeo slug="character-animator" />
      <div className="px-6 py-10">
        <div className="mx-auto max-w-7xl">
          <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <div className="mb-3 inline-flex items-center gap-2 rounded border border-white/20 bg-white/[0.06] px-3 py-1 text-xs font-mono text-white/45">
                <PersonStanding className="h-3.5 w-3.5" /> Wan-Animate · powered by ManifoldGen
              </div>
              <h1 className="max-w-3xl text-4xl font-bold tracking-tight md:text-6xl">Character Animator</h1>
              <p className="mt-4 max-w-2xl text-base leading-relaxed text-white/55">
                Hand Wan-Animate a reference character image and a driving performance video - it transfers the motion
                onto your character. Powered by ManifoldGen, OpenPaths' first-party GPU studio, with standard, fast,
                and x-fast latency lanes.
              </p>
            </div>
            <div className="grid grid-cols-3 gap-2 text-xs font-mono text-white/45">
              {TIERS.map(t => (
                <span key={t.id} className="rounded border border-white/20 px-3 py-2">${t.rate.toFixed(2)} / s · {t.label}</span>
              ))}
            </div>
          </div>

          <div className="grid gap-6 lg:grid-cols-[430px_minmax(0,1fr)]">
            <section className="rounded-lg border border-white/20 bg-white/[0.05] p-5">
              <h2 className="mb-4 font-mono text-sm font-bold uppercase tracking-wider text-white/70">Animate</h2>
              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">OpenPaths API key</span>
                <input value={apiKey} onChange={e => setApiKey(e.target.value)} placeholder="op-..." className="w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none" />
              </label>
              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Character image URL</span>
                <input value={characterUrl} onChange={e => setCharacterUrl(normalizeUploadedAssetUrl(e.target.value))} placeholder="https://.../character.jpg" className="w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none" />
              </label>
              <label className="mb-3 flex cursor-pointer items-center justify-center gap-2 rounded border border-dashed border-white/15 bg-black px-3 py-3 text-xs font-mono text-white/45 hover:text-white">
                <Upload className="h-4 w-4" />
                {uploading === 'character' ? 'Uploading...' : 'Upload a reference character image'}
                <input type="file" accept="image/*" className="hidden" onChange={e => e.target.files?.[0] && uploadMedia(e.target.files[0], setCharacterUrl)} />
              </label>
              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Driving video URL</span>
                <input value={drivingUrl} onChange={e => setDrivingUrl(normalizeUploadedAssetUrl(e.target.value))} placeholder="https://.../performance.mp4" className="w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none" />
              </label>
              <label className="mb-3 flex cursor-pointer items-center justify-center gap-2 rounded border border-dashed border-white/15 bg-black px-3 py-3 text-xs font-mono text-white/45 hover:text-white">
                <Upload className="h-4 w-4" />
                {uploading === 'driving' ? 'Uploading...' : 'Upload a driving performance video'}
                <input type="file" accept="video/*" className="hidden" onChange={e => e.target.files?.[0] && uploadMedia(e.target.files[0], setDrivingUrl)} />
              </label>
              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Refinement prompt (optional)</span>
                <textarea value={prompt} onChange={e => setPrompt(e.target.value)} rows={3} placeholder="Describe how the performance should land on the character" className="w-full resize-y rounded border border-white/20 bg-black px-3 py-2 text-sm text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none" />
              </label>
              <div className="grid grid-cols-2 gap-3">
                <label>
                  <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Latency lane</span>
                  <select value={model} onChange={e => setModel(e.target.value)} className="w-full rounded border border-white/20 bg-black px-2 py-2 text-xs font-mono text-white focus:border-white/50 focus:outline-none">
                    {TIERS.map(t => <option key={t.id} value={t.id}>{t.label} · ${t.rate.toFixed(2)}/s</option>)}
                  </select>
                </label>
                <label>
                  <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Duration (seconds)</span>
                  <input value={duration} onChange={e => setDuration(e.target.value)} inputMode="numeric" min={1} max={8} placeholder="5" className="w-full rounded border border-white/20 bg-black px-3 py-2 text-xs font-mono text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none" />
                </label>
              </div>
              <button type="button" onClick={generate} disabled={loading || uploading !== ''} className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded bg-white px-4 py-3 text-sm font-mono font-bold text-black transition-colors hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60">
                {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
                {loading ? 'Animating...' : `Animate character (~$${price.toFixed(2)})`}
              </button>
              <p className="mt-2 text-center text-[11px] font-mono text-white/40">Clamped to 1–8 seconds; bills per second at the selected lane.</p>
              {error && <p className="mt-3 rounded border border-red-400/20 bg-red-400/10 px-3 py-2 text-xs font-mono text-red-200">{error}</p>}
            </section>

            <section className="overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
              <div className="border-b border-white/20 px-4 py-3">
                <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">Result</h2>
              </div>
              <div className="flex min-h-[430px] items-center justify-center p-5">
                {resultUrl ? (
                  <div className="w-full overflow-hidden rounded border border-white/20 bg-black">
                    <video src={resultUrl} controls autoPlay className="h-auto w-full" />
                    <div className="flex border-t border-white/20 text-xs font-mono text-white/50">
                      <a href={resultUrl} download="openpaths-wan-animate.mp4" className="flex flex-1 items-center justify-center gap-2 px-3 py-2 hover:text-white"><Download className="h-3.5 w-3.5" /> Download</a>
                      <a href={resultUrl} target="_blank" rel="noopener noreferrer" className="flex flex-1 items-center justify-center gap-2 border-l border-white/20 px-3 py-2 hover:text-white"><ExternalLink className="h-3.5 w-3.5" /> Open</a>
                    </div>
                  </div>
                ) : loading ? (
                  <p className="text-sm font-mono text-white/45">Transferring the performance onto your character...</p>
                ) : (
                  <p className="max-w-md text-center text-xs font-mono text-white/50">Add a reference character image and a driving performance video above - Wan-Animate renders the animated character here.</p>
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
