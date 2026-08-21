import React from 'react';
import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, ArrowRight, ExternalLink, GitCompareArrows, Plus, Search } from 'lucide-react';
import { Seo } from '../components/Seo';
import { ArtificialAnalysisBenchmarkSection } from '../components/ArtificialAnalysisCharts';
import {
  EVALUATION_DEFINITIONS,
  artificialAnalysisModels,
  artificialAnalysisSnapshot,
  displayScore,
  findArtificialAnalysisModel,
  formatDollars,
  formatNumber,
  topArtificialAnalysisModels,
  type ArtificialAnalysisModel,
} from '../lib/artificialAnalysis';

export function CompareIndex() {
  const models = topArtificialAnalysisModels(12);
  const anchor = models[0];
  const popularPairs = models.slice(1, 10);

  return (
    <>
      <Seo
        title="Compare AI Models by Evals, Speed, and Price | OpenPaths"
        description="Compare frontier AI models head-to-head using Artificial Analysis evals, pricing, context windows, speed, and benchmark run costs."
        path="/compare"
      />
      <div className="mx-auto max-w-7xl px-6 py-12">
        <section className="mb-10">
          <div className="mb-4 inline-flex items-center gap-2 font-mono text-xs uppercase tracking-[0.22em] text-cyan-300">
            <GitCompareArrows className="h-4 w-4" />
            AI model comparison
          </div>
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <h1 className="max-w-4xl text-4xl font-semibold tracking-tight md:text-6xl">Compare AI models head-to-head</h1>
              <p className="mt-5 max-w-3xl text-lg leading-relaxed text-white/60">
                Pick two or more crawled models and compare intelligence, coding, agentic work, token pricing, speed, context, and benchmark run costs.
              </p>
            </div>
            <Link
              to="/evals"
              className="inline-flex items-center justify-center gap-2 rounded-lg border border-white/20 px-4 py-3 font-mono text-sm text-white/65 transition-colors hover:border-white/50 hover:text-white"
            >
              Browse evals <ArrowRight className="h-4 w-4" />
            </Link>
          </div>
        </section>

        <section className="mb-10 rounded-lg border border-white/20 bg-white/[0.05] p-5">
          <div className="mb-4 flex items-center gap-2 font-mono text-xs uppercase tracking-[0.16em] text-white/45">
            <Search className="h-4 w-4" />
            URL format
          </div>
          <p className="max-w-4xl text-white/60">
            Use <code>/compare/model-a-vs-model-b</code> or keep adding models with <code>-vs-model-c</code>. Slugs and common shorthand both work, including <Link to="/compare/gpt-5.5-vs-opus4.7" className="text-cyan-200 underline decoration-cyan-200/25 underline-offset-4">gpt-5.5-vs-opus4.7</Link>.
          </p>
        </section>

        {anchor && (
          <section className="mb-10">
            <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
              <h2 className="text-2xl font-semibold tracking-tight">Popular comparisons</h2>
              <span className="font-mono text-xs text-white/50">Based on the current Artificial Analysis snapshot</span>
            </div>
            <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
              {popularPairs.map(model => (
                <Link
                  key={model.slug}
                  to={`/compare/${anchor.slug}-vs-${model.slug}`}
                  className="group rounded-lg border border-white/20 bg-white/[0.06] p-4 transition-colors hover:border-white/50 hover:bg-white/[0.08]"
                >
                  <div className="mb-4 flex items-center justify-between gap-3">
                    <div>
                      <p className="font-mono text-xs text-white/50">{anchor.creator.name} vs {model.creator.name}</p>
                      <h3 className="mt-1 text-lg font-semibold tracking-tight text-white">{anchor.shortName} vs {model.shortName}</h3>
                    </div>
                    <ArrowRight className="h-4 w-4 flex-none text-white/45 transition-colors group-hover:text-white" />
                  </div>
                  <div className="grid grid-cols-2 gap-3 font-mono text-xs text-white/50">
                    <span>AA {displayScore(model.evaluations.intelligenceIndex)}</span>
                    <span>{formatDollars(model.prices.blendedPerMTokens, 3)} blend</span>
                  </div>
                </Link>
              ))}
            </div>
          </section>
        )}

        <ArtificialAnalysisBenchmarkSection compact />
      </div>
    </>
  );
}

