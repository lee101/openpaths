import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Check, Copy, Loader2, Upload, Wand2 } from 'lucide-react';
import { CodeBlock } from './CodeBlock';
import { SubscribePrompt } from './SubscribePrompt';
import { Field, apiError, inputCls, pollVideoJob, responseJSON, selectCls, snippetFor } from './VideoSpacePanel';
import { EMPTY_ADVANCED, getVideoParamSpec } from '../data/videoModelParams';
import { buildVideoPayload } from '../lib/videoPayload';
import { customPreset, renderSphere, SPHERE_PRESETS, sphereBlob, type SpherePreset } from '../lib/lightSphere';
import { ensureSpaceApiKey, paywallFromResponse, type Paywall } from '../lib/paywall';
import { normalizeUploadedAssetUrl } from '../lib/uploadUrls';
import { prepareUploadFile } from '../lib/imageUpload';
import { AUTH_EVENT } from '../lib/api';

type Lang = 'python' | 'javascript' | 'curl';
const RATES: Record<string, number> = { '480P': 0.055, '768P': 0.088, '1080P': 0.176, '2K': 0.352 };
const DEMO_SOURCE = '/static/relight/source.mp4';
const DEMO_OUTPUT = '/static/relight/relit.mp4';

function SphereThumb({ preset, active, onPick }: { preset: SpherePreset; active: boolean; onPick: () => void; key?: React.Key }) {
  const ref = useRef<HTMLCanvasElement>(null);
  useEffect(() => {
    if (ref.current) renderSphere(ref.current, preset, 96);
  }, [preset]);
  return (
    <button type="button" onClick={onPick} aria-pressed={active} data-testid={`relight-preset-${preset.id}`} className={`rounded border p-1.5 text-left transition-colors ${active ? 'border-white bg-white/10' : 'border-white/20 hover:border-white/50'}`}>
      <canvas ref={ref} className="aspect-square w-full rounded bg-black" />
      <span className="mt-1 block truncate font-mono text-[10px] text-white/70">{preset.name}</span>
    </button>
  );
}

