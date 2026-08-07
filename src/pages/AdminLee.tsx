import React, { useEffect, useMemo, useState } from 'react';
import { AlertTriangle, ArrowDownToLine, Database, RefreshCw, ShieldCheck } from 'lucide-react';
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
          className="inline-flex items-center justify-center gap-2 rounded-lg border border-white/10 bg-white text-black px-4 py-3 font-mono text-sm font-bold hover:bg-white/90 disabled:opacity-60"
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

      <div className="mb-8 rounded-lg border border-white/10 bg-white/[0.03] p-5">
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

      <div className="overflow-hidden rounded-lg border border-white/10">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[1080px] text-left">
            <thead className="bg-white/[0.04] text-xs font-mono uppercase tracking-[0.16em] text-white/45">
              <tr>
                <th className="px-4 py-3">User</th>
                <th className="px-4 py-3">Stripe net</th>
                <th className="px-4 py-3">Stripe gross</th>
                <th className="px-4 py-3">API spend</th>
                <th className="px-4 py-3">Provider base</th>
                <th className="px-4 py-3">Requests</th>
                <th className="px-4 py-3">Balance</th>
                <th className="px-4 py-3">Last API use</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/10">
              {loading && users.length === 0 && (
                <tr>
                  <td colSpan={8} className="px-4 py-10 text-center font-mono text-sm text-white/40">
                    Loading admin spend...
                  </td>
                </tr>
              )}
              {!loading && users.length === 0 && !error && (
                <tr>
                  <td colSpan={8} className="px-4 py-10 text-center font-mono text-sm text-white/40">
                    No users found.
                  </td>
                </tr>
              )}
              {users.map(user => (
                <tr key={user.user_id} className="text-sm text-white/80">
                  <td className="px-4 py-4">
                    <div className="font-medium text-white">{user.name || 'Unnamed user'}</div>
                    <div className="font-mono text-xs text-white/45">{user.email}</div>
                    <div className="mt-1 flex gap-2 font-mono text-[11px] uppercase tracking-[0.14em] text-white/35">
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
                    {user.provider_estimated && <span className="block text-xs text-white/35">estimate</span>}
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
