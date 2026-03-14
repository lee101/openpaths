import React, { useState, useEffect, useCallback } from 'react';
import { CreditCard, Key, Wallet, Plus, Copy, Check, Activity, X, TrendingUp, LogOut, KeyRound, Eye, EyeOff, Save } from 'lucide-react';
import { motion, AnimatePresence } from 'motion/react';
import { loadStripe } from '@stripe/stripe-js';
import { EmbeddedCheckoutProvider, EmbeddedCheckout } from '@stripe/react-stripe-js';
import { DEFAULT_API_KEY_NAME, TOKEN_STORAGE_KEY, clearStoredSession, getStoredAPIKey, getStoredToken, getStoredUser, persistAuth, storeAPIKey } from '../lib/session';

const API_BASE = '';

let stripePromise: ReturnType<typeof loadStripe> | null = null;
function getStripe(pk: string) {
  if (!stripePromise && pk) stripePromise = loadStripe(pk);
  return stripePromise;
}

function api(path: string, opts: RequestInit = {}) {
  const token = localStorage.getItem(TOKEN_STORAGE_KEY);
  return fetch(API_BASE + path, {
    ...opts,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...opts.headers,
    },
  });
}

// --- Auth forms ---

function AuthForms({ onAuth }: { onAuth: (token: string, user: any) => void }) {
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
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
      if (!res.ok) { setError(data.error?.message || 'Failed'); return; }
      persistAuth(data.token, data.user);
      onAuth(data.token, data.user);
    } catch { setError('Network error'); } finally { setLoading(false); }
  };

  return (
    <div className="max-w-md mx-auto mt-20 px-6">
      <h1 className="text-3xl font-bold tracking-tight mb-8">{mode === 'login' ? 'Sign In' : 'Create Account'}</h1>
      <form onSubmit={submit} className="space-y-4">
        {mode === 'register' && (
          <input value={name} onChange={e => setName(e.target.value)} placeholder="Name" className="w-full bg-black border border-white/10 rounded-lg py-3 px-4 text-white font-mono focus:outline-none focus:border-white/30" />
        )}
        <input type="email" value={email} onChange={e => setEmail(e.target.value)} placeholder="Email" required className="w-full bg-black border border-white/10 rounded-lg py-3 px-4 text-white font-mono focus:outline-none focus:border-white/30" data-testid="auth-email" />
        <input type="password" value={password} onChange={e => setPassword(e.target.value)} placeholder="Password" required minLength={8} className="w-full bg-black border border-white/10 rounded-lg py-3 px-4 text-white font-mono focus:outline-none focus:border-white/30" data-testid="auth-password" />
        {error && <p className="text-red-400 text-sm font-mono">{error}</p>}
        <button type="submit" disabled={loading} className="w-full bg-white text-black py-3 rounded-lg font-mono font-bold text-sm hover:bg-white/90 transition-colors disabled:opacity-50" data-testid="auth-submit">
          {loading ? 'Loading...' : mode === 'login' ? 'Sign In' : 'Create Account'}
        </button>
      </form>
      <button onClick={() => { setMode(mode === 'login' ? 'register' : 'login'); setError(''); }} className="mt-4 text-sm font-mono text-white/40 hover:text-white transition-colors" data-testid="auth-toggle">
        {mode === 'login' ? 'Need an account? Register' : 'Already have an account? Sign In'}
      </button>
    </div>
  );
}

// --- Usage graph (unchanged) ---

