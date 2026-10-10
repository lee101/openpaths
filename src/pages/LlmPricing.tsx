import React from 'react';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { ArrowRight, Calculator as CalculatorIcon, ExternalLink, Search, Zap } from 'lucide-react';
import { Seo } from '../components/Seo';
import { ShareButton } from '../components/ShareButton';
import { artificialAnalysisSnapshot, displayScore, formatDollars } from '../lib/artificialAnalysis';
import { faqLd } from '../lib/seoSchema';
import { hubSeo, modelSeo, pairSeo, providerSeo, scenarioSeo } from '../lib/llmPricingSeo';
import {
  DEFAULT_WORKLOAD,
  PRICED_MODELS,
  PRICING_UPDATED,
  SCENARIOS,
  aaFor,
  aaIndexPerDollar,
  aaUrl,
  blendedPrice,
  compareScenarios,
  costFor,
  fmtInt,
  fmtMoney,
  fmtRate,
  fmtTokens,
  formatRatio,
  getPricedModel,
  getScenario,
  modelFaq,
  neighbours,
  pairFaq,
  pairSlug,
  pairVerdict,
  parsePairQuery,
  parsePairSlug,
  popularPairModels,
  priceLine,
  rankForScenario,
  resolveModel,
  tierNotes,
  type PricedModel,
  type Workload,
} from '../lib/llmPricing';

const card = 'rounded-lg border border-white/20 bg-white/[0.05]';
const label = 'font-mono text-xs uppercase tracking-[0.16em] text-white/60';
const linkCls = 'text-cyan-200 underline decoration-cyan-200/25 underline-offset-4 hover:text-cyan-100';

function Shell({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-7xl px-6 py-12">{children}</div>;
}

function Crumbs({ items }: { items: Array<{ to?: string; label: string }> }) {
  return (
    <nav aria-label="Breadcrumb" className="mb-8 flex flex-wrap items-center gap-2 font-mono text-xs text-white/50">
      {items.map((item, i) => (
        <React.Fragment key={item.label}>
          {i > 0 && <span>/</span>}
          {item.to ? <Link to={item.to} className="hover:text-white">{item.label}</Link> : <span className="text-white/75">{item.label}</span>}
        </React.Fragment>
      ))}
    </nav>
  );
}

function Hero({ eyebrow, title, intro }: { eyebrow: string; title: string; intro: React.ReactNode }) {
  return (
    <section className="mb-8">
      <div className="mb-4 inline-flex items-center gap-2 font-mono text-xs uppercase tracking-[0.22em] text-cyan-300">
        <CalculatorIcon className="h-4 w-4" />
        {eyebrow}
      </div>
      <h1 className="max-w-5xl text-4xl font-semibold tracking-tight md:text-5xl">{title}</h1>
      <p className="mt-5 max-w-3xl text-lg leading-relaxed text-white/60">{intro}</p>
    </section>
  );
}

function Section({ title, children, right }: { title: string; children: React.ReactNode; right?: React.ReactNode }) {
  return (
    <section className={`${card} mb-8 overflow-hidden`}>
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-white/20 px-4 py-3">
        <h2 className={label}>{title}</h2>
        {right}
      </div>
      <div className="p-4">{children}</div>
    </section>
  );
}

function Faq({ items }: { items: Array<{ q: string; a: string }> }) {
  return (
    <Section title="Frequently asked">
      <dl className="space-y-4">
        {items.map(item => (
          <div key={item.q}>
            <dt className="font-medium text-white">{item.q}</dt>
            <dd className="mt-1 text-white/60">{item.a}</dd>
          </div>
        ))}
      </dl>
    </Section>
  );
}

function useWorkload(initial: Workload): [Workload, (patch: Partial<Workload>) => void] {
  const [params, setParams] = useSearchParams();
  const read = (key: string, fallback: number) => {
    const raw = params.get(key);
    const n = raw === null ? NaN : Number(raw);
    return Number.isFinite(n) && n >= 0 ? n : fallback;
  };
  const workload: Workload = {
    requests: Math.min(read('r', initial.requests), 1e10),
    inputTokens: Math.min(read('i', initial.inputTokens), 2_000_000),
    outputTokens: Math.min(read('o', initial.outputTokens), 500_000),
    cacheHit: Math.min(read('c', Math.round(initial.cacheHit * 100)), 100) / 100,
  };
  const update = (patch: Partial<Workload>) => {
    const next = { ...workload, ...patch };
    const p = new URLSearchParams(params);
    p.set('r', String(next.requests));
    p.set('i', String(next.inputTokens));
    p.set('o', String(next.outputTokens));
    p.set('c', String(Math.round(next.cacheHit * 100)));
    setParams(p, { replace: true });
  };
  return [workload, update];
}

function NumberField({ id, text, value, onChange, max, step = 1, suffix }: { id: string; text: string; value: number; onChange: (v: number) => void; max: number; step?: number; suffix?: string }) {
  return (
    <label htmlFor={id} className="block">
      <span className={label}>{text}</span>
      <div className="mt-2 flex items-center gap-2">
        <input
          id={id}
          type="number"
          inputMode="numeric"
          min={0}
          max={max}
          step={step}
          value={Number.isFinite(value) ? value : 0}
          onChange={e => onChange(Math.max(0, Math.min(max, Number(e.target.value) || 0)))}
          className="h-11 w-full rounded-lg border border-white/20 bg-black px-3 font-mono text-sm text-white outline-none focus:border-cyan-300"
        />
        {suffix && <span className="font-mono text-xs text-white/50">{suffix}</span>}
      </div>
    </label>
  );
}

function WorkloadInputs({ workload, onChange }: { workload: Workload; onChange: (patch: Partial<Workload>) => void }) {
  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <NumberField id="wl-req" text="Requests / month" value={workload.requests} onChange={v => onChange({ requests: v })} max={1e10} step={1000} />
      <NumberField id="wl-in" text="Input tokens / request" value={workload.inputTokens} onChange={v => onChange({ inputTokens: v })} max={2_000_000} step={100} />
      <NumberField id="wl-out" text="Output tokens / request" value={workload.outputTokens} onChange={v => onChange({ outputTokens: v })} max={500_000} step={50} />
      <NumberField id="wl-cache" text="Input served from cache" value={Math.round(workload.cacheHit * 100)} onChange={v => onChange({ cacheHit: v / 100 })} max={100} step={5} suffix="%" />
    </div>
  );
}

