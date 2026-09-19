import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { BookOpen, Loader2, Plus, Search, Sparkles } from 'lucide-react';
import { Seo } from '../components/Seo';
import { SkillDetailPane } from '../components/skills/SkillDetailPane';
import { fetchSkillsMeta, listSkills, searchSkills, type Skill, type SkillsMeta } from '../data/skills';

export function Skills() {
  const [meta, setMeta] = useState<SkillsMeta | null>(null);
  const [query, setQuery] = useState('');
  const [source, setSource] = useState('');
  const [category, setCategory] = useState('');
  const [results, setResults] = useState<Skill[]>([]);
  const [total, setTotal] = useState(0);
  const [semantic, setSemantic] = useState(false);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState('');
  const debounce = useRef<number | undefined>(undefined);

  useEffect(() => {
    fetchSkillsMeta().then(setMeta).catch(() => {});
  }, []);
  const load = useCallback(async (q: string, src: string, cat: string) => {
    setLoading(true);
    try {
      const trimmed = q.trim();
      const r = trimmed
        ? await searchSkills(trimmed)
        : await listSkills({ source: src || undefined, category: cat || undefined, limit: 60 });
      let list = r.skills;
      // search endpoint has no filters: apply client-side like app-site does.
      if (trimmed) {
        if (src) list = list.filter(s => s.source === src);
        if (cat) list = list.filter(s => s.category === cat);
      }
      setResults(list);
      setTotal(r.count);
      setSemantic(trimmed ? r.semantic : false);
    } catch {
      setResults([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (debounce.current) clearTimeout(debounce.current);
    debounce.current = window.setTimeout(() => void load(query, source, category), query ? 200 : 0);
    return () => { if (debounce.current) clearTimeout(debounce.current); };
  }, [query, source, category, load]);

  useEffect(() => {
    if (!selected && results.length > 0) setSelected(results[0].slug);
  }, [results, selected]);

  const popularTags = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const s of results) for (const t of s.tags ?? []) counts[t] = (counts[t] ?? 0) + 1;
    return Object.entries(counts).sort((a, b) => b[1] - a[1]).slice(0, 8).map(([t]) => t);
  }, [results]);

  return (
    <>
      <Seo title="Skills marketplace | OpenPaths" description="Search versioned agent skills: copy the markdown, pin a version, browse attached files." path="/skills" />
      <div className="min-h-screen bg-black">
        <section className="border-b border-white/20 bg-white/[0.05]">
          <div className="mx-auto max-w-7xl px-6 py-10">
            <div className="mb-4 inline-flex items-center gap-2 rounded border border-white/20 bg-black px-3 py-2 font-mono text-xs uppercase tracking-[0.18em] text-white/50">
              <BookOpen className="h-4 w-4" /> Skills marketplace
            </div>
            <h1 className="max-w-4xl text-4xl font-bold tracking-tight md:text-6xl">Skills</h1>
            <p className="mt-4 max-w-3xl text-base leading-relaxed text-white/60">
              Versioned, owner-managed agent skills{(meta?.total ?? 0) > 0 && <> — {meta!.total.toLocaleString('en-US')} published</>}.
              Search, pin a version, copy the markdown, browse the files.
            </p>
            <div className="mt-5 flex flex-wrap gap-2">
              <Link to="/skills/new" className="inline-flex items-center gap-1.5 rounded border border-white bg-white px-3 py-1.5 font-mono text-xs font-bold text-black hover:bg-white/90">
                <Plus className="h-3.5 w-3.5" /> Publish a skill
              </Link>
            </div>
          </div>
        </section>

        <main className="mx-auto max-w-7xl px-6 py-8">
          <label className="relative mb-5 block">
            <Search className="pointer-events-none absolute left-4 top-1/2 h-5 w-5 -translate-y-1/2 text-white/50" />
            <input
              value={query}
              onChange={e => setQuery(e.target.value)}
              placeholder="Search skills by intent, tag, or use case..."
              className="h-14 w-full rounded border border-white/20 bg-white/[0.06] pl-12 pr-12 text-base text-white outline-none transition-colors placeholder:text-white/45 focus:border-white/60"
            />
            {loading && <Loader2 className="absolute right-4 top-1/2 h-5 w-5 -translate-y-1/2 animate-spin text-white/45" />}
          </label>

          <div className="mb-4 flex flex-wrap items-center gap-1.5">
            <FilterButton active={source === ''} onClick={() => setSource('')} label="All sources" />
            {(meta?.sources ?? []).map(s => (
              <FilterButton key={s.id} active={source === s.id} onClick={() => setSource(source === s.id ? '' : s.id)} label={`${s.label} ${s.count}`} />
            ))}
            {semantic && (
              <span className="ml-auto inline-flex items-center gap-1 font-mono text-xs text-white/45">
                <Sparkles className="h-3.5 w-3.5" /> semantic
              </span>
            )}
          </div>

          {(meta?.categories.length ?? 0) > 0 && (
            <div className="mb-4 flex flex-wrap gap-1.5">
              <FilterButton active={category === ''} onClick={() => setCategory('')} label="All categories" />
              {meta!.categories.map(c => (
                <FilterButton key={c.name} active={category === c.name} onClick={() => setCategory(category === c.name ? '' : c.name)} label={`${c.name} ${c.count}`} />
              ))}
            </div>
          )}

          {popularTags.length > 0 && (
            <div className="mb-6 flex flex-wrap items-center gap-2 text-sm text-white/45">
              {popularTags.map(tag => (
                <button key={tag} type="button" onClick={() => setQuery(tag)} className="rounded border border-white/20 px-2 py-1 font-mono text-xs text-white/55 hover:border-white/45 hover:text-white">
                  #{tag}
                </button>
              ))}
              {query && semantic && <span className="font-mono text-xs text-white/45">· semantic search</span>}
              {!query && <span className="font-mono text-xs text-white/45">{total} skills</span>}
            </div>
          )}

          <div className="grid min-w-0 gap-4 lg:grid-cols-[340px_minmax(0,1fr)]">
            <aside className="max-h-[70vh] min-w-0 max-w-full space-y-1 overflow-y-auto rounded-xl border border-white/15 bg-white/[0.03] p-2">
              {loading && results.length === 0 && (
                <p className="px-3 py-6 text-center text-sm text-white/45"><Loader2 className="mx-auto h-4 w-4 animate-spin" /></p>
              )}
              {!loading && results.length === 0 && (
                <p className="px-3 py-6 text-center font-mono text-sm text-white/55">No skills found. Try a different search.</p>
              )}
              {results.map(s => {
                const active = selected === s.slug;
                return (
                  <button
                    key={s.slug}
                    onClick={() => setSelected(s.slug)}
                    className={`w-full min-w-0 rounded-lg px-3 py-2.5 text-left transition-colors ${active ? 'bg-white text-black' : 'text-white/70 hover:bg-white/[0.06] hover:text-white'}`}
                  >
                    <span className="flex items-center gap-2 text-sm font-bold">
                      <BookOpen className="h-3.5 w-3.5 shrink-0" />
                      <span className="truncate">{s.name}</span>
                    </span>
                    <span className={`mt-1 block truncate text-xs ${active ? 'text-black/60' : 'text-white/40'}`}>{s.description}</span>
                    <span className={`mt-1 block font-mono text-[10px] ${active ? 'text-black/50' : 'text-white/35'}`}>v{s.currentVersion}{s.source ? ` · ${s.source}` : ''}</span>
                  </button>
                );
              })}
            </aside>

            {!selected ? (
              <section className="grid min-h-[380px] place-items-center rounded-xl border border-dashed border-white/15 p-8 text-center text-white/45">
                <div>
                  <BookOpen className="mx-auto mb-3 h-8 w-8" />
                  <p>Search, then pick a skill to see versions, files, and copyable markdown.</p>
                </div>
              </section>
            ) : (
              <div className="min-w-0 rounded-xl border border-white/15 bg-white/[0.03] p-5">
                <SkillDetailPane key={selected} slug={selected} linkTitle />
              </div>
            )}
          </div>
        </main>
      </div>
    </>
  );
}

function FilterButton({ active, onClick, label }: { active: boolean; onClick: () => void; label: string; key?: React.Key }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`rounded-full border px-2.5 py-1 font-mono text-xs transition-colors ${active ? 'border-white/40 bg-white text-black' : 'border-white/10 text-white/50 hover:bg-white/[0.04]'}`}
    >
      {label}
    </button>
  );
}
