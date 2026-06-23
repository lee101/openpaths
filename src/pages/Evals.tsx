import React from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, Database, ExternalLink } from 'lucide-react';
import { Seo } from '../components/Seo';
import { ArtificialAnalysisBenchmarkSection } from '../components/ArtificialAnalysisCharts';
import {
  EVALUATION_DEFINITIONS,
  artificialAnalysisModels,
  artificialAnalysisSnapshot,
  displayScore,
  formatDollars,
  formatNumber,
  topArtificialAnalysisModels,
} from '../lib/artificialAnalysis';

const TABLE_EVALS = EVALUATION_DEFINITIONS.filter(definition => (
  ['intelligenceIndex', 'codingIndex', 'agenticIndex', 'gdpvalElo', 'gpqa', 'hle', 'terminalBenchHard', 'lcr'].includes(definition.key)
));

export function Evals() {
  const leaders = topArtificialAnalysisModels(10);

  return (
    <>
      <Seo
        title="AI Model Evals, Pricing, and Speed | OpenPaths"
        description="Artificial Analysis benchmark snapshot for comparing frontier AI model intelligence, coding, agentic performance, speed, and token pricing."
        path="/evals"
      />
      <div className="mx-auto max-w-7xl px-6 py-12">
        <section className="mb-10">
          <div className="mb-4 inline-flex items-center gap-2 font-mono text-xs uppercase tracking-[0.22em] text-cyan-300">
            <Database className="h-4 w-4" />
            Model evals
          </div>
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <h1 className="max-w-4xl text-4xl font-semibold tracking-tight md:text-6xl">Frontier model benchmark map</h1>
              <p className="mt-5 max-w-3xl text-lg leading-relaxed text-white/60">
                A local snapshot of Artificial Analysis model scores, prices, token usage, and latency metrics, normalized for OpenPaths comparisons.
              </p>
              <Link to="/image-evals" className="mt-3 inline-flex items-center gap-1 font-mono text-sm text-cyan-200 hover:text-white">
                Looking for image generators? See the Image Eval leaderboard <ArrowRight className="h-3.5 w-3.5" />
              </Link>
            </div>
            <a
              href={artificialAnalysisSnapshot.sourceUrl}
              target="_blank"
              rel="noreferrer noopener"
              className="inline-flex items-center justify-center gap-2 rounded-lg border border-white/10 px-4 py-3 font-mono text-sm text-white/65 transition-colors hover:border-white/30 hover:text-white"
            >
              Source <ExternalLink className="h-4 w-4" />
            </a>
          </div>
        </section>

        <ArtificialAnalysisBenchmarkSection />

        <section className="mt-10 overflow-hidden rounded-lg border border-white/10 bg-white/[0.02]">
          <div className="flex flex-wrap items-center justify-between gap-4 border-b border-white/10 px-4 py-3">
            <h2 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">Eval Breakdown</h2>
            <span className="font-mono text-xs text-white/35">{artificialAnalysisModels.length} crawled models</span>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[980px] text-left text-sm">
              <thead className="font-mono text-xs uppercase tracking-[0.14em] text-white/35">
                <tr>
                  <th className="px-4 py-3">Model</th>
                  <th className="px-4 py-3">Price</th>
                  <th className="px-4 py-3">Speed</th>
                  {TABLE_EVALS.map(definition => (
                    <th key={definition.key} className="px-4 py-3">{definition.shortLabel}</th>
                  ))}
                  <th className="px-4 py-3">Compare</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/10">
                {leaders.map(model => (
                  <tr key={model.id} className="text-white/70">
                    <td className="px-4 py-4">
                      <div className="font-mono text-white">{model.shortName}</div>
                      <div className="text-xs text-white/40">{model.creator.name}</div>
                    </td>
                    <td className="px-4 py-4 font-mono">{formatDollars(model.prices.blendedPerMTokens, 3)}</td>
                    <td className="px-4 py-4 font-mono">{formatNumber(model.performance.outputTokensPerSecond)} tok/s</td>
                    {TABLE_EVALS.map(definition => (
                      <td key={definition.key} className="px-4 py-4 font-mono">
                        {displayScore(model.evaluations[definition.key], definition.kind)}
                      </td>
                    ))}
                    <td className="px-4 py-4">
                      <Link
                        to={`/compare/${model.slug}-vs-${leaders[0].slug === model.slug ? leaders[1]?.slug : leaders[0].slug}`}
                        className="inline-flex items-center gap-1 font-mono text-xs text-cyan-200 hover:text-white"
                      >
                        Open <ArrowRight className="h-3.5 w-3.5" />
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </>
  );
}
