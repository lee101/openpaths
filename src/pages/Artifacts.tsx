import React, { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Seo } from '../components/Seo';
import { Boxes, Plus, Search, Globe, Lock, Eye, Loader2 } from 'lucide-react';
import {
  Artifact,
  isLoggedIn,
  listMine,
  listPublic,
  searchMine,
  searchPublic,
} from '../lib/artifacts';

type Tab = 'public' | 'mine';

export function Artifacts() {
  const navigate = useNavigate();
  const [tab, setTab] = useState<Tab>(isLoggedIn() ? 'mine' : 'public');
  const [items, setItems] = useState<Artifact[]>([]);
  const [query, setQuery] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const loggedIn = isLoggedIn();

  const load = useMemo(() => async (t: Tab, q: string) => {
    setLoading(true);
    setError('');
    try {
      let res: Artifact[];
      if (q.trim()) {
        res = t === 'mine' ? await searchMine(q) : await searchPublic(q);
      } else {
        res = t === 'mine' ? await listMine() : await listPublic();
      }
      setItems(res);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load');
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load(tab, '');
    setQuery('');
  }, [tab, load]);

  const onSearch = (e: React.FormEvent) => {
    e.preventDefault();
    void load(tab, query);
  };

  return (
    <>
      <Seo
        title="Artifacts Gallery | OpenPaths"
        description="Build, host, and share interactive AI artifacts. An IDE-style canvas where any OpenPaths model can write, edit, and publish web apps to a public gallery."
        path="/artifacts"
      />
      <div className="max-w-7xl mx-auto px-6 py-12">
        <div className="flex flex-col gap-5 md:flex-row md:items-end md:justify-between mb-8">
          <div>
            <div className="flex items-center gap-2 text-cyan-300 font-mono text-xs uppercase tracking-[0.24em] mb-3">
              <Boxes className="w-4 h-4" />
              Build &amp; publish
            </div>
            <h1 className="text-4xl md:text-5xl font-semibold tracking-tight">Artifacts</h1>
            <p className="mt-4 text-white/60 text-lg max-w-2xl font-light">
              An IDE on the web. Prompt any model on OpenPaths to write, edit, and ship self-contained apps —
              preview live, then publish to the gallery or keep them private.
            </p>
          </div>
          <button
            onClick={() => navigate('/artifacts/new')}
            className="inline-flex items-center justify-center gap-2 rounded-lg bg-cyan-300 px-4 py-3 font-mono text-sm font-semibold text-black transition-colors hover:bg-cyan-200"
          >
            <Plus className="h-4 w-4" /> New artifact
          </button>
        </div>

        <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="inline-flex rounded-lg border border-white/10 bg-white/[0.03] p-1">
            <TabButton active={tab === 'public'} onClick={() => setTab('public')} icon={<Globe className="h-3.5 w-3.5" />} label="Public" />
            <TabButton active={tab === 'mine'} onClick={() => setTab('mine')} icon={<Lock className="h-3.5 w-3.5" />} label="Mine" />
          </div>
          <form onSubmit={onSearch} className="relative flex-1 sm:max-w-md">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-white/40" />
            <input
              value={query}
              onChange={e => setQuery(e.target.value)}
              placeholder={tab === 'public' ? 'Search public artifacts ($1 / 1k)' : 'Search your artifacts'}
              className="w-full rounded-lg border border-white/10 bg-black py-2.5 pl-10 pr-3 font-mono text-sm text-white outline-none transition-colors focus:border-cyan-300"
            />
          </form>
        </div>

        {tab === 'mine' && !loggedIn && (
          <div className="rounded-lg border border-white/10 bg-white/[0.02] p-8 text-center text-white/60">
            <p>
              <Link to="/account" className="text-cyan-300 underline underline-offset-4">Sign in</Link> to see and build your own artifacts.
            </p>
          </div>
        )}

        {error && (
          <div className="mb-6 rounded-lg border border-red-400/20 bg-red-500/10 p-4 font-mono text-sm text-red-200">{error}</div>
        )}

        {loading ? (
          <div className="flex items-center justify-center py-24 text-white/40">
            <Loader2 className="h-5 w-5 animate-spin" />
          </div>
        ) : items.length === 0 ? (
          (tab === 'mine' && !loggedIn) ? null : (
            <div className="rounded-lg border border-dashed border-white/10 py-24 text-center">
              <Boxes className="mx-auto mb-4 h-8 w-8 text-white/20" />
              <p className="text-white/50 font-mono text-sm">
                {tab === 'mine' ? 'No artifacts yet — create your first one.' : 'No public artifacts found.'}
              </p>
            </div>
          )
        ) : (
          <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
            {items.map((a: Artifact) => (
              <ArtifactCard key={a.id} artifact={a} />
            ))}
          </div>
        )}
      </div>
    </>
  );
}

function TabButton({ active, onClick, icon, label }: { active: boolean; onClick: () => void; icon: React.ReactNode; label: string }) {
  return (
    <button
      onClick={onClick}
      className={`inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 font-mono text-xs transition-colors ${
        active ? 'bg-white/10 text-white' : 'text-white/50 hover:text-white'
      }`}
    >
      {icon} {label}
    </button>
  );
}

function ArtifactCard({ artifact }: { artifact: Artifact; key?: React.Key }) {
  return (
    <Link
      to={`/artifacts/${artifact.slug || artifact.id}`}
      className="group flex flex-col overflow-hidden rounded-xl border border-white/10 bg-white/[0.02] transition-colors hover:border-white/25 hover:bg-white/[0.04]"
    >
      <div className="aspect-[16/9] w-full overflow-hidden bg-gradient-to-br from-white/[0.06] to-transparent">
        {artifact.image_url ? (
          <img src={artifact.image_url} alt="" loading="lazy" className="h-full w-full object-cover" />
        ) : (
          <div className="flex h-full w-full items-center justify-center text-white/15">
            <Boxes className="h-10 w-10" />
          </div>
        )}
      </div>
      <div className="flex flex-1 flex-col p-5">
        <div className="mb-2 flex items-center gap-2">
          <h3 className="flex-1 truncate text-lg font-semibold tracking-tight">{artifact.title}</h3>
          {artifact.visibility !== 'public' && <Lock className="h-3.5 w-3.5 flex-none text-white/30" />}
        </div>
        <p className="line-clamp-2 flex-1 text-sm font-light text-white/55">{artifact.description || 'No description.'}</p>
        <div className="mt-4 flex items-center justify-between font-mono text-[11px] text-white/35">
          <span className="inline-flex items-center gap-1"><Eye className="h-3 w-3" /> {artifact.view_count}</span>
          {artifact.tags?.length > 0 && <span className="truncate">{artifact.tags.slice(0, 3).join(' · ')}</span>}
        </div>
      </div>
    </Link>
  );
}
