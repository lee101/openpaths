import React, { useEffect, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { ArrowRight, Bolt, Loader2, Search, Sparkles, Tag } from 'lucide-react';
import { Seo } from '../components/Seo';
import { fallbackPrompts, type LibraryPrompt, type PromptMeta } from '../data/promptLibrary';
import { fetchPromptMeta, fetchPrompts, promptPlaygroundHref } from '../lib/promptLibrary';

type Scope = 'all' | 'category' | 'model' | 'type';

interface PromptsProps {
  scope?: Scope;
}

export function Prompts({ scope = 'all' }: PromptsProps) {
  const params = useParams();
  // model slugs can contain a slash (e.g. openpaths/auto-code) -> captured via splat.
  const scopeSlug = scope === 'model' ? (params['*'] || '') : (params.slug || '');

  const [meta, setMeta] = useState<PromptMeta | null>(null);
  const [query, setQuery] = useState('');
  const [activeType, setActiveType] = useState('');
  const [results, setResults] = useState<LibraryPrompt[]>(fallbackPrompts);
  const [total, setTotal] = useState(fallbackPrompts.length);
  const [loading, setLoading] = useState(true);
  const [searching, setSearching] = useState(false);
  const [semantic, setSemantic] = useState(false);

  useEffect(() => {
    let cancelled = false;
    fetchPromptMeta().then(m => {
      if (!cancelled && m) setMeta(m);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  // Reset transient state when the scope changes (navigating between subpages).
  useEffect(() => {
    setQuery('');
    setActiveType('');
  }, [scope, scopeSlug]);

  useEffect(() => {
    let cancelled = false;
    const filters = {
      q: query.trim(),
      category: scope === 'category' ? scopeSlug : undefined,
      model: scope === 'model' ? scopeSlug : undefined,
      type: scope === 'type' ? scopeSlug : activeType || undefined,
      limit: 60,
    };
    const isSearch = !!filters.q;
    if (isSearch) setSearching(true);
    else setLoading(true);
    const timer = window.setTimeout(async () => {
      const res = await fetchPrompts(filters);
      if (cancelled) return;
      setResults(res.results);
      setTotal(res.total);
      setSemantic(res.semantic);
      setSearching(false);
      setLoading(false);
    }, isSearch ? 200 : 0);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [query, scope, scopeSlug, activeType]);

  const heading = useScopeHeading(scope, scopeSlug, meta);
  const popularTags = useMemo(() => {
    const counts = new Map<string, number>();
    for (const p of results) for (const t of p.tags || []) counts.set(t, (counts.get(t) || 0) + 1);
    return [...counts.entries()].sort((a, b) => b[1] - a[1]).slice(0, 14).map(([t]) => t);
  }, [results]);

  const total_ = meta?.total ?? total;

  return (
    <>
      <Seo title={heading.seoTitle} description={heading.seoDescription} path={heading.path} />
      <div className="min-h-screen bg-black">
        <section className="border-b border-white/20 bg-white/[0.05]">
          <div className="mx-auto max-w-7xl px-6 py-10">
            <div className="mb-4 inline-flex items-center gap-2 rounded border border-white/20 bg-black px-3 py-2 font-mono text-xs uppercase tracking-[0.18em] text-white/50">
              <Sparkles className="h-4 w-4" /> Prompt library
            </div>
            <h1 className="max-w-4xl text-4xl font-bold tracking-tight md:text-6xl">{heading.title}</h1>
            <p className="mt-4 max-w-3xl text-base leading-relaxed text-white/60">{heading.subtitle}</p>
            <div className="mt-5 flex flex-wrap gap-2 text-xs font-mono text-white/45">
              <span className="rounded border border-white/20 px-2 py-1">{total_.toLocaleString('en-US')} prompts</span>
              {meta && <span className="rounded border border-white/20 px-2 py-1">{meta.categories.length} categories</span>}
              {meta && <span className="rounded border border-white/20 px-2 py-1">{meta.models.length} models</span>}
            </div>
          </div>
        </section>

        <main className="mx-auto max-w-7xl px-6 py-8">
          {/* Search */}
          <label className="relative mb-5 block">
            <Search className="pointer-events-none absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-white/50" />
            <input
              value={query}
              onChange={e => setQuery(e.target.value)}
              placeholder="Search prompts by intent, model, tag, or use case..."
              className="h-14 w-full rounded border border-white/20 bg-white/[0.06] pl-12 pr-12 text-base text-white outline-none transition-colors placeholder:text-white/45 focus:border-white/60"
            />
            {searching && <Loader2 className="absolute right-4 top-1/2 h-5 w-5 -translate-y-1/2 animate-spin text-white/45" />}
          </label>

          {/* Type filter (only on the global index) */}
          {scope === 'all' && meta && (
            <div className="mb-4 flex flex-wrap gap-2">
              <FilterChip active={activeType === ''} onClick={() => setActiveType('')} label="All types" />
              {meta.types.map(t => (
                <FilterChip key={t.slug} active={activeType === t.slug} onClick={() => setActiveType(t.slug)} label={t.name.replace(' prompts', '')} count={meta.counts.byType[t.slug] ?? meta.counts.free} />
              ))}
            </div>
          )}

          {/* Category browse links (global index only) */}
          {scope === 'all' && meta && (
            <div className="mb-6 flex flex-wrap gap-2">
              {meta.categories.map(c => (
                <Link key={c.slug} to={`/prompts/category/${c.slug}`} className="rounded border border-white/20 px-3 py-1.5 font-mono text-xs text-white/55 hover:border-white/50 hover:text-white">
                  {c.shortName} <span className="text-white/45">{meta.counts.byCategory[c.slug] ?? 0}</span>
                </Link>
              ))}
            </div>
          )}

          {/* Popular tags */}
          {popularTags.length > 0 && (
            <div className="mb-8 flex flex-wrap items-center gap-2 text-sm text-white/45">
              <Tag className="h-4 w-4" />
              {popularTags.map(tag => (
                <button key={tag} type="button" onClick={() => setQuery(tag)} className="rounded border border-white/20 px-2 py-1 font-mono text-xs text-white/55 hover:border-white/45 hover:text-white">
                  {tag}
                </button>
              ))}
              {query && semantic && <span className="font-mono text-xs text-white/45">· semantic search</span>}
            </div>
          )}

          {loading && results.length === 0 ? (
            <div className="flex items-center gap-2 py-16 text-white/55"><Loader2 className="h-5 w-5 animate-spin" /> Loading prompts…</div>
          ) : results.length === 0 ? (
            <div className="py-16 text-center font-mono text-sm text-white/55">No prompts found. Try a different search.</div>
          ) : (
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {results.map(prompt => (
                <PromptCard key={prompt.slug} prompt={prompt} />
              ))}
            </div>
          )}
        </main>
      </div>
    </>
  );
}

export function PromptCard({ prompt }: { prompt: LibraryPrompt; key?: React.Key }) {
  return (
    <div className="group flex flex-col rounded border border-white/20 bg-white/[0.05] p-4 transition-colors hover:border-white/45">
      <div className="mb-2 flex items-center gap-2 font-mono text-[11px] uppercase tracking-wide text-white/55">
        <span className="rounded border border-white/20 px-1.5 py-0.5">{prompt.modalityName.replace(' prompts', '')}</span>
        {prompt.isFree && <span className="inline-flex items-center gap-1 rounded border border-emerald-400/30 px-1.5 py-0.5 text-emerald-300/80"><Bolt className="h-3 w-3" /> free</span>}
      </div>
      <Link to={prompt.url} className="text-lg font-semibold text-white group-hover:underline">{prompt.title}</Link>
      <p className="mt-1 line-clamp-2 text-sm text-white/55">{prompt.summary}</p>
      <div className="mt-3 flex flex-wrap gap-1.5">
        {prompt.tags.slice(0, 4).map(tag => (
          <span key={tag} className="rounded bg-white/[0.07] px-1.5 py-0.5 font-mono text-[11px] text-white/55">{tag}</span>
        ))}
      </div>
      <div className="mt-4 flex items-center justify-between border-t border-white/20 pt-3">
        <span className="font-mono text-xs text-white/45">{prompt.modelName}</span>
        <div className="flex items-center gap-3">
          <Link to={prompt.url} className="font-mono text-xs text-white/55 hover:text-white">Details</Link>
          <Link to={promptPlaygroundHref(prompt)} className="inline-flex items-center gap-1 font-mono text-xs text-white hover:text-white/80">
            Try <ArrowRight className="h-3 w-3" />
          </Link>
        </div>
      </div>
    </div>
  );
}

function FilterChip({ active, onClick, label, count }: { active: boolean; onClick: () => void; label: string; count?: number; key?: React.Key }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded border px-3 py-1.5 font-mono text-xs transition-colors ${active ? 'border-white bg-white text-black' : 'border-white/20 text-white/55 hover:border-white/50 hover:text-white'}`}
    >
      {label}{typeof count === 'number' ? <span className={active ? 'text-black/50' : 'text-white/45'}> {count}</span> : null}
    </button>
  );
}

function useScopeHeading(scope: Scope, slug: string, meta: PromptMeta | null) {
  const base = 'AI prompt library with copy-ready prompts for code, image, video, and music models - search by intent and open any prompt in the OpenPaths playground.';
  if (scope === 'category') {
    const c = meta?.categories.find(x => x.slug === slug);
    const name = c?.name || titleize(slug);
    return { title: name, subtitle: c?.description || base, seoTitle: `${name} | OpenPaths Prompt Library`, seoDescription: c?.description || base, path: `/prompts/category/${slug}` };
  }
  if (scope === 'model') {
    const m = meta?.models.find(x => x.slug === slug);
    const name = m?.name || titleize(slug);
    return { title: `${name} prompts`, subtitle: m?.description || `Curated prompts tuned for ${name}.`, seoTitle: `${name} Prompts | OpenPaths`, seoDescription: m?.description || `Prompts for ${name}.`, path: `/prompts/model/${slug}` };
  }
  if (scope === 'type') {
    const t = meta?.types.find(x => x.slug === slug);
    const name = t?.name || `${titleize(slug)} prompts`;
    return { title: name, subtitle: t?.description || base, seoTitle: `${name} | OpenPaths Prompt Library`, seoDescription: t?.description || base, path: `/prompts/type/${slug}` };
  }
  return { title: 'AI Prompt Library', subtitle: base, seoTitle: 'AI Prompt Library | OpenPaths', seoDescription: base, path: '/prompts' };
}

function titleize(slug: string): string {
  return slug
    .split(/[-/]/)
    .filter(Boolean)
    .map(s => s.charAt(0).toUpperCase() + s.slice(1))
    .join(' ');
}
