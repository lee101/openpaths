import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  Bot,
  Check,
  CheckCircle2,
  Clock3,
  Database,
  FilePlus2,
  FileSearch,
  FileText,
  FlaskConical,
  Loader2,
  Play,
  Save,
  Search,
  Settings2,
  Trash2,
  Upload,
  Wrench,
  X,
  XCircle,
} from 'lucide-react';
import {
  Agent,
  AgentRun,
  KnowledgeResult,
  Preset,
  RunStep,
  ToolSpec,
  addSource,
  deleteAgent,
  deleteSource,
  fetchPresets,
  getAgent,
  listAgentRuns,
  runAgentStream,
  searchAgentKnowledge,
  updateAgent,
  uploadSource,
} from '../lib/agents';

type DetailTab = 'configure' | 'knowledge' | 'test';
type SourceComposer = 'text' | 'database' | null;

const tabs: { key: DetailTab; label: string; icon: React.ComponentType<{ className?: string }> }[] = [
  { key: 'configure', label: 'Configure', icon: Settings2 },
  { key: 'knowledge', label: 'Knowledge', icon: FileSearch },
  { key: 'test', label: 'Test & runs', icon: FlaskConical },
];

export function AgentDetail() {
  const { id = '' } = useParams();
  const nav = useNavigate();
  const [agent, setAgent] = useState<Agent | null>(null);
  const [tools, setTools] = useState<ToolSpec[]>([]);
  const [presets, setPresets] = useState<Preset[]>([]);
  const [tab, setTabState] = useState<DetailTab>(() => {
    const requested = new URLSearchParams(window.location.search).get('tab');
    return requested === 'knowledge' || requested === 'test' ? requested : 'configure';
  });
  const [saving, setSaving] = useState(false);
  const [err, setErr] = useState('');
  const [notice, setNotice] = useState('');
  const [uploading, setUploading] = useState(false);

  const [sourceComposer, setSourceComposer] = useState<SourceComposer>(null);
  const [sourceName, setSourceName] = useState('');
  const [sourceContent, setSourceContent] = useState('');
  const [addingSource, setAddingSource] = useState(false);
  const [knowledgeQuery, setKnowledgeQuery] = useState('');
  const [knowledgeResults, setKnowledgeResults] = useState<KnowledgeResult[]>([]);
  const [searching, setSearching] = useState(false);

  const [input, setInput] = useState('');
  const [steps, setSteps] = useState<RunStep[]>([]);
  const [running, setRunning] = useState(false);
  const [run, setRun] = useState<AgentRun | null>(null);
  const [runs, setRuns] = useState<AgentRun[]>([]);
  const [runsLoading, setRunsLoading] = useState(true);
  const logRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    fetchPresets().then(response => {
      setTools(response.tools || []);
      setPresets(response.presets || []);
    }).catch(() => {});
    getAgent(id).then(setAgent).catch(error => setErr(String(error.message || error)));
    listAgentRuns(id).then(setRuns).catch(() => {}).finally(() => setRunsLoading(false));
    const builderNotice = sessionStorage.getItem('agent_builder_notice');
    if (builderNotice) {
      setNotice(builderNotice);
      sessionStorage.removeItem('agent_builder_notice');
    }
  }, [id]);

  useEffect(() => {
    if (!agent?.sources?.some(source => source.status === 'ingesting')) return;
    const timer = window.setInterval(() => getAgent(id).then(setAgent).catch(() => {}), 2000);
    return () => window.clearInterval(timer);
  }, [agent?.sources, id]);

  useEffect(() => { logRef.current?.scrollTo(0, logRef.current.scrollHeight); }, [steps, run]);

  const selectedPreset = useMemo(() => presets.find(preset => preset.key === agent?.preset), [agent?.preset, presets]);
  const readySources = agent?.sources?.filter(source => source.status === 'ready').length || 0;
  const totalChunks = agent?.sources?.reduce((sum, source) => sum + (source.chunk_count || 0), 0) || 0;

  function setTab(next: DetailTab) {
    setTabState(next);
    window.history.replaceState({}, '', `/agents/${id}?tab=${next}`);
  }

  if (err && !agent) return <div className="mx-auto max-w-4xl px-6 py-12 font-mono text-sm text-red-400">{err}</div>;
  if (!agent) return <div className="mx-auto flex max-w-4xl items-center gap-2 px-6 py-12 font-mono text-sm text-white/35"><Loader2 className="h-4 w-4 animate-spin" /> Loading agent…</div>;

  const set = (patch: Partial<Agent>) => setAgent({ ...agent, ...patch });
  const setCfg = (patch: Partial<Agent['config']>) => setAgent({ ...agent, config: { ...agent.config, ...patch } });
  const toggleTool = (name: string) => {
    const current = agent.config.tools || [];
    setCfg({ tools: current.includes(name) ? current.filter(tool => tool !== name) : [...current, name] });
  };

  async function save() {
    setSaving(true); setErr('');
    try {
      const updated = await updateAgent(id, {
        name: agent.name,
        description: agent.description,
        system_prompt: agent.system_prompt,
        model: agent.model,
        config: agent.config,
      });
      setAgent({ ...updated, sources: agent.sources });
      setNotice('Agent configuration saved.');
    } catch (error: any) { setErr(String(error.message || error)); }
    finally { setSaving(false); }
  }

  async function remove() {
    if (!confirm(`Delete “${agent.name}” and its connected knowledge?`)) return;
    try {
      await deleteAgent(id);
      nav('/agents');
    } catch (error: any) { setErr(String(error.message || error)); }
  }

  async function refresh() { setAgent(await getAgent(id)); }

  async function onUpload(event: React.ChangeEvent<HTMLInputElement>) {
    const selected = Array.from(event.target.files || []) as File[];
    event.target.value = '';
    if (!selected.length) return;
    const accepted = selected.filter(file => file.size <= 25 * 1024 * 1024);
    const failed = selected.filter(file => file.size > 25 * 1024 * 1024).map(file => file.name);
    setUploading(true); setErr('');
    for (let index = 0; index < accepted.length; index++) {
      setNotice(`Converting ${index + 1} of ${accepted.length}: ${accepted[index].name}`);
      try { await uploadSource(id, accepted[index]); }
      catch { failed.push(accepted[index].name); }
    }
    try { await refresh(); } catch {}
    setUploading(false);
    setNotice(failed.length ? `Finished, but these files could not be added: ${failed.join(', ')}` : `${accepted.length} file${accepted.length === 1 ? '' : 's'} added and indexing.`);
  }

  function openSourceComposer(kind: Exclude<SourceComposer, null>) {
    setSourceComposer(kind);
    setSourceName(kind === 'text' ? 'Pasted text' : 'Database');
    setSourceContent('');
  }

  async function submitSource() {
    if (!sourceContent.trim() || addingSource) return;
    setAddingSource(true); setErr('');
    try {
      if (sourceComposer === 'database') {
        await addSource(id, { kind: 'database', name: sourceName || 'Database', dsn: sourceContent.trim(), driver: 'pgx' });
      } else {
        await addSource(id, { kind: 'document', name: sourceName || 'Pasted text', content: sourceContent });
      }
      await refresh();
      setSourceComposer(null);
      setSourceContent('');
      setNotice(sourceComposer === 'database' ? 'Read-only database connected.' : 'Text added and indexing.');
    } catch (error: any) { setErr(String(error.message || error)); }
    finally { setAddingSource(false); }
  }

  async function removeSource(sourceID: string, name: string) {
    if (!confirm(`Remove “${name}” from this agent?`)) return;
    try {
      await deleteSource(id, sourceID);
      await refresh();
      setKnowledgeResults(results => results.filter(result => result.data_source_id !== sourceID));
    } catch (error: any) { setErr(String(error.message || error)); }
  }

  async function searchKnowledge(event: React.FormEvent) {
    event.preventDefault();
    if (!knowledgeQuery.trim() || searching) return;
    setSearching(true); setErr('');
    try { setKnowledgeResults(await searchAgentKnowledge(id, knowledgeQuery.trim())); }
    catch (error: any) { setErr(String(error.message || error)); }
    finally { setSearching(false); }
  }

  async function doRun() {
    if (!input.trim() || running) return;
    const runInput = input.trim();
    setRunning(true); setSteps([]); setRun(null); setErr('');
    await runAgentStream(id, runInput,
      step => setSteps(previous => [...previous, step]),
      completed => {
        setRun(completed);
        setRuns(previous => [completed, ...previous.filter(item => item.id !== completed.id)]);
        setRunning(false);
      },
      error => { setErr(error); setRunning(false); });
  }

  function inspectRun(previous: AgentRun) {
    setInput(previous.input);
    setSteps(previous.steps || []);
    setRun(previous);
  }

  return (
    <div className="mx-auto max-w-7xl px-6 py-10 sm:py-12">
      <Link to="/agents" className="inline-flex items-center gap-1.5 font-mono text-xs text-white/40 hover:text-white"><ArrowLeft className="h-3.5 w-3.5" /> Agents</Link>

      <header className="mt-5 flex flex-col justify-between gap-6 sm:flex-row sm:items-end">
        <div className="min-w-0">
          <div className="flex items-center gap-3">
            <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl border border-white/10 bg-white/[0.04]"><Bot className="h-5 w-5 text-white/65" /></div>
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h1 className="truncate text-2xl font-semibold tracking-tight sm:text-3xl">{agent.name}</h1>
                {agent.preset && <span className="rounded-md border border-white/10 px-2 py-0.5 font-mono text-[9px] uppercase tracking-wider text-white/35">{agent.preset} preset</span>}
              </div>
              <p className="mt-1 max-w-2xl truncate text-sm text-white/40">{agent.description || 'Custom OpenPaths agent'}</p>
            </div>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <SummaryPill icon={Wrench} value={`${agent.config.tools?.length || 0} tools`} />
          <SummaryPill icon={FileText} value={`${agent.sources?.length || 0} sources`} />
          <button onClick={() => setTab('test')} className="ml-1 inline-flex items-center gap-2 rounded-lg bg-white px-3.5 py-2 font-mono text-xs font-bold text-black hover:bg-white/90"><Play className="h-3.5 w-3.5 fill-current" /> Test agent</button>
        </div>
      </header>

      {notice && <Notice tone="info" onClose={() => setNotice('')}>{notice}</Notice>}
      {err && <Notice tone="error" onClose={() => setErr('')}>{err}</Notice>}

      <nav className="mt-8 flex gap-1 overflow-x-auto border-b border-white/10" aria-label="Agent sections">
        {tabs.map(item => {
          const Icon = item.icon;
          const active = tab === item.key;
          return (
            <button key={item.key} onClick={() => setTab(item.key)} aria-current={active ? 'page' : undefined} className={`relative flex shrink-0 items-center gap-2 px-4 py-3 font-mono text-xs transition-colors ${active ? 'text-white' : 'text-white/40 hover:text-white/70'}`}>
              <Icon className="h-3.5 w-3.5" /> {item.label}
              {active && <span className="absolute inset-x-2 bottom-0 h-px bg-white" />}
            </button>
          );
        })}
      </nav>

      {tab === 'configure' && (
        <div className="mt-7 grid gap-6 lg:grid-cols-[minmax(0,1.15fr)_minmax(18rem,.85fr)]">
          <section className="space-y-6 rounded-xl border border-white/10 bg-white/[0.025] p-5 sm:p-6">
            <SectionHeading title="Identity & behavior" description="Define what this agent is and how it should respond." />
            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="Name"><input value={agent.name} onChange={event => set({ name: event.target.value })} className={inputCls} /></Field>
              <Field label="Model" hint="Use auto for intelligent routing"><input value={agent.model} onChange={event => set({ model: event.target.value })} placeholder="auto" className={`${inputCls} font-mono`} /></Field>
            </div>
            <Field label="Description"><input value={agent.description || ''} onChange={event => set({ description: event.target.value })} className={inputCls} /></Field>
            <Field label="Instructions" hint="Role, boundaries, process, and desired output">
              <textarea value={agent.system_prompt || ''} onChange={event => set({ system_prompt: event.target.value })} rows={9} className={`${inputCls} resize-y font-mono text-xs leading-5`} />
            </Field>
            <div className="grid gap-5 sm:grid-cols-2">
              <Field label={`Maximum steps: ${agent.config.max_steps || 8}`} hint="Limits tool-loop depth">
                <input aria-label="Maximum steps" type="range" min={1} max={20} value={agent.config.max_steps || 8} onChange={event => setCfg({ max_steps: Number(event.target.value) })} className="mt-2 w-full accent-white" />
              </Field>
              <Field label={`Temperature: ${agent.config.temperature ?? 0.4}`} hint="Lower is more predictable">
                <input aria-label="Temperature" type="range" min={0} max={1.5} step={0.1} value={agent.config.temperature ?? 0.4} onChange={event => setCfg({ temperature: Number(event.target.value) })} className="mt-2 w-full accent-white" />
              </Field>
            </div>
            <div className="flex justify-end border-t border-white/10 pt-5">
              <button onClick={save} disabled={saving || !agent.name.trim()} className="inline-flex items-center gap-2 rounded-lg bg-white px-4 py-2 font-mono text-xs font-bold text-black hover:bg-white/90 disabled:opacity-40">
                {saving ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Save className="h-3.5 w-3.5" />} {saving ? 'Saving…' : 'Save changes'}
              </button>
            </div>
          </section>

          <div className="space-y-6">
            <section className="rounded-xl border border-white/10 bg-white/[0.025] p-5 sm:p-6">
              <SectionHeading title="Capabilities" description="Grant only the tools this agent needs." />
              <div className="mt-5 space-y-2">
                {tools.map(tool => {
                  const enabled = agent.config.tools?.includes(tool.name);
                  return (
                    <button key={tool.name} onClick={() => toggleTool(tool.name)} aria-pressed={enabled} className={`w-full rounded-lg border p-3 text-left transition-colors ${enabled ? 'border-white/30 bg-white/[0.07]' : 'border-white/10 bg-black/20 hover:border-white/20'}`}>
                      <div className="flex items-center gap-2">
                        <span className={`flex h-4 w-4 items-center justify-center rounded border ${enabled ? 'border-white bg-white text-black' : 'border-white/25'}`}>{enabled && <Check className="h-3 w-3" />}</span>
                        <span className="font-mono text-xs text-white/75">{tool.label}</span>
                        {tool.experimental && <span className="rounded bg-amber-400/10 px-1 font-mono text-[9px] text-amber-200">beta</span>}
                      </div>
                      <p className="mt-1.5 pl-6 text-[11px] leading-4 text-white/35">{tool.description}</p>
                    </button>
                  );
                })}
              </div>
            </section>
            <section className="rounded-xl border border-red-400/10 bg-red-400/[0.025] p-5">
              <h2 className="text-sm font-medium text-red-100/80">Danger zone</h2>
              <p className="mt-1 text-xs leading-5 text-white/35">Deleting an agent also removes its sources, indexed chunks, and run history.</p>
              <button onClick={remove} className="mt-4 inline-flex items-center gap-2 rounded-lg border border-red-400/20 px-3 py-2 font-mono text-xs text-red-200 hover:bg-red-400/10"><Trash2 className="h-3.5 w-3.5" /> Delete agent</button>
            </section>
          </div>
        </div>
      )}

      {tab === 'knowledge' && (
        <div className="mt-7 grid gap-6 lg:grid-cols-[minmax(0,1.25fr)_minmax(19rem,.75fr)]">
          <section className="rounded-xl border border-white/10 bg-white/[0.025] p-5 sm:p-6">
            <div className="flex flex-col justify-between gap-4 sm:flex-row sm:items-start">
              <SectionHeading title="Connected knowledge" description={`${readySources} ready sources · ${totalChunks} searchable chunks`} />
              <div className="flex flex-wrap gap-2">
                <button onClick={() => openSourceComposer('text')} className="inline-flex items-center gap-1.5 rounded-lg border border-white/15 px-3 py-2 font-mono text-xs text-white/65 hover:border-white/30 hover:text-white"><FilePlus2 className="h-3.5 w-3.5" /> Paste text</button>
                <label className={`inline-flex cursor-pointer items-center gap-1.5 rounded-lg bg-white px-3 py-2 font-mono text-xs font-bold text-black hover:bg-white/90 ${uploading ? 'pointer-events-none opacity-50' : ''}`}>
                  {uploading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Upload className="h-3.5 w-3.5" />} {uploading ? 'Converting…' : 'Upload files'}
                  <input type="file" multiple className="hidden" accept=".pdf,.doc,.docx,.pptx,.xls,.xlsx,.html,.htm,.md,.markdown,.txt,.csv,.json,.epub,.ipynb,.zip,.rtf,.odt" onChange={onUpload} />
                </label>
                <button onClick={() => openSourceComposer('database')} className="inline-flex items-center gap-1.5 rounded-lg border border-white/15 px-3 py-2 font-mono text-xs text-white/65 hover:border-white/30 hover:text-white"><Database className="h-3.5 w-3.5" /> Database</button>
              </div>
            </div>

            {sourceComposer && (
              <div className="mt-5 rounded-xl border border-sky-400/15 bg-sky-400/[0.035] p-4">
                <div className="flex items-start justify-between">
                  <div>
                    <h3 className="text-sm font-medium">{sourceComposer === 'database' ? 'Connect a read-only database' : 'Add pasted knowledge'}</h3>
                    <p className="mt-1 text-[11px] leading-4 text-white/35">{sourceComposer === 'database' ? 'Queries are restricted to read-only SELECT statements.' : 'Text is converted into searchable chunks just like an uploaded file.'}</p>
                  </div>
                  <button onClick={() => setSourceComposer(null)} aria-label="Close source form" className="text-white/30 hover:text-white"><X className="h-4 w-4" /></button>
                </div>
                <div className="mt-4 grid gap-3">
                  <Field label="Source name"><input value={sourceName} onChange={event => setSourceName(event.target.value)} className={inputCls} /></Field>
                  <Field label={sourceComposer === 'database' ? 'PostgreSQL connection string' : 'Content'}>
                    {sourceComposer === 'database' ? (
                      <input type="password" autoComplete="off" value={sourceContent} onChange={event => setSourceContent(event.target.value)} placeholder="postgres://readonly:password@host/database" className={`${inputCls} font-mono text-xs`} />
                    ) : (
                      <textarea value={sourceContent} onChange={event => setSourceContent(event.target.value)} rows={7} placeholder="Paste notes, policies, documentation, or reference material…" className={`${inputCls} resize-y text-xs leading-5`} />
                    )}
                  </Field>
                  <div className="flex justify-end gap-2">
                    <button onClick={() => setSourceComposer(null)} className="rounded-lg px-3 py-2 font-mono text-xs text-white/45 hover:text-white">Cancel</button>
                    <button onClick={submitSource} disabled={!sourceContent.trim() || addingSource} className="inline-flex items-center gap-2 rounded-lg bg-white px-3 py-2 font-mono text-xs font-bold text-black disabled:opacity-40">{addingSource && <Loader2 className="h-3.5 w-3.5 animate-spin" />} Add source</button>
                  </div>
                </div>
              </div>
            )}

            <div className="mt-6 space-y-2">
              {!agent.sources?.length && (
                <label className="flex cursor-pointer flex-col items-center rounded-xl border border-dashed border-white/15 px-5 py-12 text-center hover:border-white/30 hover:bg-white/[0.02]">
                  <Upload className="h-6 w-6 text-white/35" />
                  <span className="mt-3 text-sm font-medium text-white/70">Give this agent something to know</span>
                  <span className="mt-1 max-w-md text-xs leading-5 text-white/35">Upload PDF, Word, PowerPoint, Excel, HTML, Markdown, CSV, or JSON files. Documents are normalized and indexed automatically.</span>
                  <input type="file" multiple className="hidden" onChange={onUpload} />
                </label>
              )}
              {(agent.sources || []).map(source => (
                <div key={source.id} className="flex items-center justify-between gap-3 rounded-lg border border-white/10 bg-black/20 px-3 py-3">
                  <div className="flex min-w-0 items-center gap-3">
                    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-white/5">
                      {source.status === 'ingesting' ? <Loader2 className="h-4 w-4 animate-spin text-sky-300" /> : source.status === 'error' ? <XCircle className="h-4 w-4 text-red-300" /> : source.status === 'ready' ? <CheckCircle2 className="h-4 w-4 text-emerald-300" /> : source.kind === 'database' ? <Database className="h-4 w-4 text-white/40" /> : <FileText className="h-4 w-4 text-white/40" />}
                    </div>
                    <div className="min-w-0">
                      <div className="truncate font-mono text-xs text-white/75">{source.name}</div>
                      <div className="mt-1 flex flex-wrap gap-x-2 font-mono text-[9px] text-white/30">
                        <span>{source.status === 'ingesting' ? 'indexing' : source.status}</span><span>{source.chunk_count} chunks</span>
                        {source.meta?.parser && <span>{source.meta.parser}</span>}
                        {source.meta?.images_kept > 0 && <span>{source.meta.images_kept} useful images</span>}
                      </div>
                    </div>
                  </div>
                  <button aria-label={`Remove ${source.name}`} onClick={() => removeSource(source.id, source.name)} className="shrink-0 rounded-md p-2 text-white/25 hover:bg-red-400/10 hover:text-red-300"><Trash2 className="h-3.5 w-3.5" /></button>
                </div>
              ))}
            </div>
          </section>

          <aside className="rounded-xl border border-white/10 bg-white/[0.025] p-5 sm:p-6">
            <SectionHeading title="Test retrieval" description="Preview what the agent can find before you run it." />
            <form onSubmit={searchKnowledge} className="mt-5 flex gap-2">
              <div className="relative min-w-0 flex-1"><Search className="absolute left-3 top-2.5 h-3.5 w-3.5 text-white/30" /><input value={knowledgeQuery} onChange={event => setKnowledgeQuery(event.target.value)} placeholder="Search connected knowledge" className={`${inputCls} pl-9 text-xs`} /></div>
              <button aria-label="Search knowledge" disabled={!knowledgeQuery.trim() || searching || !readySources} className="rounded-lg border border-white/15 px-3 text-white/60 hover:text-white disabled:opacity-30">{searching ? <Loader2 className="h-4 w-4 animate-spin" /> : <Search className="h-4 w-4" />}</button>
            </form>
            <div className="mt-4 space-y-3">
              {!knowledgeResults.length && <div className="rounded-lg border border-dashed border-white/10 px-4 py-8 text-center text-[11px] leading-5 text-white/30">{readySources ? 'Search to inspect the chunks your agent will receive.' : 'Add and finish indexing a source to test retrieval.'}</div>}
              {knowledgeResults.map((result, index) => (
                <article key={result.id || `${result.data_source_id}-${result.chunk_index}`} className="rounded-lg border border-white/10 bg-black/25 p-3">
                  <div className="flex items-center justify-between gap-2 font-mono text-[9px] text-white/30"><span className="truncate">{result.title || `Result ${index + 1}`}</span><span>chunk {result.chunk_index + 1}</span></div>
                  <p className="mt-2 line-clamp-6 whitespace-pre-wrap text-[11px] leading-5 text-white/55">{result.content}</p>
                </article>
              ))}
            </div>
          </aside>
        </div>
      )}

      {tab === 'test' && (
        <div className="mt-7 grid gap-6 lg:grid-cols-[minmax(0,1.35fr)_minmax(18rem,.65fr)]">
          <section className="rounded-xl border border-white/10 bg-white/[0.025] p-5 sm:p-6">
            <SectionHeading title="Run agent" description="Try a real task and inspect every model turn and tool call." />
            {!!selectedPreset?.example_prompts?.length && (
              <div className="mt-4 flex flex-wrap gap-2">
                {selectedPreset.example_prompts.map(prompt => <button key={prompt} onClick={() => setInput(prompt)} className="rounded-full border border-white/10 px-3 py-1.5 text-left text-[10px] text-white/45 hover:border-white/25 hover:text-white/70">{prompt}</button>)}
              </div>
            )}
            <div className="mt-5">
              <textarea value={input} onChange={event => setInput(event.target.value)} onKeyDown={event => { if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') doRun(); }} rows={4} placeholder="Ask the agent to research, create, compare, or complete a task…" className={`${inputCls} resize-y leading-6`} />
              <div className="mt-2 flex items-center justify-between gap-3">
                <span className="font-mono text-[9px] text-white/25">Ctrl/⌘ + Enter to run</span>
                <button onClick={doRun} disabled={running || !input.trim()} className="inline-flex items-center gap-2 rounded-lg bg-white px-4 py-2 font-mono text-xs font-bold text-black hover:bg-white/90 disabled:opacity-40">{running ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Play className="h-3.5 w-3.5 fill-current" />} {running ? 'Running…' : 'Run agent'}</button>
              </div>
            </div>
            <div ref={logRef} className="mt-5 min-h-80 max-h-[38rem] overflow-y-auto rounded-xl border border-white/10 bg-black/35 p-3 font-mono text-xs">
              {!steps.length && !run && (
                <div className="flex min-h-72 flex-col items-center justify-center px-6 text-center">
                  <FlaskConical className="h-6 w-6 text-white/20" />
                  <div className="mt-3 text-white/45">The execution trace will appear here.</div>
                  <div className="mt-1 max-w-sm text-[10px] leading-4 text-white/25">You’ll see which model ran, what tools it called, how long each step took, and the final answer.</div>
                </div>
              )}
              <div className="space-y-2">{steps.map((step, index) => <StepView key={index} step={step} />)}</div>
              {run && (
                <div className="mt-3 rounded-lg border border-emerald-400/20 bg-emerald-400/5 p-4">
                  <div className="mb-2 flex items-center justify-between text-emerald-200"><span>final answer</span><span className="text-[9px]">{formatCost(run.cost_cents)}</span></div>
                  <div className="whitespace-pre-wrap font-sans text-sm leading-6 text-white/75">{run.output}</div>
                </div>
              )}
            </div>
          </section>

          <aside className="rounded-xl border border-white/10 bg-white/[0.025] p-5 sm:p-6">
            <SectionHeading title="Recent runs" description="Reopen a previous input and trace." />
            <div className="mt-5 space-y-2">
              {runsLoading && <div className="flex items-center gap-2 font-mono text-xs text-white/30"><Loader2 className="h-3.5 w-3.5 animate-spin" /> Loading history…</div>}
              {!runsLoading && !runs.length && <div className="rounded-lg border border-dashed border-white/10 px-4 py-8 text-center text-[11px] leading-5 text-white/30">No runs yet. Your first completed run will be saved here.</div>}
              {runs.map(previous => (
                <button key={previous.id} onClick={() => inspectRun(previous)} className={`w-full rounded-lg border p-3 text-left transition-colors ${run?.id === previous.id ? 'border-white/25 bg-white/[0.06]' : 'border-white/10 bg-black/20 hover:border-white/20'}`}>
                  <div className="line-clamp-2 text-xs leading-5 text-white/60">{previous.input}</div>
                  <div className="mt-2 flex items-center justify-between gap-2 font-mono text-[9px] text-white/25"><span className="inline-flex items-center gap-1"><Clock3 className="h-3 w-3" />{formatDate(previous.created_at)}</span><span>{formatCost(previous.cost_cents)}</span></div>
                </button>
              ))}
            </div>
          </aside>
        </div>
      )}
    </div>
  );
}

function SummaryPill({ icon: Icon, value }: { icon: React.ComponentType<{ className?: string }>; value: string }) {
  return <span className="inline-flex items-center gap-1.5 rounded-lg border border-white/10 bg-white/[0.03] px-2.5 py-2 font-mono text-[10px] text-white/40"><Icon className="h-3 w-3" />{value}</span>;
}

function Notice({ tone, onClose, children }: { tone: 'info' | 'error'; onClose: () => void; children: React.ReactNode }) {
  return <div className={`mt-5 flex items-start justify-between gap-4 rounded-lg border px-4 py-3 text-xs ${tone === 'error' ? 'border-red-400/20 bg-red-400/5 text-red-200' : 'border-sky-400/20 bg-sky-400/5 text-sky-100/70'}`}><span>{children}</span><button onClick={onClose} aria-label="Dismiss message" className="text-white/35 hover:text-white">×</button></div>;
}

function SectionHeading({ title, description }: { title: string; description: string }) {
  return <div><h2 className="text-base font-semibold">{title}</h2><p className="mt-1 text-xs leading-5 text-white/35">{description}</p></div>;
}

function StepView({ step }: { step: RunStep; key?: React.Key }) {
  const color = step.type === 'tool' ? 'text-sky-300' : step.type === 'error' ? 'text-red-400' : step.type === 'final' ? 'text-emerald-300' : 'text-white/50';
  return (
    <div className="rounded-lg border border-white/10 bg-white/[0.035] p-3">
      <div className={`flex flex-wrap items-center gap-x-2 ${color}`}>
        <span>{step.type}{step.tool ? `: ${step.tool}` : ''}</span>
        {step.model && <span className="text-white/25">{step.model}</span>}
        {!!step.elapsed_ms && <span className="text-white/25">{step.elapsed_ms}ms</span>}
      </div>
      {step.input && <div className="mt-2 whitespace-pre-wrap break-words text-white/35">{step.input}</div>}
      {step.output && <div className="mt-2 whitespace-pre-wrap break-words text-white/65">{step.output.slice(0, 1800)}</div>}
    </div>
  );
}

const inputCls = 'w-full rounded-lg border border-white/10 bg-black/35 px-3 py-2.5 text-sm text-white outline-none transition-colors placeholder:text-white/20 focus:border-white/30';

function Field({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <div className="mb-1.5 flex items-baseline justify-between gap-3"><span className="text-xs font-medium text-white/60">{label}</span>{hint && <span className="text-[9px] text-white/25">{hint}</span>}</div>
      {children}
    </label>
  );
}

function formatCost(costUnits: number) {
  return costUnits ? `$${(costUnits / 10000).toFixed(4)}` : '$0.0000';
}

function formatDate(value?: string) {
  if (!value) return 'just now';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 'recently';
  return date.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}
