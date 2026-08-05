import React, { useEffect, useMemo, useState } from 'react';
import { Check, Copy, Loader2, Upload, Video, Wand2 } from 'lucide-react';
import { CodeBlock } from './CodeBlock';
import type { VideoDemo } from '../data/videoDemos';
import {
  getVideoParamSpec,
  EMPTY_ADVANCED,
  type VideoAdvancedValues,
  type VideoInputMode,
  type VideoParamSpec,
} from '../data/videoModelParams';
import { buildVideoPayload, type VideoBaseValues } from '../lib/videoPayload';
import { prepareUploadFile } from '../lib/imageUpload';
import { normalizeUploadedAssetUrl } from '../lib/uploadUrls';

type Lang = 'python' | 'javascript' | 'curl';

const DEFAULT_PROMPT =
  'A cinematic product demo shot of a compact AI routing console, slow camera push-in, clean studio lighting, no readable text.';
const DEFAULT_MOTION_PROMPT =
  'Slow cinematic aerial push-in toward the sunlit coastline, waves rolling and crashing against the cliffs, clouds drifting naturally, stable composition, no added text.';
const DEFAULT_INPUT_IMAGE = 'https://openpaths.io/static/blog/video-tips/coast-poster.webp';
const DEFAULT_OUTPUT_VIDEO = 'https://openpaths.io/static/blog/video-tips/coast.mp4';

function starterDemo(modelId: string, spec: VideoParamSpec): VideoDemo {
  const imageToVideo = spec.imageToVideo;
  return {
    prompt: imageToVideo ? DEFAULT_MOTION_PROMPT : DEFAULT_PROMPT,
    outputUrl: DEFAULT_OUTPUT_VIDEO,
    resolution: spec.resolutions[0] as VideoDemo['resolution'],
    duration: (imageToVideo ? '6' : spec.durations[0]) as VideoDemo['duration'],
    aspectRatio: '16:9',
    generateAudio: false,
    imageUrl: imageToVideo ? DEFAULT_INPUT_IMAGE : undefined,
    imageUrls: spec.referenceToVideo ? [DEFAULT_INPUT_IMAGE] : undefined,
  };
}

function starterInputMode(starter: VideoDemo, spec: VideoParamSpec): VideoInputMode {
  if (starter.videoUrls?.length && spec.inputModes?.includes('video-to-video')) return 'video-to-video';
  if ((starter.imageUrl || starter.imageUrls?.length) && spec.inputModes?.includes('image-to-video')) return 'image-to-video';
  return spec.inputModes?.[0] || 'text-to-video';
}

function storedAPIKey() {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem('op_api_key') || '';
}

function snippetFor(payload: Record<string, unknown>, lang: Lang, apiKey: string): string {
  const apiBase = 'https://openpaths.io/v1';
  const bearer = apiKey.trim() || 'op-...';
  const body = JSON.stringify(payload, null, 2);
  if (lang === 'python') {
    return `import json
import time
from openai import OpenAI

client = OpenAI(
    api_key="${bearer}",
    base_url="${apiBase}",
)

result = client.post(
    "/videos/generations",
    body=json.loads(r'''${body}'''),
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
print(video["video_url"])`;
  }
  if (lang === 'javascript') {
    return `import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "${bearer}",
  baseURL: "${apiBase}",
});

let result = await client.post("/videos/generations", {
  body: ${body},
});

while (result.status && !["completed", "failed"].includes(result.status)) {
  await new Promise((resolve) => setTimeout(resolve, 2000));
  result = await client.get(\`/videos/generations/\${result.id}\`);
}

if (result.status === "failed") {
  throw new Error(result.error?.message || "Video generation failed");
}

console.log((result.result ?? result).video_url);`;
  }
  return `curl "${apiBase}/videos/generations" \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${bearer}" \\
  -d @- <<'JSON'
${body}
JSON`;
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">{label}</span>
      {children}
    </label>
  );
}

