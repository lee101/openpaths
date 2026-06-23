import React, { useEffect, useRef, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import {
  Agent, ToolSpec, RunStep, AgentRun,
  getAgent, updateAgent, deleteAgent, fetchPresets,
  addSource, uploadSource, deleteSource, runAgentStream,
} from '../lib/agents';

export function AgentDetail() {
  const { id = '' } = useParams();
  const nav = useNavigate();
  const [agent, setAgent] = useState<Agent | null>(null);
  const [tools, setTools] = useState<ToolSpec[]>([]);
  const [saving, setSaving] = useState(false);
  const [err, setErr] = useState('');

  const [input, setInput] = useState('');
  const [steps, setSteps] = useState<RunStep[]>([]);
  const [running, setRunning] = useState(false);
  const [run, setRun] = useState<AgentRun | null>(null);
  const logRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetchPresets().then(r => setTools(r.tools || [])).catch(() => {});
    getAgent(id).then(setAgent).catch(e => setErr(String(e.message || e)));
  }, [id]);

  useEffect(() => { logRef.current?.scrollTo(0, logRef.current.scrollHeight); }, [steps, run]);

  if (err) return <div className="mx-auto max-w-4xl px-6 py-12 font-mono text-sm text-red-400">{err}</div>;
  if (!agent) return <div className="mx-auto max-w-4xl px-6 py-12 font-mono text-sm text-white/35">Loading…</div>;

  const set = (patch: Partial<Agent>) => setAgent({ ...agent, ...patch });
  const setCfg = (patch: Partial<Agent['config']>) => setAgent({ ...agent, config: { ...agent.config, ...patch } });
  const toggleTool = (name: string) => {
    const cur = agent.config.tools || [];
    setCfg({ tools: cur.includes(name) ? cur.filter(t => t !== name) : [...cur, name] });
  };

  async function save() {
    setSaving(true); setErr('');
    try {
      const updated = await updateAgent(id, {
        name: agent.name, description: agent.description, system_prompt: agent.system_prompt,
        model: agent.model, config: agent.config,
      });
      setAgent({ ...updated, sources: agent.sources });
    } catch (e: any) { setErr(String(e.message || e)); }
    finally { setSaving(false); }
  }

  async function remove() {
    if (!confirm('Delete this agent?')) return;
    await deleteAgent(id);
    nav('/agents');
  }

  async function refresh() { setAgent(await getAgent(id)); }

  async function onUpload(e: React.ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0];
    if (!f) return;
    try { await uploadSource(id, f); await refresh(); } catch (er: any) { setErr(String(er.message || er)); }
    e.target.value = '';
  }

  async function addText() {
    const content = prompt('Paste document text to embed:');
    if (!content) return;
    const name = prompt('Source name:', 'Pasted text') || 'Pasted text';
    await addSource(id, { kind: 'document', name, content });
    await refresh();
  }

  async function addDatabase() {
    const dsn = prompt('Read-only database DSN (postgres://user:pass@host/db):');
    if (!dsn) return;
    const name = prompt('Source name:', 'Database') || 'Database';
    await addSource(id, { kind: 'database', name, dsn, driver: 'pgx' });
    await refresh();
  }

  async function doRun() {
    if (!input.trim() || running) return;
    setRunning(true); setSteps([]); setRun(null); setErr('');
    await runAgentStream(id, input,
      s => setSteps(prev => [...prev, s]),
      r => { setRun(r); setRunning(false); },
      e => { setErr(e); setRunning(false); });
  }

  return (
    <div className="mx-auto max-w-7xl px-6 py-10">
      <Link to="/agents" className="text-xs font-mono text-white/40 hover:text-white">← Agents</Link>
      <div className="mt-3 grid gap-8 lg:grid-cols-2">
        {/* Builder */}
        <div className="space-y-5">
          <div className="flex items-center justify-between">
            <h1 className="text-2xl font-bold">Builder</h1>
            <div className="flex gap-2">
              <button onClick={save} disabled={saving} className="bg-white text-black px-3 py-1.5 text-sm font-mono font-bold rounded hover:bg-white/90 disabled:opacity-50">
                {saving ? 'Saving…' : 'Save'}
              </button>
              <button onClick={remove} className="border border-white/20 text-white/70 px-3 py-1.5 text-sm font-mono rounded hover:bg-white/10">Delete</button>
            </div>
          </div>

          <Field label="Name">
            <input value={agent.name} onChange={e => set({ name: e.target.value })} className={inputCls} />
          </Field>
          <Field label="Description">
            <input value={agent.description || ''} onChange={e => set({ description: e.target.value })} className={inputCls} />
          </Field>
          <Field label="Model">
            <input value={agent.model} onChange={e => set({ model: e.target.value })} placeholder="auto" className={inputCls} />
          </Field>
          <Field label="System prompt">
            <textarea value={agent.system_prompt || ''} onChange={e => set({ system_prompt: e.target.value })} rows={5} className={inputCls} />
          </Field>
          <Field label={`Max steps (${agent.config.max_steps || 8})`}>
            <input type="range" min={1} max={20} value={agent.config.max_steps || 8} onChange={e => setCfg({ max_steps: Number(e.target.value) })} className="w-full" />
          </Field>

          <div>
            <div className="text-xs font-mono text-white/50 mb-2">Tools</div>
            <div className="grid sm:grid-cols-2 gap-2">
              {tools.map(t => {
                const on = (agent.config.tools || []).includes(t.name);
                return (
                  <button key={t.name} onClick={() => toggleTool(t.name)}
                    className={`text-left rounded border p-3 transition-colors ${on ? 'border-white/40 bg-white/10' : 'border-white/10 bg-white/5 hover:border-white/20'}`}>
                    <div className="flex items-center gap-2">
                      <span className={`inline-block w-3 h-3 rounded-sm border ${on ? 'bg-white border-white' : 'border-white/40'}`} />
                      <span className="font-mono text-sm">{t.label}</span>
                      {t.experimental && <span className="text-[9px] font-mono px-1 rounded bg-amber-500/20 text-amber-300">beta</span>}
                    </div>
                    <div className="mt-1 text-[11px] font-mono text-white/40">{t.description}</div>
                  </button>
                );
              })}
            </div>
          </div>

          {/* Data sources */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <div className="text-xs font-mono text-white/50">Data sources</div>
              <div className="flex gap-2">
                <button onClick={addText} className="text-xs font-mono border border-white/20 rounded px-2 py-1 hover:bg-white/10">Paste text</button>
                <label className="text-xs font-mono border border-white/20 rounded px-2 py-1 hover:bg-white/10 cursor-pointer">
                  Upload<input type="file" className="hidden" onChange={onUpload} />
                </label>
                <button onClick={addDatabase} className="text-xs font-mono border border-white/20 rounded px-2 py-1 hover:bg-white/10">Database</button>
              </div>
            </div>
            <div className="space-y-2">
              {(agent.sources || []).length === 0 && <div className="text-xs font-mono text-white/35">No sources connected.</div>}
              {(agent.sources || []).map(s => (
                <div key={s.id} className="flex items-center justify-between rounded border border-white/10 bg-white/5 px-3 py-2">
                  <div>
                    <span className="font-mono text-sm">{s.name}</span>
                    <span className="ml-2 text-[10px] font-mono text-white/40">{s.kind} · {s.status} · {s.chunk_count} chunks</span>
                  </div>
                  <button onClick={async () => { await deleteSource(id, s.id); await refresh(); }} className="text-white/40 hover:text-red-400 text-sm">✕</button>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Run console */}
        <div className="space-y-4">
          <h1 className="text-2xl font-bold">Run</h1>
          <textarea value={input} onChange={e => setInput(e.target.value)} rows={3} placeholder="Ask the agent to do something…" className={inputCls} />
          <button onClick={doRun} disabled={running} className="bg-white text-black px-4 py-2 text-sm font-mono font-bold rounded hover:bg-white/90 disabled:opacity-50">
            {running ? 'Running…' : 'Run agent'}
          </button>

          <div ref={logRef} className="h-[28rem] overflow-y-auto rounded border border-white/10 bg-black/40 p-3 space-y-2 font-mono text-xs">
            {steps.length === 0 && !run && <div className="text-white/30">Step trace will appear here.</div>}
            {steps.map((s, i) => <StepView key={i} s={s} />)}
            {run && (
              <div className="mt-3 rounded border border-emerald-500/30 bg-emerald-500/5 p-3">
                <div className="text-emerald-300 mb-1">final · ${(run.cost_cents / 100).toFixed(4)}</div>
                <div className="whitespace-pre-wrap text-white/80">{run.output}</div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

const StepView: React.FC<{ s: RunStep }> = ({ s }) => {
  const color =
    s.type === 'tool' ? 'text-sky-300' :
    s.type === 'error' ? 'text-red-400' :
    s.type === 'final' ? 'text-emerald-300' : 'text-white/50';
  return (
    <div className="rounded border border-white/10 bg-white/5 p-2">
      <div className={color}>
        {s.type}{s.tool ? `: ${s.tool}` : ''}{s.model ? ` · ${s.model}` : ''}{s.elapsed_ms ? ` · ${s.elapsed_ms}ms` : ''}
      </div>
      {s.input && <div className="mt-1 text-white/40 whitespace-pre-wrap break-words">{s.input}</div>}
      {s.output && <div className="mt-1 text-white/70 whitespace-pre-wrap break-words">{s.output.slice(0, 1200)}</div>}
    </div>
  );
};

const inputCls = 'w-full rounded border border-white/10 bg-white/5 px-3 py-2 text-sm font-mono focus:border-white/30 outline-none';

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <div className="text-xs font-mono text-white/50 mb-1">{label}</div>
      {children}
    </label>
  );
}
