import React, { useState } from 'react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import { ArrowLeft, Loader2 } from 'lucide-react';
import { Seo } from '../components/Seo';
import { SkillDetailPane } from '../components/skills/SkillDetailPane';
import { getSkill } from '../data/skills';

// SkillDetail is the full /skills/:slug page: the shared detail pane plus a
// version-history list that pins any version into the pane.
export function SkillDetail() {
  const { slug = '' } = useParams();
  const decoded = slug;
  const [params, setParams] = useSearchParams();
  const pinned = params.get('version') ?? '';
  const [versions, setVersions] = useState<string[]>([]);
  const [current, setCurrent] = useState('');
  const [versionsLoading, setVersionsLoading] = useState(true);

  React.useEffect(() => {
    let cancelled = false;
    setVersionsLoading(true);
    getSkill(decoded)
      .then(d => {
        if (cancelled) return;
        setVersions(d.versions);
        setCurrent(d.skill.currentVersion);
      })
      .catch(() => { if (!cancelled) setVersions([]); })
      .finally(() => { if (!cancelled) setVersionsLoading(false); });
  }, [decoded]);

  const pin = (v: string) => {
    const next = new URLSearchParams(params);
    if (v) next.set('version', v);
    else next.delete('version');
    setParams(next);
  };

  return (
    <>
      <Seo title={`${decoded} skill | OpenPaths`} description={`Versioned agent skill ${decoded}: copy the markdown, pin a version, browse files.`} path={`/skills/${slug}`} />
      <div className="min-h-screen bg-black">
        <div className="mx-auto max-w-5xl px-6 py-10">
          <div className="mb-6 flex items-center gap-2 font-mono text-xs text-white/45">
            <Link to="/skills" className="inline-flex items-center gap-1 hover:text-white"><ArrowLeft className="h-3.5 w-3.5" /> Skills</Link>
            <span>/</span>
            <span className="truncate text-white/70">{decoded}</span>
          </div>

          <SkillDetailPane key={`${decoded}@${pinned}`} slug={decoded} initialVersion={pinned} />

          <section className="mt-10">
            <h2 className="mb-3 font-mono text-xs uppercase tracking-wide text-white/45">Version history</h2>
            {versionsLoading ? (
              <div className="flex items-center gap-2 text-white/55"><Loader2 className="h-4 w-4 animate-spin" /> Loading versions…</div>
            ) : versions.length === 0 ? (
              <p className="font-mono text-sm text-white/40">No version history available.</p>
            ) : (
              <ol className="space-y-1.5">
                {versions.map(v => {
                  const active = (pinned || current) === v;
                  return (
                    <li key={v}>
                      <button
                        onClick={() => pin(v === current ? '' : v)}
                        className={`flex w-full items-center justify-between rounded border px-3 py-2 font-mono text-xs ${active ? 'border-white/50 bg-white/[0.08] text-white' : 'border-white/15 text-white/60 hover:border-white/40 hover:text-white'}`}
                      >
                        <span>v{v}</span>
                        {v === current && <span className="rounded border border-emerald-400/30 px-1.5 py-0.5 text-[10px] text-emerald-300">current</span>}
                      </button>
                    </li>
                  );
                })}
              </ol>
            )}
          </section>
        </div>
      </div>
    </>
  );
}
