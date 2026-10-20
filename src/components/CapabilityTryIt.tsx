import React, { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, Check, Copy, Loader2, Play } from 'lucide-react';
import { CodeBlock } from './CodeBlock';
import { SubscribePrompt } from './SubscribePrompt';
import { AUTH_EVENT, getApiKey } from '../lib/api';
import { ensureSpaceApiKey, paywallBeforeRun, paywallFromResponse, type Paywall } from '../lib/paywall';
import { isFreeModel, priceSummary } from '../lib/modelPricing';
import type { Model } from '../data/models';
import { DEFAULT_INPUT, buildSnippets, type Capability, type Lang } from '../data/capabilities';

type Result =
  | { kind: 'text'; text: string; meta?: string }
  | { kind: 'embedding'; dims: number; head: number[]; meta?: string }
  | { kind: 'audio'; src: string }
  | { kind: 'image'; src: string };

const inputCls = 'w-full rounded border border-white/20 bg-black px-3 py-2 text-sm font-mono text-white placeholder:text-white/45 focus:border-white/50 focus:outline-none';

async function readJSON(resp: Response) {
  const text = await resp.text();
  if (!text) return {};
  try {
    return JSON.parse(text);
  } catch {
    throw new Error(resp.ok ? 'The API returned invalid JSON.' : text.slice(0, 240));
  }
}

function errorMessage(data: any, fallback: string) {
  if (typeof data?.error === 'string') return data.error;
  return data?.error?.message || fallback;
}

