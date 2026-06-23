import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { FileText, Image as ImageIcon, Loader2, Search, Sparkles, X } from 'lucide-react';
import { Seo } from '../components/Seo';
import {
  getSavedResponse,
  isAuthenticated,
  searchSavedResponses,
  type SavedResponse,
  type SearchResponse,
} from '../lib/usageSearch';

const PAGE_SIZE = 48;

function timeAgo(iso: string): string {
  const t = new Date(iso).getTime();
  if (!Number.isFinite(t)) return '';
  const s = Math.max(1, Math.floor((Date.now() - t) / 1000));
  if (s < 60) return `${s}s ago`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  return `${d}d ago`;
}

function truncate(s: string | undefined, n: number): string {
  if (!s) return '';
  return s.length > n ? s.slice(0, n).trimEnd() + '…' : s;
}

function ModeBadge({ mode }: { mode: SearchResponse['mode'] }) {
  if (mode === 'semantic') {
    return (
      <span className="inline-flex items-center gap-1 text-[11px] font-mono text-violet-200 bg-violet-500/10 border border-violet-400/20 px-2 py-0.5 rounded-full">
        <Sparkles className="w-3 h-3" /> semantic
      </span>
    );
  }
  if (mode === 'trigram') {
    return <span className="text-[11px] font-mono text-white/50 bg-white/5 border border-white/10 px-2 py-0.5 rounded-full">exact</span>;
  }
  return <span className="text-[11px] font-mono text-white/40 bg-white/5 border border-white/10 px-2 py-0.5 rounded-full">recent</span>;
}

function ImageCard({ item, onClick }: { item: SavedResponse; onClick: () => void; key?: React.Key }) {
  return (
    <button
      onClick={onClick}
      className="group relative aspect-square overflow-hidden rounded-xl border border-white/10 bg-white/5 hover:border-white/30 transition-colors"
    >
      <img
        src={item.thumb_url || item.image_url}
        alt={item.prompt}
        loading="lazy"
        className="h-full w-full object-cover transition-transform group-hover:scale-105"
      />
      <div className="absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/80 to-transparent p-2 opacity-0 group-hover:opacity-100 transition-opacity">
        <p className="text-[11px] text-white/90 line-clamp-2 text-left">{item.prompt}</p>
      </div>
    </button>
  );
}

function TextCard({ item, onClick }: { item: SavedResponse; onClick: () => void; key?: React.Key }) {
  return (
    <button
      onClick={onClick}
      className="w-full text-left rounded-xl border border-white/10 bg-white/5 hover:border-white/30 transition-colors p-4"
    >
      <div className="flex items-center gap-2 mb-2 text-[11px] font-mono text-white/40">
        <span className="text-white/70">{item.model}</span>
        {item.provider && <span>· {item.provider}</span>}
        <span className="ml-auto">{timeAgo(item.created_at)}</span>
      </div>
      <p className="text-sm text-white/90 line-clamp-2">{truncate(item.prompt, 220)}</p>
      {item.output && <p className="mt-2 text-xs text-white/50 line-clamp-2">{truncate(item.output, 260)}</p>}
    </button>
  );
}

