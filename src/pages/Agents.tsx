import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import {
  ArrowRight,
  Bot,
  BrainCircuit,
  Check,
  Clapperboard,
  FileText,
  Library,
  Loader2,
  Monitor,
  Plus,
  Sparkles,
  Upload,
  Wrench,
  X,
} from 'lucide-react';
import { Agent, AgentConfig, Preset, ToolSpec, listAgents, fetchPresets, createAgent, uploadSource } from '../lib/agents';
import { getStoredAPIKey, getStoredToken } from '../lib/session';

type Draft = {
  name: string;
  description: string;
  systemPrompt: string;
  model: string;
  config: AgentConfig;
};

const blankDraft: Draft = {
  name: '',
  description: '',
  systemPrompt: 'You are a capable assistant. Use the available tools when they help, and be clear when you need more information.',
  model: 'auto',
  config: { tools: ['search_documents'], max_steps: 8 },
};

const presetIcons: Record<string, React.ComponentType<{ className?: string }>> = {
  creative: Clapperboard,
  researcher: Library,
  operator: Monitor,
  generalist: Sparkles,
};

export function Agents() {
  const nav = useNavigate();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [presets, setPresets] = useState<Preset[]>([]);
  const [tools, setTools] = useState<ToolSpec[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState('');
  const [builderOpen, setBuilderOpen] = useState(false);
  const [selectedPreset, setSelectedPreset] = useState<Preset | null>(null);
  const [draft, setDraft] = useState<Draft>(blankDraft);
  const [files, setFiles] = useState<File[]>([]);
  const [creating, setCreating] = useState(false);
  const [progress, setProgress] = useState('');
  const loggedIn = !!(getStoredToken() || getStoredAPIKey());

  useEffect(() => {
    fetchPresets()
      .then(r => {
        setPresets(r.presets || []);
        setTools(r.tools || []);
        const requested = new URLSearchParams(window.location.search).get('preset');
        if (loggedIn && requested) {
          const preset = requested === 'custom' ? null : (r.presets || []).find(item => item.key === requested);
          if (requested === 'custom' || preset) openBuilder(preset || null);
          window.history.replaceState({}, '', '/agents');
        }
      })
      .catch(e => setErr(String(e.message || e)));
    if (loggedIn) {
      listAgents().then(setAgents).catch(e => setErr(String(e.message || e))).finally(() => setLoading(false));
    } else {
      setLoading(false);
    }
  }, [loggedIn]);

  const toolLabels = useMemo(() => new Map(tools.map(tool => [tool.name, tool.label])), [tools]);

  function openBuilder(preset: Preset | null) {
    setSelectedPreset(preset);
    setDraft(preset ? {
      name: preset.name,
      description: preset.description,
      systemPrompt: preset.system_prompt,
      model: preset.model,
      config: { ...preset.config, tools: [...(preset.config.tools || [])] },
    } : { ...blankDraft, config: { ...blankDraft.config, tools: [...blankDraft.config.tools] } });
    setFiles([]);
    setProgress('');
    setErr('');
    setBuilderOpen(true);
  }

  function toggleTool(name: string) {
    const current = draft.config.tools || [];
    setDraft(value => ({
      ...value,
      config: { ...value.config, tools: current.includes(name) ? current.filter(tool => tool !== name) : [...current, name] },
    }));
  }

  function addFiles(incoming: FileList | null) {
    if (!incoming) return;
    const selected = Array.from(incoming);
    const accepted = selected.filter(file => file.size <= 25 * 1024 * 1024);
    const oversized = selected.filter(file => file.size > 25 * 1024 * 1024);
    if (oversized.length) setErr(`${oversized.map(file => file.name).join(', ')} exceeded the 25 MB file limit.`);
    setFiles(current => [...current, ...accepted].slice(0, 8));
  }

  async function buildAgent() {
    if (!draft.name.trim() || creating) return;
    setCreating(true);
    setErr('');
    try {
      setProgress('Creating agent…');
      const created = await createAgent({
        name: draft.name.trim(),
        description: draft.description.trim(),
        system_prompt: draft.systemPrompt.trim(),
        model: draft.model.trim() || 'auto',
        config: draft.config,
        preset: selectedPreset?.key,
      });
      const failed: string[] = [];
      for (let i = 0; i < files.length; i++) {
        setProgress(`Converting file ${i + 1} of ${files.length}: ${files[i].name}`);
        try {
          await uploadSource(created.id, files[i]);
        } catch {
          failed.push(files[i].name);
        }
      }
      if (failed.length) {
        sessionStorage.setItem('agent_builder_notice', `Agent created. These files could not be added: ${failed.join(', ')}`);
      }
      nav(`/agents/${created.id}`);
    } catch (e: any) {
      setErr(String(e.message || e));
      setProgress('');
      setCreating(false);
    }
  }

  return (
    <div className="mx-auto max-w-7xl px-6 py-12 sm:py-16">
      <section className="relative overflow-hidden rounded-2xl border border-white/20 bg-gradient-to-br from-white/[0.08] via-white/[0.03] to-transparent px-6 py-9 sm:px-10 sm:py-12">
        <div className="pointer-events-none absolute -right-24 -top-36 h-80 w-80 rounded-full bg-violet-500/10 blur-3xl" />
        <div className="relative max-w-3xl">
          <div className="mb-5 inline-flex items-center gap-2 rounded-full border border-white/20 bg-black/30 px-3 py-1 font-mono text-[11px] text-white/55">
            <BrainCircuit className="h-3.5 w-3.5" /> OpenPaths Agents
          </div>
          <h1 className="text-4xl font-semibold tracking-tight sm:text-5xl">Give a model a job, tools, and knowledge.</h1>
          <p className="mt-5 max-w-2xl text-base leading-7 text-white/55">
            Start with a working preset or build your own. Add documents, choose exactly which tools it can use, then run it with a visible step-by-step trace.
          </p>
          <div className="mt-7 flex flex-wrap gap-3">
            <button onClick={() => openBuilder(null)} className="inline-flex items-center gap-2 rounded-lg bg-white px-4 py-2.5 font-mono text-sm font-bold text-black transition-colors hover:bg-white/90">
              <Plus className="h-4 w-4" /> Build your own
            </button>
            <a href="#presets" className="inline-flex items-center gap-2 rounded-lg border border-white/15 px-4 py-2.5 font-mono text-sm text-white/70 transition-colors hover:border-white/50 hover:text-white">
              Explore presets <ArrowRight className="h-4 w-4" />
            </a>
          </div>
        </div>
      </section>

      {err && !builderOpen && <div className="mt-5 rounded-lg border border-red-400/20 bg-red-400/5 px-4 py-3 font-mono text-sm text-red-300">{err}</div>}

      <section className="mt-10 grid gap-px overflow-hidden rounded-xl border border-white/20 bg-white/10 sm:grid-cols-3">
        <WorkflowStep icon={Bot} number="01" title="Choose its role" text="Use a preset or define the instructions, model, and tools yourself." />
        <WorkflowStep icon={FileText} number="02" title="Add knowledge" text="Upload files that are converted into clean, searchable Markdown." />
        <WorkflowStep icon={Wrench} number="03" title="Run with a trace" text="See model turns, tool calls, timing, output, and cost as it works." />
      </section>

      {loggedIn && (
        <section className="mt-14">
          <div className="mb-5 flex items-end justify-between gap-4">
            <div>
              <div className="font-mono text-[11px] uppercase tracking-[0.2em] text-white/50">Workspace</div>
              <h2 className="mt-1 text-2xl font-semibold">Your agents</h2>
            </div>
            <button onClick={() => openBuilder(null)} className="inline-flex items-center gap-1.5 rounded-lg border border-white/15 px-3 py-2 font-mono text-xs text-white/70 hover:border-white/50 hover:text-white">
              <Plus className="h-3.5 w-3.5" /> New agent
            </button>
          </div>
          {loading ? (
            <div className="flex items-center gap-2 font-mono text-sm text-white/50"><Loader2 className="h-4 w-4 animate-spin" /> Loading agents…</div>
          ) : agents.length === 0 ? (
            <button onClick={() => openBuilder(null)} className="w-full rounded-xl border border-dashed border-white/15 px-5 py-8 text-left transition-colors hover:border-white/50 hover:bg-white/[0.06]">
              <div className="font-medium">No agents yet</div>
              <div className="mt-1 font-mono text-xs text-white/55">Choose a preset below or create a custom agent.</div>
            </button>
          ) : (
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {agents.map(agent => (
                <Link key={agent.id} to={`/agents/${agent.id}`} className="group block rounded-xl border border-white/20 bg-white/[0.06] p-5 transition-colors hover:border-white/45 hover:bg-white/[0.08]">
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex h-9 w-9 items-center justify-center rounded-lg border border-white/20 bg-black/30"><Bot className="h-4 w-4 text-white/65" /></div>
                    <ArrowRight className="h-4 w-4 text-white/35 transition-transform group-hover:translate-x-0.5 group-hover:text-white/60" />
                  </div>
                  <div className="mt-5 font-semibold">{agent.name}</div>
                  <div className="mt-1 min-h-9 text-xs leading-5 text-white/45 line-clamp-2">{agent.description || 'Custom OpenPaths agent'}</div>
                  <div className="mt-4 flex flex-wrap gap-1.5">
                    <Tag>{agent.model}</Tag>
                    {(agent.config?.tools || []).slice(0, 2).map(tool => <Tag key={tool}>{toolLabels.get(tool) || tool}</Tag>)}
                  </div>
                </Link>
              ))}
            </div>
          )}
        </section>
      )}

      <section id="presets" className="mt-16 scroll-mt-24">
        <div className="mb-6 max-w-2xl">
          <div className="font-mono text-[11px] uppercase tracking-[0.2em] text-white/50">Ready to use</div>
          <h2 className="mt-1 text-2xl font-semibold">Preset agents</h2>
          <p className="mt-2 text-sm leading-6 text-white/45">Open any preset to see its instructions, tools, and example tasks. You can customize everything before creating it.</p>
        </div>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {presets.map(preset => {
            const Icon = presetIcons[preset.key] || Bot;
            return (
              <button key={preset.key} onClick={() => openBuilder(preset)} className="group flex min-h-72 flex-col rounded-xl border border-white/20 bg-white/[0.06] p-5 text-left transition-all hover:-translate-y-0.5 hover:border-white/45 hover:bg-white/[0.08]">
                <div className="flex items-start justify-between">
                  <div className="flex h-10 w-10 items-center justify-center rounded-lg border border-white/20 bg-black/30"><Icon className="h-5 w-5 text-white/70" /></div>
                  <ArrowRight className="h-4 w-4 text-white/35 transition-transform group-hover:translate-x-0.5 group-hover:text-white/60" />
                </div>
                <div className="mt-5 font-semibold">{preset.name}</div>
                <div className="mt-2 flex-1 text-xs leading-5 text-white/45">{preset.description}</div>
                {preset.example_prompts?.[0] && <div className="mt-4 border-l border-white/15 pl-3 text-[11px] leading-4 text-white/50">“{preset.example_prompts[0]}”</div>}
                <div className="mt-5 font-mono text-[11px] text-white/60">{loggedIn ? 'Preview & build' : 'See how it works'} →</div>
              </button>
            );
          })}
        </div>
      </section>

      {!loggedIn && (
        <section className="mt-14 flex flex-col items-start justify-between gap-5 rounded-xl border border-white/20 bg-white/[0.06] p-6 sm:flex-row sm:items-center">
          <div>
            <h2 className="text-lg font-semibold">Ready to build and run?</h2>
            <p className="mt-1 text-sm text-white/45">Sign in to save agents, connect private files, and view run history.</p>
          </div>
          <Link to="/account?next=%2Fagents" className="shrink-0 rounded-lg bg-white px-4 py-2.5 font-mono text-sm font-bold text-black hover:bg-white/90">Sign in to OpenPaths</Link>
        </section>
      )}

      {builderOpen && (
        <BuilderDialog
          loggedIn={loggedIn}
          preset={selectedPreset}
          draft={draft}
          setDraft={setDraft}
          tools={tools}
          files={files}
          addFiles={addFiles}
          removeFile={index => setFiles(current => current.filter((_, i) => i !== index))}
          toggleTool={toggleTool}
          creating={creating}
          progress={progress}
          error={err}
          onBuild={buildAgent}
          onClose={() => !creating && setBuilderOpen(false)}
        />
      )}
    </div>
  );
}

function BuilderDialog({ loggedIn, preset, draft, setDraft, tools, files, addFiles, removeFile, toggleTool, creating, progress, error, onBuild, onClose }: {
  loggedIn: boolean;
  preset: Preset | null;
  draft: Draft;
  setDraft: React.Dispatch<React.SetStateAction<Draft>>;
  tools: ToolSpec[];
  files: File[];
  addFiles: (files: FileList | null) => void;
  removeFile: (index: number) => void;
  toggleTool: (name: string) => void;
  creating: boolean;
  progress: string;
  error: string;
  onBuild: () => void;
  onClose: () => void;
}) {
  const Icon = preset ? presetIcons[preset.key] || Bot : Bot;
  const nameInputRef = useRef<HTMLInputElement>(null);
  const closeRef = useRef(onClose);
  closeRef.current = onClose;

  useEffect(() => {
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    nameInputRef.current?.focus();
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') closeRef.current();
    };
    window.addEventListener('keydown', onKeyDown);
    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener('keydown', onKeyDown);
    };
  }, []);

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-black/80 p-0 backdrop-blur-sm sm:items-center sm:p-5" role="dialog" aria-modal="true" aria-label={preset ? `Build ${preset.name}` : 'Build an agent'} onMouseDown={event => event.target === event.currentTarget && onClose()}>
      <div className="max-h-[94vh] w-full max-w-4xl overflow-y-auto rounded-t-2xl border border-white/20 bg-[#0a0a0a] shadow-2xl sm:rounded-2xl">
        <div className="sticky top-0 z-10 flex items-start justify-between gap-4 border-b border-white/20 bg-[#0a0a0a]/95 px-5 py-4 backdrop-blur sm:px-7">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg border border-white/20 bg-white/[0.07]"><Icon className="h-5 w-5 text-white/70" /></div>
            <div>
              <div className="font-mono text-[10px] uppercase tracking-[0.18em] text-white/50">{preset ? 'Preset agent' : 'Custom agent'}</div>
              <h2 className="text-lg font-semibold">{preset?.name || 'Build your own agent'}</h2>
            </div>
          </div>
          <button onClick={onClose} disabled={creating} aria-label="Close builder" className="rounded-lg p-2 text-white/55 hover:bg-white/10 hover:text-white disabled:opacity-30"><X className="h-5 w-5" /></button>
        </div>

        <div className="grid lg:grid-cols-[1fr_18rem]">
          <div className="space-y-6 p-5 sm:p-7">
            {preset && (
              <div className="rounded-xl border border-white/20 bg-white/[0.06] p-4">
                <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-white/50">What it does</div>
                <p className="mt-2 text-sm leading-6 text-white/60">{preset.description}</p>
                {!!preset.example_prompts?.length && (
                  <div className="mt-4 space-y-2">
                    <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-white/50">Try asking</div>
                    {preset.example_prompts.map(prompt => <div key={prompt} className="flex gap-2 text-xs leading-5 text-white/50"><ArrowRight className="mt-1 h-3 w-3 shrink-0" />{prompt}</div>)}
                  </div>
                )}
              </div>
            )}

            <div className="grid gap-4 sm:grid-cols-2">
              <Field label="Name">
                <input ref={nameInputRef} value={draft.name} onChange={event => setDraft(value => ({ ...value, name: event.target.value }))} placeholder="e.g. Customer research assistant" className="input" />
              </Field>
              <Field label="Model">
                <input value={draft.model} onChange={event => setDraft(value => ({ ...value, model: event.target.value }))} placeholder="auto" className="input font-mono" />
              </Field>
            </div>
            <Field label="Description">
              <input value={draft.description} onChange={event => setDraft(value => ({ ...value, description: event.target.value }))} placeholder="What should this agent help with?" className="input" />
            </Field>
            <Field label="Instructions" hint="Tell the agent its role, boundaries, and what a good answer looks like.">
              <textarea value={draft.systemPrompt} onChange={event => setDraft(value => ({ ...value, systemPrompt: event.target.value }))} rows={6} className="textarea resize-y font-mono text-xs leading-5" />
            </Field>

            <div>
              <div className="mb-2 text-xs font-medium text-white/70">Tools</div>
              <div className="grid gap-2 sm:grid-cols-2">
                {tools.map(tool => {
                  const enabled = draft.config.tools.includes(tool.name);
                  return (
                    <button key={tool.name} type="button" onClick={() => toggleTool(tool.name)} className={`rounded-lg border p-3 text-left transition-colors ${enabled ? 'border-white/30 bg-white/[0.11]' : 'border-white/20 bg-white/[0.05] hover:border-white/40'}`}>
                      <div className="flex items-center gap-2">
                        <span className={`flex h-4 w-4 items-center justify-center rounded border ${enabled ? 'border-white bg-white text-black' : 'border-white/25'}`}>{enabled && <Check className="h-3 w-3" />}</span>
                        <span className="font-mono text-xs text-white/75">{tool.label}</span>
                        {tool.experimental && <span className="rounded bg-amber-400/10 px-1 font-mono text-[9px] text-amber-200">beta</span>}
                      </div>
                      <div className="mt-2 text-[11px] leading-4 text-white/50">{tool.description}</div>
                    </button>
                  );
                })}
              </div>
            </div>
          </div>

          <aside className="border-t border-white/20 bg-white/[0.04] p-5 sm:p-7 lg:border-l lg:border-t-0">
            <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-white/50">Knowledge</div>
            <h3 className="mt-1 text-sm font-semibold">Add files now</h3>
            <p className="mt-2 text-xs leading-5 text-white/55">Files become structured Markdown and searchable chunks. Useful document images are compressed to WebP; small or oddly shaped images are skipped.</p>
            <label className="mt-4 flex cursor-pointer flex-col items-center rounded-xl border border-dashed border-white/15 px-4 py-6 text-center transition-colors hover:border-white/50 hover:bg-white/[0.06]">
              <Upload className="h-5 w-5 text-white/45" />
              <span className="mt-2 font-mono text-xs text-white/65">Choose files</span>
              <span className="mt-1 text-[10px] text-white/45">PDF, Word, PowerPoint, Excel, HTML, Markdown, CSV, JSON · 25 MB each</span>
              <input type="file" multiple className="hidden" accept=".pdf,.doc,.docx,.pptx,.xls,.xlsx,.html,.htm,.md,.markdown,.txt,.csv,.json,.epub,.ipynb,.zip,.rtf,.odt" onChange={event => { addFiles(event.target.files); event.target.value = ''; }} />
            </label>
            <div className="mt-3 space-y-2">
              {files.map((file, index) => (
                <div key={`${file.name}-${index}`} className="flex items-center gap-2 rounded-lg border border-white/20 bg-black/30 px-3 py-2">
                  <FileText className="h-3.5 w-3.5 shrink-0 text-white/50" />
                  <div className="min-w-0 flex-1">
                    <div className="truncate font-mono text-[10px] text-white/65">{file.name}</div>
                    <div className="text-[9px] text-white/45">{formatBytes(file.size)}</div>
                  </div>
                  <button onClick={() => removeFile(index)} aria-label={`Remove ${file.name}`} className="text-white/40 hover:text-white"><X className="h-3.5 w-3.5" /></button>
                </div>
              ))}
            </div>

            <div className="mt-6 border-t border-white/20 pt-5">
              {error && <div className="mb-3 text-xs leading-5 text-red-300">{error}</div>}
              {progress && <div className="mb-3 flex items-start gap-2 font-mono text-[10px] leading-4 text-white/45"><Loader2 className="mt-0.5 h-3 w-3 shrink-0 animate-spin" />{progress}</div>}
              {loggedIn ? (
                <button onClick={onBuild} disabled={!draft.name.trim() || creating} className="flex w-full items-center justify-center gap-2 rounded-lg bg-white px-4 py-2.5 font-mono text-sm font-bold text-black hover:bg-white/90 disabled:cursor-not-allowed disabled:opacity-40">
                  {creating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />} Create agent
                </button>
              ) : (
                <Link to={`/account?next=${encodeURIComponent(`/agents?preset=${preset?.key || 'custom'}`)}`} className="flex w-full items-center justify-center gap-2 rounded-lg bg-white px-4 py-2.5 font-mono text-sm font-bold text-black hover:bg-white/90">Sign in to build <ArrowRight className="h-4 w-4" /></Link>
              )}
              <p className="mt-2 text-center text-[10px] leading-4 text-white/40">You can change tools, instructions, and sources later.</p>
            </div>
          </aside>
        </div>
      </div>
    </div>
  );
}

function WorkflowStep({ icon: Icon, number, title, text }: { icon: React.ComponentType<{ className?: string }>; number: string; title: string; text: string }) {
  return (
    <div className="bg-black p-5 sm:p-6">
      <div className="flex items-center justify-between"><Icon className="h-4 w-4 text-white/45" /><span className="font-mono text-[10px] text-white/35">{number}</span></div>
      <div className="mt-5 text-sm font-medium">{title}</div>
      <div className="mt-1 text-xs leading-5 text-white/55">{text}</div>
    </div>
  );
}

function Tag({ children }: { children: React.ReactNode; key?: React.Key }) {
  return <span className="rounded-md border border-white/20 bg-white/[0.07] px-2 py-1 font-mono text-[9px] text-white/45">{children}</span>;
}

function Field({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <div className="mb-1.5 flex items-baseline justify-between gap-3">
        <span className="text-xs font-medium text-white/70">{label}</span>
        {hint && <span className="hidden text-[10px] text-white/45 sm:block">{hint}</span>}
      </div>
      {children}
    </label>
  );
}

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}
