import React from 'react';
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Scatter, ScatterChart, Tooltip, XAxis, YAxis, ZAxis } from 'recharts';
import { BarChart3, CircleDollarSign, Gauge, Trophy } from 'lucide-react';
import {
  artificialAnalysisSnapshot,
  displayScore,
  formatDollars,
  formatNumber,
  modelColor,
  topArtificialAnalysisModels,
  type ArtificialAnalysisModel,
} from '../lib/artificialAnalysis';

type BenchmarkSectionProps = {
  compact?: boolean;
};

type ChartPoint = {
  name: string;
  creator: string;
  color: string;
  intelligence: number;
  speed: number;
  price: number;
};

export function ArtificialAnalysisBenchmarkSection({ compact = false }: BenchmarkSectionProps) {
  const models = topArtificialAnalysisModels(compact ? 8 : 12);
  const scatterData = models
    .filter(model => model.evaluations.intelligenceIndex != null && model.performance.outputTokensPerSecond != null && model.prices.blendedPerMTokens != null)
    .map(model => ({
      name: model.shortName,
      creator: model.creator.name,
      color: modelColor(model),
      intelligence: model.evaluations.intelligenceIndex ?? 0,
      speed: model.performance.outputTokensPerSecond ?? 0,
      price: model.prices.blendedPerMTokens ?? 0,
    }));

  return (
    <section className={compact ? 'mt-10' : ''}>
      <div className="mb-5 flex flex-col gap-2 md:flex-row md:items-end md:justify-between">
        <div>
          <div className="mb-2 flex items-center gap-2 font-mono text-xs uppercase tracking-[0.18em] text-cyan-300">
            <BarChart3 className="h-4 w-4" />
            External benchmarks
          </div>
          <h2 className={compact ? 'text-2xl font-semibold tracking-tight' : 'text-3xl font-semibold tracking-tight'}>
            Artificial Analysis model data
          </h2>
        </div>
        <a
          href={artificialAnalysisSnapshot.sourceUrl}
          target="_blank"
          rel="noreferrer noopener"
          className="font-mono text-xs text-white/55 underline decoration-white/15 underline-offset-4 hover:text-white"
        >
          Crawled {new Date(artificialAnalysisSnapshot.crawledAt).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC' })}
        </a>
      </div>

      {!compact && (
        <div className="mb-6 grid gap-4 md:grid-cols-4">
          <BenchmarkMetric icon={<Trophy className="h-5 w-5" />} label="Models" value={String(artificialAnalysisSnapshot.modelCount)} />
          <BenchmarkMetric icon={<Gauge className="h-5 w-5" />} label="Top AA Index" value={displayScore(models[0]?.evaluations.intelligenceIndex)} />
          <BenchmarkMetric icon={<CircleDollarSign className="h-5 w-5" />} label="Lowest Blend" value={formatDollars(lowestBlend(models), 3)} />
          <BenchmarkMetric icon={<BarChart3 className="h-5 w-5" />} label="Fastest Output" value={`${formatNumber(fastest(models))} tok/s`} />
        </div>
      )}

      <div className="grid gap-5 lg:grid-cols-2">
        <div className="overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
          <div className="border-b border-white/20 px-4 py-3">
            <h3 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">Intelligence Index</h3>
          </div>
          <div className="h-[340px] p-3">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={models.map(toBarModel)} layout="vertical" margin={{ top: 8, right: 22, bottom: 8, left: 12 }}>
                <CartesianGrid stroke="rgba(255,255,255,0.08)" horizontal={false} />
                <XAxis type="number" tick={{ fill: 'rgba(255,255,255,0.45)', fontSize: 12, fontFamily: 'monospace' }} axisLine={false} tickLine={false} />
                <YAxis
                  dataKey="name"
                  type="category"
                  width={compact ? 112 : 150}
                  tick={{ fill: 'rgba(255,255,255,0.58)', fontSize: 11, fontFamily: 'monospace' }}
                  axisLine={false}
                  tickLine={false}
                />
                <Tooltip content={<BarTooltip />} cursor={{ fill: 'rgba(255,255,255,0.04)' }} />
                <Bar dataKey="intelligence" fill="#22d3ee" radius={[0, 4, 4, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
          <div className="border-b border-white/20 px-4 py-3">
            <h3 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">Intelligence vs. Price</h3>
          </div>
          <div className="h-[340px] p-3">
            <ResponsiveContainer width="100%" height="100%">
              <ScatterChart margin={{ top: 14, right: 16, bottom: 16, left: 8 }}>
                <CartesianGrid stroke="rgba(255,255,255,0.08)" />
                <XAxis
                  type="number"
                  dataKey="price"
                  name="Blended price"
                  tick={{ fill: 'rgba(255,255,255,0.45)', fontSize: 12, fontFamily: 'monospace' }}
                  axisLine={{ stroke: 'rgba(255,255,255,0.12)' }}
                  tickLine={false}
                  label={{ value: 'USD / 1M tokens', position: 'insideBottom', offset: -10, fill: 'rgba(255,255,255,0.45)', fontSize: 12 }}
                />
                <YAxis
                  type="number"
                  dataKey="intelligence"
                  name="AA Index"
                  tick={{ fill: 'rgba(255,255,255,0.45)', fontSize: 12, fontFamily: 'monospace' }}
                  axisLine={false}
                  tickLine={false}
                  width={42}
                />
                <ZAxis type="number" dataKey="speed" range={[70, 430]} />
                <Tooltip content={<ScatterTooltip />} cursor={{ stroke: 'rgba(255,255,255,0.18)' }} />
                <Scatter data={scatterData} fill="#a78bfa" shape={<BenchmarkDot />} />
              </ScatterChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>
    </section>
  );
}

function BenchmarkMetric({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="rounded-lg border border-white/20 bg-white/[0.06] p-4">
      <div className="mb-3 flex items-center justify-between text-white/45">
        <p className="font-mono text-xs uppercase tracking-[0.16em]">{label}</p>
        {icon}
      </div>
      <p className="font-mono text-2xl text-white">{value}</p>
    </div>
  );
}

function toBarModel(model: ArtificialAnalysisModel) {
  return {
    name: model.shortName,
    intelligence: model.evaluations.intelligenceIndex ?? 0,
    creator: model.creator.name,
  };
}

function lowestBlend(models: ArtificialAnalysisModel[]) {
  const values = models
    .map(model => model.prices.blendedPerMTokens)
    .filter(value => typeof value === 'number' && value > 0)
    .map(Number);
  return values.length ? Math.min(...values) : null;
}

function fastest(models: ArtificialAnalysisModel[]) {
  const values = models
    .map(model => model.performance.outputTokensPerSecond)
    .filter(value => typeof value === 'number' && value > 0)
    .map(Number);
  return values.length ? Math.max(...values) : null;
}

function BarTooltip({ active, payload }: any) {
  if (!active || !payload?.length) return null;
  const item = payload[0].payload;
  return (
    <div className="rounded-lg border border-white/20 bg-black px-3 py-2 text-sm shadow-xl">
      <p className="font-mono text-white">{item.name}</p>
      <p className="text-white/55">{item.creator}</p>
      <p className="mt-1 font-mono text-cyan-200">{displayScore(item.intelligence)} AA Index</p>
    </div>
  );
}

function ScatterTooltip({ active, payload }: any) {
  if (!active || !payload?.length) return null;
  const item = payload[0].payload as ChartPoint;
  return (
    <div className="rounded-lg border border-white/20 bg-black px-3 py-2 text-sm shadow-xl">
      <p className="font-mono text-white">{item.name}</p>
      <p className="text-white/55">{item.creator}</p>
      <p className="mt-1 font-mono text-cyan-200">{displayScore(item.intelligence)} AA Index</p>
      <p className="font-mono text-white/60">{formatDollars(item.price, 3)} blended</p>
      <p className="font-mono text-white/60">{formatNumber(item.speed)} tok/s</p>
    </div>
  );
}

function BenchmarkDot(props: any) {
  const { cx, cy, payload } = props;
  return <circle cx={cx} cy={cy} r={5.5} fill={payload.color} stroke="rgba(255,255,255,0.9)" strokeWidth={1} />;
}