const selectCls =
  'w-full bg-black border border-white/10 rounded px-3 py-2 text-xs font-mono text-white focus:outline-none focus:border-white/30';
const inputCls = selectCls + ' placeholder:text-white/25';

async function responseJSON(resp: Response) {
  const text = await resp.text();
  if (!text) return {};
  try {
    return JSON.parse(text);
  } catch {
    throw new Error(resp.ok ? 'The API returned invalid JSON.' : text.slice(0, 240));
  }
}

function apiError(data: any, fallback: string) {
  if (typeof data?.error === 'string') return data.error;
  return data?.error?.message || fallback;
}

async function pollVideoJob(apiKey: string, id: string): Promise<any> {
  const deadline = Date.now() + 15 * 60 * 1000;
  while (Date.now() < deadline) {
    await new Promise(resolve => window.setTimeout(resolve, 2000));
    const resp = await fetch(`/v1/videos/generations/${encodeURIComponent(id)}`, {
      headers: { Authorization: `Bearer ${apiKey}` },
    });
    const data = await responseJSON(resp);
    if (!resp.ok) throw new Error(apiError(data, `Video status request failed (${resp.status})`));
    if (data?.status === 'completed') return data.result || data;
    if (data?.status === 'failed') throw new Error(apiError(data, 'Video generation failed'));
  }
  throw new Error('Video generation timed out after 15 minutes.');
}

