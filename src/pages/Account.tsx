import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import {
  Activity,
  ArrowUpRight,
  BarChart2,
  Check,
  ChevronLeft,
  CircleDollarSign,
  Copy,
  CreditCard,
  Eye,
  EyeOff,
  FileText,
  Image as ImageIcon,
  Key,
  LogOut,
  Plus,
  Repeat,
  Save,
  Shield,
  ShieldCheck,
  Sparkles,
  TriangleAlert,
  Wallet,
  X,
  Trash2,
} from 'lucide-react';
import {
  ResponsiveContainer,
  BarChart,
  Bar,
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  Cell,
  CartesianGrid,
  PieChart,
  Pie,
  Legend,
} from 'recharts';
import { motion, AnimatePresence } from 'motion/react';
import {
  Elements,
  PaymentElement,
  useElements,
  useStripe,
} from '@stripe/react-stripe-js';
import { TopUpModal, getStripe } from '../components/TopUpModal';
import { GuardrailsPanel } from '../components/GuardrailsPanel';
import { AUTH_EVENT, clearApiKey, setApiKey as storeApiKey } from '../lib/api';

const API_BASE = '';
const RECOMMENDED_THRESHOLD_USD = 100;
const RECOMMENDED_TOPUP_USD = 200;
const QUICK_TOPUP_AMOUNTS = [25, 100, 200, 500];

declare global {
  interface Window {
    userData?: { id: string; email: string; name: string; secret: string; authenticated: boolean };
  }
}

type PaymentMethod = {
  id: string;
  card?: {
    brand: string;
    last4: string;
    exp_month: number;
    exp_year: number;
  };
};

type AutotopupSettings = {
  enabled: boolean;
  threshold_cents: number;
  amount_cents: number;
};

function maskApiKey(key: string): string {
  const visible = Math.min(11, key.length);
  return key.slice(0, visible) + '•'.repeat(Math.max(8, key.length - visible));
}

function getUserData(): { apiKey: string | null; user: any } {
  if (window.userData?.secret) {
    return {
      apiKey: window.userData.secret,
      user: { id: window.userData.id, email: window.userData.email, name: window.userData.name },
    };
  }
  const apiKey = localStorage.getItem('op_api_key');
  let user = null;
  try {
    user = JSON.parse(localStorage.getItem('op_user') || 'null');
  } catch {}
  return { apiKey, user };
}

async function api(path: string, opts: RequestInit = {}) {
  const { apiKey } = getUserData();
  const isAuthEndpoint = path.startsWith('/auth/');
  if (!apiKey && !isAuthEndpoint) {
    return new Response(JSON.stringify({ error: { message: 'Not authenticated' } }), { status: 401 });
  }
  const headers: Record<string, string> = { 'Content-Type': 'application/json', ...(opts.headers as Record<string, string>) };
  if (apiKey) headers.Authorization = `Bearer ${apiKey}`;
  const res = await fetch(API_BASE + path, { ...opts, headers });
  if (res.status === 401 && !isAuthEndpoint) {
    window.userData = undefined;
    clearApiKey();
    window.dispatchEvent(new Event('auth-change'));
  }
  return res;
}

function parseBalanceUnits(data: any): number | null {
  if (typeof data?.balance_cents === 'number' && Number.isFinite(data.balance_cents)) {
    return data.balance_cents;
  }
  if (typeof data?.balance_usd === 'number' && Number.isFinite(data.balance_usd)) {
    return Math.round(data.balance_usd * 10000);
  }
  return null;
}

function usdToUnits(amount: number): number {
  return Math.round(amount * 10000);
}

function unitsToUSD(units: number): number {
  return units / 10000;
}

function formatBalanceUnits(units: number): string {
  const decimals = Math.abs(units) % 100 === 0 ? 2 : 4;
  return `$${(units / 10000).toFixed(decimals)}`;
}

function formatSignedUnits(units: number): string {
  if (units === 0) return formatBalanceUnits(0);
  return `${units > 0 ? '+' : '-'}${formatBalanceUnits(Math.abs(units))}`;
}

function formatTransactionDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return 'Unknown';
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

function formatUsdWhole(amount: number): string {
  return `$${amount.toLocaleString('en-US')}`;
}

const PRODUCT_META: Record<string, { label: string; color: string }> = {
  chat: { label: 'Chat & Text', color: '#6ee7b7' },
  image: { label: 'Image', color: '#a78bfa' },
  video: { label: 'Video', color: '#f472b6' },
  music: { label: 'Music', color: '#fbbf24' },
  speech: { label: 'Speech / TTS', color: '#38bdf8' },
  transcription: { label: 'Transcription', color: '#34d399' },
  embedding: { label: 'Embeddings', color: '#fb923c' },
  '3d': { label: '3D', color: '#c084fc' },
};

function productMeta(product: string): { label: string; color: string } {
  return PRODUCT_META[product] || { label: product.charAt(0).toUpperCase() + product.slice(1), color: '#94a3b8' };
}

function maskCard(pm: PaymentMethod): string {
  const brand = pm.card?.brand ? pm.card.brand.charAt(0).toUpperCase() + pm.card.brand.slice(1) : 'Card';
  return `${brand} ending in ${pm.card?.last4 || '....'}`;
}

function getBalanceTone(balanceUnits: number | null) {
  if (balanceUnits === null) {
    return {
      label: 'Loading',
      accent: 'text-white/60',
      pill: 'bg-white/10 text-white/70 border-white/15',
    };
  }
  if (balanceUnits < usdToUnits(RECOMMENDED_THRESHOLD_USD)) {
    return {
      label: 'Below reserve',
      accent: 'text-amber-300',
      pill: 'bg-amber-500/10 text-amber-200 border-amber-400/20',
    };
  }
  return {
    label: 'Healthy reserve',
    accent: 'text-emerald-300',
    pill: 'bg-emerald-500/10 text-emerald-200 border-emerald-400/20',
  };
}

function AuthForms({ onAuth }: { onAuth: (token: string, user: any, newApiKey?: string) => void }) {
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [name, setName] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const body = mode === 'register' ? { email, password, name } : { email, password };
      const res = await api(`/auth/${mode}`, { method: 'POST', body: JSON.stringify(body) });
      const data = await res.json();
      if (!res.ok) {
        setError(data.error?.message || 'Failed');
        return;
      }
	  const token = data.token || data.api_key;
	  if (data.api_key) storeApiKey(data.api_key);
	  if (token) localStorage.setItem('op_token', token);
      localStorage.setItem('op_user', JSON.stringify(data.user));
	  window.userData = { id: data.user.id, email: data.user.email, name: data.user.name, secret: token, authenticated: true };
      window.dispatchEvent(new Event('auth-change'));
	  onAuth(token, data.user, data.api_key);
    } catch {
      setError('Network error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-md mx-auto mt-20 px-6">
      <h1 className="text-3xl font-bold tracking-tight mb-8">{mode === 'login' ? 'Sign In' : 'Create Account'}</h1>
      <form onSubmit={submit} className="space-y-4">
        {mode === 'register' && (
          <input
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="Name"
            className="w-full bg-white/[0.06] border border-white/30 rounded-lg py-3 px-4 text-white font-mono focus:outline-none focus:border-white/50"
          />
        )}
        <input
          type="email"
          value={email}
          onChange={e => setEmail(e.target.value)}
          placeholder="Email"
          required
          className="w-full bg-white/[0.06] border border-white/30 rounded-lg py-3 px-4 text-white font-mono focus:outline-none focus:border-white/50"
          data-testid="auth-email"
        />
        <div className="relative">
          <input
            type={showPassword ? 'text' : 'password'}
            value={password}
            onChange={e => setPassword(e.target.value)}
            placeholder="Password"
            required
            minLength={8}
            className="w-full bg-white/[0.06] border border-white/30 rounded-lg py-3 px-4 pr-12 text-white font-mono focus:outline-none focus:border-white/50"
            data-testid="auth-password"
          />
          <button
            type="button"
            onClick={() => setShowPassword(v => !v)}
            aria-label={showPassword ? 'Hide password' : 'Show password'}
            data-testid="auth-password-toggle"
            className="absolute right-3 top-1/2 -translate-y-1/2 text-white/55 hover:text-white/70 transition-colors"
          >
            {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
          </button>
        </div>
        {error && <p className="text-red-400 text-sm font-mono">{error}</p>}
        <button
          type="submit"
          disabled={loading}
          className="w-full bg-white text-black py-3 rounded-lg font-mono font-bold text-sm hover:bg-white/90 transition-colors disabled:opacity-50"
          data-testid="auth-submit"
        >
          {loading ? 'Loading...' : mode === 'login' ? 'Sign In' : 'Create Account'}
        </button>
      </form>
      <button
        onClick={() => {
          setMode(mode === 'login' ? 'register' : 'login');
          setError('');
        }}
        className="mt-4 text-sm font-mono text-white/55 hover:text-white transition-colors"
        data-testid="auth-toggle"
      >
        {mode === 'login' ? 'Need an account? Register' : 'Already have an account? Sign In'}
      </button>
    </div>
  );
}

function SaveCardForm({
  onSuccess,
}: {
  onSuccess: () => Promise<void> | void;
}) {
  const stripe = useStripe();
  const elements = useElements();
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const saveCard = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!stripe || !elements) return;

    setSaving(true);
    setError('');
    try {
      const result = await stripe.confirmSetup({
        elements,
        redirect: 'if_required',
      });

      if (result.error) {
        setError(result.error.message || 'Failed to save card');
        return;
      }

      const paymentMethod = result.setupIntent?.payment_method;
      const paymentMethodID = typeof paymentMethod === 'string' ? paymentMethod : paymentMethod?.id;
      if (!paymentMethodID) {
        setError('Stripe did not return a payment method.');
        return;
      }

      const res = await api('/account/stripe/confirm', {
        method: 'POST',
        body: JSON.stringify({ payment_method_id: paymentMethodID }),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data.error?.message || 'Failed to save card');
        return;
      }

      await onSuccess();
    } catch {
      setError('Network error');
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={saveCard}>
      <div className="rounded-2xl border border-white/20 bg-black/30 p-4 mb-4">
        <PaymentElement />
      </div>
      {error && <p className="text-red-400 text-sm font-mono mb-4">{error}</p>}
      <button
        type="submit"
        disabled={saving || !stripe || !elements}
        className="w-full bg-white text-black py-3 rounded-2xl font-mono font-bold text-sm hover:bg-white/90 transition-colors disabled:opacity-50"
        data-testid="save-card-submit"
      >
        {saving ? 'Saving card...' : 'Save card for auto-topup'}
      </button>
    </form>
  );
}

