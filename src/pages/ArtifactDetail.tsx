import React, { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, GitFork, Eye, Code2, Loader2, Pencil } from 'lucide-react';
import { Seo } from '../components/Seo';
import { getStoredUser } from '../lib/session';
import {
  Artifact,
  buildPreviewDoc,
  createArtifact,
  getPublic,
  isLoggedIn,
} from '../lib/artifacts';

export function ArtifactDetail() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const [artifact, setArtifact] = useState<Artifact | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [view, setView] = useState<'preview' | 'code'>('preview');
  const [activeFile, setActiveFile] = useState('');
  const [forking, setForking] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    (async () => {
      try {
        const a = await getPublic(id);
        if (cancelled) return;
        setArtifact(a);
        setActiveFile(a.files?.[0]?.path || '');
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Not found');
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [id]);

  const previewDoc = useMemo(
    () => (artifact?.files ? buildPreviewDoc(artifact.files, artifact.entry) : ''),
    [artifact],
  );
  const file = artifact?.files?.find(f => f.path === activeFile);
  const currentUser = getStoredUser<{ id?: string }>();
  const isOwner = !!artifact && !!currentUser?.id && artifact.user_id === currentUser.id;

  const fork = async () => {
    if (!artifact) return;
    if (!isLoggedIn()) { navigate('/account'); return; }
    setForking(true);
    try {
      const copy = await createArtifact({
        title: `${artifact.title} (fork)`,
        description: artifact.description,
        image_url: artifact.image_url,
        files: artifact.files || [],
        entry: artifact.entry,
        visibility: 'private',
        tags: artifact.tags || [],
        fork_of: artifact.id,
      });
      navigate(`/artifacts/${copy.id}/edit`);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Fork failed');
      setForking(false);
    }
  };

  if (loading) return <div className="flex h-[60vh] items-center justify-center text-white/40"><Loader2 className="h-6 w-6 animate-spin" /></div>;
  if (error || !artifact) {
    return (
      <div className="mx-auto max-w-3xl px-6 py-20 text-center">
        <Seo title="Artifact not found | OpenPaths" description="This artifact could not be found." path={`/artifacts/${id}`} />
        <h1 className="text-3xl font-semibold">Artifact not found</h1>
        <p className="mt-3 text-white/50">{error || 'It may be private or deleted.'}</p>
        <Link to="/artifacts" className="mt-6 inline-block font-mono text-sm text-cyan-300 underline underline-offset-4">Back to gallery</Link>
      </div>
    );
  }

  return (
    <>
      <Seo
        title={`${artifact.title} | OpenPaths Artifacts`}
        description={artifact.description || `An interactive artifact built on OpenPaths.`}
        path={`/artifacts/${artifact.slug || artifact.id}`}
        image={artifact.image_url || undefined}
        jsonLd={{
          '@context': 'https://schema.org',
          '@type': 'CreativeWork',
          name: artifact.title,
          description: artifact.description,
          url: `https://openpaths.io/artifacts/${artifact.slug || artifact.id}`,
        }}
      />
      <div className="mx-auto max-w-6xl px-6 py-10">
        <Link to="/artifacts" className="mb-6 inline-flex items-center gap-1.5 font-mono text-xs text-white/50 hover:text-white">
          <ArrowLeft className="h-3.5 w-3.5" /> Artifacts
        </Link>

        <div className="mb-6 flex flex-wrap items-end justify-between gap-4">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight md:text-4xl">{artifact.title}</h1>
            {artifact.description && <p className="mt-3 max-w-2xl text-white/60">{artifact.description}</p>}
            <div className="mt-3 flex items-center gap-4 font-mono text-xs text-white/35">
              <span className="inline-flex items-center gap-1"><Eye className="h-3 w-3" /> {artifact.view_count} views</span>
              {artifact.tags?.length > 0 && <span>{artifact.tags.join(' · ')}</span>}
            </div>
          </div>
          <div className="flex items-center gap-2">
            {isOwner && (
              <Link to={`/artifacts/${artifact.id}/edit`} className="inline-flex items-center gap-1.5 rounded-lg border border-white/15 bg-white/[0.04] px-3 py-2 font-mono text-xs text-white hover:bg-white/[0.08]">
                <Pencil className="h-3.5 w-3.5" /> Edit
              </Link>
            )}
            <button onClick={fork} disabled={forking} className="inline-flex items-center gap-1.5 rounded-lg bg-cyan-300 px-3 py-2 font-mono text-xs font-semibold text-black hover:bg-cyan-200 disabled:opacity-50">
              {forking ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <GitFork className="h-3.5 w-3.5" />} Fork &amp; edit
            </button>
          </div>
        </div>

        <div className="mb-3 inline-flex rounded-lg border border-white/10 bg-white/[0.03] p-1">
          <button onClick={() => setView('preview')} className={`inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 font-mono text-xs ${view === 'preview' ? 'bg-white/10 text-white' : 'text-white/50 hover:text-white'}`}><Eye className="h-3.5 w-3.5" /> Preview</button>
          <button onClick={() => setView('code')} className={`inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 font-mono text-xs ${view === 'code' ? 'bg-white/10 text-white' : 'text-white/50 hover:text-white'}`}><Code2 className="h-3.5 w-3.5" /> Code</button>
        </div>

        {view === 'preview' ? (
          <div className="overflow-hidden rounded-lg border border-white/10 bg-white">
            <iframe title="preview" srcDoc={previewDoc} sandbox="allow-scripts allow-modals allow-forms allow-popups" className="h-[640px] w-full bg-white" />
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-[200px_1fr]">
            <ul className="space-y-0.5 rounded-lg border border-white/10 bg-white/[0.02] p-2">
              {artifact.files?.map(f => (
                <li key={f.path}>
                  <button onClick={() => setActiveFile(f.path)} className={`w-full truncate rounded px-2 py-1 text-left font-mono text-xs ${activeFile === f.path ? 'bg-white/10 text-white' : 'text-white/55 hover:bg-white/[0.04]'}`}>{f.path}</button>
                </li>
              ))}
            </ul>
            <pre className="max-h-[640px] overflow-auto rounded-lg border border-white/10 bg-black p-4 font-mono text-[13px] leading-relaxed text-white/80">
              <code>{file?.content || ''}</code>
            </pre>
          </div>
        )}
      </div>
    </>
  );
}
