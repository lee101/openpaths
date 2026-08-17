import React, { useEffect, useState } from 'react';
import { Loader2, Plus, Shield, Trash2, X } from 'lucide-react';
import {
  CustomFilter,
  Guardrail,
  GuardrailInput,
  PII_SLUGS,
  PROVIDER_OPTIONS,
  centsToUsd,
  createGuardrail,
  deleteGuardrail,
  emptyGuardrailInput,
  listGuardrails,
  setAssignments,
  summarize,
  updateGuardrail,
  usdToCents,
} from '../lib/guardrails';

type ApiKey = { id: string; name: string; key_prefix: string };

type Props = {
  apiKeys: ApiKey[];
};

const ACTIONS = ['block', 'redact', 'email'] as const;
const PI_ACTIONS = ['block', 'email', 'flag'] as const;

export function GuardrailsPanel({ apiKeys }: Props) {
  const [items, setItems] = useState<Guardrail[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [editing, setEditing] = useState<GuardrailInput | null>(null);
  const [editId, setEditId] = useState<string | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<string[]>([]);
  const [userDefault, setUserDefault] = useState(false);
  const [saving, setSaving] = useState(false);

  const reload = async () => {
    setLoading(true);
    setError('');
    try {
      setItems(await listGuardrails());
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void reload();
  }, []);

  const openNew = () => {
    setEditId(null);
    setEditing(emptyGuardrailInput());
    setSelectedKeys([]);
    setUserDefault(false);
  };

  const openEdit = (g: Guardrail) => {
    setEditId(g.id);
    setEditing({
      name: g.name,
      limit_cents: g.limit_cents,
      reset_interval: g.reset_interval || 'daily',
      budget_actions: g.budget_actions?.length ? g.budget_actions : ['block'],
      allowed_models: [...(g.allowed_models || [])],
      allowed_providers: [...(g.allowed_providers || [])],
      prompt_injection: {
        enabled: !!g.prompt_injection?.enabled,
        action: g.prompt_injection?.action || 'block',
        patterns: [...(g.prompt_injection?.patterns || [])],
      },
      sensitive_info: { filters: [...(g.sensitive_info?.filters || [])] },
      custom_filters: [...(g.custom_filters || [])],
    });
    const keys = (g.assignments || []).filter(a => a.target_type === 'api_key').map(a => a.target_id);
    setSelectedKeys(keys);
    setUserDefault((g.assignments || []).some(a => a.target_type === 'user'));
  };

  const save = async () => {
    if (!editing) return;
    setSaving(true);
    setError('');
    try {
      const payload = { ...editing };
      if (payload.limit_cents != null && payload.limit_cents <= 0) {
        payload.limit_cents = null;
        payload.reset_interval = null;
      }
      let g: Guardrail;
      if (editId) g = await updateGuardrail(editId, payload);
      else g = await createGuardrail(payload);
      g = await setAssignments(g.id, selectedKeys, userDefault);
      setEditing(null);
      setEditId(null);
      await reload();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  const remove = async (id: string) => {
    if (!confirm('Delete this guardrail?')) return;
    try {
      await deleteGuardrail(id);
      await reload();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Delete failed');
    }
  };

  const draft = editing;

  return (
    <div>
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Guardrails</h1>
          <p className="text-sm text-white/45 font-mono mt-2 max-w-xl">
            Apply spend limits, model/provider allowlists, prompt-injection detection, PII handling, and custom regex to API keys.
          </p>
        </div>
        <button
          type="button"
          onClick={openNew}
          className="inline-flex items-center gap-2 rounded-lg bg-white text-black px-4 py-2.5 text-sm font-mono font-bold hover:bg-white/90"
        >
          <Plus className="w-4 h-4" /> New guardrail
        </button>
      </div>

      {error && (
        <div className="mb-4 rounded-xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-300 font-mono">
          {error}
        </div>
      )}

      {loading ? (
        <div className="flex items-center gap-2 text-white/40 font-mono text-sm">
          <Loader2 className="w-4 h-4 animate-spin" /> Loading…
        </div>
      ) : items.length === 0 && !draft ? (
        <div className="rounded-2xl border border-dashed border-white/15 bg-white/[0.02] p-10 text-center">
          <Shield className="w-8 h-8 text-white/25 mx-auto mb-3" />
          <p className="text-white/50 font-mono text-sm">No guardrails yet. Create one to cap spend or scrub PII on selected keys.</p>
        </div>
      ) : (
        <div className="space-y-3 mb-8">
          {items.map(g => {
            const keyCount = (g.assignments || []).filter(a => a.target_type === 'api_key').length;
            const isDefault = (g.assignments || []).some(a => a.target_type === 'user');
            return (
              <div key={g.id} className="rounded-xl border border-white/10 bg-white/[0.02] p-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <div className="font-semibold tracking-tight">{g.name}</div>
                  <div className="mt-2 flex flex-wrap gap-2">
                    {summarize(g).map(t => (
                      <span key={t} className="text-[11px] font-mono uppercase tracking-wider px-2 py-1 rounded bg-white/5 text-white/55 border border-white/10">
                        {t}
                      </span>
                    ))}
                  </div>
                  <div className="mt-2 text-xs font-mono text-white/35">
                    {keyCount} key{keyCount === 1 ? '' : 's'}
                    {isDefault ? ' · account default' : ''}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => openEdit(g)}
                    className="rounded-lg border border-white/15 px-3 py-2 text-xs font-mono hover:bg-white/10"
                  >
                    Edit
                  </button>
                  <button
                    type="button"
                    onClick={() => void remove(g.id)}
                    className="rounded-lg border border-white/10 p-2 text-white/40 hover:text-red-300 hover:border-red-400/30"
                    aria-label="Delete"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {draft && (
        <div className="fixed inset-0 z-50 flex items-end sm:items-center justify-center bg-black/70 p-0 sm:p-6">
          <div className="w-full max-w-3xl max-h-[92vh] overflow-y-auto rounded-t-2xl sm:rounded-2xl border border-white/10 bg-[#0b0b0c] shadow-2xl">
            <div className="sticky top-0 z-10 flex items-center justify-between border-b border-white/10 bg-[#0b0b0c]/95 px-5 py-4 backdrop-blur">
              <div>
                <div className="text-xs font-mono uppercase tracking-[0.2em] text-white/35">Guardrail</div>
                <input
                  value={draft.name}
                  onChange={e => setEditing({ ...draft, name: e.target.value })}
                  className="mt-1 bg-transparent text-xl font-semibold tracking-tight outline-none border-b border-transparent focus:border-white/20 w-full"
                />
              </div>
              <button type="button" onClick={() => setEditing(null)} className="p-2 rounded-lg hover:bg-white/10" aria-label="Close">
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="p-5 space-y-8">
              <Section title="Budget Policies" subtitle="Credit limit that resets on a schedule.">
                <div className="grid sm:grid-cols-3 gap-3">
                  <label className="block text-xs font-mono text-white/45">
                    Limit (USD)
                    <input
                      type="number"
                      min={0}
                      step={1}
                      value={draft.limit_cents != null ? centsToUsd(draft.limit_cents) : ''}
                      placeholder="e.g. 10"
                      onChange={e => {
                        const v = e.target.value;
                        setEditing({
                          ...draft,
                          limit_cents: v === '' ? null : usdToCents(Number(v)),
                          reset_interval: draft.reset_interval || 'daily',
                        });
                      }}
                      className="mt-1 w-full rounded-lg bg-black border border-white/10 px-3 py-2 text-sm text-white"
                    />
                  </label>
                  <label className="block text-xs font-mono text-white/45">
                    Reset
                    <select
                      value={draft.reset_interval || 'daily'}
                      onChange={e => setEditing({ ...draft, reset_interval: e.target.value as any })}
                      className="mt-1 w-full rounded-lg bg-black border border-white/10 px-3 py-2 text-sm text-white"
                    >
                      <option value="daily">Daily</option>
                      <option value="weekly">Weekly</option>
                      <option value="monthly">Monthly</option>
                    </select>
                  </label>
                  <div className="text-xs font-mono text-white/45">
                    On exceed
                    <div className="mt-2 flex flex-wrap gap-2">
                      {(['block', 'email'] as const).map(a => (
                        <ToggleChip
                          key={a}
                          active={draft.budget_actions.includes(a)}
                          onClick={() => {
                            const has = draft.budget_actions.includes(a);
                            setEditing({
                              ...draft,
                              budget_actions: has
                                ? draft.budget_actions.filter(x => x !== a)
                                : [...draft.budget_actions, a],
                            });
                          }}
                          label={a}
                        />
                      ))}
                    </div>
                  </div>
                </div>
              </Section>

              <Section title="Model & Provider Access" subtitle="Empty allowlists mean unrestricted.">
                <TagInput
                  label="Allowed models (globs ok)"
                  values={draft.allowed_models}
                  placeholder="gpt-5*, anthropic/*, deepseek-v4-flash"
                  onChange={allowed_models => setEditing({ ...draft, allowed_models })}
                />
                <div className="mt-4">
                  <div className="text-xs font-mono text-white/45 mb-2">Allowed providers</div>
                  <div className="flex flex-wrap gap-2">
                    {PROVIDER_OPTIONS.map(p => (
                      <ToggleChip
                        key={p}
                        active={draft.allowed_providers.includes(p)}
                        label={p}
                        onClick={() => {
                          const has = draft.allowed_providers.includes(p);
                          setEditing({
                            ...draft,
                            allowed_providers: has
                              ? draft.allowed_providers.filter(x => x !== p)
                              : [...draft.allowed_providers, p],
                          });
                        }}
                      />
                    ))}
                  </div>
                </div>
              </Section>

              <Section title="Prompt Injection" subtitle="Detect jailbreak / override attempts.">
                <label className="flex items-center gap-2 text-sm font-mono text-white/70">
                  <input
                    type="checkbox"
                    checked={draft.prompt_injection.enabled}
                    onChange={e => setEditing({
                      ...draft,
                      prompt_injection: { ...draft.prompt_injection, enabled: e.target.checked },
                    })}
                  />
                  Enable builtin detection
                </label>
                <div className="mt-3 flex flex-wrap gap-2">
                  {PI_ACTIONS.map(a => (
                    <ToggleChip
                      key={a}
                      active={draft.prompt_injection.action === a}
                      label={a}
                      onClick={() => setEditing({
                        ...draft,
                        prompt_injection: { ...draft.prompt_injection, action: a },
                      })}
                    />
                  ))}
                </div>
                <TagInput
                  label="Extra patterns (regex)"
                  values={draft.prompt_injection.patterns || []}
                  placeholder="(?i)exfiltrate.*secrets"
                  onChange={patterns => setEditing({
                    ...draft,
                    prompt_injection: { ...draft.prompt_injection, patterns },
                  })}
                />
              </Section>

              <Section title="Sensitive Info Detection" subtitle="PII: block, redact, or email.">
                <div className="space-y-2">
                  {PII_SLUGS.map(slug => {
                    const existing = draft.sensitive_info.filters.find(f => f.slug === slug);
                    return (
                      <div key={slug} className="flex flex-wrap items-center gap-2 rounded-lg border border-white/10 px-3 py-2">
                        <label className="flex items-center gap-2 text-sm font-mono w-36">
                          <input
                            type="checkbox"
                            checked={!!existing}
                            onChange={e => {
                              const filters = e.target.checked
                                ? [...draft.sensitive_info.filters.filter(f => f.slug !== slug), { slug, action: 'redact' as const }]
                                : draft.sensitive_info.filters.filter(f => f.slug !== slug);
                              setEditing({ ...draft, sensitive_info: { filters } });
                            }}
                          />
                          {slug}
                        </label>
                        {existing && (
                          <div className="flex gap-1">
                            {ACTIONS.map(a => (
                              <ToggleChip
                                key={a}
                                active={existing.action === a}
                                label={a}
                                onClick={() => {
                                  const filters = draft.sensitive_info.filters.map(f =>
                                    f.slug === slug ? { ...f, action: a } : f,
                                  );
                                  setEditing({ ...draft, sensitive_info: { filters } });
                                }}
                              />
                            ))}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              </Section>

              <Section title="Custom Filters" subtitle="Your regex → block / redact / email.">
                <div className="space-y-3">
                  {draft.custom_filters.map((cf, i) => (
                    <div key={i} className="grid sm:grid-cols-[1fr_1fr_auto_auto] gap-2 items-center">
                      <input
                        value={cf.name}
                        placeholder="Name"
                        onChange={e => {
                          const custom_filters = draft.custom_filters.map((x, j) => j === i ? { ...x, name: e.target.value } : x);
                          setEditing({ ...draft, custom_filters });
                        }}
                        className="rounded-lg bg-black border border-white/10 px-3 py-2 text-sm"
                      />
                      <input
                        value={cf.pattern}
                        placeholder="Regex pattern"
                        onChange={e => {
                          const custom_filters = draft.custom_filters.map((x, j) => j === i ? { ...x, pattern: e.target.value } : x);
                          setEditing({ ...draft, custom_filters });
                        }}
                        className="rounded-lg bg-black border border-white/10 px-3 py-2 text-sm font-mono"
                      />
                      <select
                        value={cf.action}
                        onChange={e => {
                          const custom_filters = draft.custom_filters.map((x, j) => j === i ? { ...x, action: e.target.value as CustomFilter['action'] } : x);
                          setEditing({ ...draft, custom_filters });
                        }}
                        className="rounded-lg bg-black border border-white/10 px-2 py-2 text-sm"
                      >
                        {ACTIONS.map(a => <option key={a} value={a}>{a}</option>)}
                      </select>
                      <button
                        type="button"
                        onClick={() => setEditing({
                          ...draft,
                          custom_filters: draft.custom_filters.filter((_, j) => j !== i),
                        })}
                        className="p-2 text-white/40 hover:text-red-300"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  ))}
                  <button
                    type="button"
                    onClick={() => setEditing({
                      ...draft,
                      custom_filters: [...draft.custom_filters, { name: '', pattern: '', action: 'block' }],
                    })}
                    className="text-xs font-mono text-white/60 hover:text-white"
                  >
                    + Add pattern
                  </button>
                </div>
              </Section>

              <Section title="Apply to" subtitle="One guardrail per key. Account default covers unassigned keys.">
                <label className="flex items-center gap-2 text-sm font-mono text-white/70 mb-3">
                  <input type="checkbox" checked={userDefault} onChange={e => setUserDefault(e.target.checked)} />
                  Account default (all keys without their own guardrail)
                </label>
                {apiKeys.length === 0 ? (
                  <p className="text-xs font-mono text-white/35">No API keys yet.</p>
                ) : (
                  <div className="space-y-2">
                    {apiKeys.map(k => (
                      <label key={k.id} className="flex items-center gap-2 text-sm font-mono text-white/70">
                        <input
                          type="checkbox"
                          checked={selectedKeys.includes(k.id)}
                          onChange={e => {
                            setSelectedKeys(e.target.checked
                              ? [...selectedKeys, k.id]
                              : selectedKeys.filter(id => id !== k.id));
                          }}
                        />
                        {k.name || 'Key'} <span className="text-white/30">{k.key_prefix}…</span>
                      </label>
                    ))}
                  </div>
                )}
              </Section>
            </div>

            <div className="sticky bottom-0 flex justify-end gap-2 border-t border-white/10 bg-[#0b0b0c]/95 px-5 py-4 backdrop-blur">
              <button type="button" onClick={() => setEditing(null)} className="rounded-lg border border-white/15 px-4 py-2 text-sm font-mono hover:bg-white/10">
                Cancel
              </button>
              <button
                type="button"
                disabled={saving}
                onClick={() => void save()}
                className="rounded-lg bg-white text-black px-4 py-2 text-sm font-mono font-bold hover:bg-white/90 disabled:opacity-50"
              >
                {saving ? 'Saving…' : 'Save guardrail'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function Section({ title, subtitle, children }: { title: string; subtitle: string; children: React.ReactNode }) {
  return (
    <section>
      <h2 className="text-lg font-semibold tracking-tight">{title}</h2>
      <p className="text-xs font-mono text-white/40 mt-1 mb-4">{subtitle}</p>
      {children}
    </section>
  );
}

type ChipProps = { active: boolean; label: string; onClick: () => void };
function ToggleChip(props: ChipProps & React.Attributes) {
  const { active, label, onClick } = props;
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-md px-2.5 py-1 text-[11px] font-mono uppercase tracking-wider border transition-colors ${
        active ? 'bg-white text-black border-white' : 'bg-transparent text-white/50 border-white/15 hover:border-white/30'
      }`}
    >
      {label}
    </button>
  );
}

function TagInput({
  label,
  values,
  placeholder,
  onChange,
}: {
  label: string;
  values: string[];
  placeholder: string;
  onChange: (v: string[]) => void;
}) {
  const [draft, setDraft] = useState('');
  const add = () => {
    const v = draft.trim();
    if (!v) return;
    if (!values.includes(v)) onChange([...values, v]);
    setDraft('');
  };
  return (
    <div className="mt-3">
      <div className="text-xs font-mono text-white/45 mb-2">{label}</div>
      <div className="flex gap-2">
        <input
          value={draft}
          onChange={e => setDraft(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); add(); } }}
          placeholder={placeholder}
          className="flex-1 rounded-lg bg-black border border-white/10 px-3 py-2 text-sm font-mono"
        />
        <button type="button" onClick={add} className="rounded-lg border border-white/15 px-3 py-2 text-xs font-mono hover:bg-white/10">
          Add
        </button>
      </div>
      {values.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-2">
          {values.map(v => (
            <span key={v} className="inline-flex items-center gap-1 rounded bg-white/5 border border-white/10 px-2 py-1 text-xs font-mono">
              {v}
              <button type="button" onClick={() => onChange(values.filter(x => x !== v))} className="text-white/40 hover:text-white">
                <X className="w-3 h-3" />
              </button>
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