function CalcTable({ models, workload }: { models: PricedModel[]; workload: Workload }) {
  const rows = models.map(model => ({ model, cost: costFor(model, workload) })).sort((a, b) => a.cost.total - b.cost.total);
  const max = Math.max(...rows.map(r => r.cost.total), 0.0001);
  return (
    <div className="overflow-x-auto" tabIndex={0} role="region" aria-label="Cost estimate">
      <table className="w-full min-w-[760px] text-left text-sm">
        <thead className="font-mono text-xs uppercase tracking-[0.14em] text-white/50">
          <tr>
            <th className="px-3 py-3">Model</th>
            <th className="px-3 py-3">Rates used (in / out / cached)</th>
            <th className="px-3 py-3 text-right">Input</th>
            <th className="px-3 py-3 text-right">Output</th>
            <th className="px-3 py-3 text-right">Monthly</th>
            <th className="px-3 py-3 text-right">Per 1K req</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-white/10">
          {rows.map(({ model, cost }, i) => (
            <tr key={model.id} className={i === 0 ? 'bg-cyan-300/[0.06]' : undefined}>
              <td className="px-3 py-3">
                <Link to={`/llm-pricing/${model.slug}`} className="font-medium text-white hover:text-cyan-200">{model.name}</Link>
                <div className="font-mono text-xs text-white/45">{model.provider}</div>
              </td>
              <td className="px-3 py-3 font-mono text-xs text-white/70">
                {fmtRate(cost.rates.input)} / {fmtRate(cost.rates.output)} / {fmtRate(cost.rates.cacheRead)}
                {cost.rates.tier === 'long-context' && <span className="ml-2 rounded bg-amber-300/15 px-1.5 py-0.5 text-amber-200">long-context tier</span>}
                {cost.rates.tier === 'blended' && <span className="ml-2 rounded bg-sky-300/15 px-1.5 py-0.5 text-sky-200">peak/off-peak mix</span>}
              </td>
              <td className="px-3 py-3 text-right font-mono">{fmtMoney(cost.inputCost)}</td>
              <td className="px-3 py-3 text-right font-mono">{fmtMoney(cost.outputCost)}</td>
              <td className="px-3 py-3 text-right">
                <div className="font-mono font-semibold text-white">{fmtMoney(cost.total)}</div>
                <div className="ml-auto mt-1 h-1 rounded bg-cyan-300/70" style={{ width: `${Math.max(4, (cost.total / max) * 100)}%`, maxWidth: '100%' }} />
              </td>
              <td className="px-3 py-3 text-right font-mono text-white/70">{fmtMoney(cost.per1kRequests)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Calculator({ models, initial = DEFAULT_WORKLOAD, extra }: { models: PricedModel[]; initial?: Workload; extra?: React.ReactNode }) {
  const [workload, update] = useWorkload(initial);
  return (
    <Section title="Calculator" right={<span className="font-mono text-xs text-white/45">Edit any field; the URL is shareable</span>}>
      <WorkloadInputs workload={workload} onChange={update} />
      {extra}
      <div className="mt-5">
        <CalcTable models={models} workload={workload} />
      </div>
      <p className="mt-3 text-xs text-white/45">
        Estimates use published per-token rates as of {PRICING_UPDATED}, apply long-context tiers when the prompt exceeds the tier threshold, and blend peak and off-peak rates by hours. Cache write premiums are excluded.
      </p>
    </Section>
  );
}

function RateCard({ models }: { models: PricedModel[] }) {
  const rows: Array<[string, (m: PricedModel) => string]> = [
    ['Input / 1M tokens', m => (m.schedule ? `${fmtRate(m.schedule.offPeak.input)} off-peak, ${fmtRate(m.schedule.peak.input)} peak` : fmtRate(m.input))],
    ['Output / 1M tokens', m => (m.schedule ? `${fmtRate(m.schedule.offPeak.output)} off-peak, ${fmtRate(m.schedule.peak.output)} peak` : fmtRate(m.output))],
    ['Cached input / 1M', m => (m.schedule?.offPeak.cacheRead !== undefined ? `${fmtRate(m.schedule.offPeak.cacheRead)} off-peak, ${fmtRate(m.schedule.peak.cacheRead)} peak` : fmtRate(m.cacheRead))],
    ['Long-context tier', m => (m.longContext ? `> ${fmtTokens(m.longContext.threshold)}: ${fmtRate(m.longContext.input)} in / ${fmtRate(m.longContext.output)} out` : 'No tier')],
    ['Context window', m => fmtTokens(m.context)],
    ['Max output', m => (m.maxOutput ? fmtTokens(m.maxOutput) : 'n/a')],
    ['Vision / tools', m => `${m.vision ? 'Vision' : 'No vision'} / ${m.tools ? 'Tools' : 'No tools'}`],
    ['API model ID', m => m.id],
  ];
  return (
    <div className="overflow-x-auto" tabIndex={0} role="region" aria-label="Rate card">
      <table className="w-full min-w-[640px] text-left text-sm">
        <thead className="font-mono text-xs uppercase tracking-[0.14em] text-white/50">
          <tr>
            <th className="px-3 py-3">Rate</th>
            {models.map(m => <th key={m.id} className="px-3 py-3 normal-case tracking-normal text-white/80"><Link to={`/llm-pricing/${m.slug}`} className="hover:text-cyan-200">{m.name}</Link></th>)}
          </tr>
        </thead>
        <tbody className="divide-y divide-white/10">
          {rows.map(([name, fn]) => (
            <tr key={name}>
              <td className="px-3 py-3 text-white/55">{name}</td>
              {models.map(m => <td key={m.id} className="px-3 py-3 font-mono text-xs text-white/85">{fn(m)}</td>)}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TierNotes({ models }: { models: PricedModel[] }) {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      {models.map(m => (
        <div key={m.id}>
          <h3 className="mb-2 font-medium text-white">{m.name}</h3>
          <ul className="list-disc space-y-2 pl-5 text-sm text-white/60">
            {tierNotes(m).map(note => <li key={note}>{note}</li>)}
          </ul>
        </div>
      ))}
    </div>
  );
}

function AaPanel({ models }: { models: PricedModel[] }) {
  const rows = models.map(m => ({ m, aa: aaFor(m) }));
  const found = rows.filter(r => r.aa);
  return (
    <Section
      title="Artificial Analysis benchmarks"
      right={<a href={artificialAnalysisSnapshot.sourceUrl} target="_blank" rel="noreferrer noopener" className="inline-flex items-center gap-1 font-mono text-xs text-white/55 hover:text-white">artificialanalysis.ai <ExternalLink className="h-3.5 w-3.5" /></a>}
    >
      <div className="overflow-x-auto" tabIndex={0} role="region" aria-label="Artificial Analysis metrics">
        <table className="w-full min-w-[720px] text-left text-sm">
          <thead className="font-mono text-xs uppercase tracking-[0.14em] text-white/50">
            <tr>
              <th className="px-3 py-3">Model</th>
              <th className="px-3 py-3 text-right">Intelligence Index</th>
              <th className="px-3 py-3 text-right">Output tok/s</th>
              <th className="px-3 py-3 text-right">AA blended $/1M</th>
              <th className="px-3 py-3 text-right">Index per $ (3:1)</th>
              <th className="px-3 py-3">Source</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/10">
            {rows.map(({ m, aa }) => (
              <tr key={m.id}>
                <td className="px-3 py-3 text-white">{m.name}</td>
                <td className="px-3 py-3 text-right font-mono">{aa ? displayScore(aa.evaluations.intelligenceIndex, 'index') : 'n/a'}</td>
                <td className="px-3 py-3 text-right font-mono">{aa && typeof aa.performance.outputTokensPerSecond === 'number' ? Math.round(aa.performance.outputTokensPerSecond) : 'n/a'}</td>
                <td className="px-3 py-3 text-right font-mono">{aa ? formatDollars(aa.prices.blendedPerMTokens, 3) : 'n/a'}</td>
                <td className="px-3 py-3 text-right font-mono">{(() => { const v = aaIndexPerDollar(m); return v === null ? 'n/a' : v.toFixed(0); })()}</td>
                <td className="px-3 py-3">
                  {aa ? <a href={aaUrl(aa)} target="_blank" rel="noreferrer noopener" className={`${linkCls} inline-flex items-center gap-1`}>AA model page <ExternalLink className="h-3.5 w-3.5" /></a> : <span className="text-white/40">not in snapshot</span>}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <p className="mt-3 text-xs text-white/45">
        Crawled {artificialAnalysisSnapshot.crawledAt.slice(0, 10)}. Index per $ divides the Intelligence Index by OpenPaths blended price (3 input : 1 output). Higher means more benchmark intelligence per dollar.
      </p>
      <div className="mt-3 flex flex-wrap gap-4 text-sm">
        <Link to="/evals" className={linkCls}>All evals</Link>
        {found.length >= 2 && <Link to={`/compare/${found.map(r => r.aa!.slug).join('-vs-')}`} className={linkCls}>Full benchmark comparison</Link>}
      </div>
    </Section>
  );
}

function ModelLink({ m }: { m: PricedModel } & React.Attributes) {
  return (
    <Link to={`/llm-pricing/${m.slug}`} className="group rounded-lg border border-white/20 bg-white/[0.06] p-4 transition-colors hover:border-white/50">
      <div className="font-medium text-white group-hover:text-cyan-200">{m.name}</div>
      <div className="mt-1 font-mono text-xs text-white/55">{priceLine(m)}</div>
    </Link>
  );
}

function PairLink({ a, b }: { a: PricedModel; b: PricedModel } & React.Attributes) {
  return (
    <Link to={`/llm-pricing/compare/${pairSlug(a, b)}`} className="group flex items-center justify-between gap-3 rounded-lg border border-white/20 bg-white/[0.06] p-4 transition-colors hover:border-white/50">
      <span className="text-white group-hover:text-cyan-200">{a.name} vs {b.name}</span>
      <ArrowRight className="h-4 w-4 shrink-0 text-white/40" />
    </Link>
  );
}

function Cta({ model }: { model?: PricedModel }) {
  return (
    <div className="mb-8 flex flex-wrap gap-3">
      <Link to={model ? `/playground?model=${encodeURIComponent(model.id)}` : '/playground'} className="inline-flex items-center gap-2 rounded-lg bg-white px-4 py-3 font-mono text-sm font-semibold text-black">
        <Zap className="h-4 w-4" /> Try {model ? model.name : 'a model'} in the Playground
      </Link>
      <Link to="/pricing" className="inline-flex items-center gap-2 rounded-lg border border-white/20 px-4 py-3 font-mono text-sm text-white/70 hover:border-white/50 hover:text-white">
        OpenPaths pricing <ArrowRight className="h-4 w-4" />
      </Link>
    </div>
  );
}

export function LlmPricingHub() {
  const hub = hubSeo();
  const navigate = useNavigate();
  const [query, setQuery] = React.useState('');
  const [error, setError] = React.useState('');
  const defaults = ['claude-haiku-5-5', 'deepseek-v4-flash', 'gemini-3.7-flash', 'gpt-5.6-luna']
    .map(id => getPricedModel(id))
    .filter((m): m is PricedModel => Boolean(m));
  const [picked, setPicked] = React.useState<string[]>(defaults.map(m => m.id));
  const chosen = picked.map(id => PRICED_MODELS.find(m => m.id === id)).filter((m): m is PricedModel => Boolean(m));
  const providers = Array.from(new Map(PRICED_MODELS.map(m => [m.providerSlug, m.provider])).entries()).sort((a, b) => a[1].localeCompare(b[1]));
  const byPrice = [...PRICED_MODELS].sort((a, b) => blendedPrice(a) - blendedPrice(b));

  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    const pair = parsePairQuery(query);
    if (pair) return navigate(`/llm-pricing/compare/${pairSlug(pair[0], pair[1])}`);
    const single = resolveModel(query);
    if (single) return navigate(`/llm-pricing/${single.slug}`);
    setError('No match. Try "haiku 5.5 vs deepseek flash" or a model name like "gpt-5.6".');
  };

  const faq = [
    { q: 'How do I compare LLM API prices?', a: 'Type two models such as "haiku 5.5 vs deepseek flash", or use the calculator with your own requests, token counts and cache hit rate. Costs include long-context tiers and peak/off-peak rates where they exist.' },
    { q: 'Do price tiers change which model is cheaper?', a: 'Yes. Some models charge more once a prompt crosses a token threshold (for example 100K tokens), and some charge less off-peak. The comparison pages show the winner for ten realistic workloads.' },
    { q: 'Where do the benchmark numbers come from?', a: `Intelligence Index, speed and blended prices come from Artificial Analysis, crawled ${artificialAnalysisSnapshot.crawledAt.slice(0, 10)}.` },
  ];

  return (
    <>
      <Seo title={hub.title} description={hub.description} path={hub.path} jsonLd={faqLd(faq)} />
      <Shell>
        <Hero
          eyebrow="LLM pricing"
          title="LLM API pricing calculator"
          intro={<>Ask a pricing question and get tier-aware numbers: long-context surcharges, cached-token discounts, peak and off-peak rates, and Artificial Analysis benchmarks. Updated {PRICING_UPDATED}.</>}
        />

        <form onSubmit={submit} className={`${card} mb-8 p-4`}>
          <label htmlFor="pricing-q" className={label}>Ask: &quot;haiku 5.5 vs deepseek flash pricing&quot;</label>
          <div className="mt-2 flex flex-col gap-2 sm:flex-row">
            <input
              id="pricing-q"
              value={query}
              onChange={e => { setQuery(e.target.value); setError(''); }}
              placeholder="new anthropic haiku 5.5 vs deepseek flash pricing?"
              className="h-12 flex-1 rounded-lg border border-white/20 bg-black px-4 text-white outline-none focus:border-cyan-300"
            />
            <button type="submit" className="inline-flex h-12 items-center justify-center gap-2 rounded-lg bg-white px-6 font-mono text-sm font-semibold text-black">
              <Search className="h-4 w-4" /> Compare
            </button>
          </div>
          {error && <p className="mt-2 text-sm text-amber-300">{error}</p>}
        </form>

        <Section title="Popular comparisons">
          <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
            {popularPairModels().map(([a, b]) => <PairLink key={pairSlug(a, b)} a={a} b={b} />)}
          </div>
        </Section>

        <Calculator
          models={chosen}
          extra={
            <div className="mt-4 flex flex-wrap items-center gap-2">
              <select
                aria-label="Add model"
                value=""
                onChange={e => { if (e.target.value && !picked.includes(e.target.value) && picked.length < 6) setPicked([...picked, e.target.value]); }}
                className="h-10 rounded-lg border border-white/20 bg-black px-3 font-mono text-sm text-white outline-none focus:border-cyan-300"
              >
                <option value="">Add a model...</option>
                {byPrice.filter(m => !picked.includes(m.id)).map(m => <option key={m.id} value={m.id}>{m.name} ({m.provider})</option>)}
              </select>
              {chosen.map(m => (
                <button key={m.id} type="button" onClick={() => setPicked(picked.filter(id => id !== m.id))} className="rounded-lg border border-white/20 px-3 py-2 font-mono text-xs text-white/70 hover:border-white/50">
                  {m.name} x
                </button>
              ))}
            </div>
          }
        />

        <Section title="Cost by use case">
          <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
            {SCENARIOS.map(s => (
              <Link key={s.slug} to={`/llm-pricing/cost/${s.slug}`} className="rounded-lg border border-white/20 bg-white/[0.06] p-4 hover:border-white/50">
                <div className="font-medium text-white">Cheapest LLM for {s.headline}</div>
                <div className="mt-1 text-sm text-white/55">{s.description}</div>
              </Link>
            ))}
          </div>
        </Section>

        <Section title="By provider">
          <div className="flex flex-wrap gap-2">
            {providers.map(([slug, name]) => (
              <Link key={slug} to={`/llm-pricing/provider/${slug}`} className="rounded-lg border border-white/20 px-3 py-2 font-mono text-sm text-white/75 hover:border-white/50 hover:text-white">{name} pricing</Link>
            ))}
          </div>
        </Section>

        <Section title={`All ${PRICED_MODELS.length} models, cheapest first`}>
          <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
            {byPrice.map(m => <ModelLink key={m.id} m={m} />)}
          </div>
        </Section>

        <Faq items={faq} />
        <div className="text-sm text-white/55">
          Looking for the older token calculator? <Link to="/calculator" className={linkCls}>LLM cost calculator</Link>. Benchmarks: <Link to="/evals" className={linkCls}>evals</Link> and <Link to="/compare" className={linkCls}>model compare</Link>.
        </div>
      </Shell>
    </>
  );
}

function Missing({ what }: { what: string }) {
  return (
    <>
      <Seo title="Pricing page not found | OpenPaths" description={`No pricing data for ${what}.`} path="/llm-pricing" noindex />
      <Shell>
        <h1 className="text-3xl font-semibold">No pricing data for {what}</h1>
        <p className="mt-4 text-white/60">Try the <Link to="/llm-pricing" className={linkCls}>LLM pricing calculator</Link> to search all models.</p>
      </Shell>
    </>
  );
}

export function LlmPricingModel() {
  const { slug = '' } = useParams();
  const model = getPricedModel(slug) || resolveModel(slug);
  if (!model) return <Missing what={slug} />;
  const nb = neighbours(model, 3);
  const faq = modelFaq(model);
  const v = blendedPrice(model);
  const peers = [model, ...nb.cheaper.slice(0, 2)];
  const seo = modelSeo(model);
  return (
    <>
      <Seo title={seo.title} description={seo.description} path={seo.path} jsonLd={faqLd(faq)} breadcrumbName={`${model.name} pricing`} />
      <Shell>
        <Crumbs items={[{ to: '/llm-pricing', label: 'LLM pricing' }, { label: model.name }]} />
        <Hero
          eyebrow={`${model.provider} pricing`}
          title={`${model.name} API pricing`}
          intro={<>{model.name} costs <strong className="text-white">{priceLine(model)}</strong>. Blended 3:1 price is {fmtRate(v)} per 1M tokens. {model.description}</>}
        />
        <ShareButton className="mb-8" title={`${model.name} pricing | OpenPaths`} text={`${model.name}: ${priceLine(model)}`} path={`/llm-pricing/${model.slug}`} />
        <Cta model={model} />
        <Section title="Rate card"><RateCard models={[model]} /></Section>
        <Section title="Tiers and discounts"><TierNotes models={[model]} /></Section>
        <Calculator models={peers} />
        <Section title="Monthly cost by workload">
          <div className="overflow-x-auto" tabIndex={0} role="region" aria-label="Scenario costs">
            <table className="w-full min-w-[640px] text-left text-sm">
              <thead className="font-mono text-xs uppercase tracking-[0.14em] text-white/50">
                <tr><th className="px-3 py-3">Workload</th><th className="px-3 py-3 text-right">Requests</th><th className="px-3 py-3 text-right">In / out tokens</th><th className="px-3 py-3 text-right">Monthly</th></tr>
              </thead>
              <tbody className="divide-y divide-white/10">
                {SCENARIOS.map(s => (
                  <tr key={s.slug}>
                    <td className="px-3 py-3"><Link to={`/llm-pricing/cost/${s.slug}`} className="text-white hover:text-cyan-200">{s.name}</Link></td>
                    <td className="px-3 py-3 text-right font-mono">{fmtInt(s.workload.requests)}</td>
                    <td className="px-3 py-3 text-right font-mono">{fmtTokens(s.workload.inputTokens)} / {fmtTokens(s.workload.outputTokens)}</td>
                    <td className="px-3 py-3 text-right font-mono font-semibold">{fmtMoney(costFor(model, s.workload).total)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Section>
        <AaPanel models={peers} />
        <Section title="Compare with">
          <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
            {[...nb.cheaper, ...nb.pricier].map(other => <PairLink key={other.id} a={model} b={other} />)}
          </div>
          <h3 className="mb-3 mt-6 font-medium text-white">Nearby prices</h3>
          <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
            {[...nb.cheaper, ...nb.pricier].map(other => <ModelLink key={other.id} m={other} />)}
          </div>
        </Section>
        <Faq items={faq} />
        <div className="text-sm text-white/55">
          <Link to={`/llm-pricing/provider/${model.providerSlug}`} className={linkCls}>All {model.provider} pricing</Link> · <Link to="/llm-pricing" className={linkCls}>LLM pricing calculator</Link> · <Link to={`/models/${encodeURIComponent(model.id)}`} className={linkCls}>{model.name} on OpenPaths</Link>
        </div>
      </Shell>
    </>
  );
}

export function LlmPricingCompare() {
  const { pair = '' } = useParams();
  const parsed = parsePairSlug(pair);
  if (!parsed) return <Missing what={pair} />;
  const [a, b] = parsed;
  const verdict = pairVerdict(a, b);
  const results = compareScenarios(a, b);
  const faq = pairFaq(a, b);
  const path = `/llm-pricing/compare/${pairSlug(a, b)}`;
  const seo = pairSeo(a, b);
  return (
    <>
      <Seo title={seo.title} description={seo.description} path={path} jsonLd={faqLd(faq)} breadcrumbName={`${a.name} vs ${b.name}`} />
      <Shell>
        <Crumbs items={[{ to: '/llm-pricing', label: 'LLM pricing' }, { label: `${a.name} vs ${b.name}` }]} />
        <Hero
          eyebrow="Pricing comparison"
          title={`${a.name} vs ${b.name} pricing`}
          intro={<>{verdict.headline}. {a.name} is {priceLine(a)}; {b.name} is {priceLine(b)}.</>}
        />
        <ShareButton className="mb-8" title={`${a.name} vs ${b.name} pricing | OpenPaths`} text={verdict.headline} path={path} />
        <Section title="Verdict">
          <ul className="list-disc space-y-2 pl-5 text-white/70">
            {verdict.bullets.map(bullet => <li key={bullet}>{bullet}</li>)}
          </ul>
        </Section>
        <Section title="Rate card"><RateCard models={[a, b]} /></Section>
        <Section title="Price tiers explained"><TierNotes models={[a, b]} /></Section>
        <Calculator models={[a, b]} />
        <Section title="Who is cheaper, by workload">
          <div className="overflow-x-auto" tabIndex={0} role="region" aria-label="Workload winners">
            <table className="w-full min-w-[720px] text-left text-sm">
              <thead className="font-mono text-xs uppercase tracking-[0.14em] text-white/50">
                <tr>
                  <th className="px-3 py-3">Workload</th>
                  <th className="px-3 py-3 text-right">{a.name}</th>
                  <th className="px-3 py-3 text-right">{b.name}</th>
                  <th className="px-3 py-3">Cheaper</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/10">
                {results.map(r => (
                  <tr key={r.scenario.slug}>
                    <td className="px-3 py-3"><Link to={`/llm-pricing/cost/${r.scenario.slug}`} className="text-white hover:text-cyan-200">{r.scenario.name}</Link></td>
                    <td className={`px-3 py-3 text-right font-mono ${r.winner === 'a' ? 'text-cyan-200' : ''}`}>{fmtMoney(r.a.total)}{r.a.rates.tier === 'long-context' ? ' *' : ''}</td>
                    <td className={`px-3 py-3 text-right font-mono ${r.winner === 'b' ? 'text-cyan-200' : ''}`}>{fmtMoney(r.b.total)}{r.b.rates.tier === 'long-context' ? ' *' : ''}</td>
                    <td className="px-3 py-3 font-mono text-xs">{r.winner === 'tie' ? 'Tie' : `${r.winner === 'a' ? a.name : b.name} (${formatRatio(r.ratio)})`}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p className="mt-3 text-xs text-white/45">* long-context tier applied. Monthly totals for the stated request volume per workload.</p>
        </Section>
        <AaPanel models={[a, b]} />
        <Cta model={a} />
        <Section title="More comparisons">
          <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
            {popularPairModels().filter(([x, y]) => pairSlug(x, y) !== pairSlug(a, b)).slice(0, 6).map(([x, y]) => <PairLink key={pairSlug(x, y)} a={x} b={y} />)}
          </div>
          <div className="mt-4 flex flex-wrap gap-4 text-sm">
            <Link to={`/llm-pricing/${a.slug}`} className={linkCls}>{a.name} pricing</Link>
            <Link to={`/llm-pricing/${b.slug}`} className={linkCls}>{b.name} pricing</Link>
            <Link to="/llm-pricing" className={linkCls}>All LLM pricing</Link>
          </div>
        </Section>
        <Faq items={faq} />
      </Shell>
    </>
  );
}

export function LlmPricingProvider() {
  const { provider = '' } = useParams();
  const models = PRICED_MODELS.filter(m => m.providerSlug === provider).sort((a, b) => blendedPrice(a) - blendedPrice(b));
  if (models.length === 0) return <Missing what={provider} />;
  const name = models[0].provider;
  const seo = providerSeo(provider, models);
  const faq = [
    { q: `What is the cheapest ${name} model?`, a: `${models[0].name} at ${priceLine(models[0])}.` },
    { q: `What is the most expensive ${name} model?`, a: `${models[models.length - 1].name} at ${priceLine(models[models.length - 1])}.` },
    { q: `Can I use ${name} models without a ${name} account?`, a: `Yes. OpenPaths routes to ${name} models through one OpenAI-compatible API with pay-as-you-go billing.` },
  ];
  return (
    <>
      <Seo title={seo.title} description={seo.description} path={seo.path} jsonLd={faqLd(faq)} breadcrumbName={`${name} pricing`} />
      <Shell>
        <Crumbs items={[{ to: '/llm-pricing', label: 'LLM pricing' }, { label: name }]} />
        <Hero eyebrow="Provider pricing" title={`${name} API pricing`} intro={<>All {models.length} {name} models on OpenPaths, cheapest first. Click a model for tiers, cached rates and a cost calculator.</>} />
        <Calculator models={models.slice(0, 6)} />
        <Section title="Price list">
          <div className="overflow-x-auto" tabIndex={0} role="region" aria-label="Price list">
            <table className="w-full min-w-[640px] text-left text-sm">
              <thead className="font-mono text-xs uppercase tracking-[0.14em] text-white/50">
                <tr><th className="px-3 py-3">Model</th><th className="px-3 py-3 text-right">Input / 1M</th><th className="px-3 py-3 text-right">Output / 1M</th><th className="px-3 py-3 text-right">Cached / 1M</th><th className="px-3 py-3 text-right">Context</th></tr>
              </thead>
              <tbody className="divide-y divide-white/10">
                {models.map(m => (
                  <tr key={m.id}>
                    <td className="px-3 py-3"><Link to={`/llm-pricing/${m.slug}`} className="text-white hover:text-cyan-200">{m.name}</Link></td>
                    <td className="px-3 py-3 text-right font-mono">{m.schedule ? `${fmtRate(m.schedule.offPeak.input)}-${fmtRate(m.schedule.peak.input)}` : fmtRate(m.input)}</td>
                    <td className="px-3 py-3 text-right font-mono">{m.schedule ? `${fmtRate(m.schedule.offPeak.output)}-${fmtRate(m.schedule.peak.output)}` : fmtRate(m.output)}</td>
                    <td className="px-3 py-3 text-right font-mono">{fmtRate(m.cacheRead)}</td>
                    <td className="px-3 py-3 text-right font-mono">{fmtTokens(m.context)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Section>
        <AaPanel models={models.slice(0, 5)} />
        <Faq items={faq} />
        <div className="text-sm text-white/55"><Link to={`/providers/${provider}`} className={linkCls}>{name} on OpenPaths</Link> · <Link to="/llm-pricing" className={linkCls}>All LLM pricing</Link></div>
      </Shell>
    </>
  );
}

export function LlmPricingScenario() {
  const { scenario = '' } = useParams();
  const sc = getScenario(scenario);
  if (!sc) return <Missing what={scenario} />;
  const ranking = rankForScenario(sc.workload, 15);
  const w = sc.workload;
  const seo = scenarioSeo(sc);
  const faq = [
    { q: `What is the cheapest LLM API for ${sc.headline}?`, a: `${ranking[0].model.name} at about ${fmtMoney(ranking[0].cost.total)} per month for ${fmtInt(w.requests)} requests (${fmtInt(w.inputTokens)} input and ${fmtInt(w.outputTokens)} output tokens each, ${Math.round(w.cacheHit * 100)}% of input cached).` },
    { q: `How is the ${sc.headline} cost estimated?`, a: `Cost = requests x tokens x published per-token rates, with cached-input discounts, long-context tiers and peak/off-peak mixes applied. ${sc.description}` },
    { q: 'Is the cheapest model also the best?', a: 'Not necessarily. Check the Artificial Analysis Intelligence Index on each model page and test your own prompts before switching.' },
  ];
  return (
    <>
      <Seo title={seo.title} description={seo.description} path={seo.path} jsonLd={faqLd(faq)} breadcrumbName={sc.name} />
      <Shell>
        <Crumbs items={[{ to: '/llm-pricing', label: 'LLM pricing' }, { label: sc.name }]} />
        <Hero eyebrow="Cost by use case" title={`Cheapest LLM API for ${sc.headline}`} intro={<>{sc.description} Assumes {fmtInt(w.requests)} requests per month, {fmtInt(w.inputTokens)} input and {fmtInt(w.outputTokens)} output tokens per request, {Math.round(w.cacheHit * 100)}% of input cached.</>} />
        <Section title="Ranking, cheapest first" right={<Link to={`/llm-pricing?r=${w.requests}&i=${w.inputTokens}&o=${w.outputTokens}&c=${Math.round(w.cacheHit * 100)}`} className={`${linkCls} font-mono text-xs`}>Customize in calculator</Link>}>
          <CalcTable models={ranking.map(r => r.model)} workload={w} />
        </Section>
        <AaPanel models={ranking.slice(0, 4).map(r => r.model)} />
        <Section title="Other use cases">
          <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
            {SCENARIOS.filter(s => s.slug !== sc.slug).map(s => (
              <Link key={s.slug} to={`/llm-pricing/cost/${s.slug}`} className="rounded-lg border border-white/20 bg-white/[0.06] p-4 text-white hover:border-white/50">Cheapest LLM for {s.headline}</Link>
            ))}
          </div>
        </Section>
        <Faq items={faq} />
      </Shell>
    </>
  );
}
