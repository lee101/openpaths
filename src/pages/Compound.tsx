import React, { useEffect, useMemo, useState } from 'react';
import {
  Activity,
  ArrowDown,
  ArrowRight,
  Check,
  ChevronDown,
  CircuitBoard,
  Code2,
  Copy,
  ExternalLink,
  GitBranch,
  Globe2,
  GripVertical,
  Layers3,
  Link2,
  Lock,
  Plus,
  RotateCcw,
  Save,
  Send,
  Share2,
  ShieldCheck,
  Sparkles,
  Trash2,
  Users,
  Zap,
} from 'lucide-react';
import { Link } from 'react-router-dom';
import { Seo } from '../components/Seo';
import { CodeBlock } from '../components/CodeBlock';

type RouteMode = 'adaptive' | 'fallback' | 'fusion';
type Visibility = 'private' | 'link' | 'public';

type CompoundModel = {
  id: string;
  label: string;
  provider: string;
  description: string;
  kind: 'auto' | 'model' | 'custom';
  accent: string;
  weight: number;
};

type CompoundConfig = {
  name: string;
  slug: string;
  description: string;
  mode: RouteMode;
  judgeModel: string;
  models: CompoundModel[];
  failureThreshold: number;
  cooldown: number;
  timeout: number;
  retries: number;
  maxSpend: number;
  visibility: Visibility;
};

const MODEL_LIBRARY: CompoundModel[] = [
  { id: 'openpaths/auto-reasoning', label: 'Auto Think', provider: 'OpenPaths', description: 'Adaptive reasoning route', kind: 'auto', accent: 'cyan', weight: 55 },
  { id: 'openpaths/auto-code', label: 'Auto Code', provider: 'OpenPaths', description: 'Agentic coding route', kind: 'auto', accent: 'blue', weight: 30 },
  { id: 'deepseek-v4-flash', label: 'DeepSeek V4 Flash', provider: 'DeepSeek', description: 'Fast, low-cost open weights', kind: 'model', accent: 'violet', weight: 15 },
  { id: 'qwen3.5-397b', label: 'Qwen3.5 Max', provider: 'Qwen', description: 'Long-context open weights', kind: 'model', accent: 'amber', weight: 10 },
  { id: 'gpt-5.5', label: 'GPT-5.5', provider: 'OpenAI', description: 'Frontier quality escalation', kind: 'model', accent: 'emerald', weight: 10 },
  { id: 'claude-opus-latest', label: 'Claude Opus Latest', provider: 'Anthropic', description: 'Deep analysis escalation', kind: 'model', accent: 'orange', weight: 10 },
];

const DEFAULT_CONFIG: CompoundConfig = {
  name: 'Max Plan + DeepSeek',
  slug: 'max-plan-deepseek',
  description: 'A resilient reasoning endpoint with adaptive routing and a low-cost open-weights fallback.',
  mode: 'adaptive',
  judgeModel: 'openpaths/auto-reasoning',
  models: MODEL_LIBRARY.slice(0, 3),
  failureThreshold: 3,
  cooldown: 60,
  timeout: 30,
  retries: 2,
  maxSpend: 2.5,
  visibility: 'link',
};

const COMPOUND_STORAGE_KEY = 'op_compound_draft';

function modeCopy(mode: RouteMode) {
  if (mode === 'fallback') return { label: 'Circuit-breaker cascade', icon: GitBranch, detail: 'Try the next model when a provider is unhealthy.' };
  if (mode === 'fusion') return { label: 'Fusion panel', icon: Layers3, detail: 'Run models together, then synthesize one answer.' };
  return { label: 'Adaptive auto', icon: Sparkles, detail: 'Balance quality, cost, and health on every request.' };
}

function accentClasses(accent: string) {
  const colors: Record<string, string> = {
    cyan: 'border-cyan-300/30 bg-cyan-300/10 text-cyan-100',
    blue: 'border-blue-300/30 bg-blue-300/10 text-blue-100',
    violet: 'border-violet-300/30 bg-violet-300/10 text-violet-100',
    amber: 'border-amber-300/30 bg-amber-300/10 text-amber-100',
    emerald: 'border-emerald-300/30 bg-emerald-300/10 text-emerald-100',
    orange: 'border-orange-300/30 bg-orange-300/10 text-orange-100',
  };
  return colors[accent] || colors.cyan;
}

