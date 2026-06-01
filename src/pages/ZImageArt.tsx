import React, { useEffect, useMemo, useState } from 'react';
import { ArrowRight, Database, Image as ImageIcon, Loader2, Search, Sparkles } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Seo } from '../components/Seo';
import { type ArtTagFacet, type ZImageArtItem } from '../data/zimageArt';
import {
  artPromptPlaygroundHref,
  fetchArtList,
  fetchArtTags,
  lexicalZImageSearch,
  loadZImageArtIndex,
  searchZImageArt,
} from '../lib/zimageArt';

const PAGE_SIZE = 48;

const ASPECTS: { key: string; label: string }[] = [
  { key: '', label: 'All' },
  { key: 'square', label: 'Square' },
  { key: 'portrait', label: 'Portrait' },
  { key: 'wide', label: 'Wide' },
];

export function ZImageArt() {
  const [query, setQuery] = useState('');
  const [aspect, setAspect] = useState('');
  const [items, setItems] = useState<ZImageArtItem[]>([]);
  const [total, setTotal] = useState(0);
  const [aspectCounts, setAspectCounts] = useState<Record<string, number>>({});
  const [tags, setTags] = useState<ArtTagFacet[]>([]);
  const [loading, setLoading] = useState(true);
  const [searching, setSearching] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [searchMode, setSearchMode] = useState(false);

  // Load tag facets once.
  useEffect(() => {
    let cancelled = false;
    fetchArtTags(48).then(t => {
      if (!cancelled) setTags(t);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  // Main load: browse (no query) or search, keyed on query + aspect.
  useEffect(() => {
    const q = query.trim();
    let cancelled = false;
    const run = async () => {
      if (q) {
        setSearching(true);
        const results = await searchZImageArt(q, PAGE_SIZE, { aspect });
        if (cancelled) return;
        if (results && results.length) {
          setItems(results);
          setSearchMode(true);
        } else {
          // Fall back to the bundled/manifest index with lexical scoring.
          const index = await loadZImageArtIndex();
          if (cancelled) return;
          setItems(lexicalZImageSearch(index.items, q, PAGE_SIZE));
          setSearchMode(true);
        }
        setSearching(false);
        setLoading(false);
        return;
      }
      setLoading(true);
      const list = await fetchArtList({ aspect, limit: PAGE_SIZE, offset: 0 });
      if (cancelled) return;
      if (list && list.results.length) {
        setItems(list.results);
        setTotal(list.total);
        setAspectCounts(list.aspects);
        setSearchMode(false);
      } else {
        const index = await loadZImageArtIndex();
        if (cancelled) return;
        setItems(index.items.slice(0, PAGE_SIZE));
        setTotal(index.manifest?.count || index.items.length);
        setSearchMode(false);
      }
      setLoading(false);
    };
    const timer = window.setTimeout(run, q ? 200 : 0);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [query, aspect]);

  const loadMore = async () => {
    if (searchMode || loadingMore) return;
    setLoadingMore(true);
    const list = await fetchArtList({ aspect, limit: PAGE_SIZE, offset: items.length });
    if (list) {
      setItems(prev => [...prev, ...list.results]);
      setTotal(list.total);
    }
    setLoadingMore(false);
  };

  const featured = items[0];
  const hasMore = !searchMode && items.length < total;
  const totalLabel = useMemo(() => (total ? total.toLocaleString('en-US') : items.length.toLocaleString('en-US')), [total, items.length]);

  return (
    <>
      <Seo
        title="AI Art Prompt Search | OpenPaths"
        description="Search a huge library of AI-generated art by prompt, style, scene, mood, and aspect ratio (square, portrait, wide). Browse subtags and try any prompt in the OpenPaths playground."
        path="/art"
      />
      <div className="min-h-screen bg-black">
        <section className="border-b border-white/10 bg-white/[0.02]">
          <div className="mx-auto grid max-w-7xl gap-8 px-6 py-10 lg:grid-cols-[1fr_420px] lg:items-end">
            <div>
              <div className="mb-4 inline-flex items-center gap-2 rounded border border-white/10 bg-black px-3 py-2 font-mono text-xs uppercase tracking-[0.18em] text-white/50">
                <Sparkles className="h-4 w-4" />
                AI art prompt index
              </div>
              <h1 className="max-w-4xl text-4xl font-bold tracking-tight md:text-6xl">Search AI art by prompt</h1>
              <p className="mt-4 max-w-3xl text-base leading-relaxed text-white/62">
                A growing index of generated art — square, portrait, and wide — searchable by visual intent. Open any prompt straight in the image playground.
              </p>
            </div>
            <div className="grid grid-cols-3 gap-3">
              <Stat label="Indexed" value={totalLabel} />
              <Stat label="Square" value={(aspectCounts.square || 0).toLocaleString('en-US')} />
              <Stat label="Tall/Wide" value={((aspectCounts.portrait || 0) + (aspectCounts.wide || 0)).toLocaleString('en-US')} />
            </div>
          </div>
        </section>

        <main className="mx-auto max-w-7xl px-6 py-8">
          <div className="mb-5 grid gap-4 lg:grid-cols-[1fr_auto] lg:items-center">
            <label className="relative block">
              <Search className="pointer-events-none absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-white/35" />
              <input
                value={query}
                onChange={event => setQuery(event.target.value)}
                placeholder="Search prompts, style, scene, character, mood..."
                className="h-14 w-full rounded border border-white/10 bg-white/[0.03] pl-12 pr-12 text-base text-white outline-none transition-colors placeholder:text-white/30 focus:border-white/35"
              />
              {searching && <Loader2 className="absolute right-4 top-1/2 h-5 w-5 -translate-y-1/2 animate-spin text-white/45" />}
            </label>
            <Link
              to="/playground?model=zimage"
              className="inline-flex h-14 items-center justify-center gap-2 rounded border border-white bg-white px-5 font-mono text-sm font-bold text-black hover:bg-white/90"
            >
              Open image playground <ArrowRight className="h-4 w-4" />
            </Link>
          </div>

          {/* Aspect ratio filter */}
          <div className="mb-4 flex flex-wrap items-center gap-2">
            {ASPECTS.map(a => (
              <button
                key={a.key || 'all'}
                type="button"
                onClick={() => setAspect(a.key)}
                className={`rounded border px-3 py-1.5 font-mono text-xs transition-colors ${aspect === a.key ? 'border-white bg-white text-black' : 'border-white/10 text-white/55 hover:border-white/30'}`}
              >
                {a.label}
                {a.key && aspectCounts[a.key] ? <span className="ml-1 opacity-60">{aspectCounts[a.key].toLocaleString('en-US')}</span> : null}
              </button>
            ))}
          </div>

          {/* Subtag pills (crawlable links to tag pages) */}
          {tags.length > 0 && (
            <div className="mb-8 flex flex-wrap items-center gap-2 text-sm text-white/45">
              <Database className="h-4 w-4" />
              <span className="mr-1">Popular tags</span>
              {tags.slice(0, 24).map(t => (
                <Link
                  key={t.tag}
                  to={`/art/tag/${encodeURIComponent(t.tag)}`}
                  className="rounded border border-white/10 px-2 py-1 font-mono text-xs text-white/55 hover:border-white/25 hover:text-white"
                >
                  {t.tag}
                </Link>
              ))}
            </div>
          )}

          {featured && !searchMode && (
            <FeaturedCard item={featured} />
          )}

          {loading ? (
            <div className="flex min-h-80 items-center justify-center rounded border border-white/10 bg-white/[0.02] text-white/45">
              <Loader2 className="mr-2 h-5 w-5 animate-spin" />
              Loading art index
            </div>
          ) : items.length === 0 ? (
            <div className="flex min-h-60 items-center justify-center rounded border border-white/10 bg-white/[0.02] text-white/45">
              No images found{query ? ` for “${query}”` : ''}.
            </div>
          ) : (
            <>
              <div className="grid auto-rows-fr gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
                {items.map(item => (
                  <ArtCard key={item.id || item.slug} item={item} />
                ))}
              </div>
              {hasMore && (
                <div className="mt-8 flex justify-center">
                  <button
                    type="button"
                    onClick={loadMore}
                    disabled={loadingMore}
                    className="inline-flex items-center gap-2 rounded border border-white/15 px-6 py-3 font-mono text-sm text-white/75 hover:border-white/35 hover:text-white disabled:opacity-50"
                  >
                    {loadingMore ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
                    Load more
                  </button>
                </div>
              )}
            </>
          )}
        </main>
      </div>
    </>
  );
}

function aspectClass(aspect?: string): string {
  switch (aspect) {
    case 'portrait':
      return 'aspect-[3/4]';
    case 'wide':
      return 'aspect-[4/3]';
    default:
      return 'aspect-square';
  }
}

function ArtCard({ item }: { item: ZImageArtItem; key?: React.Key }) {
  const dims = item.width && item.height ? `${item.width}×${item.height}` : item.aspect;
  return (
    <article className="group flex flex-col overflow-hidden rounded border border-white/10 bg-white/[0.02]">
      <Link to={`/art/i/${encodeURIComponent(item.slug)}`} className="block overflow-hidden">
        {item.thumbUrl || item.imageUrl ? (
          <img
            src={item.thumbUrl || item.imageUrl}
            alt={item.prompt}
            loading="lazy"
            className={`${aspectClass(item.aspect)} w-full bg-white/[0.03] object-cover transition-transform duration-300 group-hover:scale-[1.03]`}
          />
        ) : (
          <PromptPlaceholder prompt={item.prompt} />
        )}
      </Link>
      <div className="flex flex-1 flex-col p-4">
        <div className="mb-2 flex items-center justify-between gap-3 font-mono text-[11px] uppercase tracking-[0.16em] text-white/35">
          <span>{item.model || 'zimage'}</span>
          {dims ? <span>{dims}</span> : null}
        </div>
        <Link to={`/art/i/${encodeURIComponent(item.slug)}`} className="line-clamp-3 min-h-[3.75rem] text-sm leading-relaxed text-white/64 hover:text-white">
          {item.prompt}
        </Link>
        <Link to={artPromptPlaygroundHref(item)} className="mt-3 inline-flex items-center gap-2 font-mono text-xs font-bold uppercase tracking-[0.14em] text-white/70 hover:text-white">
          Try prompt <ArrowRight className="h-3.5 w-3.5" />
        </Link>
      </div>
    </article>
  );
}

function FeaturedCard({ item }: { item: ZImageArtItem }) {
  return (
    <section className="mb-8 grid overflow-hidden rounded border border-white/10 bg-white/[0.02] lg:grid-cols-[minmax(0,1fr)_420px]">
      {item.imageUrl ? (
        <Link to={`/art/i/${encodeURIComponent(item.slug)}`}>
          <img src={item.imageUrl} alt={item.prompt} className="aspect-[16/10] h-full w-full bg-white/[0.03] object-cover" />
        </Link>
      ) : (
        <PromptPlaceholder prompt={item.prompt} large />
      )}
      <div className="flex flex-col justify-between gap-6 p-5">
        <div>
          <div className="mb-3 flex items-center gap-2 font-mono text-xs uppercase tracking-[0.18em] text-white/40">
            <ImageIcon className="h-4 w-4" />
            Latest in the index
          </div>
          <h2 className="text-xl font-semibold tracking-tight">{item.title || 'Generated artwork'}</h2>
          <p className="mt-3 line-clamp-5 text-sm leading-relaxed text-white/62">{item.prompt}</p>
        </div>
        <div className="flex flex-wrap gap-3">
          <Link to={`/art/i/${encodeURIComponent(item.slug)}`} className="inline-flex items-center justify-center gap-2 rounded border border-white/15 px-4 py-3 font-mono text-sm text-white/75 hover:border-white/35 hover:text-white">
            View details <ArrowRight className="h-4 w-4" />
          </Link>
          <Link to={artPromptPlaygroundHref(item)} className="inline-flex items-center justify-center gap-2 rounded bg-white px-4 py-3 font-mono text-sm font-bold text-black hover:bg-white/90">
            Try this prompt
          </Link>
        </div>
      </div>
    </section>
  );
}

function PromptPlaceholder({ prompt, large = false }: { prompt: string; large?: boolean }) {
  return (
    <div className={`flex ${large ? 'aspect-[16/10] min-h-80' : 'aspect-square'} w-full flex-col justify-between bg-[radial-gradient(circle_at_top_left,rgba(255,255,255,0.14),transparent_34%),linear-gradient(135deg,rgba(255,255,255,0.08),rgba(255,255,255,0.02))] p-5`}>
      <div className="font-mono text-[11px] uppercase tracking-[0.18em] text-white/35">Prompt</div>
      <p className="line-clamp-5 text-sm leading-relaxed text-white/58">{prompt}</p>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded border border-white/10 bg-black/40 p-3">
      <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-white/35">{label}</div>
      <div className="mt-1 truncate text-lg font-semibold tracking-tight">{value}</div>
    </div>
  );
}
