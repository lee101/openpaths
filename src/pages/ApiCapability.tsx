import React, { useMemo, useState } from 'react';
import { Link, Navigate, useParams } from 'react-router-dom';
import { ArrowRight, BookOpen, KeyRound } from 'lucide-react';
import { Seo } from '../components/Seo';
import { CapabilityTryIt } from '../components/CapabilityTryIt';
import { requestSignIn } from '../lib/paywall';
import { formatCurrency, isFreeModel, isTokenPriced, priceSummary } from '../lib/modelPricing';
import { modelPath } from '../lib/paths';
import { models, type Model } from '../data/models';
import {
  CAPABILITIES,
  VS_PAIRS,
  capabilityBySlug,
  capabilityModels,
  vsBySlug,
  type Capability,
} from '../data/capabilities';
import { HUB_SEO, capabilityJsonLd, fromPrice, hubJsonLd, vsJsonLd, vsSeo, vsVerdict, workload } from '../lib/apiPagesSeo';

const TABLE_LIMIT = 25;

const btnPrimary = 'inline-flex items-center justify-center gap-2 rounded border border-white bg-white px-5 py-3 font-mono text-sm font-bold text-black hover:bg-white/90 transition-colors';
const btnGhost = 'inline-flex items-center justify-center gap-2 rounded border border-white/20 bg-white/[0.06] px-5 py-3 font-mono text-sm text-white/70 hover:border-white/50 hover:text-white transition-colors';

function Crumbs({ items }: { items: Array<{ label: string; to?: string }> }) {
  return (
    <nav className="mb-6 flex flex-wrap items-center gap-2 font-mono text-xs text-white/50" aria-label="Breadcrumb">
      {items.map((item, i) => (
        <React.Fragment key={item.label}>
          {i > 0 && <span>/</span>}
          {item.to ? <Link to={item.to} className="hover:text-white">{item.label}</Link> : <span className="text-white/75">{item.label}</span>}
        </React.Fragment>
      ))}
    </nav>
  );
}

export function ApiHub() {
  const cards = CAPABILITIES.map(cap => {
    const list = capabilityModels(cap);
    return { cap, count: list.length || cap.staticRows?.length || 0, from: fromPrice(cap) };
  });
  return (
    <>
      <Seo
        title={HUB_SEO.title}
        description={HUB_SEO.description}
        path={HUB_SEO.path}
        jsonLd={hubJsonLd()}
      />
      <section className="mx-auto max-w-6xl px-6 py-16">
        <Crumbs items={[{ label: 'Home', to: '/' }, { label: 'API' }]} />
        <h1 className="text-4xl font-bold tracking-tight md:text-6xl">One API key for every model</h1>
        <p className="mt-5 max-w-3xl text-lg font-light leading-relaxed text-white/62">
          OpenPaths is an OpenAI- and Anthropic-compatible gateway. Pick a capability to see every model that serves it, live prices, a try-it box and copy-paste code.
        </p>
        <div className="mt-8 flex flex-col gap-3 sm:flex-row">
          <button type="button" onClick={requestSignIn} className={btnPrimary}><KeyRound className="h-4 w-4" /> Get an API key</button>
          <Link to="/docs" className={btnGhost}><BookOpen className="h-4 w-4" /> Docs</Link>
        </div>
        <div className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {cards.map(({ cap, count, from }) => (
            <Link key={cap.slug} to={cap.path} className="group rounded-xl border border-white/20 bg-white/[0.05] p-5 transition-colors hover:border-white/45 hover:bg-white/[0.07]">
              <div className="flex items-start justify-between gap-3">
                <h2 className="text-lg font-semibold tracking-tight">{cap.name}</h2>
                <ArrowRight className="mt-1 h-4 w-4 shrink-0 text-white/40 group-hover:text-white/70" />
              </div>
              <p className="mt-2 text-sm leading-relaxed text-white/55">{cap.tagline}</p>
              <p className="mt-4 font-mono text-[11px] text-white/45">
                {count} {count === 1 ? 'model' : 'models'}{from ? ` · from ${from}` : ''}
              </p>
            </Link>
          ))}
        </div>
        <div className="mt-14 border-t border-white/20 pt-10">
          <h2 className="text-2xl font-bold tracking-tight">Head-to-head comparisons</h2>
          <div className="mt-5 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
            {VS_PAIRS.slice(0, 18).map(p => (
              <Link key={p.slug} to={`/vs/${p.slug}`} className="rounded border border-white/15 px-3 py-2 font-mono text-xs text-white/65 hover:border-white/45 hover:text-white">
                {p.a.name} vs {p.b.name}
              </Link>
            ))}
          </div>
        </div>
        <div className="mt-10 flex flex-wrap gap-x-6 gap-y-2 font-mono text-xs text-white/60">
          <Link to="/pricing" className="hover:text-white">Pricing</Link>
          <Link to="/models" className="hover:text-white">All models</Link>
          <Link to="/tools" className="hover:text-white">Studios</Link>
          <Link to="/playground" className="hover:text-white">Playground</Link>
          <Link to="/alternatives" className="hover:text-white">OpenRouter alternative</Link>
        </div>
      </section>
    </>
  );
}

