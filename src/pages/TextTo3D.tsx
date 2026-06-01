import React, { useEffect, useMemo, useState } from 'react';
import { Boxes, Copy, Download, ExternalLink, Loader2, Wand2 } from 'lucide-react';
import { Seo } from '../components/Seo';
import { CodeBlock } from '../components/CodeBlock';
import { ModelViewer } from '../components/ModelViewer';

const DEFAULT_PROMPT = 'A single ornate medieval longsword, centered full object, straight blade, polished steel, simple leather grip and gold pommel, isolated on a pure white background, 3D game asset render, orthographic product view, no text, no hands, no sheath, no extra objects';
const DEFAULT_MODEL_URL = 'https://openpathsstatic.openpaths.io/static/uploads/image-to-3d/sword-pixal3d.glb';

const IMAGE_MODELS = [
  { id: 'openpaths/auto-image', label: 'Auto Image (routed)', price: 0.211 },
  { id: 'gpt-image-2', label: 'GPT Image 2', price: 0.04 },
  { id: 'ra1', label: 'RA1 Art Generator', price: 0.04 },
];

const SIZES = ['1024x1024', '768x1024', '1024x768', '1280x720', '720x1280'];

type TextTo3DBilling = {
  texture_size?: number;
  image_model?: string;
  image_cost_usd?: number;
  model_3d_cost_usd?: number;
  total_cost_usd?: number;
};

type TextTo3DResult = {
  model_glb?: { url?: string; file_name?: string; file_size?: number };
  image?: { url?: string; model?: string };
  prompt?: string;
  seed?: number;
  model?: string;
  billing?: TextTo3DBilling;
  credits_charged?: number;
};

type TextTo3DJob = {
  id?: string;
  status?: string;
  result?: TextTo3DResult;
  error?: { message?: string } | string;
};

type APIErrorBody = { error?: { message?: string } | string };

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

