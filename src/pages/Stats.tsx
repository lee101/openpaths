import React, { useEffect, useMemo, useState } from 'react';
import { AlertTriangle, BarChart3, Cpu, Database, RefreshCw } from 'lucide-react';

type UsageBreakdown = {
  task: string;
  provider: string;
  model: string;
  total_requests: number;
  total_tokens_in: number;
  total_tokens_out: number;
  total_cost_cents: number;
  avg_latency_ms: number;
  error_rate: number;
};

const PERIODS = [
  { label: '24h', value: '24h' },
  { label: '7d', value: '7d' },
  { label: '30d', value: '30d' },
];

function formatNumber(value: number): string {
  return value.toLocaleString('en-US');
}

function formatCost(cents: number): string {
  return `$${(cents / 10000).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 4 })}`;
}

function formatPct(value: number): string {
  return `${(value * 100).toFixed(2)}%`;
}

export function Stats() {
  const [period, setPeriod] = useState('24h');
  const [rows, setRows] = useState<UsageBreakdown[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await fetch(`/stats/breakdown?period=${encodeURIComponent(period)}`);
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error?.message || 'Stats unavailable');
        return;
      }
      setRows(Array.isArray(data.breakdown) ? data.breakdown : []);
    } catch {
      setError('Network error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, [period]);

  const totals = useMemo(() => {
    return rows.reduce(
      (acc, row) => {
        acc.requests += row.total_requests;
        acc.tokens += row.total_tokens_in + row.total_tokens_out;
        acc.cost += row.total_cost_cents;
        return acc;
      },
      { requests: 0, tokens: 0, cost: 0 },
    );
  }, [rows]);

  const byTask = useMemo(() => {
    const groups = new Map<string, UsageBreakdown[]>();
    rows.forEach(row => {
      const next = groups.get(row.task) || [];
      next.push(row);
      groups.set(row.task, next);
    });
    return Array.from(groups.entries()).map(([task, items]) => ({
      task,
      requests: items.reduce((sum, item) => sum + item.total_requests, 0),
      items,
    }));
  }, [rows]);

  return (
    <div className="max-w-7xl mx-auto px-6 py-12">
      <div className="flex flex-col gap-5 md:flex-row md:items-end md:justify-between mb-8">
        <div>
          <div className="flex items-center gap-2 text-cyan-300 font-mono text-xs uppercase tracking-[0.24em] mb-3">
            <BarChart3 className="w-4 h-4" />
            Anonymous usage
          </div>
          <h1 className="text-4xl md:text-5xl font-semibold tracking-tight">Model Usage Stats</h1>
          <p className="mt-3 text-white/50 max-w-2xl">
            Public aggregate request volume by task, provider, and model. No user, API key, or prompt data is exposed.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-3">
          <div className="inline-flex rounded-lg border border-white/10 bg-white/[0.03] p-1">
            {PERIODS.map(option => (
              <button
                key={option.value}
                onClick={() => setPeriod(option.value)}
                className={`rounded-md px-4 py-2 font-mono text-sm transition-colors ${
                  period === option.value ? 'bg-white text-black' : 'text-white/55 hover:text-white'
                }`}
              >
                {option.label}
              </button>
            ))}
          </div>
          <button
            onClick={() => void load()}
            disabled={loading}
            className="inline-flex items-center justify-center gap-2 rounded-lg border border-white/10 bg-white text-black px-4 py-3 font-mono text-sm font-bold hover:bg-white/90 disabled:opacity-60"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>
      </div>

      {error && (
        <div className="mb-8 flex items-start gap-3 rounded-lg border border-red-400/20 bg-red-500/10 p-4 text-red-200">
          <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0" />
          <div>
            <p className="font-mono text-sm font-bold">Stats unavailable</p>
            <p className="text-sm text-red-100/80">{error}</p>
          </div>
        </div>
      )}

      <div className="grid gap-4 md:grid-cols-3 mb-8">
        <Metric label="Requests" value={formatNumber(totals.requests)} icon={<Database className="w-5 h-5" />} />
        <Metric label="Tokens" value={formatNumber(totals.tokens)} icon={<Cpu className="w-5 h-5" />} />
        <Metric label="Spend" value={formatCost(totals.cost)} icon={<BarChart3 className="w-5 h-5" />} />
      </div>

      <div className="space-y-6">
        {loading && rows.length === 0 && (
          <div className="rounded-lg border border-white/10 p-10 text-center font-mono text-sm text-white/40">
            Loading usage stats...
          </div>
        )}
        {!loading && rows.length === 0 && !error && (
          <div className="rounded-lg border border-white/10 p-10 text-center font-mono text-sm text-white/40">
            No usage recorded for this period.
          </div>
        )}
        {byTask.map(group => (
          <section key={group.task} className="overflow-hidden rounded-lg border border-white/10">
            <div className="flex items-center justify-between gap-4 bg-white/[0.04] px-4 py-3">
              <h2 className="font-mono text-sm uppercase tracking-[0.18em] text-white/70">{group.task}</h2>
              <span className="font-mono text-sm text-white/45">{formatNumber(group.requests)} requests</span>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full min-w-[920px] text-left">
                <thead className="text-xs font-mono uppercase tracking-[0.16em] text-white/35">
                  <tr>
                    <th className="px-4 py-3">Provider</th>
                    <th className="px-4 py-3">Model</th>
                    <th className="px-4 py-3">Requests</th>
                    <th className="px-4 py-3">Tokens in</th>
                    <th className="px-4 py-3">Tokens out</th>
                    <th className="px-4 py-3">Avg latency</th>
                    <th className="px-4 py-3">Error rate</th>
                    <th className="px-4 py-3">Spend</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/10">
                  {group.items.map(row => (
                    <tr key={`${row.task}:${row.provider}:${row.model}`} className="text-sm text-white/75">
                      <td className="px-4 py-4 font-mono text-white">{row.provider}</td>
                      <td className="px-4 py-4 font-mono text-white/60">{row.model}</td>
                      <td className="px-4 py-4 font-mono">{formatNumber(row.total_requests)}</td>
                      <td className="px-4 py-4 font-mono">{formatNumber(row.total_tokens_in)}</td>
                      <td className="px-4 py-4 font-mono">{formatNumber(row.total_tokens_out)}</td>
                      <td className="px-4 py-4 font-mono">{Math.round(row.avg_latency_ms)} ms</td>
                      <td className="px-4 py-4 font-mono">{formatPct(row.error_rate)}</td>
                      <td className="px-4 py-4 font-mono">{formatCost(row.total_cost_cents)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}

function Metric({ label, value, icon }: { label: string; value: string; icon: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-white/10 bg-white/[0.03] p-5">
      <div className="mb-4 flex items-center justify-between text-white/45">
        <p className="font-mono text-xs uppercase tracking-[0.18em]">{label}</p>
        {icon}
      </div>
      <p className="font-mono text-2xl text-white">{value}</p>
    </div>
  );
}