export function ApiCapability() {
  const { slug = '' } = useParams<{ slug: string }>();
  const cap = capabilityBySlug.get(slug);
  if (!cap) return <Navigate to="/api" replace />;
  return <CapabilityView cap={cap} />;
}

function CapabilityView({ cap }: { cap: Capability }) {
  const list = useMemo(() => capabilityModels(cap), [cap]);
  const defaultModel = models.find(m => m.id === cap.defaultModel);
  const tryModels = useMemo(() => {
    if (cap.kind === 'studio') return list;
    const base = list.slice(0, 40);
    return defaultModel && !base.some(m => m.id === defaultModel.id) ? [defaultModel, ...base] : base;
  }, [cap.kind, defaultModel, list]);
  const [selected, setSelected] = useState(cap.defaultModel);
  const [showAll, setShowAll] = useState(false);
  const [sort, setSort] = useState<'popularity' | 'price'>(cap.sort || 'popularity');

  const rows = useMemo(() => {
    const sorted = [...list].sort((a, b) => (sort === 'price' ? a.priceInput - b.priceInput || a.popularity - b.popularity : a.popularity - b.popularity));
    return showAll ? sorted : sorted.slice(0, TABLE_LIMIT);
  }, [list, showAll, sort]);

  const model = tryModels.find(m => m.id === selected) || tryModels[0] || defaultModel;
  const pairs = VS_PAIRS.filter(p => p.cap.slug === cap.slug);
  const showTokens = list.some(isTokenPriced);

  return (
    <>
      <Seo title={cap.seoTitle} description={cap.seoDescription} path={cap.path} jsonLd={capabilityJsonLd(cap)} />
      <section className="mx-auto max-w-6xl px-6 py-16">
        <Crumbs items={[{ label: 'Home', to: '/' }, { label: 'API', to: '/api' }, { label: cap.name }]} />
        <h1 className="text-4xl font-bold tracking-tight md:text-6xl">{cap.h1}</h1>
        <p className="mt-5 max-w-3xl text-lg font-light leading-relaxed text-white/62">{cap.intro}</p>
        <div className="mt-8 flex flex-col gap-3 sm:flex-row">
          <button type="button" onClick={requestSignIn} className={btnPrimary}><KeyRound className="h-4 w-4" /> Get an API key</button>
          <Link to="/docs" className={btnGhost}><BookOpen className="h-4 w-4" /> API docs</Link>
          {cap.studioPath && <Link to={cap.studioPath} className={btnGhost}>{cap.studioLabel} <ArrowRight className="h-4 w-4" /></Link>}
        </div>

        <div className="mt-12">
          <CapabilityTryIt cap={cap} model={model} models={tryModels} onModelChange={setSelected} />
        </div>

        <div className="mt-16">
          <div className="mb-4 flex flex-wrap items-end justify-between gap-3">
            <div>
              <h2 className="text-2xl font-bold tracking-tight">{cap.name} models and pricing</h2>
              <p className="mt-1 text-sm text-white/55">{list.length || cap.staticRows?.length || 0} models, prices from the live OpenPaths catalog.</p>
            </div>
            {list.length > 1 && (
              <label className="flex items-center gap-2 font-mono text-xs text-white/55">
                Sort
                <select value={sort} onChange={e => setSort(e.target.value as 'popularity' | 'price')} className="rounded border border-white/20 bg-black px-2 py-1 text-white">
                  <option value="popularity">Popularity</option>
                  <option value="price">Price</option>
                </select>
              </label>
            )}
          </div>
          <div className="overflow-x-auto rounded-xl border border-white/20">
            <table className="w-full min-w-[640px] text-left text-sm">
              <thead className="border-b border-white/20 bg-white/[0.05] font-mono text-[11px] uppercase tracking-[0.14em] text-white/50">
                <tr>
                  <th className="px-4 py-3">Model</th>
                  <th className="px-4 py-3">Provider</th>
                  {showTokens && <th className="px-4 py-3">Context</th>}
                  <th className="px-4 py-3">Price</th>
                </tr>
              </thead>
              <tbody>
                {rows.map(m => (
                  <tr key={m.id} className="border-b border-white/10 last:border-0">
                    <td className="px-4 py-3"><Link to={modelPath(m.id)} className="font-medium hover:underline">{m.name}</Link><code className="block text-[11px] text-white/40">{m.id}</code></td>
                    <td className="px-4 py-3 text-white/60">{m.provider}</td>
                    {showTokens && <td className="px-4 py-3 font-mono text-xs text-white/60">{m.contextLength}</td>}
                    <td className={`px-4 py-3 font-mono text-xs ${isFreeModel(m) ? 'text-emerald-200' : 'text-white/75'}`}>{priceSummary(m)}</td>
                  </tr>
                ))}
                {(cap.staticRows || []).map(r => (
                  <tr key={r.id} className="border-b border-white/10 last:border-0">
                    <td className="px-4 py-3"><span className="font-medium">{r.name}</span><code className="block text-[11px] text-white/40">{r.id}</code></td>
                    <td className="px-4 py-3 text-white/60">{r.provider}</td>
                    {showTokens && <td className="px-4 py-3" />}
                    <td className="px-4 py-3 font-mono text-xs text-white/75">{r.price}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {list.length > TABLE_LIMIT && !showAll && (
            <button type="button" onClick={() => setShowAll(true)} className="mt-3 font-mono text-xs text-white/60 hover:text-white">Show all {list.length} models</button>
          )}
        </div>

        {pairs.length > 0 && (
          <div className="mt-14">
            <h2 className="text-2xl font-bold tracking-tight">Compare {cap.name.toLowerCase()} models</h2>
            <div className="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
              {pairs.slice(0, 12).map(p => (
                <Link key={p.slug} to={`/vs/${p.slug}`} className="rounded border border-white/15 px-3 py-2 font-mono text-xs text-white/65 hover:border-white/45 hover:text-white">
                  {p.a.name} vs {p.b.name}
                </Link>
              ))}
            </div>
          </div>
        )}

        <div className="mt-14">
          <h2 className="text-2xl font-bold tracking-tight">FAQ</h2>
          <dl className="mt-4 space-y-5">
            {cap.faq.map(item => (
              <div key={item.q}>
                <dt className="font-semibold">{item.q}</dt>
                <dd className="mt-1 text-sm leading-relaxed text-white/60">{item.a}</dd>
              </div>
            ))}
          </dl>
        </div>

        <div className="mt-14 border-t border-white/20 pt-8">
          <h2 className="mb-3 font-mono text-xs uppercase tracking-[0.18em] text-white/50">More APIs</h2>
          <div className="flex flex-wrap gap-x-5 gap-y-2 font-mono text-xs text-white/65">
            {cap.related.map(slug => {
              const r = capabilityBySlug.get(slug);
              return r ? <Link key={slug} to={r.path} className="hover:text-white">{r.name} →</Link> : null;
            })}
            <Link to="/api" className="hover:text-white">All APIs →</Link>
            <Link to="/pricing" className="hover:text-white">Pricing →</Link>
            <Link to="/models" className="hover:text-white">All models →</Link>
          </div>
        </div>
      </section>
    </>
  );
}

export function ModelVs() {
  const { pair = '' } = useParams<{ pair: string }>();
  const entry = vsBySlug.get(pair);
  const [selected, setSelected] = useState<string>(entry?.a.id || '');
  if (!entry) return <Navigate to="/api" replace />;
  const { a, b, cap } = entry;
  const tokens = isTokenPriced(a) && isTokenPriced(b);
  const rowsDef: Array<[string, (m: Model) => string]> = [
    ['Provider', m => m.provider],
    ['Model ID', m => m.id],
    ['Context', m => m.contextLength],
    ['Price', m => priceSummary(m)],
    ['Released', m => m.released],
    ['Tags', m => m.tags.join(', ')],
  ];
  if (tokens) {
    const w = workload();
    rowsDef.splice(4, 0, [`Cost for ${w.label}`, m => formatCurrency(w.cost(m))]);
  }
  const current = selected === b.id ? b : a;
  const otherPairs = VS_PAIRS.filter(p => p.slug !== entry.slug && (p.a.id === a.id || p.b.id === a.id || p.a.id === b.id || p.b.id === b.id)).slice(0, 8);
  return (
    <>
      <Seo
        title={vsSeo(entry).title}
        description={vsSeo(entry).description}
        path={vsSeo(entry).path}
        jsonLd={vsJsonLd(entry)}
      />
      <section className="mx-auto max-w-5xl px-6 py-16">
        <Crumbs items={[{ label: 'Home', to: '/' }, { label: 'API', to: '/api' }, { label: cap.name, to: cap.path }, { label: `${a.name} vs ${b.name}` }]} />
        <h1 className="text-4xl font-bold tracking-tight md:text-5xl">{a.name} vs {b.name}</h1>
        <p className="mt-5 max-w-3xl text-lg font-light leading-relaxed text-white/62">{vsVerdict(a, b)}</p>
        <div className="mt-10 overflow-x-auto rounded-xl border border-white/20">
          <table className="w-full min-w-[560px] text-left text-sm">
            <thead className="border-b border-white/20 bg-white/[0.05] font-mono text-[11px] uppercase tracking-[0.14em] text-white/50">
              <tr><th className="px-4 py-3" /><th className="px-4 py-3"><Link to={modelPath(a.id)} className="hover:underline">{a.name}</Link></th><th className="px-4 py-3"><Link to={modelPath(b.id)} className="hover:underline">{b.name}</Link></th></tr>
            </thead>
            <tbody>
              {rowsDef.map(([label, get]) => (
                <tr key={label} className="border-b border-white/10 last:border-0">
                  <td className="px-4 py-3 font-mono text-xs text-white/50">{label}</td>
                  <td className="px-4 py-3 text-white/80">{get(a)}</td>
                  <td className="px-4 py-3 text-white/80">{get(b)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div className="mt-12">
          <h2 className="mb-4 text-2xl font-bold tracking-tight">Try both with the same key</h2>
          <CapabilityTryIt cap={cap} model={current} models={[a, b]} onModelChange={setSelected} />
        </div>
        {otherPairs.length > 0 && (
          <div className="mt-12">
            <h2 className="text-xl font-bold tracking-tight">More comparisons</h2>
            <div className="mt-4 grid gap-2 sm:grid-cols-2">
              {otherPairs.map(p => <Link key={p.slug} to={`/vs/${p.slug}`} className="rounded border border-white/15 px-3 py-2 font-mono text-xs text-white/65 hover:border-white/45 hover:text-white">{p.a.name} vs {p.b.name}</Link>)}
            </div>
          </div>
        )}
        <div className="mt-12 flex flex-wrap gap-x-5 gap-y-2 font-mono text-xs text-white/65">
          <Link to={cap.path} className="hover:text-white">{cap.name} API →</Link>
          <Link to={modelPath(a.id)} className="hover:text-white">{a.name} →</Link>
          <Link to={modelPath(b.id)} className="hover:text-white">{b.name} →</Link>
          <Link to="/compare" className="hover:text-white">Benchmark comparisons →</Link>
          <Link to="/pricing" className="hover:text-white">Pricing →</Link>
        </div>
      </section>
    </>
  );
}
