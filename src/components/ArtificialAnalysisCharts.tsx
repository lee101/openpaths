import React from 'react';
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Scatter, ScatterChart, Tooltip, XAxis, YAxis, ZAxis } from 'recharts';
import { BarChart3, CircleDollarSign, Gauge, Trophy } from 'lucide-react';
import {
  artificialAnalysisModels,
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
  speed: number | null;
  bubbleSize: number;
  price: number;
};

const COMPACT_BAR_MODEL_COUNT = 12;
const FULL_BAR_MODEL_COUNT = 16;
const DEFAULT_BUBBLE_SPEED = 50;
const PRICE_TICK_CANDIDATES = [0.05, 0.1, 0.25, 0.5, 1, 2, 4, 8, 16, 32, 64];

export function ArtificialAnalysisBenchmarkSection({ compact = false }: BenchmarkSectionProps) {
  const models = topArtificialAnalysisModels(compact ? COMPACT_BAR_MODEL_COUNT : FULL_BAR_MODEL_COUNT);
  const scatterData: ChartPoint[] = artificialAnalysisModels
    .filter(model => (
      typeof model.evaluations.intelligenceIndex === 'number'
      && typeof model.prices.blendedPerMTokens === 'number'
      && model.prices.blendedPerMTokens > 0
    ))
    .map(model => ({
      name: model.shortName,
      creator: model.creator.name,
      color: modelColor(model),
      intelligence: model.evaluations.intelligenceIndex ?? 0,
      speed: model.performance.outputTokensPerSecond,
      bubbleSize: model.performance.outputTokensPerSecond ?? DEFAULT_BUBBLE_SPEED,
      price: model.prices.blendedPerMTokens ?? 0,
    }));
  const { domain: priceDomain, ticks: priceTicks } = getPriceScale(scatterData);

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
          <BenchmarkMetric icon={<CircleDollarSign className="h-5 w-5" />} label="Lowest Blend" value={formatDollars(lowestBlend(artificialAnalysisModels), 3)} />
          <BenchmarkMetric icon={<BarChart3 className="h-5 w-5" />} label="Fastest Output" value={`${formatNumber(fastest(artificialAnalysisModels))} tok/s`} />
        </div>
      )}

      <div className="grid gap-5 lg:grid-cols-2">
        <div className="overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
          <div className="flex items-center justify-between gap-3 border-b border-white/20 px-4 py-3">
            <h3 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">Intelligence Index</h3>
            <span className="font-mono text-xs text-white/40">Top {models.length}</span>
          </div>
          <div className={compact ? 'h-[380px] p-3' : 'h-[440px] p-3'}>
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
          <div className="flex items-center justify-between gap-3 border-b border-white/20 px-4 py-3">
            <h3 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">Intelligence vs. Price</h3>
            <span className="font-mono text-xs text-white/40">{scatterData.length} priced models</span>
          </div>
          <div className={compact ? 'h-[380px] p-3' : 'h-[440px] p-3'}>
            <ResponsiveContainer width="100%" height="100%">
              <ScatterChart margin={{ top: 14, right: 16, bottom: 16, left: 8 }}>
                <CartesianGrid stroke="rgba(255,255,255,0.08)" />
                <XAxis
                  type="number"
                  dataKey="price"
                  name="Blended price"
                  scale="log"
                  domain={priceDomain}
                  ticks={priceTicks}
                  tickFormatter={formatPriceTick}
                  tick={{ fill: 'rgba(255,255,255,0.45)', fontSize: 12, fontFamily: 'monospace' }}
                  axisLine={{ stroke: 'rgba(255,255,255,0.12)' }}
                  tickLine={false}
                  label={{ value: 'USD / 1M tokens · log scale', position: 'insideBottom', offset: -10, fill: 'rgba(255,255,255,0.45)', fontSize: 12 }}
                />
                <YAxis
                  type="number"
                  dataKey="intelligence"
                  name="AA Index"
                  domain={['dataMin - 2', 'dataMax + 2']}
                  tickCount={6}
                  tickFormatter={value => Number(value).toFixed(0)}
                  tick={{ fill: 'rgba(255,255,255,0.45)', fontSize: 12, fontFamily: 'monospace' }}
                  axisLine={false}
                  tickLine={false}
                  width={42}
                />
                <ZAxis type="number" dataKey="bubbleSize" range={[36, 220]} />
                <Tooltip content={<ScatterTooltip />} cursor={{ stroke: 'rgba(255,255,255,0.18)' }} />
                <Scatter data={scatterData} fill="#a78bfa" shape={<BenchmarkDot />} />
              </ScatterChart>
            </ResponsiveContainer>
          </div>
          <p className="border-t border-white/10 px-4 py-2 font-mono text-[11px] text-white/40">
            Bubble size reflects output speed. Hover a model for exact price, score, and speed.
          </p>
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

function formatPriceTick(value: number) {
  if (value < 0.1) return `$${value.toFixed(2)}`;
  if (value === 0.25) return '$0.25';
  if (value < 1) return `$${value.toFixed(1)}`;
  return `$${Number(value.toFixed(1))}`;
}

function getPriceScale(points: ChartPoint[]): { domain: [number, number]; ticks: number[] } {
  const prices = points.map(point => point.price);
  const min = Math.min(...prices);
  const max = Math.max(...prices);
  const domain: [number, number] = [min * 0.8, max * 1.25];
  const ticks = PRICE_TICK_CANDIDATES.filter(value => value >= domain[0] && value <= domain[1]);
  return { domain, ticks };
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
      <p className="font-mono text-white/60">{item.speed == null ? 'Speed N/A' : `${formatNumber(item.speed)} tok/s`}</p>
    </div>
  );
}

function BenchmarkDot(props: any) {
  const { cx, cy, payload, size } = props;
  const radius = Math.max(3.5, Math.min(Math.sqrt((size ?? 80) / Math.PI), 9));
  return (
    <circle
      cx={cx}
      cy={cy}
      r={radius}
      fill={payload.color}
      fillOpacity={0.78}
      stroke="rgba(255,255,255,0.9)"
      strokeWidth={1}
    />
  );
}
