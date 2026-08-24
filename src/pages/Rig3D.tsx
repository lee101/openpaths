import React, { useEffect, useMemo, useState } from 'react';
import { Copy, Download, ExternalLink, Loader2, PersonStanding, Wand2 } from 'lucide-react';
import { Seo } from '../components/Seo';
import { CodeBlock } from '../components/CodeBlock';
import { ModelViewer } from '../components/ModelViewer';
import { Model3DUpload } from '../components/Model3DUpload';
import { normalizeUploadedAssetUrl } from '../lib/uploadUrls';

const DEFAULT_MODEL_URL = 'https://threejs.org/examples/models/gltf/Soldier.glb';

type RiggingResult = {
  rigged_character_glb?: { url?: string };
  rigged_character_fbx?: { url?: string };
  animation_glb?: { url?: string };
  animation_fbx?: { url?: string };
  basic_animations?: {
    walking_glb?: { url?: string };
    running_glb?: { url?: string };
  };
  rig_task_id?: string;
  billing?: { animation?: boolean; external_cost_usd?: number };
  model?: string;
};

type RiggingJob = {
  id?: string;
  status?: string;
  result?: RiggingResult;
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

export function Rig3D() {
  const [apiKey, setApiKey] = useState(getStoredAPIKey);
  const [modelUrl, setModelUrl] = useState(DEFAULT_MODEL_URL);
  const [heightMeters, setHeightMeters] = useState(1.7);
  const [enableAnimation, setEnableAnimation] = useState(false);
  const [animationActionId, setAnimationActionId] = useState(92);
  const [activeSnippet, setActiveSnippet] = useState<'python' | 'javascript' | 'curl'>('python');
  const [result, setResult] = useState<RiggingResult | null>(null);
  const [riggedUrl, setRiggedUrl] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (apiKey) localStorage.setItem('op_api_key', apiKey);
  }, [apiKey]);

  const price = enableAnimation ? 0.32 : 0.20;

  const payload = useMemo(() => {
    const body: Record<string, unknown> = {
      model: 'meshy-rigging',
      model_url: modelUrl,
      height_meters: heightMeters,
    };
    if (enableAnimation) {
      body.enable_animation = true;
      body.animation_action_id = animationActionId;
    }
    return body;
  }, [modelUrl, heightMeters, enableAnimation, animationActionId]);

  const snippets = useMemo(() => {
    const bearer = apiKey || 'op-...';
    const body = JSON.stringify(payload, null, 2);
    return {
      python: `import requests

resp = requests.post(
    "https://openpaths.io/v1/3d/rigging",
    headers={"Authorization": "Bearer ${bearer}", "Content-Type": "application/json"},
    json=${body.replace(/\n/g, '\n    ')},
)
resp.raise_for_status()
print(resp.json()["rigged_character_glb"]["url"])`,
      javascript: `const resp = await fetch("https://openpaths.io/v1/3d/rigging", {
  method: "POST",
  headers: {
    "Authorization": "Bearer ${bearer}",
    "Content-Type": "application/json",
  },
  body: JSON.stringify(${body.replace(/^/gm, '  ').trim()}),
});

if (!resp.ok) throw new Error(await resp.text());
const data = await resp.json();
console.log(data.rigged_character_glb.url);`,
      curl: `curl https://openpaths.io/v1/3d/rigging \\
  -H "Authorization: Bearer ${bearer}" \\
  -H "Content-Type: application/json" \\
  -d '${body}'`,
    };
  }, [apiKey, payload]);

  async function pollJob(jobId: string): Promise<RiggingResult> {
    for (let attempt = 0; attempt < 120; attempt += 1) {
      await new Promise(resolve => window.setTimeout(resolve, attempt < 5 ? 1500 : 3000));
      const resp = await fetch(`/v1/3d/rigging/${encodeURIComponent(jobId)}`, {
        headers: { Authorization: `Bearer ${apiKey}` },
      });
      const job = await parseAPIResponse(resp) as RiggingJob;
      if (!resp.ok) throw new Error(getAPIErrorMessage(job, 'rigging status check failed'));
      if (job.status === 'completed' && job.result) return job.result;
      if (job.status === 'failed') {
        const message = typeof job.error === 'string' ? job.error : job.error?.message;
        throw new Error(message || 'rigging failed');
      }
    }
    throw new Error('Rigging is still running. Retry the same request or poll the job URL again.');
  }

  async function runRigging() {
    if (!apiKey) { setError('Set an OpenPaths API key before rigging.'); return; }
    if (!modelUrl) { setError('Upload a .glb or paste a public model URL first.'); return; }
    setLoading(true);
    setError('');
    try {
      const resp = await fetch('/v1/3d/rigging', {
        method: 'POST',
        headers: { Authorization: `Bearer ${apiKey}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const data = await parseAPIResponse(resp) as RiggingResult | RiggingJob;
      if (!resp.ok) throw new Error(getAPIErrorMessage(data, 'rigging failed'));
      const finalResult = resp.status === 202 && 'id' in data && data.id ? await pollJob(data.id) : data as RiggingResult;
      setResult(finalResult);
      const url = finalResult?.animation_glb?.url || finalResult?.rigged_character_glb?.url;
      if (url) setRiggedUrl(url);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'rigging failed');
    } finally {
      setLoading(false);
    }
  }

  async function copySnippet() {
    await navigator.clipboard.writeText(snippets[activeSnippet]);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  }

  const previewUrl = riggedUrl || DEFAULT_MODEL_URL;
  const riggedGlb = result?.rigged_character_glb?.url;
  const riggedFbx = result?.rigged_character_fbx?.url;

  return (
    <>
      <Seo
        title="3D Auto-Rigging API | OpenPaths"
        description="Upload a humanoid GLB and get back a rigged character (GLB + FBX) with optional walk/run animation, via OpenPaths and Fal Meshy."
        path="/rig-3d"
      />

      <div className="px-6 py-10">
        <div className="mx-auto max-w-7xl">
          <div className="mb-8 flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <div className="mb-3 inline-flex items-center gap-2 rounded border border-white/20 bg-white/[0.06] px-3 py-1 text-xs font-mono text-white/45">
                <PersonStanding className="h-3.5 w-3.5" /> Fal Meshy Rigging
              </div>
              <h1 className="max-w-3xl text-4xl font-bold tracking-tight md:text-6xl">3D Auto-Rigging</h1>
              <p className="mt-4 max-w-2xl text-base leading-relaxed text-white/55">
                Upload a humanoid mesh — get back a fully rigged character (GLB + FBX), ready to animate. Optionally apply a walk/run animation preset.
              </p>
            </div>
            <div className="grid grid-cols-2 gap-2 text-xs font-mono text-white/45 sm:grid-cols-3">
              <span className="rounded border border-white/20 px-3 py-2">Rig: $0.20</span>
              <span className="rounded border border-white/20 px-3 py-2">+ Animation: $0.32</span>
              <span className="rounded border border-white/20 px-3 py-2">GLB + FBX</span>
            </div>
          </div>

          <div className="grid gap-6 lg:grid-cols-[420px_minmax(0,1fr)]">
            <section className="rounded-lg border border-white/20 bg-white/[0.05] p-5">
              <div className="mb-4 flex items-center justify-between">
                <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">Rig a character</h2>
                <button type="button" onClick={() => { setModelUrl(DEFAULT_MODEL_URL); setResult(null); setRiggedUrl(''); }} className="text-xs font-mono text-white/55 hover:text-white">
                  soldier default
                </button>
              </div>

              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">OpenPaths API key</span>
                <input
                  value={apiKey}
                  onChange={e => setApiKey(e.target.value)}
                  placeholder="op-..."
                  className="w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none"
                />
              </label>

              <div className="mb-3">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Upload mesh (.glb / .gltf)</span>
                <Model3DUpload apiKey={apiKey} onUploaded={setModelUrl} minHeight={240} />
              </div>

              <label className="mb-3 block">
                <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">…or paste a public model URL</span>
                <input
                  value={modelUrl}
                  onChange={e => setModelUrl(normalizeUploadedAssetUrl(e.target.value))}
                  className="w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none"
                />
              </label>

              <div className="grid grid-cols-2 gap-3">
                <label>
                  <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Height (m)</span>
                  <input
                    type="number"
                    step="0.1"
                    min="0.1"
                    value={heightMeters}
                    onChange={e => setHeightMeters(Number(e.target.value) || 1.7)}
                    className="w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white focus:border-white/50 focus:outline-none"
                  />
                </label>
                <label className={enableAnimation ? '' : 'opacity-40'}>
                  <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Animation ID</span>
                  <input
                    type="number"
                    min="0"
                    max="696"
                    disabled={!enableAnimation}
                    value={animationActionId}
                    onChange={e => setAnimationActionId(Number(e.target.value) || 0)}
                    className="w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white focus:border-white/50 focus:outline-none disabled:cursor-not-allowed"
                  />
                </label>
              </div>

              <button type="button" onClick={() => setEnableAnimation(v => !v)} className={`mt-3 w-full rounded border px-3 py-2 text-xs font-mono transition-colors ${enableAnimation ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-200' : 'border-white/20 bg-black text-white/45'}`}>
                Apply animation preset {enableAnimation ? 'on (+$0.12)' : 'off'}
              </button>

              <button
                type="button"
                onClick={runRigging}
                disabled={loading}
                className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded bg-white px-4 py-3 text-sm font-mono font-bold text-black transition-colors hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Wand2 className="h-4 w-4" />}
                {loading ? 'Rigging (1–3 min)…' : `Rig character ($${price.toFixed(2)})`}
              </button>

              {error && <p className="mt-3 rounded border border-red-400/20 bg-red-400/10 px-3 py-2 text-xs font-mono text-red-200">{error}</p>}

              <p className="mt-4 text-xs leading-relaxed text-white/55">
                The mesh must be a humanoid character with clearly defined limbs in GLB format.
              </p>
            </section>

            <section className="overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
              <div className="flex items-center justify-between border-b border-white/20 px-4 py-3">
                <div>
                  <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">Rigged result</h2>
                  <p className="mt-1 text-xs font-mono text-white/50">{riggedUrl ? 'Rigged character' : 'Example — pre-rigged soldier mesh'}</p>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  {riggedGlb && (
                    <a href={riggedGlb} download className="inline-flex items-center gap-2 rounded border border-white/20 px-3 py-2 text-xs font-mono text-white/50 hover:text-white">
                      <Download className="h-3.5 w-3.5" /> GLB
                    </a>
                  )}
                  {riggedFbx && (
                    <a href={riggedFbx} download className="inline-flex items-center gap-2 rounded border border-white/20 px-3 py-2 text-xs font-mono text-white/50 hover:text-white">
                      <Download className="h-3.5 w-3.5" /> FBX
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
                <div className="bg-black px-4 py-3">Model: {result?.model || 'meshy-rigging'}</div>
                <div className="bg-black px-4 py-3">Animation: {result?.billing?.animation ? 'yes' : 'no'}</div>
                <div className="bg-black px-4 py-3 truncate">Task: {result?.rig_task_id || '—'}</div>
              </div>
              {result?.basic_animations && (
                <div className="flex flex-wrap gap-2 border-t border-white/20 px-4 py-3 text-xs font-mono">
                  {result.basic_animations.walking_glb?.url && (
                    <a href={result.basic_animations.walking_glb.url} download className="rounded border border-white/20 px-3 py-1.5 text-white/50 hover:text-white">Walk GLB</a>
                  )}
                  {result.basic_animations.running_glb?.url && (
                    <a href={result.basic_animations.running_glb.url} download className="rounded border border-white/20 px-3 py-1.5 text-white/50 hover:text-white">Run GLB</a>
                  )}
                </div>
              )}
            </section>
          </div>

          <div className="mt-6 grid min-w-0 gap-6 lg:grid-cols-2">
            <section className="min-w-0 overflow-hidden rounded-lg border border-white/20 bg-white/[0.05] p-5">
              <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
                <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">API code</h2>
                <div className="flex items-center gap-2">
                  {(['python', 'javascript', 'curl'] as const).map(snippet => (
                    <button
                      key={snippet}
                      type="button"
                      onClick={() => setActiveSnippet(snippet)}
                      className={`rounded border px-2.5 py-1.5 text-[10px] font-mono uppercase tracking-[0.14em] transition-colors ${activeSnippet === snippet ? 'border-white/30 bg-white/10 text-white' : 'border-white/20 text-white/50 hover:text-white'}`}
                    >
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
            </section>
            <section className="min-w-0 overflow-hidden rounded-lg border border-white/20 bg-white/[0.05] p-5">
              <h2 className="mb-3 font-mono text-sm font-bold uppercase tracking-wider text-white/70">How it works</h2>
              <ol className="space-y-3 text-sm leading-relaxed text-white/55">
                <li><span className="font-mono text-white/70">1.</span> Drop a humanoid <code className="text-white/70">.glb</code> — it previews instantly and uploads to a public URL.</li>
                <li><span className="font-mono text-white/70">2.</span> POST to <code className="text-white/70">/v1/3d/rigging</code>. Long jobs return a job id you can poll at <code className="text-white/70">/v1/3d/rigging/&#123;id&#125;</code>.</li>
                <li><span className="font-mono text-white/70">3.</span> Get back <code className="text-white/70">rigged_character_glb</code> + <code className="text-white/70">rigged_character_fbx</code>, plus walk/run animations, and an optional animation preset.</li>
              </ol>
            </section>
          </div>
        </div>
      </div>
    </>
  );
}
