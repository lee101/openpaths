import React, { useEffect, useMemo, useState } from 'react';
import { Bar, BarChart, CartesianGrid, Cell, ReferenceLine, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { Activity, ChevronRight, Zap } from 'lucide-react';
import {
  CASE_TITLES,
  DRACULA,
  SUITES,
  fetchEvalSnapshot,
  formatMicroUSD,
  formatRanAt,
  modelColor,
  modelLabel,
  type EvalSnapshot,
  type PerModel,
  type SuiteKey,
} from '../lib/liveEvals';

type TabKey = 'overview' | SuiteKey | 'speed' | 'economics';

const TABS: { key: TabKey; title: string }[] = [
  { key: 'overview', title: 'Overview' },
  ...SUITES.map(s => ({ key: s.key as TabKey, title: s.title })),
  { key: 'speed', title: 'Speed' },
  { key: 'economics', title: 'Economics' },
];

export function LiveEvalsSection() {
  const [snap, setSnap] = useState<EvalSnapshot | null>(null);
  const [loaded, setLoaded] = useState(false);
  const [tab, setTab] = useState<TabKey>('overview');

  useEffect(() => {
    let alive = true;
    fetchEvalSnapshot().then(s => {
      if (!alive) return;
      setSnap(s);
      setLoaded(true);
    });
    return () => { alive = false; };
  }, []);

  const ranked = useMemo(() => {
    if (!snap) return [];
    return [...snap.models].sort((a, b) => b.overall.avg_score - a.overall.avg_score);
  }, [snap]);

  if (!loaded) return null;
  if (!snap || snap.models.length === 0) {
    return (
      <section className="mt-8 rounded-lg border border-dashed border-[#bd93f9]/50 bg-[#282a36]/60 p-6 sm:p-10">
        <div className="mb-3 inline-flex items-center gap-2 rounded-full border border-[#bd93f9]/60 px-3 py-1 font-mono text-[11px] uppercase tracking-[0.22em] text-[#bd93f9]">
          <Zap className="h-3.5 w-3.5" />
          Live evals
        </div>
        <h2 className="text-xl sm:text-2xl font-semibold text-[#f8f8f2]">OpenPaths Live Evals — first sweep pending</h2>
        <p className="mt-2 max-w-2xl text-sm leading-relaxed text-[#b7c0dd]">
          We benchmark <span className="text-[#f8f8f2]">openpaths/auto</span> head-to-head against GPT-5.x, Claude Opus, Gemini, DeepSeek, Grok, and GLM across coding, agentic tool use, and creative SVG suites — measured live through our own gateway every day, with real TTFT, throughput, and cost.
        </p>
        <p className="mt-2 max-w-2xl text-sm leading-relaxed text-[#b7c0dd]">
          Results appear here after the first scheduled sweep (<code className="rounded bg-[#44475a] px-1.5 py-0.5 font-mono text-xs text-[#f1fa8c]">GET /v1/evals/results</code>).
        </p>
      </section>
    );
  }

  const autoVsBest = snap.auto_vs_best?.__overall__;
  const bestOverall = ranked.find(m => m.model !== 'openpaths/auto');
  const auto = snap.models.find(m => m.model === 'openpaths/auto');

  return (
    <section className="mt-8 overflow-hidden rounded-lg border border-[#44475a] bg-[#282a36]/70 shadow-[0_0_40px_rgba(189,147,249,0.06)]">
      {/* Header */}
      <div className="flex flex-col gap-2 border-b border-[#44475a] px-4 py-4 sm:flex-row sm:items-center sm:justify-between sm:px-6">
        <div>
          <div className="inline-flex items-center gap-2 font-mono text-[11px] uppercase tracking-[0.22em] text-[#ff79c6]">
            <span className="relative flex h-2 w-2">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-[#50fa7b] opacity-60" />
              <span className="relative inline-flex h-2 w-2 rounded-full bg-[#50fa7b]" />
            </span>
            Live evals
          </div>
          <h2 className="mt-1 text-lg font-semibold text-[#f8f8f2] sm:text-xl">OpenPaths Auto vs the frontier</h2>
          <p className="text-xs text-[#b7c0dd]">Run through our own gateway · last sweep {formatRanAt(snap.ran_at)}</p>
        </div>
        <div className="flex items-center gap-2 self-start sm:self-auto">
          <RunButton />
        </div>
      </div>

      {/* Auto vs best headline cards */}
      <div className="grid grid-cols-1 gap-3 p-4 sm:grid-cols-2 sm:px-6 xl:grid-cols-4">
        <AutoVsBestCard scope="__overall__" label="Overall" entry={autoVsBest} hero />
        {SUITES.map(s => (
          <AutoVsBestCard key={s.key} scope={s.key} label={s.title} entry={snap.auto_vs_best?.[s.key]} />
        ))}
      </div>

      {/* Tabs */}
      <div className="border-t border-[#44475a]">
        <div className="flex gap-1 overflow-x-auto px-3 pt-3 sm:px-5" role="tablist" aria-label="Eval categories">
          {TABS.map(t => (
            <button
              key={t.key}
              role="tab"
              aria-selected={tab === t.key}
              onClick={() => setTab(t.key)}
              className={`whitespace-nowrap rounded-t-md px-3 py-2 text-sm transition-colors ${
                tab === t.key
                  ? 'bg-[#44475a] font-medium text-[#f8f8f2]'
                  : 'text-[#b7c0dd] hover:bg-[#44475a]/50 hover:text-[#f8f8f2]'
              }`}
            >
              {t.title}
            </button>
          ))}
        </div>

        <div className="bg-[#21222c] px-3 py-5 sm:px-5 sm:py-6">
          {(tab === 'overview' || tab === 'coding' || tab === 'agentic' || tab === 'creative') && (
            <ScoreBoard snap={snap} tab={tab} />
          )}
          {tab === 'speed' && <SpeedBoard models={ranked} />}
          {tab === 'economics' && <EconomicsBoard models={ranked} />}
        </div>
      </div>

      <p className="border-t border-[#44475a] px-4 py-3 text-[11px] leading-relaxed text-[#6272a4] sm:px-6">
        Methodology: deterministic graders (exact answers, tool-call arguments, SVG constraint checks) over streaming requests through production routing.
        Cost priced at catalogue list rates from actual token counts; TTFT is time-to-first-token. Auto routes exactly as it does for API customers.
      </p>
    </section>
  );
}

function AutoVsBestCard({ scope, label, entry, hero }: { key?: React.Key; scope: string; label: string; entry?: EvalSnapshot['auto_vs_best'][string]; hero?: boolean }) {
  const autoScore = entry?.auto_score;
  const bestScore = entry?.best_score;
  const delta = autoScore != null && bestScore != null ? autoScore - bestScore : null;
  const autoWins = delta != null && delta >= 0;

  return (
    <div
      className={`rounded-md border p-4 ${
        hero ? 'border-[#bd93f9]/70 bg-gradient-to-br from-[#bd93f9]/15 via-transparent to-[#ff79c6]/10' : 'border-[#44475a] bg-[#282a36]'
      }`}
    >
      <div className={`font-mono text-[11px] uppercase tracking-[0.18em] ${hero ? 'text-[#bd93f9]' : 'text-[#6272a4]'}`}>{label}</div>
      <div className="mt-2 flex items-baseline gap-2">
        <span className="text-2xl font-bold tabular-nums text-[#f8f8f2]">{autoScore != null ? `${Math.round(autoScore * 100)}%` : '—'}</span>
        <span className="text-xs text-[#b7c0dd]">Auto</span>
        {delta != null && (
          <span className={`ml-auto rounded-full px-2 py-0.5 text-[11px] font-semibold ${autoWins ? 'bg-[#50fa7b]/15 text-[#50fa7b]' : 'bg-[#ff5555]/15 text-[#ff5555]'}`}>
            {autoWins ? '+' : ''}{Math.round(delta * 100)} pts vs best
          </span>
        )}
      </div>
      <div className="mt-1 flex items-center gap-1.5 text-xs text-[#b7c0dd]">
        <ChevronRight className="h-3 w-3 text-[#6272a4]" />
        best single model:{' '}
        <span className="font-medium text-[#f8f8f2]">{entry?.best_model ? modelLabel(entry.best_model) : '—'}</span>
        <span className="tabular-nums text-[#6272a4]">{bestScore != null ? `· ${Math.round(bestScore * 100)}%` : ''}</span>
      </div>
    </div>
  );
}

function ScoreBoard({ snap, tab }: { snap: EvalSnapshot; tab: TabKey }) {
  const scopeAgg = (m: PerModel): { score: number; passRate: number } => {
    if (tab === 'overview') return { score: m.overall.avg_score, passRate: m.overall.pass_rate };
    const s = m.by_suite[tab];
    return s && s.cases > 0 ? { score: s.avg_score, passRate: s.pass_rate } : { score: 0, passRate: 0 };
  };

  const rows = [...snap.models]
    .map(m => ({ model: m.model, ...scopeAgg(m) }))
    .filter(r => tab !== 'overview' || r.score > 0)
    .sort((a, b) => b.score - a.score);

  const chartData = rows.map(r => ({ name: modelLabel(r.model), model: r.model, score: Math.round(r.score * 100) }));

  const activeSuite = SUITES.find(s => s.key === tab);
  const suiteCases = tab !== 'overview' ? snap.cases.filter(c => c.suite === tab) : [];

  return (
    <div className="space-y-6">
      {activeSuite && <p className="text-sm leading-relaxed text-[#b7c0dd]">{activeSuite.blurb}</p>}
      {tab === 'overview' && (
        <p className="text-sm leading-relaxed text-[#b7c0dd]">
          Average score across all {snap.cases.length} graded cases. Every case runs on every model — including{' '}
          <span className="font-medium text-[#f8f8f2]">openpaths/auto</span>, routed identically to production traffic.
        </p>
      )}

      <div className="h-72 w-full sm:h-80">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={chartData} layout="vertical" margin={{ top: 4, right: 32, bottom: 4, left: 8 }}>
            <CartesianGrid stroke={DRACULA.currentLine} strokeOpacity={0.4} horizontal={false} />
            <XAxis type="number" domain={[0, 100]} tick={{ fill: DRACULA.comment, fontSize: 11 }} axisLine={{ stroke: DRACULA.currentLine }} tickLine={false} unit="%" />
            <YAxis type="category" dataKey="name" width={128} tick={{ fill: DRACULA.foreground, fontSize: 12 }} axisLine={false} tickLine={false} />
            <Tooltip
              cursor={{ fill: '#44475a33' }}
              contentStyle={{ background: DRACULA.bg, border: `1px solid ${DRACULA.currentLine}`, borderRadius: 6, color: DRACULA.foreground }}
              formatter={(value) => [`${value}%`, 'Score']}
            />
            <ReferenceLine x={100} stroke={DRACULA.currentLine} strokeDasharray="3 3" />
            <Bar dataKey="score" radius={[0, 4, 4, 0]} barSize={18}>
              {chartData.map(d => (
                <Cell key={d.model} fill={d.model === 'openpaths/auto' ? DRACULA.purple : modelColor(d.model)} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>

      {suiteCases.length > 0 && <CaseMatrix cases={suiteCases} models={rows.map(r => r.model)} />}
    </div>
  );
}

function CaseMatrix({ cases, models }: { cases: EvalSnapshot['cases']; models: string[] }) {
  return (
    <div>
      <h3 className="mb-2 font-mono text-[11px] uppercase tracking-[0.18em] text-[#6272a4]">Per-case results</h3>
      <div className="-mx-3 overflow-x-auto px-3 pb-1 sm:mx-0 sm:px-0">
        <table className="w-full min-w-[560px] border-separate border-spacing-y-1 text-left text-sm">
          <thead>
            <tr>
              <th className="sticky left-0 z-10 bg-[#21222c] pr-3 font-mono text-[11px] font-normal uppercase tracking-wider text-[#6272a4]">Case</th>
              {models.map(m => (
                <th key={m} className="px-1 pb-1 text-center font-mono text-[11px] font-normal uppercase tracking-wider text-[#b7c0dd]">
                  {modelLabel(m)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {cases.map(c => (
              <tr key={c.case_id}>
                <td className="sticky left-0 z-10 max-w-[180px] truncate bg-[#21222c] pr-3 text-[13px] text-[#f8f8f2]" title={CASE_TITLES[c.case_id] ?? c.case_id}>
                  {CASE_TITLES[c.case_id] ?? c.case_id}
                </td>
                {models.map(m => {
                  const r = c.results[m];
                  if (!r) {
                    return <td key={m} className="px-1"><div className="flex h-9 items-center justify-center rounded bg-[#44475a]/30 text-xs text-[#6272a4]">—</div></td>;
                  }
                  const bg = r.passed
                    ? 'rgba(80,250,123,0.18)'
                    : r.error
                      ? 'rgba(98,114,164,0.25)'
                      : r.score > 0
                        ? `rgba(255,184,108,${0.12 + 0.35 * r.score})`
                        : 'rgba(255,85,85,0.20)';
                  const fg = r.passed ? DRACULA.green : r.error ? DRACULA.comment : r.score > 0 ? DRACULA.orange : DRACULA.red;
                  return (
                    <td key={m} className="px-1">
                      <div
                        className="flex h-9 cursor-help items-center justify-center rounded text-xs font-semibold tabular-nums"
                        style={{ backgroundColor: bg, color: fg }}
                        title={r.error ?? r.answer_preview}
                      >
                        {r.error ? '!' : Math.round(r.score * 100)}
                      </div>
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function SpeedBoard({ models }: { models: PerModel[] }) {
  const ttft = models.filter(m => m.overall.median_ttft_ms > 0).sort((a, b) => a.overall.median_ttft_ms - b.overall.median_ttft_ms);
  const tps = models.filter(m => m.overall.avg_tps > 0).sort((a, b) => b.overall.avg_tps - a.overall.avg_tps);

  const toChart = (list: PerModel[], key: 'median_ttft_ms' | 'avg_tps') =>
    list.map(m => ({ name: modelLabel(m.model), model: m.model, value: Math.round(m.overall[key]) }));

  return (
    <div className="grid grid-cols-1 gap-8 xl:grid-cols-2">
      <div>
        <MetricHead label="Median TTFT — lower is better" icon={<Activity className="h-3.5 w-3.5 text-[#8be9fd]" />} />
        <ChartH value={toChart(ttft, 'median_ttft_ms')} unit="ms" better="low" />
      </div>
      <div>
        <MetricHead label="Output tokens/sec — higher is better" icon={<Zap className="h-3.5 w-3.5 text-[#f1fa8c]" />} />
        <ChartH value={toChart(tps, 'avg_tps')} unit=" tok/s" better="high" />
      </div>
    </div>
  );
}

function MetricHead({ label, icon }: { label: string; icon: React.ReactNode }) {
  return (
    <h3 className="mb-2 inline-flex items-center gap-1.5 font-mono text-[11px] uppercase tracking-[0.18em] text-[#b7c0dd]">
      {icon}
      {label}
    </h3>
  );
}

function ChartH({ value, unit, better }: { value: { name: string; model: string; value: number }[]; unit: string; better: 'low' | 'high' }) {
  const best = better === 'low' ? Math.min(...value.map(v => v.value)) : Math.max(...value.map(v => v.value));
  return (
    <div className="h-72 w-full sm:h-80">
      <ResponsiveContainer width="100%" height="100%">
        <BarChart data={value} layout="vertical" margin={{ top: 4, right: 48, bottom: 4, left: 8 }}>
          <CartesianGrid stroke={DRACULA.currentLine} strokeOpacity={0.4} horizontal={false} />
          <XAxis type="number" tick={{ fill: DRACULA.comment, fontSize: 11 }} axisLine={{ stroke: DRACULA.currentLine }} tickLine={false} />
          <YAxis type="category" dataKey="name" width={128} tick={{ fill: DRACULA.foreground, fontSize: 12 }} axisLine={false} tickLine={false} />
          <Tooltip
            cursor={{ fill: '#44475a33' }}
            contentStyle={{ background: DRACULA.bg, border: `1px solid ${DRACULA.currentLine}`, borderRadius: 6, color: DRACULA.foreground }}
            formatter={(v) => [`${v}${unit}`, '']}
          />
          <Bar dataKey="value" radius={[0, 4, 4, 0]} barSize={16}>
            {value.map(d => (
              <Cell key={d.model} fill={d.model === 'openpaths/auto' ? DRACULA.purple : d.value === best ? DRACULA.green : modelColor(d.model)} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
}

function EconomicsBoard({ models }: { models: PerModel[] }) {
  const priced = models.filter(m => m.overall.cost_per_case_micro_usd > 0);
  const costRows = [...priced].sort((a, b) => a.overall.cost_per_case_micro_usd - b.overall.cost_per_case_micro_usd)
    .map(m => ({ name: modelLabel(m.model), model: m.model, value: Number((m.overall.cost_per_case_micro_usd / 100).toFixed(3)) }));
  // Quality points per dollar: pts = avg_score*100; $ = micro/1e6.
  const qpr = [...priced]
    .map(m => ({ name: modelLabel(m.model), model: m.model, value: Number(((m.overall.avg_score * 100 * 1_000_000) / m.overall.cost_per_case_micro_usd / 1000).toFixed(1)) }))
    .sort((a, b) => b.value - a.value);

  return (
    <div className="space-y-8">
      <p className="text-sm leading-relaxed text-[#b7c0dd]">
        Cost per graded case from actual token counts at list prices. Quality per dollar = quality points (0–100) per dollar spent.
      </p>
      <div className="grid grid-cols-1 gap-8 xl:grid-cols-2">
        <div>
          <MetricHead label="Cost per case — lower is better" icon={<Activity className="h-3.5 w-3.5 text-[#ffb86c]" />} />
          <ChartH value={costRows} unit="$" better="low" />
        </div>
        <div>
          <MetricHead label="Quality points per $1 (thousands) — higher is better" icon={<Zap className="h-3.5 w-3.5 text-[#50fa7b]" />} />
          <ChartH value={qpr} unit="k pts/$" better="high" />
        </div>
      </div>
    </div>
  );
}

function RunButton() {
  const [state, setState] = useState<'idle' | 'running' | 'denied'>('idle');
  const hasKey = typeof window !== 'undefined' && !!localStorage.getItem('op_api_key');
  if (!hasKey) return null;

  const run = async () => {
    setState('running');
    try {
      const res = await fetch('/v1/evals/run', { method: 'POST', headers: { Authorization: `Bearer ${localStorage.getItem('op_api_key')}` } });
      setState(res.status === 202 ? 'idle' : res.status === 409 ? 'idle' : 'denied');
    } catch {
      setState('denied');
    }
  };

  return (
    <button
      onClick={run}
      disabled={state === 'running'}
      className="inline-flex items-center gap-1.5 rounded-md border border-[#bd93f9]/60 px-3 py-1.5 text-xs font-medium text-[#bd93f9] transition-colors hover:bg-[#bd93f9]/10 disabled:opacity-50"
    >
      <Activity className={`h-3.5 w-3.5 ${state === 'running' ? 'animate-pulse' : ''}`} />
      {state === 'running' ? 'Sweep started…' : 'Run sweep now'}
    </button>
  );
}
