import React, { useEffect, useMemo, useState } from 'react';
import { Check, Copy, Loader2, Plus, Trash2, Upload, Wand2, X } from 'lucide-react';
import { CodeBlock } from './CodeBlock';
import { SubscribePrompt } from './SubscribePrompt';
import { ensureSpaceApiKey, paywallFromResponse, type Paywall } from '../lib/paywall';
import { AUTH_EVENT } from '../lib/api';
import { prepareUploadFile } from '../lib/imageUpload';
import { normalizeUploadedAssetUrl } from '../lib/uploadUrls';
import { Field, apiError, inputCls, pollVideoJob, responseJSON, selectCls, snippetFor } from './VideoSpacePanel';
import {
  FLUX3_MAX_ASPECTS,
  FLUX3_MAX_EXAMPLES,
  FLUX3_MAX_FPS,
  FLUX3_MAX_MAX_KEYFRAMES,
  FLUX3_MAX_MAX_SECONDS,
  FLUX3_MAX_MIN_SECONDS,
  FLUX3_MAX_MODEL_ID,
  FLUX3_MAX_RATES,
  type Flux3MaxMode,
} from '../data/flux3MaxExamples';

type Lang = 'python' | 'javascript' | 'curl';
type Keyframe = { id: number; url: string; index: number };

const MODES: { id: Flux3MaxMode; label: string }[] = [
  { id: 'text', label: 'Text → Video' },
  { id: 'image', label: 'Image → Video' },
  { id: 'keyframes', label: 'Keyframes → Video' },
];

let nextKeyframeId = 1;
const keyframeRows = (rows: { url: string; index: number }[]): Keyframe[] => rows.map(row => ({ id: nextKeyframeId++, ...row }));

function storedAPIKey() {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem('op_api_key') || '';
}