export function CapabilityTryIt({ cap, model, models, onModelChange }: { cap: Capability; model: Model | undefined; models?: Model[]; onModelChange?: (id: string) => void }) {
  const modelId = model?.id || cap.defaultModel;
  const [input, setInput] = useState(DEFAULT_INPUT[cap.kind]);
  const [file, setFile] = useState<File | null>(null);
  const [lang, setLang] = useState<Lang>('python');
  const [copied, setCopied] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<Result | null>(null);
  const [paywall, setPaywall] = useState<Paywall | null>(null);
  const [signedIn, setSignedIn] = useState(false);

  useEffect(() => {
    const sync = () => {
      setSignedIn(!!getApiKey());
      setPaywall(current => (current?.kind === 'login' ? null : current));
    };
    sync();
    window.addEventListener(AUTH_EVENT, sync);
    return () => window.removeEventListener(AUTH_EVENT, sync);
  }, []);

  useEffect(() => {
    setResult(null);
    setError('');
    setPaywall(null);
  }, [modelId]);

  const snippets = useMemo(() => buildSnippets(cap, modelId, input), [cap, modelId, input]);
  const free = !!model && isFreeModel(model);
  const runnable = cap.kind !== 'studio';

  async function run() {
    setError('');
    setPaywall(null);
    setResult(null);
    if (cap.kind === 'stt' && !file) {
      setError('Choose an audio file to transcribe.');
      return;
    }
    if (cap.kind !== 'stt' && !input.trim()) {
      setError('Enter some input first.');
      return;
    }
    const key = getApiKey() || (await ensureSpaceApiKey());
    if (!key) {
      setPaywall(paywallBeforeRun() || { kind: 'login', message: 'Sign in to run this model.', subscribeUrl: '/pricing' });
      return;
    }
    setLoading(true);
    try {
      const headers: Record<string, string> = { Authorization: `Bearer ${key}` };
      let resp: Response;
      if (cap.kind === 'stt') {
        const form = new FormData();
        form.append('model', modelId);
        form.append('file', file as File);
        resp = await fetch(`/v1${cap.endpoint}`, { method: 'POST', headers, body: form });
      } else {
        headers['Content-Type'] = 'application/json';
        resp = await fetch(`/v1${cap.endpoint}`, { method: 'POST', headers, body: JSON.stringify(requestBody(cap, modelId, input)) });
      }
      const data: any = await readJSON(resp);
      const wall = paywallFromResponse(resp, data);
      if (wall) {
        setPaywall(wall);
        return;
      }
      if (!resp.ok) throw new Error(errorMessage(data, `Request failed (${resp.status})`));
      setResult(parseResult(cap, data));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Request failed.');
    } finally {
      setLoading(false);
    }
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(snippets[lang]);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1400);
    } catch {}
  }

  return (
    <div className="grid gap-6 lg:grid-cols-2" data-testid="capability-tryit">
      <div className="rounded-xl border border-white/20 bg-white/[0.05] p-5">
        <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
          <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">{runnable ? 'Try it' : 'Request'}</h2>
          {model && <span className={`rounded border px-2 py-0.5 font-mono text-[11px] ${free ? 'border-emerald-300/40 text-emerald-200' : 'border-white/20 text-white/60'}`}>{priceSummary(model)}</span>}
        </div>

        {models && models.length > 1 && onModelChange && (
          <label className="mb-3 block">
            <span className="mb-1 block font-mono text-[11px] uppercase tracking-[0.16em] text-white/50">Model</span>
            <select value={modelId} onChange={e => onModelChange(e.target.value)} className={inputCls} data-testid="tryit-model">
              {models.map(m => <option key={m.id} value={m.id}>{m.name} ({m.id})</option>)}
            </select>
          </label>
        )}

        {runnable && cap.kind !== 'stt' && (
          <label className="block">
            <span className="mb-1 block font-mono text-[11px] uppercase tracking-[0.16em] text-white/50">{cap.kind === 'embeddings' ? 'Text' : cap.kind === 'tts' ? 'Text to speak' : 'Prompt'}</span>
            <textarea value={input} onChange={e => setInput(e.target.value)} rows={4} className={inputCls} data-testid="tryit-input" />
          </label>
        )}
        {cap.kind === 'stt' && (
          <label className="block">
            <span className="mb-1 block font-mono text-[11px] uppercase tracking-[0.16em] text-white/50">Audio file</span>
            <input type="file" accept="audio/*,video/webm" onChange={e => setFile(e.target.files?.[0] || null)} className={inputCls} data-testid="tryit-file" />
          </label>
        )}

        {runnable ? (
          <button type="button" onClick={run} disabled={loading} className="mt-4 inline-flex items-center gap-2 rounded border border-white bg-white px-4 py-2 font-mono text-sm font-bold text-black hover:bg-white/90 disabled:opacity-60" data-testid="tryit-run">
            {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
            {signedIn ? 'Run' : free ? 'Sign up free and run' : 'Sign up and run'}
          </button>
        ) : (
          cap.studioPath && (
            <Link to={cap.studioPath} className="inline-flex items-center gap-2 rounded border border-white bg-white px-4 py-2 font-mono text-sm font-bold text-black hover:bg-white/90">
              {cap.studioLabel || 'Open studio'} <ArrowRight className="h-4 w-4" />
            </Link>
          )
        )}
        {runnable && cap.studioPath && (
          <Link to={cap.studioPath} className="ml-3 mt-4 inline-flex items-center gap-1 font-mono text-xs text-white/55 hover:text-white">
            {cap.studioLabel} <ArrowRight className="h-3.5 w-3.5" />
          </Link>
        )}

        {paywall && <SubscribePrompt paywall={paywall} modelName={model?.name || modelId} priceLabel={model ? priceSummary(model) : undefined} onDismiss={() => setPaywall(null)} />}
        {error && <p className="mt-3 font-mono text-xs text-red-300" role="alert" data-testid="tryit-error">{error}</p>}
        {result && <ResultView result={result} />}
        {!signedIn && runnable && !paywall && (
          <p className="mt-3 font-mono text-[11px] leading-relaxed text-white/45">
            Calls run against your own OpenPaths key. {free ? 'This model bills $0 per token, so an empty balance works.' : 'Paid models need prepaid credits.'}
          </p>
        )}
      </div>

      <div className="min-w-0">
        <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
          <h2 className="font-mono text-sm font-bold uppercase tracking-wider text-white/70">API code</h2>
          <div className="flex items-center gap-2">
            {(['python', 'javascript', 'curl'] as const).map(l => (
              <button key={l} type="button" onClick={() => setLang(l)} className={`rounded border px-2.5 py-1.5 text-[10px] font-mono uppercase tracking-[0.14em] ${lang === l ? 'border-white/30 bg-white/10 text-white' : 'border-white/20 text-white/50 hover:text-white'}`}>
                {l === 'javascript' ? 'JS' : l}
              </button>
            ))}
            <button type="button" onClick={copy} className="inline-flex items-center gap-1.5 rounded border border-white/20 px-2.5 py-1.5 text-[10px] font-mono uppercase tracking-[0.14em] text-white/45 hover:text-white">
              {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />} {copied ? 'Copied' : 'Copy'}
            </button>
          </div>
        </div>
        <CodeBlock language={lang === 'curl' ? 'bash' : lang} label={lang === 'curl' ? 'cURL' : lang === 'javascript' ? 'JavaScript' : 'Python'} code={snippets[lang]} containerClassName="overflow-hidden rounded-lg border border-white/20 bg-black/40" headerClassName="bg-white/[0.06]" preClassName="text-xs" />
      </div>
    </div>
  );
}

