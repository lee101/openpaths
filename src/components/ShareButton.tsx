import React from 'react';
import { Check, Link2, Share2 } from 'lucide-react';

export function ShareButton({ title, text, path, className = '', compact = false }: { title: string; text?: string; path?: string; className?: string; compact?: boolean }) {
  const [copied, setCopied] = React.useState(false);
  const url = typeof window === 'undefined' ? '' : new URL(path || window.location.pathname, window.location.origin).toString();
  const canShare = typeof navigator !== 'undefined' && typeof navigator.share === 'function';

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1800);
    } catch {
      window.prompt('Copy link', url);
    }
  };

  const share = async () => {
    try {
      await navigator.share({ title, text, url });
    } catch (err) {
      if ((err as Error)?.name !== 'AbortError') copy();
    }
  };

  const btn = `inline-flex ${compact ? 'h-8 text-xs' : 'h-10 text-sm'} items-center gap-2 rounded-lg border border-white/25 px-3 font-mono text-white/80 transition-colors hover:border-white/60 hover:text-white`;
  return (
    <div className={`flex flex-wrap items-center gap-2 ${className}`}>
      {canShare && (
        <button type="button" onClick={share} className={btn}>
          <Share2 className="h-4 w-4" aria-hidden /> Share
        </button>
      )}
      <button type="button" onClick={copy} className={btn} aria-live="polite">
        {copied ? <Check className="h-4 w-4 text-emerald-300" aria-hidden /> : <Link2 className="h-4 w-4" aria-hidden />}
        {copied ? 'Link copied' : 'Copy link'}
      </button>
    </div>
  );
}
