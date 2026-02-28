import React, { useState } from 'react';
import { CreditCard, Key, Wallet, Plus, Copy, Check, Activity, ArrowUpRight, ExternalLink, X, TrendingUp } from 'lucide-react';
import { motion, AnimatePresence } from 'motion/react';

const USAGE_DATA = [
  { date: 'Jan 1', requests: 28000, cost: 4.20 },
  { date: 'Jan 8', requests: 35000, cost: 5.80 },
  { date: 'Jan 15', requests: 42000, cost: 7.10 },
  { date: 'Jan 22', requests: 38000, cost: 6.40 },
  { date: 'Jan 29', requests: 51000, cost: 8.90 },
  { date: 'Feb 5', requests: 47000, cost: 7.60 },
  { date: 'Feb 12', requests: 62000, cost: 10.20 },
  { date: 'Feb 19', requests: 58000, cost: 9.50 },
  { date: 'Feb 26', requests: 71000, cost: 12.40 },
];

const STRIPE_AMOUNTS = [10, 25, 50, 100];

function UsageGraph({ data }: { data: typeof USAGE_DATA }) {
  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null);
  const maxReq = Math.max(...data.map(d => d.requests));
  const w = 600, h = 200, px = 40, py = 20;
  const chartW = w - px * 2, chartH = h - py * 2;

  const points = data.map((d, i) => ({
    x: px + (i / (data.length - 1)) * chartW,
    y: py + chartH - (d.requests / maxReq) * chartH,
  }));

  const linePath = points.map((p, i) => `${i === 0 ? 'M' : 'L'}${p.x},${p.y}`).join(' ');
  const areaPath = `${linePath} L${points[points.length - 1].x},${h - py} L${points[0].x},${h - py} Z`;

  return (
    <div className="border border-white/10 bg-white/[0.02] rounded-xl p-6" data-testid="usage-graph">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-bold tracking-tight flex items-center gap-2">
          <TrendingUp className="w-5 h-5 text-white/40" /> Usage Over Time
        </h3>
        <span className="text-xs font-mono text-white/40">Last 9 weeks</span>
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

function StripeCheckoutModal({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [amount, setAmount] = useState(25);
  const [loading, setLoading] = useState(false);

  const handleCheckout = () => {
    setLoading(true);
    setTimeout(() => {
      window.open('https://checkout.stripe.com/pay/demo', '_blank');
      setLoading(false);
      onClose();
    }, 800);
  };

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4"
          data-testid="stripe-modal-backdrop"
          onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
        >
          <motion.div
            initial={{ opacity: 0, scale: 0.95, y: 10 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.95, y: 10 }}
            className="bg-[#0a0a0a] border border-white/10 rounded-xl w-full max-w-md p-6"
            data-testid="stripe-modal"
          >
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-xl font-bold tracking-tight">Add Funds</h2>
              <button onClick={onClose} className="text-white/40 hover:text-white transition-colors" data-testid="stripe-modal-close">
                <X className="w-5 h-5" />
              </button>
            </div>

            <p className="text-sm text-white/60 mb-6">Select an amount to add to your OpenPath balance via Stripe.</p>

            <div className="grid grid-cols-2 gap-3 mb-6">
              {STRIPE_AMOUNTS.map(a => (
                <button
                  key={a}
                  onClick={() => setAmount(a)}
                  data-testid={`amount-${a}`}
                  className={`py-3 rounded-lg border text-sm font-mono font-bold transition-colors ${
                    amount === a
                      ? 'bg-white text-black border-white'
                      : 'bg-white/5 text-white/60 border-white/10 hover:border-white/30'
                  }`}
                >
                  ${a}
                </button>
              ))}
            </div>

            <div className="mb-6">
              <label className="text-xs font-mono text-white/40 block mb-2">Custom amount</label>
              <div className="relative">
                <span className="absolute left-3 top-1/2 -translate-y-1/2 text-white/40 font-mono">$</span>
                <input
                  type="number"
                  min="5"
                  max="500"
                  value={amount}
                  onChange={e => setAmount(Number(e.target.value))}
                  data-testid="custom-amount-input"
                  className="w-full bg-black border border-white/10 rounded-lg py-3 pl-8 pr-4 text-white font-mono focus:outline-none focus:border-white/30"
                />
              </div>
            </div>

            <button
              onClick={handleCheckout}
              disabled={loading || amount < 5}
              data-testid="stripe-checkout-btn"
              className="w-full bg-white text-black py-3 rounded-lg font-mono font-bold text-sm hover:bg-white/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
            >
              {loading ? (
                <span className="animate-spin w-4 h-4 border-2 border-black/20 border-t-black rounded-full" />
              ) : (
                <>
                  <CreditCard className="w-4 h-4" /> Pay ${amount} with Stripe
                </>
              )}
            </button>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}

export function Account() {
  const [activeTab, setActiveTab] = useState<'overview' | 'keys' | 'billing'>('overview');
  const [copied, setCopied] = useState(false);
  const [stripeModalOpen, setStripeModalOpen] = useState(false);

  const copyKey = () => {
    navigator.clipboard.writeText('op_live_8f92a1b3c4d5e6f7g8h9i0j1k2l3m4n5');
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const openStripePortal = () => {
    window.open('https://billing.stripe.com/p/login/demo', '_blank');
  };

  return (
    <div className="max-w-6xl mx-auto px-6 py-12 flex flex-col md:flex-row gap-12">
      <aside className="w-full md:w-64 shrink-0">
        <div className="mb-8">
          <h2 className="text-xl font-bold tracking-tight mb-1">Account</h2>
          <p className="text-sm font-mono text-white/40">user@example.com</p>
        </div>

        <nav className="flex flex-col gap-2 font-mono text-sm">
          <button
            onClick={() => setActiveTab('overview')}
            data-testid="tab-overview"
            className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors text-left ${activeTab === 'overview' ? 'bg-white/10 text-white' : 'text-white/60 hover:bg-white/5 hover:text-white'}`}
          >
            <Activity className="w-4 h-4" /> Overview
          </button>
          <button
            onClick={() => setActiveTab('keys')}
            data-testid="tab-keys"
            className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors text-left ${activeTab === 'keys' ? 'bg-white/10 text-white' : 'text-white/60 hover:bg-white/5 hover:text-white'}`}
          >
            <Key className="w-4 h-4" /> API Keys
          </button>
          <button
            onClick={() => setActiveTab('billing')}
            data-testid="tab-billing"
            className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors text-left ${activeTab === 'billing' ? 'bg-white/10 text-white' : 'text-white/60 hover:bg-white/5 hover:text-white'}`}
          >
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
                <div className="text-4xl font-light tracking-tight mb-4" data-testid="balance">$42.50</div>
                <button onClick={() => setActiveTab('billing')} className="text-xs font-mono text-white border border-white/20 px-3 py-1.5 rounded hover:bg-white/10 transition-colors">
                  Add Funds
                </button>
              </div>
              <div className="border border-white/10 bg-white/[0.02] rounded-xl p-6">
                <div className="text-sm font-mono text-white/40 mb-2">Requests (30d)</div>
                <div className="text-4xl font-light tracking-tight mb-4" data-testid="requests-count">1.2M</div>
                <div className="text-xs font-mono text-green-400 flex items-center gap-1">
                  <ArrowUpRight className="w-3 h-3" /> +14% from last month
                </div>
              </div>
            </div>

            <div className="mb-12">
              <UsageGraph data={USAGE_DATA} />
            </div>

            <h2 className="text-xl font-bold tracking-tight mb-4">Recent Activity</h2>
            <div className="border border-white/10 rounded-xl overflow-hidden">
              <table className="w-full text-left text-sm" data-testid="activity-table">
                <thead className="bg-white/5 font-mono text-xs text-white/40 border-b border-white/10">
                  <tr>
                    <th className="px-6 py-3 font-normal">Model</th>
                    <th className="px-6 py-3 font-normal">Requests</th>
                    <th className="px-6 py-3 font-normal">Cost</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/10 font-mono">
                  <tr>
                    <td className="px-6 py-4">anthropic/claude-3.5-sonnet</td>
                    <td className="px-6 py-4 text-white/60">45,210</td>
                    <td className="px-6 py-4">$12.40</td>
                  </tr>
                  <tr>
                    <td className="px-6 py-4">openai/gpt-4o</td>
                    <td className="px-6 py-4 text-white/60">12,050</td>
                    <td className="px-6 py-4">$8.20</td>
                  </tr>
                  <tr>
                    <td className="px-6 py-4">meta-llama/llama-3.1-70b-instruct</td>
                    <td className="px-6 py-4 text-white/60">89,400</td>
                    <td className="px-6 py-4">$2.15</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </motion.div>
        )}

        {activeTab === 'keys' && (
          <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
            <div className="flex justify-between items-center mb-8">
              <h1 className="text-3xl font-bold tracking-tight">API Keys</h1>
              <button className="bg-white text-black px-4 py-2 text-sm font-mono font-bold hover:bg-white/90 transition-colors flex items-center gap-2 rounded" data-testid="create-key-btn">
                <Plus className="w-4 h-4" /> Create Key
              </button>
            </div>

            <div className="border border-white/10 bg-white/[0.02] rounded-xl p-6 mb-8" data-testid="api-key-card">
              <div className="flex justify-between items-start mb-4">
                <div>
                  <h3 className="font-bold mb-1">Default Project Key</h3>
                  <p className="text-xs font-mono text-white/40">Created on Oct 24, 2023</p>
                </div>
                <div className="px-2 py-1 bg-green-500/10 text-green-400 text-[10px] font-mono rounded border border-green-500/20" data-testid="key-status">Active</div>
              </div>
              <div className="flex items-center gap-4 bg-black border border-white/10 rounded-lg p-3">
                <code className="flex-1 font-mono text-sm text-white/80" data-testid="api-key-value">op_live_8f92a1b3c4d5e6f7g8h9i0j1k2l3m4n5</code>
                <button
                  onClick={copyKey}
                  className="p-2 hover:bg-white/10 rounded transition-colors text-white/60 hover:text-white"
                  title="Copy to clipboard"
                  data-testid="copy-key-btn"
                >
                  {copied ? <Check className="w-4 h-4 text-green-400" /> : <Copy className="w-4 h-4" />}
                </button>
              </div>
            </div>

            <p className="text-sm text-white/40 font-light">
              Do not share your API key in publicly accessible areas such as GitHub, client-side code, and so forth.
            </p>
          </motion.div>
        )}

        {activeTab === 'billing' && (
          <motion.div initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }}>
            <div className="flex items-center justify-between mb-8">
              <h1 className="text-3xl font-bold tracking-tight">Billing & Payments</h1>
              <button
                onClick={openStripePortal}
                data-testid="stripe-portal-link"
                className="flex items-center gap-2 text-sm font-mono text-white/60 hover:text-white border border-white/10 px-4 py-2 rounded-lg hover:bg-white/5 transition-colors"
              >
                <ExternalLink className="w-4 h-4" /> Manage Billing
              </button>
            </div>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-12">
              <div className="border border-white/10 bg-white/[0.02] rounded-xl p-6 flex flex-col" data-testid="stripe-card">
                <div className="mb-6">
                  <CreditCard className="w-8 h-8 text-white/60 mb-4" />
                  <h3 className="text-xl font-bold mb-2">Credit Card</h3>
                  <p className="text-sm text-white/60 font-light">Add funds securely using Stripe. Supports all major credit cards.</p>
                </div>
                <div className="mt-auto">
                  <button
                    onClick={() => setStripeModalOpen(true)}
                    data-testid="add-funds-stripe-btn"
                    className="w-full bg-white text-black px-4 py-3 text-sm font-mono font-bold hover:bg-white/90 transition-colors rounded"
                  >
                    Add Funds with Stripe
                  </button>
                </div>
              </div>

              <div className="border border-[#14F195]/30 bg-[#14F195]/5 rounded-xl p-6 flex flex-col relative overflow-hidden" data-testid="solana-card">
                <div className="absolute -right-10 -top-10 w-32 h-32 bg-[#9945FF]/20 blur-3xl rounded-full pointer-events-none" />
                <div className="mb-6 relative z-10">
                  <svg className="w-8 h-8 text-[#14F195] mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M14.5 3h-10l-2 5h10l2-5Z"/><path d="M11.5 11h-10l-2 5h10l2-5Z"/><path d="M14.5 19h-10l-2 5h10l2-5Z"/></svg>
                  <h3 className="text-xl font-bold mb-2">Solana Payment</h3>
                  <p className="text-sm text-white/60 font-light">Pay instantly with SOL or USDC on the Solana network. Zero wait times.</p>
                </div>
                <div className="mt-auto relative z-10">
                  <button className="w-full bg-gradient-to-r from-[#9945FF] to-[#14F195] text-black px-4 py-3 text-sm font-mono font-bold hover:opacity-90 transition-opacity rounded flex items-center justify-center gap-2" data-testid="connect-wallet-btn">
                    <Wallet className="w-4 h-4" /> Connect Wallet
                  </button>
                </div>
              </div>
            </div>

            <h2 className="text-xl font-bold tracking-tight mb-4">Payment History</h2>
            <div className="border border-white/10 rounded-xl overflow-hidden">
              <table className="w-full text-left text-sm" data-testid="payment-history-table">
                <thead className="bg-white/5 font-mono text-xs text-white/40 border-b border-white/10">
                  <tr>
                    <th className="px-6 py-3 font-normal">Date</th>
                    <th className="px-6 py-3 font-normal">Method</th>
                    <th className="px-6 py-3 font-normal">Amount</th>
                    <th className="px-6 py-3 font-normal">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/10 font-mono">
                  <tr>
                    <td className="px-6 py-4 text-white/60">Oct 24, 2023</td>
                    <td className="px-6 py-4">Solana (USDC)</td>
                    <td className="px-6 py-4">$50.00</td>
                    <td className="px-6 py-4 text-green-400">Completed</td>
                  </tr>
                  <tr>
                    <td className="px-6 py-4 text-white/60">Sep 12, 2023</td>
                    <td className="px-6 py-4">Stripe (...4242)</td>
                    <td className="px-6 py-4">$25.00</td>
                    <td className="px-6 py-4 text-green-400">Completed</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </motion.div>
        )}
      </main>

      <StripeCheckoutModal open={stripeModalOpen} onClose={() => setStripeModalOpen(false)} />
    </div>
  );
}