function FrameSlot({ label, hint, url, disabled, onChange, upload, uploading, testId }: {
  label: string; hint: string; url: string; disabled: boolean; testId: string; uploading: boolean;
  onChange: (url: string) => void; upload: (file?: File) => void;
}) {
  return (
    <div className="rounded border border-white/15 bg-white/[0.03] p-2.5">
      <div className="mb-2 flex items-center justify-between font-mono text-[10px] uppercase tracking-wider text-white/55">
        <span>{label}</span>
        {url && (
          <button type="button" disabled={disabled} onClick={() => onChange('')} className="inline-flex items-center gap-1 text-white/45 hover:text-white" data-testid={`${testId}-clear`}>
            <X className="h-3 w-3" /> clear
          </button>
        )}
      </div>
      {url
        ? <img src={url} alt={label} className="mb-2 max-h-40 w-full rounded bg-black object-contain" data-testid={`${testId}-preview`} />
        : <p className="mb-2 rounded bg-black/40 px-3 py-4 text-center text-xs text-white/40">{hint}</p>}
      <div className="flex gap-2">
        <input value={url} disabled={disabled} onChange={e => onChange(e.target.value)} placeholder="https://example.com/frame.webp" className={inputCls} data-testid={`${testId}-url`} />
        <label className="inline-flex shrink-0 cursor-pointer items-center justify-center rounded border border-white/12 px-3 text-white/55 transition-colors hover:border-white/50 hover:text-white">
          {uploading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
          <span className="sr-only">Upload {label}</span>
          <input type="file" accept="image/*" disabled={disabled} className="sr-only" onChange={e => { upload(e.target.files?.[0]); e.target.value = ''; }} data-testid={`${testId}-upload`} />
        </label>
      </div>
    </div>
  );
}

export function Flux3MaxSpacePanel({ modelName, priceLabel }: { modelName: string; priceLabel?: string }) {
  const [mode, setMode] = useState<Flux3MaxMode>('text');
  const [prompt, setPrompt] = useState(FLUX3_MAX_EXAMPLES.text.prompt);
  const [startUrl, setStartUrl] = useState('');
  const [endUrl, setEndUrl] = useState('');
  const [keyframes, setKeyframes] = useState<Keyframe[]>(() => keyframeRows([{ url: '', index: 0 }, { url: '', index: 5 * FLUX3_MAX_FPS }]));
  const [duration, setDuration] = useState(5);
  const [resolution, setResolution] = useState<keyof typeof FLUX3_MAX_RATES>('720p');
  const [aspect, setAspect] = useState<(typeof FLUX3_MAX_ASPECTS)[number]>('auto');
  const [reasoning, setReasoning] = useState<'low' | 'high'>('high');
  const [audio, setAudio] = useState(true);
  const [seed, setSeed] = useState('');
  const [outputUrl, setOutputUrl] = useState(FLUX3_MAX_EXAMPLES.text.outputUrl);
  const [posterUrl, setPosterUrl] = useState<string | undefined>(FLUX3_MAX_EXAMPLES.text.posterUrl);
  const [generated, setGenerated] = useState(false);
  const [lang, setLang] = useState<Lang>('python');
  const [copied, setCopied] = useState(false);
  const [apiKey, setApiKey] = useState(storedAPIKey);
  const [loading, setLoading] = useState(false);
  const [uploadingSlot, setUploadingSlot] = useState<string | null>(null);
  const [error, setError] = useState('');
  const [paywall, setPaywall] = useState<Paywall | null>(null);

  const maxIndex = duration * FLUX3_MAX_FPS;
  const priceUSD = FLUX3_MAX_RATES[resolution] * duration;

  useEffect(() => {
    if (apiKey.trim()) localStorage.setItem('op_api_key', apiKey.trim());
  }, [apiKey]);

  useEffect(() => {
    const sync = () => {
      const key = storedAPIKey();
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

  const resolveKey = async () => {
    const key = apiKey.trim() || await ensureSpaceApiKey();
    if (key && key !== apiKey) setApiKey(key);
    if (!key) setPaywall({ kind: 'login', message: 'Sign in to run this model.', subscribeUrl: '/pricing' });
    return key;
  };

  const showExample = (next: Flux3MaxMode) => {
    const example = FLUX3_MAX_EXAMPLES[next];
    setMode(next);
    setPrompt(example.prompt);
    setStartUrl(example.imageUrl || '');
    setEndUrl('');
    if (example.keyframes) setKeyframes(keyframeRows(example.keyframes.map(frame => ({ url: frame.imageUrl, index: frame.frameIndex }))));
    setDuration(5);
    setOutputUrl(example.outputUrl);
    setPosterUrl(example.posterUrl);
    setGenerated(false);
    setError('');
  };

  const changeDuration = (next: number) => {
    setDuration(next);
    setKeyframes(rows => rows.map(row => ({ ...row, index: Math.min(row.index, next * FLUX3_MAX_FPS) })));
  };

  const updateKeyframe = (id: number, patch: Partial<Keyframe>) => setKeyframes(rows => rows.map(row => (row.id === id ? { ...row, ...patch } : row)));

  const addKeyframe = () => setKeyframes(rows => {
    if (rows.length >= FLUX3_MAX_MAX_KEYFRAMES) return rows;
    const used = new Set(rows.map(row => row.index));
    let index = Math.round(maxIndex / 2);
    while (used.has(index) && index < maxIndex) index += 1;
    while (used.has(index) && index > 0) index -= 1;
    return [...rows, { id: nextKeyframeId++, url: '', index }];
  });

  const spaceEvenly = () => setKeyframes(rows => rows.map((row, i) => ({ ...row, index: rows.length > 1 ? Math.round((maxIndex * i) / (rows.length - 1)) : 0 })));

  const payload = useMemo(() => {
    const body: Record<string, unknown> = {
      model: FLUX3_MAX_MODEL_ID,
      prompt,
      duration,
      resolution,
      aspect_ratio: aspect,
      reasoning_level: reasoning,
      generate_audio: audio,
    };
    if (mode === 'image') {
      body.image_url = startUrl.trim() || 'https://example.com/first-frame.webp';
      if (endUrl.trim()) body.end_image_url = endUrl.trim();
    }
    if (mode === 'keyframes') {
      body.keyframes = keyframes.map(frame => ({ image_url: frame.url.trim() || 'https://example.com/keyframe.webp', frame_index: frame.index }));
    }
    const parsedSeed = Number(seed.trim());
    if (seed.trim() && Number.isFinite(parsedSeed)) body.seed = Math.trunc(parsedSeed);
    return body;
  }, [mode, prompt, duration, resolution, aspect, reasoning, audio, startUrl, endUrl, keyframes, seed]);
  const snippet = useMemo(() => snippetFor(payload, lang, apiKey), [payload, lang, apiKey]);

  const copy = async () => {
    await navigator.clipboard.writeText(snippet);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  };

  const uploadImage = async (slot: string, file: File | undefined, apply: (url: string) => void) => {
    if (!file) return;
    const key = await resolveKey();
    if (!key) return;
    setUploadingSlot(slot);
    setError('');
    setPaywall(null);
    try {
      const form = new FormData();
      form.append('file', await prepareUploadFile(file));
      const resp = await fetch('/v1/files/upload', { method: 'POST', headers: { Authorization: `Bearer ${key}` }, body: form });
      const data = await responseJSON(resp);
      const gate = paywallFromResponse(resp, data);
      if (gate) { setPaywall(gate); return; }
      if (!resp.ok || !data?.url) throw new Error(apiError(data, 'Image upload failed'));
      apply(normalizeUploadedAssetUrl(data.url));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Image upload failed');
    } finally {
      setUploadingSlot(null);
    }
  };

  const validate = (): string => {
    if (!prompt.trim()) return 'Enter a prompt before generating.';
    if (mode === 'image' && !startUrl.trim()) return 'Add a start image before generating.';
    if (mode === 'keyframes') {
      if (keyframes.some(frame => !frame.url.trim())) return 'Every keyframe needs an image.';
      if (new Set(keyframes.map(frame => frame.index)).size !== keyframes.length) return 'Keyframe positions must be unique.';
    }
    return '';
  };

  const generate = async () => {
    const problem = validate();
    if (problem) { setError(problem); return; }
    const key = await resolveKey();
    if (!key) return;
    setLoading(true);
    setError('');
    setPaywall(null);
    try {
      const resp = await fetch('/v1/videos/generations?async=true', {
        method: 'POST',
        headers: { Authorization: `Bearer ${key}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
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
      setPosterUrl(undefined);
      setGenerated(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Video generation failed');
    } finally {
      setLoading(false);
    }
  };

  const busy = loading || uploadingSlot !== null;

  return (
    <div className="grid grid-cols-[minmax(0,1fr)] gap-0 lg:grid-cols-[minmax(0,1fr)_minmax(360px,440px)]" data-testid="mp-video-panel">
      <div className="relative border-b border-white/20 bg-black lg:border-b-0 lg:border-r">
        <video
          key={outputUrl}
          src={outputUrl}
          poster={posterUrl}
          controls
          muted
          loop
          playsInline
          preload="metadata"
          className="mx-auto aspect-video w-full bg-black object-contain lg:max-h-[calc(100vh-8.5rem)] lg:min-h-[320px]"
          data-testid="mp-video-output"
        />
        <div className="absolute left-3 top-3 flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.12em] text-white/70">
          <span className="rounded border border-white/15 bg-black/75 px-2 py-1 backdrop-blur">{generated ? 'Generated output' : 'Example output'}</span>
        </div>
        <div className="flex flex-wrap items-center gap-2 border-t border-white/15 px-3 py-2.5">
          <span className="font-mono text-[10px] uppercase tracking-wider text-white/45">Load example</span>
          {MODES.map(({ id, label }) => (
            <button key={id} type="button" onClick={() => showExample(id)} disabled={busy}
              className="rounded border border-white/20 px-2.5 py-1 font-mono text-[11px] text-white/65 transition-colors hover:border-white/50 hover:text-white disabled:opacity-50"
              data-testid={`flux3-example-${id}`}>
              {label}
            </button>
          ))}
        </div>
      </div>

      <div className="p-4">
        <h2 className="sr-only">Generate with {modelName}</h2>

        <div className="mb-3 grid grid-cols-3 gap-1 rounded border border-white/20 bg-black p-1" role="tablist">
          {MODES.map(({ id, label }) => (
            <button key={id} type="button" role="tab" aria-selected={mode === id} disabled={busy}
              onClick={() => { setMode(id); setError(''); }}
              className={`rounded px-2 py-1.5 font-mono text-[11px] transition-colors ${mode === id ? 'bg-white font-bold text-black' : 'text-white/55 hover:text-white'}`}
              data-testid={`flux3-mode-${id}`}>
              {label}
            </button>
          ))}
        </div>

        <Field label={mode === 'text' ? 'Prompt' : 'Motion prompt'}>
          <textarea value={prompt} onChange={e => setPrompt(e.target.value)} rows={6} className={`${inputCls} resize-y text-sm leading-relaxed`} data-testid="mp-video-prompt" />
        </Field>

        {mode === 'image' && (
          <div className="mt-3 space-y-2.5">
            <FrameSlot label="Start frame" hint="The video starts from this image." url={startUrl} disabled={busy} testId="flux3-start"
              uploading={uploadingSlot === 'start'} onChange={setStartUrl} upload={file => void uploadImage('start', file, setStartUrl)} />
            <FrameSlot label="End frame (optional)" hint="The video lands on this image." url={endUrl} disabled={busy} testId="flux3-end"
              uploading={uploadingSlot === 'end'} onChange={setEndUrl} upload={file => void uploadImage('end', file, setEndUrl)} />
          </div>
        )}

        {mode === 'keyframes' && (
          <div className="mt-3 space-y-2.5">
            {keyframes.map((frame, i) => (
              <div key={frame.id} data-testid={`flux3-keyframe-${i}`}>
                <FrameSlot label={`Keyframe ${i + 1}`} hint="Pinned to the frame position below." url={frame.url} disabled={busy} testId={`flux3-kf${i}`}
                  uploading={uploadingSlot === `kf${frame.id}`} onChange={url => updateKeyframe(frame.id, { url })}
                  upload={file => void uploadImage(`kf${frame.id}`, file, url => updateKeyframe(frame.id, { url }))} />
                <div className="mt-1.5 grid grid-cols-[minmax(0,1fr)_auto] items-end gap-2">
                  <Field label={`Frame index · ${(frame.index / FLUX3_MAX_FPS).toFixed(2)} s`}>
                    <input type="number" min={0} max={maxIndex} step={1} disabled={busy} value={frame.index} className={inputCls} data-testid={`flux3-kf${i}-index`}
                      onChange={e => updateKeyframe(frame.id, { index: Math.max(0, Math.min(maxIndex, Math.round(Number(e.target.value) || 0))) })} />
                  </Field>
                  <button type="button" disabled={busy || keyframes.length <= 1} onClick={() => setKeyframes(rows => rows.filter(row => row.id !== frame.id))}
                    className="inline-flex items-center gap-1 rounded border border-white/20 px-3 py-1.5 font-mono text-xs text-white/55 hover:text-white disabled:opacity-40">
                    <Trash2 className="h-3.5 w-3.5" /> Remove
                  </button>
                </div>
              </div>
            ))}
            <div className="flex flex-wrap gap-2">
              <button type="button" disabled={busy || keyframes.length >= FLUX3_MAX_MAX_KEYFRAMES} onClick={addKeyframe}
                className="inline-flex items-center gap-1 rounded border border-white/20 px-3 py-1.5 font-mono text-xs text-white/65 hover:text-white disabled:opacity-40" data-testid="flux3-add-keyframe">
                <Plus className="h-3.5 w-3.5" /> Add keyframe ({keyframes.length}/{FLUX3_MAX_MAX_KEYFRAMES})
              </button>
              <button type="button" disabled={busy || keyframes.length < 2} onClick={spaceEvenly}
                className="rounded border border-white/20 px-3 py-1.5 font-mono text-xs text-white/65 hover:text-white disabled:opacity-40">
                Space evenly
              </button>
            </div>
            <p className="font-mono text-[10px] text-white/45">{FLUX3_MAX_FPS} fps · positions 0 to {maxIndex} at {duration} s · unique indexes</p>
          </div>
        )}

        <div className="mt-3">
          <Field label={`Duration · ${duration} seconds`}>
            <input type="range" min={FLUX3_MAX_MIN_SECONDS} max={FLUX3_MAX_MAX_SECONDS} step={1} value={duration} disabled={busy} onChange={e => changeDuration(Number(e.target.value))} className="w-full accent-white" data-testid="mp-video-duration" />
          </Field>
        </div>

        <div className="my-3 grid grid-cols-2 gap-2.5 sm:grid-cols-4">
          <Field label="Resolution">
            <select value={resolution} onChange={e => setResolution(e.target.value as keyof typeof FLUX3_MAX_RATES)} className={selectCls} data-testid="mp-video-resolution">
              {(Object.keys(FLUX3_MAX_RATES) as (keyof typeof FLUX3_MAX_RATES)[]).map(v => <option key={v} value={v} className="bg-black">{v}</option>)}
            </select>
          </Field>
          <Field label="Aspect">
            <select value={aspect} onChange={e => setAspect(e.target.value as (typeof FLUX3_MAX_ASPECTS)[number])} className={selectCls} data-testid="mp-video-aspect-ratio">
              {FLUX3_MAX_ASPECTS.map(v => <option key={v} value={v} className="bg-black">{v}</option>)}
            </select>
          </Field>
          <Field label="Reasoning">
            <select value={reasoning} onChange={e => setReasoning(e.target.value as 'low' | 'high')} className={selectCls} data-testid="flux3-reasoning">
              <option value="high" className="bg-black">high</option>
              <option value="low" className="bg-black">low</option>
            </select>
          </Field>
          <Field label="Audio">
            <button type="button" onClick={() => setAudio(v => !v)}
              className={`w-full rounded border px-3 py-2 font-mono text-xs transition-colors ${audio ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-200' : 'border-white/20 bg-black text-white/50'}`}
              data-testid="mp-video-generate-audio">
              {audio ? 'on' : 'off'}
            </button>
          </Field>
        </div>
        <Field label="Seed">
          <input value={seed} onChange={e => setSeed(e.target.value.replace(/[^0-9]/g, '').slice(0, 10))} inputMode="numeric" placeholder="random" className={inputCls} data-testid="mp-video-seed" />
        </Field>

        <p className="mt-3 font-mono text-xs text-white/60" data-testid="flux3-price">
          ${priceUSD.toFixed(2)} for this clip · {duration} s × ${FLUX3_MAX_RATES[resolution].toFixed(2)}/s at {resolution}
        </p>

        <button
          type="button"
          onClick={generate}
          disabled={busy}
          className="mt-3 inline-flex w-full items-center justify-center gap-2 rounded border border-white bg-white px-4 py-2.5 font-mono text-sm font-bold text-black transition-colors hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60"
          data-testid="mp-video-generate"
        >
          {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Wand2 className="h-4 w-4" />}
          {loading ? 'Generating video…' : 'Generate video here'}
        </button>
        {paywall && <SubscribePrompt paywall={paywall} modelName={modelName} priceLabel={priceLabel} onDismiss={() => setPaywall(null)} />}
        {error && <p className="mt-3 rounded border border-red-400/20 bg-red-400/10 px-3 py-2 text-xs font-mono text-red-200" role="alert">{error}</p>}
      </div>

      <div className="border-t border-white/20 p-4 lg:col-span-2">
        <div className="mb-3 flex flex-wrap items-end justify-between gap-3">
          <div>
            <h3 className="font-mono text-xs font-bold uppercase tracking-[0.14em] text-white/65">Video API example</h3>
            <p className="mt-1 text-xs text-white/50">One endpoint for text, image and keyframes. Mode, prompt, frames and settings update this code immediately.</p>
          </div>
          <div className="w-full sm:ml-auto sm:w-64">
            <Field label="OpenPaths API key">
              <input type="password" value={apiKey} onChange={e => setApiKey(e.target.value)} placeholder="op-..." autoComplete="off" className={inputCls} data-testid="mp-video-api-key" />
            </Field>
          </div>
          <button type="button" onClick={copy} className="inline-flex items-center gap-2 rounded border border-white/12 px-3 py-2 font-mono text-xs text-white/60 transition-colors hover:border-white/50 hover:text-white">
            {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
            {copied ? 'Copied' : 'Copy request'}
          </button>
        </div>
        <div className="mb-3 flex flex-wrap gap-2">
          {(['python', 'javascript', 'curl'] as const).map(l => (
            <button key={l} type="button" onClick={() => setLang(l)}
              className={`rounded px-3 py-1.5 font-mono text-xs transition-colors ${lang === l ? 'bg-white text-black' : 'border border-white/20 text-white/45 hover:text-white'}`}>
              {l === 'javascript' ? 'JavaScript' : l === 'python' ? 'Python' : 'cURL'}
            </button>
          ))}
        </div>
        <CodeBlock code={snippet} language={lang === 'curl' ? 'bash' : lang} containerClassName="overflow-hidden rounded-lg border border-white/20 bg-black/60" preClassName="max-h-[420px] text-xs" />
      </div>
    </div>
  );
}
