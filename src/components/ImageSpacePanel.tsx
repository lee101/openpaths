import React, { useEffect, useMemo, useState } from 'react';
import { Check, Copy, Image as ImageIcon, Loader2, Upload, Wand2 } from 'lucide-react';
import { CodeBlock } from './CodeBlock';
import type { ImageDemo } from '../data/imageDemos';
import { OPENPATHS_IMAGE_MODELS } from '../lib/artificialAnalysisImages';
import { prepareUploadFile } from '../lib/imageUpload';
import { normalizeUploadedAssetUrl } from '../lib/uploadUrls';

type Lang = 'python' | 'javascript' | 'curl';
type ImageResult = { url?: string; b64_json?: string };

const DEFAULT_PROMPT = 'A cinematic photograph of a red fox wearing a tiny astronaut helmet, sitting on a mossy rock in a foggy pine forest at golden hour, shallow depth of field, ultra detailed.';
const DEFAULT_OUTPUT = 'https://openpathsstatic.openpaths.io/static/blog/image-eval/zimage.webp';
const DEFAULT_INPUT = 'https://openpathsstatic.openpaths.io/static/uploads/playground/hidream-edit/perfume.jpg';
const BFL_IMAGE_SIZES = ['1024x1024', '1152x768', '768x1152', '1360x768', '768x1360', '1920x1088', '1088x1920', '2048x880', '2048x2048'];

function storedAPIKey() {
  if (typeof window === 'undefined') return '';
  return localStorage.getItem('op_api_key') || '';
}

function responseSource(result?: ImageResult) {
  if (result?.url) return result.url;
  if (result?.b64_json) return `data:image/png;base64,${result.b64_json}`;
  return '';
}

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

function snippetFor(payload: Record<string, unknown>, lang: Lang, apiKey: string) {
  const apiBase = 'https://openpaths.io/v1';
  const bearer = apiKey.trim() || 'op-...';
  const body = JSON.stringify(payload, null, 2);
  if (lang === 'python') {
    return `import json
from openai import OpenAI

client = OpenAI(
    api_key="${bearer}",
    base_url="${apiBase}",
)

result = client.post(
    "/images/generations",
    body=json.loads(r'''${body}'''),
    cast_to=dict,
)

image = result["data"][0]
print(image.get("url") or image["b64_json"][:32])`;
  }
  if (lang === 'javascript') {
    return `import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "${bearer}",
  baseURL: "${apiBase}",
});

const result = await client.post("/images/generations", {
  body: ${body},
});

console.log(result.data[0].url ?? result.data[0].b64_json.slice(0, 32));`;
  }
  return `curl "${apiBase}/images/generations" \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${bearer}" \\
  -d @- <<'JSON'
${body}
JSON`;
}

const inputCls = 'w-full rounded border border-white/10 bg-black px-3 py-2 text-xs font-mono text-white placeholder:text-white/25 focus:border-white/30 focus:outline-none';