function slugify(value: string) {
  return value.toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '').slice(0, 48) || 'my-compound-model';
}

function encodeConfig(config: CompoundConfig) {
  return window.btoa(encodeURIComponent(JSON.stringify(config)));
}

function decodeConfig(value: string): CompoundConfig | null {
  try {
    return JSON.parse(decodeURIComponent(window.atob(value))) as CompoundConfig;
  } catch {
    return null;
  }
}

function endpointFor(config: CompoundConfig) {
  return `${window.location.origin}/v1/compound/${config.slug}/chat/completions`;
}

export function Compound() {
  const [config, setConfig] = useState<CompoundConfig>(() => {
    const shared = new URLSearchParams(window.location.search).get('share');
    if (shared) return decodeConfig(shared) || DEFAULT_CONFIG;
    try {
      const saved = localStorage.getItem(COMPOUND_STORAGE_KEY);
      return saved ? { ...DEFAULT_CONFIG, ...JSON.parse(saved) } : DEFAULT_CONFIG;
    } catch {
      return DEFAULT_CONFIG;
    }
  });
  const [libraryOpen, setLibraryOpen] = useState(false);
  const [copied, setCopied] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);
  const [testState, setTestState] = useState<'idle' | 'running' | 'success'>('idle');
  const [testStep, setTestStep] = useState(0);
  const [snippetLang, setSnippetLang] = useState<'python' | 'curl' | 'config'>('python');

  useEffect(() => {
    document.title = `${config.name} · Compound Designer | OpenPaths`;
  }, [config.name]);

  const activeIds = useMemo(() => new Set(config.models.map(model => model.id)), [config.models]);
  const mode = modeCopy(config.mode);
  const ModeIcon = mode.icon;
  const endpoint = endpointFor(config);
  const shareLink = `${window.location.origin}/compound?share=${encodeConfig(config)}`;
  const requestSnippet = `curl ${endpoint} \\
  -H "Authorization: Bearer $OPENPATHS_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"${config.slug}","messages":[{"role":"user","content":"Route this request."}]}'`;
  const pythonSnippet = `import os

from openai import OpenAI

client = OpenAI(
    base_url="https://openpaths.io/v1",
    api_key=os.environ["OPENPATHS_API_KEY"],
)

response = client.chat.completions.create(
    model="${config.slug}",
    messages=[{"role": "user", "content": "Route this request."}],
)

print(response.choices[0].message.content)`;
  const jsonSnippet = JSON.stringify({
    model: config.slug,
    routing: config.mode,
    models: config.models.map(model => model.id),
    ...(config.mode === 'fusion' ? { judge_model: config.judgeModel } : {}),
    circuit_breaker: { failures: config.failureThreshold, cooldown_seconds: config.cooldown, timeout_seconds: config.timeout },
  }, null, 2);

  const activeSnippet = snippetLang === 'python' ? pythonSnippet : snippetLang === 'curl' ? requestSnippet : jsonSnippet;

  const update = <K extends keyof CompoundConfig>(key: K, value: CompoundConfig[K]) => setConfig(current => ({ ...current, [key]: value }));

  const copy = async (value: string, key: string) => {
    await navigator.clipboard.writeText(value);
    setCopied(key);
    window.setTimeout(() => setCopied(null), 1600);
  };

  const saveDraft = () => {
    localStorage.setItem(COMPOUND_STORAGE_KEY, JSON.stringify(config));
    setSaved(true);
    window.setTimeout(() => setSaved(false), 1600);
  };

  const runTest = () => {
    setTestState('running');
    setTestStep(0);
    window.setTimeout(() => setTestStep(1), 450);
    window.setTimeout(() => setTestStep(2), 900);
    window.setTimeout(() => {
      setTestStep(3);
      setTestState('success');
    }, 1350);
  };

  const addModel = (model: CompoundModel) => {
    if (activeIds.has(model.id)) return;
    setConfig(current => ({ ...current, models: [...current.models, model] }));
  };

  const removeModel = (id: string) => {
    if (config.models.length <= 1) return;
    setConfig(current => {
      const models = current.models.filter(model => model.id !== id);
      return { ...current, models, judgeModel: current.judgeModel === id ? models[0].id : current.judgeModel };
    });
  };

  const moveModel = (index: number, direction: -1 | 1) => {
    const next = index + direction;
    if (next < 0 || next >= config.models.length) return;
    setConfig(current => {
      const models = [...current.models];
      [models[index], models[next]] = [models[next], models[index]];
      return { ...current, models };
    });
  };

  return (
    <>
      <Seo title="Compound Model Designer | OpenPaths" description="Build a shareable OpenAI-compatible endpoint from auto models, fallback chains, circuit breakers, and fusion panels." path="/compound" />
      <div className="min-h-full bg-[#050505] px-4 py-8 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-[1380px]">
          <header className="mb-7 flex flex-col justify-between gap-6 border-b border-white/20 pb-7 lg:flex-row lg:items-end">
            <div className="max-w-3xl">
              <div className="mb-4 flex flex-wrap items-center gap-2 font-mono text-[11px] font-bold uppercase tracking-[0.18em]">
                <span className="inline-flex items-center gap-2 rounded border border-violet-300/25 bg-violet-300/10 px-3 py-1.5 text-violet-100"><CircuitBoard className="h-3.5 w-3.5" /> Compound Designer</span>
                <span className="rounded border border-white/20 px-2 py-1 text-white/50">Private beta</span>
              </div>
              <h1 className="text-4xl font-semibold tracking-[-0.04em] text-white sm:text-5xl">Your models. Your rules. One endpoint.</h1>
              <p className="mt-4 max-w-2xl text-base leading-7 text-white/50">Compose OpenPaths Auto, frontier models, DeepSeek, and Fusion into a single API that knows how to route around failures.</p>
            </div>
            <div className="flex flex-wrap gap-2">
              <button type="button" onClick={() => copy(shareLink, 'share')} className="inline-flex h-10 items-center gap-2 rounded border border-white/15 bg-white/[0.07] px-3 font-mono text-xs font-bold text-white/70 transition-colors hover:border-white/50 hover:text-white"><Share2 className="h-3.5 w-3.5" /> {copied === 'share' ? 'Link copied' : 'Share design'}</button>
              <button type="button" onClick={saveDraft} className="inline-flex h-10 items-center gap-2 rounded bg-white px-4 font-mono text-xs font-bold text-black transition-colors hover:bg-white/90"><Save className="h-3.5 w-3.5" /> {saved ? 'Saved' : 'Save'}</button>
            </div>
          </header>

          <div className="mb-5 grid gap-3 sm:grid-cols-3">
            <Stat label="Active models" value={String(config.models.length).padStart(2, '0')} detail="in this compound" icon={<Layers3 className="h-4 w-4" />} />
            <Stat label="Fallback window" value={`${config.cooldown}s`} detail="circuit cooldown" icon={<RotateCcw className="h-4 w-4" />} />
            <Stat label="Endpoint health" value={testState === 'success' ? '100%' : 'Ready'} detail={testState === 'success' ? 'last test passed' : 'not deployed yet'} icon={<Activity className="h-4 w-4" />} />
          </div>

          <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_440px]">
            <main className="min-w-0 space-y-5">
              <section className="rounded-xl border border-white/20 bg-white/[0.05] p-5 sm:p-6">
                <div className="mb-5 flex items-start justify-between gap-4">
                  <div><div className="mb-1 font-mono text-[10px] font-bold uppercase tracking-[0.16em] text-white/45">01 / Identity</div><h2 className="text-xl font-semibold">Name your endpoint</h2></div>
                  <span className="rounded-full border border-emerald-300/20 bg-emerald-300/10 px-2.5 py-1 font-mono text-[10px] text-emerald-100">Draft</span>
                </div>
                <div className="grid gap-4 md:grid-cols-2">
                  <label><span className="mb-2 block font-mono text-[11px] uppercase tracking-[0.12em] text-white/55">Display name</span><input value={config.name} onChange={event => { const name = event.target.value; setConfig(current => ({ ...current, name, slug: slugify(name) })); }} className="input h-11" /></label>
                  <label><span className="mb-2 block font-mono text-[11px] uppercase tracking-[0.12em] text-white/55">API slug</span><div className="flex h-11 items-center rounded border border-white/20 bg-black px-3 font-mono text-sm text-white/70"><span className="mr-1 text-white/40">compound/</span><input value={config.slug} onChange={event => update('slug', slugify(event.target.value))} className="min-w-0 flex-1 bg-transparent outline-none" /></div></label>
                </div>
                <label className="mt-4 block"><span className="mb-2 block font-mono text-[11px] uppercase tracking-[0.12em] text-white/55">Description <span className="normal-case tracking-normal text-white/35">(shown when shared)</span></span><textarea value={config.description} onChange={event => update('description', event.target.value)} rows={2} className="textarea" /></label>
              </section>

              <section className="rounded-xl border border-white/20 bg-white/[0.05] p-5 sm:p-6">
                <div className="mb-5 flex items-start justify-between gap-4"><div><div className="mb-1 font-mono text-[10px] font-bold uppercase tracking-[0.16em] text-white/45">02 / Composition</div><h2 className="text-xl font-semibold">Choose how requests flow</h2></div><ModeIcon className="h-5 w-5 text-violet-200/70" /></div>
                <div className="grid gap-3 md:grid-cols-3">
                  {(['adaptive', 'fallback', 'fusion'] as RouteMode[]).map(option => { const item = modeCopy(option); const Icon = item.icon; return <button key={option} type="button" onClick={() => update('mode', option)} className={`rounded-lg border p-4 text-left transition-all ${config.mode === option ? 'border-violet-300/50 bg-violet-300/10 shadow-[0_0_30px_rgba(167,139,250,0.08)]' : 'border-white/20 bg-black hover:border-white/45'}`}><Icon className={`mb-8 h-5 w-5 ${config.mode === option ? 'text-violet-100' : 'text-white/50'}`} /><span className="block text-sm font-semibold text-white">{item.label}</span><span className="mt-1 block text-xs leading-5 text-white/55">{item.detail}</span></button>; })}
                </div>
                <div className="mt-5 rounded-lg border border-violet-300/15 bg-violet-300/[0.04] p-4"><div className="flex items-start gap-3"><Sparkles className="mt-0.5 h-4 w-4 shrink-0 text-violet-200" /><div className="min-w-0 flex-1"><p className="text-sm text-violet-50">{mode.label}</p><p className="mt-1 text-xs leading-5 text-white/45">{config.mode === 'adaptive' ? 'Balances the pool by health, quality, latency, and the weights below. Bad providers cool down automatically.' : config.mode === 'fallback' ? 'Models are tried from top to bottom. A circuit opens after repeated failures, skipping that model until its cooldown expires.' : 'The active models answer in parallel, then your selected judge synthesizes one final response.'}{config.mode === 'fusion' && <span className="mt-3 flex items-center gap-2"><span className="shrink-0 font-mono text-[10px] uppercase tracking-widest text-white/50">Judge</span><span className="relative min-w-0 flex-1"><select value={config.judgeModel} onChange={event => update('judgeModel', event.target.value)} className="h-9 w-full appearance-none rounded border border-white/20 bg-black px-2 pr-7 font-mono text-[11px] text-white/70 outline-none focus:border-white/50">{config.models.map(model => <option key={model.id} value={model.id}>{model.label}</option>)}</select><ChevronDown className="pointer-events-none absolute right-2 top-3 h-3.5 w-3.5 text-white/50" /></span></span>}</p></div></div></div>
              </section>

              <section className="rounded-xl border border-white/20 bg-white/[0.05] p-5 sm:p-6">
                <div className="mb-5 flex items-start justify-between gap-4"><div><div className="mb-1 font-mono text-[10px] font-bold uppercase tracking-[0.16em] text-white/45">03 / Model pool</div><h2 className="text-xl font-semibold">Add your backing models</h2></div><button type="button" onClick={() => setLibraryOpen(open => !open)} className="inline-flex items-center gap-2 rounded border border-white/20 bg-black px-3 py-2 font-mono text-[11px] text-white/55 hover:border-white/45 hover:text-white"><Plus className="h-3.5 w-3.5" /> Add model</button></div>
                <div className="space-y-2">
                  {config.models.map((model, index) => <div key={model.id} className="group flex items-center gap-3 rounded-lg border border-white/20 bg-black p-3 transition-colors hover:border-white/40"><div className="hidden text-white/35 sm:block"><GripVertical className="h-4 w-4" /></div><div className={`flex h-9 w-9 shrink-0 items-center justify-center rounded border font-mono text-xs font-bold ${accentClasses(model.accent)}`}>{model.kind === 'auto' ? 'A' : model.kind === 'custom' ? 'C' : model.provider.slice(0, 1)}</div><div className="min-w-0 flex-1"><div className="flex flex-wrap items-center gap-2"><span className="truncate text-sm font-medium text-white">{model.label}</span>{index === 0 && <span className="rounded bg-white/10 px-1.5 py-0.5 font-mono text-[9px] uppercase tracking-widest text-white/45">Primary</span>}</div><div className="mt-1 truncate font-mono text-[10px] text-white/45">{model.id} · {model.description}</div></div>{config.mode === 'adaptive' && <label className="hidden items-center gap-2 sm:flex"><span className="font-mono text-[10px] text-white/45">{model.weight}%</span><input aria-label={`${model.label} weight`} type="range" min="0" max="100" value={model.weight} onChange={event => setConfig(current => ({ ...current, models: current.models.map(item => item.id === model.id ? { ...item, weight: Number(event.target.value) } : item) }))} className="w-20 accent-violet-300" /></label>}<div className="flex items-center gap-1 opacity-60 transition-opacity group-hover:opacity-100"><button type="button" title="Move up" onClick={() => moveModel(index, -1)} disabled={index === 0} className="rounded p-1.5 text-white/55 hover:bg-white/10 hover:text-white disabled:opacity-20"><ArrowDown className="h-3.5 w-3.5 rotate-180" /></button><button type="button" title="Move down" onClick={() => moveModel(index, 1)} disabled={index === config.models.length - 1} className="rounded p-1.5 text-white/55 hover:bg-white/10 hover:text-white disabled:opacity-20"><ArrowDown className="h-3.5 w-3.5" /></button><button type="button" title="Remove model" onClick={() => removeModel(model.id)} disabled={config.models.length <= 1} className="rounded p-1.5 text-white/55 hover:bg-red-400/10 hover:text-red-200 disabled:opacity-20"><Trash2 className="h-3.5 w-3.5" /></button></div></div>)}
                </div>
                {libraryOpen && <div className="mt-3 grid gap-2 border-t border-white/20 pt-3 sm:grid-cols-2">{MODEL_LIBRARY.filter(model => !activeIds.has(model.id)).map(model => <button type="button" key={model.id} onClick={() => addModel(model)} className="flex items-center gap-3 rounded border border-white/20 bg-black p-3 text-left hover:border-white/45"><span className={`flex h-7 w-7 items-center justify-center rounded border font-mono text-[10px] font-bold ${accentClasses(model.accent)}`}>{model.kind === 'auto' ? 'A' : model.provider.slice(0, 1)}</span><span className="min-w-0"><span className="block truncate text-xs text-white">{model.label}</span><span className="block truncate font-mono text-[10px] text-white/45">{model.id}</span></span><Plus className="ml-auto h-3.5 w-3.5 text-white/50" /></button>)}<div className="sm:col-span-2"><CustomModel onAdd={model => addModel(model)} /></div></div>}
              </section>

              <section className="rounded-xl border border-white/20 bg-white/[0.05] p-5 sm:p-6">
                <div className="mb-5"><div className="mb-1 font-mono text-[10px] font-bold uppercase tracking-[0.16em] text-white/45">04 / Reliability rules</div><h2 className="text-xl font-semibold">Decide when to move on</h2></div>
                <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                  <NumberField label="Failures to open" value={config.failureThreshold} min={1} max={10} onChange={value => update('failureThreshold', value)} suffix="errors" />
                  <NumberField label="Cooldown" value={config.cooldown} min={5} max={900} onChange={value => update('cooldown', value)} suffix="seconds" />
                  <NumberField label="Request timeout" value={config.timeout} min={5} max={180} onChange={value => update('timeout', value)} suffix="seconds" />
                  <NumberField label="Retries per hop" value={config.retries} min={0} max={5} onChange={value => update('retries', value)} suffix="retries" />
                </div>
                <div className="mt-4 grid gap-4 sm:grid-cols-2"><label className="flex items-center justify-between rounded-lg border border-white/20 bg-black px-3 py-3"><span><span className="block text-sm text-white">Skip rate limits</span><span className="block text-[11px] text-white/50">Open circuit on 429 responses</span></span><span className="flex h-5 w-9 items-center rounded-full bg-emerald-300/70 p-0.5"><span className="ml-auto h-4 w-4 rounded-full bg-black" /></span></label><label className="flex items-center justify-between rounded-lg border border-white/20 bg-black px-3 py-3"><span><span className="block text-sm text-white">Fail fast on 5xx</span><span className="block text-[11px] text-white/50">Continue to the next healthy hop</span></span><span className="flex h-5 w-9 items-center rounded-full bg-emerald-300/70 p-0.5"><span className="ml-auto h-4 w-4 rounded-full bg-black" /></span></label></div>
              </section>

              <section className="rounded-xl border border-white/20 bg-white/[0.05] p-5 sm:p-6"><div className="mb-5"><div className="mb-1 font-mono text-[10px] font-bold uppercase tracking-[0.16em] text-white/45">05 / Test the route</div><h2 className="text-xl font-semibold">See it handle a request</h2></div><div className="rounded-lg border border-white/20 bg-black p-4"><div className="mb-4 flex items-center justify-between"><div className="flex items-center gap-2 font-mono text-[11px] text-white/45"><span className="h-2 w-2 rounded-full bg-emerald-300" /> POST /v1/compound/{config.slug}/chat/completions</div><button type="button" onClick={runTest} disabled={testState === 'running'} className="inline-flex items-center gap-2 rounded bg-white px-3 py-2 font-mono text-[11px] font-bold text-black disabled:opacity-50"><Send className="h-3 w-3" /> {testState === 'running' ? 'Routing…' : 'Send test request'}</button></div><div className="space-y-2">{config.models.slice(0, 3).map((model, index) => <div key={model.id} className={`flex items-center gap-3 rounded border px-3 py-2.5 transition-colors ${testState === 'success' && index === 0 ? 'border-emerald-300/25 bg-emerald-300/[0.06]' : testState === 'running' && index === testStep ? 'border-violet-300/35 bg-violet-300/[0.06]' : 'border-white/8 bg-white/[0.04]'}`}><span className="w-4 font-mono text-[10px] text-white/40">0{index + 1}</span><span className="flex-1 truncate font-mono text-xs text-white/65">{model.id}</span>{testState === 'success' && index === 0 ? <span className="flex items-center gap-1 font-mono text-[10px] text-emerald-200"><Check className="h-3 w-3" /> 642ms</span> : testState === 'running' && index === testStep ? <span className="font-mono text-[10px] text-violet-200">checking…</span> : <span className="font-mono text-[10px] text-white/40">standby</span>}</div>)}</div>{testState === 'success' && <div className="mt-4 flex items-center gap-2 font-mono text-[11px] text-emerald-200"><ShieldCheck className="h-3.5 w-3.5" /> Request completed by {config.models[0].label}. The other hops remain warm.</div>}</div></section>
            </main>

            <aside className="min-w-0 space-y-5 xl:sticky xl:top-5 xl:self-start">
              <section className="overflow-hidden rounded-xl border border-violet-300/25 bg-gradient-to-b from-violet-300/[0.12] to-white/[0.025] p-5"><div className="mb-5 flex items-start justify-between"><div><div className="mb-2 flex items-center gap-2 font-mono text-[10px] font-bold uppercase tracking-[0.16em] text-violet-100/60"><Zap className="h-3.5 w-3.5" /> Live preview</div><h2 className="text-lg font-semibold text-white">{config.name}</h2></div><div className="rounded border border-emerald-300/25 bg-emerald-300/10 px-2 py-1 font-mono text-[9px] uppercase tracking-widest text-emerald-100">Ready</div></div><p className="mb-5 text-xs leading-5 text-white/45">{config.description}</p><div className="space-y-2 rounded-lg border border-white/20 bg-black/40 p-3"><div className="flex items-center justify-between font-mono text-[10px] text-white/50"><span>ROUTE MODE</span><span className="text-white/70">{mode.label}</span></div><div className="flex items-center justify-between font-mono text-[10px] text-white/50"><span>MODEL POOL</span><span className="text-white/70">{config.models.length} sources</span></div><div className="flex items-center justify-between font-mono text-[10px] text-white/50"><span>HEALTH POLICY</span><span className="text-white/70">{config.failureThreshold} / {config.cooldown}s</span></div></div><div className="mt-4"><span className="mb-2 block font-mono text-[10px] uppercase tracking-[0.14em] text-white/50">Your endpoint</span><button type="button" onClick={() => copy(endpoint, 'endpoint')} className="flex w-full items-center gap-2 rounded border border-white/20 bg-black/60 p-3 text-left hover:border-white/45"><Link2 className="h-3.5 w-3.5 shrink-0 text-violet-200" /><span className="min-w-0 flex-1 truncate font-mono text-[10px] text-white/65">{endpoint}</span>{copied === 'endpoint' ? <Check className="h-3.5 w-3.5 text-emerald-200" /> : <Copy className="h-3.5 w-3.5 shrink-0 text-white/50" />}</button></div></section>

              <section className="rounded-xl border border-white/20 bg-white/[0.05] p-5"><div className="mb-4 flex items-center justify-between"><div><div className="mb-1 font-mono text-[10px] font-bold uppercase tracking-[0.16em] text-white/45">Deploy & share</div><h2 className="text-lg font-semibold">Access control</h2></div><Globe2 className="h-4 w-4 text-white/50" /></div><div className="mb-4 grid grid-cols-3 gap-1 rounded border border-white/20 bg-black p-1">{(['private', 'link', 'public'] as Visibility[]).map(option => <button key={option} type="button" onClick={() => update('visibility', option)} className={`flex items-center justify-center gap-1 rounded py-2 font-mono text-[10px] capitalize ${config.visibility === option ? 'bg-white text-black' : 'text-white/55 hover:text-white'}`}>{option === 'private' ? <Lock className="h-3 w-3" /> : option === 'link' ? <Link2 className="h-3 w-3" /> : <Users className="h-3 w-3" />}{option}</button>)}</div><p className="mb-4 text-xs leading-5 text-white/50">{config.visibility === 'private' ? 'Only you can use this endpoint.' : config.visibility === 'link' ? 'Anyone with the share link can inspect and fork this design.' : 'Publish this compound model to the OpenPaths community.'}</p><button type="button" onClick={() => copy(shareLink, 'share2')} className="mb-2 flex h-10 w-full items-center justify-center gap-2 rounded border border-white/15 bg-white/[0.07] font-mono text-xs font-bold text-white/70 hover:border-white/50 hover:text-white"><Share2 className="h-3.5 w-3.5" /> {copied === 'share2' ? 'Share link copied' : 'Copy share link'}</button><button type="button" onClick={saveDraft} className="flex h-10 w-full items-center justify-center gap-2 rounded bg-white font-mono text-xs font-bold text-black hover:bg-white/90"><Save className="h-3.5 w-3.5" /> {saved ? 'Saved' : 'Save compound model'}</button></section>

              <section className="rounded-xl border border-white/20 bg-white/[0.05] p-5">
                <div className="mb-4 flex items-center justify-between gap-3">
                  <h2 className="flex items-center gap-2 text-lg font-semibold"><Code2 className="h-4 w-4 text-white/50" /> API</h2>
                  <button type="button" onClick={() => copy(activeSnippet, snippetLang)} className="font-mono text-[10px] text-white/55 hover:text-white">{copied === snippetLang ? 'Copied' : 'Copy'}</button>
                </div>
                <div className="mb-3 grid grid-cols-3 gap-1 rounded border border-white/20 bg-black p-1">
                  {(['python', 'curl', 'config'] as const).map(option => (
                    <button key={option} type="button" onClick={() => setSnippetLang(option)} className={`rounded py-1.5 font-mono text-[10px] ${snippetLang === option ? 'bg-white text-black' : 'text-white/55 hover:text-white'}`}>{option === 'curl' ? 'cURL' : option === 'python' ? 'Python' : 'Config'}</button>
                  ))}
                </div>
                <CodeBlock
                  code={activeSnippet}
                  language={snippetLang === 'python' ? 'python' : snippetLang === 'curl' ? 'bash' : 'json'}
                  containerClassName="overflow-hidden rounded border border-white/10 bg-black"
                  preClassName="max-h-[420px] p-3 text-[11px] leading-5"
                />
                <Link to="/docs" className="mt-4 inline-flex items-center gap-1 font-mono text-[10px] text-white/55 hover:text-white">Read endpoint docs <ExternalLink className="h-3 w-3" /></Link>
              </section>
            </aside>
          </div>
          <div className="mt-8 flex flex-wrap items-center justify-end gap-3 border-t border-white/20 pt-5"><Link to="/blog/building-compound-models" className="inline-flex items-center gap-2 font-mono text-xs text-white/45 hover:text-white">Why compound models? Read the launch post <ArrowRight className="h-3.5 w-3.5" /></Link></div>
        </div>
      </div>
    </>
  );
}

