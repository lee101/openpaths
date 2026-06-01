import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Box, Copy, Download, ExternalLink, Loader2, Upload, Wand2 } from 'lucide-react';
import { Seo } from '../components/Seo';
import { CodeBlock } from '../components/CodeBlock';
import { ModelViewer } from '../components/ModelViewer';
import { Model3DDrop } from '../components/Model3DDrop';
import { prepareUploadFile } from '../lib/imageUpload';
import { normalizeUploadedAssetUrl } from '../lib/uploadUrls';

const DEFAULT_IMAGE_URL = 'https://openpathsstatic.openpaths.io/static/uploads/image-to-3d/sword-reference.jpg';
const DEFAULT_MODEL_URL = 'https://openpathsstatic.openpaths.io/static/uploads/image-to-3d/sword-pixal3d.glb';
const DEFAULT_PROMPT = 'A single ornate medieval longsword, centered full object, straight blade, polished steel, simple leather grip and gold pommel, isolated on a pure white background, 3D game asset render, orthographic product view, no text, no hands, no sheath, no extra objects';

type ImageTo3DResult = {
  model_glb?: {
    url?: string;
    content_type?: string;
    file_name?: string;
    file_size?: number;
  };
  seed?: number;
  billing?: {
    texture_size?: number;
    external_cost_usd?: number;
    external_cost_cents?: number;
  };
};

type ImageTo3DJob = {
  id?: string;
  status?: string;
  result?: ImageTo3DResult;
  error?: { message?: string } | string;
};

type APIErrorBody = {
  error?: { message?: string } | string;
};

function getStoredAPIKey() {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem('op_api_key') || '';
}

function formatUSD(value?: number) {
  if (!value) return '$0.30';
  return `$${value.toFixed(2)}`;
}

async function parseAPIResponse(resp: Response) {
  const text = await resp.text();
  if (!text) return null;
  try {
    return JSON.parse(text);
  } catch {
    if (!resp.ok) {
      throw new Error(text);
    }
    throw new Error('Invalid JSON response: ' + text.slice(0, 160));
  }
}

function getAPIErrorMessage(data: unknown, fallback: string) {
  const error = (data as APIErrorBody | null)?.error;
  if (typeof error === 'string') return error;
  return error?.message || fallback;
}

function getClipboardImageFiles(data: DataTransfer | null): File[] {
  if (!data) return [];

  const files = Array.from(data.files || []).filter(file => file.type.startsWith('image/'));
  if (files.length > 0) return files;

  return Array.from(data.items || [])
    .filter(item => item.kind === 'file' && item.type.startsWith('image/'))
    .map(item => item.getAsFile())
    .filter((file): file is File => Boolean(file));
}