export function Compare() {
  const params = useParams();
  const pair = params['*'] || '';
  const tokens = pair.split('-vs-').map(token => decodeURIComponent(token.trim())).filter(Boolean);
  const models = uniqueModels(tokens.map(token => findArtificialAnalysisModel(token)));

  if (models.length < 2 || models.length !== tokens.length) {
    return <CompareNotFound pair={pair} />;
  }

  const modelNames = models.map(model => model.shortName);
  const heading = modelNames.join(' vs ');
  const title = `${heading} | AI Model Comparison`;

  return (
    <>
      <Seo
        title={`${title} | OpenPaths`}
        description={`Compare ${formatNameList(modelNames)} using Artificial Analysis evals, speed, context, and token pricing.`}
        path={`/compare/${pair}`}
      />
      <div className="mx-auto max-w-7xl px-6 py-12">
        <Link to="/evals" className="mb-10 inline-flex items-center gap-2 font-mono text-sm text-white/55 transition-colors hover:text-white">
          <ArrowLeft className="h-4 w-4" />
          Evals
        </Link>

        <section className="mb-8">
          <div className="mb-4 inline-flex items-center gap-2 font-mono text-xs uppercase tracking-[0.22em] text-cyan-300">
            <GitCompareArrows className="h-4 w-4" />
            Model compare
          </div>
          <h1 className="max-w-5xl text-4xl font-semibold tracking-tight md:text-6xl">{heading}</h1>
          <p className="mt-5 max-w-3xl text-lg leading-relaxed text-white/60">
            Stored Artificial Analysis data for evals, pricing, context, speed, and Intelligence Index run costs.
          </p>
        </section>

        <AddModelCompare models={models} />

        <div className="mb-8 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {models.map(model => (
            <React.Fragment key={model.slug}>
              <ModelSummary model={model} />
            </React.Fragment>
          ))}
        </div>

        <section className="overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
          <div className="flex flex-wrap items-center justify-between gap-4 border-b border-white/20 px-4 py-3">
            <h2 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">Head-to-head breakdown</h2>
            <a
              href={artificialAnalysisSnapshot.sourceUrl}
              target="_blank"
              rel="noreferrer noopener"
              className="inline-flex items-center gap-1 font-mono text-xs text-white/55 hover:text-white"
            >
              Artificial Analysis <ExternalLink className="h-3.5 w-3.5" />
            </a>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[900px] text-left text-sm">
              <thead className="font-mono text-xs uppercase tracking-[0.14em] text-white/50">
                <tr>
                  <th className="px-4 py-3">Metric</th>
                  {models.map(model => (
                    <th key={model.slug} className="px-4 py-3">{model.shortName}</th>
                  ))}
                  <th className="px-4 py-3">Best</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/10">
                <MetricRow label="Blended price / 1M tokens" models={models} value={model => model.prices.blendedPerMTokens} format={value => formatDollars(value, 3)} lowerIsBetter />
                <MetricRow label="Input price / 1M tokens" models={models} value={model => model.prices.inputPerMTokens} format={value => formatDollars(value, 3)} lowerIsBetter />
                <MetricRow label="Output price / 1M tokens" models={models} value={model => model.prices.outputPerMTokens} format={value => formatDollars(value, 3)} lowerIsBetter />
                <MetricRow label="Output speed" models={models} value={model => model.performance.outputTokensPerSecond} format={value => `${formatNumber(value)} tok/s`} />
                <MetricRow label="Context window" models={models} value={model => model.contextWindowTokens} format={formatNumber} />
                <MetricRow label="AA Index run cost" models={models} value={model => model.costs.intelligenceIndexTotal} format={value => formatDollars(value, 2)} lowerIsBetter />
                {EVALUATION_DEFINITIONS.map(definition => (
                  <React.Fragment key={definition.key}>
                    <MetricRow
                      label={definition.label}
                      models={models}
                      value={model => model.evaluations[definition.key]}
                      format={value => displayScore(value, definition.kind)}
                      lowerIsBetter={!definition.higherIsBetter}
                    />
                  </React.Fragment>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </>
  );
}

function AddModelCompare({ models }: { models: ArtificialAnalysisModel[] }) {
  const [selectedSlug, setSelectedSlug] = React.useState('');
  const selectedSlugs = new Set(models.map(model => model.slug));
  const availableModels = artificialAnalysisModels.filter(model => !selectedSlugs.has(model.slug));
  const effectiveSelectedSlug = availableModels.some(model => model.slug === selectedSlug)
    ? selectedSlug
    : availableModels[0]?.slug || '';
  const nextPath = effectiveSelectedSlug
    ? `/compare/${[...models.map(model => model.slug), effectiveSelectedSlug].join('-vs-')}`
    : '/compare';

  if (availableModels.length === 0) return null;

  return (
    <section className="mb-8 rounded-lg border border-white/20 bg-white/[0.05] p-4">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          <h2 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">Add another model</h2>
          <p className="mt-1 text-sm text-white/45">Build a three, four, or n-way comparison from the stored Artificial Analysis snapshot.</p>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <select
            value={effectiveSelectedSlug}
            onChange={event => setSelectedSlug(event.target.value)}
            className="h-11 min-w-[260px] rounded-lg border border-white/20 bg-black px-3 font-mono text-sm text-white outline-none transition-colors hover:border-white/45 focus:border-cyan-300"
          >
            {availableModels.map(model => (
              <option key={model.slug} value={model.slug}>
                {model.shortName} - {model.creator.name}
              </option>
            ))}
          </select>
          <Link
            to={nextPath}
            className="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-cyan-300 px-4 font-mono text-sm font-semibold text-black transition-colors hover:bg-cyan-200"
          >
            <Plus className="h-4 w-4" />
            Compare
          </Link>
        </div>
      </div>
    </section>
  );
}

function ModelSummary({ model }: { model: ArtificialAnalysisModel }) {
  return (
    <div className="rounded-lg border border-white/20 bg-white/[0.06] p-5">
      <div className="mb-5 flex items-start justify-between gap-4">
        <div>
          <p className="font-mono text-xs uppercase tracking-[0.16em] text-white/50">{model.creator.name}</p>
          <h2 className="mt-2 text-2xl font-semibold tracking-tight">{model.shortName}</h2>
        </div>
        <span className="rounded-md border border-white/20 px-2 py-1 font-mono text-xs text-white/45">
          {model.reasoning ? 'Reasoning' : 'Non-reasoning'}
        </span>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <Fact label="AA Index" value={displayScore(model.evaluations.intelligenceIndex)} />
        <Fact label="Coding" value={displayScore(model.evaluations.codingIndex)} />
        <Fact label="Agentic" value={displayScore(model.evaluations.agenticIndex)} />
        <Fact label="GDPval" value={displayScore(model.evaluations.gdpvalElo, 'elo')} />
        <Fact label="Blend price" value={formatDollars(model.prices.blendedPerMTokens, 3)} />
        <Fact label="Output speed" value={`${formatNumber(model.performance.outputTokensPerSecond)} tok/s`} />
      </div>
    </div>
  );
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-md border border-white/20 bg-black/20 p-3">
      <p className="font-mono text-[10px] uppercase tracking-[0.14em] text-white/50">{label}</p>
      <p className="mt-1 font-mono text-lg text-white">{value}</p>
    </div>
  );
}