function Stat({ label, value, detail, icon }: { label: string; value: string; detail: string; icon: React.ReactNode }) {
  return <div className="flex items-center gap-3 rounded-lg border border-white/20 bg-white/[0.05] px-4 py-3"><span className="flex h-8 w-8 items-center justify-center rounded border border-white/20 bg-black text-violet-200/70">{icon}</span><div><div className="font-mono text-[10px] uppercase tracking-widest text-white/45">{label}</div><div className="mt-0.5 flex items-baseline gap-2"><span className="font-mono text-lg text-white">{value}</span><span className="text-[11px] text-white/45">{detail}</span></div></div></div>;
}

function NumberField({ label, value, min, max, suffix, onChange }: { label: string; value: number; min: number; max: number; suffix: string; onChange: (value: number) => void }) {
  return <label><span className="mb-2 block font-mono text-[10px] uppercase tracking-[0.1em] text-white/50">{label}</span><div className="flex h-10 items-center rounded border border-white/20 bg-black px-3"><input type="number" value={value} min={min} max={max} onChange={event => onChange(Math.max(min, Math.min(max, Number(event.target.value) || min)))} className="min-w-0 flex-1 bg-transparent font-mono text-sm text-white outline-none" /><span className="font-mono text-[10px] text-white/40">{suffix}</span></div></label>;
}

function CustomModel({ onAdd }: { onAdd: (model: CompoundModel) => void }) {
  const [value, setValue] = useState('');
  const add = () => { const id = value.trim(); if (!id) return; onAdd({ id, label: id.split('/').pop() || id, provider: 'Custom', description: 'Custom OpenAI-compatible model', kind: 'custom', accent: 'cyan', weight: 10 }); setValue(''); };
  return <div className="flex gap-2"><input value={value} onChange={event => setValue(event.target.value)} onKeyDown={event => { if (event.key === 'Enter') add(); }} placeholder="provider/model-id" className="input h-10 flex-1 font-mono text-xs" /><button type="button" onClick={add} className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded border border-white/20 bg-white/[0.07] text-white/70 hover:border-white/45 hover:text-white" aria-label="Add custom model"><Plus className="h-4 w-4" /></button></div>;
}