export function requestBody(cap: Capability, model: string, input: string): Record<string, unknown> {
  switch (cap.kind) {
    case 'chat':
      return { model, messages: [{ role: 'user', content: input }], max_tokens: 512 };
    case 'messages':
      return { model, max_tokens: 512, messages: [{ role: 'user', content: input }] };
    case 'embeddings':
      return { model, input };
    case 'tts':
      return { model, input };
    case 'image':
      return { model, prompt: input, size: '1024x1024', n: 1 };
    default:
      return { model };
  }
}

export function parseResult(cap: Capability, data: any): Result {
  switch (cap.kind) {
    case 'chat': {
      const msg = data?.choices?.[0]?.message;
      const text = msg?.content || msg?.reasoning_content || msg?.reasoning || '';
      const usage = data?.usage;
      return { kind: 'text', text: String(text) || '(empty response)', meta: usage ? `${usage.prompt_tokens ?? '?'} in / ${usage.completion_tokens ?? '?'} out tokens` : undefined };
    }
    case 'messages': {
      const text = Array.isArray(data?.content) ? data.content.map((p: any) => p?.text || '').join('') : '';
      return { kind: 'text', text: text || '(empty response)' };
    }
    case 'embeddings': {
      const vec: number[] = data?.data?.[0]?.embedding || [];
      return { kind: 'embedding', dims: vec.length, head: vec.slice(0, 8), meta: data?.usage?.total_tokens ? `${data.usage.total_tokens} tokens` : undefined };
    }
    case 'tts': {
      const format = data?.format || 'mp3';
      const src = data?.audio_url || (data?.audio ? `data:audio/${format};base64,${data.audio}` : '');
      if (!src) throw new Error('No audio was returned.');
      return { kind: 'audio', src };
    }
    case 'stt':
      return { kind: 'text', text: data?.text || '(no speech detected)' };
    case 'image': {
      const item = data?.data?.[0];
      const src = item?.url || (item?.b64_json ? `data:image/png;base64,${item.b64_json}` : '');
      if (!src) throw new Error('No image was returned.');
      return { kind: 'image', src };
    }
    default:
      throw new Error('Unsupported capability.');
  }
}

function ResultView({ result }: { result: Result }) {
  return (
    <div className="mt-4 rounded border border-white/20 bg-black/50 p-3" data-testid="tryit-result">
      {result.kind === 'text' && (
        <>
          <p className="whitespace-pre-wrap text-sm leading-relaxed text-white/85">{result.text}</p>
          {result.meta && <p className="mt-2 font-mono text-[11px] text-white/45">{result.meta}</p>}
        </>
      )}
      {result.kind === 'embedding' && (
        <>
          <p className="font-mono text-sm text-white/85">{result.dims} dimensions</p>
          <p className="mt-1 break-all font-mono text-[11px] text-white/55">[{result.head.map(v => v.toFixed(4)).join(', ')}, ...]</p>
          {result.meta && <p className="mt-2 font-mono text-[11px] text-white/45">{result.meta}</p>}
        </>
      )}
      {result.kind === 'audio' && <audio controls src={result.src} className="w-full" />}
      {result.kind === 'image' && <img src={result.src} alt="Generated result" className="w-full rounded" />}
    </div>
  );
}
