import React from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, CreditCard, Lock, Sparkles, X } from 'lucide-react';
import { requestCreditTopUp } from '../lib/api';
import { requestSignIn, type Paywall } from '../lib/paywall';

export function SubscribePrompt({
  paywall,
  modelName,
  priceLabel,
  onDismiss,
}: {
  paywall: Paywall;
  modelName: string;
  priceLabel?: string;
  onDismiss?: () => void;
}) {
  const login = paywall.kind === 'login';
  const pricingHref = paywall.subscribeUrl.replace(/^https:\/\/openpaths\.io/, '') || '/pricing';
  return (
    <div className="relative mt-4 overflow-hidden rounded-xl border border-white/25 bg-gradient-to-br from-white/[0.10] via-white/[0.04] to-emerald-400/[0.08] p-5" role="dialog" aria-label={login ? 'Sign in to generate' : 'Add credits to generate'} data-testid="subscribe-prompt">
      <div className="pointer-events-none absolute -right-10 -top-10 h-32 w-32 rounded-full bg-emerald-400/10 blur-2xl" />
      {onDismiss && (
        <button type="button" onClick={onDismiss} className="absolute right-3 top-3 text-white/40 hover:text-white" aria-label="Dismiss">
          <X className="h-4 w-4" />
        </button>
      )}
      <div className="mb-2 inline-flex items-center gap-2 rounded border border-white/20 bg-black/40 px-2 py-0.5 font-mono text-[10px] uppercase tracking-[0.18em] text-white/60">
        {login ? <Lock className="h-3 w-3" /> : <Sparkles className="h-3 w-3" />}
        {login ? 'Account required' : 'Prepaid credits'}
      </div>
      <h3 className="pr-6 text-lg font-bold tracking-tight">{login ? `Sign in to run ${modelName}` : `Add credits to run ${modelName}`}</h3>
      <p className="mt-1.5 text-sm leading-relaxed text-white/65">
        {login
          ? 'One OpenPaths account runs image edit, video, 3D and every catalog model from this page or the same API.'
          : 'Your balance is empty. Prepaid credits power every OpenPaths space and API call, with transparent pay-as-you-go pricing.'}
      </p>
      <ul className="mt-3 space-y-1 font-mono text-xs text-white/60">
        {priceLabel && <li><span className="text-emerald-300">{priceLabel}</span> · billed only for completed results</li>}
        <li>Same key works from Python, JavaScript and cURL</li>
      </ul>
      <div className="mt-4 flex flex-wrap gap-2">
        {login ? (
          <button type="button" onClick={requestSignIn} className="inline-flex items-center gap-2 rounded border border-white bg-white px-4 py-2 font-mono text-sm font-bold text-black hover:bg-white/90" data-testid="subscribe-prompt-signin">
            Sign in or create account <ArrowRight className="h-4 w-4" />
          </button>
        ) : (
          <button type="button" onClick={requestCreditTopUp} className="inline-flex items-center gap-2 rounded border border-white bg-white px-4 py-2 font-mono text-sm font-bold text-black hover:bg-white/90" data-testid="subscribe-prompt-topup">
            <CreditCard className="h-4 w-4" /> Add credits
          </button>
        )}
        <Link to={pricingHref} className="inline-flex items-center gap-2 rounded border border-white/20 px-4 py-2 font-mono text-sm text-white/70 hover:border-white/50 hover:text-white" data-testid="subscribe-prompt-pricing">
          See pricing
        </Link>
      </div>
    </div>
  );
}
