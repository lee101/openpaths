import React, { useEffect, useMemo, useState } from 'react';
import { Activity, AppWindow, BarChart3, Database, RefreshCw } from 'lucide-react';
import { Link } from 'react-router-dom';
import { Seo } from '../components/Seo';
import { seedApps, seedAppStats } from '../data/seedApps';
import { AppUsageStats, appOgImage, formatTokens, host, sourceLabel } from '../lib/appStats';

const PERIODS = ['24h', '7d', '30d'];
const seededAppStats = seedApps.map(seedAppStats);

export function Apps() {
  const [period, setPeriod] = useState('30d');
  const [apps, setApps] = useState<AppUsageStats[]>(seededAppStats);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await fetch(`/stats/apps?period=${encodeURIComponent(period)}&limit=100`);
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error?.message || 'App stats unavailable');
        return;
      }
      setApps(Array.isArray(data.apps) ? data.apps : []);
    } catch {
      setError('Network error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, [period]);

  const totals = useMemo(() => apps.reduce((acc, app) => {
    acc.tokens += app.total_tokens;
    acc.requests += app.total_requests;
    return acc;
  }, { tokens: 0, requests: 0 }), [apps]);
  const maxTokens = useMemo(() => Math.max(1, ...apps.map(app => app.total_tokens)), [apps]);

  return (
    <>
      <Seo
        title="Apps | OpenPaths App Usage Stats"
        description="Opt-in app and agent usage stats across OpenPaths and OpenRouter, including model and token breakdowns."
        path="/apps/"
        image={appOgImage('openpaths-apps')}
      />
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 md:py-12">
        <div className="mb-6 grid gap-5 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
          <div>
            <div className="mb-3 flex items-center gap-2 font-mono text-xs uppercase tracking-[0.24em] text-cyan-300">
              <AppWindow className="h-4 w-4" />
              Opt-in usage tracking
            </div>
            <h1 className="text-3xl font-semibold tracking-tight sm:text-4xl md:text-5xl">Apps And Agents</h1>
            <p className="mt-3 max-w-3xl text-sm leading-6 text-white/55 sm:text-base">
              Apps appear here when they send attribution headers, or when the daily OpenRouter crawler refreshes public rankings.
            </p>
          </div>
          <div className="flex items-center gap-2 sm:justify-end">
            <div className="grid flex-1 grid-cols-3 rounded-lg border border-white/10 bg-white/[0.03] p-1 sm:flex-none">
              {PERIODS.map(option => (
                <button
                  key={option}
                  onClick={() => setPeriod(option)}
                  className={`rounded-md px-3 py-2 font-mono text-xs transition-colors sm:px-4 sm:text-sm ${period === option ? 'bg-white text-black' : 'text-white/55 hover:text-white'}`}
                >
                  {option}
                </button>
              ))}
            </div>
            <button
              onClick={() => void load()}
              disabled={loading}
              className="inline-flex h-10 w-10 items-center justify-center rounded-lg border border-white/10 bg-white text-black hover:bg-white/90 disabled:opacity-60 sm:w-auto sm:px-4"
              aria-label="Refresh app stats"
            >
              <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              <span className="hidden font-mono text-sm font-bold sm:inline">Refresh</span>
            </button>
          </div>
        </div>

        <div className="mb-6 grid gap-3 sm:grid-cols-3">
          <Metric icon={<AppWindow className="h-5 w-5" />} label="Apps" value={apps.length.toLocaleString('en-US')} />
          <Metric icon={<Activity className="h-5 w-5" />} label="Requests" value={totals.requests.toLocaleString('en-US')} />
          <Metric icon={<BarChart3 className="h-5 w-5" />} label="Tokens" value={formatTokens(totals.tokens)} />
        </div>

        {error && <div className="mb-6 rounded-lg border border-red-400/20 bg-red-500/10 p-4 text-sm text-red-100">{error}</div>}

        <div className="rounded-lg border border-white/10 bg-white/[0.02] p-2">
          <div className="mb-2 flex items-center justify-between px-2 py-2">
            <div className="flex items-center gap-2 font-mono text-xs uppercase tracking-[0.18em] text-white/45">
              <Database className="h-4 w-4" />
              Usage leaderboard
            </div>
            <div className="hidden font-mono text-xs text-white/30 sm:block">{period}</div>
          </div>
          {apps.length === 0 && !loading ? (
            <div className="rounded-lg border border-dashed border-white/10 px-4 py-12 text-center font-mono text-sm text-white/35">No app usage has been recorded for this period.</div>
          ) : (
            <div className="space-y-2">
              {apps.map((app, index) => {
                const percent = Math.max(4, Math.min(100, (app.total_tokens / maxTokens) * 100));
                return (
                  <article key={app.app_id} className="rounded-lg border border-white/10 bg-black/35 p-3 transition-colors hover:border-white/20 sm:p-4">
                    <div className="grid gap-3 lg:grid-cols-[minmax(0,1.15fr)_minmax(260px,0.85fr)_150px] lg:items-start">
                      <div className="min-w-0">
                        <div className="flex min-w-0 items-start gap-3">
                          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded border border-white/10 bg-white/[0.04] font-mono text-xs text-white/45">
                            {index + 1}
                          </div>
                          <img src={app.favicon_url || '/favicon.ico'} alt="" className="h-9 w-9 shrink-0 rounded bg-white/10 object-cover" />
                          <div className="min-w-0">
                            <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1">
                              <Link to={`/apps/${encodeURIComponent(app.slug)}/`} className="truncate text-base font-semibold text-white hover:underline sm:text-lg">
                                {app.name}
                              </Link>
                              <span className={`rounded px-2 py-0.5 font-mono text-[10px] uppercase tracking-[0.12em] ${app.source === 'openpaths' ? 'bg-emerald-400/10 text-emerald-200' : 'bg-cyan-400/10 text-cyan-200'}`}>
                                {sourceLabel(app.source)}
                              </span>
                            </div>
                            <div className="mt-1 truncate font-mono text-xs text-white/35">{host(app.url) || sourceLabel(app.source)}</div>
                          </div>
                        </div>
                        {app.description && <p className="mt-3 line-clamp-2 text-sm leading-5 text-white/50">{app.description}</p>}
                        {app.categories?.length > 0 && (
                          <div className="mt-3 flex flex-wrap gap-1.5">
                            {app.categories.slice(0, 3).map(category => (
                              <span key={`${app.app_id}-${category}`} className="rounded border border-white/10 px-2 py-1 font-mono text-[11px] text-white/45">
                                {category}
                              </span>
                            ))}
                          </div>
                        )}
                      </div>

                      <div className="min-w-0">
                        <div className="mb-2 font-mono text-[10px] uppercase tracking-[0.16em] text-white/30">Top models</div>
                        <div className="space-y-2">
                          {(app.models || []).slice(0, 3).map(model => (
                            <div key={`${app.app_id}-${model.source}-${model.model || 'all'}-${model.provider}`} className="min-w-0">
                              <div className="flex min-w-0 items-center justify-between gap-3">
                                <span className="truncate font-mono text-xs text-white/60">{model.model || model.source}</span>
                                <span className="shrink-0 font-mono text-xs text-white/80">{formatTokens(model.total_tokens)}</span>
                              </div>
                              <div className="mt-1 h-1 overflow-hidden rounded-full bg-white/10">
                                <div className="h-full rounded-full bg-white/45" style={{ width: `${Math.max(4, Math.min(100, (model.total_tokens / Math.max(1, app.total_tokens)) * 100))}%` }} />
                              </div>
                            </div>
                          ))}
                          {(app.models || []).length === 0 && (
                            <div className="font-mono text-xs text-white/30">No model breakdown yet</div>
                          )}
                        </div>
                      </div>

                      <div className="lg:text-right">
                        <div className="flex items-end justify-between gap-3 lg:block">
                          <div>
                            <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-white/30">Tokens</div>
                            <Link to={`/apps/${encodeURIComponent(app.slug)}/`} className="font-mono text-xl font-semibold text-white hover:underline sm:text-2xl">
                              {formatTokens(app.total_tokens)}
                            </Link>
                          </div>
                          {app.total_requests > 0 && (
                            <div className="font-mono text-xs text-white/35 lg:mt-1">
                              {app.total_requests.toLocaleString('en-US')} req
                            </div>
                          )}
                        </div>
                        <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-white/10">
                          <div className="h-full rounded-full bg-cyan-300/70" style={{ width: `${percent}%` }} />
                        </div>
                      </div>
                    </div>
                  </article>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </>
  );
}

function Metric({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="rounded-lg border border-white/10 bg-white/[0.03] p-3 sm:p-4">
      <div className="mb-2 flex items-center gap-2 text-white/45 sm:mb-3">{icon}<span className="font-mono text-[11px] uppercase tracking-[0.16em] sm:text-xs">{label}</span></div>
      <div className="font-mono text-xl font-semibold text-white sm:text-2xl">{value}</div>
    </div>
  );
}
