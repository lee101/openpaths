import React, { useEffect, useMemo, useState } from 'react';
import { AlertTriangle, ArrowDownToLine, ArrowUpDown, ChevronDown, ChevronUp, Database, RefreshCw, ShieldCheck } from 'lucide-react';
import { Link, useParams } from 'react-router-dom';
import { getStoredAPIKey } from '../lib/session';

const API_BASE = '';

type AdminSpendUser = {
  user_id: string;
  email: string;
  name: string;
  created_at: string;
  last_request_at?: string;
  disabled: boolean;
  is_admin: boolean;
  balance_cents: number;
  stripe_gross_cents: number;
  stripe_refunded_cents: number;
  stripe_net_cents: number;
  api_requests: number;
  api_spend_cents: number;
  provider_base_cost_cents: number;
  provider_estimated: boolean;
};

type AdminSpendTotals = {
  user_count: number;
  stripe_gross_cents: number;
  stripe_refunded_cents: number;
  stripe_net_cents: number;
  api_requests: number;
  api_spend_cents: number;
  provider_base_cost_cents: number;
  provider_estimated: boolean;
};

function formatStripeCents(cents: number): string {
  return `$${(cents / 100).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

function formatUsageUnits(units: number): string {
  return `$${(units / 10000).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 4 })}`;
}

function formatNumber(value: number): string {
  return value.toLocaleString('en-US');
}

function formatDate(value?: string): string {
  if (!value) return 'Never';
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  }).format(new Date(value));
}

async function adminApi(path: string) {
  const apiKey = getStoredAPIKey();
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (apiKey) headers.Authorization = `Bearer ${apiKey}`;
  return fetch(API_BASE + path, { headers });
}

interface MaxPlanStatus {
  enabled: boolean;
  email: string;
  credential_user: string;
  credential_count: number;
  healthy_count: number;
  auth_mode?: string;
  refreshable?: boolean;
}

export function AdminLee() {
  const [users, setUsers] = useState<AdminSpendUser[]>([]);
  const [totals, setTotals] = useState<AdminSpendTotals | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [maxPlan, setMaxPlan] = useState<MaxPlanStatus | null>(null);
  const [maxPlanMsg, setMaxPlanMsg] = useState('');
  const [sort, setSort] = useState<{ key: keyof AdminSpendUser; direction: 'asc' | 'desc' }>({ key: 'api_spend_cents', direction: 'desc' });

  const loadMaxPlan = async () => {
    try {
      const res = await adminApi('/admin/openai-max-plan');
      const data = await res.json().catch(() => ({}));
      if (res.ok) setMaxPlan(data as MaxPlanStatus);
    } catch {
      /* non-fatal */
    }
  };

  const refreshMaxPlan = async () => {
    setMaxPlanMsg('Refreshing...');
    try {
      const apiKey = getStoredAPIKey();
      const res = await fetch(API_BASE + '/admin/openai-max-plan/refresh', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(apiKey ? { Authorization: `Bearer ${apiKey}` } : {}),
        },
      });
      const data = await res.json().catch(() => ({}));
      setMaxPlanMsg(res.ok ? 'Refresh triggered.' : data.error?.message || 'Refresh failed');
      setTimeout(() => void loadMaxPlan(), 1500);
    } catch {
      setMaxPlanMsg('Network error');
    }
  };

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await adminApi('/admin/users/spend');
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        setError(data.error?.message || 'Admin dashboard unavailable');
        return;
      }
      setUsers(Array.isArray(data.users) ? data.users : []);
      setTotals(data.totals || null);
    } catch {
      setError('Network error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
    void loadMaxPlan();
  }, []);

  const marginUnits = useMemo(() => {
    if (!totals) return 0;
    return totals.api_spend_cents - totals.provider_base_cost_cents;
  }, [totals]);

  const sortedUsers = useMemo(() => {
    return [...users].sort((a, b) => {
      const av = a[sort.key];
      const bv = b[sort.key];
      const left = av == null ? '' : av;
      const right = bv == null ? '' : bv;
      const result = typeof left === 'number' && typeof right === 'number'
        ? left - right
        : String(left).localeCompare(String(right));
      return sort.direction === 'asc' ? result : -result;
    });
  }, [sort, users]);

  const sortBy = (key: keyof AdminSpendUser) => {
    setSort(current => current.key === key
      ? { key, direction: current.direction === 'asc' ? 'desc' : 'asc' }
      : { key, direction: 'desc' });
  };

  return (
    <div className="max-w-7xl mx-auto px-6 py-12">
      <div className="flex flex-col gap-5 md:flex-row md:items-end md:justify-between mb-8">
        <div>
          <div className="flex items-center gap-2 text-emerald-300 font-mono text-xs uppercase tracking-[0.24em] mb-3">
            <ShieldCheck className="w-4 h-4" />
            Admin only
          </div>
          <h1 className="text-4xl md:text-5xl font-semibold tracking-tight">Adminlee Spend</h1>
          <p className="mt-3 text-white/50 max-w-2xl">
            All users, Stripe deposits, API spend, and estimated provider base cost from recorded usage.
          </p>
        </div>
        <button
          onClick={() => void load()}
          disabled={loading}
          className="inline-flex items-center justify-center gap-2 rounded-lg border border-white/20 bg-white text-black px-4 py-3 font-mono text-sm font-bold hover:bg-white/90 disabled:opacity-60"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
          Refresh
        </button>
      </div>

      {error && (
        <div className="mb-8 flex items-start gap-3 rounded-lg border border-red-400/20 bg-red-500/10 p-4 text-red-200">
          <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0" />
          <div>
            <p className="font-mono text-sm font-bold">Access blocked</p>
            <p className="text-sm text-red-100/80">{error}</p>
          </div>
        </div>
      )}

      <div className="grid gap-4 md:grid-cols-4 mb-8">
        <Metric label="Stripe net" value={formatStripeCents(totals?.stripe_net_cents || 0)} icon={<ArrowDownToLine className="w-5 h-5" />} />
        <Metric label="API spend" value={formatUsageUnits(totals?.api_spend_cents || 0)} icon={<Database className="w-5 h-5" />} />
        <Metric label="Provider base" value={formatUsageUnits(totals?.provider_base_cost_cents || 0)} icon={<Database className="w-5 h-5" />} />
        <Metric label="Gross margin" value={formatUsageUnits(marginUnits)} icon={<ShieldCheck className="w-5 h-5" />} />
      </div>

      <div className="mb-8 rounded-lg border border-white/20 bg-white/[0.06] p-5">
        <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
          <div>
            <div className="flex items-center gap-2 text-white/45 text-xs font-mono uppercase tracking-[0.18em] mb-2">
              <ShieldCheck className="w-4 h-4" /> Shared OpenAI max plan
            </div>
            {maxPlan?.enabled ? (
              <div className="text-sm text-white/80">
                <span className="font-mono text-white">{maxPlan.email}</span>
                {' — '}
                <span className={maxPlan.healthy_count > 0 ? 'text-emerald-300' : 'text-amber-300'}>
                  {maxPlan.healthy_count}/{maxPlan.credential_count} credential(s) healthy
                </span>
                {maxPlan.credential_count === 0 && (
                  <span className="text-white/50"> — sign in with OpenAI on the Account page, then refresh</span>
                )}
                {maxPlan.credential_count > 0 && !maxPlan.refreshable && (
                  <div className="mt-1 text-xs text-amber-300">
                    auth_mode={maxPlan.auth_mode || 'unknown'} — no refresh token, so this is a pasted API key
                    that cannot be rotated. Sign in with OpenAI on the Account page to store a real max-plan
                    credential.
                  </div>
                )}
              </div>
            ) : (
              <div className="text-sm text-white/60">
                Disabled. Set <span className="font-mono">ADMIN_OPENAI_MAX_PLAN_EMAIL</span> and restart the API.
              </div>
            )}
            {maxPlanMsg && <div className="mt-1 text-xs text-white/50">{maxPlanMsg}</div>}
          </div>
          {maxPlan?.enabled && (
            <button
              onClick={() => void refreshMaxPlan()}
              className="shrink-0 rounded-md border border-emerald-400/30 bg-emerald-500/10 px-4 py-2 text-sm font-medium text-emerald-200 hover:bg-emerald-500/20"
            >
              Refresh now
            </button>
          )}
        </div>
      </div>

      <div className="overflow-hidden rounded-lg border border-white/20">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[1080px] text-left">
            <thead className="bg-white/[0.07] text-xs font-mono uppercase tracking-[0.16em] text-white/45">
              <tr>
                <SortableHeader label="User" sortKey="name" sort={sort} onSort={sortBy} />
                <SortableHeader label="Stripe net" sortKey="stripe_net_cents" sort={sort} onSort={sortBy} />
                <SortableHeader label="Stripe gross" sortKey="stripe_gross_cents" sort={sort} onSort={sortBy} />
                <SortableHeader label="API spend" sortKey="api_spend_cents" sort={sort} onSort={sortBy} />
                <SortableHeader label="Provider base" sortKey="provider_base_cost_cents" sort={sort} onSort={sortBy} />
                <SortableHeader label="Requests" sortKey="api_requests" sort={sort} onSort={sortBy} />
                <SortableHeader label="Balance" sortKey="balance_cents" sort={sort} onSort={sortBy} />
                <SortableHeader label="Last API use" sortKey="last_request_at" sort={sort} onSort={sortBy} />
              </tr>
            </thead>
            <tbody className="divide-y divide-white/10">
              {loading && users.length === 0 && (
                <tr>
                  <td colSpan={8} className="px-4 py-10 text-center font-mono text-sm text-white/55">
                    Loading admin spend...
                  </td>
                </tr>
              )}
              {!loading && users.length === 0 && !error && (
                <tr>
                  <td colSpan={8} className="px-4 py-10 text-center font-mono text-sm text-white/55">
                    No users found.
                  </td>
                </tr>
              )}
              {sortedUsers.map(user => (
                <tr key={user.user_id} className="text-sm text-white/80 hover:bg-white/[0.03]">
                  <td className="px-4 py-4">
                    <Link to={`/admin/users/${user.user_id}/usage`} className="font-medium text-white hover:text-emerald-300 hover:underline">
                      {user.name || 'Unnamed user'}
                    </Link>
                    <div className="font-mono text-xs text-white/45">{user.email}</div>
                    <div className="mt-1 flex gap-2 font-mono text-[11px] uppercase tracking-[0.14em] text-white/50">
                      {user.is_admin && <span className="text-emerald-300">Admin</span>}
                      {user.disabled && <span className="text-red-300">Disabled</span>}
                    </div>
                  </td>
                  <td className="px-4 py-4 font-mono">{formatStripeCents(user.stripe_net_cents)}</td>
                  <td className="px-4 py-4 font-mono text-white/55">
                    {formatStripeCents(user.stripe_gross_cents)}
                    {user.stripe_refunded_cents > 0 && <span className="block text-xs text-red-300">-{formatStripeCents(user.stripe_refunded_cents)} refunded</span>}
                  </td>
                  <td className="px-4 py-4 font-mono">{formatUsageUnits(user.api_spend_cents)}</td>
                  <td className="px-4 py-4 font-mono">
                    {formatUsageUnits(user.provider_base_cost_cents)}
                    {user.provider_estimated && <span className="block text-xs text-white/50">estimate</span>}
                  </td>
                  <td className="px-4 py-4 font-mono">{formatNumber(user.api_requests)}</td>
                  <td className="px-4 py-4 font-mono">{formatUsageUnits(user.balance_cents)}</td>
                  <td className="px-4 py-4 font-mono text-xs text-white/55">{formatDate(user.last_request_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function SortableHeader({
  label,
  sortKey,
  sort,
  onSort,
}: {
  label: string;
  sortKey: keyof AdminSpendUser;
  sort: { key: keyof AdminSpendUser; direction: 'asc' | 'desc' };
  onSort: (key: keyof AdminSpendUser) => void;
}) {
  const active = sort.key === sortKey;
  return (
    <th className="px-4 py-3">
      <button onClick={() => onSort(sortKey)} className="inline-flex items-center gap-1.5 hover:text-white transition-colors">
        {label}
        {active ? (sort.direction === 'asc' ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />) : <ArrowUpDown className="h-3.5 w-3.5 opacity-40" />}
      </button>
    </th>
  );
}

type AdminUsageEvent = {
  id: string;
  model: string;
  provider: string;
  tokens_in: number;
  tokens_out: number;
  cost_cents: number;
  status_code: number;
  error?: string;
  api_key_name?: string;
  app_url?: string;
  app_title?: string;
  created_at: string;
};

type AdminUsageModel = {
  model: string;
  provider: string;
  total_requests: number;
  total_tokens_in: number;
  total_tokens_out: number;
  total_cost_cents: number;
};

type AdminUsageApp = {
  app_id?: string;
  app_url?: string;
  app_title?: string;
  total_requests: number;
  total_tokens_in: number;
  total_tokens_out: number;
  total_cost_cents: number;
  models?: AdminUsageModel[];
};

type AdminActivityDay = { date: string; total_requests: number; total_cost_cents: number };

export function AdminUserUsage() {
  const { userId } = useParams<{ userId: string }>();
  const [period, setPeriod] = useState<'24h' | '7d' | '30d' | '90d'>('30d');
  const [data, setData] = useState<any>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!userId) return;
    setLoading(true);
    adminApi(`/admin/users/${encodeURIComponent(userId)}/usage?period=${period}&limit=100`)
      .then(async res => {
        const body = await res.json().catch(() => ({}));
        if (!res.ok) throw new Error(body.error?.message || 'Unable to load user usage');
        setData(body);
        setError('');
      })
      .catch(err => setError(err.message || 'Network error'))
      .finally(() => setLoading(false));
  }, [period, userId]);

  const models: AdminUsageModel[] = Array.isArray(data?.models) ? data.models : [];
  const apps: AdminUsageApp[] = Array.isArray(data?.apps) ? data.apps : [];
  const events: AdminUsageEvent[] = Array.isArray(data?.events) ? data.events : [];
  const activity: AdminActivityDay[] = Array.isArray(data?.activity) ? data.activity : [];
  const totalRequests = models.reduce((sum, item) => sum + item.total_requests, 0);
  const totalTokens = models.reduce((sum, item) => sum + item.total_tokens_in + item.total_tokens_out, 0);
  const totalCost = models.reduce((sum, item) => sum + item.total_cost_cents, 0);

  return (
    <div className="max-w-7xl mx-auto px-6 py-12">
      <Link to="/admin" className="mb-8 inline-flex items-center gap-2 font-mono text-sm text-white/50 hover:text-white">
        ← Back to Adminlee
      </Link>
      <div className="mb-8 flex flex-col gap-5 md:flex-row md:items-end md:justify-between">
        <div>
          <p className="mb-2 font-mono text-xs uppercase tracking-[0.22em] text-emerald-300">User activity</p>
          <h1 className="text-4xl font-semibold tracking-tight">{data?.user?.name || data?.user?.email || 'Usage detail'}</h1>
          <p className="mt-2 font-mono text-sm text-white/45">{data?.user?.email || userId}</p>
        </div>
        <div className="flex gap-2 font-mono text-sm">
          {(['24h', '7d', '30d', '90d'] as const).map(value => (
            <button key={value} onClick={() => setPeriod(value)} className={`rounded-lg px-3 py-1.5 ${period === value ? 'bg-white text-black font-bold' : 'bg-white/5 text-white/60 hover:bg-white/10'}`}>
              {value}
            </button>
          ))}
        </div>
      </div>
      {error && <div className="mb-6 rounded-lg border border-red-400/20 bg-red-500/10 p-4 text-red-200">{error}</div>}
      {loading ? <p className="font-mono text-sm text-white/40">Loading usage…</p> : (
        <>
          <div className="mb-8 grid gap-4 md:grid-cols-3">
            <Metric label="Requests" value={formatNumber(totalRequests)} icon={<Database className="h-5 w-5" />} />
            <Metric label="Tokens" value={formatNumber(totalTokens)} icon={<Database className="h-5 w-5" />} />
            <Metric label="API spend" value={formatUsageUnits(totalCost)} icon={<ShieldCheck className="h-5 w-5" />} />
          </div>
          <DailyActivitySummary data={activity} />
          <UsageTable title="By model" models={models} />
          <div className="mt-8 grid gap-8 lg:grid-cols-2">
            <AppUsageTable apps={apps} />
            <RecentUsageTable events={events} />
          </div>
        </>
      )}
    </div>
  );
}

function DailyActivitySummary({ data }: { data: AdminActivityDay[] }) {
  const recent = data.slice(-14).reverse();
  const max = Math.max(1, ...recent.map(day => day.total_requests));
  return (
    <section className="mb-8 rounded-lg border border-white/10 p-5">
      <h2 className="font-semibold">Daily activity</h2>
      <p className="mt-1 font-mono text-xs text-white/40">Recent request volume over the last year</p>
      {recent.length === 0 ? <p className="mt-6 font-mono text-sm text-white/40">No activity recorded.</p> : <div className="mt-5 grid gap-2 sm:grid-cols-2 lg:grid-cols-7">{recent.map(day => <div key={day.date} className="rounded border border-white/10 bg-white/[0.03] p-3"><div className="font-mono text-[11px] text-white/45">{day.date}</div><div className="mt-2 h-1.5 rounded bg-white/10"><div className="h-full rounded bg-emerald-300/80" style={{ width: `${Math.max(5, (day.total_requests / max) * 100)}%` }} /></div><div className="mt-2 font-mono text-xs text-white/70">{formatNumber(day.total_requests)} req</div><div className="font-mono text-[11px] text-emerald-300">{formatUsageUnits(day.total_cost_cents)}</div></div>)}</div>}
    </section>
  );
}

function UsageTable({ title, models }: { title: string; models: AdminUsageModel[] }) {
  return (
    <section className="overflow-hidden rounded-lg border border-white/10">
      <div className="border-b border-white/10 bg-white/[0.03] px-5 py-4"><h2 className="font-semibold">{title}</h2></div>
      <div className="overflow-x-auto"><table className="w-full text-left text-sm"><thead className="bg-white/[0.04] font-mono text-xs uppercase tracking-[0.14em] text-white/45"><tr><th className="px-5 py-3">Model</th><th className="px-5 py-3">Provider</th><th className="px-5 py-3 text-right">Requests</th><th className="px-5 py-3 text-right">Tokens</th><th className="px-5 py-3 text-right">Spend</th></tr></thead><tbody className="divide-y divide-white/10">{models.map(item => <tr key={`${item.model}-${item.provider}`}><td className="px-5 py-3 font-mono text-white">{item.model}</td><td className="px-5 py-3 text-white/50">{item.provider}</td><td className="px-5 py-3 text-right font-mono">{formatNumber(item.total_requests)}</td><td className="px-5 py-3 text-right font-mono text-white/60">{formatNumber(item.total_tokens_in + item.total_tokens_out)}</td><td className="px-5 py-3 text-right font-mono text-emerald-300">{formatUsageUnits(item.total_cost_cents)}</td></tr>)}</tbody></table></div>
      {models.length === 0 && <p className="px-5 py-8 font-mono text-sm text-white/40">No model usage in this period.</p>}
    </section>
  );
}

function AppUsageTable({ apps }: { apps: AdminUsageApp[] }) {
  return (
    <section className="overflow-hidden rounded-lg border border-white/10"><div className="border-b border-white/10 bg-white/[0.03] px-5 py-4"><h2 className="font-semibold">By app</h2><p className="mt-1 font-mono text-xs text-white/40">Caller attribution from Referer and X-Title</p></div><div className="divide-y divide-white/10">{apps.map((app, index) => <details key={`${app.app_id}-${app.app_url}-${index}`} className="group px-5 py-4"><summary className="cursor-pointer list-none"><div className="flex items-center justify-between gap-4"><div><div className="text-white">{app.app_title || app.app_url || 'Unnamed app'}</div><div className="mt-1 truncate font-mono text-xs text-white/40">{app.app_url || 'No URL supplied'}</div></div><div className="shrink-0 text-right font-mono text-xs"><div className="text-white/70">{formatNumber(app.total_requests)} req</div><div className="text-emerald-300">{formatUsageUnits(app.total_cost_cents)}</div></div></div></summary>{app.models?.length ? <div className="mt-3 space-y-2 border-l border-white/10 pl-3">{app.models.map(model => <div key={`${model.model}-${model.provider}`} className="flex justify-between gap-3 font-mono text-xs"><span className="text-white/60">{model.model} <span className="text-white/30">({model.provider})</span></span><span className="text-white/50">{formatNumber(model.total_tokens_in + model.total_tokens_out)} tok · {formatUsageUnits(model.total_cost_cents)}</span></div>)}</div> : null}</details>)}</div>{apps.length === 0 && <p className="px-5 py-8 font-mono text-sm text-white/40">No app attribution in this period.</p>}</section>
  );
}

function RecentUsageTable({ events }: { events: AdminUsageEvent[] }) {
  return (
    <section className="overflow-hidden rounded-lg border border-white/10"><div className="border-b border-white/10 bg-white/[0.03] px-5 py-4"><h2 className="font-semibold">Recent activity</h2><p className="mt-1 font-mono text-xs text-white/40">Latest 100 request records; prompts are not shown</p></div><div className="max-h-[560px] overflow-auto"><table className="w-full text-left text-xs"><thead className="sticky top-0 bg-[#0b0b0b] font-mono uppercase tracking-[0.12em] text-white/40"><tr><th className="px-4 py-3">Time</th><th className="px-4 py-3">Model</th><th className="px-4 py-3">App</th><th className="px-4 py-3 text-right">Tokens</th><th className="px-4 py-3 text-right">Spend</th></tr></thead><tbody className="divide-y divide-white/10">{events.map(event => <tr key={event.id}><td className="whitespace-nowrap px-4 py-3 font-mono text-white/45">{formatDate(event.created_at)}</td><td className="px-4 py-3"><div className="font-mono text-white/80">{event.model}</div><div className="text-white/35">{event.provider}</div></td><td className="max-w-[180px] truncate px-4 py-3 text-white/50" title={event.app_url || event.app_title}>{event.app_title || event.app_url || '—'}</td><td className="px-4 py-3 text-right font-mono text-white/60">{formatNumber(event.tokens_in + event.tokens_out)}</td><td className="px-4 py-3 text-right font-mono text-emerald-300">{formatUsageUnits(event.cost_cents)}</td></tr>)}</tbody></table></div>{events.length === 0 && <p className="px-5 py-8 font-mono text-sm text-white/40">No activity in this period.</p>}</section>
  );
}

function Metric({ label, value, icon }: { label: string; value: string; icon: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-white/20 bg-white/[0.06] p-5">
      <div className="mb-4 flex items-center justify-between text-white/45">
        <p className="font-mono text-xs uppercase tracking-[0.18em]">{label}</p>
        {icon}
      </div>
      <p className="font-mono text-2xl text-white">{value}</p>
    </div>
  );
}