function UsageGraph({ data }: { data: { date: string; requests: number; cost: number }[] }) {
  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null);
  if (!data.length) return null;
  const maxReq = Math.max(...data.map(d => d.requests));
  const w = 600, h = 200, px = 40, py = 20;
  const chartW = w - px * 2, chartH = h - py * 2;
  const points = data.map((d, i) => ({
    x: px + (i / Math.max(data.length - 1, 1)) * chartW,
    y: py + chartH - (d.requests / (maxReq || 1)) * chartH,
  }));
  const linePath = points.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x},${p.y}`).join(' ');
  const areaPath = `${linePath} L${points[points.length - 1].x},${h - py} L${points[0].x},${h - py} Z`;

  return (
    <div className="border border-white/10 bg-white/[0.02] rounded-xl p-6" data-testid="usage-graph">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-bold tracking-tight flex items-center gap-2">
          <TrendingUp className="w-5 h-5 text-white/40" /> Usage Over Time
        </h3>
      </div>
      <svg viewBox={`0 0 ${w} ${h}`} className="w-full" role="img" aria-label="Usage over time chart">
        {[0, 0.25, 0.5, 0.75, 1].map(frac => {
          const y = py + chartH - frac * chartH;
          return (
            <g key={frac}>
              <line x1={px} y1={y} x2={w - px} y2={y} stroke="rgba(255,255,255,0.06)" />
              <text x={px - 6} y={y + 4} textAnchor="end" fill="rgba(255,255,255,0.25)" fontSize="9" fontFamily="monospace">
                {Math.round(maxReq * frac / 1000)}k
              </text>
            </g>
          );
        })}
        <defs>
          <linearGradient id="areaGrad" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="white" stopOpacity="0.1" />
            <stop offset="100%" stopColor="white" stopOpacity="0" />
          </linearGradient>
        </defs>
        <path d={areaPath} fill="url(#areaGrad)" />
        <path d={linePath} fill="none" stroke="white" strokeWidth="2" strokeLinejoin="round" />
        {points.map((p, i) => (
          <g key={i} onMouseEnter={() => setHoveredIdx(i)} onMouseLeave={() => setHoveredIdx(null)}>
            <circle cx={p.x} cy={p.y} r={hoveredIdx === i ? 5 : 3} fill="white" className="transition-all" />
            {hoveredIdx === i && (
              <g>
                <rect x={p.x - 50} y={p.y - 38} width="100" height="28" rx="4" fill="black" stroke="rgba(255,255,255,0.2)" />
                <text x={p.x} y={p.y - 20} textAnchor="middle" fill="white" fontSize="10" fontFamily="monospace">
                  {data[i].requests.toLocaleString()} req
                </text>
              </g>
            )}
          </g>
        ))}
        {data.map((d, i) => (
          <text key={i} x={points[i].x} y={h - 4} textAnchor="middle" fill="rgba(255,255,255,0.25)" fontSize="8" fontFamily="monospace">
            {d.date}
          </text>
        ))}
      </svg>
    </div>
  );
}

// --- Stripe Checkout Modal (embedded) ---

const STRIPE_AMOUNTS = [10, 25, 50, 100];

function StripeCheckoutModal({ open, onClose, stripePk }: { open: boolean; onClose: () => void; stripePk: string }) {
  const [amount, setAmount] = useState(25);
  const [clientSecret, setClientSecret] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const startCheckout = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await api('/account/stripe/checkout', {
        method: 'POST',
        body: JSON.stringify({ amount_usd: amount }),
      });
      const data = await res.json();
      if (!res.ok) { setError(data.error?.message || 'Failed to create checkout'); return; }
      setClientSecret(data.client_secret);
    } catch { setError('Network error'); } finally { setLoading(false); }
  };

  const handleClose = () => {
    setClientSecret(null);
    setError('');
    onClose();
  };

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          initial={{ opacity: 0 }} animate={{ opacity: 1 }} exit={{ opacity: 0 }}
          className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4"
          data-testid="stripe-modal-backdrop"
          onClick={e => { if (e.target === e.currentTarget) handleClose(); }}
        >
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: 10 }} animate={{ opacity: 1, scale: 1, y: 0 }} exit={{ opacity: 0, scale: 0.95, y: 10 }}
            className="bg-[#0a0a0a] border border-white/10 rounded-xl w-full max-w-lg p-6 max-h-[90vh] overflow-y-auto"
            data-testid="stripe-modal"
          >
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-bold tracking-tight">{clientSecret ? 'Complete Payment' : 'Add Funds'}</h2>
              <button onClick={handleClose} className="text-white/40 hover:text-white transition-colors" data-testid="stripe-modal-close">
                <X className="w-5 h-5" />
              </button>
            </div>

            {!clientSecret ? (
              <>
                <p className="text-sm text-white/60 mb-6">Select an amount to add to your OpenPaths balance.</p>
                <div className="grid grid-cols-2 gap-3 mb-6">
                  {STRIPE_AMOUNTS.map(a => (
                    <button key={a} onClick={() => setAmount(a)} data-testid={`amount-${a}`}
                      className={`py-3 rounded-lg border text-sm font-mono font-bold transition-colors ${amount === a ? 'bg-white text-black border-white' : 'bg-white/5 text-white/60 border-white/10 hover:border-white/30'}`}
                    >${a}</button>
                  ))}
                </div>
                <div className="mb-6">
                  <label className="text-xs font-mono text-white/40 block mb-2">Custom amount</label>
                  <div className="relative">
                    <span className="absolute left-3 top-1/2 -translate-y-1/2 text-white/40 font-mono">$</span>
                    <input type="number" min="1" max="500" value={amount} onChange={e => setAmount(Number(e.target.value))}
                      data-testid="custom-amount-input"
                      className="w-full bg-black border border-white/10 rounded-lg py-3 pl-8 pr-4 text-white font-mono focus:outline-none focus:border-white/30" />
                  </div>
                </div>
                {error && <p className="text-red-400 text-sm font-mono mb-4">{error}</p>}
                <button onClick={startCheckout} disabled={loading || amount < 1}
                  data-testid="stripe-checkout-btn"
                  className="w-full bg-white text-black py-3 rounded-lg font-mono font-bold text-sm hover:bg-white/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2">
                  {loading ? <span className="animate-spin w-4 h-4 border-2 border-black/20 border-t-black rounded-full" /> : <><CreditCard className="w-4 h-4" /> Pay ${amount} with Stripe</>}
                </button>
              </>
            ) : (
              <div data-testid="embedded-checkout-container">
                <EmbeddedCheckoutProvider stripe={getStripe(stripePk)!} options={{ clientSecret }}>
                  <EmbeddedCheckout />
                </EmbeddedCheckoutProvider>
              </div>
            )}
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

// --- BYOK Panel ---

const BYOK_PROVIDERS = [
  { key: 'openai', label: 'OpenAI API Key', placeholder: 'sk-...', hint: 'Provider: OpenAI (GPT-5, GPT-5.3 Codex)' },
  { key: 'anthropic', label: 'Anthropic API Key', placeholder: 'sk-ant-...', hint: 'Provider: Anthropic (Claude Opus, Sonnet)' },
  { key: 'google', label: 'Google Gemini API Key', placeholder: 'AIza...', hint: 'Provider: Google (Gemini 3.1 Pro)' },
  { key: 'deepseek', label: 'DeepSeek API Key', placeholder: 'sk-...', hint: 'Provider: DeepSeek (V3.2, Reasoner)' },
  { key: 'openrouter', label: 'OpenRouter API Key', placeholder: 'sk-or-...', hint: 'Provider: OpenRouter (Multi-provider routing)' },
  { key: 'xai', label: 'xAI API Key', placeholder: 'xai-...', hint: 'Provider: xAI (Grok 4)' },
  { key: 'mistral', label: 'Mistral API Key', placeholder: 'sk-...', hint: 'Provider: Mistral (Large, Codestral)' },
  { key: 'together', label: 'Together AI API Key', placeholder: 'sk-...', hint: 'Provider: Together (Qwen, GLM, FLUX)' },
  { key: 'netwrck', label: 'Netwrck API Key', placeholder: 'sk-...', hint: 'Provider: Netwrck (RA1 Art Generator)' },
  { key: 'fal', label: 'FAL API Key', placeholder: 'fal_...', hint: 'Provider: FAL (FLUX Klein 4B)' },
];

const BYOK_JSON_FIELDS = [
  { key: 'openai_codex', label: 'OpenAI Codex Max', placeholder: '{"token": "...", "expiresAt": "..."}', hint: '~/.codex/auth.json - OpenAI Codex Max authentication. Used for GPT-5.3 Codex and Spark models. Token refreshes automatically every 3 hours.' },
  { key: 'claude_code', label: 'Claude Code Pro/Max', placeholder: '{"claudeAiOauth": {"accessToken": "...", "refreshToken": "...", "expiresAt": "..."}}', hint: '~/.claude/.credentials.json from `claude setup-token`. Used instead of Anthropic API key.' },
];

const BYOKKeyInput: React.FC<{
  label: string; placeholder: string; hint: string; value: string; onChange: (v: string) => void;
}> = ({ label, placeholder, hint, value, onChange }) => {
  const [show, setShow] = useState(false);
  return (
    <div className="space-y-0.5">
      <label className="text-xs font-medium text-white/60">{label}</label>
      <div className="relative">
        <input
          type={show ? 'text' : 'password'}
          placeholder={placeholder}
          value={value}
          onChange={e => onChange(e.target.value)}
          className="w-full bg-black border border-white/10 rounded-lg py-2 pl-3 pr-10 text-sm text-white font-mono focus:outline-none focus:border-white/30 transition-colors"
        />
        <button type="button" onClick={() => setShow(!show)} className="absolute right-3 top-2 text-white/30 hover:text-white/60">
          {show ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
        </button>
      </div>
      <p className="text-[10px] text-white/30 leading-tight">{hint}</p>
    </div>
  );
}

function BYOKPanel() {
  const [keys, setKeys] = useState<Record<string, string>>({});
  const [jsonFields, setJsonFields] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    api('/account/provider-keys').then(r => r.json()).then(data => {
      if (!data.keys) return;
      const loaded: Record<string, string> = {};
      for (const k of data.keys) {
        if (k.has_key) loaded[k.provider] = k.key_preview || '********';
        if (k.has_auth) {
          if (!loaded[k.provider]) loaded[k.provider] = '';
          setJsonFields(prev => ({ ...prev, [k.provider]: '(saved)' }));
        }
      }
    }).catch(() => {});
  }, []);

  const handleSave = async () => {
    setSaving(true);
    setError('');
    setSaved(false);

    const bulkKeys: { provider: string; api_key: string; auth_json: string }[] = [];
    for (const p of BYOK_PROVIDERS) {
      const val = keys[p.key];
      if (val && !val.includes('*')) {
        bulkKeys.push({ provider: p.key, api_key: val, auth_json: '' });
      }
    }
    for (const f of BYOK_JSON_FIELDS) {
      const val = jsonFields[f.key];
      if (val && val !== '(saved)') {
        bulkKeys.push({ provider: f.key, api_key: '', auth_json: val });
      }
    }

    if (bulkKeys.length === 0) {
      setSaving(false);
      return;
    }

    try {
      const res = await api('/account/provider-keys/bulk', {
        method: 'POST',
        body: JSON.stringify({ keys: bulkKeys }),
      });
      const data = await res.json();
      if (!res.ok) { setError(data.error?.message || 'Failed to save'); return; }
      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
    } catch { setError('Network error'); } finally { setSaving(false); }
  };

  return (
    <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
      <div className="mb-8">
        <h1 className="text-3xl font-bold tracking-tight mb-2">Bring Your Own Keys</h1>
        <p className="text-sm text-white/40">Use your own API keys to bypass credit usage. Requests using your own keys are free on the platform.</p>
      </div>

      <div className="border border-white/10 bg-white/[0.02] rounded-xl overflow-hidden">
        <div className="p-6 space-y-4">
          {BYOK_PROVIDERS.map((p, i) => (
            <BYOKKeyInput
              key={i}
              label={p.label}
              placeholder={p.placeholder}
              hint={p.hint}
              value={keys[p.key] || ''}
              onChange={(v: string) => setKeys(prev => ({ ...prev, [p.key]: v }))}
            />
          ))}

          <div className="border-t border-white/10 pt-4 mt-4">
            <h3 className="text-sm font-bold mb-3">OAuth / Auth JSON</h3>
            {BYOK_JSON_FIELDS.map(f => (
              <div key={f.key} className="mb-4">
                <label className="text-xs font-medium text-white/60 block mb-1">{f.label}</label>
                <textarea
                  placeholder={f.placeholder}
                  value={jsonFields[f.key] || ''}
                  onChange={e => setJsonFields(prev => ({ ...prev, [f.key]: e.target.value }))}
                  rows={2}
                  className="w-full bg-black border border-white/10 rounded-lg py-2 px-3 text-sm text-white font-mono focus:outline-none focus:border-white/30 transition-colors resize-none"
                />
                <p className="text-[10px] text-white/30 leading-tight mt-0.5">{f.hint}</p>
              </div>
            ))}
          </div>
        </div>

        <div className="px-6 py-4 bg-white/[0.02] border-t border-white/10 flex items-center justify-between">
          {error && <p className="text-red-400 text-sm font-mono">{error}</p>}
          {saved && <p className="text-green-400 text-sm font-mono">Keys saved</p>}
          {!error && !saved && <span />}
          <button onClick={handleSave} disabled={saving}
            className="bg-white text-black px-4 py-2 rounded-lg text-sm font-mono font-bold flex items-center gap-2 hover:bg-white/90 transition-colors disabled:opacity-50">
            {saving ? <span className="animate-spin w-4 h-4 border-2 border-black/20 border-t-black rounded-full" /> : <><Save className="w-4 h-4" /> Save Keys</>}
          </button>
        </div>
      </div>

      <div className="mt-6 border border-green-500/20 bg-green-500/5 rounded-xl p-4">
        <h3 className="text-sm font-bold text-green-400 mb-1">How it works</h3>
        <ul className="text-xs text-white/50 space-y-1">
          <li>When you set your own API key for a provider, all requests routed to that provider use YOUR key at no cost.</li>
          <li>OpenAI Codex Max auth tokens are refreshed automatically every 3 hours to keep them active.</li>
          <li>GPT-5.3 Codex and Codex Spark models work with your OpenAI API key or Codex Max auth.</li>
          <li>Your keys are stored encrypted and never shared.</li>
        </ul>
      </div>
    </motion.div>
  );
}

// --- Main Account component ---

export function Account() {
  const [activeTab, setActiveTab] = useState<'overview' | 'keys' | 'billing' | 'byok'>('overview');
  const [user, setUser] = useState<any>(null);
  const [token, setToken] = useState<string | null>(null);
  const [stripePk, setStripePk] = useState('');
  const [deviceApiKey, setDeviceApiKey] = useState(() => getStoredAPIKey());

  // Data states
  const [balance, setBalance] = useState<number | null>(null);
  const [transactions, setTransactions] = useState<any[]>([]);
  const [apiKeys, setApiKeys] = useState<any[]>([]);
  const [copied, setCopied] = useState<string | null>(null);
  const [stripeModalOpen, setStripeModalOpen] = useState(false);
  const [newKeyName, setNewKeyName] = useState('');
  const [newKeyResult, setNewKeyResult] = useState<string | null>(null);

  // Check payment return
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (params.get('payment') === 'success') {
      setActiveTab('billing');
      window.history.replaceState({}, '', '/account');
    }
  }, []);

  // Init auth from localStorage
  useEffect(() => {
    const t = getStoredToken();
    const u = getStoredUser();
    if (t && u) {
      setToken(t);
      setUser(u);
    }
  }, []);

  // Fetch stripe config
  useEffect(() => {
    fetch(API_BASE + '/account/stripe/config').then(r => r.json()).then(d => {
      if (d.publishable_key) setStripePk(d.publishable_key);
    }).catch(() => {});
  }, []);

  const fetchBalance = useCallback(() => {
    if (!token) return;
    api('/account/balance').then(r => r.json()).then(d => {
      if (d.balance_usd !== undefined) setBalance(d.balance_usd);
    }).catch(() => {});
  }, [token]);

  const fetchTransactions = useCallback(() => {
    if (!token) return;
    api('/account/transactions?limit=20').then(r => r.json()).then(d => {
      if (d.transactions) setTransactions(d.transactions);
    }).catch(() => {});
  }, [token]);

  const fetchKeys = useCallback(() => {
    if (!token) return;
    api('/account/keys').then(r => r.json()).then(d => {
      if (d.keys) setApiKeys(d.keys);
    }).catch(() => {});
  }, [token]);

  useEffect(() => {
    if (token) { fetchBalance(); fetchTransactions(); fetchKeys(); }
  }, [token, fetchBalance, fetchTransactions, fetchKeys]);

  useEffect(() => {
    if (!token) return;

    const existing = getStoredAPIKey();
    if (existing) {
      setDeviceApiKey(existing);
      return;
    }

    let cancelled = false;

    api('/account/keys', {
      method: 'POST',
      body: JSON.stringify({ name: DEFAULT_API_KEY_NAME }),
    })
      .then(async res => ({ ok: res.ok, data: await res.json() }))
      .then(({ ok, data }) => {
        if (cancelled || !ok || !data.key) return;
        storeAPIKey(data.key);
        setDeviceApiKey(data.key);
        setNewKeyResult(prev => prev || data.key);
        fetchKeys();
      })
      .catch(() => {});

    return () => {
      cancelled = true;
    };
  }, [token, fetchKeys]);

  const handleAuth = (t: string, u: any) => { setToken(t); setUser(u); };

  const logout = () => {
    clearStoredSession();
    setToken(null);
    setUser(null);
    setDeviceApiKey('');
  };

  const copyKey = (key: string) => {
    navigator.clipboard.writeText(key);
    setCopied(key);
    setTimeout(() => setCopied(null), 2000);
  };

  const createKey = async () => {
    const res = await api('/account/keys', { method: 'POST', body: JSON.stringify({ name: newKeyName || DEFAULT_API_KEY_NAME }) });
    const data = await res.json();
    if (res.ok) {
      setNewKeyResult(data.key);
      storeAPIKey(data.key);
      setDeviceApiKey(data.key);
      setNewKeyName('');
      fetchKeys();
    }
  };

  const revokeKey = async (id: string) => {
    await api(`/account/keys/${id}`, { method: 'DELETE' });
    fetchKeys();
  };

  if (!token) return <AuthForms onAuth={handleAuth} />;

  const balanceDisplay = balance !== null ? `$${balance.toFixed(2)}` : '--';

  return (
    <div className="max-w-6xl mx-auto px-6 py-12 flex flex-col md:flex-row gap-12">
      <aside className="w-full md:w-64 shrink-0">
        <div className="mb-8">
          <h2 className="text-xl font-bold tracking-tight mb-1">Account</h2>
          <p className="text-sm font-mono text-white/40">{user?.email}</p>
          <button onClick={logout} className="mt-2 text-xs font-mono text-white/30 hover:text-white flex items-center gap-1" data-testid="logout-btn">
            <LogOut className="w-3 h-3" /> Sign Out
          </button>
        </div>
        <nav className="flex flex-col gap-2 font-mono text-sm">
          <button onClick={() => setActiveTab('overview')} data-testid="tab-overview"
            className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors text-left ${activeTab === 'overview' ? 'bg-white/10 text-white' : 'text-white/60 hover:bg-white/5 hover:text-white'}`}>
            <Activity className="w-4 h-4" /> Overview
          </button>
          <button onClick={() => setActiveTab('keys')} data-testid="tab-keys"
            className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors text-left ${activeTab === 'keys' ? 'bg-white/10 text-white' : 'text-white/60 hover:bg-white/5 hover:text-white'}`}>
            <Key className="w-4 h-4" /> API Keys
          </button>
          <button onClick={() => setActiveTab('byok')} data-testid="tab-byok"
            className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors text-left ${activeTab === 'byok' ? 'bg-white/10 text-white' : 'text-white/60 hover:bg-white/5 hover:text-white'}`}>
            <KeyRound className="w-4 h-4" /> Bring Your Own Keys
          </button>
          <button onClick={() => setActiveTab('billing')} data-testid="tab-billing"
            className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors text-left ${activeTab === 'billing' ? 'bg-white/10 text-white' : 'text-white/60 hover:bg-white/5 hover:text-white'}`}>
            <CreditCard className="w-4 h-4" /> Billing
          </button>
        </nav>
      </aside>

      <main className="flex-1 min-w-0">
        {activeTab === 'overview' && (
          <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
            <h1 className="text-3xl font-bold tracking-tight mb-8">Overview</h1>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-6 mb-12">
              <div className="border border-white/10 bg-white/[0.02] rounded-xl p-6">
                <div className="text-sm font-mono text-white/40 mb-2">Current Balance</div>
                <div className="text-4xl font-light tracking-tight mb-4" data-testid="balance">{balanceDisplay}</div>
                <button onClick={() => setActiveTab('billing')} className="text-xs font-mono text-white border border-white/20 px-3 py-1.5 rounded hover:bg-white/10 transition-colors">Add Funds</button>
              </div>
              <div className="border border-white/10 bg-white/[0.02] rounded-xl p-6">
                <div className="text-sm font-mono text-white/40 mb-2">API Keys</div>
                <div className="text-4xl font-light tracking-tight mb-4" data-testid="keys-count">{apiKeys.length}</div>
                <button onClick={() => setActiveTab('keys')} className="text-xs font-mono text-white border border-white/20 px-3 py-1.5 rounded hover:bg-white/10 transition-colors">Manage</button>
              </div>
            </div>

            <div className="border border-white/10 bg-white/[0.02] rounded-xl p-6 mb-12" data-testid="api-ready-card">
              <div className="text-sm font-mono text-white/40 mb-2">API Ready On This Device</div>
              <div className="font-mono text-sm text-white/80 break-all mb-3" data-testid="active-api-key">
                {deviceApiKey || 'Generating your API key...'}
              </div>
              <a href="/docs" className="text-xs font-mono text-white border border-white/20 px-3 py-1.5 rounded hover:bg-white/10 transition-colors inline-block">Open API Docs</a>
            </div>

            {transactions.length > 0 && (
              <>
                <h2 className="text-xl font-bold tracking-tight mb-4">Recent Transactions</h2>
                <div className="border border-white/10 rounded-xl overflow-hidden">
                  <table className="w-full text-left text-sm" data-testid="activity-table">
                    <thead className="bg-white/5 font-mono text-xs text-white/40 border-b border-white/10">
                      <tr>
                        <th className="px-6 py-3 font-normal">Type</th>
                        <th className="px-6 py-3 font-normal">Description</th>
                        <th className="px-6 py-3 font-normal">Amount</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-white/10 font-mono">
                      {transactions.slice(0, 10).map((tx: any) => (
                        <tr key={tx.id}>
                          <td className="px-6 py-4 text-white/60">{tx.tx_type}</td>
                          <td className="px-6 py-4">{tx.description}</td>
                          <td className={`px-6 py-4 ${tx.amount_cents > 0 ? 'text-green-400' : 'text-red-400'}`}>
                            {tx.amount_cents > 0 ? '+' : ''}{(tx.amount_cents / 10000).toFixed(4)}
                          </td>
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
            </div>

            {newKeyResult && (
              <div className="border border-green-500/30 bg-green-500/5 rounded-xl p-4 mb-6" data-testid="new-key-banner">
                <p className="text-sm text-green-400 mb-2 font-bold">New key created:</p>
                <div className="flex items-center gap-2 bg-black border border-white/10 rounded-lg p-3">
                  <code className="flex-1 font-mono text-sm text-white/80 break-all">{newKeyResult}</code>
                  <button onClick={() => copyKey(newKeyResult)} className="p-2 hover:bg-white/10 rounded transition-colors">
                    {copied === newKeyResult ? <Check className="w-4 h-4 text-green-400" /> : <Copy className="w-4 h-4 text-white/60" />}
                  </button>
                </div>
              </div>
            )}

            <div className="flex gap-3 mb-6">
              <input value={newKeyName} onChange={e => setNewKeyName(e.target.value)} placeholder="Key name (optional)"
                className="flex-1 bg-black border border-white/10 rounded-lg py-2 px-4 text-white font-mono text-sm focus:outline-none focus:border-white/30" />
              <button onClick={createKey} className="bg-white text-black px-4 py-2 text-sm font-mono font-bold hover:bg-white/90 transition-colors flex items-center gap-2 rounded" data-testid="create-key-btn">
                <Plus className="w-4 h-4" /> Create Key
              </button>
            </div>

            {apiKeys.length === 0 ? (
              <p className="text-white/40 font-mono text-sm">No API keys yet. Create one to get started.</p>
            ) : (
              <div className="space-y-3">
                {apiKeys.map((k: any) => (
                  <div key={k.id} className="border border-white/10 bg-white/[0.02] rounded-xl p-4 flex items-center justify-between" data-testid="api-key-card">
                    <div>
                      <div className="font-bold text-sm">{k.name}</div>
                      <code className="font-mono text-xs text-white/40">{k.key_prefix}...</code>
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="px-2 py-1 bg-green-500/10 text-green-400 text-[10px] font-mono rounded border border-green-500/20" data-testid="key-status">Active</span>
                      <button onClick={() => revokeKey(k.id)} className="text-xs font-mono text-red-400/60 hover:text-red-400 px-2 py-1">Revoke</button>
                    </div>
                  </div>
                ))}
              </div>
            )}

            <p className="text-sm text-white/40 font-light mt-6">Do not share your API key in publicly accessible areas such as GitHub, client-side code, and so forth.</p>
          </motion.div>
        )}

        {activeTab === 'byok' && (
          <BYOKPanel />
        )}

        {activeTab === 'billing' && (
          <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
            <div className="flex items-center justify-between mb-8">
              <h1 className="text-3xl font-bold tracking-tight">Billing & Payments</h1>
            </div>

            <div className="border border-white/10 bg-white/[0.02] rounded-xl p-6 mb-8">
              <div className="text-sm font-mono text-white/40 mb-2">Current Balance</div>
              <div className="text-4xl font-light tracking-tight mb-4" data-testid="billing-balance">{balanceDisplay}</div>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-12">
              <div className="border border-white/10 bg-white/[0.02] rounded-xl p-6 flex flex-col" data-testid="stripe-card">
                <div className="mb-6">
                  <CreditCard className="w-8 h-8 text-white/60 mb-4" />
                  <h3 className="text-xl font-bold mb-2">Credit Card</h3>
                  <p className="text-sm text-white/60 font-light">Add funds securely using Stripe. Supports all major credit cards.</p>
                </div>
                <div className="mt-auto">
                  <button onClick={() => setStripeModalOpen(true)} data-testid="add-funds-stripe-btn"
                    className="w-full bg-white text-black px-4 py-3 text-sm font-mono font-bold hover:bg-white/90 transition-colors rounded">
                    Add Funds with Stripe
                  </button>
                </div>
              </div>

              <div className="border border-[#14F195]/30 bg-[#14F195]/5 rounded-xl p-6 flex flex-col relative overflow-hidden" data-testid="solana-card">
                <div className="absolute -right-10 -top-10 w-32 h-32 bg-[#9945FF]/20 blur-3xl rounded-full pointer-events-none" />
                <div className="mb-6 relative z-10">
                  <svg className="w-8 h-8 text-[#14F195] mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M14.5 3h-10l-2 5h10l2-5Z"/><path d="M11.5 11h-10l-2 5h10l2-5Z"/><path d="M14.5 19h-10l-2 5h10l2-5Z"/></svg>
                  <h3 className="text-xl font-bold mb-2">Solana Payment</h3>
                  <p className="text-sm text-white/60 font-light">Pay instantly with SOL or USDC on the Solana network.</p>
                </div>
                <div className="mt-auto relative z-10">
                  <button className="w-full bg-gradient-to-r from-[#9945FF] to-[#14F195] text-black px-4 py-3 text-sm font-mono font-bold hover:opacity-90 transition-opacity rounded flex items-center justify-center gap-2" data-testid="connect-wallet-btn">
                    <Wallet className="w-4 h-4" /> Connect Wallet
                  </button>
                </div>
              </div>
            </div>

            {transactions.length > 0 && (
              <>
                <h2 className="text-xl font-bold tracking-tight mb-4">Transaction History</h2>
                <div className="border border-white/10 rounded-xl overflow-hidden">
                  <table className="w-full text-left text-sm" data-testid="payment-history-table">
                    <thead className="bg-white/5 font-mono text-xs text-white/40 border-b border-white/10">
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
                          <td className="px-6 py-4 text-white/60">{new Date(tx.created_at).toLocaleDateString()}</td>
                          <td className="px-6 py-4">{tx.tx_type}</td>
                          <td className="px-6 py-4 text-white/60">{tx.description}</td>
                          <td className={`px-6 py-4 ${tx.amount_cents > 0 ? 'text-green-400' : 'text-red-400'}`}>
                            ${Math.abs(tx.amount_cents / 10000).toFixed(4)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </>
            )}
          </motion.div>
        )}
      </main>

      <StripeCheckoutModal open={stripeModalOpen} onClose={() => { setStripeModalOpen(false); fetchBalance(); fetchTransactions(); }} stripePk={stripePk} />
    </div>
  );
}
