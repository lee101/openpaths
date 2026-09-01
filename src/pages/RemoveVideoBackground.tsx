import React, { useEffect, useMemo, useState } from 'react';
import { Copy, Download, ExternalLink, Loader2, Scissors, Sparkles, Upload } from 'lucide-react';
import { ToolSeo } from '../components/ToolSeo';
import { CodeBlock } from '../components/CodeBlock';
import { normalizeUploadedAssetUrl } from '../lib/uploadUrls';

const MODEL_ID = 'remove-video-background';
const RATE_PER_SECOND = 0.0051;

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

export function RemoveVideoBackground() {
  const [apiKey, setApiKey] = useState(getStoredAPIKey);
  const [videoUrl, setVideoUrl] = useState('');
  const [backgroundColor, setBackgroundColor] = useState('');
  const [preserveAudio, setPreserveAudio] = useState(true);
  const [duration, setDuration] = useState('5');
  const [resultUrl, setResultUrl] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [activeSnippet, setActiveSnippet] = useState<'python' | 'javascript' | 'curl'>('python');
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (apiKey) localStorage.setItem('op_api_key', apiKey);
  }, [apiKey]);

  const durationSeconds = Math.min(30, Math.max(1, Math.round(Number(duration) || 5)));
  const body = useMemo(() => {
    const parsed: Record<string, unknown> = {
      model: MODEL_ID,
      video_url: videoUrl.trim(),
      duration: durationSeconds,
      generate_audio: preserveAudio,
    };
    if (backgroundColor.trim()) parsed.background_color = backgroundColor.trim();
    return parsed;
  }, [videoUrl, durationSeconds, preserveAudio, backgroundColor]);

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

  async function uploadVideo(file: File) {
    setUploading(true);
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
      setVideoUrl(normalizeUploadedAssetUrl(data.url));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      setUploading(false);
    }
  }

  async function generate() {
    if (!apiKey) { setError('Set an OpenPaths API key first.'); return; }
    if (!videoUrl.trim()) { setError('Paste a video URL or upload a clip.'); return; }
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

  const price = RATE_PER_SECOND * durationSeconds;

  return (
    <>
      <ToolSeo slug="remove-video-background" />
      <div className="px-6 py-10">
        <div className="mx-auto max-w-7xl">
          <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <div className="mb-3 inline-flex items-center gap-2 rounded border border-white/20 bg-white/[0.06] px-3 py-1 text-xs font-mono text-white/45">
                <Scissors className="h-3.5 w-3.5" /> Transparent video · powered by ManifoldGen
              </div>
              <h1 className="max-w-3xl text-4xl font-bold tracking-tight md:text-6xl">Background Remover</h1>
              <p className="mt-4 max-w-2xl text-base leading-relaxed text-white/55">
                Key any clip to alpha-transparent WebM in one pass - optionally compositing a solid backdrop color
                behind the subject. Powered by ManifoldGen, OpenPaths' first-party GPU studio.
              </p>
            </div>
            <div className="grid grid-cols-1 gap-2 text-xs font-mono text-white/45">
              <span className="rounded border border-white/20 px-3 py-2">~$0.005 / sec · alpha WebM out</span>
            </div>
          </div>

          <div className="grid gap-6 lg:grid-cols-[430px_minmax(0,1fr)]">
            <section className="rounded-lg border border-white/20 bg-white/[0.05] p-5">
              <h2 className="mb-4 font-mono text-sm font-bold uppercase tracking-wider text-white/70">Remove background</h2>
              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">OpenPaths API key</span>
                <input value={apiKey} onChange={e => setApiKey(e.target.value)} placeholder="op-..." className="w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none" />
              </label>
              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Source video URL</span>
                <input value={videoUrl} onChange={e => setVideoUrl(normalizeUploadedAssetUrl(e.target.value))} placeholder="https://.../clip.mp4" className="w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none" />
              </label>
              <label className="mb-3 flex cursor-pointer items-center justify-center gap-2 rounded border border-dashed border-white/15 bg-black px-3 py-3 text-xs font-mono text-white/45 hover:text-white">
                <Upload className="h-4 w-4" />
                {uploading ? 'Uploading...' : 'Upload, paste, or drop a source clip'}
                <input type="file" accept="video/*" className="hidden" onChange={e => e.target.files?.[0] && uploadVideo(e.target.files[0])} />
              </label>
              <div className="grid grid-cols-2 gap-3">
                <label>
                  <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Backdrop color (optional)</span>
                  <span className="flex items-center gap-2">
                    <input type="color" value={backgroundColor || '#22c55e'} onChange={e => setBackgroundColor(e.target.value)} className="h-9 w-10 cursor-pointer rounded border border-white/20 bg-black p-1" aria-label="Pick backdrop color" />
                    <input value={backgroundColor} onChange={e => setBackgroundColor(e.target.value)} placeholder="#22c55e" className="w-full rounded border border-white/20 bg-black px-2 py-2 text-xs font-mono text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none" />
                  </span>
                </label>
                <label>
                  <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Duration (seconds)</span>
                  <input value={duration} onChange={e => setDuration(e.target.value)} inputMode="numeric" min={1} max={30} placeholder="5" className="w-full rounded border border-white/20 bg-black px-3 py-2 text-xs font-mono text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none" />
                </label>
              </div>
              <label className="mt-3 flex items-center justify-between gap-3 rounded border border-white/20 bg-black px-3 py-2 text-xs font-mono text-white/55">
                <span>Keep original audio</span>
                <input type="checkbox" checked={preserveAudio} onChange={e => setPreserveAudio(e.target.checked)} className="h-4 w-4 accent-white" />
              </label>
              {backgroundColor.trim() && (
                <p className="mt-3 text-center text-[11px] font-mono text-white/40">Leave backdrop empty for alpha transparency; pick a swatch to composite a solid color.</p>
              )}
              <button type="button" onClick={generate} disabled={loading || uploading} className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded bg-white px-4 py-3 text-sm font-mono font-bold text-black transition-colors hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60">
                {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
                {loading ? 'Keying...' : `Remove background (~$${price.toFixed(2)})`}
              </button>
              <p className="mt-2 text-center text-[11px] font-mono text-white/40">Clamped to 1–30 seconds; bills per second of output.</p>
              {error && <p className="mt-3 rounded border border-red-400/20 bg-red-400/10 px-3 py-2 text-xs font-mono text-red-200">{error}</p>}
            </section>

            <section className="overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
              <div className="border-b border-white/20 px-4 py-3">
                <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">Result</h2>
              </div>
              <div className="flex min-h-[430px] items-center justify-center p-5">
                {resultUrl ? (
                  <div className="w-full overflow-hidden rounded border border-white/20">
                    <video src={resultUrl} controls autoPlay className="h-auto w-full" style={{ backgroundImage: 'repeating-conic-gradient(#3f3f46 0% 25%, #27272a 0% 50%)', backgroundSize: '24px 24px' }} />
                    <div className="flex border-t border-white/20 bg-black text-xs font-mono text-white/50">
                      <a href={resultUrl} download="openpaths-background-removal.webm" className="flex flex-1 items-center justify-center gap-2 px-3 py-2 hover:text-white"><Download className="h-3.5 w-3.5" /> Download</a>
                      <a href={resultUrl} target="_blank" rel="noopener noreferrer" className="flex flex-1 items-center justify-center gap-2 border-l border-white/20 px-3 py-2 hover:text-white"><ExternalLink className="h-3.5 w-3.5" /> Open</a>
                    </div>
                  </div>
                ) : loading ? (
                  <p className="text-sm font-mono text-white/45">Keying your clip...</p>
                ) : (
                  <p className="max-w-md text-center text-xs font-mono text-white/50">Drop a clip above - the keyed result renders over a checkerboard so you can verify the alpha channel.</p>
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