function PaymentMethodSetupModal({
  open,
  stripePk,
  clientSecret,
  loading,
  error,
  onClose,
  onSaved,
}: {
  open: boolean;
  stripePk: string;
  clientSecret: string | null;
  loading: boolean;
  error: string;
  onClose: () => void;
  onSaved: () => Promise<void> | void;
}) {
  const appearance = {
    theme: 'night' as const,
    variables: {
      colorBackground: '#090909',
      colorText: '#ffffff',
      colorPrimary: '#ffffff',
      colorDanger: '#f87171',
      borderRadius: '16px',
    },
  };

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4"
          onClick={e => {
            if (e.target === e.currentTarget) onClose();
          }}
        >
          <motion.div
            initial={{ opacity: 0, scale: 0.96, y: 10 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.96, y: 10 }}
            className="bg-[#090909] border border-white/20 rounded-3xl w-full max-w-xl p-6 md:p-8"
            data-testid="payment-method-modal"
          >
            <div className="flex items-start justify-between gap-4 mb-6">
              <div>
                <p className="text-xs font-mono uppercase tracking-[0.2em] text-white/50 mb-2">Stripe card</p>
                <h2 className="text-2xl font-bold tracking-tight">Save a card for auto-topup</h2>
                <p className="text-sm text-white/55 mt-2">Your card stays in Stripe. OpenPaths uses it only for the prepaid auto-topup rule you save here.</p>
              </div>
              <button onClick={onClose} className="text-white/55 hover:text-white transition-colors shrink-0">
                <X className="w-5 h-5" />
              </button>
            </div>

            {loading && <p className="text-sm font-mono text-white/50">Preparing secure Stripe form...</p>}
            {!loading && error && <p className="text-sm font-mono text-red-400">{error}</p>}
            {!loading && !error && clientSecret && stripePk && (
              <Elements stripe={getStripe(stripePk)!} options={{ clientSecret, appearance }}>
                <SaveCardForm onSuccess={onSaved} />
              </Elements>
            )}
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

type ActivityDay = { date: string; total_requests: number; total_cost_cents: number };

function fmtDay(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

// GitHub-style contribution heatmap of API usage frequency over ~1 year.
function ContributionHeatmap({ data }: { data: ActivityDay[] }) {
  const byDate = new Map(data.map(d => [d.date, d]));
  const max = data.reduce((m, d) => Math.max(m, d.total_requests), 0);

  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const start = new Date(today);
  start.setDate(start.getDate() - 364);
  start.setDate(start.getDate() - start.getDay()); // align to Sunday

  const weeks: Date[][] = [];
  const cur = new Date(start);
  while (cur <= today) {
    const week: Date[] = [];
    for (let i = 0; i < 7; i++) {
      week.push(new Date(cur));
      cur.setDate(cur.getDate() + 1);
    }
    weeks.push(week);
  }

  const colors = [
    'rgba(255,255,255,0.05)',
    'rgba(110,231,183,0.28)',
    'rgba(110,231,183,0.5)',
    'rgba(110,231,183,0.72)',
    'rgba(110,231,183,1)',
  ];
  const level = (reqs: number) => {
    if (!reqs || max <= 0) return 0;
    const r = reqs / max;
    if (r > 0.66) return 4;
    if (r > 0.33) return 3;
    if (r > 0.1) return 2;
    return 1;
  };

  const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
  let lastMonth = -1;
  const monthLabels = weeks.map((week) => {
    const m = week[0].getMonth();
    if (m !== lastMonth) {
      lastMonth = m;
      return monthNames[m];
    }
    return '';
  });

  const totalRequests = data.reduce((s, d) => s + d.total_requests, 0);
  const activeDays = data.filter(d => d.total_requests > 0).length;

  return (
    <div>
      <div className="flex items-baseline justify-between mb-4">
        <p className="text-xs text-white/55 font-mono">
          {totalRequests.toLocaleString()} requests · {activeDays} active {activeDays === 1 ? 'day' : 'days'} in the last year
        </p>
      </div>
      <div className="overflow-x-auto pb-1">
        <div className="inline-flex flex-col gap-1" style={{ minWidth: 'max-content' }}>
          <div className="flex gap-[3px] ml-[26px] mb-1">
            {monthLabels.map((label, i) => (
              <div key={i} className="text-[9px] text-white/45 font-mono" style={{ width: 11 }}>
                {label}
              </div>
            ))}
          </div>
          <div className="flex">
            <div className="flex flex-col gap-[3px] mr-1 text-[9px] text-white/45 font-mono justify-around" style={{ height: 7 * 14 }}>
              <span>Mon</span>
              <span>Wed</span>
              <span>Fri</span>
            </div>
            <div className="flex gap-[3px]">
              {weeks.map((week, wi) => (
                <div key={wi} className="flex flex-col gap-[3px]">
                  {week.map((day, di) => {
                    const key = fmtDay(day);
                    const entry = byDate.get(key);
                    const reqs = entry?.total_requests || 0;
                    const future = day > today;
                    return (
                      <div
                        key={di}
                        title={future ? '' : `${key}: ${reqs.toLocaleString()} request${reqs === 1 ? '' : 's'}${entry ? ` · $${(entry.total_cost_cents / 10000).toFixed(4)}` : ''}`}
                        style={{
                          width: 11,
                          height: 11,
                          borderRadius: 2,
                          background: future ? 'transparent' : colors[level(reqs)],
                        }}
                      />
                    );
                  })}
                </div>
              ))}
            </div>
          </div>
          <div className="flex items-center gap-1 justify-end mt-2 text-[9px] text-white/45 font-mono">
            <span>Less</span>
            {colors.map((c, i) => (
              <div key={i} style={{ width: 11, height: 11, borderRadius: 2, background: c }} />
            ))}
            <span>More</span>
          </div>
        </div>
      </div>
    </div>
  );
}

function Toggle({ on, onClick }: { on: boolean; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`relative h-6 w-11 rounded-full transition-colors ${on ? 'bg-emerald-500/80' : 'bg-white/15'}`}
      aria-pressed={on}
    >
      <span className={`absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white transition-transform ${on ? 'translate-x-5' : 'translate-x-0'}`} />
    </button>
  );
}

// ResponseSavingCard lets the user opt into persisting their generation inputs +
// outputs, which makes /usage/prompts and /usage/images searchable.
function ResponseSavingCard() {
  const [text, setText] = useState(false);
  const [images, setImages] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    api('/account/usage/settings')
      .then(r => (r.ok ? r.json() : null))
      .then(d => {
        if (d) {
          setText(!!d.text_enabled);
          setImages(!!d.image_enabled);
        }
      })
      .catch(() => {})
      .finally(() => setLoaded(true));
  }, []);

  const persist = async (nextText: boolean, nextImages: boolean) => {
    setText(nextText);
    setImages(nextImages);
    setSaving(true);
    try {
      await api('/account/usage/settings', {
        method: 'POST',
        body: JSON.stringify({ text_enabled: nextText, image_enabled: nextImages }),
      });
    } catch {
      /* keep optimistic UI */
    } finally {
      setSaving(false);
    }
  };

  const anyOn = text || images;

  return (
    <div className="rounded-3xl border border-white/20 bg-black/35 p-6 mb-8" data-testid="response-saving-card">
      <div className="flex items-center gap-3 mb-1">
        <div className="rounded-2xl bg-violet-500/15 p-2.5">
          <Save className="w-5 h-5 text-violet-300" />
        </div>
        <div>
          <h3 className="text-lg font-semibold tracking-tight">Response saving</h3>
          <p className="text-xs font-mono uppercase tracking-[0.16em] text-white/50">Searchable prompt &amp; image history</p>
        </div>
        {saving && <span className="ml-auto text-xs text-white/55">saving…</span>}
      </div>
      <p className="text-sm text-white/55 mt-3 mb-5 max-w-2xl">
        Save the inputs and outputs of your generations to your private history, then search them semantically (find similar
        prompts, outputs, and images) or by exact text. Off by default; only your own account can see them.
      </p>

      <div className="space-y-3">
        <div className="flex items-center justify-between rounded-2xl border border-white/20 bg-white/[0.06] px-4 py-3">
          <div className="flex items-center gap-3">
            <FileText className="w-4 h-4 text-white/50" />
            <div>
              <div className="text-sm text-white">Save text generations</div>
              <div className="text-xs text-white/55">Chat &amp; messages — prompt, transcript, output</div>
            </div>
          </div>
          <Toggle on={text} onClick={() => loaded && persist(!text, images)} />
        </div>
        <div className="flex items-center justify-between rounded-2xl border border-white/20 bg-white/[0.06] px-4 py-3">
          <div className="flex items-center gap-3">
            <ImageIcon className="w-4 h-4 text-white/50" />
            <div>
              <div className="text-sm text-white">Save image generations</div>
              <div className="text-xs text-white/55">Prompt + generated image URL</div>
            </div>
          </div>
          <Toggle on={images} onClick={() => loaded && persist(text, !images)} />
        </div>
      </div>

      {anyOn && (
        <div className="mt-5 flex flex-wrap gap-3">
          {text && (
            <Link
              to="/usage/prompts"
              className="inline-flex items-center gap-2 rounded-xl border border-white/15 px-4 py-2 text-sm text-white hover:bg-white/10 transition-colors"
            >
              <Sparkles className="w-4 h-4" /> Search prompt history
            </Link>
          )}
          {images && (
            <Link
              to="/usage/images"
              className="inline-flex items-center gap-2 rounded-xl border border-white/15 px-4 py-2 text-sm text-white hover:bg-white/10 transition-colors"
            >
              <ImageIcon className="w-4 h-4" /> Search image history
            </Link>
          )}
        </div>
      )}
    </div>
  );
}

interface OpenAIDeviceAuthState {
  login_id: string;
  verification_url: string;
  user_code: string;
  interval_seconds: number;
  expires_at?: string;
}

interface OpenAIBrowserAuthState {
  login_id: string;
  authorization_url: string;
  redirect_uri: string;
  expires_at?: string;
}

const OPENAI_DEVICE_AUTH_SESSION_KEY = 'op_openai_device_auth';
const OPENAI_BROWSER_AUTH_SESSION_KEY = 'op_openai_browser_auth';

function loadPendingOpenAIAuth<T extends { expires_at?: string }>(key: string): T | null {
  if (typeof window === 'undefined') return null;
  try {
    const value = JSON.parse(window.sessionStorage.getItem(key) || 'null') as T | null;
    if (!value) return null;
    if (value.expires_at && new Date(value.expires_at).getTime() <= Date.now()) {
      window.sessionStorage.removeItem(key);
      return null;
    }
    return value;
  } catch {
    window.sessionStorage.removeItem(key);
    return null;
  }
}