export function ImageTo3D() {
  const [apiKey, setApiKey] = useState(getStoredAPIKey);
  const [imageUrl, setImageUrl] = useState(DEFAULT_IMAGE_URL);
  const [activeSnippet, setActiveSnippet] = useState<'python' | 'javascript' | 'curl'>('python');
  const [uploadedImageUrl, setUploadedImageUrl] = useState('');
  const [engine, setEngine] = useState<'pixal3d' | 'meshy-v6' | 'tripo'>('pixal3d');
  const [textureSize, setTextureSize] = useState(1024);
  const [resolution, setResolution] = useState(1024);
  const [remesh, setRemesh] = useState(true);
  const [topology, setTopology] = useState<'triangle' | 'quad'>('triangle');
  const [targetPolycount, setTargetPolycount] = useState(30000);
  const [shouldTexture, setShouldTexture] = useState(true);
  const [result, setResult] = useState<ImageTo3DResult | null>(null);
  const [modelUrl, setModelUrl] = useState(DEFAULT_MODEL_URL);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [draggingReference, setDraggingReference] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (apiKey) localStorage.setItem('op_api_key', apiKey);
  }, [apiKey]);

  const payload = useMemo(() => {
    if (engine === 'tripo') {
      return {
        model: 'tripo-p1-image-to-3d',
        image_url: imageUrl,
        target_polycount: targetPolycount,
        should_texture: shouldTexture,
      };
    }
    if (engine === 'meshy-v6') {
      return {
        model: 'meshy-v6-image-to-3d',
        image_url: imageUrl,
        topology,
        target_polycount: targetPolycount,
        should_texture: shouldTexture,
        remesh,
      };
    }
    return {
      model: 'pixal3d-image-to-3d',
      image_url: imageUrl,
      resolution,
      texture_size: textureSize,
      remesh,
      ss_guidance_strength: 7.5,
      ss_guidance_rescale: 0.7,
      ss_sampling_steps: 12,
      ss_rescale_t: 5,
      shape_slat_guidance_strength: 7.5,
      shape_slat_guidance_rescale: 0.5,
      shape_slat_sampling_steps: 12,
      shape_slat_rescale_t: 3,
      tex_slat_guidance_strength: 1,
      tex_slat_sampling_steps: 12,
      tex_slat_rescale_t: 3,
      mesh_scale: 1,
      max_num_tokens: 49152,
      decimation_target: 200000,
    };
  }, [engine, imageUrl, remesh, resolution, textureSize, topology, targetPolycount, shouldTexture]);

  const snippets = useMemo(() => {
    const bearer = apiKey || 'op-...';
    const body = JSON.stringify(payload, null, 2);
    return {
      python: `import json
import requests

payload = json.loads(r'''${body}''')

resp = requests.post(
    "https://openpaths.io/v1/3d/generations",
    headers={
        "Authorization": "Bearer ${bearer}",
        "Content-Type": "application/json",
    },
    json=payload,
)
resp.raise_for_status()

model_url = resp.json()["model_glb"]["url"]
print(model_url)`,
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
    if (!apiKey) {
      setError('Set an OpenPaths API key before uploading.');
      return;
    }
    setUploading(true);
    setError('');
    try {
      const uploadFile = await prepareUploadFile(file);
      const form = new FormData();
      form.append('file', uploadFile);
      const resp = await fetch('/v1/files/upload', {
        method: 'POST',
        headers: { Authorization: `Bearer ${apiKey}` },
        body: form,
      });
      const data = await parseAPIResponse(resp);
      if (!resp.ok) throw new Error(data?.error?.message || data?.error || 'Upload failed');
      const publicUrl = normalizeUploadedAssetUrl(data.url);
      setImageUrl(publicUrl);
      setUploadedImageUrl(publicUrl);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      setUploading(false);
    }
  }, [apiKey]);

  useEffect(() => {
    const handlePaste = (event: ClipboardEvent) => {
      if (uploading) return;
      const [file] = getClipboardImageFiles(event.clipboardData);
      if (!file) return;
      event.preventDefault();
      void uploadReference(file);
    };

    window.addEventListener('paste', handlePaste);
    return () => window.removeEventListener('paste', handlePaste);
  }, [uploadReference, uploading]);

  const referenceDropHandlers = {
    onDragEnter: (event: React.DragEvent<HTMLLabelElement>) => {
      event.preventDefault();
      event.stopPropagation();
      if (!uploading) setDraggingReference(true);
    },
    onDragOver: (event: React.DragEvent<HTMLLabelElement>) => {
      event.preventDefault();
      event.stopPropagation();
      event.dataTransfer.dropEffect = uploading ? 'none' : 'copy';
      if (!uploading) setDraggingReference(true);
    },
    onDragLeave: (event: React.DragEvent<HTMLLabelElement>) => {
      event.preventDefault();
      event.stopPropagation();
      if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
        setDraggingReference(false);
      }
    },
    onDrop: (event: React.DragEvent<HTMLLabelElement>) => {
      event.preventDefault();
      event.stopPropagation();
      setDraggingReference(false);
      if (uploading) return;
      const [file] = Array.from<File>(event.dataTransfer.files).filter(item => item.type.startsWith('image/') || /\.(png|jpe?g|webp|gif|bmp|avif)$/i.test(item.name));
      if (file) void uploadReference(file);
    },
  };

  async function poll3DJob(jobId: string): Promise<ImageTo3DResult> {
    for (let attempt = 0; attempt < 180; attempt += 1) {
      await new Promise(resolve => window.setTimeout(resolve, attempt < 5 ? 1500 : 2500));
      const resp = await fetch(`/v1/3d/generations/${encodeURIComponent(jobId)}`, {
        headers: { Authorization: `Bearer ${apiKey}` },
      });
      const job = await parseAPIResponse(resp) as ImageTo3DJob;
      if (!resp.ok) throw new Error((job as any)?.error?.message || (job as any)?.error || '3D generation status check failed');
      if (job.status === 'completed' && job.result) return job.result;
      if (job.status === 'failed') {
        const message = typeof job.error === 'string' ? job.error : job.error?.message;
        throw new Error(message || '3D generation failed');
      }
    }
    throw new Error('3D generation is still running. Retry the same request or poll the job URL again.');
  }

  async function generate3D() {
    if (!apiKey) {
      setError('Set an OpenPaths API key before generating.');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const resp = await fetch('/v1/3d/generations', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${apiKey}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });
      const data = await parseAPIResponse(resp) as ImageTo3DResult | ImageTo3DJob;
      if (!resp.ok) throw new Error(getAPIErrorMessage(data, '3D generation failed'));
      const finalResult = resp.status === 202 && 'id' in data && data.id ? await poll3DJob(data.id) : data as ImageTo3DResult;
      setResult(finalResult);
      if (finalResult?.model_glb?.url) setModelUrl(finalResult.model_glb.url);
    } catch (err) {
      setError(err instanceof Error ? err.message : '3D generation failed');
    } finally {
      setLoading(false);
    }
  }

  async function copyCurl() {
    await navigator.clipboard.writeText(snippets.curl);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  }

  async function copySnippet() {
    await navigator.clipboard.writeText(snippets[activeSnippet]);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  }

  const billed = result?.billing?.external_cost_usd || (engine === 'meshy-v6' ? 0.8 : engine === 'tripo' ? (shouldTexture ? 0.5 : 0.4) : (textureSize === 1024 ? 0.3 : 0.42));

  return (
    <>
      <Seo
        title="Image to 3D API | OpenPaths"
        description="Generate textured GLB models from a single image through OpenPaths and Fal Pixal3D."
        path="/image-to-3d"
      />

      <div className="px-6 py-10">
        <div className="mx-auto max-w-7xl">
          <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <div className="mb-3 inline-flex items-center gap-2 rounded border border-white/10 bg-white/[0.03] px-3 py-1 text-xs font-mono text-white/45">
                <Box className="h-3.5 w-3.5" /> {engine === 'meshy-v6' ? 'Fal Meshy v6' : engine === 'tripo' ? 'Fal Tripo p1' : 'Fal Pixal3D'}
              </div>
              <h1 className="max-w-3xl text-4xl font-bold tracking-tight md:text-6xl">Image to 3D</h1>
              <p className="mt-4 max-w-2xl text-base leading-relaxed text-white/55">
                Convert an image into a textured GLB.
              </p>
            </div>
            <div className="grid grid-cols-2 gap-2 text-xs font-mono text-white/45 sm:grid-cols-4">
              <span className="rounded border border-white/10 px-3 py-2">1024: $0.30</span>
              <span className="rounded border border-white/10 px-3 py-2">2048: $0.42</span>
              <span className="rounded border border-white/10 px-3 py-2">4096: $0.42</span>
              <span className="rounded border border-white/10 px-3 py-2">GLB output</span>
            </div>
          </div>

          <div className="grid gap-6 lg:grid-cols-[420px_minmax(0,1fr)]">
            <section className="rounded-lg border border-white/10 bg-white/[0.02] p-5">
              <div className="mb-4 flex items-center justify-between">
                <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">Generate</h2>
                <button type="button" onClick={() => { setImageUrl(DEFAULT_IMAGE_URL); setUploadedImageUrl(''); setModelUrl(DEFAULT_MODEL_URL); setResult(null); }} className="text-xs font-mono text-white/40 hover:text-white">
                  sword default
                </button>
              </div>

              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">OpenPaths API key</span>
                <input
                  value={apiKey}
                  onChange={e => setApiKey(e.target.value)}
                  placeholder="op-..."
                  className="w-full rounded border border-white/10 bg-black px-3 py-2 text-sm font-mono text-white placeholder:text-white/25 focus:border-white/30 focus:outline-none"
                />
              </label>

              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Image URL</span>
                <input
                  value={imageUrl}
                  onChange={e => setImageUrl(normalizeUploadedAssetUrl(e.target.value))}
                  className="w-full rounded border border-white/10 bg-black px-3 py-2 text-sm font-mono text-white placeholder:text-white/25 focus:border-white/30 focus:outline-none"
                />
                {imageUrl && (
                  <div className="mt-2 flex items-center gap-3 rounded border border-white/10 bg-white/[0.02] p-2">
                    <img src={imageUrl} alt="Selected reference" className="h-14 w-14 flex-none rounded border border-white/10 bg-black object-cover" />
                    <a href={imageUrl} target="_blank" rel="noreferrer" className="min-w-0 truncate text-xs font-mono text-white/45 hover:text-white">
                      {imageUrl}
                    </a>
                  </div>
                )}
              </label>

              <label
                className={`mb-3 flex cursor-pointer items-center gap-3 rounded border border-dashed px-3 py-3 text-xs font-mono transition-colors ${draggingReference ? 'border-emerald-300/50 bg-emerald-400/10 text-emerald-100' : 'border-white/10 bg-black/50 text-white/45 hover:border-white/20'}`}
                {...referenceDropHandlers}
              >
                <Upload className="h-4 w-4" />
                <span>{uploading ? 'Uploading reference...' : draggingReference ? 'Drop image to upload' : 'Upload, paste, or drop reference image'}</span>
                <input
                  type="file"
                  accept="image/*"
                  disabled={uploading}
                  onChange={e => e.target.files?.[0] && uploadReference(e.target.files[0])}
                  className="hidden"
                />
              </label>

              {uploadedImageUrl && (
                <div className="mb-4 overflow-hidden rounded border border-emerald-400/25 bg-emerald-400/10">
                  <div className="flex gap-3 p-3">
                    <img src={uploadedImageUrl} alt="Uploaded reference" className="h-20 w-20 flex-none rounded border border-white/10 bg-black object-cover" />
                    <div className="min-w-0">
                      <div className="mb-1 text-[10px] font-mono uppercase tracking-[0.16em] text-emerald-200">Upload complete</div>
                      <p className="text-xs leading-relaxed text-white/55">This image is now selected for 3D generation.</p>
                      <a href={uploadedImageUrl} target="_blank" rel="noreferrer" className="mt-2 block truncate text-xs font-mono text-emerald-200/80 hover:text-emerald-100">
                        {uploadedImageUrl}
                      </a>
                    </div>
                  </div>
                </div>
              )}

              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Engine</span>
                <select value={engine} onChange={e => { setEngine(e.target.value as 'pixal3d' | 'meshy-v6' | 'tripo'); setResult(null); }} className="w-full rounded border border-white/10 bg-black px-3 py-2 text-sm font-mono text-white focus:border-white/30 focus:outline-none">
                  <option value="pixal3d">Pixal3D — fast, textured ($0.30–$0.42)</option>
                  <option value="meshy-v6">Meshy v6 — higher quality ($0.80)</option>
                  <option value="tripo">Tripo p1 — sharp geometry ($0.40–$0.50)</option>
                </select>
              </label>

              {engine === 'pixal3d' ? (
                <>
                  <div className="grid grid-cols-2 gap-3">
                    <label>
                      <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Structure</span>
                      <select value={resolution} onChange={e => setResolution(Number(e.target.value))} className="w-full rounded border border-white/10 bg-black px-3 py-2 text-sm font-mono text-white focus:border-white/30 focus:outline-none">
                        <option value={1024}>1024</option>
                        <option value={1536}>1536</option>
                      </select>
                    </label>
                    <label>
                      <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Texture</span>
                      <select value={textureSize} onChange={e => setTextureSize(Number(e.target.value))} className="w-full rounded border border-white/10 bg-black px-3 py-2 text-sm font-mono text-white focus:border-white/30 focus:outline-none">
                        <option value={1024}>1024</option>
                        <option value={2048}>2048</option>
                        <option value={4096}>4096</option>
                      </select>
                    </label>
                  </div>

                  <button type="button" onClick={() => setRemesh(v => !v)} className={`mt-3 w-full rounded border px-3 py-2 text-xs font-mono transition-colors ${remesh ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-200' : 'border-white/10 bg-black text-white/45'}`}>
                    Remesh {remesh ? 'on' : 'off'}
                  </button>
                </>
              ) : engine === 'meshy-v6' ? (
                <>
                  <div className="grid grid-cols-2 gap-3">
                    <label>
                      <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Topology</span>
                      <select value={topology} onChange={e => setTopology(e.target.value as 'triangle' | 'quad')} className="w-full rounded border border-white/10 bg-black px-3 py-2 text-sm font-mono text-white focus:border-white/30 focus:outline-none">
                        <option value="triangle">triangle</option>
                        <option value="quad">quad</option>
                      </select>
                    </label>
                    <label>
                      <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Target polycount</span>
                      <input type="number" min={1000} max={300000} step={1000} value={targetPolycount} onChange={e => setTargetPolycount(Number(e.target.value) || 30000)} className="w-full rounded border border-white/10 bg-black px-3 py-2 text-sm font-mono text-white focus:border-white/30 focus:outline-none" />
                    </label>
                  </div>

                  <div className="mt-3 grid grid-cols-2 gap-3">
                    <button type="button" onClick={() => setShouldTexture(v => !v)} className={`rounded border px-3 py-2 text-xs font-mono transition-colors ${shouldTexture ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-200' : 'border-white/10 bg-black text-white/45'}`}>
                      Texture {shouldTexture ? 'on' : 'off'}
                    </button>
                    <button type="button" onClick={() => setRemesh(v => !v)} className={`rounded border px-3 py-2 text-xs font-mono transition-colors ${remesh ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-200' : 'border-white/10 bg-black text-white/45'}`}>
                      Remesh {remesh ? 'on' : 'off'}
                    </button>
                  </div>
                </>
              ) : (
                <>
                  <label className="block">
                    <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Face limit (0 = adaptive)</span>
                    <input type="number" min={0} max={500000} step={1000} value={targetPolycount} onChange={e => setTargetPolycount(Number(e.target.value) || 0)} className="w-full rounded border border-white/10 bg-black px-3 py-2 text-sm font-mono text-white focus:border-white/30 focus:outline-none" />
                  </label>

                  <button type="button" onClick={() => setShouldTexture(v => !v)} className={`mt-3 w-full rounded border px-3 py-2 text-xs font-mono transition-colors ${shouldTexture ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-200' : 'border-white/10 bg-black text-white/45'}`}>
                    Texture {shouldTexture ? 'on ($0.50)' : 'off ($0.40)'}
                  </button>
                </>
              )}

              <button
                type="button"
                onClick={generate3D}
                disabled={loading}
                className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded bg-white px-4 py-3 text-sm font-mono font-bold text-black transition-colors hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Wand2 className="h-4 w-4" />}
                {loading ? 'Generating GLB...' : `Generate GLB (${formatUSD(billed)})`}
              </button>

              {error && <p className="mt-3 rounded border border-red-400/20 bg-red-400/10 px-3 py-2 text-xs font-mono text-red-200">{error}</p>}

              <div className="mt-5 overflow-hidden rounded border border-white/10 bg-black">
                <img src={imageUrl} alt="Input reference" className="h-auto w-full" />
              </div>
            </section>

            <section className="overflow-hidden rounded-lg border border-white/10 bg-white/[0.02]">
              <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
                <div>
                  <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">3D Viewer</h2>
                  <p className="mt-1 text-xs font-mono text-white/35">{result ? 'Generated result' : 'Default Pixal3D sword result'}</p>
                </div>
                <div className="flex items-center gap-2">
                  <button type="button" onClick={copyCurl} className="inline-flex items-center gap-2 rounded border border-white/10 px-3 py-2 text-xs font-mono text-white/50 hover:text-white">
                    <Copy className="h-3.5 w-3.5" /> {copied ? 'Copied' : 'Curl'}
                  </button>
                  <a href={modelUrl} target="_blank" rel="noreferrer" className="inline-flex items-center gap-2 rounded border border-white/10 px-3 py-2 text-xs font-mono text-white/50 hover:text-white">
                    <ExternalLink className="h-3.5 w-3.5" /> Open
                  </a>
                  <a href={modelUrl} download className="inline-flex items-center gap-2 rounded border border-white/10 px-3 py-2 text-xs font-mono text-white/50 hover:text-white">
                    <Download className="h-3.5 w-3.5" /> GLB
                  </a>
                </div>
              </div>
              <div className="min-h-[460px]">
                <ModelViewer src={modelUrl} />
              </div>
              <div className="grid gap-px bg-white/10 text-xs font-mono text-white/45 sm:grid-cols-3">
                <div className="bg-black px-4 py-3">Model: {result?.model || 'pixal3d-image-to-3d'}</div>
                <div className="bg-black px-4 py-3">Texture: {result?.billing?.texture_size || textureSize}</div>
                <div className="bg-black px-4 py-3">Seed: {result?.seed || '1761685686'}</div>
              </div>
            </section>
          </div>

          <div className="mt-6 grid min-w-0 gap-6 lg:grid-cols-2">
            <section className="min-w-0 overflow-hidden rounded-lg border border-white/10 bg-white/[0.02] p-5">
              <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
                <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">API code</h2>
                <div className="flex items-center gap-2">
                  {(['python', 'javascript', 'curl'] as const).map(snippet => (
                    <button
                      key={snippet}
                      type="button"
                      onClick={() => setActiveSnippet(snippet)}
                      className={`rounded border px-2.5 py-1.5 text-[10px] font-mono uppercase tracking-[0.14em] transition-colors ${activeSnippet === snippet ? 'border-white/30 bg-white/10 text-white' : 'border-white/10 text-white/35 hover:text-white'}`}
                    >
                      {snippet === 'javascript' ? 'JS' : snippet}
                    </button>
                  ))}
                  <button type="button" onClick={copySnippet} className="inline-flex items-center gap-1.5 rounded border border-white/10 px-2.5 py-1.5 text-[10px] font-mono uppercase tracking-[0.14em] text-white/45 hover:text-white">
                    <Copy className="h-3 w-3" /> {copied ? 'Copied' : 'Copy'}
                  </button>
                </div>
              </div>
              <CodeBlock
                language={activeSnippet === 'javascript' ? 'javascript' : activeSnippet === 'curl' ? 'bash' : 'python'}
                label={activeSnippet === 'javascript' ? 'JavaScript' : activeSnippet === 'curl' ? 'cURL' : 'Python'}
                code={snippets[activeSnippet]}
                containerClassName="border border-white/10 rounded-lg overflow-hidden bg-black/40"
                headerClassName="bg-white/[0.03]"
                preClassName="text-xs"
              />
            </section>
            <section className="min-w-0 overflow-hidden rounded-lg border border-white/10 bg-white/[0.02] p-5">
              <h2 className="mb-3 font-mono text-sm font-bold uppercase tracking-wider text-white/70">Preview your own 3D file</h2>
              <p className="mb-4 text-sm leading-relaxed text-white/55">
                Already have a model? Drop a <code className="text-white/70">.glb</code> or <code className="text-white/70">.gltf</code> to preview it instantly — nothing is uploaded.
              </p>
              <Model3DDrop minHeight={260} />
              <p className="mt-4 break-words text-xs font-mono text-white/35">Sword default prompt: {DEFAULT_PROMPT}</p>
            </section>
          </div>
        </div>
      </div>
    </>
  );
}
