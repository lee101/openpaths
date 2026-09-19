import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { Check, Copy, Loader2, Pencil } from 'lucide-react';
import { getStoredUser } from '../../lib/session';
import {
  fetchSkillFileText,
  getSkill,
  langFromPath,
  skillFileRawUrl,
  updateSkill,
  type SkillFileManifest,
  type SkillGetResult,
} from '../../data/skills';
import { SkillFileEditor } from './SkillFileEditor';
import { SkillFileViewer, SkillMarkdown, previewKindForSkillFile } from './SkillFileViewer';

type Tab = 'description' | 'body' | 'setup' | 'prompts' | 'files';

const TABS: { id: Tab; label: string }[] = [
  { id: 'description', label: 'Description' },
  { id: 'body', label: 'Body' },
  { id: 'setup', label: 'Setup script' },
  { id: 'prompts', label: 'Prompts' },
  { id: 'files', label: 'Files' },
];

// SkillDetailPane fetches one skill (optionally at a pinned version) and shows
// its description/body/setup/prompts tabs plus the file browser. Shared by the
// /skills marketplace side pane and the /skills/:slug full page.
export function SkillDetailPane({ slug, initialVersion, linkTitle }: {
  slug: string;
  initialVersion?: string;
  linkTitle?: boolean;
  key?: React.Key;
}) {
  const [version, setVersion] = useState(initialVersion ?? '');
  const [detail, setDetail] = useState<SkillGetResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tab, setTab] = useState<Tab>('description');
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    setVersion(initialVersion ?? '');
  }, [slug, initialVersion]);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const d = await getSkill(slug, version || undefined);
      setDetail(d);
    } catch (e) {
      setDetail(null);
      setError(e instanceof Error ? e.message : 'Skill not found');
    } finally {
      setLoading(false);
    }
  }, [slug, version]);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      setLoading(true);
      setError('');
      try {
        const d = await getSkill(slug, version || undefined);
        if (!cancelled) setDetail(d);
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Skill not found');
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [slug, version]);

  const copyMarkdown = async () => {
    if (!detail || typeof navigator === 'undefined' || !navigator.clipboard) return;
    try {
      await navigator.clipboard.writeText(detail.markdown || detail.skill.body || '');
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch { /* clipboard unavailable */ }
  };

  const versions = useMemo(() => {
    if (!detail) return [];
    const list = [...detail.versions];
    if (detail.skill.currentVersion && !list.includes(detail.skill.currentVersion)) list.push(detail.skill.currentVersion);
    return list;
  }, [detail]);

  if (loading) {
    return <div className="flex items-center gap-2 py-12 text-white/55"><Loader2 className="h-5 w-5 animate-spin" /> Loading skill…</div>;
  }
  if (error || !detail) {
    return <div className="rounded border border-white/20 bg-white/[0.04] px-4 py-10 text-center font-mono text-sm text-white/55">{error || 'Skill not found.'}</div>;
  }

  const { skill } = detail;
  const pinned = version && version !== skill.currentVersion;

  return (
    <section className="min-w-0 space-y-4">
      <div>
        <div className="flex flex-wrap items-center gap-2 font-mono text-[11px] uppercase tracking-wide text-white/45">
          {skill.source && <span className="rounded border border-white/20 px-2 py-1">{skill.source}</span>}
          {skill.category && <span className="rounded border border-white/20 px-2 py-1">{skill.category}</span>}
          <span className="rounded border border-white/20 px-2 py-1">v{skill.currentVersion}</span>
        </div>
        {linkTitle ? (
          <Link to={`/skills/${encodeURIComponent(skill.slug)}`} className="mt-2 block text-2xl font-bold tracking-tight hover:underline">{skill.name}</Link>
        ) : (
          <h2 className="mt-2 text-2xl font-bold tracking-tight">{skill.name}</h2>
        )}
        <p className="mt-1 font-mono text-xs text-white/40">{skill.slug}</p>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <label className="font-mono text-xs text-white/45">
          Version{' '}
          <select
            value={version}
            onChange={e => setVersion(e.target.value)}
            className="rounded border border-white/20 bg-black px-2 py-1 font-mono text-xs text-white/80"
          >
            <option value="">current ({skill.currentVersion})</option>
            {versions.filter(v => v !== skill.currentVersion).map(v => (
              <option key={v} value={v}>{v}</option>
            ))}
          </select>
        </label>
        <button
          onClick={copyMarkdown}
          className="inline-flex items-center gap-1.5 rounded border border-white/20 px-2.5 py-1.5 font-mono text-xs text-white/70 hover:border-white/50 hover:text-white"
        >
          {copied ? <><Check className="h-3.5 w-3.5" /> Copied</> : <><Copy className="h-3.5 w-3.5" /> Copy markdown</>}
        </button>
        <OwnerEditLink skill={skill} />
      </div>

      {pinned && (
        <div className="rounded border border-amber-400/30 bg-amber-400/10 px-3 py-2 font-mono text-xs text-amber-200">
          Viewing pinned version {version} — current is {skill.currentVersion}.{' '}
          <button onClick={() => setVersion('')} className="underline hover:text-amber-100">Back to current</button>
        </div>
      )}

      <div className="flex flex-wrap gap-1.5 border-b border-white/15 pb-2">
        {TABS.map(t => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`rounded px-3 py-1.5 font-mono text-xs ${tab === t.id ? 'bg-white text-black' : 'border border-white/20 text-white/60 hover:border-white/50 hover:text-white'}`}
          >
            {t.label}{t.id === 'files' && detail.files.length > 0 && ` (${detail.files.length})`}
          </button>
        ))}
      </div>

      {tab === 'description' && (
        <p className="text-sm leading-relaxed text-white/75">{skill.description || 'No description.'}</p>
      )}
      {tab === 'body' && (
        skill.body
          ? <div className="rounded border border-white/15 bg-black/40 p-4"><SkillMarkdown source={skill.body} /></div>
          : <EmptyNote label="No body for this version." />
      )}
      {tab === 'setup' && (
        skill.setupScript
          ? <pre className="overflow-x-auto rounded border border-white/15 bg-black/60 p-4 font-mono text-xs leading-relaxed text-white/85">{skill.setupScript}</pre>
          : <EmptyNote label="No setup script for this version." />
      )}
      {tab === 'prompts' && (
        <div className="space-y-4">
          <div>
            <h3 className="mb-1 font-mono text-xs uppercase tracking-wide text-white/45">Setup prompt</h3>
            {skill.setupPrompt ? <p className="whitespace-pre-wrap text-sm leading-relaxed text-white/75">{skill.setupPrompt}</p> : <EmptyNote label="None." />}
          </div>
          <div>
            <h3 className="mb-1 font-mono text-xs uppercase tracking-wide text-white/45">Skill prompt</h3>
            {skill.skillPrompt ? <p className="whitespace-pre-wrap text-sm leading-relaxed text-white/75">{skill.skillPrompt}</p> : <EmptyNote label="None." />}
          </div>
        </div>
      )}
      {tab === 'files' && (
        <SkillFileBrowser slug={skill.slug} version={version || undefined} files={detail.files} ownerId={skill.ownerId} onSaved={load} />
      )}

      {skill.tags.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {skill.tags.map(t => (
            <span key={t} className="rounded bg-white/[0.07] px-2 py-1 font-mono text-xs text-white/45">#{t}</span>
          ))}
        </div>
      )}
    </section>
  );
}