export function ImageSpacePanel({
  modelId,
  modelName,
  imageToImage,
  demo,
}: {
  modelId: string;
  modelName: string;
  imageToImage: boolean;
  demo?: ImageDemo;
}) {
  const isBFLFlux2Pro = modelId === 'flux-2-pro-preview';
  const requiresInput = imageToImage && !isBFLFlux2Pro;
  const catalogExample = OPENPATHS_IMAGE_MODELS.find(item => item.openpathsId === modelId);
  const initialPayload = useMemo<Record<string, unknown>>(() => demo?.payload || {
    model: modelId,
    prompt: DEFAULT_PROMPT,
    ...(isBFLFlux2Pro
      ? { size: '1024x1024', n: 1, output_format: 'webp', safety_tolerance: 5, disable_pup: false }
      : imageToImage ? { image_url: DEFAULT_INPUT } : { size: '1024x1024', n: 1 }),
  }, [demo, imageToImage, isBFLFlux2Pro, modelId]);
  const starterPrompt = String(demo?.prompt || initialPayload.prompt || '');
  const starterInput = String(demo?.imageUrl || initialPayload.image_url || (Array.isArray(initialPayload.reference_image_urls) ? initialPayload.reference_image_urls[0] : '') || (requiresInput ? DEFAULT_INPUT : ''));
  const starterOutput = demo?.outputUrl || catalogExample?.image || DEFAULT_OUTPUT;

  const [apiKey, setApiKey] = useState(storedAPIKey);
  const [prompt, setPrompt] = useState(starterPrompt);
  const [inputUrl, setInputUrl] = useState(starterInput);
  const [size, setSize] = useState(String(initialPayload.size || '1024x1024'));
  const [outputFormat, setOutputFormat] = useState(String(initialPayload.output_format || 'webp'));
  const [promptUpsampling, setPromptUpsampling] = useState(initialPayload.disable_pup !== true);
  const [outputUrl, setOutputUrl] = useState(starterOutput);
  const [generated, setGenerated] = useState(false);
  const [lang, setLang] = useState<Lang>('python');
  const [copied, setCopied] = useState(false);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    setPrompt(starterPrompt);
    setInputUrl(starterInput);
    setSize(String(initialPayload.size || '1024x1024'));
    setOutputFormat(String(initialPayload.output_format || 'webp'));
    setPromptUpsampling(initialPayload.disable_pup !== true);
    setOutputUrl(starterOutput);
    setGenerated(false);
    setError('');
  }, [initialPayload, starterInput, starterOutput, starterPrompt]);

  useEffect(() => {
    if (apiKey.trim()) localStorage.setItem('op_api_key', apiKey.trim());
  }, [apiKey]);

  const payload = useMemo(() => {
    const next = { ...initialPayload, model: modelId };
    if ('prompt' in next || !demo) next.prompt = prompt;
    if (inputUrl) {
      if (Array.isArray(next.reference_image_urls)) next.reference_image_urls = [inputUrl];
      else if ('image_url' in next || imageToImage) next.image_url = inputUrl;
    }
    if (isBFLFlux2Pro) {
      next.size = size;
      next.output_format = outputFormat;
      next.safety_tolerance = 5;
      next.disable_pup = !promptUpsampling;
      if (inputUrl) next.reference_image_urls = [inputUrl];
      else delete next.reference_image_urls;
    }
    return next;
  }, [demo, imageToImage, initialPayload, inputUrl, isBFLFlux2Pro, modelId, outputFormat, prompt, promptUpsampling, size]);
  const snippet = useMemo(() => snippetFor(payload, lang, apiKey), [apiKey, lang, payload]);

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
      const resp = await fetch('/v1/files/upload', { method: 'POST', headers: { Authorization: `Bearer ${apiKey.trim()}` }, body: form });
      const data = await responseJSON(resp);
      if (!resp.ok || !data?.url) throw new Error(apiError(data, 'Image upload failed'));
      setInputUrl(normalizeUploadedAssetUrl(data.url));
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
    if (('prompt' in payload) && !prompt.trim()) {
      setError('Enter a prompt before generating.');
      return;
    }
    if (requiresInput && !inputUrl.trim()) {
      setError('Add an input image before generating with this model.');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const resp = await fetch('/v1/images/generations', {
        method: 'POST',
        headers: { Authorization: `Bearer ${apiKey.trim()}`, 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      const data = await responseJSON(resp);
      if (!resp.ok) throw new Error(apiError(data, `Image generation failed (${resp.status})`));
      const src = responseSource(data?.data?.[0]);
      if (!src) throw new Error('The API completed without returning an image.');
      setOutputUrl(src);
      setGenerated(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Image generation failed');
    } finally {
      setLoading(false);
    }
  };

  const copy = async () => {
    await navigator.clipboard.writeText(snippet);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  };

  return (
    <div className="grid lg:grid-cols-[minmax(0,1.05fr)_minmax(0,0.95fr)]" data-testid="mp-image-panel">
      <div className="border-b border-white/10 lg:border-b-0 lg:border-r">
        <div className={`grid gap-px bg-white/10 ${inputUrl ? 'grid-cols-2' : 'grid-cols-1'}`}>
          {inputUrl && (
            <div className="bg-black">
              <div className="border-b border-white/10 px-4 py-2 text-[10px] font-mono uppercase tracking-[0.16em] text-white/35">Input</div>
              <img src={inputUrl} alt={`${modelName} input`} className="aspect-square w-full object-contain p-3" data-testid="mp-image-input-preview" />
            </div>
          )}
          <div className="relative bg-black">
            <div className="border-b border-white/10 px-4 py-2 text-[10px] font-mono uppercase tracking-[0.16em] text-white/35">{generated ? 'Generated output' : demo ? 'Output' : 'Starter output preview'}</div>
            <img src={outputUrl} alt={`${modelName} output`} className="aspect-square w-full object-contain p-3" data-testid="mp-image-output" />
          </div>
        </div>
        <div className="border-t border-white/10 p-5">
          <h2 className="text-xl font-bold tracking-tight">{demo?.title || `${modelName} starter example`}</h2>
          <p className="mt-2 text-sm leading-relaxed text-white/55">{demo?.description || 'A preloaded request and output keep this model page useful on first paint. Generate to replace it with your result.'}</p>
        </div>
      </div>

      <div className="p-5">
        <h2 className="text-xl font-bold tracking-tight">Create an image in place</h2>
        <p className="mt-1 text-sm text-white/45">Run {modelName} here; the API example below stays synchronized.</p>

        <label className="mt-4 block">
          <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">OpenPaths API key</span>
          <input type="password" value={apiKey} onChange={e => setApiKey(e.target.value)} placeholder="op-..." autoComplete="off" className={inputCls} data-testid="mp-image-api-key" />
        </label>

        {('prompt' in payload) && (
          <label className="mt-3 block">
            <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Prompt</span>
            <textarea value={prompt} onChange={e => setPrompt(e.target.value)} rows={6} className={`${inputCls} resize-y text-sm leading-relaxed`} data-testid="mp-image-prompt" />
          </label>
        )}

        {imageToImage && (
          <label className="mt-3 block">
            <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Input image URL{isBFLFlux2Pro ? ' (optional)' : ''}</span>
            <span className="flex gap-2">
              <input value={inputUrl} onChange={e => setInputUrl(e.target.value)} placeholder="https://example.com/input.webp" className={inputCls} data-testid="mp-image-input-url" />
              <span className="inline-flex shrink-0 cursor-pointer items-center justify-center rounded border border-white/12 px-3 text-white/55 hover:border-white/30 hover:text-white">
                {uploading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Upload className="h-4 w-4" />}
                <span className="sr-only">Upload input image</span>
                <input type="file" accept="image/*" className="sr-only" onChange={e => uploadImage(e.target.files?.[0])} />
              </span>
            </span>
          </label>
        )}

        {isBFLFlux2Pro && (
          <div className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
            <label className="block">
              <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Size</span>
              <select value={size} onChange={e => setSize(e.target.value)} className={inputCls} data-testid="mp-image-size">
                {BFL_IMAGE_SIZES.map(value => <option key={value} value={value} className="bg-black">{value}</option>)}
              </select>
            </label>
            <label className="block">
              <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Output</span>
              <select value={outputFormat} onChange={e => setOutputFormat(e.target.value)} className={inputCls} data-testid="mp-image-output-format">
                {['webp', 'png', 'jpeg'].map(value => <option key={value} value={value} className="bg-black">{value}</option>)}
              </select>
            </label>
            <label className="block">
              <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Prompt upsampling</span>
              <button type="button" onClick={() => setPromptUpsampling(value => !value)} className={`w-full rounded border px-3 py-2 font-mono text-xs ${promptUpsampling ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-200' : 'border-white/10 bg-black text-white/50'}`} data-testid="mp-image-prompt-upsampling">
                {promptUpsampling ? 'on' : 'off'}
              </button>
            </label>
            <div>
              <span className="mb-1.5 block text-[10px] font-mono uppercase tracking-wider text-white/40">Safety</span>
              <div className="rounded border border-white/10 bg-white/[0.03] px-3 py-2 font-mono text-xs text-white/55" data-testid="mp-image-safety-tolerance"><span className="text-white">5</span> · fixed</div>
            </div>
          </div>
        )}

        <button type="button" onClick={generate} disabled={loading || uploading} className="mt-4 inline-flex w-full items-center justify-center gap-2 rounded border border-white bg-white px-4 py-2.5 font-mono text-sm font-bold text-black hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-60" data-testid="mp-image-generate">
          {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Wand2 className="h-4 w-4" />}
          {loading ? 'Generating image…' : 'Generate image here'}
        </button>
        {error && <p className="mt-3 rounded border border-red-400/20 bg-red-400/10 px-3 py-2 text-xs font-mono text-red-200" role="alert">{error}</p>}
      </div>

      <div className="border-t border-white/10 p-5 lg:col-span-2">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
          <div>
            <h3 className="font-mono text-xs font-bold uppercase tracking-[0.14em] text-white/65">Image API example</h3>
            <p className="mt-1 text-xs text-white/35">Prompt and input image changes update this executable request immediately.</p>
          </div>
          <button type="button" onClick={copy} className="inline-flex items-center gap-2 rounded border border-white/12 px-3 py-2 font-mono text-xs text-white/60 hover:border-white/30 hover:text-white">
            {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
            {copied ? 'Copied' : 'Copy request'}
          </button>
        </div>
        <div className="mb-3 flex flex-wrap gap-2">
          {(['python', 'javascript', 'curl'] as const).map(item => (
            <button key={item} type="button" onClick={() => setLang(item)} className={`rounded px-3 py-1.5 font-mono text-xs ${lang === item ? 'bg-white text-black' : 'border border-white/10 text-white/45 hover:text-white'}`}>
              {item === 'javascript' ? 'JavaScript' : item === 'python' ? 'Python' : 'cURL'}
            </button>
          ))}
        </div>
        <CodeBlock code={snippet} language={lang === 'curl' ? 'bash' : lang} containerClassName="overflow-hidden rounded-lg border border-white/10 bg-black/60" preClassName="max-h-[420px] text-xs" />
        <div className="mt-3 flex items-center gap-2 text-[11px] text-white/32"><ImageIcon className="h-3.5 w-3.5" /> Output replaces the preview without leaving this model page.</div>
      </div>
    </div>
  );
}