function MetricRow({
  label,
  models,
  value,
  format,
  lowerIsBetter = false,
}: {
  label: string;
  models: ArtificialAnalysisModel[];
  value: (model: ArtificialAnalysisModel) => number | null;
  format: (value: number | null) => string;
  lowerIsBetter?: boolean;
}) {
  const values = models.map(model => ({ model, value: value(model) }));
  const winners = getWinners(values, lowerIsBetter);
  return (
    <tr className="text-white/70">
      <td className="px-4 py-3 font-mono text-white/85">{label}</td>
      {values.map(item => (
        <td key={item.model.slug} className={`px-4 py-3 font-mono ${winners.has(item.model.slug) ? 'text-cyan-200' : ''}`}>
          {format(item.value)}
        </td>
      ))}
      <td className="px-4 py-3 font-mono text-white/45">
        {winners.size === 0 ? 'N/A' : formatNameList(values.filter(item => winners.has(item.model.slug)).map(item => item.model.shortName))}
      </td>
    </tr>
  );
}

function getWinners(values: Array<{ model: ArtificialAnalysisModel; value: number | null }>, lowerIsBetter: boolean) {
  const validValues = values.filter(item => typeof item.value === 'number' && Number.isFinite(item.value));
  if (validValues.length === 0) return new Set<string>();
  const bestValue = lowerIsBetter
    ? Math.min(...validValues.map(item => item.value as number))
    : Math.max(...validValues.map(item => item.value as number));
  return new Set(validValues.filter(item => item.value === bestValue).map(item => item.model.slug));
}

function uniqueModels(models: Array<ArtificialAnalysisModel | undefined>) {
  const seen = new Set<string>();
  return models.filter((model): model is ArtificialAnalysisModel => {
    if (!model || seen.has(model.slug)) return false;
    seen.add(model.slug);
    return true;
  });
}

function formatNameList(names: string[]) {
  if (names.length <= 2) return names.join(' and ');
  return `${names.slice(0, -1).join(', ')}, and ${names[names.length - 1]}`;
}

function CompareNotFound({ pair }: { pair: string }) {
  const examples = topArtificialAnalysisModels(8);
  return (
    <div className="mx-auto max-w-5xl px-6 py-16">
      <Seo title="Model comparison not found | OpenPaths" description="Choose two crawled Artificial Analysis models to compare." path={`/compare/${pair}`} />
      <Link to="/evals" className="mb-10 inline-flex items-center gap-2 font-mono text-sm text-white/55 transition-colors hover:text-white">
        <ArrowLeft className="h-4 w-4" />
        Evals
      </Link>
      <h1 className="text-4xl font-semibold tracking-tight">Comparison not found</h1>
      <p className="mt-4 max-w-2xl text-white/60">
        Use the form `model-a-vs-model-b` or keep adding models with `-vs-model-c`. Shorthand works when it uniquely matches stored data, for example `gpt5.5-vs-opus4.7`.
      </p>
      <div className="mt-8 flex flex-wrap gap-3">
        {examples.slice(1).map(model => (
          <Link
            key={model.slug}
            to={`/compare/${examples[0].slug}-vs-${model.slug}`}
            className="rounded-lg border border-white/20 px-3 py-2 font-mono text-sm text-white/60 hover:border-white/50 hover:text-white"
          >
            {examples[0].shortName} vs {model.shortName}
          </Link>
        ))}
      </div>
    </div>
  );
}