export function TextTo3D() {
  const [apiKey, setApiKey] = useState(getStoredAPIKey);
  const [prompt, setPrompt] = useState(DEFAULT_PROMPT);
  const [imageModel, setImageModel] = useState('openpaths/auto-image');
  const [size, setSize] = useState('1024x1024');
  const [resolution, setResolution] = useState(1024);
  const [textureSize, setTextureSize] = useState(1024);
  const [remesh, setRemesh] = useState(true);
  const [activeSnippet, setActiveSnippet] = useState<'python' | 'javascript' | 'curl'>('python');
  const [result, setResult] = useState<TextTo3DResult | null>(null);
  const [modelUrl, setModelUrl] = useState(DEFAULT_MODEL_URL);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (apiKey) localStorage.setItem('op_api_key', apiKey);
  }, [apiKey]);

  const payload = useMemo(() => ({
    model: 'pixal3d-image-to-3d',
    image_model: imageModel,
    prompt,
    size,
    resolution,
    texture_size: textureSize,
    remesh,
  }), [imageModel, prompt, remesh, resolution, size, textureSize]);

  const imagePrice = IMAGE_MODELS.find(m => m.id === imageModel)?.price ?? 0.04;
  const model3dPrice = textureSize === 1024 ? 0.30 : 0.42;
  const estTotal = imagePrice + model3dPrice;
  const billed = result?.billing?.total_cost_usd ?? estTotal;

  const snippets = useMemo(() => {
    const bearer = apiKey || 'op-...';
    const body = JSON.stringify(payload, null, 2);
    return {
      python: `import json
import requests

payload = json.loads(r'''${body}''')

resp = requests.post(
    "https://openpaths.io/v1/3d/text-generations",
    headers={
        "Authorization": "Bearer ${bearer}",
        "Content-Type": "application/json",
    },
    json=payload,
)
resp.raise_for_status()

data = resp.json()
print(data["model_glb"]["url"])`,
      javascript: `const resp = await fetch("https://openpaths.io/v1/3d/text-generations", {
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
      curl: `curl https://openpaths.io/v1/3d/text-generations \\
  -H "Authorization: Bearer ${bearer}" \\
  -H "Content-Type: application/json" \\
  -d '${body}'`,
    };
  }, [apiKey, payload]);

  async function pollJob(jobId: string): Promise<TextTo3DResult> {
    for (let attempt = 0; attempt < 200; attempt += 1) {
      await new Promise(resolve => window.setTimeout(resolve, attempt < 5 ? 1500 : 2500));
      const resp = await fetch(`/v1/3d/text-generations/${encodeURIComponent(jobId)}`, {
        headers: { Authorization: `Bearer ${apiKey}` },
      });
      const job = await parseAPIResponse(resp) as TextTo3DJob;
      if (!resp.ok) throw new Error((job as any)?.error?.message || (job as any)?.error || 'Text-to-3D status check failed');
      if (job.status === 'completed' && job.result) return job.result;
      if (job.status === 'failed') {
        const message = typeof job.error === 'string' ? job.error : job.error?.message;
        throw new Error(message || 'Text-to-3D generation failed');
      }
    }
    throw new Error('Still running. Retry the same request or poll the job URL again.');
  }

  async function generate() {
    if (!apiKey) {
      setError('Set an OpenPaths API key before generating.');
      return;
    }
    if (!prompt.trim()) {
      setError('Enter a prompt to generate from.');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const resp = await fetch('/v1/3d/text-generations', {
        method: 'POST',
        headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const data = await parseAPIResponse(resp) as TextTo3DResult | TextTo3DJob;
      if (!resp.ok) throw new Error(getAPIErrorMessage(data, 'Text-to-3D generation failed'));
      const finalResult = resp.status === 202 && 'id' in data && data.id ? await pollJob(data.id) : data as TextTo3DResult;
      setResult(finalResult);
      if (finalResult?.model_glb?.url) setModelUrl(finalResult.model_glb.url);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Text-to-3D generation failed');
    } finally {
      setLoading(false);
    }
  }

  async function copySnippet() {
    await navigator.clipboard.writeText(snippets[activeSnippet]);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  }

  return (
    <>
      <Seo
        title="Text to 3D API | OpenPaths"
        description="Generate textured GLB models straight from a text prompt. OpenPaths auto-generates an image then converts it to 3D with Pixal3D."
        path="/text-to-3d"
      />

      <div className="px-6 py-10">
        <div className="mx-auto max-w-7xl">
          <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <div className="mb-3 inline-flex items-center gap-2 rounded border border-white/10 bg-white/[0.03] px-3 py-1 text-xs font-mono text-white/45">
                <Boxes className="h-3.5 w-3.5" /> Auto Image + Fal Pixal3D
              </div>
              <h1 className="max-w-3xl text-4xl font-bold tracking-tight md:text-6xl">Text to 3D</h1>
              <p className="mt-4 max-w-2xl text-base leading-relaxed text-white/55">
                Type a prompt. OpenPaths generates an image with the auto image route, then turns it into a textured GLB. You pay the image price plus the Pixal3D price.
              </p>
            </div>
            <div className="grid grid-cols-2 gap-2 text-xs font-mono text-white/45 sm:grid-cols-3">
              <span className="rounded border border-white/10 px-3 py-2">Image: ${imagePrice.toFixed(3)}</span>
              <span className="rounded border border-white/10 px-3 py-2">3D: ${model3dPrice.toFixed(2)}</span>
              <span className="rounded border border-white/10 px-3 py-2">Total: ~${estTotal.toFixed(2)}</span>
            </div>
          </div>

          <div className="grid gap-6 lg:grid-cols-[420px_minmax(0,1fr)]">
            <section className="rounded-lg border border-white/10 bg-white/[0.02] p-5">
              <div className="mb-4 flex items-center justify-between">
                <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">Generate</h2>
                <button type="button" onClick={() => { setPrompt(DEFAULT_PROMPT); setResult(null); setModelUrl(DEFAULT_MODEL_URL); }} className="text-xs font-mono text-white/40 hover:text-white">
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
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Prompt</span>
                <textarea
                  value={prompt}
                  onChange={e => setPrompt(e.target.value)}
                  rows={5}
                  placeholder="Describe a single object on a clean background..."
                  className="w-full resize-y rounded border border-white/10 bg-black px-3 py-2 text-sm text-white placeholder:text-white/25 focus:border-white/30 focus:outline-none"
                />
              </label>

              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Image model</span>
                <select value={imageModel} onChange={e => setImageModel(e.target.value)} className="w-full rounded border border-white/10 bg-black px-3 py-2 text-sm font-mono text-white focus:border-white/30 focus:outline-none">
                  {IMAGE_MODELS.map(m => (
                    <option key={m.id} value={m.id}>{m.label} — ${m.price.toFixed(3)}</option>
                  ))}
                </select>
              </label>

              <div className="grid grid-cols-3 gap-3">
                <label>
                  <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Size</span>
                  <select value={size} onChange={e => setSize(e.target.value)} className="w-full rounded border border-white/10 bg-black px-2 py-2 text-xs font-mono text-white focus:border-white/30 focus:outline-none">
                    {SIZES.map(s => <option key={s} value={s}>{s}</option>)}
                  </select>
                </label>
                <label>
                  <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Structure</span>
                  <select value={resolution} onChange={e => setResolution(Number(e.target.value))} className="w-full rounded border border-white/10 bg-black px-2 py-2 text-xs font-mono text-white focus:border-white/30 focus:outline-none">
                    <option value={1024}>1024</option>
                    <option value={1536}>1536</option>
                  </select>
                </label>
                <label>
                  <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Texture</span>
                  <select value={textureSize} onChange={e => setTextureSize(Number(e.target.value))} className="w-full rounded border border-white/10 bg-black px-2 py-2 text-xs font-mono text-white focus:border-white/30 focus:outline-none">
                    <option value={1024}>1024</option>
                    <option value={2048}>2048</option>
                    <option value={4096}>4096</option>
                  </select>
                </label>
              </div>

              <button type="button" onClick={() => setRemesh(v => !v)} className={`mt-3 w-full rounded border px-3 py-2 text-xs font-mono transition-colors ${remesh ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-200' : 'border-white/10 bg-black text-white/45'}`}>
                Remesh {remesh ? 'on' : 'off'}
              </button>

              <button
                type="button"
                onClick={generate}
                disabled={loading}
                className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded bg-white px-4 py-3 text-sm font-mono font-bold text-black transition-colors hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Wand2 className="h-4 w-4" />}
                {loading ? 'Generating 3D...' : `Generate 3D (~$${billed.toFixed(2)})`}
              </button>

              <p className="mt-2 text-center text-[10px] font-mono text-white/30">Image + 3D can take 40–90s.</p>

              {error && <p className="mt-3 rounded border border-red-400/20 bg-red-400/10 px-3 py-2 text-xs font-mono text-red-200">{error}</p>}
            </section>

            <section className="overflow-hidden rounded-lg border border-white/10 bg-white/[0.02]">
              <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
                <div>
                  <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">3D Viewer</h2>
                  <p className="mt-1 text-xs font-mono text-white/35">{result ? 'Generated result' : 'Default Pixal3D sword result'}</p>
                </div>
                <div className="flex items-center gap-2">
                  <a href={modelUrl} target="_blank" rel="noreferrer" className="inline-flex items-center gap-2 rounded border border-white/10 px-3 py-2 text-xs font-mono text-white/50 hover:text-white">
                    <ExternalLink className="h-3.5 w-3.5" /> Open
                  </a>
                  <a href={modelUrl} download className="inline-flex items-center gap-2 rounded border border-white/10 px-3 py-2 text-xs font-mono text-white/50 hover:text-white">
                    <Download className="h-3.5 w-3.5" /> GLB
                  </a>
                </div>
              </div>
              <div className="min-h-[420px]">
                <ModelViewer src={modelUrl} />
              </div>
              {result?.image?.url && (
                <div className="flex items-center gap-3 border-t border-white/10 px-4 py-3">
                  <img src={result.image.url} alt="Generated image" className="h-16 w-16 flex-none rounded border border-white/10 bg-black object-cover" />
                  <div className="min-w-0 text-xs font-mono text-white/45">
                    <div className="mb-1 text-white/60">Generated image{result.image.model ? ` · ${result.image.model}` : ''}</div>
                    <a href={result.image.url} target="_blank" rel="noreferrer" className="block truncate hover:text-white">{result.image.url}</a>
                  </div>
                </div>
              )}
              <div className="grid gap-px bg-white/10 text-xs font-mono text-white/45 sm:grid-cols-3">
                <div className="bg-black px-4 py-3">Image: ${(result?.billing?.image_cost_usd ?? imagePrice).toFixed(3)}</div>
                <div className="bg-black px-4 py-3">3D: ${(result?.billing?.model_3d_cost_usd ?? model3dPrice).toFixed(2)}</div>
                <div className="bg-black px-4 py-3">Total: ${(result?.billing?.total_cost_usd ?? estTotal).toFixed(2)}</div>
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
              <h2 className="mb-3 font-mono text-sm font-bold uppercase tracking-wider text-white/70">How it works &amp; pricing</h2>
              <ol className="mb-4 list-decimal space-y-2 pl-5 text-sm leading-relaxed text-white/55">
                <li>Your prompt is sent to the auto image route (or the model you pick).</li>
                <li>The generated image is rehosted to a public URL.</li>
                <li>Fal Pixal3D converts that image into a textured GLB.</li>
              </ol>
              <p className="text-sm leading-relaxed text-white/55">
                You are billed for <strong className="text-white/80">both</strong> steps: the image price (≈$0.04 for GPT Image 2 / RA1, $0.211 for auto image) plus the Pixal3D price ($0.30 at 1024 texture, $0.42 at 2048/4096). Pass <code className="text-white/70">image_url</code> instead of a prompt to skip the image step and only pay the 3D price.
              </p>
            </section>
          </div>
        </div>
      </div>
    </>
  );
}
