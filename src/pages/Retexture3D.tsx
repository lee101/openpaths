import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Copy, Download, ExternalLink, Loader2, Palette, Upload, Wand2 } from 'lucide-react';
import { Seo } from '../components/Seo';
import { CodeBlock } from '../components/CodeBlock';
import { ModelViewer } from '../components/ModelViewer';
import { Model3DUpload } from '../components/Model3DUpload';
import { prepareUploadFile } from '../lib/imageUpload';
import { normalizeUploadedAssetUrl } from '../lib/uploadUrls';

const DEFAULT_MESH_URL = 'https://threejs.org/examples/models/gltf/Soldier.glb';
const DEFAULT_IMAGE_URL = 'https://openpathsstatic.openpaths.io/static/uploads/retexture-3d/armor-reference.jpg';

type RetextureResult = {
  model_glb?: { url?: string };
  seed?: number;
  billing?: { external_cost_usd?: number };
  model?: string;
};

type RetextureJob = {
  id?: string;
  status?: string;
  result?: RetextureResult;
  error?: { message?: string } | string;
};

function getStoredAPIKey() {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem('op_api_key') || '';
}

async function parseAPIResponse(resp: Response) {
  const text = await resp.text();
  if (!text) return null;
  try { return JSON.parse(text); } catch {
    if (!resp.ok) throw new Error(text);
    throw new Error('Invalid JSON response: ' + text.slice(0, 160));
  }
}

function getAPIErrorMessage(data: unknown, fallback: string) {
  const error = (data as { error?: { message?: string } | string } | null)?.error;
  if (typeof error === 'string') return error;
  return error?.message || fallback;
}

