import React, { useMemo, useState } from 'react';
import { Check, ChevronDown, Copy, GitMerge, Loader2, Plus, Send, Sparkles, Trash2, X } from 'lucide-react';
import { Seo } from '../components/Seo';
import { CodeBlock } from '../components/CodeBlock';

type FusionPreset = 'quality' | 'budget' | 'custom';
type FusionBackend = 'openpaths' | 'openrouter';

type FusionModel = {
  id: string;
  openpathsId: string;
  label: string;
  provider: string;
  quality?: boolean;
  budget?: boolean;
};

type FusionResponse = {
  finalText: string;
  raw: unknown;
  latencyMs: number;
};

const MODEL_OPTIONS: FusionModel[] = [
  { id: '~openai/gpt-latest', openpathsId: 'gpt-5.5', label: 'OpenAI GPT Latest', provider: 'OpenAI', quality: true },
  { id: '~google/gemini-pro-latest', openpathsId: 'gemini-latest', label: 'Google Gemini Pro Latest', provider: 'Google', quality: true },
  { id: '~google/gemini-flash-latest', openpathsId: 'gemini-3.7-flash', label: 'Gemini Flash Latest', provider: 'Google', budget: true },
  { id: 'deepseek/deepseek-v3.2', openpathsId: 'deepseek-v4-flash', label: 'DeepSeek V3.2', provider: 'DeepSeek', budget: true, quality: true },
  { id: '~moonshotai/kimi-latest', openpathsId: 'kimi-k2.5', label: 'Kimi Latest', provider: 'Moonshot', budget: true },
  { id: 'x-ai/grok-4', openpathsId: 'grok-latest', label: 'Grok Latest', provider: 'xAI' },
  { id: 'qwen/qwen3-max', openpathsId: 'qwen3.5-397b', label: 'Qwen3 Max', provider: 'Qwen' },
  { id: 'glm-5.2', openpathsId: 'glm-5.2', label: 'GLM-5.2', provider: 'ZAI' },
  { id: '~anthropic/claude-opus-latest', openpathsId: 'claude-opus-latest', label: 'Claude Opus Latest', provider: 'Anthropic' },
];

const QUALITY_MODELS = MODEL_OPTIONS.filter(model => model.quality).map(model => model.id);
const BUDGET_MODELS = MODEL_OPTIONS.filter(model => model.budget).map(model => model.id);
const DEFAULT_PROMPT = 'Survey the strongest arguments for and against a carbon tax. Where do experts disagree?';
const FUSION_KEY = 'op_fusion_key';
const DEFAULT_FUSION_PROMPT = 'Compare the panel responses, keep the strongest claims, resolve contradictions, and return one concise best answer.';

function modelLabel(id: string) {
  return MODEL_OPTIONS.find(model => model.id === id)?.label || id;
}

function nativeModelId(id: string) {
  return MODEL_OPTIONS.find(model => model.id === id)?.openpathsId || id;
}

function fusionPayload(backend: FusionBackend, prompt: string, selectedModels: string[], fuseWith: string, forceFusion: boolean, maxToolCalls: number, temperature: number, fusionPrompt: string) {
  const panelModels = selectedModels.slice(0, 8);
  const judgeModel = fuseWith === 'auto' ? panelModels[0] : fuseWith;
  if (backend === 'openpaths') {
    const nativePanel = panelModels.map(nativeModelId);
    const nativeJudge = nativeModelId(judgeModel);
    return {
      model: nativeJudge,
      messages: [{ role: 'user', content: prompt }],
      fusion: {
        type: 'openpaths',
        analysis_models: nativePanel,
        model: nativeJudge,
        prompt: fusionPrompt,
        max_completion_tokens: 4096,
      },
      temperature,
    };
  }
  return {
    model: judgeModel,
    messages: [
      {
        role: 'user',
        content: prompt,
      },
    ],
    tools: [
      {
        type: 'openrouter:fusion',
        parameters: {
          analysis_models: panelModels,
          ...(fuseWith === 'auto' ? {} : { model: fuseWith }),
          max_tool_calls: maxToolCalls,
          temperature,
        },
      },
    ],
    ...(forceFusion ? { tool_choice: 'required' } : {}),
  };
}

function extractFinalText(data: any) {
  const message = data?.choices?.[0]?.message;
  if (typeof message?.content === 'string') return message.content;
  if (Array.isArray(message?.content)) {
    return message.content
      .map((part: any) => typeof part?.text === 'string' ? part.text : '')
      .filter(Boolean)
      .join('\n');
  }
  return JSON.stringify(data, null, 2);
}

