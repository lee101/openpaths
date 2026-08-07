import React, { useEffect, useState } from 'react';
import { CreditCard, X } from 'lucide-react';
import { motion, AnimatePresence } from 'motion/react';
import { loadStripe } from '@stripe/stripe-js';
import { EmbeddedCheckout, EmbeddedCheckoutProvider } from '@stripe/react-stripe-js';
import { api } from '../lib/api';

const RECOMMENDED_THRESHOLD_USD = 100;
const RECOMMENDED_TOPUP_USD = 200;
const QUICK_TOPUP_AMOUNTS = [25, 100, 200, 500];

let stripePromise: ReturnType<typeof loadStripe> | null = null;
export function getStripe(pk: string) {
  if (!stripePromise && pk) stripePromise = loadStripe(pk);
  return stripePromise;
}

function formatUsdWhole(amount: number): string {
  return `$${amount.toLocaleString('en-US')}`;
}

export function TopUpModal({
  open,
  onClose,
  stripePk,
  initialAmount = RECOMMENDED_TOPUP_USD,
}: {
  open: boolean;
  onClose: () => void;
  stripePk?: string;
  initialAmount?: number;
}) {
  const [pk, setPk] = useState(stripePk || '');
  const [amount, setAmount] = useState(initialAmount);
  const [clientSecret, setClientSecret] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (stripePk) setPk(stripePk);
  }, [stripePk]);

  useEffect(() => {
    if (!open || pk) return;
    fetch('/account/stripe/config')
      .then(r => r.json())
      .then(d => {
        if (d.publishable_key) setPk(d.publishable_key);
      })
      .catch(() => {});
  }, [open, pk]);

  useEffect(() => {
    if (!open) return;
    setAmount(initialAmount);
    setClientSecret(null);
    setError('');
  }, [initialAmount, open]);

  const startCheckout = async () => {
    if (!pk) {
      setError('Stripe billing is not configured yet.');
      return;
    }

    setLoading(true);
    setError('');
    try {
      const res = await api('/account/stripe/checkout', {
        method: 'POST',
        body: JSON.stringify({ amount_usd: amount }),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data.error?.message || 'Failed to create checkout');
        return;
      }
      setClientSecret(data.client_secret);
    } catch {
      setError('Network error');
    } finally {
      setLoading(false);
    }
  };

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4"
          data-testid="stripe-modal-backdrop"
          onClick={e => {
            if (e.target === e.currentTarget) onClose();
          }}
        >
          <motion.div
            initial={{ opacity: 0, scale: 0.96, y: 10 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.96, y: 10 }}
            className="bg-[#090909] border border-white/10 rounded-3xl w-full max-w-2xl p-6 md:p-8 max-h-[90vh] overflow-y-auto shadow-[0_24px_80px_rgba(0,0,0,0.45)]"
            data-testid="stripe-modal"
          >
            <div className="flex items-start justify-between gap-4 mb-6">
              <div>
                <p className="text-xs font-mono uppercase tracking-[0.2em] text-white/35 mb-2">Prepaid credits</p>
                <h2 className="text-2xl font-bold tracking-tight">{clientSecret ? 'Complete Stripe payment' : 'Add funds'}</h2>
                <p className="text-sm text-white/55 mt-2 max-w-xl">
                  Stripe is the payment source of truth. OpenPaths keeps the spend model prepaid so your credit balance stays predictable.
                </p>
              </div>
              <button onClick={onClose} className="text-white/40 hover:text-white transition-colors shrink-0" data-testid="stripe-modal-close">
                <X className="w-5 h-5" />
              </button>
            </div>

            {!clientSecret ? (
              <>
                <div className="rounded-2xl border border-emerald-400/20 bg-[linear-gradient(135deg,rgba(16,185,129,0.12),rgba(255,255,255,0.02))] p-5 mb-6">
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <p className="text-xs font-mono uppercase tracking-[0.18em] text-emerald-200/70 mb-2">Recommended</p>
                      <h3 className="text-lg font-semibold text-white">Keep at least {formatUsdWhole(RECOMMENDED_THRESHOLD_USD)} ready</h3>
                      <p className="text-sm text-white/60 mt-2">Most teams avoid interruptions by keeping a larger prepaid reserve and using auto-topup as backup.</p>
                    </div>
                    <div className="rounded-2xl border border-white/10 bg-black/30 px-4 py-3">
                      <div className="text-xs font-mono text-white/45 mb-1">Top-up now</div>
                      <div className="text-2xl font-semibold">{formatUsdWhole(RECOMMENDED_TOPUP_USD)}</div>
                    </div>
                  </div>
                </div>

                <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-6">
                  {QUICK_TOPUP_AMOUNTS.map(a => (
                    <button
                      key={a}
                      onClick={() => setAmount(a)}
                      data-testid={`amount-${a}`}
                      className={`rounded-2xl border px-4 py-4 text-left transition-colors ${
                        amount === a
                          ? 'border-white bg-white text-black'
                          : 'border-white/10 bg-white/[0.03] text-white hover:border-white/30'
                      }`}
                    >
                      <div className="text-xs font-mono uppercase tracking-[0.16em] opacity-60">Credit pack</div>
                      <div className="text-xl font-semibold mt-2">{formatUsdWhole(a)}</div>
                    </button>
                  ))}
                </div>

                <div className="mb-6">
                  <label className="text-xs font-mono uppercase tracking-[0.16em] text-white/40 block mb-2">Custom amount</label>
                  <div className="relative">
                    <span className="absolute left-4 top-1/2 -translate-y-1/2 text-white/40 font-mono">$</span>
                    <input
                      type="number"
                      min="1"
                      max="500"
                      value={amount}
                      onChange={e => setAmount(Number(e.target.value))}
                      data-testid="custom-amount-input"
                      className="w-full bg-black border border-white/10 rounded-2xl py-4 pl-10 pr-4 text-white font-mono focus:outline-none focus:border-white/30"
                    />
                  </div>
                </div>

                {error && <p className="text-red-400 text-sm font-mono mb-4">{error}</p>}

                <button
                  onClick={startCheckout}
                  disabled={loading || amount < 1}
                  data-testid="stripe-checkout-btn"
                  className="w-full bg-white text-black py-4 rounded-2xl font-mono font-bold text-sm hover:bg-white/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
                >
                  {loading ? (
                    <span className="animate-spin w-4 h-4 border-2 border-black/20 border-t-black rounded-full" />
                  ) : (
                    <>
                      <CreditCard className="w-4 h-4" />
                      Pay {formatUsdWhole(amount)} with Stripe
                    </>
                  )}
                </button>
              </>
            ) : (
              <div data-testid="embedded-checkout-container">
                <EmbeddedCheckoutProvider stripe={getStripe(pk)!} options={{ clientSecret }}>
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
