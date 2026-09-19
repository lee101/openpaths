import React, { useEffect, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, Loader2 } from 'lucide-react';
import { Seo } from '../components/Seo';
import { createSkill, deleteSkill, getSkill, isLoggedIn, updateSkill } from '../data/skills';

type Mode = 'new' | 'edit';

// SkillEdit is the owner form behind /skills/new and /skills/:slug/edit. It
// uses the stored op key for writes; 403 (not owner) and 409 (slug taken) come
// back from jsonOrThrow and are surfaced inline. Saving an edit creates a new
// immutable version server-side (version field optional — blank auto-bumps).
export function SkillEdit({ mode }: { mode: Mode }) {
  const { slug = '' } = useParams();
  const decoded = slug;
  const navigate = useNavigate();
  const [name, setName] = useState('');
  const [slugField, setSlugField] = useState('');
  const [description, setDescription] = useState('');
  const [body, setBody] = useState('');
  const [category, setCategory] = useState('');
  const [tags, setTags] = useState('');
  const [setupScript, setSetupScript] = useState('');
  const [setupPrompt, setSetupPrompt] = useState('');
  const [skillPrompt, setSkillPrompt] = useState('');
  const [version, setVersion] = useState('');
  const [loading, setLoading] = useState(mode === 'edit');
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (mode !== 'edit') return;
    let cancelled = false;
    getSkill(decoded)
      .then(d => {
        if (cancelled) return;
        setName(d.skill.name);
        setSlugField(d.skill.slug);
        setDescription(d.skill.description);
        setBody(d.skill.body ?? '');
        setCategory(d.skill.category ?? '');
        setTags((d.skill.tags ?? []).join(', '));
        setSetupScript(d.skill.setupScript ?? '');
        setSetupPrompt(d.skill.setupPrompt ?? '');
        setSkillPrompt(d.skill.skillPrompt ?? '');
      })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : 'Could not load skill'); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [mode, decoded]);

  if (!isLoggedIn()) {
    return (
      <div className="min-h-screen bg-black">
        <div className="mx-auto max-w-2xl px-6 py-16 text-center">
          <h1 className="text-2xl font-bold">Sign in to {mode === 'new' ? 'publish' : 'edit'} a skill</h1>
          <p className="mt-3 text-sm leading-relaxed text-white/60">
            Skill writes need your OpenPaths API key. Save one under Account → API keys first.
          </p>
          <Link to="/account/apikeys" className="mt-6 inline-block rounded border border-white bg-white px-5 py-2.5 font-mono text-sm font-bold text-black hover:bg-white/90">
            Go to API keys
          </Link>
        </div>
      </div>
    );
  }

  const save = async () => {
    setSaving(true);
    setError('');
    const input = {
      name: name.trim(),
      description: description.trim(),
      body,
      category: category.trim() || undefined,
      tags: tags.split(',').map(t => t.trim()).filter(Boolean),
      setupScript: setupScript || undefined,
      setupPrompt: setupPrompt || undefined,
      skillPrompt: skillPrompt || undefined,
      version: version.trim() || undefined,
    };
    try {
      if (mode === 'new') {
        const created = await createSkill({ ...input, slug: slugField.trim() || undefined });
        navigate(`/skills/${encodeURIComponent(created.slug)}`);
      } else {
        await updateSkill(decoded, input);
        navigate(`/skills/${encodeURIComponent(decoded)}`);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (!window.confirm(`Delete ${decoded}? This cannot be undone.`)) return;
    setDeleting(true);
    setError('');
    try {
      await deleteSkill(decoded);
      navigate('/skills');
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Delete failed');
    } finally {
      setDeleting(false);
    }
  };

  if (loading) {
    return <div className="flex min-h-screen items-center gap-2 bg-black px-6 text-white/55"><Loader2 className="h-5 w-5 animate-spin" /> Loading skill…</div>;
  }

  return (
    <>
      <Seo title={mode === 'new' ? 'Publish a skill | OpenPaths' : `Edit ${decoded} | OpenPaths`} description="Owner skill editor." path={mode === 'new' ? '/skills/new' : `/skills/${slug}/edit`} />
      <div className="min-h-screen bg-black">
        <div className="mx-auto max-w-3xl px-6 py-10">
          <Link to={mode === 'new' ? '/skills' : `/skills/${encodeURIComponent(decoded)}`} className="inline-flex items-center gap-1 font-mono text-xs text-white/45 hover:text-white">
            <ArrowLeft className="h-3.5 w-3.5" /> {mode === 'new' ? 'Skills' : decoded}
          </Link>
          <h1 className="mt-3 text-3xl font-bold tracking-tight md:text-4xl">{mode === 'new' ? 'Publish a skill' : `Edit ${decoded}`}</h1>
          <p className="mt-2 text-sm leading-relaxed text-white/60">
            {mode === 'new'
              ? 'Creates v1.0.0 under your account — you become the owner.'
              : 'Saving creates a new immutable version (blank version auto-bumps the patch).'}
          </p>

          <div className="mt-8 space-y-5">
            {mode === 'new' && (
              <Field label="Slug (optional — derived from name when blank)">
                <input value={slugField} onChange={e => setSlugField(e.target.value)} placeholder="my-skill" className={inputCls} />
              </Field>
            )}
            <Field label="Name">
              <input value={name} onChange={e => setName(e.target.value)} placeholder="My skill" className={inputCls} />
            </Field>
            <Field label="Description">
              <textarea value={description} onChange={e => setDescription(e.target.value)} rows={3} placeholder="What the skill does..." className={inputCls} />
            </Field>
            <div className="grid gap-5 sm:grid-cols-2">
              <Field label="Category">
                <input value={category} onChange={e => setCategory(e.target.value)} placeholder="productivity" className={inputCls} />
              </Field>
              <Field label="Tags (comma separated)">
                <input value={tags} onChange={e => setTags(e.target.value)} placeholder="review, git" className={inputCls} />
              </Field>
            </div>
            <Field label="Body (markdown)">
              <textarea value={body} onChange={e => setBody(e.target.value)} rows={10} spellCheck={false} className={`${inputCls} font-mono text-sm`} />
            </Field>
            <Field label="Setup script">
              <textarea value={setupScript} onChange={e => setSetupScript(e.target.value)} rows={4} spellCheck={false} placeholder="#!/bin/sh..." className={`${inputCls} font-mono text-sm`} />
            </Field>
            <Field label="Setup prompt">
              <textarea value={setupPrompt} onChange={e => setSetupPrompt(e.target.value)} rows={3} className={inputCls} />
            </Field>
            <Field label="Skill prompt">
              <textarea value={skillPrompt} onChange={e => setSkillPrompt(e.target.value)} rows={3} className={inputCls} />
            </Field>
            {mode === 'edit' && (
              <Field label="New version (optional — blank auto-bumps patch, semver like 1.1.0)">
                <input value={version} onChange={e => setVersion(e.target.value)} placeholder="1.0.1" className={inputCls} />
              </Field>
            )}

            {error && <p className="rounded border border-red-400/40 bg-red-400/10 px-3 py-2 font-mono text-xs text-red-200">{error}</p>}

            <div className="flex flex-wrap gap-3">
              <button onClick={save} disabled={saving || !name.trim()} className="rounded border border-white bg-white px-5 py-2.5 font-mono text-sm font-bold text-black hover:bg-white/90 disabled:opacity-50">
                {saving ? 'Saving…' : mode === 'new' ? 'Publish v1.0.0' : 'Save new version'}
              </button>
              {mode === 'edit' && (
                <button onClick={remove} disabled={deleting} className="rounded border border-red-400/40 px-5 py-2.5 font-mono text-sm text-red-300 hover:bg-red-400/10 disabled:opacity-50">
                  {deleting ? 'Deleting…' : 'Delete skill'}
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}

const inputCls = 'w-full rounded border border-white/20 bg-white/[0.06] px-3 py-2.5 text-base text-white outline-none transition-colors placeholder:text-white/35 focus:border-white/60';

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block">
      <span className="mb-1.5 block font-mono text-xs uppercase tracking-wide text-white/50">{label}</span>
      {children}
    </label>
  );
}
