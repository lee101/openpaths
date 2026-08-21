import React, { useEffect, useMemo, useState } from 'react';
import { Copy, Download, ExternalLink, Loader2, Upload, Video, Wand2 } from 'lucide-react';
import { Seo } from '../components/Seo';
import { CodeBlock } from '../components/CodeBlock';
import { normalizeUploadedAssetUrl } from '../lib/uploadUrls';

const DEFAULT_PROMPT = 'The camera slowly pulls back to reveal a glowing city skyline at dusk.';
const MODELS = [
  { id: 'grok-imagine-video', label: 'Grok Imagine Video', pricePerSecond: 0.08 },
];
const DURATIONS = ['6', '8', '10', '4', '2'];

type APIErrorBody = { error?: { message?: string } | string };
type VideoResponse = { video_url?: string; result?: { video_url?: string }; id?: string; status?: string };
type VideoMode = 'extension' | 'edit';

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
    if (!resp.ok) throw new Error(text);
    throw new Error('Invalid JSON response: ' + text.slice(0, 160));
  }
}

function getAPIErrorMessage(data: unknown, fallback: string) {
  const error = (data as APIErrorBody | null)?.error;
  if (typeof error === 'string') return error;
  return error?.message || fallback;
}

async function pollJob(apiKey: string, jobId: string, mode: VideoMode) {
  for (let i = 0; i < 90; i++) {
    await new Promise(resolve => window.setTimeout(resolve, 2000));
    const resp = await fetch(`/v1/videos/${mode === 'edit' ? 'edits' : 'extensions'}/${encodeURIComponent(jobId)}`, {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    const data = await parseAPIResponse(resp) as VideoResponse;
    const url = data?.result?.video_url || data?.video_url;
    if (url) return url;
    if (data?.status === 'failed') throw new Error(`Video ${mode} failed`);
  }
  throw new Error(`Timed out waiting for video ${mode}`);
}

export function VideoExtension() {
  const [apiKey, setApiKey] = useState(getStoredAPIKey);
  const [prompt, setPrompt] = useState(DEFAULT_PROMPT);
  const [videoUrl, setVideoUrl] = useState('');
  const [mode, setMode] = useState<VideoMode>('extension');
  const [model, setModel] = useState('grok-imagine-video');
  const [duration, setDuration] = useState('6');
  const [outputFormat, setOutputFormat] = useState<'mp4' | 'webm'>('webm');
  const [resultUrl, setResultUrl] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [activeSnippet, setActiveSnippet] = useState<'javascript' | 'curl'>('javascript');
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (apiKey) localStorage.setItem('op_api_key', apiKey);
  }, [apiKey]);

  const endpoint = mode === 'edit' ? 'edits' : 'extensions';
  const payload = useMemo(() => ({
    model,
    prompt,
    ...(mode === 'extension' ? { duration: Number(duration) } : {}),
    ...(outputFormat === 'webm' ? { output_format: 'webm' } : {}),
    video: { url: videoUrl || 'https://example.com/source.mp4' },
  }), [duration, mode, model, outputFormat, prompt, videoUrl]);

  const snippets = useMemo(() => {
    const bearer = apiKey || 'op-...';
    const body = JSON.stringify(payload, null, 2);
    return {
      javascript: `const response = await fetch("https://openpaths.io/v1/videos/${endpoint}", {
  method: "POST",
  headers: {
    "Authorization": "Bearer ${bearer}",
    "Content-Type": "application/json",
  },
  body: JSON.stringify(${body.replace(/^/gm, '  ').trim()}),
});

const data = await response.json();
console.log(data.result?.video_url ?? data.video_url ?? data.id);`,
      curl: `curl https://openpaths.io/v1/videos/${endpoint} \\
  -H "Authorization: Bearer ${bearer}" \\
  -H "Content-Type: application/json" \\
  -d '${body}'`,
    };
  }, [apiKey, endpoint, payload]);

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

  async function runVideo() {
    if (!apiKey) { setError(`Set an OpenPaths API key before ${mode === 'edit' ? 'editing' : 'extending'}.`); return; }
    if (!videoUrl.trim()) { setError('Upload a source MP4 or paste a public video URL.'); return; }
    if (!prompt.trim()) { setError(mode === 'edit' ? 'Enter the edit to apply.' : 'Enter what should happen next.'); return; }
    setLoading(true);
    setError('');
    setResultUrl('');
    try {
      const resp = await fetch(`/v1/videos/${endpoint}`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...payload, video: { url: videoUrl.trim() } }),
      });
      const data = await parseAPIResponse(resp) as VideoResponse;
      if (!resp.ok) throw new Error(getAPIErrorMessage(data, `Video ${mode} failed`));
      const url = data?.video_url || data?.result?.video_url || (data?.id ? await pollJob(apiKey, data.id, mode) : '');
      if (!url) throw new Error('No video URL returned');
      setResultUrl(url);
    } catch (err) {
      setError(err instanceof Error ? err.message : `Video ${mode} failed`);
    } finally {
      setLoading(false);
    }
  }

  async function copySnippet() {
    await navigator.clipboard.writeText(snippets[activeSnippet]);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  }

  const price = (MODELS.find(m => m.id === model)?.pricePerSecond ?? 0.08) * Number(duration || 6);
  const actionLabel = mode === 'edit' ? 'Edit' : 'Extend';

  return (
    <>
      <Seo
        title="Video Edit and Extension API | OpenPaths"
        description="Edit or extend an existing MP4 with Grok Imagine Video through OpenPaths."
        path="/video-extension"
      />
      <div className="px-6 py-10">
        <div className="mx-auto max-w-7xl">
          <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <div className="mb-3 inline-flex items-center gap-2 rounded border border-white/20 bg-white/[0.06] px-3 py-1 text-xs font-mono text-white/45">
                <Video className="h-3.5 w-3.5" /> Grok video edit and extension
              </div>
              <h1 className="max-w-3xl text-4xl font-bold tracking-tight md:text-6xl">Video Edit / Extension</h1>
              <p className="mt-4 max-w-2xl text-base leading-relaxed text-white/55">
                Modify a short MP4 or continue it from the last frame. Upload a clip or paste a public URL, describe the change, and get back one Grok Imagine video.
              </p>
            </div>
            <div className="grid grid-cols-2 gap-2 text-xs font-mono text-white/45">
              <span className="rounded border border-white/20 px-3 py-2">{model}</span>
              <span className="rounded border border-white/20 px-3 py-2">{mode === 'extension' ? `~$${price.toFixed(2)}+` : 'input based'}</span>
            </div>
          </div>

          <div className="grid gap-6 lg:grid-cols-[430px_minmax(0,1fr)]">
            <section className="rounded-lg border border-white/20 bg-white/[0.05] p-5">
              <h2 className="mb-4 font-mono text-sm font-bold uppercase tracking-wider text-white/70">{actionLabel}</h2>
              <div className="mb-3 grid grid-cols-2 gap-2 rounded border border-white/20 bg-black p-1">
                {(['extension', 'edit'] as const).map(option => (
                  <button key={option} type="button" onClick={() => setMode(option)} className={`rounded px-3 py-2 text-xs font-mono uppercase tracking-[0.14em] transition-colors ${mode === option ? 'bg-white text-black' : 'text-white/45 hover:text-white'}`}>
                    {option === 'extension' ? 'Extend' : 'Edit'}
                  </button>
                ))}
              </div>
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
                {uploading ? 'Uploading...' : 'Upload source MP4'}
                <input type="file" accept="video/mp4,video/*" className="hidden" onChange={e => e.target.files?.[0] && uploadVideo(e.target.files[0])} />
              </label>
              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">{mode === 'edit' ? 'Requested edit' : 'What happens next'}</span>
                <textarea value={prompt} onChange={e => setPrompt(e.target.value)} rows={5} className="w-full resize-y rounded border border-white/20 bg-black px-3 py-2 text-sm text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none" />
              </label>
              <div className="grid grid-cols-2 gap-3">
                <label>
                  <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Model</span>
                  <select value={model} onChange={e => setModel(e.target.value)} className="w-full rounded border border-white/20 bg-black px-2 py-2 text-xs font-mono text-white focus:border-white/50 focus:outline-none">
                    {MODELS.map(m => <option key={m.id} value={m.id}>{m.label}</option>)}
                  </select>
                </label>
                {mode === 'extension' ? <label>
                  <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Extension seconds</span>
                  <select value={duration} onChange={e => setDuration(e.target.value)} className="w-full rounded border border-white/20 bg-black px-2 py-2 text-xs font-mono text-white focus:border-white/50 focus:outline-none">
                    {DURATIONS.map(s => <option key={s} value={s}>{s}s</option>)}
                  </select>
                </label> : <div className="rounded border border-white/20 bg-black px-3 py-2 text-xs font-mono text-white/55">Duration, aspect ratio, and resolution are inherited from the input video.</div>}
              </div>
              <label className="mt-3 flex items-center justify-between gap-3 rounded border border-white/20 bg-black px-3 py-2 text-xs font-mono text-white/55">
                <span>Backend WebM reencode</span>
                <input type="checkbox" checked={outputFormat === 'webm'} onChange={e => setOutputFormat(e.target.checked ? 'webm' : 'mp4')} className="h-4 w-4 accent-white" />
              </label>
              <button type="button" onClick={runVideo} disabled={loading || uploading} className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded bg-white px-4 py-3 text-sm font-mono font-bold text-black transition-colors hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60">
                {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Wand2 className="h-4 w-4" />}
                {loading ? `${actionLabel}ing...` : mode === 'extension' ? `Extend video (~$${price.toFixed(2)}+)` : 'Edit video'}
              </button>
              {error && <p className="mt-3 rounded border border-red-400/20 bg-red-400/10 px-3 py-2 text-xs font-mono text-red-200">{error}</p>}
            </section>

            <section className="overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
              <div className="border-b border-white/20 px-4 py-3">
                <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">Result</h2>
              </div>
              <div className="flex min-h-[430px] items-center justify-center p-5">
                {resultUrl ? (
                  <div className="w-full overflow-hidden rounded border border-white/20 bg-black">
                    <video src={resultUrl} controls className="h-auto w-full" />
                    <div className="flex border-t border-white/20 text-xs font-mono text-white/50">
                      <a href={resultUrl} download={`openpaths-${mode === 'edit' ? 'edited' : 'extended'}-video.mp4`} className="flex flex-1 items-center justify-center gap-2 px-3 py-2 hover:text-white"><Download className="h-3.5 w-3.5" /> Download</a>
                      <a href={resultUrl} target="_blank" rel="noopener noreferrer" className="flex flex-1 items-center justify-center gap-2 border-l border-white/20 px-3 py-2 hover:text-white"><ExternalLink className="h-3.5 w-3.5" /> Open</a>
                    </div>
                  </div>
                ) : (
                  <p className="text-sm font-mono text-white/45">{loading ? `${actionLabel}ing video...` : `Your ${mode === 'edit' ? 'edited' : 'extended'} video will appear here.`}</p>
                )}
              </div>
            </section>
          </div>

          <section className="mt-6 min-w-0 overflow-hidden rounded-lg border border-white/20 bg-white/[0.05] p-5">
            <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
              <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">API code</h2>
              <div className="flex items-center gap-2">
                {(['javascript', 'curl'] as const).map(snippet => (
                  <button key={snippet} type="button" onClick={() => setActiveSnippet(snippet)} className={`rounded border px-2.5 py-1.5 text-[10px] font-mono uppercase tracking-[0.14em] transition-colors ${activeSnippet === snippet ? 'border-white/30 bg-white/10 text-white' : 'border-white/20 text-white/50 hover:text-white'}`}>
                    {snippet === 'javascript' ? 'JS' : snippet}
                  </button>
                ))}
                <button type="button" onClick={copySnippet} className="inline-flex items-center gap-1.5 rounded border border-white/20 px-2.5 py-1.5 text-[10px] font-mono uppercase tracking-[0.14em] text-white/45 hover:text-white">
                  <Copy className="h-3 w-3" /> {copied ? 'Copied' : 'Copy'}
                </button>
              </div>
            </div>
            <CodeBlock language={activeSnippet === 'curl' ? 'bash' : 'javascript'} label={activeSnippet === 'curl' ? 'cURL' : 'JavaScript'} code={snippets[activeSnippet]} containerClassName="overflow-hidden rounded-lg border border-white/20 bg-black/40" headerClassName="bg-white/[0.06]" preClassName="text-xs" />
          </section>
        </div>
      </div>
    </>
  );
}
