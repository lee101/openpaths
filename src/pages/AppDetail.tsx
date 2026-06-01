import { useEffect, useMemo, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, BarChart3, ExternalLink, Globe2, Layers3 } from 'lucide-react';
import { Seo } from '../components/Seo';
import { findSeedApp, seedAppStats } from '../data/seedApps';
import { AppUsageStats, appOgImage, formatTokens, host, sourceLabel } from '../lib/appStats';

export function AppDetail() {
  const { slug = '' } = useParams();
  const seed = findSeedApp(slug);
  const [period, setPeriod] = useState('30d');
  const [app, setApp] = useState<AppUsageStats | null>(seed ? seedAppStats(seed) : null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    fetch(`/stats/apps/${encodeURIComponent(slug)}?period=${encodeURIComponent(period)}`)
      .then(res => res.ok ? res.json() : Promise.reject(new Error('not found')))
      .then(data => {
        if (!cancelled && data?.app) setApp(data.app);
      })
      .catch(() => {
        if (!cancelled && seed) setApp(seedAppStats(seed));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [period, seed, slug]);

  const detailPath = `/apps/${slug}/`;
  const title = app ? `${app.name} usage stats | OpenPaths Apps` : 'App usage stats | OpenPaths';
  const description = app
    ? `${app.name} model and token usage on OpenPaths, including top models, total tokens, and opt-in attribution stats.`
    : 'OpenPaths app usage stats and model breakdowns.';
  const maxModelTokens = useMemo(() => Math.max(1, ...(app?.models || []).map(model => model.total_tokens)), [app]);

  if (!app && !loading) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-14 sm:px-6">
        <Seo title="App not found | OpenPaths" description="No app usage stats were found for this app." path={detailPath} />
        <Link to="/apps/" className="mb-6 inline-flex items-center gap-2 font-mono text-sm text-white/55 hover:text-white">
          <ArrowLeft className="h-4 w-4" />
          Apps
        </Link>
        <h1 className="text-3xl font-semibold tracking-tight">App not found</h1>
        <p className="mt-3 text-white/50">This app has not appeared in public or opt-in usage tracking yet.</p>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 md:py-12">
      <Seo title={title} description={description} path={detailPath} image={appOgImage(slug)} />
      <Link to="/apps/" className="mb-6 inline-flex items-center gap-2 font-mono text-sm text-white/55 hover:text-white">
        <ArrowLeft className="h-4 w-4" />
        Apps
      </Link>

      {app && (
        <>
          <section className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_340px]">
            <div>
              <div className="mb-4 flex flex-wrap items-center gap-3">
                <img src={app.favicon_url || '/favicon.ico'} alt="" className="h-14 w-14 rounded-lg border border-white/10 bg-white/10 object-cover" />
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <h1 className="text-3xl font-semibold tracking-tight sm:text-5xl">{app.name}</h1>
                    <span className={`rounded px-2 py-1 font-mono text-[10px] uppercase tracking-[0.12em] ${app.source === 'openpaths' ? 'bg-emerald-400/10 text-emerald-200' : 'bg-cyan-400/10 text-cyan-200'}`}>
                      {sourceLabel(app.source)}
                    </span>
                  </div>
                  <a href={app.url} target="_blank" rel="noopener noreferrer" className="mt-2 inline-flex max-w-full items-center gap-2 truncate font-mono text-sm text-white/40 hover:text-white">
                    <Globe2 className="h-4 w-4 shrink-0" />
                    {host(app.url)}
                    <ExternalLink className="h-3.5 w-3.5 shrink-0" />
                  </a>
                </div>
              </div>
              <p className="max-w-3xl text-base leading-7 text-white/60">{app.description}</p>
              {app.categories?.length > 0 && (
                <div className="mt-5 flex flex-wrap gap-2">
                  {app.categories.map(category => (
                    <span key={category} className="rounded border border-white/10 px-3 py-1.5 font-mono text-xs text-white/50">{category}</span>
                  ))}
                </div>
              )}
            </div>

            <aside className="rounded-lg border border-white/10 bg-white/[0.03] p-4">
              <div className="mb-4 flex items-center justify-between">
                <div className="font-mono text-xs uppercase tracking-[0.18em] text-white/40">Usage window</div>
                <select
                  value={period}
                  onChange={event => setPeriod(event.target.value)}
                  className="rounded border border-white/10 bg-black px-3 py-2 font-mono text-sm text-white"
                >
                  <option value="24h">24h</option>
                  <option value="7d">7d</option>
                  <option value="30d">30d</option>
                </select>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <Stat label="Tokens" value={formatTokens(app.total_tokens)} />
                <Stat label="Requests" value={app.total_requests.toLocaleString('en-US')} />
              </div>
              <div className="mt-4 rounded border border-white/10 bg-black/30 p-3">
                <div className="mb-2 flex items-center gap-2 font-mono text-xs uppercase tracking-[0.16em] text-white/35">
                  <Layers3 className="h-4 w-4" />
                  Attribution
                </div>
                <p className="text-sm leading-6 text-white/50">
                  These stats combine public OpenRouter crawl data and OpenPaths opt-in attribution where available.
                </p>
              </div>
            </aside>
          </section>

          <section className="mt-8 rounded-lg border border-white/10 bg-white/[0.02] p-3 sm:p-4">
            <div className="mb-4 flex items-center gap-2 font-mono text-xs uppercase tracking-[0.18em] text-white/45">
              <BarChart3 className="h-4 w-4" />
              Model usage
            </div>
            {(app.models || []).length > 0 ? (
              <div className="space-y-3">
                {app.models.map(model => (
                  <div key={`${model.source}-${model.provider}-${model.model || 'all'}`} className="rounded-lg border border-white/10 bg-black/30 p-3">
                    <div className="flex min-w-0 flex-wrap items-center justify-between gap-3">
                      <div className="min-w-0">
                        <div className="truncate font-mono text-sm text-white">{model.model || sourceLabel(model.source)}</div>
                        <div className="mt-1 font-mono text-xs text-white/35">{model.provider || sourceLabel(model.source)}</div>
                      </div>
                      <div className="font-mono text-lg font-semibold text-white">{formatTokens(model.total_tokens)}</div>
                    </div>
                    <div className="mt-3 h-1.5 overflow-hidden rounded-full bg-white/10">
                      <div className="h-full rounded-full bg-cyan-300/70" style={{ width: `${Math.max(4, Math.min(100, (model.total_tokens / maxModelTokens) * 100))}%` }} />
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="rounded-lg border border-dashed border-white/10 px-4 py-12 text-center text-sm text-white/40">
                Model-level breakdown will appear after the next crawler refresh or opt-in usage rollup.
              </div>
            )}
          </section>
        </>
      )}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded border border-white/10 bg-black/30 p-3">
      <div className="font-mono text-[10px] uppercase tracking-[0.16em] text-white/30">{label}</div>
      <div className="mt-2 font-mono text-xl font-semibold text-white">{value}</div>
    </div>
  );
}