function DetailModal({ id, onClose }: { id: string; onClose: () => void }) {
  const [item, setItem] = useState<SavedResponse | null>(null);
  const [similar, setSimilar] = useState<SavedResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeId, setActiveId] = useState(id);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    getSavedResponse(activeId)
      .then(res => {
        if (cancelled) return;
        setItem(res.item);
        setSimilar(res.similar || []);
      })
      .catch(() => {})
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, [activeId]);

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/80 p-4 sm:p-8" onClick={onClose}>
      <div
        className="relative w-full max-w-3xl rounded-2xl border border-white/10 bg-neutral-950 p-6 my-4"
        onClick={e => e.stopPropagation()}
      >
        <button onClick={onClose} className="absolute right-4 top-4 text-white/40 hover:text-white">
          <X className="w-5 h-5" />
        </button>
        {loading || !item ? (
          <div className="flex items-center justify-center py-16 text-white/40">
            <Loader2 className="w-6 h-6 animate-spin" />
          </div>
        ) : (
          <>
            <div className="flex items-center gap-2 mb-3 text-[11px] font-mono text-white/40">
              <span className="text-white/70">{item.model}</span>
              {item.provider && <span>· {item.provider}</span>}
              <span className="ml-auto">{new Date(item.created_at).toLocaleString()}</span>
            </div>
            {item.kind === 'image' ? (
              <img src={item.image_url} alt={item.prompt} className="w-full rounded-xl border border-white/10 mb-4" />
            ) : null}
            <div className="mb-1 text-xs uppercase tracking-wide text-white/40">Prompt</div>
            <p className="whitespace-pre-wrap text-sm text-white/90 mb-4">{item.prompt}</p>
            {item.kind === 'text' && item.input && item.input !== item.prompt && (
              <details className="mb-4">
                <summary className="cursor-pointer text-xs uppercase tracking-wide text-white/40">Full input</summary>
                <pre className="mt-2 whitespace-pre-wrap text-xs text-white/60 max-h-60 overflow-y-auto">{item.input}</pre>
              </details>
            )}
            {item.kind === 'text' && item.output && (
              <>
                <div className="mb-1 text-xs uppercase tracking-wide text-white/40">Output</div>
                <pre className="whitespace-pre-wrap text-sm text-white/80 mb-4 max-h-96 overflow-y-auto">{item.output}</pre>
              </>
            )}
            {item.kind === 'image' && item.output && (
              <p className="text-xs text-white/50 mb-4">Revised: {item.output}</p>
            )}

            {similar.length > 0 && (
              <div className="mt-6 border-t border-white/10 pt-4">
                <div className="mb-3 flex items-center gap-2 text-xs uppercase tracking-wide text-white/40">
                  <Sparkles className="w-3.5 h-3.5" /> Similar {item.kind === 'image' ? 'images' : 'prompts'}
                </div>
                {item.kind === 'image' ? (
                  <div className="grid grid-cols-4 gap-2 sm:grid-cols-6">
                    {similar.map(s => (
                      <button
                        key={s.id}
                        onClick={() => setActiveId(s.id)}
                        className="aspect-square overflow-hidden rounded-lg border border-white/10 hover:border-white/30"
                      >
                        <img src={s.thumb_url || s.image_url} alt={s.prompt} loading="lazy" className="h-full w-full object-cover" />
                      </button>
                    ))}
                  </div>
                ) : (
                  <div className="space-y-2">
                    {similar.map(s => (
                      <button
                        key={s.id}
                        onClick={() => setActiveId(s.id)}
                        className="block w-full text-left text-sm text-white/70 hover:text-white rounded-lg border border-white/10 hover:border-white/30 px-3 py-2"
                      >
                        {truncate(s.prompt, 140)}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

function UsageSearchPage({ kind }: { kind: 'text' | 'image' }) {
  const [authed] = useState(isAuthenticated());
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SavedResponse[]>([]);
  const [total, setTotal] = useState(0);
  const [mode, setMode] = useState<SearchResponse['mode']>('browse');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selected, setSelected] = useState<string | null>(null);
  const debounce = useRef<ReturnType<typeof setTimeout>>();

  const run = useCallback(
    (q: string) => {
      setLoading(true);
      setError('');
      searchSavedResponses(kind, q, PAGE_SIZE, 0)
        .then(res => {
          setResults(res.results || []);
          setTotal(res.total || 0);
          setMode(res.mode);
        })
        .catch(() => setError('Could not load your saved responses.'))
        .finally(() => setLoading(false));
    },
    [kind],
  );

  useEffect(() => {
    if (!authed) {
      setLoading(false);
      return;
    }
    run('');
  }, [authed, run]);

  useEffect(() => {
    if (!authed) return;
    clearTimeout(debounce.current);
    debounce.current = setTimeout(() => run(query), 280);
    return () => clearTimeout(debounce.current);
  }, [query, authed, run]);

  const isImage = kind === 'image';
  const title = isImage ? 'Image history' : 'Prompt history';
  const Icon = isImage ? ImageIcon : FileText;

  return (
    <div className="mx-auto max-w-6xl px-4 py-10 sm:py-14">
      <Seo
        title={`${title} · OpenPaths`}
        description={`Search your saved ${isImage ? 'generated images' : 'prompts and outputs'} semantically.`}
      />
      <div className="mb-6">
        <div className="flex items-center gap-3">
          <Icon className="h-6 w-6 text-white/70" />
          <h1 className="text-2xl font-semibold text-white">{title}</h1>
        </div>
        <p className="mt-2 text-sm text-white/50">
          Your private, searchable history of saved {isImage ? 'image generations' : 'text generations'}. Semantic search finds
          similar {isImage ? 'images' : 'prompts and outputs'} even without exact keyword matches.
        </p>
        <div className="mt-4 flex gap-2 text-sm">
          <Link
            to="/usage/prompts"
            className={`rounded-full px-4 py-1.5 transition-colors ${!isImage ? 'bg-white/10 text-white' : 'text-white/50 hover:bg-white/5'}`}
          >
            Prompts
          </Link>
          <Link
            to="/usage/images"
            className={`rounded-full px-4 py-1.5 transition-colors ${isImage ? 'bg-white/10 text-white' : 'text-white/50 hover:bg-white/5'}`}
          >
            Images
          </Link>
          <Link to="/account" className="ml-auto rounded-full px-4 py-1.5 text-white/50 hover:bg-white/5 transition-colors">
            Settings
          </Link>
        </div>
      </div>

      {!authed ? (
        <div className="rounded-2xl border border-white/10 bg-white/5 p-10 text-center">
          <p className="text-white/70">Sign in to search your saved responses.</p>
          <Link to="/account" className="mt-4 inline-block rounded-xl border border-white/20 px-4 py-2 text-sm text-white hover:bg-white/10">
            Go to account
          </Link>
        </div>
      ) : (
        <>
          <div className="relative mb-6">
            <Search className="pointer-events-none absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-white/30" />
            <input
              value={query}
              onChange={e => setQuery(e.target.value)}
              placeholder={`Search your ${isImage ? 'images' : 'prompts and outputs'}…`}
              className="w-full rounded-2xl border border-white/10 bg-white/5 py-3 pl-12 pr-4 text-white placeholder-white/30 focus:border-white/30 focus:outline-none"
            />
          </div>

          <div className="mb-4 flex items-center gap-3 text-sm text-white/40">
            {loading ? (
              <span className="inline-flex items-center gap-2">
                <Loader2 className="h-4 w-4 animate-spin" /> Loading…
              </span>
            ) : (
              <>
                <span>
                  {results.length} of {total} saved
                </span>
                <ModeBadge mode={mode} />
              </>
            )}
          </div>

          {error && <p className="text-sm text-rose-300">{error}</p>}

          {!loading && total === 0 && !error && (
            <div className="rounded-2xl border border-white/10 bg-white/5 p-10 text-center">
              <p className="text-white/70">No saved {isImage ? 'images' : 'prompts'} yet.</p>
              <p className="mt-2 text-sm text-white/40">
                Turn on response saving in your account settings, then generate {isImage ? 'an image' : 'some text'}.
              </p>
              <Link to="/account" className="mt-4 inline-block rounded-xl border border-white/20 px-4 py-2 text-sm text-white hover:bg-white/10">
                Enable in settings
              </Link>
            </div>
          )}

          {results.length > 0 &&
            (isImage ? (
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
                {results.map(item => (
                  <ImageCard key={item.id} item={item} onClick={() => setSelected(item.id)} />
                ))}
              </div>
            ) : (
              <div className="space-y-3">
                {results.map(item => (
                  <TextCard key={item.id} item={item} onClick={() => setSelected(item.id)} />
                ))}
              </div>
            ))}
        </>
      )}

      {selected && <DetailModal id={selected} onClose={() => setSelected(null)} />}
    </div>
  );
}

export function UsagePrompts() {
  return <UsageSearchPage kind="text" />;
}

export function UsageImages() {
  return <UsageSearchPage kind="image" />;
}