export function Fusion() {
  const [apiKey, setApiKey] = useState(() => localStorage.getItem(FUSION_KEY) || '');
  const [backend, setBackend] = useState<FusionBackend>('openpaths');
  const [prompt, setPrompt] = useState(DEFAULT_PROMPT);
  const [fusionPrompt, setFusionPrompt] = useState(DEFAULT_FUSION_PROMPT);
  const [preset, setPreset] = useState<FusionPreset>('quality');
  const [selectedModels, setSelectedModels] = useState<string[]>(QUALITY_MODELS);
  const [customModel, setCustomModel] = useState('');
  const [fuseWith, setFuseWith] = useState('auto');
  const [forceFusion, setForceFusion] = useState(true);
  const [maxToolCalls, setMaxToolCalls] = useState(8);
  const [temperature, setTemperature] = useState(0.4);
  const [showCode, setShowCode] = useState(false);
  const [copied, setCopied] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [response, setResponse] = useState<FusionResponse | null>(null);

  const payload = useMemo(
    () => fusionPayload(backend, prompt, selectedModels, fuseWith, forceFusion, maxToolCalls, temperature, fusionPrompt),
    [backend, forceFusion, fuseWith, fusionPrompt, maxToolCalls, prompt, selectedModels, temperature],
  );

  const endpoint = backend === 'openpaths' ? `${window.location.origin}/v1/chat/completions` : 'https://openrouter.ai/api/v1/chat/completions';
  const keyLabel = backend === 'openpaths' ? '<OPENPATHS_API_KEY>' : '<OPENROUTER_API_KEY>';
  const code = useMemo(() => `const response = await fetch('${endpoint}', {
  method: 'POST',
  headers: {
    Authorization: 'Bearer ${keyLabel}',
    'Content-Type': 'application/json',
  },
  body: JSON.stringify(${JSON.stringify(payload, null, 2)}),
});

const data = await response.json();
console.log(data.choices[0].message.content);`, [endpoint, keyLabel, payload]);

  const applyPreset = (next: FusionPreset) => {
    setPreset(next);
    if (next === 'quality') setSelectedModels(QUALITY_MODELS);
    if (next === 'budget') setSelectedModels(BUDGET_MODELS);
  };

  const toggleModel = (id: string) => {
    setPreset('custom');
    setSelectedModels(current => {
      if (current.includes(id)) {
        if (fuseWith === id) setFuseWith('auto');
        return current.filter(model => model !== id);
      }
      if (current.length >= 8) return current;
      return [...current, id];
    });
  };

  const addModelCopy = (id: string) => {
    setPreset('custom');
    setSelectedModels(current => {
      if (current.length >= 8) return current;
      return [...current, id];
    });
  };

  const removeModelAt = (index: number) => {
    setPreset('custom');
    setSelectedModels(current => {
      const next = current.filter((_, i) => i !== index);
      if (!next.includes(fuseWith)) setFuseWith('auto');
      return next;
    });
  };

  const addCustomModel = () => {
    const id = customModel.trim();
    if (!id || selectedModels.length >= 8) return;
    setPreset('custom');
    setSelectedModels(current => [...current, id]);
    setCustomModel('');
  };

  const copyCode = async () => {
    await navigator.clipboard.writeText(code);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  };

  const runFusion = async () => {
    if (!apiKey.trim()) {
      setError(`Add an ${backend === 'openpaths' ? 'OpenPaths' : 'OpenRouter'} API key to run fusion.`);
      return;
    }
    if (selectedModels.length === 0) {
      setError('Select at least one analysis model.');
      return;
    }
    setLoading(true);
    setError(null);
    setResponse(null);
    localStorage.setItem(FUSION_KEY, apiKey.trim());
    const started = performance.now();
    try {
      const resp = await fetch(endpoint, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${apiKey.trim()}`,
          'Content-Type': 'application/json',
          ...(backend === 'openrouter' ? { 'HTTP-Referer': window.location.origin, 'X-Title': 'OpenPaths Fusion' } : {}),
        },
        body: JSON.stringify(payload),
      });
      const text = await resp.text();
      const data = text ? JSON.parse(text) : {};
      if (!resp.ok) {
        throw new Error(data?.error?.message || `${resp.status}: ${text.slice(0, 240)}`);
      }
      setResponse({
        finalText: extractFinalText(data),
        raw: data,
        latencyMs: Math.round(performance.now() - started),
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Fusion request failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <Seo
        title="Model Fusion Beta | OpenPaths"
        description="Run multiple models side by side with OpenRouter fusion, analyze the panel, and fuse the strongest result into one answer."
        path="/fusion"
      />

      <div className="min-h-full bg-black px-4 py-6 sm:px-6 lg:px-8">
        <div className="mx-auto flex max-w-7xl flex-col gap-5">
          <header className="flex flex-col gap-4 border-b border-white/20 pb-5 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <div className="mb-3 inline-flex items-center gap-2 rounded border border-cyan-300/20 bg-cyan-300/10 px-3 py-1 font-mono text-xs font-bold uppercase tracking-[0.16em] text-cyan-100">
                <GitMerge className="h-3.5 w-3.5" /> Model Fusion <span className="rounded bg-white px-1.5 py-0.5 text-[10px] text-black">Beta</span>
              </div>
              <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">Fuse multiple model answers into one stronger result.</h1>
              <p className="mt-3 max-w-3xl text-sm leading-6 text-white/55">
                Run a panel of models in parallel, let a judge compare consensus and contradictions, then return one final answer using the original prompt and all panel responses.
              </p>
            </div>
            <button
              type="button"
              onClick={runFusion}
              disabled={loading || selectedModels.length === 0}
              className="inline-flex h-11 items-center justify-center gap-2 rounded bg-white px-5 font-mono text-sm font-bold text-black transition-colors hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
              Run Fusion
            </button>
          </header>

          <div className="grid gap-5 lg:grid-cols-[420px_minmax(0,1fr)]">
            <aside className="space-y-4">
              <section className="rounded border border-white/20 bg-white/[0.06] p-4">
                <label className="mb-2 block font-mono text-xs font-bold uppercase tracking-[0.14em] text-white/55">Fusion backend</label>
                <div className="mb-4 flex rounded border border-white/20 bg-black p-1">
                  {(['openpaths', 'openrouter'] as FusionBackend[]).map(option => (
                    <button
                      key={option}
                      type="button"
                      onClick={() => setBackend(option)}
                      className={`h-9 flex-1 rounded font-mono text-xs font-bold capitalize transition-colors ${backend === option ? 'bg-white text-black' : 'text-white/45 hover:text-white'}`}
                    >
                      {option}
                    </button>
                  ))}
                </div>
                <label className="mb-2 block font-mono text-xs font-bold uppercase tracking-[0.14em] text-white/55">
                  {backend === 'openpaths' ? 'OpenPaths API key' : 'OpenRouter API key'}
                </label>
                <input
                  value={apiKey}
                  onChange={event => setApiKey(event.target.value)}
                  type="password"
                  placeholder={backend === 'openpaths' ? 'op-...' : 'sk-or-...'}
                  className="h-11 w-full rounded border border-white/20 bg-black px-3 font-mono text-sm text-white outline-none transition-colors placeholder:text-white/45 focus:border-white/60"
                />
                <p className="mt-2 text-xs leading-5 text-white/50">
                  {backend === 'openpaths'
                    ? 'Runs panel and judge calls through OpenPaths billing and model aliases.'
                    : 'Uses OpenRouter server tools directly with openrouter:fusion.'}
                </p>
              </section>

              <section className="rounded border border-white/20 bg-white/[0.06] p-4">
                <div className="mb-3 flex rounded border border-white/20 bg-black p-1">
                  {(['quality', 'budget', 'custom'] as FusionPreset[]).map(option => (
                    <button
                      key={option}
                      type="button"
                      onClick={() => applyPreset(option)}
                      className={`h-9 flex-1 rounded font-mono text-xs font-bold capitalize transition-colors ${preset === option ? 'bg-white text-black' : 'text-white/45 hover:text-white'}`}
                    >
                      {option}
                    </button>
                  ))}
                </div>

                <div className="space-y-2">
                  {MODEL_OPTIONS.map(model => {
                    const count = selectedModels.filter(id => id === model.id).length;
                    const active = count > 0;
                    return (
                      <div key={model.id} className={`flex items-center gap-1 rounded border transition-colors ${active ? 'border-white/30 bg-white/[0.11]' : 'border-white/20 bg-black'}`}>
                        <button
                          type="button"
                          onClick={() => toggleModel(model.id)}
                          className="flex flex-1 items-center gap-3 px-3 py-2.5 text-left"
                        >
                          <span className={`flex h-5 w-5 shrink-0 items-center justify-center rounded border ${active ? 'border-white bg-white text-black' : 'border-white/20 text-transparent'}`}>
                            <Check className="h-3 w-3" />
                          </span>
                          <span className="min-w-0 flex-1">
                            <span className="block truncate font-mono text-sm text-white">{model.label}</span>
                            <span className="block truncate font-mono text-[11px] text-white/50">{model.id}</span>
                          </span>
                          <span className="font-mono text-[10px] uppercase tracking-[0.12em] text-white/45">{model.provider}</span>
                          {count > 1 && (
                            <span className="rounded bg-white/20 px-1.5 py-0.5 font-mono text-[10px] text-white">{count}x</span>
                          )}
                        </button>
                        <button
                          type="button"
                          onClick={() => addModelCopy(model.id)}
                          disabled={selectedModels.length >= 8}
                          title="Add another copy"
                          className="flex h-full items-center px-2 text-white/45 hover:text-white disabled:opacity-20"
                          aria-label={`Add another ${model.label}`}
                        >
                          <Plus className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    );
                  })}
                </div>

                <div className="mt-3 flex gap-2">
                  <input
                    value={customModel}
                    onChange={event => setCustomModel(event.target.value)}
                    onKeyDown={event => {
                      if (event.key === 'Enter') addCustomModel();
                    }}
                    placeholder="provider/model-id"
                    className="h-10 min-w-0 flex-1 rounded border border-white/20 bg-black px-3 font-mono text-xs text-white outline-none placeholder:text-white/45 focus:border-white/60"
                  />
                  <button type="button" onClick={addCustomModel} className="inline-flex h-10 w-10 items-center justify-center rounded border border-white/20 bg-white/[0.07] text-white/70 hover:border-white/45 hover:text-white" aria-label="Add model">
                    <Plus className="h-4 w-4" />
                  </button>
                </div>

                {selectedModels.some(id => !MODEL_OPTIONS.some(model => model.id === id)) && (
                  <div className="mt-3 flex flex-wrap gap-2">
                    {selectedModels.map((id, i) => !MODEL_OPTIONS.some(model => model.id === id) ? (
                      <button key={i} type="button" onClick={() => removeModelAt(i)} className="inline-flex max-w-full items-center gap-1 rounded border border-white/20 bg-white/[0.07] px-2 py-1 font-mono text-[11px] text-white/55 hover:text-white">
                        <span className="truncate">{id}</span><X className="h-3 w-3" />
                      </button>
                    ) : null)}
                  </div>
                )}
              </section>

              <section className="rounded border border-white/20 bg-white/[0.06] p-4">
                <label className="mb-2 block font-mono text-xs font-bold uppercase tracking-[0.14em] text-white/55">Fuse with</label>
                <div className="relative">
                  <select
                    value={fuseWith}
                    onChange={event => setFuseWith(event.target.value)}
                    className="h-11 w-full appearance-none rounded border border-white/20 bg-black px-3 pr-9 font-mono text-sm text-white outline-none focus:border-white/60"
                  >
                    <option value="auto">Auto (first source)</option>
                    {Array.from(new Set<string>(selectedModels)).map(id => (
                      <option key={id} value={id}>{modelLabel(id)}</option>
                    ))}
                  </select>
                  <ChevronDown className="pointer-events-none absolute right-3 top-3.5 h-4 w-4 text-white/50" />
                </div>

                <div className="mt-4 grid grid-cols-2 gap-3">
                  <label className="block">
                    <span className="mb-1 block font-mono text-[11px] uppercase tracking-[0.12em] text-white/50">Tool calls</span>
                    <input
                      value={maxToolCalls}
                      onChange={event => setMaxToolCalls(Math.max(1, Math.min(16, Number(event.target.value) || 1)))}
                      min={1}
                      max={16}
                      type="number"
                      className="h-10 w-full rounded border border-white/20 bg-black px-3 font-mono text-sm text-white outline-none focus:border-white/60"
                    />
                  </label>
                  <label className="block">
                    <span className="mb-1 block font-mono text-[11px] uppercase tracking-[0.12em] text-white/50">Temperature</span>
                    <input
                      value={temperature}
                      onChange={event => setTemperature(Math.max(0, Math.min(2, Number(event.target.value) || 0)))}
                      min={0}
                      max={2}
                      step={0.1}
                      type="number"
                      className="h-10 w-full rounded border border-white/20 bg-black px-3 font-mono text-sm text-white outline-none focus:border-white/60"
                    />
                  </label>
                </div>

                <label className="mt-4 flex items-center justify-between gap-3 rounded border border-white/20 bg-black px-3 py-2.5">
                  <span>
                    <span className="block font-mono text-sm text-white">Force fusion</span>
                    <span className="block font-mono text-[11px] text-white/50">Sets tool_choice to required.</span>
                  </span>
                  <input type="checkbox" checked={forceFusion} onChange={event => setForceFusion(event.target.checked)} className="h-4 w-4 accent-white" />
                </label>
                {backend === 'openpaths' && (
                  <p className="mt-3 rounded border border-white/20 bg-black px-3 py-2 text-xs leading-5 text-white/50">
                    OpenPaths fusion always runs the selected panel and judge. The force-fusion toggle only affects OpenRouter server-tool requests.
                  </p>
                )}
              </section>
            </aside>

            <main className="space-y-4">
              <section className="rounded border border-white/20 bg-white/[0.06] p-4">
                <div className="mb-3 flex items-center justify-between gap-3">
                  <label className="font-mono text-xs font-bold uppercase tracking-[0.14em] text-white/55">Base prompt</label>
                  <span className="font-mono text-[11px] text-white/45">{selectedModels.length}/8 panel models</span>
                </div>
                <textarea
                  value={prompt}
                  onChange={event => setPrompt(event.target.value)}
                  className="min-h-[170px] w-full resize-y rounded border border-white/20 bg-black p-4 text-sm leading-6 text-white outline-none placeholder:text-white/45 focus:border-white/60"
                  placeholder="Ask a question that benefits from multiple perspectives..."
                />
                <label className="mb-2 mt-4 block font-mono text-xs font-bold uppercase tracking-[0.14em] text-white/55">Fusion prompt</label>
                <textarea
                  value={fusionPrompt}
                  onChange={event => setFusionPrompt(event.target.value)}
                  className="min-h-[92px] w-full resize-y rounded border border-white/20 bg-black p-3 text-sm leading-6 text-white outline-none placeholder:text-white/45 focus:border-white/60"
                  placeholder="Tell the judge how to merge panel answers..."
                />
                <div className="mt-3 flex flex-wrap gap-2">
                  {selectedModels.map((id, i) => (
                    <span key={i} className="inline-flex items-center gap-2 rounded border border-white/20 bg-black px-2.5 py-1.5 font-mono text-xs text-white/55">
                      {modelLabel(id)}
                      <button type="button" onClick={() => removeModelAt(i)} className="text-white/45 hover:text-white" aria-label={`Remove ${modelLabel(id)}`}>
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </span>
                  ))}
                </div>
              </section>

              <section className="rounded border border-white/20 bg-white/[0.06]">
                <div className="flex items-center justify-between border-b border-white/20 px-4 py-3">
                  <div className="inline-flex items-center gap-2 font-mono text-xs font-bold uppercase tracking-[0.14em] text-white/55">
                    <Sparkles className="h-3.5 w-3.5" /> Result
                  </div>
                  {response && <span className="font-mono text-xs text-white/50">{(response.latencyMs / 1000).toFixed(1)}s</span>}
                </div>
                <div className="min-h-[240px] p-4">
                  {error && <div className="rounded border border-red-400/25 bg-red-500/10 p-4 text-sm leading-6 text-red-100">{error}</div>}
                  {loading && (
                    <div className="flex min-h-[200px] items-center justify-center gap-3 font-mono text-sm text-white/45">
                      <Loader2 className="h-5 w-5 animate-spin" /> Running panel, judge, and final fusion...
                    </div>
                  )}
                  {!loading && !error && !response && (
                    <div className="flex min-h-[200px] items-center justify-center text-center font-mono text-sm text-white/45">
                      The fused answer will appear here.
                    </div>
                  )}
                  {response && (
                    <div className="prose prose-invert max-w-none whitespace-pre-wrap text-sm leading-6 text-white/80">
                      {response.finalText}
                    </div>
                  )}
                </div>
              </section>

              <section className="rounded border border-white/20 bg-white/[0.06]">
                <div className="flex items-center justify-between border-b border-white/20 px-4 py-3">
                  <button type="button" onClick={() => setShowCode(open => !open)} className="font-mono text-xs font-bold uppercase tracking-[0.14em] text-white/45 hover:text-white">
                    {showCode ? 'Hide request' : 'Show request'}
                  </button>
                  <button type="button" onClick={copyCode} className="inline-flex items-center gap-2 rounded border border-white/20 px-2.5 py-1.5 font-mono text-xs text-white/50 hover:border-white/45 hover:text-white">
                    {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                    {copied ? 'Copied' : 'Copy'}
                  </button>
                </div>
                {showCode && (
                  <div className="p-4">
                    <CodeBlock code={code} language="typescript" />
                  </div>
                )}
              </section>
            </main>
          </div>
        </div>
      </div>
    </>
  );
}