export function Retexture3D() {
  const [apiKey, setApiKey] = useState(getStoredAPIKey);
  const [meshUrl, setMeshUrl] = useState(DEFAULT_MESH_URL);
  const [imageUrl, setImageUrl] = useState(DEFAULT_IMAGE_URL);
  const [resolution, setResolution] = useState(1024);
  const [activeSnippet, setActiveSnippet] = useState<'python' | 'javascript' | 'curl'>('python');
  const [result, setResult] = useState<RetextureResult | null>(null);
  const [outputUrl, setOutputUrl] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [uploadingImage, setUploadingImage] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (apiKey) localStorage.setItem('op_api_key', apiKey);
  }, [apiKey]);

  const price = resolution === 512 ? 0.20 : 0.24;

  const payload = useMemo(() => ({
    model: 'trellis-2-retexture',
    image_url: imageUrl,
    mesh_url: meshUrl,
    resolution,
  }), [imageUrl, meshUrl, resolution]);

  const snippets = useMemo(() => {
    const bearer = apiKey || 'op-...';
    const body = JSON.stringify(payload, null, 2);
    return {
      python: `import requests

resp = requests.post(
    "https://openpaths.io/v1/3d/generations",
    headers={"Authorization": "Bearer ${bearer}", "Content-Type": "application/json"},
    json=${body.replace(/\n/g, '\n    ')},
)
resp.raise_for_status()
print(resp.json()["model_glb"]["url"])`,
      javascript: `const resp = await fetch("https://openpaths.io/v1/3d/generations", {
  method: "POST",
  headers: {
    "Authorization": "Bearer ${bearer}",
    "Content-Type": "application/json",
  },
  body: JSON.stringify(${body.replace(/^/gm, '  ').trim()}),
});

if (!resp.ok) throw new Error(await resp.text());
const data = await resp.json();
console.log(data.model_glb.url);`,
      curl: `curl https://openpaths.io/v1/3d/generations \\
  -H "Authorization: Bearer ${bearer}" \\
  -H "Content-Type: application/json" \\
  -d '${body}'`,
    };
  }, [apiKey, payload]);

  const uploadReference = useCallback(async (file: File) => {
    if (!apiKey) { setError('Set an OpenPaths API key before uploading.'); return; }
    setUploadingImage(true);
    setError('');
    try {
      const uploadFile = await prepareUploadFile(file);
      const form = new FormData();
      form.append('file', uploadFile);
      const resp = await fetch('/v1/files/upload', { method: 'POST', headers: { Authorization: `Bearer ${apiKey}` }, body: form });
      const data = await parseAPIResponse(resp);
      if (!resp.ok) throw new Error(data?.error?.message || data?.error || 'Upload failed');
      setImageUrl(normalizeUploadedAssetUrl(data.url));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      setUploadingImage(false);
    }
  }, [apiKey]);

  async function pollJob(jobId: string): Promise<RetextureResult> {
    for (let attempt = 0; attempt < 180; attempt += 1) {
      await new Promise(resolve => window.setTimeout(resolve, attempt < 5 ? 1500 : 3000));
      const resp = await fetch(`/v1/3d/generations/${encodeURIComponent(jobId)}`, { headers: { Authorization: `Bearer ${apiKey}` } });
      const job = await parseAPIResponse(resp) as RetextureJob;
      if (!resp.ok) throw new Error(getAPIErrorMessage(job, 'retexture status check failed'));
      if (job.status === 'completed' && job.result) return job.result;
      if (job.status === 'failed') {
        const message = typeof job.error === 'string' ? job.error : job.error?.message;
        throw new Error(message || 'retexture failed');
      }
    }
    throw new Error('Retexture is still running. Retry the same request or poll the job URL again.');
  }

  async function runRetexture() {
    if (!apiKey) { setError('Set an OpenPaths API key before retexturing.'); return; }
    if (!meshUrl) { setError('Upload a mesh (.glb) or paste a public mesh URL first.'); return; }
    if (!imageUrl) { setError('Add a reference image to guide the texture.'); return; }
    setLoading(true);
    setError('');
    try {
      const resp = await fetch('/v1/3d/generations', {
        method: 'POST',
        headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const data = await parseAPIResponse(resp) as RetextureResult | RetextureJob;
      if (!resp.ok) throw new Error(getAPIErrorMessage(data, 'retexture failed'));
      const finalResult = resp.status === 202 && 'id' in data && data.id ? await pollJob(data.id) : data as RetextureResult;
      setResult(finalResult);
      if (finalResult?.model_glb?.url) setOutputUrl(finalResult.model_glb.url);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'retexture failed');
    } finally {
      setLoading(false);
    }
  }

  async function copySnippet() {
    await navigator.clipboard.writeText(snippets[activeSnippet]);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  }

  const previewUrl = outputUrl || DEFAULT_MESH_URL;

  return (
    <>
      <Seo
        title="3D Retexture API | OpenPaths"
        description="Re-texture an existing 3D mesh from a reference image — upload a GLB and a style image, get back a textured GLB, via OpenPaths and Fal Trellis-2."
        path="/retexture-3d"
      />

      <div className="px-6 py-10">
        <div className="mx-auto max-w-7xl">
          <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <div className="mb-3 inline-flex items-center gap-2 rounded border border-white/20 bg-white/[0.06] px-3 py-1 text-xs font-mono text-white/45">
                <Palette className="h-3.5 w-3.5" /> Fal Trellis-2
              </div>
              <h1 className="max-w-3xl text-4xl font-bold tracking-tight md:text-6xl">3D Retexture</h1>
              <p className="mt-4 max-w-2xl text-base leading-relaxed text-white/55">
                Re-skin an existing mesh: upload a 3D model and a reference image, and Trellis-2 paints a fresh texture onto it — preview the result in the browser.
              </p>
            </div>
            <div className="grid grid-cols-2 gap-2 text-xs font-mono text-white/45 sm:grid-cols-3">
              <span className="rounded border border-white/20 px-3 py-2">512p: $0.20</span>
              <span className="rounded border border-white/20 px-3 py-2">1024p: $0.24</span>
              <span className="rounded border border-white/20 px-3 py-2">GLB output</span>
            </div>
          </div>

          <div className="grid gap-6 lg:grid-cols-[420px_minmax(0,1fr)]">
            <section className="rounded-lg border border-white/20 bg-white/[0.05] p-5">
              <div className="mb-4 flex items-center justify-between">
                <button type="button" onClick={() => { setMeshUrl(DEFAULT_MESH_URL); setImageUrl(DEFAULT_IMAGE_URL); setResult(null); setOutputUrl(''); }} className="text-xs font-mono text-white/55 hover:text-white">
                  defaults
                </button>
              </div>

              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">OpenPaths API key</span>
                <input value={apiKey} onChange={e => setApiKey(e.target.value)} placeholder="op-..." className="w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none" />
              </label>

              <div className="mb-3">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Mesh (.glb / .gltf)</span>
                <Model3DUpload apiKey={apiKey} onUploaded={setMeshUrl} minHeight={220} />
              </div>

              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">…or paste a public mesh URL</span>
                <input value={meshUrl} onChange={e => setMeshUrl(normalizeUploadedAssetUrl(e.target.value))} className="w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white focus:border-white/50 focus:outline-none" />
              </label>

              <label className="mb-2 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Reference image (texture style)</span>
                <input value={imageUrl} onChange={e => setImageUrl(normalizeUploadedAssetUrl(e.target.value))} className="w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white focus:border-white/50 focus:outline-none" />
              </label>
              <label className="mb-3 flex cursor-pointer items-center gap-3 rounded border border-dashed border-white/20 bg-black/50 px-3 py-3 text-xs font-mono text-white/45 hover:border-white/40">
                <Upload className="h-4 w-4" />
                <span>{uploadingImage ? 'Uploading image…' : 'Upload or drop a reference image'}</span>
                <input type="file" accept="image/*" disabled={uploadingImage} onChange={e => e.target.files?.[0] && uploadReference(e.target.files[0])} className="hidden" />
              </label>
              {imageUrl && (
                <div className="mb-3 flex items-center gap-3 rounded border border-white/20 bg-white/[0.05] p-2">
                  <img src={imageUrl} alt="Reference" className="h-14 w-14 flex-none rounded border border-white/20 bg-black object-cover" />
                  <a href={imageUrl} target="_blank" rel="noreferrer" className="min-w-0 truncate text-xs font-mono text-white/45 hover:text-white">{imageUrl}</a>
                </div>
              )}

              <label className="mb-1 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Resolution</span>
                <select value={resolution} onChange={e => setResolution(Number(e.target.value))} className="w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white focus:border-white/50 focus:outline-none">
                  <option value={512}>512p — $0.20</option>
                  <option value={1024}>1024p — $0.24</option>
                </select>
              </label>

              <button type="button" onClick={runRetexture} disabled={loading} className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded bg-white px-4 py-3 text-sm font-mono font-bold text-black transition-colors hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60">
                {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Wand2 className="h-4 w-4" />}
                {loading ? 'Retexturing…' : `Retexture ($${price.toFixed(2)})`}
              </button>

              {error && <p className="mt-3 rounded border border-red-400/20 bg-red-400/10 px-3 py-2 text-xs font-mono text-red-200">{error}</p>}
            </section>

            <section className="overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
              <div className="flex items-center justify-between border-b border-white/20 px-4 py-3">
                <div>
                  <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">Retextured result</h2>
                  <p className="mt-1 text-xs font-mono text-white/50">{outputUrl ? 'Retextured mesh' : 'Example — input soldier mesh + armor reference'}</p>
                </div>
                <div className="flex items-center gap-2">
                  {result?.model_glb?.url && (
                    <a href={result.model_glb.url} download className="inline-flex items-center gap-2 rounded border border-white/20 px-3 py-2 text-xs font-mono text-white/50 hover:text-white">
                      <Download className="h-3.5 w-3.5" /> GLB
                    </a>
                  )}
                  <a href={previewUrl} target="_blank" rel="noreferrer" className="inline-flex items-center gap-2 rounded border border-white/20 px-3 py-2 text-xs font-mono text-white/50 hover:text-white">
                    <ExternalLink className="h-3.5 w-3.5" /> Open
                  </a>
                </div>
              </div>
              <div className="min-h-[460px]">
                <ModelViewer src={previewUrl} />
              </div>
              <div className="grid gap-px bg-white/10 text-xs font-mono text-white/45 sm:grid-cols-3">
                <div className="bg-black px-4 py-3">Model: {result?.model || 'trellis-2-retexture'}</div>
                <div className="bg-black px-4 py-3">Resolution: {resolution}</div>
                <div className="bg-black px-4 py-3">Seed: {result?.seed ?? '—'}</div>
              </div>
            </section>
          </div>

          <div className="mt-6 min-w-0 overflow-hidden rounded-lg border border-white/20 bg-white/[0.05] p-5">
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
            <CodeBlock
              language={activeSnippet === 'javascript' ? 'javascript' : activeSnippet === 'curl' ? 'bash' : 'python'}
              label={activeSnippet === 'javascript' ? 'JavaScript' : activeSnippet === 'curl' ? 'cURL' : 'Python'}
              code={snippets[activeSnippet]}
              containerClassName="border border-white/20 rounded-lg overflow-hidden bg-black/40"
              headerClassName="bg-white/[0.06]"
              preClassName="text-xs"
            />
          </div>
        </div>
      </div>
    </>
  );
}