function OwnerEditLink({ skill }: { skill: { slug: string; ownerId: string } }) {
  const user = getStoredUser<{ id?: string }>();
  if (!user?.id || !skill.ownerId || user.id !== skill.ownerId) return null;
  return (
    <Link
      to={`/skills/${encodeURIComponent(skill.slug)}/edit`}
      className="inline-flex items-center gap-1.5 rounded border border-white/20 px-2.5 py-1.5 font-mono text-xs text-white/70 hover:border-white/50 hover:text-white"
    >
      <Pencil className="h-3.5 w-3.5" /> Edit
    </Link>
  );
}

function EmptyNote({ label }: { label: string }) {
  return <p className="font-mono text-sm text-white/40">{label}</p>;
}

function SkillFileBrowser({ slug, version, files, ownerId, onSaved }: {
  slug: string;
  version?: string;
  files: SkillFileManifest[];
  ownerId: string;
  onSaved: () => void;
}) {
  const [active, setActive] = useState(files[0]?.path ?? '');
  const [text, setText] = useState<string | null>(null);
  const [textLoading, setTextLoading] = useState(false);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState('');
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState('');

  const user = getStoredUser<{ id?: string }>();
  const isOwner = !!user?.id && !!ownerId && user.id === ownerId;
  const manifest = files.find(f => f.path === active);
  const kind = manifest ? previewKindForSkillFile(manifest.path, manifest.mime) : 'none';
  const isTextKind = kind === 'text' || kind === 'markdown';

  useEffect(() => {
    if (!files.some(f => f.path === active)) setActive(files[0]?.path ?? '');
  }, [files, active]);

  useEffect(() => {
    setEditing(false);
    setSaveError('');
    if (!manifest || !isTextKind) { setText(null); return; }
    let cancelled = false;
    setTextLoading(true);
    fetchSkillFileText(slug, manifest.path, version)
      .then(t => { if (!cancelled) { setText(t); setDraft(t); } })
      .catch(() => { if (!cancelled) setText(null); })
      .finally(() => { if (!cancelled) setTextLoading(false); });
    return () => { cancelled = true; };
  }, [slug, manifest?.path, manifest?.mime, version]); // eslint-disable-line react-hooks/exhaustive-deps

  const save = async () => {
    if (!manifest) return;
    setSaving(true);
    setSaveError('');
    try {
      await updateSkill(slug, { files: [{ path: manifest.path, content: draft }] });
      setEditing(false);
      onSaved();
    } catch (e) {
      setSaveError(e instanceof Error ? e.message : 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  if (files.length === 0) return <EmptyNote label="No files attached to this version." />;
  const src = manifest ? skillFileRawUrl(slug, manifest.path, version) : undefined;

  return (
    <div className="grid gap-3 lg:grid-cols-[220px_minmax(0,1fr)]">
      <div className="space-y-1">
        {files.map(f => (
          <button
            key={f.path}
            onClick={() => setActive(f.path)}
            className={`block w-full truncate rounded px-2.5 py-1.5 text-left font-mono text-xs ${f.path === active ? 'bg-white text-black' : 'text-white/60 hover:bg-white/[0.07] hover:text-white'}`}
            title={f.path}
          >
            {f.path}
          </button>
        ))}
      </div>
      <div className="min-w-0 rounded border border-white/15">
        {manifest && (
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-white/15 px-3 py-2">
            <span className="truncate font-mono text-xs text-white/60">{manifest.path} <span className="text-white/35">· {(manifest.size / 1024).toFixed(1)} KB</span></span>
            <div className="flex items-center gap-2">
              {isOwner && isTextKind && !editing && text != null && (
                <button onClick={() => { setDraft(text); setEditing(true); }} className="inline-flex items-center gap-1 rounded border border-white/20 px-2 py-1 font-mono text-xs text-white/70 hover:border-white/50 hover:text-white">
                  <Pencil className="h-3 w-3" /> Edit
                </button>
              )}
              {src && <a href={src} download={manifest.path} className="rounded border border-white/20 px-2 py-1 font-mono text-xs text-white/70 hover:border-white/50 hover:text-white">Download</a>}
            </div>
          </div>
        )}
        {textLoading
          ? <div className="flex items-center gap-2 p-6 text-white/55"><Loader2 className="h-4 w-4 animate-spin" /> Loading file…</div>
          : editing && manifest
            ? (
              <div>
                <SkillFileEditor value={draft} onChange={setDraft} lang={langFromPath(manifest.path)} ariaLabel={`Edit ${manifest.path}`} className="h-[420px] bg-black" />
                <div className="flex items-center gap-2 border-t border-white/15 px-3 py-2">
                  <button onClick={save} disabled={saving} className="rounded bg-white px-3 py-1.5 font-mono text-xs font-bold text-black disabled:opacity-50">
                    {saving ? 'Saving…' : 'Save as new version'}
                  </button>
                  <button onClick={() => setEditing(false)} className="rounded border border-white/20 px-3 py-1.5 font-mono text-xs text-white/70 hover:border-white/50 hover:text-white">Cancel</button>
                  {saveError && <span className="font-mono text-xs text-red-300">{saveError}</span>}
                </div>
              </div>
            )
            : manifest && (isTextKind
              ? (text != null
                ? <SkillFileViewer path={manifest.path} mime={manifest.mime} text={text} />
                : <EmptyNote label="Could not load file text." />)
              : <SkillFileViewer path={manifest.path} mime={manifest.mime} src={src} />)}
      </div>
    </div>
  );
}