export function RelightSpacePanel({ modelId, modelName }: { modelId: string; modelName: string }) {
  const spec = useMemo(() => getVideoParamSpec(modelId), [modelId]);
  const [lang, setLang] = useState<Lang>('python');
  const [copied, setCopied] = useState(false);
  const [apiKey, setApiKey] = useState(() => (typeof window === 'undefined' ? '' : localStorage.getItem('op_api_key') || ''));
  const [videoUrl, setVideoUrl] = useState(DEMO_SOURCE);
  const [presetId, setPresetId] = useState(SPHERE_PRESETS[0].id);
  const [az, setAz] = useState(-50);
  const [el, setEl] = useState(15);
  const [color, setColor] = useState('#ffb36b');
  const [intensity, setIntensity] = useState(1);
  const [ownSphere, setOwnSphere] = useState<{ file: File; preview: string } | null>(null);
  const [uploadedSphere, setUploadedSphere] = useState<{ key: string; url: string } | null>(null);
  const [resolution, setResolution] = useState('480P');
  const [aspect, setAspect] = useState('16:9');
  const [seed, setSeed] = useState('');
  const [outputUrl, setOutputUrl] = useState(DEMO_OUTPUT);
  const [generated, setGenerated] = useState(false);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');
  const [paywall, setPaywall] = useState<Paywall | null>(null);
  const bigRef = useRef<HTMLCanvasElement>(null);

  const preset = useMemo<SpherePreset>(
    () => (presetId === 'custom' ? customPreset(az, el, color, intensity) : SPHERE_PRESETS.find(p => p.id === presetId) || SPHERE_PRESETS[0]),
    [presetId, az, el, color, intensity],
  );
  const sphereKey = ownSphere ? `own:${ownSphere.file.name}:${ownSphere.file.size}:${ownSphere.file.lastModified}` : `${preset.id}:${JSON.stringify(preset.lights)}`;
  const sphereUrl = uploadedSphere?.key === sphereKey ? uploadedSphere.url : '';

  useEffect(() => {
    if (!ownSphere && bigRef.current) renderSphere(bigRef.current, preset, 256);
  }, [preset, ownSphere]);

  useEffect(() => {
    if (apiKey.trim()) localStorage.setItem('op_api_key', apiKey.trim());
  }, [apiKey]);

  useEffect(() => {
    const sync = () => {
      const key = localStorage.getItem('op_api_key') || '';
      if (key) setApiKey(key);
      setPaywall(current => (current?.kind === 'login' ? null : current));
    };
    window.addEventListener('auth-change', sync);
    window.addEventListener(AUTH_EVENT, sync);
    return () => {
      window.removeEventListener('auth-change', sync);
      window.removeEventListener(AUTH_EVENT, sync);
    };
  }, []);

  useEffect(() => () => { if (ownSphere) URL.revokeObjectURL(ownSphere.preview); }, [ownSphere]);

  const payload = useMemo(
    () => buildVideoPayload(modelId, {
      prompt: '', resolution, duration: 'auto', aspectRatio: aspect, generateAudio: false,
      videoUrl, imageUrl: sphereUrl && sphereUrl.length < 200 ? sphereUrl : 'https://example.com/sphere.png',
    }, { ...EMPTY_ADVANCED, seed }, spec),
    [modelId, resolution, aspect, videoUrl, sphereUrl, seed, spec],
  );
  const snippet = useMemo(() => snippetFor(payload, lang, apiKey), [payload, lang, apiKey]);

  const resolveKey = async () => {
    const key = apiKey.trim() || await ensureSpaceApiKey();
    if (key && key !== apiKey) setApiKey(key);
    if (!key) setPaywall({ kind: 'login', message: 'Sign in to run this model.', subscribeUrl: '/pricing' });
    return key;
  };

  const upload = async (key: string, body: Blob | File, name: string) => {
    const form = new FormData();
    form.append('file', body, name);
    const resp = await fetch('/v1/files/upload', { method: 'POST', headers: { Authorization: `Bearer ${key}` }, body: form });
    const data = await responseJSON(resp);
    const gate = paywallFromResponse(resp, data);
    if (gate) { setPaywall(gate); return ''; }
    if (!resp.ok || !data?.url) throw new Error(apiError(data, 'Upload failed'));
    return normalizeUploadedAssetUrl(data.url) as string;
  };

  const toDataUrl = (blob: Blob) =>
    new Promise<string>((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(String(reader.result));
      reader.onerror = () => reject(new Error('Could not read sphere image'));
      reader.readAsDataURL(blob);
    });

  const uploadVideo = async (file?: File) => {
    if (!file) return;
    const key = await resolveKey();
    if (!key) return;
    setUploading(true);
    setError('');
    try {
      const url = await upload(key, file, file.name);
      if (url) setVideoUrl(url);
    } catch (err) {
      setError(`${err instanceof Error ? err.message : 'Upload failed'}. Paste a public video URL instead.`);
    } finally {
      setUploading(false);
    }
  };

  const pickOwnSphere = (file?: File) => {
    if (!file) return;
    setOwnSphere({ file, preview: URL.createObjectURL(file) });
  };

  const copy = async () => {
    await navigator.clipboard.writeText(snippet);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  };

  const generate = async () => {
    if (!videoUrl.trim()) {
      setError('Add a source video URL or upload a clip.');
      return;
    }
    const key = await resolveKey();
    if (!key) return;
    setLoading(true);
    setError('');
    setPaywall(null);
    try {
      let sphere = sphereUrl;
      if (!sphere) {
        const blob = ownSphere ? await prepareUploadFile(ownSphere.file) : await sphereBlob(preset, 512);
        sphere = await toDataUrl(blob);
        setUploadedSphere({ key: sphereKey, url: sphere });
      }
      const body = { ...payload, reference_image_url: sphere };
      const resp = await fetch('/v1/videos/generations?async=true', {
        method: 'POST',
        headers: { Authorization: `Bearer ${key}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      let data = await responseJSON(resp);
      const gate = paywallFromResponse(resp, data);
      if (gate) { setPaywall(gate); return; }
      if (!resp.ok) throw new Error(apiError(data, `Video generation failed (${resp.status})`));
      if (data?.status === 'failed') throw new Error(apiError(data, 'Video generation failed'));
      if (data?.id && data?.status !== 'completed') data = await pollVideoJob(key, data.id);
      const result = data?.result || data;
      if (!result?.video_url) throw new Error('The API completed without returning a video URL.');
      setOutputUrl(result.video_url);
      setGenerated(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Video generation failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="grid gap-0 lg:grid-cols-[minmax(0,1.05fr)_minmax(0,0.95fr)]" data-testid="mp-relight-panel">
      <div className="border-b border-white/20 lg:border-b-0 lg:border-r">
        <div className="relative bg-black">
          <video key={outputUrl} src={outputUrl} controls muted loop playsInline preload="auto" className="aspect-video w-full bg-black object-contain" data-testid="mp-relight-output" />
          <div className="absolute left-3 top-3 rounded border border-white/15 bg-black/75 px-2 py-1 font-mono text-[10px] uppercase tracking-[0.14em] text-white/65 backdrop-blur">
            {generated ? 'Relit output' : 'Example output'}
          </div>
        </div>
        <div className="grid gap-4 border-t border-white/20 p-4 sm:grid-cols-[160px_minmax(0,1fr)]">
          <div>
            <div className="mb-2 text-[10px] font-mono uppercase tracking-[0.16em] text-white/50">Lighting sphere</div>
            {ownSphere ? (
              <img src={ownSphere.preview} alt="Uploaded lighting sphere" className="aspect-square w-full rounded border border-white/20 bg-black object-contain" data-testid="mp-relight-sphere" />
            ) : (
              <canvas ref={bigRef} className="aspect-square w-full rounded border border-white/20 bg-black" data-testid="mp-relight-sphere" />
            )}
          </div>
          <div className="min-w-0">
            <div className="mb-2 text-[10px] font-mono uppercase tracking-[0.16em] text-white/50">Source clip</div>
            <video src={videoUrl} controls muted playsInline preload="metadata" className="w-full rounded bg-black" data-testid="mp-relight-source" />
          </div>
        </div>
      </div>

      <div className="p-5">
        <div className="mb-4">
          <h2 className="text-xl font-bold tracking-tight">Relight with {modelName}</h2>
          <p className="mt-1 text-sm text-white/45">Pick a lighting sphere, add a clip, and run it here. The API request below stays in sync.</p>
        </div>

        <Field label="OpenPaths API key">
          <input type="password" value={apiKey} onChange={e => setApiKey(e.target.value)} placeholder="op-..." autoComplete="off" className={inputCls} data-testid="mp-relight-api-key" />
        </Field>

        <div className="mt-3">
          <Field label="Source video URL (2.3-15 seconds)">
            <input value={videoUrl} onChange={e => setVideoUrl(e.target.value)} placeholder="https://example.com/clip.mp4" className={inputCls} data-testid="mp-relight-video-url" />
            <input type="file" accept="video/*" disabled={uploading || loading} onChange={e => void uploadVideo(e.target.files?.[0])} className="mt-2 text-xs" data-testid="mp-relight-video-upload" />
          </Field>
        </div>

        <div className="mt-3">
          <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/55">Lighting</span>
          <div className="grid grid-cols-3 gap-2 sm:grid-cols-4">
            {SPHERE_PRESETS.map(p => (
              <SphereThumb key={p.id} preset={p} active={!ownSphere && presetId === p.id} onPick={() => { setOwnSphere(null); setPresetId(p.id); }} />
            ))}
            <SphereThumb preset={customPreset(az, el, color, intensity)} active={!ownSphere && presetId === 'custom'} onPick={() => { setOwnSphere(null); setPresetId('custom'); }} />
          </div>
        </div>

        {!ownSphere && presetId === 'custom' && (
          <div className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-4" data-testid="mp-relight-custom">
            <Field label={`Direction ${az}°`}>
              <input type="range" min={-180} max={180} value={az} onChange={e => setAz(Number(e.target.value))} className="w-full" data-testid="mp-relight-azimuth" />
            </Field>
            <Field label={`Elevation ${el}°`}>
              <input type="range" min={-80} max={80} value={el} onChange={e => setEl(Number(e.target.value))} className="w-full" data-testid="mp-relight-elevation" />
            </Field>
            <Field label="Colour">
              <input type="color" value={color} onChange={e => setColor(e.target.value)} className="h-9 w-full rounded border border-white/30 bg-transparent" data-testid="mp-relight-color" />
            </Field>
            <Field label={`Intensity ${intensity.toFixed(2)}`}>
              <input type="range" min={0.2} max={1.5} step={0.05} value={intensity} onChange={e => setIntensity(Number(e.target.value))} className="w-full" />
            </Field>
          </div>
        )}

        <div className="mt-3 flex items-center gap-2 text-xs text-white/60">
          <Upload className="h-3.5 w-3.5" />
          <label className="cursor-pointer underline decoration-white/30 hover:text-white">
            Upload your own sphere image
            <input type="file" accept="image/*" className="sr-only" onChange={e => pickOwnSphere(e.target.files?.[0])} data-testid="mp-relight-sphere-upload" />
          </label>
          {ownSphere && <button type="button" onClick={() => setOwnSphere(null)} className="text-white/45 hover:text-white">use a preset</button>}
        </div>

        <div className="my-4 grid grid-cols-2 gap-3 sm:grid-cols-3">
          <Field label="Resolution">
            <select value={resolution} onChange={e => setResolution(e.target.value)} className={selectCls} data-testid="mp-relight-resolution">
              {Object.keys(RATES).map(v => <option key={v} value={v} className="bg-black">{v}</option>)}
            </select>
          </Field>
          <Field label="Aspect">
            <select value={aspect} onChange={e => setAspect(e.target.value)} className={selectCls} data-testid="mp-relight-aspect">
              {spec.aspectRatios.map(v => <option key={v} value={v} className="bg-black">{v}</option>)}
            </select>
          </Field>
          <Field label="Seed">
            <input value={seed} onChange={e => setSeed(e.target.value)} inputMode="numeric" placeholder="random" className={inputCls} data-testid="mp-relight-seed" />
          </Field>
        </div>

        <p className="mb-3 text-sm text-white/80" data-testid="mp-relight-pricing">
          ${RATES[resolution].toFixed(3)} per second at {resolution}, billed on clip length (3-15 seconds, rounded up). A 5 second clip costs ${(RATES[resolution] * 5).toFixed(3)}.
        </p>

        <button type="button" onClick={generate} disabled={loading || uploading} className="inline-flex w-full items-center justify-center gap-2 rounded border border-white bg-white px-4 py-2.5 font-mono text-sm font-bold text-black transition-colors hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60" data-testid="mp-relight-generate">
          {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Wand2 className="h-4 w-4" />}
          {loading ? 'Relighting video...' : 'Relight video here'}
        </button>
        {paywall && <SubscribePrompt paywall={paywall} modelName={modelName} priceLabel={`$${RATES[resolution].toFixed(3)}/s`} onDismiss={() => setPaywall(null)} />}
        {error && <p className="mt-3 rounded border border-red-400/20 bg-red-400/10 px-3 py-2 text-xs font-mono text-red-200" role="alert">{error}</p>}
      </div>

      <div className="border-t border-white/20 p-5 lg:col-span-2">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
          <div>
            <h3 className="font-mono text-xs font-bold uppercase tracking-[0.14em] text-white/65">Video relighting API example</h3>
            <p className="mt-1 text-xs text-white/50">reference_image_url takes a public URL or a data URI; running here sends the selected sphere as a PNG data URI.</p>
          </div>
          <button type="button" onClick={copy} className="inline-flex items-center gap-2 rounded border border-white/12 px-3 py-2 font-mono text-xs text-white/60 transition-colors hover:border-white/50 hover:text-white">
            {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
            {copied ? 'Copied' : 'Copy request'}
          </button>
        </div>
        <div className="mb-3 flex flex-wrap gap-2">
          {(['python', 'javascript', 'curl'] as const).map(l => (
            <button key={l} type="button" onClick={() => setLang(l)} className={`rounded px-3 py-1.5 font-mono text-xs transition-colors ${lang === l ? 'bg-white text-black' : 'border border-white/20 text-white/45 hover:text-white'}`}>
              {l === 'javascript' ? 'JavaScript' : l === 'python' ? 'Python' : 'cURL'}
            </button>
          ))}
        </div>
        <CodeBlock code={snippet} language={lang === 'curl' ? 'bash' : lang} containerClassName="overflow-hidden rounded-lg border border-white/20 bg-black/60" preClassName="max-h-[420px] text-xs" />
      </div>
    </div>
  );
}
