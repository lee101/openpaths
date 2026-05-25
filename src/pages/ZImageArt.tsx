import React, { useEffect, useMemo, useState } from 'react';
import { ArrowRight, Database, Image as ImageIcon, Loader2, Search, Sparkles } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Seo } from '../components/Seo';
import { ZIMAGE_ART_MANIFEST_URL, type ZImageArtItem, type ZImageArtManifest } from '../data/zimageArt';
import { artPromptPlaygroundHref, lexicalZImageSearch, loadZImageArtIndex, searchZImageArt } from '../lib/zimageArt';

const DEFAULT_QUERY = 'anime lantern city at night';

export function ZImageArt() {
  const [items, setItems] = useState<ZImageArtItem[]>([]);
  const [manifest, setManifest] = useState<ZImageArtManifest | null>(null);
  const [source, setSource] = useState<'remote' | 'fallback'>('fallback');
  const [query, setQuery] = useState(DEFAULT_QUERY);
  const [loading, setLoading] = useState(true);
  const [searching, setSearching] = useState(false);
  const [results, setResults] = useState<ZImageArtItem[]>([]);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    loadZImageArtIndex().then(data => {
      if (cancelled) return;
      setItems(data.items);
      setResults(data.items.slice(0, 48));
      setManifest(data.manifest);
      setSource(data.source);
    }).finally(() => {
      if (!cancelled) setLoading(false);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    const q = query.trim();
    if (!q) {
      setResults(items.slice(0, 48));
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(async () => {
      setSearching(true);
      const semantic = await searchZImageArt(q, 48);
      if (cancelled) return;
      setResults(semantic?.length ? semantic : lexicalZImageSearch(items, q, 48));
      setSearching(false);
    }, 180);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [items, query]);

  const featured = results[0] || items[0];
  const generatedCount = manifest?.generatedCount || items.length;
  const totalCount = manifest?.count || items.length;
  const subtitle = source === 'remote'
    ? `${totalCount.toLocaleString('en-US')} indexed prompts from OpenPaths static storage.`
    : `Remote index not loaded yet. Showing the bundled seed image while ${ZIMAGE_ART_MANIFEST_URL} is populated.`;

  const tags = useMemo(() => {
    const counts = new Map<string, number>();
    for (const item of items) {
      for (const tag of item.tags || []) counts.set(tag, (counts.get(tag) || 0) + 1);
    }
    return [...counts.entries()]
      .sort((a, b) => b[1] - a[1])
      .slice(0, 12)
      .map(([tag]) => tag);
  }, [items]);

  return (
    <>
      <Seo
        title="ZImage Prompt Search | OpenPaths"
        description="Browse and search a large ZImage generated-art prompt index, then try any prompt against OpenPaths image generation models."
        path="/art"
      />
      <div className="min-h-screen bg-black">
        <section className="border-b border-white/10 bg-white/[0.02]">
          <div className="mx-auto grid max-w-7xl gap-8 px-6 py-10 lg:grid-cols-[1fr_420px] lg:items-end">
            <div>
              <div className="mb-4 inline-flex items-center gap-2 rounded border border-white/10 bg-black px-3 py-2 font-mono text-xs uppercase tracking-[0.18em] text-white/50">
                <Sparkles className="h-4 w-4" />
                ZImage prompt index
              </div>
              <h1 className="max-w-4xl text-4xl font-bold tracking-tight md:text-6xl">Search generated art by prompt</h1>
              <p className="mt-4 max-w-3xl text-base leading-relaxed text-white/62">
                Browse ZImage outputs stored on OpenPaths static storage, search by visual intent, and send any prompt straight into the image playground.
              </p>
            </div>
            <div className="grid grid-cols-3 gap-3">
              <Stat label="Indexed" value={totalCount.toLocaleString('en-US')} />
              <Stat label="Generated" value={generatedCount.toLocaleString('en-US')} />
              <Stat label="Model" value="zimage" />
            </div>
          </div>
        </section>

        <main className="mx-auto max-w-7xl px-6 py-8">
          <div className="mb-6 grid gap-4 lg:grid-cols-[1fr_auto] lg:items-center">
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
              Open ZImage playground <ArrowRight className="h-4 w-4" />
            </Link>
          </div>

          <div className="mb-8 flex flex-wrap items-center gap-2 text-sm text-white/45">
            <Database className="h-4 w-4" />
            <span>{subtitle}</span>
            {tags.map(tag => (
              <button key={tag} type="button" onClick={() => setQuery(tag)} className="rounded border border-white/10 px-2 py-1 font-mono text-xs text-white/55 hover:border-white/25 hover:text-white">
                {tag}
              </button>
            ))}
          </div>

          {featured && (
            <section className="mb-8 grid overflow-hidden rounded border border-white/10 bg-white/[0.02] lg:grid-cols-[minmax(0,1fr)_420px]">
              {featured.imageUrl ? (
                <img src={featured.imageUrl} alt={featured.prompt} className="aspect-[16/10] h-full w-full bg-white/[0.03] object-cover" />
              ) : (
                <PromptPlaceholder prompt={featured.prompt} large />
              )}
              <div className="flex flex-col justify-between gap-6 p-5">
                <div>
                  <div className="mb-3 flex items-center gap-2 font-mono text-xs uppercase tracking-[0.18em] text-white/40">
                    <ImageIcon className="h-4 w-4" />
                    Top match
                  </div>
                  <h2 className="text-xl font-semibold tracking-tight">{featured.title || 'Generated ZImage artwork'}</h2>
                  <p className="mt-3 text-sm leading-relaxed text-white/62">{featured.prompt}</p>
                </div>
                <Link to={artPromptPlaygroundHref(featured)} className="inline-flex items-center justify-center gap-2 rounded border border-white/15 px-4 py-3 font-mono text-sm text-white/75 hover:border-white/35 hover:text-white">
                  Try this prompt <ArrowRight className="h-4 w-4" />
                </Link>
              </div>
            </section>
          )}

          {loading ? (
            <div className="flex min-h-80 items-center justify-center rounded border border-white/10 bg-white/[0.02] text-white/45">
              <Loader2 className="mr-2 h-5 w-5 animate-spin" />
              Loading art index
            </div>
          ) : (
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
              {results.map(item => (
                <article key={item.id || item.slug} className="group overflow-hidden rounded border border-white/10 bg-white/[0.02]">
                  {item.thumbUrl || item.imageUrl ? (
                    <img src={item.thumbUrl || item.imageUrl} alt={item.prompt} loading="lazy" className="aspect-square w-full bg-white/[0.03] object-cover transition-transform duration-300 group-hover:scale-[1.02]" />
                  ) : (
                    <PromptPlaceholder prompt={item.prompt} />
                  )}
                  <div className="p-4">
                    <div className="mb-2 flex items-center justify-between gap-3 font-mono text-[11px] uppercase tracking-[0.16em] text-white/35">
                      <span>{item.model || 'zimage'}</span>
                      {item.steps ? <span>{item.steps} steps</span> : null}
                    </div>
                    <p className="line-clamp-4 min-h-20 text-sm leading-relaxed text-white/64">{item.prompt}</p>
                    <Link to={artPromptPlaygroundHref(item)} className="mt-4 inline-flex items-center gap-2 font-mono text-xs font-bold uppercase tracking-[0.14em] text-white/70 hover:text-white">
                      Try prompt <ArrowRight className="h-3.5 w-3.5" />
                    </Link>
                  </div>
                </article>
              ))}
            </div>
          )}
        </main>
      </div>
    </>
  );
}

function PromptPlaceholder({ prompt, large = false }: { prompt: string; large?: boolean }) {
  return (
    <div className={`flex ${large ? 'aspect-[16/10] min-h-80' : 'aspect-square'} w-full flex-col justify-between bg-[radial-gradient(circle_at_top_left,rgba(255,255,255,0.14),transparent_34%),linear-gradient(135deg,rgba(255,255,255,0.08),rgba(255,255,255,0.02))] p-5`}>
      <div className="font-mono text-[11px] uppercase tracking-[0.18em] text-white/35">Prompt queued</div>
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
