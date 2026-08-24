import { useCallback, useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Activity, ArrowRight, CheckCircle2, Clock, ExternalLink, XCircle } from 'lucide-react';
import { Seo } from '../components/Seo';

interface Probe {
  model: string;
  provider: string;
  latency_ms: number;
  ok: boolean;
  status_code: number;
  error?: string | null;
  probed_at: string;
}

interface ProbesResponse {
  probes?: Probe[];
  summary?: { total: number; ok: number; failed: number };
  latest_probed_at?: string | null;
}

function shortError(text: string): string {
  const oneLine = text.replace(/\s+/g, ' ').trim();
  return oneLine.length > 90 ? `${oneLine.slice(0, 89)}…` : oneLine;
}

const STALE_MS = 30 * 60 * 1000;

function relativeTime(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  if (!Number.isFinite(diff)) return 'unknown';
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return 'just now';
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

export function Status() {
  const [probes, setProbes] = useState<Probe[]>([]);
  const [latestProbedAt, setLatestProbedAt] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [updatedAt, setUpdatedAt] = useState<Date | null>(null);

  const load = useCallback(async () => {
    try {
      const res = await fetch('/stats/model-probes');
      const data: ProbesResponse = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError('Status unavailable');
        return;
      }
      setProbes(Array.isArray(data.probes) ? data.probes : []);
      setLatestProbedAt(data.latest_probed_at ?? null);
      setUpdatedAt(new Date());
      setError('');
    } catch {
      setError('Status unavailable');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
    const id = setInterval(() => void load(), 60_000);
    return () => clearInterval(id);
  }, [load]);

  const now = Date.now();
  const failed = probes.filter(p => !p.ok);
  const stale = probes.filter(p => p.ok && now - new Date(p.probed_at).getTime() > STALE_MS);
  const hasData = probes.length > 0;

  let banner: 'operational' | 'degraded' | 'unavailable' = 'unavailable';
  if (hasData) {
    banner = failed.length === 0 && stale.length === 0 ? 'operational' : 'degraded';
  }

  return (
    <>
      <Seo
        title="System Status | OpenPaths"
        description="Live chat-completion probe results for models served through the OpenPaths API, plus links to real-traffic latency, TTFT, and throughput observability."
        path="/status"
        jsonLd={{
          '@context': 'https://schema.org',
          '@type': 'WebPage',
          name: 'OpenPaths System Status',
          url: 'https://openpaths.io/status',
          description: 'Live model probe results and public observability for the OpenPaths API.',
        }}
      />
      <div className="mx-auto max-w-6xl px-6 py-16">
        <section className="mb-10">
          <div className="mb-4 inline-flex items-center gap-2 font-mono text-xs uppercase tracking-[0.22em] text-cyan-300">
            <Activity className="h-4 w-4" />
            Status
          </div>
          <h1 className="text-4xl font-semibold tracking-tight md:text-5xl">System status</h1>
          <p className="mt-4 max-w-3xl text-lg leading-relaxed text-white/60">
            Every few minutes we send a real chat completion to each routed model from production and record
            whether it answered. This page shows those live probes — not synthetic uptime percentages.
          </p>
        </section>

        <section aria-live="polite">
          {loading ? (
            <div className="rounded-lg border border-white/20 bg-white/[0.05] px-6 py-8 font-mono text-sm text-white/50">
              Loading probes…
            </div>
          ) : error || !hasData ? (
            <div className="rounded-lg border border-red-500/40 bg-red-500/10 px-6 py-8">
              <div className="flex items-center gap-3 font-mono text-sm text-red-300">
                <XCircle className="h-5 w-5" />
                Status unavailable
              </div>
              <p className="mt-2 text-sm text-white/60">
                Probe data could not be loaded. It will retry automatically in 60 seconds.
              </p>
            </div>
          ) : (
            <>
              <div
                className={`flex flex-wrap items-center gap-4 rounded-lg border px-6 py-6 ${
                  banner === 'operational'
                    ? 'border-emerald-500/40 bg-emerald-500/10'
                    : 'border-amber-500/40 bg-amber-500/10'
                }`}
              >
                {banner === 'operational' ? (
                  <CheckCircle2 className="h-7 w-7 shrink-0 text-emerald-400" />
                ) : (
                  <XCircle className="h-7 w-7 shrink-0 text-amber-400" />
                )}
                <div>
                  <div
                    className={`font-mono text-lg ${
                      banner === 'operational' ? 'text-emerald-300' : 'text-amber-300'
                    }`}
                  >
                    {banner === 'operational'
                      ? 'All systems operational'
                      : `${failed.length + stale.length} of ${probes.length} probed models degraded`}
                  </div>
                  <div className="mt-1 font-mono text-xs text-white/50">
                    {probes.filter(p => p.ok).length}/{probes.length} probes passing
                    {latestProbedAt ? ` · newest probe ${relativeTime(latestProbedAt)}` : ''}
                  </div>
                </div>
                <div className="ml-auto flex items-center gap-2 font-mono text-xs text-white/50">
                  <Clock className="h-3.5 w-3.5" />
                  {updatedAt ? `updated ${relativeTime(updatedAt.toISOString())}` : ''}
                </div>
              </div>

              {(failed.length > 0 || stale.length > 0) && (
                <ul className="mt-4 space-y-2 rounded-lg border border-amber-500/30 bg-amber-500/5 px-6 py-4 font-mono text-sm text-amber-200/90">
                  {failed.map(p => (
                    <li key={`fail-${p.model}`}>
                      {p.model}: last probe failed ({p.error ? shortError(p.error) : `HTTP ${p.status_code}`}){' '}
                      {relativeTime(p.probed_at)}
                    </li>
                  ))}
                  {stale.map(p => (
                    <li key={`stale-${p.model}`}>
                      {p.model}: no fresh probe since {relativeTime(p.probed_at)}
                    </li>
                  ))}
                </ul>
              )}
            </>
          )}
        </section>

        <section className="mt-12 overflow-hidden rounded-lg border border-white/20 bg-white/[0.05]">
          <div className="flex flex-wrap items-center justify-between gap-4 border-b border-white/20 px-4 py-3">
            <h2 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">Model probes</h2>
            <span className="font-mono text-xs text-white/50">auto-refreshes every 60s</span>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[720px] text-left text-sm">
              <thead className="font-mono text-xs uppercase tracking-[0.14em] text-white/50">
                <tr>
                  <th className="px-4 py-3">Model</th>
                  <th className="px-4 py-3">Provider</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Latency</th>
                  <th className="px-4 py-3">Last checked</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/10">
                {!loading && !error && probes.length === 0 && (
                  <tr>
                    <td colSpan={5} className="px-4 py-6 text-white/50">
                      Status unavailable
                    </td>
                  </tr>
                )}
                {probes.map(probe => {
                  const isStale = now - new Date(probe.probed_at).getTime() > STALE_MS;
                  return (
                    <tr key={probe.model} className="text-white/70">
                      <td className="px-4 py-3 font-mono text-white">{probe.model}</td>
                      <td className="px-4 py-3 font-mono text-white/55">{probe.provider}</td>
                      <td className="px-4 py-3">
                        <span className="inline-flex items-center gap-2 font-mono">
                          <span
                            className={`inline-block h-2 w-2 rounded-full ${
                              !probe.ok ? 'bg-red-500' : isStale ? 'bg-amber-400' : 'bg-emerald-400'
                            }`}
                          />
                          <span className={probe.ok && !isStale ? 'text-emerald-300' : probe.ok ? 'text-amber-300' : 'text-red-300'}>
                            {!probe.ok ? 'failing' : isStale ? 'stale' : 'ok'}
                          </span>
                        </span>
                      </td>
                      <td className="px-4 py-3 font-mono">{probe.ok ? `${probe.latency_ms} ms` : '—'}</td>
                      <td className="px-4 py-3 font-mono text-white/55">{relativeTime(probe.probed_at)}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </section>

        <section className="mt-10 grid gap-6 md:grid-cols-2">
          <Link
            to="/stats"
            className="group rounded-lg border border-white/20 bg-white/[0.05] px-6 py-6 transition-colors hover:border-white/50"
          >
            <h2 className="font-mono text-sm uppercase tracking-[0.16em] text-cyan-300">
              Full observability dashboard
            </h2>
            <p className="mt-3 text-sm leading-relaxed text-white/60">
              Per-model average latency, time to first token, and throughput measured from real production
              traffic — not just probes.
            </p>
            <span className="mt-4 inline-flex items-center gap-1 font-mono text-sm text-white group-hover:text-cyan-200">
              Open /stats <ArrowRight className="h-3.5 w-3.5" />
            </span>
          </Link>
          <div className="rounded-lg border border-white/20 bg-white/[0.05] px-6 py-6">
            <h2 className="font-mono text-sm uppercase tracking-[0.16em] text-white/65">How we measure</h2>
            <p className="mt-3 text-sm leading-relaxed text-white/60">
              A background job sends a small chat completion to each routed model every few minutes and stores
              the result: success or failure, HTTP status, and end-to-end latency. Probes older than 30 minutes
              are marked stale. We do not publish invented uptime percentages; what you see here is the raw
              probe record.
            </p>
            <a
              href="/docs"
              className="mt-4 inline-flex items-center gap-1 font-mono text-sm text-cyan-200 hover:text-white"
            >
              API docs <ExternalLink className="h-3.5 w-3.5" />
            </a>
          </div>
        </section>
      </div>
    </>
  );
}