export function VideoSpacePanel({ modelId, modelName, demo }: { modelId: string; modelName: string; demo?: VideoDemo }) {
  const spec = useMemo(() => getVideoParamSpec(modelId), [modelId]);
  const starter = useMemo(() => demo || starterDemo(modelId, spec), [demo, modelId, spec]);

  const [lang, setLang] = useState<Lang>('python');
  const [copied, setCopied] = useState(false);
  const [apiKey, setApiKey] = useState(storedAPIKey);
  const [prompt, setPrompt] = useState(starter.prompt);
  const [inputMode, setInputMode] = useState<VideoInputMode>(() => starterInputMode(starter, spec));
  const [imageUrl, setImageUrl] = useState(starter.imageUrl || starter.imageUrls?.[0] || '');
  const [endImageUrl, setEndImageUrl] = useState(starter.endImageUrl || '');
  const [videoUrl, setVideoUrl] = useState(starter.videoUrls?.[0] || '');
  const [resolution, setResolution] = useState(starter.resolution);
  const [duration, setDuration] = useState<string>(starter.duration);
  const [aspectRatio, setAspectRatio] = useState(starter.aspectRatio);
  const [generateAudio, setGenerateAudio] = useState(starter.generateAudio);
  const [adv, setAdv] = useState<VideoAdvancedValues>(EMPTY_ADVANCED);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [outputUrl, setOutputUrl] = useState(starter.outputUrl);
  const [generated, setGenerated] = useState(false);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    setPrompt(starter.prompt);
    setInputMode(starterInputMode(starter, spec));
    setImageUrl(starter.imageUrl || starter.imageUrls?.[0] || '');
    setEndImageUrl(starter.endImageUrl || '');
    setVideoUrl(starter.videoUrls?.[0] || '');
    setResolution(starter.resolution);
    setDuration(starter.duration);
    setAspectRatio(starter.aspectRatio);
    setGenerateAudio(starter.generateAudio);
    setOutputUrl(starter.outputUrl);
    setGenerated(false);
    setAdv(EMPTY_ADVANCED);
    setError('');
  }, [starter, spec]);

  useEffect(() => {
    if (apiKey.trim()) localStorage.setItem('op_api_key', apiKey.trim());
  }, [apiKey]);

  const setAdvField = (k: keyof VideoAdvancedValues, v: string) => setAdv(prev => ({ ...prev, [k]: v }));

  const base: VideoBaseValues = {
    prompt,
    resolution,
    duration,
    aspectRatio,
    generateAudio,
    imageUrl: spec.inputModes ? (inputMode === 'image-to-video' ? imageUrl : undefined) : (spec.imageToVideo ? imageUrl : undefined),
    endImageUrl: spec.inputModes && inputMode !== 'image-to-video' ? undefined : endImageUrl,
    videoUrl: spec.inputModes && inputMode === 'video-to-video' ? videoUrl : undefined,
    imageUrls: spec.referenceToVideo && imageUrl ? [imageUrl] : starter.imageUrls,
    videoUrls: starter.videoUrls,
    audioUrls: starter.audioUrls,
  };
  const payload = useMemo(
    () => buildVideoPayload(modelId, base, adv, spec),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [modelId, spec, prompt, resolution, duration, aspectRatio, generateAudio, inputMode, imageUrl, endImageUrl, videoUrl, adv, starter],
  );
  const snippet = useMemo(() => snippetFor(payload, lang, apiKey), [payload, lang, apiKey]);

  const copy = async () => {
    await navigator.clipboard.writeText(snippet);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  };

  const uploadImage = async (file?: File) => {
    if (!file) return;
    if (!apiKey.trim()) {
      setError('Add your OpenPaths API key before uploading an image.');
      return;
    }
    setUploading(true);
    setError('');
    try {
      const form = new FormData();
      form.append('file', await prepareUploadFile(file));
      const resp = await fetch('/v1/files/upload', {
        method: 'POST',
        headers: { Authorization: `Bearer ${apiKey.trim()}` },
        body: form,
      });
      const data = await responseJSON(resp);
      if (!resp.ok || !data?.url) throw new Error(apiError(data, 'Image upload failed'));
      setImageUrl(normalizeUploadedAssetUrl(data.url));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Image upload failed');
    } finally {
      setUploading(false);
    }
  };

  const generate = async () => {
    if (!apiKey.trim()) {
      setError('Add your OpenPaths API key before generating.');
      return;
    }
    if (!prompt.trim()) {
      setError('Enter a motion prompt before generating.');
      return;
    }
    if (((spec.inputModes && inputMode === 'image-to-video') || spec.imageToVideo || spec.referenceToVideo) && !imageUrl.trim()) {
      setError('Add an input image URL before generating with this model.');
      return;
    }
    if (spec.inputModes && inputMode === 'video-to-video' && !videoUrl.trim()) {
      setError('Add an input video URL before generating in video-to-video mode.');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const resp = await fetch('/v1/videos/generations?async=true', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${apiKey.trim()}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });
      let data = await responseJSON(resp);
      if (!resp.ok) throw new Error(apiError(data, `Video generation failed (${resp.status})`));
      if (data?.status === 'failed') throw new Error(apiError(data, 'Video generation failed'));
      if (data?.id && data?.status !== 'completed') data = await pollVideoJob(apiKey.trim(), data.id);
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

  const hasAdvanced =
    spec.seed ||
    spec.negativePrompt ||
    !!spec.numFrames ||
    !!spec.framesPerSecond ||
    !!spec.guidanceScale ||
    !!spec.numInferenceSteps ||
    !!spec.outputFormats;
  const usesImageInput = spec.inputModes ? inputMode === 'image-to-video' : spec.imageToVideo || spec.referenceToVideo;
  const previewImage = usesImageInput ? imageUrl || starter.imageUrls?.[0] : undefined;

  return (
    <div className="grid gap-0 lg:grid-cols-[minmax(0,1.05fr)_minmax(0,0.95fr)]" data-testid="mp-video-panel">
      <div className="border-b border-white/10 lg:border-b-0 lg:border-r">
        <div className="relative bg-black">
          <video
            key={outputUrl}
            src={outputUrl}
            controls
            muted
            loop
            playsInline
            preload="auto"
            poster={generated ? undefined : previewImage}
            className="aspect-video w-full bg-black object-contain"
            data-testid="mp-video-output"
          />
          <div className="absolute left-3 top-3 rounded border border-white/15 bg-black/75 px-2 py-1 font-mono text-[10px] uppercase tracking-[0.14em] text-white/65 backdrop-blur">
            {generated ? 'Generated output' : demo ? 'Example output' : 'Starter output preview'}
          </div>
        </div>
        <div className={`grid gap-4 border-t border-white/10 p-4 ${previewImage ? 'md:grid-cols-[140px_minmax(0,1fr)]' : ''}`}>
          {previewImage ? (
            <div>
              <div className="mb-2 text-[10px] font-mono uppercase tracking-[0.16em] text-white/35">Input image</div>
              <img
                src={previewImage}
                alt={`${modelName} reference input`}
                className="aspect-square w-full rounded-lg border border-white/10 bg-white/[0.03] object-contain"
                data-testid="mp-video-input-preview"
              />
            </div>
          ) : null}
          <div className="min-w-0">
            <div className="mb-2 text-[10px] font-mono uppercase tracking-[0.16em] text-white/35">Current prompt</div>
            <p className="text-sm leading-relaxed text-white/58">{prompt}</p>
            <div className="mt-4 flex flex-wrap gap-2 text-[10px] font-mono uppercase tracking-[0.14em] text-white/35">
              <span className="rounded border border-white/10 px-2 py-1">{resolution}</span>
              <span className="rounded border border-white/10 px-2 py-1">{duration === 'auto' ? 'auto duration' : `${duration}s`}</span>
              <span className="rounded border border-white/10 px-2 py-1">{aspectRatio}</span>
            </div>
          </div>
        </div>
      </div>

      <div className="p-5">
        <div className="mb-4">
          <h2 className="text-xl font-bold tracking-tight">Generate with {modelName}</h2>
          <p className="mt-1 text-sm text-white/45">Edit the example and run it here. The API request below stays in sync.</p>
        </div>

        <Field label="OpenPaths API key">
          <input
            type="password"
            value={apiKey}
            onChange={e => setApiKey(e.target.value)}
            placeholder="op-..."
            autoComplete="off"
            className={inputCls}
            data-testid="mp-video-api-key"
          />
        </Field>

        <div className="mt-3">
          <Field label={usesImageInput || inputMode === 'video-to-video' ? 'Motion prompt' : 'Prompt'}>
            <textarea
              value={prompt}
              onChange={e => setPrompt(e.target.value)}
              rows={5}
              className={`${inputCls} resize-y text-sm leading-relaxed`}
              data-testid="mp-video-prompt"
            />
          </Field>
        </div>

        {spec.inputModes && (
          <div className="mt-3 grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end">
            <Field label="Video mode">
              <select
                value={inputMode}
                onChange={e => {
                  const next = e.target.value as VideoInputMode;
                  setInputMode(next);
                  if (next === 'image-to-video' && !imageUrl) setImageUrl(DEFAULT_INPUT_IMAGE);
                }}
                className={selectCls}
                data-testid="mp-video-input-mode"
              >
                {spec.inputModes.map(mode => (
                  <option key={mode} value={mode} className="bg-black">
                    {mode === 'text-to-video' ? 'Text / Image → Video: text' : mode === 'image-to-video' ? 'Text / Image → Video: image' : 'Video → Video'}
                  </option>
                ))}
              </select>
            </Field>
            {spec.safetyTolerance !== undefined && (
              <div className="rounded border border-white/10 bg-white/[0.03] px-3 py-2 font-mono text-xs text-white/50" data-testid="mp-video-safety-tolerance">
                Safety tolerance <span className="text-white">{spec.safetyTolerance}</span> · fixed
              </div>
            )}
          </div>
        )}

        {usesImageInput && (
          <div className="mt-3">
            <Field label={spec.referenceToVideo ? 'Reference image URL' : 'Input image URL'}>
              <div className="flex gap-2">
                <input
                  value={imageUrl}
                  onChange={e => setImageUrl(e.target.value)}
                  placeholder="https://example.com/first-frame.webp"
                  className={inputCls}
                  data-testid="mp-video-image-url"
                />
                <label className="inline-flex shrink-0 cursor-pointer items-center justify-center rounded border border-white/12 px-3 text-white/55 transition-colors hover:border-white/30 hover:text-white">
                  {uploading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
                  <span className="sr-only">Upload input image</span>
                  <input type="file" accept="image/*" className="sr-only" onChange={e => uploadImage(e.target.files?.[0])} />
                </label>
              </div>
            </Field>
          </div>
        )}

        {usesImageInput && (spec.inputModes || (spec.imageToVideo && spec.endImage)) && (
          <div className="mt-3">
            <Field label="End image URL (optional)">
              <input value={endImageUrl} onChange={e => setEndImageUrl(e.target.value)} placeholder="https://example.com/last-frame.webp" className={inputCls} data-testid="mp-video-end-image-url" />
            </Field>
          </div>
        )}

        {spec.inputModes && inputMode === 'video-to-video' && (
          <div className="mt-3">
            <Field label="Input video URL">
              <input
                value={videoUrl}
                onChange={e => setVideoUrl(e.target.value)}
                placeholder="https://example.com/input-video.mp4"
                className={inputCls}
                data-testid="mp-video-video-url"
              />
            </Field>
          </div>
        )}

        <div className="my-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
          <Field label="Resolution">
            <select value={resolution} onChange={e => setResolution(e.target.value as typeof resolution)} className={selectCls} data-testid="mp-video-resolution">
              {spec.resolutions.map(v => <option key={v} value={v} className="bg-black">{v}</option>)}
            </select>
          </Field>
          <Field label="Duration">
            <select value={duration} onChange={e => setDuration(e.target.value)} className={selectCls} data-testid="mp-video-duration">
              {spec.durations.map(v => <option key={v} value={v} className="bg-black">{v}</option>)}
            </select>
          </Field>
          <Field label="Aspect">
            <select value={aspectRatio} onChange={e => setAspectRatio(e.target.value as typeof aspectRatio)} className={selectCls} data-testid="mp-video-aspect-ratio">
              {spec.aspectRatios.map(v => <option key={v} value={v} className="bg-black">{v}</option>)}
            </select>
          </Field>
          <Field label={spec.enableSafetyChecker ? 'Safety' : 'Audio'}>
            <button
              type="button"
              onClick={() => spec.generateAudio && setGenerateAudio(v => !v)}
              className={`w-full rounded border px-3 py-2 font-mono text-xs transition-colors ${
                spec.enableSafetyChecker || generateAudio
                  ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-200'
                  : 'border-white/10 bg-black text-white/50'
              }`}
              data-testid="mp-video-generate-audio"
            >
              {spec.enableSafetyChecker || generateAudio ? 'on' : 'off'}
            </button>
          </Field>
        </div>

        {hasAdvanced && (
          <div className="mb-4">
            <button
              type="button"
              onClick={() => setShowAdvanced(v => !v)}
              className="mb-3 font-mono text-[11px] uppercase tracking-wider text-white/45 hover:text-white"
              data-testid="mp-video-advanced-toggle"
            >
              {showAdvanced ? '− Advanced args' : '+ Advanced args'}
            </button>
            {showAdvanced && (
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3" data-testid="mp-video-advanced">
                {spec.negativePrompt && (
                  <Field label="Negative prompt">
                    <input value={adv.negativePrompt} onChange={e => setAdvField('negativePrompt', e.target.value)} placeholder="blurry, distorted" className={inputCls} data-testid="mp-video-negative-prompt" />
                  </Field>
                )}
                {spec.seed && (
                  <Field label="Seed">
                    <input value={adv.seed} onChange={e => setAdvField('seed', e.target.value)} inputMode="numeric" placeholder="random" className={inputCls} data-testid="mp-video-seed" />
                  </Field>
                )}
                {spec.numFrames && (
                  <Field label="Num frames">
                    <input value={adv.numFrames} onChange={e => setAdvField('numFrames', e.target.value)} inputMode="numeric" placeholder={spec.numFrames.placeholder} className={inputCls} data-testid="mp-video-num-frames" />
                  </Field>
                )}
                {spec.framesPerSecond && (
                  <Field label="FPS">
                    <select value={adv.framesPerSecond} onChange={e => setAdvField('framesPerSecond', e.target.value)} className={selectCls} data-testid="mp-video-fps">
                      <option value="" className="bg-black">default</option>
                      {spec.framesPerSecond.options.map(o => <option key={o} value={o} className="bg-black">{o}</option>)}
                    </select>
                  </Field>
                )}
                {spec.guidanceScale && (
                  <Field label="Guidance">
                    <input value={adv.guidanceScale} onChange={e => setAdvField('guidanceScale', e.target.value)} inputMode="decimal" placeholder={spec.guidanceScale.placeholder} className={inputCls} data-testid="mp-video-guidance-scale" />
                  </Field>
                )}
                {spec.numInferenceSteps && (
                  <Field label="Steps">
                    <input value={adv.numInferenceSteps} onChange={e => setAdvField('numInferenceSteps', e.target.value)} inputMode="numeric" placeholder={spec.numInferenceSteps.placeholder} className={inputCls} data-testid="mp-video-num-inference-steps" />
                  </Field>
                )}
                {spec.outputFormats && (
                  <Field label="Format">
                    <select value={adv.outputFormat} onChange={e => setAdvField('outputFormat', e.target.value)} className={selectCls} data-testid="mp-video-output-format">
                      <option value="" className="bg-black">default</option>
                      {spec.outputFormats.map(o => <option key={o} value={o} className="bg-black">{o}</option>)}
                    </select>
                  </Field>
                )}
              </div>
            )}
          </div>
        )}

        <button
          type="button"
          onClick={generate}
          disabled={loading || uploading}
          className="inline-flex w-full items-center justify-center gap-2 rounded border border-white bg-white px-4 py-2.5 font-mono text-sm font-bold text-black transition-colors hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60"
          data-testid="mp-video-generate"
        >
          {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Wand2 className="h-4 w-4" />}
          {loading ? 'Generating video…' : 'Generate video here'}
        </button>
        {error && <p className="mt-3 rounded border border-red-400/20 bg-red-400/10 px-3 py-2 text-xs font-mono text-red-200" role="alert">{error}</p>}
      </div>

      <div className="border-t border-white/10 p-5 lg:col-span-2">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
          <div>
            <h3 className="font-mono text-xs font-bold uppercase tracking-[0.14em] text-white/65">Video API example</h3>
            <p className="mt-1 text-xs text-white/35">Prompt, image, and settings update this code immediately.</p>
          </div>
          <button type="button" onClick={copy} className="inline-flex items-center gap-2 rounded border border-white/12 px-3 py-2 font-mono text-xs text-white/60 transition-colors hover:border-white/30 hover:text-white">
            {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
            {copied ? 'Copied' : 'Copy request'}
          </button>
        </div>
        <div className="mb-3 flex flex-wrap gap-2">
          {(['python', 'javascript', 'curl'] as const).map(l => (
            <button
              key={l}
              type="button"
              onClick={() => setLang(l)}
              className={`rounded px-3 py-1.5 font-mono text-xs transition-colors ${lang === l ? 'bg-white text-black' : 'border border-white/10 text-white/45 hover:text-white'}`}
            >
              {l === 'javascript' ? 'JavaScript' : l === 'python' ? 'Python' : 'cURL'}
            </button>
          ))}
        </div>
        <CodeBlock
          code={snippet}
          language={lang === 'curl' ? 'bash' : lang}
          containerClassName="overflow-hidden rounded-lg border border-white/10 bg-black/60"
          preClassName="max-h-[420px] text-xs"
        />
        <div className="mt-3 flex items-center gap-2 text-[11px] text-white/32">
          <Video className="h-3.5 w-3.5" /> Output replaces the preview without leaving this model page.
        </div>
      </div>
    </div>
  );
}