function OpenAIMaxPlanPanel({
  connected,
  updatedAt,
  notice,
  error,
  deviceAuth,
  deviceStatus,
  deviceMessage,
  deviceLoading,
  browserAuth,
  browserInput,
  browserLoading,
  copied,
  onStartDevice,
  onPollDevice,
  onStartBrowser,
  onBrowserInput,
  onCompleteBrowser,
  onCopy,
}: {
  connected: boolean;
  updatedAt?: string;
  notice: string | null;
  error: string | null;
  deviceAuth: OpenAIDeviceAuthState | null;
  deviceStatus: 'idle' | 'pending' | 'polling' | 'error';
  deviceMessage: string;
  deviceLoading: boolean;
  browserAuth: OpenAIBrowserAuthState | null;
  browserInput: string;
  browserLoading: boolean;
  copied: string | null;
  onStartDevice: () => void;
  onPollDevice: () => void;
  onStartBrowser: () => void;
  onBrowserInput: (value: string) => void;
  onCompleteBrowser: () => void;
  onCopy: (value: string) => void;
}) {
  return (
    <section className="mb-8 border border-sky-300/20 bg-[radial-gradient(circle_at_top_right,rgba(14,165,233,0.14),transparent_42%),rgba(255,255,255,0.02)] rounded-3xl p-6" data-testid="openai-max-plan-panel">
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <div className="flex items-center gap-2 text-xs font-mono uppercase tracking-[0.18em] text-sky-100/45 mb-2">
            <ShieldCheck className="w-4 h-4" /> OpenAI Max plan
          </div>
          <h2 className="text-xl font-semibold tracking-tight">Sign in with OpenAI</h2>
          <p className="text-sm text-white/55 mt-2 max-w-2xl">
            Connect a ChatGPT subscription with Codex access. Device code is the simplest path; browser login is available if device authorization is disabled or blocked.
          </p>
        </div>
        <div className="flex shrink-0 flex-col gap-2 sm:flex-row">
          <button
            onClick={onStartDevice}
            disabled={deviceLoading || browserLoading}
            className="rounded-2xl bg-white text-black px-4 py-3 text-sm font-mono font-bold hover:bg-white/90 transition-colors disabled:opacity-50"
            data-testid="openai-auth-start"
          >
            {deviceLoading ? 'Starting...' : 'Sign in with device code'}
          </button>
          <button
            onClick={onStartBrowser}
            disabled={deviceLoading || browserLoading}
            className="rounded-2xl border border-white/15 bg-black/20 px-4 py-3 text-sm font-mono text-white hover:border-white/50 transition-colors disabled:opacity-50"
            data-testid="openai-browser-auth-start"
          >
            {browserLoading && !browserAuth ? 'Starting...' : 'Use browser callback'}
          </button>
        </div>
      </div>

      {deviceAuth && (
        <div className="mt-5 rounded-2xl border border-sky-400/20 bg-sky-500/10 p-4" data-testid="openai-device-auth-panel">
          <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div>
              <div className="text-xs font-mono uppercase tracking-[0.14em] text-sky-100/50 mb-2">Device sign-in</div>
              <p className="text-sm text-white/70 mb-3">{deviceMessage || 'Open the sign-in link and enter this code.'}</p>
              <div className="flex flex-wrap items-center gap-3">
                <code className="rounded-xl border border-white/20 bg-black/35 px-4 py-3 font-mono text-lg tracking-[0.18em] text-white" data-testid="openai-device-code">
                  {deviceAuth.user_code}
                </code>
                <button
                  onClick={() => onCopy(deviceAuth.user_code)}
                  className="rounded-xl border border-white/20 bg-black/25 p-3 text-white/60 transition-colors hover:text-white"
                  aria-label="Copy OpenAI device code"
                  data-testid="openai-device-code-copy"
                >
                  {copied === deviceAuth.user_code ? <Check className="h-4 w-4 text-green-400" /> : <Copy className="h-4 w-4" />}
                </button>
              </div>
              {deviceAuth.expires_at && (
                <p className="mt-3 text-xs font-mono text-white/50">
                  Expires {new Date(deviceAuth.expires_at).toLocaleTimeString()}
                </p>
              )}
            </div>
            <div className="flex flex-col gap-3 sm:flex-row md:flex-col">
              <a
                href={deviceAuth.verification_url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center gap-2 rounded-2xl bg-white px-4 py-3 text-sm font-mono font-bold text-black transition-colors hover:bg-white/90"
                data-testid="openai-device-auth-link"
              >
                Open sign-in link <ArrowUpRight className="h-4 w-4" />
              </a>
              <button
                onClick={onPollDevice}
                disabled={deviceStatus === 'polling'}
                className="rounded-2xl border border-white/15 bg-black/20 px-4 py-3 text-sm font-mono text-white transition-colors hover:border-white/50 disabled:opacity-50"
                data-testid="openai-device-auth-poll"
              >
                {deviceStatus === 'polling' ? 'Checking...' : 'Check status'}
              </button>
            </div>
          </div>
        </div>
      )}

      {browserAuth && (
        <div className="mt-5 rounded-2xl border border-violet-400/20 bg-violet-500/10 p-4" data-testid="openai-browser-auth-panel">
          <div className="flex flex-col gap-4">
            <div>
              <div className="text-xs font-mono uppercase tracking-[0.14em] text-violet-100/55 mb-2">Browser callback fallback</div>
              <p className="text-sm text-white/70">
                Open the login page. When OpenAI redirects to <code className="font-mono text-white/90">localhost:1455</code>, copy the full URL from the address bar—even if the page cannot connect—and paste it below.
              </p>
            </div>
            <div className="flex flex-col gap-3 md:flex-row">
              <a
                href={browserAuth.authorization_url}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex shrink-0 items-center justify-center gap-2 rounded-2xl bg-white px-4 py-3 text-sm font-mono font-bold text-black transition-colors hover:bg-white/90"
                data-testid="openai-browser-auth-link"
              >
                Open OpenAI login <ArrowUpRight className="h-4 w-4" />
              </a>
              <input
                value={browserInput}
                onChange={event => onBrowserInput(event.target.value)}
                placeholder="http://localhost:1455/auth/callback?code=...&state=..."
                className="min-w-0 flex-1 rounded-2xl border border-white/20 bg-black/35 px-4 py-3 text-sm font-mono text-white placeholder:text-white/45 focus:border-violet-300/40 focus:outline-none"
                data-testid="openai-browser-callback-input"
              />
              <button
                onClick={onCompleteBrowser}
                disabled={browserLoading || !browserInput.trim()}
                className="shrink-0 rounded-2xl border border-violet-300/30 bg-violet-400/10 px-4 py-3 text-sm font-mono font-bold text-violet-100 transition-colors hover:bg-violet-400/20 disabled:opacity-50"
                data-testid="openai-browser-auth-complete"
              >
                {browserLoading ? 'Connecting...' : 'Connect'}
              </button>
            </div>
            {browserAuth.expires_at && (
              <p className="text-xs font-mono text-white/50">Expires {new Date(browserAuth.expires_at).toLocaleTimeString()}</p>
            )}
          </div>
        </div>
      )}

      {notice && (
        <div className="mt-4 rounded-2xl border border-emerald-400/20 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
          {notice}
        </div>
      )}
      {error && (
        <div className="mt-4 rounded-2xl border border-red-400/20 bg-red-500/10 px-4 py-3 text-sm text-red-200">
          {error}
        </div>
      )}
      <div className="mt-5 rounded-2xl border border-white/20 bg-black/20 p-4">
        <div className="text-xs font-mono uppercase tracking-[0.14em] text-white/50 mb-2">Status</div>
        {connected ? (
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-sm text-white/75">OpenAI Codex sign-in saved.</p>
              <p className="text-xs text-white/55 mt-1">OAuth tokens refresh automatically before expiry; rejected credentials are refreshed and retried once.</p>
              {updatedAt && <p className="text-xs text-white/55 mt-1">Last updated {new Date(updatedAt).toLocaleString()}</p>}
            </div>
            <span className="rounded-full border border-emerald-400/20 bg-emerald-500/10 px-3 py-1 text-xs font-mono text-emerald-200">Connected</span>
          </div>
        ) : (
          <p className="text-sm text-white/45">No OpenAI Max plan sign-in is connected.</p>
        )}
      </div>
    </section>
  );
}

export function Account() {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState<'overview' | 'keys' | 'billing' | 'analytics' | 'guardrails'>(
    typeof window !== 'undefined' && window.location.pathname.startsWith('/usage') ? 'analytics' :
      (typeof window !== 'undefined' && (window.location.pathname === '/apikeys' || window.location.pathname === '/account/apikeys') ? 'keys' : 'overview'),
  );
  const [user, setUser] = useState<any>(null);
  const [apiKey, setApiKey] = useState<string | null>(null);
  const [stripePk, setStripePk] = useState('');

  const [balanceUnits, setBalanceUnits] = useState<number | null>(null);
  const [transactions, setTransactions] = useState<any[]>([]);
  const [apiKeys, setApiKeys] = useState<any[]>([]);
  const [selectedKeyIds, setSelectedKeyIds] = useState<string[]>([]);
  const selectingKeysRef = useRef(false);
  const [providerKeys, setProviderKeys] = useState<any[]>([]);
  const [paymentMethods, setPaymentMethods] = useState<PaymentMethod[]>([]);
  const [hasPaymentMethod, setHasPaymentMethod] = useState(false);
  const [autotopupSettings, setAutotopupSettings] = useState<AutotopupSettings>({
    enabled: false,
    threshold_cents: usdToUnits(RECOMMENDED_THRESHOLD_USD),
    amount_cents: usdToUnits(RECOMMENDED_TOPUP_USD),
  });

  const [copied, setCopied] = useState<string | null>(null);
  const [stripeModalOpen, setStripeModalOpen] = useState(false);
  const [checkoutAmount, setCheckoutAmount] = useState(RECOMMENDED_TOPUP_USD);
  const [newKeyName, setNewKeyName] = useState('');
  const [newKeyResult, setNewKeyResult] = useState<string | null>(null);
  const [newKeyVisible, setNewKeyVisible] = useState(false);
  const [openAIAuthNotice, setOpenAIAuthNotice] = useState<string | null>(null);
  const [openAIAuthError, setOpenAIAuthError] = useState<string | null>(null);
  const [openAIDeviceAuth, setOpenAIDeviceAuth] = useState<OpenAIDeviceAuthState | null>(() => loadPendingOpenAIAuth(OPENAI_DEVICE_AUTH_SESSION_KEY));
  const [openAIDeviceStatus, setOpenAIDeviceStatus] = useState<'idle' | 'pending' | 'polling' | 'error'>(() =>
    loadPendingOpenAIAuth(OPENAI_DEVICE_AUTH_SESSION_KEY) ? 'pending' : 'idle',
  );
  const [openAIDeviceMessage, setOpenAIDeviceMessage] = useState('');
  const [openAIDeviceLoading, setOpenAIDeviceLoading] = useState(false);
  const [openAIBrowserAuth, setOpenAIBrowserAuth] = useState<OpenAIBrowserAuthState | null>(() => loadPendingOpenAIAuth(OPENAI_BROWSER_AUTH_SESSION_KEY));
  const [openAIBrowserInput, setOpenAIBrowserInput] = useState('');
  const [openAIBrowserLoading, setOpenAIBrowserLoading] = useState(false);

  const [billingNotice, setBillingNotice] = useState<string | null>(null);
  const [billingError, setBillingError] = useState<string | null>(null);
  const [savingAutotopup, setSavingAutotopup] = useState(false);
  const [postTopupPrompt, setPostTopupPrompt] = useState(false);
  const autotopupCardRef = useRef<HTMLDivElement | null>(null);
  const [cardModalOpen, setCardModalOpen] = useState(false);
  const [cardSetupSecret, setCardSetupSecret] = useState<string | null>(null);
  const [cardSetupLoading, setCardSetupLoading] = useState(false);
  const [cardSetupError, setCardSetupError] = useState('');

  // Analytics state
  const [analyticsPeriod, setAnalyticsPeriod] = useState<'24h' | '7d' | '30d' | '90d'>('30d');
  const [spendTimeSeries, setSpendTimeSeries] = useState<{ timestamp: string; value: number }[]>([]);
  const [spendByKey, setSpendByKey] = useState<{ api_key_id: string; key_prefix: string; key_name: string; total_requests: number; total_cost_cents: number }[]>([]);
  const [spendByProvider, setSpendByProvider] = useState<{ provider: string; total_requests: number; total_cost_cents: number }[]>([]);
  const [spendByProduct, setSpendByProduct] = useState<{ product: string; total_requests: number; total_tokens_in: number; total_tokens_out: number; total_cost_cents: number }[]>([]);
  const [activity, setActivity] = useState<{ date: string; total_requests: number; total_cost_cents: number }[]>([]);
  const [drilldown, setDrilldown] = useState<{ type: 'key' | 'provider'; id: string; label: string; models: { model: string; provider: string; total_requests: number; total_cost_cents: number }[] } | null>(null);
  const [analyticsLoading, setAnalyticsLoading] = useState(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get('payment') === 'success') {
      setActiveTab('billing');
      setPostTopupPrompt(true);
      setBillingNotice('Funds added successfully. Set up auto-topup next so this balance keeps itself ready.');
      window.history.replaceState({}, '', '/account');
    }
    if (params.get('unsubscribe') === 'true') {
      setActiveTab('keys');
      api('/account/unsubscribe', { method: 'POST' })
        .then(r => r.json())
        .then(d => {
          setOpenAIAuthNotice(d.unsubscribed
            ? `${d.email} is unsubscribed from OpenPaths emails.`
            : 'Could not unsubscribe automatically — use the unsubscribe link in a recent email.');
        })
        .catch(() => {
          setOpenAIAuthNotice('Log in to unsubscribe, or use the unsubscribe link in a recent email.');
        });
      window.history.replaceState({}, '', '/account');
    }
    const openAIAuth = params.get('openai_auth');
    if (openAIAuth) {
      setActiveTab('keys');
      const message = params.get('openai_auth_message') || (openAIAuth === 'success' ? 'OpenAI sign-in saved.' : 'OpenAI sign-in failed.');
      if (openAIAuth === 'success') {
        setOpenAIAuthNotice(message);
      } else {
        setOpenAIAuthError(message);
      }
      window.history.replaceState({}, '', '/account');
    }
  }, []);

  useEffect(() => {
    const { apiKey: k, user: u } = getUserData();
    if (k && u) {
      setApiKey(k);
      setUser(u);
    }
  }, []);

  useEffect(() => {
    const sync = () => {
      const { apiKey: k, user: u } = getUserData();
      setApiKey(k);
      setUser(u);
    };
    window.addEventListener(AUTH_EVENT, sync);
    return () => window.removeEventListener(AUTH_EVENT, sync);
  }, []);

  useEffect(() => {
    fetch(API_BASE + '/account/stripe/config')
      .then(r => r.json())
      .then(d => {
        if (d.publishable_key) setStripePk(d.publishable_key);
      })
      .catch(() => {});
  }, []);

  const fetchBalance = useCallback(() => {
    if (!apiKey) return Promise.resolve();
    return api('/account/balance')
      .then(r => r.json())
      .then(d => {
        const nextBalance = parseBalanceUnits(d);
        if (nextBalance !== null) setBalanceUnits(nextBalance);
      })
      .catch(() => {});
  }, [apiKey]);

  const fetchTransactions = useCallback(() => {
    if (!apiKey) return Promise.resolve();
    return api('/account/transactions?limit=20')
      .then(r => r.json())
      .then(d => {
        if (d.transactions) setTransactions(d.transactions);
      })
      .catch(() => {});
  }, [apiKey]);

  const fetchKeys = useCallback(() => {
    if (!apiKey) return Promise.resolve();
    return api('/account/keys')
      .then(r => r.json())
      .then(d => {
        if (d.keys) setApiKeys(d.keys);
      })
      .catch(() => {});
  }, [apiKey]);

  const fetchProviderKeys = useCallback(() => {
    if (!apiKey) return Promise.resolve();
    return api('/account/provider-keys')
      .then(r => r.json())
      .then(d => {
        if (d.keys) setProviderKeys(d.keys);
      })
      .catch(() => {});
  }, [apiKey]);

  const fetchPaymentMethods = useCallback(() => {
    if (!apiKey) return Promise.resolve();
    return api('/account/stripe/payment-methods')
      .then(async r => {
        if (!r.ok) throw new Error('payment methods unavailable');
        return r.json();
      })
      .then(d => {
        const methods = Array.isArray(d.payment_methods) ? d.payment_methods : [];
        setPaymentMethods(methods);
        setHasPaymentMethod(methods.length > 0 || Boolean(d.default_payment_method_id));
      })
      .catch(() => {
        setPaymentMethods([]);
        setHasPaymentMethod(false);
      });
  }, [apiKey]);

  const fetchAutotopup = useCallback(() => {
    if (!apiKey) return Promise.resolve();
    return api('/account/autotopup/settings')
      .then(async r => {
        if (!r.ok) throw new Error('autotopup unavailable');
        return r.json();
      })
      .then(d => {
        setAutotopupSettings({
          enabled: Boolean(d.enabled),
          threshold_cents: typeof d.threshold_cents === 'number' ? d.threshold_cents : usdToUnits(RECOMMENDED_THRESHOLD_USD),
          amount_cents: typeof d.amount_cents === 'number' ? d.amount_cents : usdToUnits(RECOMMENDED_TOPUP_USD),
        });
        if (typeof d.has_payment_method === 'boolean') {
          setHasPaymentMethod(d.has_payment_method);
        }
      })
      .catch(() => {});
  }, [apiKey]);

  const refreshBilling = useCallback(async () => {
    await Promise.all([fetchBalance(), fetchTransactions(), fetchPaymentMethods(), fetchAutotopup()]);
  }, [fetchAutotopup, fetchBalance, fetchPaymentMethods, fetchTransactions]);

  useEffect(() => {
    if (!postTopupPrompt || activeTab !== 'billing') return;
    window.setTimeout(() => {
      autotopupCardRef.current?.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }, 150);
  }, [activeTab, postTopupPrompt]);

  const fetchAnalytics = useCallback(async (period: string) => {
    if (!apiKey) return;
    setAnalyticsLoading(true);
    const interval = period === '24h' ? '1h' : period === '7d' ? '6h' : '1d';
    try {
      const [tsRes, keyRes, provRes, prodRes] = await Promise.all([
        api(`/account/stats/timeseries?period=${period}&interval=${interval}&metric=cost`),
        api(`/account/stats/by-api-key?period=${period}`),
        api(`/account/stats/by-provider?period=${period}`),
        api(`/account/stats/by-product?period=${period}`),
      ]);
      const [tsData, keyData, provData, prodData] = await Promise.all([tsRes.json(), keyRes.json(), provRes.json(), prodRes.json()]);
      setSpendTimeSeries(Array.isArray(tsData.data) ? tsData.data : []);
      setSpendByKey(Array.isArray(keyData.keys) ? keyData.keys : []);
      setSpendByProvider(Array.isArray(provData.providers) ? provData.providers : []);
      setSpendByProduct(Array.isArray(prodData.products) ? prodData.products : []);
    } catch {}
    setAnalyticsLoading(false);
  }, [apiKey]);

  // The contribution heatmap always shows a fixed ~1 year window, independent
  // of the period selector, so it is loaded once when the tab opens.
  const fetchActivity = useCallback(async () => {
    if (!apiKey) return;
    try {
      const res = await api('/account/stats/activity?days=365');
      const data = await res.json();
      setActivity(Array.isArray(data.data) ? data.data : []);
    } catch {}
  }, [apiKey]);

  const loadDrilldown = async (type: 'key' | 'provider', id: string, label: string) => {
    const period = analyticsPeriod;
    const url = type === 'key'
      ? `/account/stats/by-api-key/${encodeURIComponent(id)}/models?period=${period}`
      : `/account/stats/by-provider/${encodeURIComponent(id)}/models?period=${period}`;
    const res = await api(url);
    const data = await res.json();
    const models = Array.isArray(data.models) ? data.models : [];
    setDrilldown({ type, id, label, models });
  };

  useEffect(() => {
    if (!apiKey) return;
    void Promise.all([fetchBalance(), fetchTransactions(), fetchKeys(), fetchProviderKeys(), fetchPaymentMethods(), fetchAutotopup()]);
  }, [apiKey, fetchAutotopup, fetchBalance, fetchKeys, fetchProviderKeys, fetchPaymentMethods, fetchTransactions]);

  useEffect(() => {
    if (activeTab === 'analytics' && apiKey) {
      setDrilldown(null);
      void fetchAnalytics(analyticsPeriod);
    }
  }, [activeTab, analyticsPeriod, apiKey, fetchAnalytics]);

  useEffect(() => {
    if (activeTab === 'analytics' && apiKey) {
      void fetchActivity();
    }
  }, [activeTab, apiKey, fetchActivity]);

  useEffect(() => {
    setNewKeyVisible(false);
  }, [newKeyResult]);

  const handleAuth = (key: string, u: any, newApiKey?: string) => {
    setApiKey(key);
    setUser(u);
    if (newApiKey) {
      setNewKeyResult(newApiKey);
      setActiveTab('keys');
    }
    const next = new URLSearchParams(window.location.search).get('next');
    if (next?.startsWith('/') && !next.startsWith('//')) window.location.assign(next);
  };

  const logout = () => {
    fetch('/auth/logout', { method: 'POST' }).catch(() => {});
    window.userData = undefined;
    clearApiKey();
    window.sessionStorage.removeItem(OPENAI_DEVICE_AUTH_SESSION_KEY);
    window.sessionStorage.removeItem(OPENAI_BROWSER_AUTH_SESSION_KEY);
    window.dispatchEvent(new Event('auth-change'));
    setApiKey(null);
    setUser(null);
  };

  const copyKey = (key: string) => {
    navigator.clipboard.writeText(key);
    setCopied(key);
    setTimeout(() => setCopied(null), 2000);
  };

  const createKey = async () => {
    const res = await api('/account/keys', { method: 'POST', body: JSON.stringify({ name: newKeyName || 'Default' }) });
    const data = await res.json();
    if (res.ok) {
      setNewKeyResult(data.key);
      setNewKeyVisible(false);
      storeApiKey(data.key);
      setApiKey(data.key);
      setNewKeyName('');
      void fetchKeys();
    }
  };

  const revokeKey = async (id: string) => {
    await api(`/account/keys/${id}`, { method: 'DELETE' });
    setSelectedKeyIds(ids => ids.filter(keyId => keyId !== id));
    void fetchKeys();
  };

  const toggleKeySelection = (id: string) => {
    setSelectedKeyIds(ids => ids.includes(id) ? ids.filter(keyId => keyId !== id) : [...ids, id]);
  };
  const beginKeySelection = (id: string) => {
    selectingKeysRef.current = true;
    setSelectedKeyIds([id]);
  };
  const extendKeySelection = (id: string) => {
    if (selectingKeysRef.current) setSelectedKeyIds(ids => ids.includes(id) ? ids : [...ids, id]);
  };
  const bulkRevokeKeys = async () => {
    const ids = selectedKeyIds;
    if (!ids.length) return;
    const res = await api('/account/keys', { method: 'DELETE', body: JSON.stringify({ ids }) });
    if (res.ok) {
      setSelectedKeyIds([]);
      void fetchKeys();
    }
  };

  useEffect(() => {
    const stopSelecting = () => { selectingKeysRef.current = false; };
    window.addEventListener('mouseup', stopSelecting);
    return () => window.removeEventListener('mouseup', stopSelecting);
  }, []);

  const startOpenAIDeviceAuth = async () => {
    setOpenAIDeviceLoading(true);
    setOpenAIAuthNotice(null);
    setOpenAIAuthError(null);
    setOpenAIBrowserAuth(null);
    setOpenAIBrowserInput('');
    window.sessionStorage.removeItem(OPENAI_BROWSER_AUTH_SESSION_KEY);
    setOpenAIDeviceMessage('');
    setOpenAIDeviceStatus('idle');
    try {
      const res = await api('/account/openai/device/start', { method: 'POST', body: JSON.stringify({}) });
      const data = await res.json();
      if (!res.ok || !data.login_id || !data.verification_url || !data.user_code) {
        setOpenAIAuthError(data.error?.message || 'Failed to start OpenAI device sign-in');
        setOpenAIDeviceStatus('error');
        return;
      }
      const pending: OpenAIDeviceAuthState = {
        login_id: data.login_id,
        verification_url: data.verification_url,
        user_code: data.user_code,
        interval_seconds: Number(data.interval_seconds) || 5,
        expires_at: data.expires_at,
      };
      setOpenAIDeviceAuth(pending);
      window.sessionStorage.setItem(OPENAI_DEVICE_AUTH_SESSION_KEY, JSON.stringify(pending));
      setOpenAIDeviceStatus('pending');
      setOpenAIDeviceMessage('Open the sign-in link and enter the code. This page will finish automatically.');
    } catch {
      setOpenAIAuthError('Network error');
      setOpenAIDeviceStatus('error');
    } finally {
      setOpenAIDeviceLoading(false);
    }
  };

  const pollOpenAIDeviceAuth = useCallback(async () => {
    if (!openAIDeviceAuth || openAIDeviceStatus === 'polling') return;
    setOpenAIDeviceStatus('polling');
    try {
      const res = await api('/account/openai/device/poll', {
        method: 'POST',
        body: JSON.stringify({ login_id: openAIDeviceAuth.login_id }),
      });
      const data = await res.json();
      if (res.status === 202 || data.status === 'pending') {
        setOpenAIDeviceStatus('pending');
        setOpenAIDeviceMessage(data.message || 'Waiting for OpenAI device authorization...');
        return;
      }
      if (!res.ok || data.status !== 'success') {
        setOpenAIDeviceStatus('error');
        setOpenAIAuthError(data.error?.message || 'OpenAI device sign-in failed');
        if (res.status === 400 || res.status === 410) {
          setOpenAIDeviceAuth(null);
          window.sessionStorage.removeItem(OPENAI_DEVICE_AUTH_SESSION_KEY);
        }
        return;
      }
      setOpenAIDeviceAuth(null);
      window.sessionStorage.removeItem(OPENAI_DEVICE_AUTH_SESSION_KEY);
      setOpenAIDeviceStatus('idle');
      setOpenAIDeviceMessage('');
      setOpenAIAuthNotice(data.message || 'OpenAI Max plan sign-in saved.');
      await fetchProviderKeys();
    } catch {
      setOpenAIDeviceStatus('pending');
      setOpenAIDeviceMessage('Still waiting for OpenAI device authorization...');
    }
  }, [fetchProviderKeys, openAIDeviceAuth, openAIDeviceStatus]);

  useEffect(() => {
    if (!openAIDeviceAuth || openAIDeviceStatus !== 'pending') return;
    const delay = Math.max(2, openAIDeviceAuth.interval_seconds || 5) * 1000;
    const timer = window.setTimeout(() => {
      void pollOpenAIDeviceAuth();
    }, delay);
    return () => window.clearTimeout(timer);
  }, [openAIDeviceAuth, openAIDeviceStatus, pollOpenAIDeviceAuth]);

  const startOpenAIBrowserAuth = async () => {
    setOpenAIBrowserLoading(true);
    setOpenAIAuthNotice(null);
    setOpenAIAuthError(null);
    setOpenAIDeviceAuth(null);
    window.sessionStorage.removeItem(OPENAI_DEVICE_AUTH_SESSION_KEY);
    setOpenAIDeviceStatus('idle');
    setOpenAIDeviceMessage('');
    setOpenAIBrowserInput('');
    try {
      const res = await api('/account/openai/browser/start', { method: 'POST', body: JSON.stringify({}) });
      const data = await res.json();
      if (!res.ok || !data.login_id || !data.authorization_url) {
        setOpenAIAuthError(data.error?.message || 'Failed to start OpenAI browser sign-in');
        return;
      }
      const pending: OpenAIBrowserAuthState = {
        login_id: data.login_id,
        authorization_url: data.authorization_url,
        redirect_uri: data.redirect_uri,
        expires_at: data.expires_at,
      };
      setOpenAIBrowserAuth(pending);
      window.sessionStorage.setItem(OPENAI_BROWSER_AUTH_SESSION_KEY, JSON.stringify(pending));
    } catch {
      setOpenAIAuthError('Network error');
    } finally {
      setOpenAIBrowserLoading(false);
    }
  };

  const completeOpenAIBrowserAuth = async () => {
    if (!openAIBrowserAuth || !openAIBrowserInput.trim()) return;
    setOpenAIBrowserLoading(true);
    setOpenAIAuthNotice(null);
    setOpenAIAuthError(null);
    try {
      const res = await api('/account/openai/browser/complete', {
        method: 'POST',
        body: JSON.stringify({
          login_id: openAIBrowserAuth.login_id,
          callback_url: openAIBrowserInput.trim(),
        }),
      });
      const data = await res.json();
      if (!res.ok || data.status !== 'success') {
        setOpenAIAuthError(data.error?.message || 'OpenAI browser sign-in failed');
        if ((res.status === 400 && data.error?.code !== 'invalid_request') || res.status === 410) {
          setOpenAIBrowserAuth(null);
          window.sessionStorage.removeItem(OPENAI_BROWSER_AUTH_SESSION_KEY);
        }
        return;
      }
      setOpenAIBrowserAuth(null);
      window.sessionStorage.removeItem(OPENAI_BROWSER_AUTH_SESSION_KEY);
      setOpenAIBrowserInput('');
      setOpenAIAuthNotice(data.message || 'OpenAI Max plan sign-in saved.');
      await fetchProviderKeys();
    } catch {
      setOpenAIAuthError('Network error');
    } finally {
      setOpenAIBrowserLoading(false);
    }
  };

  const openCheckout = (amountUSD: number) => {
    setCheckoutAmount(amountUSD);
    setStripeModalOpen(true);
  };

  const startCardSetup = async () => {
    setCardModalOpen(true);
    setCardSetupLoading(true);
    setCardSetupError('');
    setCardSetupSecret(null);
    setBillingError(null);

    try {
      const res = await api('/account/stripe/setup', { method: 'POST', body: JSON.stringify({}) });
      const data = await res.json();
      if (!res.ok) {
        setCardSetupError(data.error?.message || 'Failed to prepare Stripe card form');
        return;
      }
      setCardSetupSecret(data.client_secret);
    } catch {
      setCardSetupError('Network error');
    } finally {
      setCardSetupLoading(false);
    }
  };

  const handleCardSaved = async () => {
    setCardModalOpen(false);
    setCardSetupSecret(null);
    setBillingNotice(
      postTopupPrompt
        ? 'Card saved. Review the recommended auto-topup rule and save it to finish setup.'
        : 'Card saved. Auto-topup can now be enabled.',
    );
    await refreshBilling();
  };

  const deletePaymentMethod = async (paymentMethodID: string) => {
    setBillingError(null);
    setBillingNotice(null);
    const res = await api(`/account/stripe/payment-methods/${paymentMethodID}`, { method: 'DELETE' });
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      setBillingError(data.error?.message || 'Failed to remove card');
      return;
    }
    setBillingNotice('Saved card removed.');
    await refreshBilling();
  };

  const saveAutotopup = async () => {
    setSavingAutotopup(true);
    setBillingNotice(null);
    setBillingError(null);
    try {
      const res = await api('/account/autotopup/settings', {
        method: 'POST',
        body: JSON.stringify(autotopupSettings),
      });
      const data = await res.json();
      if (!res.ok) {
        setBillingError(data.error?.message || 'Failed to save auto-topup');
        return;
      }
      setBillingNotice(
        autotopupSettings.enabled
          ? `Auto-topup will add ${formatBalanceUnits(autotopupSettings.amount_cents)} when your balance falls below ${formatBalanceUnits(autotopupSettings.threshold_cents)}.`
          : 'Auto-topup disabled.',
      );
      if (autotopupSettings.enabled) {
        setPostTopupPrompt(false);
      }
      await refreshBilling();
    } catch {
      setBillingError('Network error');
    } finally {
      setSavingAutotopup(false);
    }
  };

  const enableRecommendedAutotopup = async () => {
    const recommended = {
      enabled: true,
      threshold_cents: usdToUnits(RECOMMENDED_THRESHOLD_USD),
      amount_cents: usdToUnits(RECOMMENDED_TOPUP_USD),
    };
    setAutotopupSettings(recommended);
    setSavingAutotopup(true);
    setBillingNotice(null);
    setBillingError(null);
    try {
      const res = await api('/account/autotopup/settings', {
        method: 'POST',
        body: JSON.stringify(recommended),
      });
      const data = await res.json();
      if (!res.ok) {
        setBillingError(data.error?.message || 'Failed to save auto-topup');
        return;
      }
      setPostTopupPrompt(false);
      setBillingNotice(`Auto-topup will add ${formatUsdWhole(RECOMMENDED_TOPUP_USD)} when your balance falls below ${formatUsdWhole(RECOMMENDED_THRESHOLD_USD)}.`);
      await refreshBilling();
    } catch {
      setBillingError('Network error');
    } finally {
      setSavingAutotopup(false);
    }
  };

  if (!apiKey) return <AuthForms onAuth={handleAuth} />;

  const balanceDisplay = balanceUnits !== null ? formatBalanceUnits(balanceUnits) : '--';
  const balanceTone = getBalanceTone(balanceUnits);
  const reserveGapUnits = balanceUnits === null ? 0 : Math.max(usdToUnits(RECOMMENDED_THRESHOLD_USD) - balanceUnits, 0);
  const hasLowReserve = balanceUnits !== null && balanceUnits < usdToUnits(RECOMMENDED_THRESHOLD_USD);
  const hasCards = paymentMethods.length > 0 || hasPaymentMethod;
  const recommendedRuleCopy = `${formatUsdWhole(RECOMMENDED_TOPUP_USD)} when balance falls below ${formatUsdWhole(RECOMMENDED_THRESHOLD_USD)}`;

  return (
    <div className="max-w-7xl mx-auto px-6 py-12 flex flex-col md:flex-row gap-12">
      <aside className="w-full md:w-64 shrink-0">
        <div className="mb-8">
          <h2 className="text-xl font-bold tracking-tight mb-1">Account</h2>
          <p className="text-sm font-mono text-white/55">{user?.email}</p>
          <button onClick={logout} className="mt-2 text-xs font-mono text-white/45 hover:text-white flex items-center gap-1" data-testid="logout-btn">
            <LogOut className="w-3 h-3" /> Sign Out
          </button>
        </div>
        <nav className="flex flex-col gap-2 font-mono text-sm">
          <button
            onClick={() => { setActiveTab('overview'); navigate('/account'); }}
            data-testid="tab-overview"
            className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors text-left ${
              activeTab === 'overview' ? 'bg-white/10 text-white' : 'text-white/60 hover:bg-white/10 hover:text-white'
            }`}
          >
            <Activity className="w-4 h-4" /> Overview
          </button>
          <button
            onClick={() => { setActiveTab('keys'); navigate('/account/apikeys'); }}
            data-testid="tab-keys"
            className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors text-left ${
              activeTab === 'keys' ? 'bg-white/10 text-white' : 'text-white/60 hover:bg-white/10 hover:text-white'
            }`}
          >
            <Key className="w-4 h-4" /> API Keys
          </button>
          <button
            onClick={() => { setActiveTab('guardrails'); navigate('/account?tab=guardrails'); }}
            data-testid="tab-guardrails"
            className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors text-left ${
              activeTab === 'guardrails' ? 'bg-white/10 text-white' : 'text-white/60 hover:bg-white/10 hover:text-white'
            }`}
          >
            <Shield className="w-4 h-4" /> Guardrails
          </button>
          <button
            onClick={() => { setActiveTab('billing'); navigate('/account?tab=billing'); }}
            data-testid="tab-billing"
            className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors text-left ${
              activeTab === 'billing' ? 'bg-white/10 text-white' : 'text-white/60 hover:bg-white/10 hover:text-white'
            }`}
          >
            <CreditCard className="w-4 h-4" /> Billing
          </button>
          <button
            onClick={() => { setActiveTab('analytics'); navigate('/usage'); }}
            data-testid="tab-analytics"
            className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors text-left ${
              activeTab === 'analytics' ? 'bg-white/10 text-white' : 'text-white/60 hover:bg-white/10 hover:text-white'
            }`}
          >
            <BarChart2 className="w-4 h-4" /> Usage
          </button>
        </nav>
      </aside>

      <main className="flex-1 min-w-0">
        {activeTab === 'overview' && (
          <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
            <div className="relative overflow-hidden rounded-[28px] border border-white/20 bg-[radial-gradient(circle_at_top_left,rgba(16,185,129,0.16),transparent_42%),radial-gradient(circle_at_top_right,rgba(245,158,11,0.14),transparent_34%),linear-gradient(180deg,rgba(255,255,255,0.04),rgba(255,255,255,0.02))] p-7 md:p-8 mb-8">
              <div className="absolute inset-0 bg-[linear-gradient(120deg,transparent,rgba(255,255,255,0.05),transparent)] opacity-40 pointer-events-none" />
              <div className="relative flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
                <div className="max-w-2xl">
                  <p className="text-xs font-mono uppercase tracking-[0.24em] text-white/50 mb-3">Prepaid credits</p>
                  <h1 className="text-4xl md:text-5xl font-semibold tracking-tight mb-4">Keep spend predictable.</h1>
                  <p className="text-base text-white/65 max-w-xl">
                    Stripe handles the payment side. OpenPaths stays prepaid, so the balance you see here is still the source of truth for runtime usage.
                  </p>
                  <div className="mt-6 flex flex-wrap items-center gap-3">
                    <button
                      onClick={() => openCheckout(RECOMMENDED_TOPUP_USD)}
                      className="rounded-2xl bg-white text-black px-5 py-3 text-sm font-mono font-bold hover:bg-white/90 transition-colors"
                    >
                      Add {formatUsdWhole(RECOMMENDED_TOPUP_USD)}
                    </button>
                    <button
                      onClick={() => setActiveTab('billing')}
                      className="rounded-2xl border border-white/15 bg-white/[0.06] px-5 py-3 text-sm font-mono text-white hover:border-white/50 transition-colors inline-flex items-center gap-2"
                    >
                      Review billing <ArrowUpRight className="w-4 h-4" />
                    </button>
                  </div>
                </div>

                <div className="grid sm:grid-cols-2 gap-4 min-w-[min(100%,28rem)]">
                  <div className="rounded-3xl border border-white/20 bg-black/35 p-5" data-testid="balance-card">
                    <div className="text-xs font-mono uppercase tracking-[0.16em] text-white/50 mb-2">Current balance</div>
                    <div className="text-4xl font-light tracking-tight mb-3" data-testid="balance">{balanceDisplay}</div>
                    <span className={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-mono ${balanceTone.pill}`}>{balanceTone.label}</span>
                  </div>

                  <div className="rounded-3xl border border-white/20 bg-black/35 p-5" data-testid="overview-autotopup-card">
                    <div className="text-xs font-mono uppercase tracking-[0.16em] text-white/50 mb-2">Auto-topup rule</div>
                    <div className="text-2xl font-semibold tracking-tight mb-2">{autotopupSettings.enabled ? 'Enabled' : 'Recommended'}</div>
                    <p className="text-sm text-white/60 mb-4">
                      {autotopupSettings.enabled
                        ? `${formatBalanceUnits(autotopupSettings.amount_cents)} when balance falls below ${formatBalanceUnits(autotopupSettings.threshold_cents)}`
                        : recommendedRuleCopy}
                    </p>
                    <button
                      onClick={() => setActiveTab('billing')}
                      className="text-xs font-mono text-white border border-white/15 px-3 py-2 rounded-xl hover:bg-white/10 transition-colors"
                    >
                      {autotopupSettings.enabled ? 'Manage rule' : 'Set up auto-topup'}
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <ResponseSavingCard />

            {hasLowReserve && (
              <div className="grid grid-cols-1 lg:grid-cols-[1.4fr_1fr] gap-6 mb-8">
                <div className="rounded-3xl border border-amber-400/20 bg-[linear-gradient(135deg,rgba(245,158,11,0.13),rgba(255,255,255,0.02))] p-6" data-testid="low-balance-banner">
                  <div className="flex items-start gap-4">
                    <div className="rounded-2xl bg-amber-500/15 p-3 mt-1">
                      <TriangleAlert className="w-5 h-5 text-amber-300" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <h2 className="text-xl font-semibold tracking-tight">You are below the recommended reserve</h2>
                      <p className="text-sm text-white/65 mt-2">
                        You are short {formatBalanceUnits(reserveGapUnits)} from the recommended {formatUsdWhole(RECOMMENDED_THRESHOLD_USD)} buffer.
                      </p>
                      <div className="mt-4 flex flex-wrap gap-3">
                        <button
                          onClick={() => openCheckout(RECOMMENDED_TOPUP_USD)}
                          className="rounded-2xl bg-white text-black px-4 py-3 text-sm font-mono font-bold hover:bg-white/90 transition-colors"
                        >
                          Add {formatUsdWhole(RECOMMENDED_TOPUP_USD)}
                        </button>
                        <button
                          onClick={() => setActiveTab('billing')}
                          className="rounded-2xl border border-white/15 bg-black/20 px-4 py-3 text-sm font-mono text-white hover:border-white/50 transition-colors"
                        >
                          Configure billing
                        </button>
                      </div>
                    </div>
                  </div>
                </div>

                <div className="rounded-3xl border border-white/20 bg-white/[0.06] p-6">
                  <div className="text-xs font-mono uppercase tracking-[0.16em] text-white/50 mb-3">Quick top-up</div>
                  <div className="grid grid-cols-2 gap-3">
                    {QUICK_TOPUP_AMOUNTS.slice(0, 4).map(amount => (
                      <button
                        key={amount}
                        onClick={() => openCheckout(amount)}
                        className={`rounded-2xl border px-4 py-4 text-left transition-colors ${
                          amount === RECOMMENDED_TOPUP_USD
                            ? 'border-emerald-300/25 bg-emerald-500/10'
                            : 'border-white/20 bg-black/20 hover:border-white/50'
                        }`}
                      >
                        <div className="text-2xl font-semibold">{formatUsdWhole(amount)}</div>
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            )}

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-6 mb-10">
              <div className="border border-white/20 bg-white/[0.05] rounded-3xl p-6">
                <div className="text-sm font-mono text-white/55 mb-2">API Keys</div>
                <div className="text-4xl font-light tracking-tight mb-4" data-testid="keys-count">{apiKeys.length}</div>
                <button onClick={() => { setActiveTab('keys'); navigate('/account/apikeys'); }} className="text-xs font-mono text-white border border-white/20 px-3 py-2 rounded-xl hover:bg-white/10 transition-colors">
                  Manage keys
                </button>
              </div>
              <div className="border border-white/20 bg-white/[0.05] rounded-3xl p-6">
                <div className="text-sm font-mono text-white/55 mb-2">Saved payment method</div>
                <div className="text-2xl font-semibold tracking-tight mb-4">{hasCards ? 'Ready for auto-topup' : 'No card saved'}</div>
                <button
                  onClick={() => setActiveTab('billing')}
                  className="text-xs font-mono text-white border border-white/20 px-3 py-2 rounded-xl hover:bg-white/10 transition-colors"
                >
                  {hasCards ? 'Review billing' : 'Add a card'}
                </button>
              </div>
            </div>

            {transactions.length > 0 && (
              <>
                <div className="flex items-center justify-between mb-4">
                  <h2 className="text-xl font-bold tracking-tight">Recent transactions</h2>
                  <button onClick={() => setActiveTab('billing')} className="text-xs font-mono text-white/45 hover:text-white transition-colors">
                    View billing history
                  </button>
                </div>
                <div className="border border-white/20 rounded-3xl overflow-hidden">
                  <table className="w-full text-left text-sm" data-testid="activity-table">
                    <thead className="bg-white/10 font-mono text-xs text-white/55 border-b border-white/20">
                      <tr>
                        <th className="px-6 py-3 font-normal">Date</th>
                        <th className="px-6 py-3 font-normal">Description</th>
                        <th className="px-6 py-3 font-normal">Amount</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-white/10 font-mono">
                      {transactions.slice(0, 10).map((tx: any) => (
                        <tr key={tx.id}>
                          <td className="px-6 py-4 text-white/60">{formatTransactionDate(tx.created_at)}</td>
                          <td className="px-6 py-4">{tx.description}</td>
                          <td className={`px-6 py-4 ${tx.amount_cents > 0 ? 'text-emerald-300' : 'text-red-400'}`}>{formatSignedUnits(tx.amount_cents)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </>
            )}
          </motion.div>
        )}

        {activeTab === 'keys' && (
          <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
            <div className="flex justify-between items-center mb-8">
              <h1 className="text-3xl font-bold tracking-tight">API Keys</h1>
              {selectedKeyIds.length > 0 && (
                <button onClick={() => void bulkRevokeKeys()} className="flex items-center gap-2 rounded border border-red-400/30 px-3 py-2 text-xs font-mono text-red-300 hover:bg-red-500/10">
                  <Trash2 className="w-4 h-4" /> Delete {selectedKeyIds.length} selected
                </button>
              )}
            </div>

            <OpenAIMaxPlanPanel
              connected={providerKeys.some(k => k.provider === 'openai_codex' && k.has_auth)}
              updatedAt={providerKeys.find(k => k.provider === 'openai_codex')?.updated_at}
              notice={openAIAuthNotice}
              error={openAIAuthError}
              deviceAuth={openAIDeviceAuth}
              deviceStatus={openAIDeviceStatus}
              deviceMessage={openAIDeviceMessage}
              deviceLoading={openAIDeviceLoading}
              browserAuth={openAIBrowserAuth}
              browserInput={openAIBrowserInput}
              browserLoading={openAIBrowserLoading}
              copied={copied}
              onStartDevice={() => void startOpenAIDeviceAuth()}
              onPollDevice={() => void pollOpenAIDeviceAuth()}
              onStartBrowser={() => void startOpenAIBrowserAuth()}
              onBrowserInput={setOpenAIBrowserInput}
              onCompleteBrowser={() => void completeOpenAIBrowserAuth()}
              onCopy={copyKey}
            />

            {newKeyResult && (
              <div className="border border-green-500/30 bg-green-500/5 rounded-xl p-4 mb-6" data-testid="new-key-banner">
                <p className="text-sm text-green-400 mb-2 font-bold">New key created. Save it now because it will not be shown again.</p>
                <div className="flex items-center gap-2 bg-white/[0.06] border border-white/30 rounded-lg p-3">
                  <code className="flex-1 font-mono text-sm text-white/80 break-all" data-testid="new-key-value">
                    {newKeyVisible ? newKeyResult : maskApiKey(newKeyResult)}
                  </code>
                  <button
                    type="button"
                    onClick={() => setNewKeyVisible(v => !v)}
                    aria-label={newKeyVisible ? 'Hide API key' : 'Show API key'}
                    data-testid="new-key-toggle"
                    className="p-2 hover:bg-white/10 rounded transition-colors"
                  >
                    {newKeyVisible ? <EyeOff className="w-4 h-4 text-white/60" /> : <Eye className="w-4 h-4 text-white/60" />}
                  </button>
                  <button
                    type="button"
                    onClick={() => copyKey(newKeyResult)}
                    aria-label="Copy API key"
                    data-testid="new-key-copy"
                    className="p-2 hover:bg-white/10 rounded transition-colors"
                  >
                    {copied === newKeyResult ? <Check className="w-4 h-4 text-green-400" /> : <Copy className="w-4 h-4 text-white/60" />}
                  </button>
                </div>
              </div>
            )}

            <div className="flex gap-3 mb-6">
              <input
                value={newKeyName}
                onChange={e => setNewKeyName(e.target.value)}
                placeholder="Key name (optional)"
                className="flex-1 bg-white/[0.06] border border-white/30 rounded-lg py-2 px-4 text-white font-mono text-sm focus:outline-none focus:border-white/50"
              />
              <button
                onClick={createKey}
                className="bg-white text-black px-4 py-2 text-sm font-mono font-bold hover:bg-white/90 transition-colors flex items-center gap-2 rounded"
                data-testid="create-key-btn"
              >
                <Plus className="w-4 h-4" /> Create Key
              </button>
            </div>

            {apiKeys.length === 0 ? (
              <p className="text-white/55 font-mono text-sm">No API keys yet. Create one to get started.</p>
            ) : (
              <div className="space-y-3">
                {apiKeys.map((k: any) => (
                  <div key={k.id} onMouseDown={() => beginKeySelection(k.id)} onMouseEnter={() => extendKeySelection(k.id)} className={`border bg-white/[0.05] rounded-xl p-4 flex items-center justify-between select-none cursor-pointer ${selectedKeyIds.includes(k.id) ? 'border-red-400/60 bg-red-500/10' : 'border-white/20'}`} data-testid="api-key-card">
                    <div>
                      <input type="checkbox" checked={selectedKeyIds.includes(k.id)} onChange={() => toggleKeySelection(k.id)} onClick={e => e.stopPropagation()} aria-label={`Select ${k.name}`} className="mr-3 accent-red-400" />
                      <div className="font-bold text-sm">{k.name}</div>
                      <code className="font-mono text-xs text-white/55">{k.key_prefix}...</code>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="px-2 py-1 bg-green-500/10 text-green-400 text-[10px] font-mono rounded border border-green-500/20" data-testid="key-status">
                        Active
                      </span>
                      <button onClick={() => revokeKey(k.id)} className="text-xs font-mono text-red-400/60 hover:text-red-400 px-2 py-1">
                        Revoke
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}

            <p className="text-sm text-white/55 font-light mt-6">Do not share your API key in publicly accessible areas such as GitHub or client-side code.</p>
          </motion.div>
        )}

        {activeTab === 'guardrails' && (
          <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
            <GuardrailsPanel apiKeys={apiKeys} />
          </motion.div>
        )}

        {activeTab === 'billing' && (
          <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
            <div className="flex items-start justify-between gap-4 mb-8">
              <div>
                <h1 className="text-3xl font-bold tracking-tight">Billing & Payments</h1>
                <p className="text-sm text-white/55 mt-2">Stripe is the payment source of truth. Credits remain prepaid on OpenPaths.</p>
              </div>
            </div>

            {billingNotice && (
              <div className="mb-6 rounded-2xl border border-emerald-400/20 bg-emerald-500/10 px-4 py-3 text-sm text-emerald-200">
                {billingNotice}
              </div>
            )}
            {billingError && (
              <div className="mb-6 rounded-2xl border border-red-400/20 bg-red-500/10 px-4 py-3 text-sm text-red-200">
                {billingError}
              </div>
            )}

            <div className="grid gap-6 lg:grid-cols-[1.25fr_0.95fr] mb-8">
              <div className="relative overflow-hidden rounded-[28px] border border-white/20 bg-[radial-gradient(circle_at_top_left,rgba(16,185,129,0.18),transparent_40%),linear-gradient(180deg,rgba(255,255,255,0.04),rgba(255,255,255,0.02))] p-7">
                <div className="absolute inset-0 bg-[linear-gradient(115deg,transparent,rgba(255,255,255,0.05),transparent)] opacity-40 pointer-events-none" />
                <div className="relative">
                  <div className="flex flex-wrap items-start justify-between gap-4 mb-6">
                    <div>
                      <p className="text-xs font-mono uppercase tracking-[0.2em] text-white/50 mb-2">Prepaid balance</p>
                      <div className="text-5xl font-light tracking-tight mb-3" data-testid="billing-balance">{balanceDisplay}</div>
                      <span className={`inline-flex items-center rounded-full border px-3 py-1 text-xs font-mono ${balanceTone.pill}`}>{balanceTone.label}</span>
                    </div>
                    <div className="rounded-3xl border border-white/20 bg-black/30 px-5 py-4 min-w-[12rem]">
                      <div className="text-xs font-mono uppercase tracking-[0.16em] text-white/50 mb-2">Recommended reserve</div>
                      <div className="text-lg font-semibold">{formatUsdWhole(RECOMMENDED_THRESHOLD_USD)}</div>
                    </div>
                  </div>

                  <p className="text-sm text-white/60 mb-6">
                    Keep a prepaid buffer here and let Stripe refill it when needed.
                  </p>

                  <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4" data-testid="billing-quick-topups">
                    {QUICK_TOPUP_AMOUNTS.map(amount => (
                      <button
                        key={amount}
                        onClick={() => openCheckout(amount)}
                        className={`rounded-2xl border px-4 py-4 text-left transition-colors ${
                          amount === RECOMMENDED_TOPUP_USD
                            ? 'border-emerald-300/25 bg-emerald-500/10'
                            : 'border-white/20 bg-black/25 hover:border-white/50'
                        }`}
                      >
                        <div className="text-2xl font-semibold">{formatUsdWhole(amount)}</div>
                      </button>
                    ))}
                  </div>

                  <button
                    onClick={() => openCheckout(RECOMMENDED_TOPUP_USD)}
                    data-testid="add-funds-stripe-btn"
                    className="w-full md:w-auto bg-white text-black px-5 py-3 text-sm font-mono font-bold hover:bg-white/90 transition-colors rounded-2xl"
                  >
                    Add {formatUsdWhole(RECOMMENDED_TOPUP_USD)} with Stripe
                  </button>
                </div>
              </div>

              <div className="grid gap-6">
                {postTopupPrompt && !autotopupSettings.enabled && (
                  <div className="rounded-[28px] border border-emerald-400/25 bg-[linear-gradient(135deg,rgba(16,185,129,0.16),rgba(255,255,255,0.025))] p-6" data-testid="post-topup-autotopup-prompt">
                    <div className="flex items-start gap-4">
                      <div className="rounded-2xl bg-emerald-400/15 p-3">
                        <Repeat className="w-5 h-5 text-emerald-200" />
                      </div>
                      <div className="min-w-0">
                        <p className="text-xs font-mono uppercase tracking-[0.18em] text-emerald-100/60 mb-2">Next step</p>
                        <h2 className="text-xl font-semibold tracking-tight">Keep credits topped up automatically</h2>
                        <p className="text-sm text-white/65 mt-2">
                          Use the recommended rule: add {formatUsdWhole(RECOMMENDED_TOPUP_USD)} whenever your balance falls below {formatUsdWhole(RECOMMENDED_THRESHOLD_USD)}.
                        </p>
                        <div className="mt-4 flex flex-wrap gap-3">
                          {hasCards ? (
                            <button
                              onClick={enableRecommendedAutotopup}
                              disabled={savingAutotopup}
                              className="rounded-2xl bg-white text-black px-4 py-3 text-sm font-mono font-bold hover:bg-white/90 transition-colors disabled:opacity-50"
                              data-testid="post-topup-enable-autotopup"
                            >
                              {savingAutotopup ? 'Saving...' : 'Enable recommended rule'}
                            </button>
                          ) : (
                            <button
                              onClick={startCardSetup}
                              disabled={!stripePk}
                              className="rounded-2xl bg-white text-black px-4 py-3 text-sm font-mono font-bold hover:bg-white/90 transition-colors disabled:opacity-50"
                              data-testid="post-topup-save-card"
                            >
                              Save card for auto-topup
                            </button>
                          )}
                          <button
                            onClick={() => setPostTopupPrompt(false)}
                            className="rounded-2xl border border-white/15 bg-black/20 px-4 py-3 text-sm font-mono text-white hover:border-white/50 transition-colors"
                            data-testid="post-topup-dismiss"
                          >
                            Not now
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                )}

                <div ref={autotopupCardRef} className="border border-white/20 bg-white/[0.05] rounded-[28px] p-6" data-testid="autotopup-card">
                  <div className="flex items-start justify-between gap-4 mb-5">
                    <div>
                      <div className="flex items-center gap-2 text-xs font-mono uppercase tracking-[0.18em] text-white/50 mb-2">
                        <Repeat className="w-4 h-4" /> Auto-topup
                      </div>
                      <h2 className="text-xl font-semibold tracking-tight">{autotopupSettings.enabled ? 'Enabled' : 'Backup funding rule'}</h2>
                    </div>
                    <button
                      onClick={() =>
                        setAutotopupSettings(prev => ({
                          ...prev,
                          enabled: !prev.enabled,
                        }))
                      }
                      data-testid="autotopup-toggle"
                      className={`relative inline-flex h-8 w-14 rounded-full transition-colors ${
                        autotopupSettings.enabled ? 'bg-emerald-400' : 'bg-white/15'
                      }`}
                      aria-label="Toggle auto-topup"
                    >
                      <span
                        className={`absolute left-1 top-1 h-6 w-6 rounded-full bg-black transition-transform ${
                          autotopupSettings.enabled ? 'translate-x-6' : 'translate-x-0'
                        }`}
                      />
                    </button>
                  </div>

                  <div className="rounded-2xl border border-white/20 bg-black/20 p-4 mb-4">
                    <div className="flex items-center justify-between gap-3 mb-3">
                      <div>
                        <p className="text-xs font-mono uppercase tracking-[0.14em] text-white/50">Recommended default</p>
                        <p className="text-sm text-white/60 mt-1">{recommendedRuleCopy}</p>
                      </div>
                      <button
                        onClick={() =>
                          setAutotopupSettings({
                            enabled: true,
                            threshold_cents: usdToUnits(RECOMMENDED_THRESHOLD_USD),
                            amount_cents: usdToUnits(RECOMMENDED_TOPUP_USD),
                          })
                        }
                        className="shrink-0 rounded-xl border border-white/15 px-3 py-2 text-xs font-mono text-white hover:bg-white/10 transition-colors"
                        data-testid="autotopup-use-recommended"
                      >
                        Use recommended
                      </button>
                    </div>
                  </div>

                  <div className="mb-4">
                    <label className="text-xs font-mono uppercase tracking-[0.14em] text-white/50 block mb-2">Trigger when balance falls below</label>
                    <div className="relative">
                      <span className="absolute left-4 top-1/2 -translate-y-1/2 text-white/50 font-mono">$</span>
                      <input
                        type="number"
                        min="1"
                        max="100"
                        value={unitsToUSD(autotopupSettings.threshold_cents)}
                        onChange={e =>
                          setAutotopupSettings(prev => ({
                            ...prev,
                            threshold_cents: usdToUnits(Number(e.target.value) || 0),
                          }))
                        }
                        data-testid="autotopup-threshold-input"
                        className="w-full bg-white/[0.06] border border-white/30 rounded-2xl py-3 pl-10 pr-4 text-white font-mono focus:outline-none focus:border-white/50"
                      />
                    </div>
                  </div>

                  <div className="mb-5">
                    <label className="text-xs font-mono uppercase tracking-[0.14em] text-white/50 block mb-2">Top-up amount</label>
                    <div className="relative">
                      <span className="absolute left-4 top-1/2 -translate-y-1/2 text-white/50 font-mono">$</span>
                      <input
                        type="number"
                        min="5"
                        max="500"
                        value={unitsToUSD(autotopupSettings.amount_cents)}
                        onChange={e =>
                          setAutotopupSettings(prev => ({
                            ...prev,
                            amount_cents: usdToUnits(Number(e.target.value) || 0),
                          }))
                        }
                        data-testid="autotopup-amount-input"
                        className="w-full bg-white/[0.06] border border-white/30 rounded-2xl py-3 pl-10 pr-4 text-white font-mono focus:outline-none focus:border-white/50"
                      />
                    </div>
                  </div>

                  <div className="rounded-2xl border border-white/20 bg-black/20 p-4 mb-5">
                    <div className="flex items-start gap-3">
                      <ShieldCheck className="w-5 h-5 text-white/45 mt-0.5" />
                      <div>
                        <p className="text-sm text-white/70">
                          {hasCards ? 'A Stripe card is on file.' : 'Save a Stripe card first before enabling auto-topup.'}
                        </p>
                        <p className="text-xs text-white/55 mt-1">This only funds prepaid credits. It does not switch you to postpaid billing.</p>
                      </div>
                    </div>
                  </div>

                  <div className="flex flex-wrap gap-3">
                    {!hasCards && (
                      <button
                        onClick={startCardSetup}
                        disabled={!stripePk}
                        className="rounded-2xl bg-white text-black px-4 py-3 text-sm font-mono font-bold hover:bg-white/90 transition-colors disabled:opacity-50"
                        data-testid="add-card-btn"
                      >
                        Save a card
                      </button>
                    )}
                    <button
                      onClick={saveAutotopup}
                      disabled={savingAutotopup || (autotopupSettings.enabled && !hasCards)}
                      className="rounded-2xl border border-white/15 bg-white/[0.06] px-4 py-3 text-sm font-mono text-white hover:border-white/50 transition-colors disabled:opacity-50"
                      data-testid="autotopup-save-btn"
                    >
                      {savingAutotopup ? 'Saving...' : autotopupSettings.enabled ? 'Save auto-topup rule' : 'Save as disabled'}
                    </button>
                  </div>
                </div>

                <div className="border border-white/20 bg-white/[0.05] rounded-[28px] p-6" data-testid="stripe-card">
                  <div className="flex items-center justify-between gap-4 mb-5">
                    <div>
                      <div className="flex items-center gap-2 text-xs font-mono uppercase tracking-[0.18em] text-white/50 mb-2">
                        <CreditCard className="w-4 h-4" /> Saved cards
                      </div>
                      <h2 className="text-xl font-semibold tracking-tight">{hasCards ? 'Stripe payment method on file' : 'No Stripe card saved'}</h2>
                    </div>
                    {stripePk && (
                      <button
                        onClick={startCardSetup}
                        className="rounded-2xl border border-white/15 bg-white/[0.06] px-4 py-3 text-sm font-mono text-white hover:border-white/50 transition-colors"
                      >
                        {hasCards ? 'Replace card' : 'Add card'}
                      </button>
                    )}
                  </div>

                  {paymentMethods.length > 0 ? (
                    <div className="space-y-3">
                      {paymentMethods.map(pm => (
                        <div key={pm.id} className="rounded-2xl border border-white/20 bg-black/20 p-4 flex items-center justify-between gap-3" data-testid="saved-card">
                          <div>
                            <div className="font-semibold">{maskCard(pm)}</div>
                            <div className="text-xs font-mono text-white/55 mt-1">
                              Expires {String(pm.card?.exp_month || '').padStart(2, '0')}/{pm.card?.exp_year}
                            </div>
                          </div>
                          <button onClick={() => deletePaymentMethod(pm.id)} className="text-xs font-mono text-red-300 hover:text-red-200 transition-colors">
                            Remove
                          </button>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="rounded-2xl border border-dashed border-white/20 bg-black/20 p-5">
                      <p className="text-sm text-white/60">Add a Stripe card if you want prepaid credits to refill automatically.</p>
                    </div>
                  )}
                </div>

                <div className="border border-[#14F195]/30 bg-[#14F195]/5 rounded-[28px] p-6 flex flex-col relative overflow-hidden" data-testid="solana-card">
                  <div className="absolute -right-10 -top-10 w-32 h-32 bg-[#9945FF]/20 blur-3xl rounded-full pointer-events-none" />
                  <div className="mb-6 relative z-10">
                    <CircleDollarSign className="w-8 h-8 text-[#14F195] mb-4" />
                    <h3 className="text-xl font-bold mb-2">Crypto top-up</h3>
                    <p className="text-sm text-white/60 font-light">Still available if you want to pay with SOL or USDC instead of Stripe.</p>
                  </div>
                  <div className="mt-auto relative z-10">
                    <button className="w-full bg-gradient-to-r from-[#9945FF] to-[#14F195] text-black px-4 py-3 text-sm font-mono font-bold hover:opacity-90 transition-opacity rounded-2xl flex items-center justify-center gap-2" data-testid="connect-wallet-btn">
                      <Wallet className="w-4 h-4" /> Connect Wallet
                    </button>
                  </div>
                </div>
              </div>
            </div>

            {transactions.length > 0 && (
              <>
                <h2 className="text-xl font-bold tracking-tight mb-4">Transaction history</h2>
                <div className="border border-white/20 rounded-3xl overflow-hidden">
                  <table className="w-full text-left text-sm" data-testid="payment-history-table">
                    <thead className="bg-white/10 font-mono text-xs text-white/55 border-b border-white/20">
                      <tr>
                        <th className="px-6 py-3 font-normal">Date</th>
                        <th className="px-6 py-3 font-normal">Type</th>
                        <th className="px-6 py-3 font-normal">Description</th>
                        <th className="px-6 py-3 font-normal">Amount</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-white/10 font-mono">
                      {transactions.map((tx: any) => (
                        <tr key={tx.id}>
                          <td className="px-6 py-4 text-white/60">{formatTransactionDate(tx.created_at)}</td>
                          <td className="px-6 py-4">{tx.tx_type}</td>
                          <td className="px-6 py-4 text-white/60">{tx.description}</td>
                          <td className={`px-6 py-4 ${tx.amount_cents > 0 ? 'text-emerald-300' : 'text-red-400'}`}>{formatSignedUnits(tx.amount_cents)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </>
            )}
          </motion.div>
        )}

        {activeTab === 'analytics' && (
          <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
            <div className="flex items-center justify-between mb-8">
              <div>
                <h1 className="text-2xl font-bold tracking-tight">Usage & Spend</h1>
                <p className="text-sm text-white/55 font-mono mt-1">Spend by product, by API key, and how often you use the platform</p>
              </div>
              <div className="flex gap-2 font-mono text-sm">
                {(['24h', '7d', '30d', '90d'] as const).map(p => (
                  <button
                    key={p}
                    onClick={() => setAnalyticsPeriod(p)}
                    className={`px-3 py-1.5 rounded-lg transition-colors ${analyticsPeriod === p ? 'bg-white text-black font-bold' : 'bg-white/10 text-white/60 hover:bg-white/10 hover:text-white'}`}
                  >
                    {p}
                  </button>
                ))}
              </div>
            </div>

            {analyticsLoading ? (
              <div className="flex items-center justify-center h-48 text-white/55 font-mono text-sm">Loading analytics…</div>
            ) : drilldown ? (
              <div>
                <button
                  onClick={() => setDrilldown(null)}
                  className="flex items-center gap-2 text-white/60 hover:text-white font-mono text-sm mb-6 transition-colors"
                >
                  <ChevronLeft className="w-4 h-4" /> Back to overview
                </button>
                <h2 className="text-lg font-bold tracking-tight mb-1">
                  {drilldown.type === 'key' ? 'API Key' : 'Provider'}: <span className="font-mono text-white/70">{drilldown.label}</span>
                </h2>
                <p className="text-sm text-white/55 font-mono mb-6">Model breakdown · {analyticsPeriod}</p>
                {drilldown.models.length === 0 ? (
                  <p className="text-white/55 font-mono text-sm">No usage data for this period.</p>
                ) : (
                  <div className="border border-white/20 rounded-3xl overflow-hidden">
                    <table className="w-full text-left text-sm font-mono">
                      <thead className="bg-white/10 text-xs text-white/55 border-b border-white/20">
                        <tr>
                          <th className="px-6 py-3 font-normal">Model</th>
                          <th className="px-6 py-3 font-normal">Provider</th>
                          <th className="px-6 py-3 font-normal text-right">Requests</th>
                          <th className="px-6 py-3 font-normal text-right">Spend</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-white/10">
                        {drilldown.models.map((m, i) => (
                          <tr key={i} className="hover:bg-white/10">
                            <td className="px-6 py-3 text-white/90">{m.model}</td>
                            <td className="px-6 py-3 text-white/50 capitalize">{m.provider}</td>
                            <td className="px-6 py-3 text-right text-white/70">{m.total_requests.toLocaleString()}</td>
                            <td className="px-6 py-3 text-right text-emerald-300">{formatBalanceUnits(m.total_cost_cents)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
            ) : (
              <div className="space-y-8">
                {/* Activity contribution heatmap */}
                <div className="border border-white/20 rounded-3xl p-6">
                  <h2 className="text-base font-bold tracking-tight mb-1">Activity</h2>
                  <p className="text-xs text-white/55 font-mono mb-5">How often you use the platform · last year</p>
                  {activity.length === 0 ? (
                    <div className="h-32 flex items-center justify-center text-white/45 font-mono text-sm">No activity yet</div>
                  ) : (
                    <ContributionHeatmap data={activity} />
                  )}
                </div>

                {/* Spend by product */}
                <div className="border border-white/20 rounded-3xl p-6">
                  <h2 className="text-base font-bold tracking-tight mb-1">Spend by product</h2>
                  <p className="text-xs text-white/55 font-mono mb-5">Cost across capabilities · {analyticsPeriod}</p>
                  {spendByProduct.length === 0 ? (
                    <div className="h-48 flex items-center justify-center text-white/45 font-mono text-sm">No data for this period</div>
                  ) : (
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6 items-center">
                      <ResponsiveContainer width="100%" height={240}>
                        <PieChart>
                          <Pie
                            data={spendByProduct}
                            dataKey="total_cost_cents"
                            nameKey="product"
                            cx="50%"
                            cy="50%"
                            innerRadius={55}
                            outerRadius={90}
                            paddingAngle={2}
                            stroke="none"
                          >
                            {spendByProduct.map((p, i) => (
                              <Cell key={i} fill={productMeta(p.product).color} />
                            ))}
                          </Pie>
                          <Tooltip
                            contentStyle={{ background: '#111', border: '1px solid rgba(255,255,255,0.12)', borderRadius: 8, fontFamily: 'monospace', fontSize: 12 }}
                            labelStyle={{ color: 'rgba(255,255,255,0.5)' }}
                            formatter={(v: number, _: string, props: any) => [`$${(v / 10000).toFixed(4)} · ${props.payload.total_requests.toLocaleString()} reqs`, productMeta(props.payload.product).label]}
                          />
                          <Legend
                            formatter={(value: string) => <span style={{ color: 'rgba(255,255,255,0.6)', fontFamily: 'monospace', fontSize: 11 }}>{productMeta(value).label}</span>}
                          />
                        </PieChart>
                      </ResponsiveContainer>
                      <div className="border border-white/20 rounded-2xl overflow-hidden">
                        <table className="w-full text-left text-sm font-mono">
                          <thead className="bg-white/10 text-xs text-white/55 border-b border-white/20">
                            <tr>
                              <th className="px-4 py-3 font-normal">Product</th>
                              <th className="px-4 py-3 font-normal text-right">Requests</th>
                              <th className="px-4 py-3 font-normal text-right">Spend</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-white/10">
                            {spendByProduct.map((p, i) => (
                              <tr key={i} className="hover:bg-white/10">
                                <td className="px-4 py-3 text-white/90">
                                  <span className="inline-block w-2.5 h-2.5 rounded-sm mr-2 align-middle" style={{ background: productMeta(p.product).color }} />
                                  {productMeta(p.product).label}
                                </td>
                                <td className="px-4 py-3 text-right text-white/60">{p.total_requests.toLocaleString()}</td>
                                <td className="px-4 py-3 text-right text-emerald-300">{formatBalanceUnits(p.total_cost_cents)}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </div>
                  )}
                </div>

                {/* Spend over time */}
                <div className="border border-white/20 rounded-3xl p-6">
                  <h2 className="text-base font-bold tracking-tight mb-1">Spend over time</h2>
                  <p className="text-xs text-white/55 font-mono mb-5">Cost in USD · {analyticsPeriod}</p>
                  {spendTimeSeries.length === 0 ? (
                    <div className="h-48 flex items-center justify-center text-white/45 font-mono text-sm">No data for this period</div>
                  ) : (
                    <ResponsiveContainer width="100%" height={200}>
                      <LineChart data={spendTimeSeries} margin={{ top: 4, right: 8, bottom: 4, left: 0 }}>
                        <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
                        <XAxis
                          dataKey="timestamp"
                          tickFormatter={(v: string) => {
                            const d = new Date(v);
                            return analyticsPeriod === '24h'
                              ? d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
                              : d.toLocaleDateString([], { month: 'short', day: 'numeric' });
                          }}
                          tick={{ fill: 'rgba(255,255,255,0.35)', fontSize: 11, fontFamily: 'monospace' }}
                          axisLine={false}
                          tickLine={false}
                        />
                        <YAxis
                          tickFormatter={(v: number) => `$${(v / 10000).toFixed(2)}`}
                          tick={{ fill: 'rgba(255,255,255,0.35)', fontSize: 11, fontFamily: 'monospace' }}
                          axisLine={false}
                          tickLine={false}
                          width={60}
                        />
                        <Tooltip
                          contentStyle={{ background: '#111', border: '1px solid rgba(255,255,255,0.12)', borderRadius: 8, fontFamily: 'monospace', fontSize: 12 }}
                          labelStyle={{ color: 'rgba(255,255,255,0.5)' }}
                          itemStyle={{ color: '#6ee7b7' }}
                          formatter={(v: number) => [`$${(v / 10000).toFixed(4)}`, 'Spend']}
                          labelFormatter={(v: string) => new Date(v).toLocaleString()}
                        />
                        <Line type="monotone" dataKey="value" stroke="#6ee7b7" strokeWidth={2} dot={false} />
                      </LineChart>
                    </ResponsiveContainer>
                  )}
                </div>

                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                  {/* Spend by API key */}
                  <div className="border border-white/20 rounded-3xl p-6">
                    <h2 className="text-base font-bold tracking-tight mb-1">Spend by API key</h2>
                    <p className="text-xs text-white/55 font-mono mb-5">Click a bar to drill down · {analyticsPeriod}</p>
                    {spendByKey.length === 0 ? (
                      <div className="h-48 flex items-center justify-center text-white/45 font-mono text-sm">No data for this period</div>
                    ) : (
                      <ResponsiveContainer width="100%" height={220}>
                        <BarChart data={spendByKey} margin={{ top: 4, right: 8, bottom: 30, left: 0 }}
                          onClick={(e: any) => {
                            if (e?.activePayload?.[0]?.payload) {
                              const d = e.activePayload[0].payload;
                              void loadDrilldown('key', d.api_key_id, `${d.key_name} (${d.key_prefix}…)`);
                            }
                          }}
                        >
                          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
                          <XAxis
                            dataKey="key_name"
                            tick={{ fill: 'rgba(255,255,255,0.35)', fontSize: 11, fontFamily: 'monospace' }}
                            axisLine={false}
                            tickLine={false}
                            angle={-20}
                            textAnchor="end"
                          />
                          <YAxis
                            tickFormatter={(v: number) => `$${(v / 10000).toFixed(2)}`}
                            tick={{ fill: 'rgba(255,255,255,0.35)', fontSize: 11, fontFamily: 'monospace' }}
                            axisLine={false}
                            tickLine={false}
                            width={60}
                          />
                          <Tooltip
                            contentStyle={{ background: '#111', border: '1px solid rgba(255,255,255,0.12)', borderRadius: 8, fontFamily: 'monospace', fontSize: 12 }}
                            labelStyle={{ color: 'rgba(255,255,255,0.5)' }}
                            itemStyle={{ color: '#6ee7b7' }}
                            formatter={(v: number, _: string, props: any) => [`$${(v / 10000).toFixed(4)} · ${props.payload.total_requests.toLocaleString()} reqs`, 'Spend']}
                          />
                          <Bar dataKey="total_cost_cents" radius={[4, 4, 0, 0]} cursor="pointer">
                            {spendByKey.map((_, i) => (
                              <Cell key={i} fill={`hsl(${160 + i * 28}, 70%, 55%)`} />
                            ))}
                          </Bar>
                        </BarChart>
                      </ResponsiveContainer>
                    )}
                    {spendByKey.length > 0 && (
                      <p className="text-xs text-white/45 font-mono mt-3 text-center">Click a bar to see model breakdown</p>
                    )}
                  </div>

                  {/* Spend by provider */}
                  <div className="border border-white/20 rounded-3xl p-6">
                    <h2 className="text-base font-bold tracking-tight mb-1">Spend by provider</h2>
                    <p className="text-xs text-white/55 font-mono mb-5">Click a bar to drill down · {analyticsPeriod}</p>
                    {spendByProvider.length === 0 ? (
                      <div className="h-48 flex items-center justify-center text-white/45 font-mono text-sm">No data for this period</div>
                    ) : (
                      <ResponsiveContainer width="100%" height={220}>
                        <BarChart data={spendByProvider} margin={{ top: 4, right: 8, bottom: 30, left: 0 }}
                          onClick={(e: any) => {
                            if (e?.activePayload?.[0]?.payload) {
                              const d = e.activePayload[0].payload;
                              void loadDrilldown('provider', d.provider, d.provider);
                            }
                          }}
                        >
                          <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.06)" />
                          <XAxis
                            dataKey="provider"
                            tick={{ fill: 'rgba(255,255,255,0.35)', fontSize: 11, fontFamily: 'monospace' }}
                            axisLine={false}
                            tickLine={false}
                            angle={-20}
                            textAnchor="end"
                          />
                          <YAxis
                            tickFormatter={(v: number) => `$${(v / 10000).toFixed(2)}`}
                            tick={{ fill: 'rgba(255,255,255,0.35)', fontSize: 11, fontFamily: 'monospace' }}
                            axisLine={false}
                            tickLine={false}
                            width={60}
                          />
                          <Tooltip
                            contentStyle={{ background: '#111', border: '1px solid rgba(255,255,255,0.12)', borderRadius: 8, fontFamily: 'monospace', fontSize: 12 }}
                            labelStyle={{ color: 'rgba(255,255,255,0.5)' }}
                            itemStyle={{ color: '#a78bfa' }}
                            formatter={(v: number, _: string, props: any) => [`$${(v / 10000).toFixed(4)} · ${props.payload.total_requests.toLocaleString()} reqs`, 'Spend']}
                          />
                          <Bar dataKey="total_cost_cents" radius={[4, 4, 0, 0]} cursor="pointer">
                            {spendByProvider.map((_, i) => (
                              <Cell key={i} fill={`hsl(${260 + i * 30}, 70%, 60%)`} />
                            ))}
                          </Bar>
                        </BarChart>
                      </ResponsiveContainer>
                    )}
                    {spendByProvider.length > 0 && (
                      <p className="text-xs text-white/45 font-mono mt-3 text-center">Click a bar to see model breakdown</p>
                    )}
                  </div>
                </div>

                {/* Summary table */}
                {(spendByKey.length > 0 || spendByProvider.length > 0) && (
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <div className="border border-white/20 rounded-3xl overflow-hidden">
                      <table className="w-full text-left text-sm font-mono">
                        <thead className="bg-white/10 text-xs text-white/55 border-b border-white/20">
                          <tr>
                            <th className="px-6 py-3 font-normal">API Key</th>
                            <th className="px-6 py-3 font-normal text-right">Requests</th>
                            <th className="px-6 py-3 font-normal text-right">Spend</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-white/10">
                          {spendByKey.map((k, i) => (
                            <tr
                              key={i}
                              className="hover:bg-white/10 cursor-pointer"
                              onClick={() => void loadDrilldown('key', k.api_key_id, `${k.key_name} (${k.key_prefix}…)`)}
                            >
                              <td className="px-6 py-3 text-white/90">
                                <span>{k.key_name}</span>
                                <span className="text-white/45 ml-2">{k.key_prefix}…</span>
                              </td>
                              <td className="px-6 py-3 text-right text-white/60">{k.total_requests.toLocaleString()}</td>
                              <td className="px-6 py-3 text-right text-emerald-300">{formatBalanceUnits(k.total_cost_cents)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                    <div className="border border-white/20 rounded-3xl overflow-hidden">
                      <table className="w-full text-left text-sm font-mono">
                        <thead className="bg-white/10 text-xs text-white/55 border-b border-white/20">
                          <tr>
                            <th className="px-6 py-3 font-normal">Provider</th>
                            <th className="px-6 py-3 font-normal text-right">Requests</th>
                            <th className="px-6 py-3 font-normal text-right">Spend</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-white/10">
                          {spendByProvider.map((p, i) => (
                            <tr
                              key={i}
                              className="hover:bg-white/10 cursor-pointer capitalize"
                              onClick={() => void loadDrilldown('provider', p.provider, p.provider)}
                            >
                              <td className="px-6 py-3 text-white/90">{p.provider}</td>
                              <td className="px-6 py-3 text-right text-white/60">{p.total_requests.toLocaleString()}</td>
                              <td className="px-6 py-3 text-right text-violet-300">{formatBalanceUnits(p.total_cost_cents)}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                )}
              </div>
            )}
          </motion.div>
        )}
      </main>

      <TopUpModal
        open={stripeModalOpen}
        onClose={() => {
          setStripeModalOpen(false);
          void refreshBilling();
        }}
        stripePk={stripePk}
        initialAmount={checkoutAmount}
      />
      <PaymentMethodSetupModal
        open={cardModalOpen}
        stripePk={stripePk}
        clientSecret={cardSetupSecret}
        loading={cardSetupLoading}
        error={cardSetupError}
        onClose={() => {
          setCardModalOpen(false);
          setCardSetupSecret(null);
          setCardSetupError('');
        }}
        onSaved={handleCardSaved}
      />
    </div>
  );
}
